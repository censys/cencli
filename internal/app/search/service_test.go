package search

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"
)

func TestSearchService(t *testing.T) {
	testCases := []struct {
		name         string
		client       func(ctrl *gomock.Controller) client.Client
		orgID        mo.Option[identifiers.OrganizationID]
		collectionID mo.Option[identifiers.CollectionID]
		query        string
		fields       []string
		pagination   func() (pageSize mo.Option[uint64], maxPages mo.Option[uint64])
		ctx          func() context.Context
		assert       func(t *testing.T, res Result, err cenclierrors.CencliError)
	}{
		{
			name: "success - no collection - no org",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.None[int64](),
					mo.None[string](),
				).Return(client.Result[components.SearchQueryResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
					},
					Data: &components.SearchQueryResponse{
						Hits: []components.SearchQueryHit{{
							HostV1: &components.HostAssetWithMatchedServices{
								Resource: components.Host{
									IP: strPtr("127.0.0.1"),
								},
							},
						}},
					},
				}, nil)
				return mockClient
			},
			query:  "query",
			fields: []string{"field"},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Hits, 1)
				require.IsType(t, &assets.Host{}, res.Hits[0])
				require.Equal(t, "127.0.0.1", *res.Hits[0].(*assets.Host).IP)
			},
		},
		{
			name: "success - with matched services",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.None[int64](),
					mo.None[string](),
				).Return(client.Result[components.SearchQueryResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
					},
					Data: &components.SearchQueryResponse{
						Hits: []components.SearchQueryHit{{
							HostV1: &components.HostAssetWithMatchedServices{
								Resource: components.Host{
									IP: strPtr("127.0.0.1"),
								},
								MatchedServices: []components.MatchedService{
									{
										Port:              intPtr(22),
										Protocol:          strPtr("SSH"),
										TransportProtocol: strPtr(components.TransportProtocolTCP),
									},
								},
							},
						}},
					},
				}, nil)
				return mockClient
			},
			query:  "query",
			fields: []string{"field"},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Hits, 1)
				require.IsType(t, &assets.Host{}, res.Hits[0])
				require.Equal(t, "127.0.0.1", *res.Hits[0].(*assets.Host).IP)
				require.Equal(t, "SSH", *res.Hits[0].(*assets.Host).MatchedServices[0].Protocol)
				require.Equal(t, components.TransportProtocolTCP, *res.Hits[0].(*assets.Host).MatchedServices[0].TransportProtocol)
				require.Equal(t, 22, *res.Hits[0].(*assets.Host).MatchedServices[0].Port)
			},
		},
		{
			name: "success - with matched services",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.None[int64](),
					mo.None[string](),
				).Return(client.Result[components.SearchQueryResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
					},
					Data: &components.SearchQueryResponse{
						Hits: []components.SearchQueryHit{{
							HostV1: &components.HostAssetWithMatchedServices{
								Resource: components.Host{
									IP: strPtr("127.0.0.1"),
								},
								MatchedServices: []components.MatchedService{
									{
										Port:              intPtr(22),
										Protocol:          strPtr("SSH"),
										TransportProtocol: strPtr(components.TransportProtocolTCP),
									},
								},
							},
						}},
					},
				}, nil)
				return mockClient
			},
			query:  "query",
			fields: []string{"field"},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Hits, 1)
				require.IsType(t, &assets.Host{}, res.Hits[0])
				require.Equal(t, "127.0.0.1", *res.Hits[0].(*assets.Host).IP)
				require.Equal(t, "SSH", *res.Hits[0].(*assets.Host).MatchedServices[0].Protocol)
				require.Equal(t, components.TransportProtocolTCP, *res.Hits[0].(*assets.Host).MatchedServices[0].TransportProtocol)
				require.Equal(t, 22, *res.Hits[0].(*assets.Host).MatchedServices[0].Port)
			},
		},
		{
			name: "success - with collection - no org",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().SearchCollection(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.None[int64](),
					mo.None[string](),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{{
								HostV1: &components.HostAssetWithMatchedServices{
									Resource: components.Host{
										IP: strPtr("127.0.0.1"),
									},
								},
							}},
						},
					}, nil)
				return mockClient
			},
			query:        "query",
			collectionID: mo.Some(identifiers.NewCollectionID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))),
			fields:       []string{"field"},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Hits, 1)
				require.IsType(t, &assets.Host{}, res.Hits[0])
				require.Equal(t, "127.0.0.1", *res.Hits[0].(*assets.Host).IP)
			},
		},
		{
			name: "success - no collection - with org",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.Some("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
					"query",
					[]string{"field"},
					mo.None[int64](),
					mo.None[string](),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{{
								HostV1: &components.HostAssetWithMatchedServices{
									Resource: components.Host{
										IP: strPtr("127.0.0.1"),
									},
								},
							}},
						},
					}, nil)
				return mockClient
			},
			query:  "query",
			fields: []string{"field"},
			orgID:  mo.Some(identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))),
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Hits, 1)
				require.IsType(t, &assets.Host{}, res.Hits[0])
				require.Equal(t, "127.0.0.1", *res.Hits[0].(*assets.Host).IP)
			},
		},
		{
			name: "client structured error",
			client: func(ctrl *gomock.Controller) client.Client {
				detail := "Invalid search query format"
				status := int64(400)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
					Detail: &detail,
					Status: &status,
				})
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Search(gomock.Any(), mo.None[string](), "8.8.8.8", []string{"field"}, mo.None[int64](), mo.None[string]()).Return(
					client.Result[components.SearchQueryResponse]{},
					structuredErr,
				)
				return mockClient
			},
			query:  "8.8.8.8",
			fields: []string{"field"},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NotNil(t, err)
				var cencliErr client.ClientStructuredError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "Invalid search query format")
				require.Contains(t, err.Error(), "400")
			},
		},
		{
			name: "client generic error",
			client: func(ctrl *gomock.Controller) client.Client {
				genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "Bad Request",
					StatusCode: 400,
					Body:       "Invalid search query format",
				})
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Search(gomock.Any(), mo.None[string](), "8.8.8.8", []string{"field"}, mo.None[int64](), mo.None[string]()).Return(
					client.Result[components.SearchQueryResponse]{},
					genericErr,
				)
				return mockClient
			},
			query:  "8.8.8.8",
			fields: []string{"field"},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NotNil(t, err)
				var cencliErr client.ClientGenericError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "Bad Request")
				require.Contains(t, err.Error(), "400")
			},
		},
		{
			name: "client unknown error",
			client: func(ctrl *gomock.Controller) client.Client {
				unknownErr := client.NewClientError(errors.New("network timeout"))
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Search(gomock.Any(), mo.None[string](), "8.8.8.8", []string{"field"}, mo.None[int64](), mo.None[string]()).Return(
					client.Result[components.SearchQueryResponse]{},
					unknownErr,
				)
				return mockClient
			},
			query:  "8.8.8.8",
			fields: []string{"field"},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NotNil(t, err)
				var cencliErr client.ClientError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "network timeout")
			},
		},
		{
			name: "pagination - single page with results",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(2)),
					mo.None[string](),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.1"),
										},
									},
								},
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.2"),
										},
									},
								},
							},
							TotalHits:     2,
							NextPageToken: "", // No next page
						},
					}, nil)
				return mockClient
			},
			query:  "query",
			fields: []string{"field"},
			pagination: func() (pageSize mo.Option[uint64], maxPages mo.Option[uint64]) {
				return mo.Some(uint64(2)), mo.None[uint64]()
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Hits, 2)
				require.Equal(t, int64(2), res.TotalHits)
			},
		},
		{
			name: "pagination - multiple pages with results",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				// First page
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(2)),
					mo.None[string](),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.1"),
										},
									},
								},
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.2"),
										},
									},
								},
							},
							TotalHits:     5,
							NextPageToken: "token1",
						},
					}, nil)
				// Second page
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(2)),
					mo.Some("token1"),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.3"),
										},
									},
								},
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.4"),
										},
									},
								},
							},
							TotalHits:     5,
							NextPageToken: "token2",
						},
					}, nil)
				// Third page
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(2)),
					mo.Some("token2"),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.5"),
										},
									},
								},
							},
							TotalHits:     5,
							NextPageToken: "", // No next page
						},
					}, nil)
				return mockClient
			},
			query:  "query",
			fields: []string{"field"},
			pagination: func() (pageSize mo.Option[uint64], maxPages mo.Option[uint64]) {
				return mo.Some(uint64(2)), mo.None[uint64]()
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Hits, 5)
				require.Equal(t, int64(5), res.TotalHits)
				// Verify all IPs are present
				ips := make([]string, len(res.Hits))
				for i, hit := range res.Hits {
					ips[i] = *hit.(*assets.Host).IP
				}
				require.ElementsMatch(t, []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "127.0.0.4", "127.0.0.5"}, ips)
			},
		},
		{
			name: "pagination - limited by maxPages",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				// First page
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(2)),
					mo.None[string](),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.1"),
										},
									},
								},
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.2"),
										},
									},
								},
							},
							TotalHits:     10,
							NextPageToken: "token1",
						},
					}, nil)
				// Second page
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(2)),
					mo.Some("token1"),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.3"),
										},
									},
								},
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.4"),
										},
									},
								},
							},
							TotalHits:     10,
							NextPageToken: "token2", // More pages available but we stop here
						},
					}, nil)
				// No third page call expected due to maxPages=2
				return mockClient
			},
			query:  "query",
			fields: []string{"field"},
			pagination: func() (pageSize mo.Option[uint64], maxPages mo.Option[uint64]) {
				return mo.Some(uint64(2)), mo.Some(uint64(2)) // Limit to 2 pages
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Hits, 4) // Only 2 pages worth of results
				require.Equal(t, int64(10), res.TotalHits)
			},
		},
		{
			name: "pagination - collection search with multiple pages",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				// First page
				mockClient.EXPECT().SearchCollection(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(1)),
					mo.None[string](),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.1"),
										},
									},
								},
							},
							TotalHits:     2,
							NextPageToken: "token1",
						},
					}, nil)
				// Second page
				mockClient.EXPECT().SearchCollection(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(1)),
					mo.Some("token1"),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.2"),
										},
									},
								},
							},
							TotalHits:     2,
							NextPageToken: "",
						},
					}, nil)
				return mockClient
			},
			query:        "query",
			collectionID: mo.Some(identifiers.NewCollectionID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))),
			fields:       []string{"field"},
			pagination: func() (pageSize mo.Option[uint64], maxPages mo.Option[uint64]) {
				return mo.Some(uint64(1)), mo.None[uint64]()
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Hits, 2)
				require.Equal(t, int64(2), res.TotalHits)
			},
		},
		{
			name: "pagination - page with no results but next token exists",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				// First page with results
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(2)),
					mo.None[string](),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.1"),
										},
									},
								},
							},
							TotalHits:     5,
							NextPageToken: "token1",
						},
					}, nil)
				// Second page with no results but next token (filtered results scenario)
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(2)),
					mo.Some("token1"),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits:          []components.SearchQueryHit{}, // No hits
							TotalHits:     5,
							NextPageToken: "token2", // But still has next token
						},
					}, nil)
				// Should not call third page due to empty results
				return mockClient
			},
			query:  "query",
			fields: []string{"field"},
			pagination: func() (pageSize mo.Option[uint64], maxPages mo.Option[uint64]) {
				return mo.Some(uint64(2)), mo.None[uint64]()
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Hits, 1) // Only first page results
				require.Equal(t, int64(5), res.TotalHits)
			},
		},
		{
			name: "pagination - invalid maxPages",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				// No calls expected due to validation error
				return mockClient
			},
			query:  "query",
			fields: []string{"field"},
			pagination: func() (pageSize mo.Option[uint64], maxPages mo.Option[uint64]) {
				return mo.Some(uint64(2)), mo.Some(uint64(0)) // Invalid maxPages
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "max pages must be greater than 0")
			},
		},
		{
			name: "pagination - error on second page",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				// First page succeeds
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(2)),
					mo.None[string](),
				).
					Return(client.Result[components.SearchQueryResponse]{
						Metadata: client.Metadata{
							Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
							Response: &http.Response{StatusCode: 200},
							Latency:  100 * time.Millisecond,
						},
						Data: &components.SearchQueryResponse{
							Hits: []components.SearchQueryHit{
								{
									HostV1: &components.HostAssetWithMatchedServices{
										Resource: components.Host{
											IP: strPtr("127.0.0.1"),
										},
									},
								},
							},
							TotalHits:     5,
							NextPageToken: "token1",
						},
					}, nil)
				// Second page fails
				mockClient.EXPECT().Search(
					gomock.Any(),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.Some(int64(2)),
					mo.Some("token1"),
				).
					Return(client.Result[components.SearchQueryResponse]{}, client.NewClientError(errors.New("network error")))
				return mockClient
			},
			query:  "query",
			fields: []string{"field"},
			pagination: func() (pageSize mo.Option[uint64], maxPages mo.Option[uint64]) {
				return mo.Some(uint64(2)), mo.None[uint64]()
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Hits, 1)
				require.NotNil(t, res.PartialError)
				require.Contains(t, res.PartialError.Error(), "network error")
			},
		},
		{
			name: "context cancellation propagates",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl)
			},
			query:  "query",
			fields: []string{"field"},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.ErrorIs(t, err, context.Canceled)
			},
		},
		{
			name: "succeeds without service-level retry",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Search(
					gomock.AssignableToTypeOf(context.Background()),
					mo.None[string](),
					"query",
					[]string{"field"},
					mo.None[int64](),
					mo.None[string](),
				).Return(client.Result[components.SearchQueryResponse]{
					Metadata: client.Metadata{},
					Data:     &components.SearchQueryResponse{},
				}, nil)
				return mockClient
			},
			query:  "query",
			fields: []string{"field"},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, int64(0), res.TotalHits)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := tc.client(ctrl)
			svc := New(mockClient)

			pageSize := mo.None[uint64]()
			maxPages := mo.None[uint64]()
			if tc.pagination != nil {
				pageSize, maxPages = tc.pagination()
			}

			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}

			params := Params{
				OrgID:        tc.orgID,
				CollectionID: tc.collectionID,
				Query:        tc.query,
				Fields:       tc.fields,
				PageSize:     pageSize,
				MaxPages:     maxPages,
			}
			res, err := svc.Search(ctx, params)
			tc.assert(t, res, err)
		})
	}
}

func TestSearchService_DeadlineBeforeFirstRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockClient(ctrl)
	// No expectations: service should return before invoking client due to expired context

	svc := New(mockClient)
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	params := Params{
		Query:  "query",
		Fields: []string{"field"},
	}
	_, err := svc.Search(ctx, params)
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSearchService_DeadlineExpiresBetweenPages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockClient(ctrl)
	// First page call sleeps past the overall context deadline, then returns a next page token
	mockClient.EXPECT().Search(
		gomock.Any(),
		mo.None[string](),
		"query",
		[]string{"field"},
		mo.Some(int64(1)),
		mo.None[string](),
	).DoAndReturn(func(ctx context.Context, org mo.Option[string], query string, fields []string, pageSize mo.Option[int64], pageToken mo.Option[string]) (client.Result[components.SearchQueryResponse], client.ClientError) {
		// Simulate slow first page so that context deadline is exceeded before next iteration
		time.Sleep(200 * time.Millisecond)
		return client.Result[components.SearchQueryResponse]{
			Metadata: client.Metadata{
				Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
				Response: &http.Response{StatusCode: 200},
			},
			Data: &components.SearchQueryResponse{
				Hits: []components.SearchQueryHit{{
					HostV1: &components.HostAssetWithMatchedServices{Resource: components.Host{IP: strPtr("127.0.0.1")}},
				}},
				TotalHits:     2,
				NextPageToken: "next",
			},
		}, nil
	})
	// No expectation for a second Search call; if made, gomock will fail the test

	svc := New(mockClient)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	params := Params{
		Query:    "query",
		Fields:   []string{"field"},
		PageSize: mo.Some(uint64(1)),
	}
	res, err := svc.Search(ctx, params)
	require.NoError(t, err)
	require.Len(t, res.Hits, 1)
	require.NotNil(t, res.PartialError)
	require.ErrorIs(t, res.PartialError, context.DeadlineExceeded)
}

func strPtr[T ~string](v T) *T { return &v }

func intPtr(v int) *int { return &v }
