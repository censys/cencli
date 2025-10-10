package censeye

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
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

func TestInvestigateHost(t *testing.T) {
	// Save and restore the default config
	originalConfig := defaultCenseyeConfig
	defer func() { defaultCenseyeConfig = originalConfig }()

	// Override with a simple test config for testing
	// This allows us to test filtering and extraction without complex mock responses
	defaultCenseyeConfig = censeyeConfig{
		Filters:          []string{"host.location.", "host.autonomous_system."},
		RgxFilters:       []*regexp.Regexp{regexp.MustCompile(`host\.name=`)},
		KeyValuePrefixes: []string{"host.services.http.request.headers"},
		ExtractionRules:  nil, // No complex extraction rules for simpler testing
	}

	testCases := []struct {
		name      string
		client    func(ctrl *gomock.Controller) client.Client
		orgID     mo.Option[identifiers.OrganizationID]
		host      *assets.Host
		rarityMin uint64
		rarityMax uint64
		ctx       func() context.Context
		assert    func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - counts within rarity range",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				// Expect 1 rule from host.ip field
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Len(1), // Should have 1 count condition for the IP
				).Return(client.Result[components.ValueCountsResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.ValueCountsResponse{
						AndCountResults: []float64{50}, // Count within range [10, 100]
					},
				}, nil)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("192.168.1.1"),
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Entries, 1, "should have one entry for the IP field")

				entry := res.Entries[0]
				assert.Equal(t, int64(50), entry.Count)
				assert.True(t, entry.Interesting, "count of 50 should be interesting (within range [10,100])")
				assert.Equal(t, `host.ip="192.168.1.1"`, entry.Query)
				assert.Contains(t, entry.SearchURL, "https://platform.censys.io/search?q=")
			},
		},
		{
			name: "success - count below min not interesting",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Len(1),
				).Return(client.Result[components.ValueCountsResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.ValueCountsResponse{
						AndCountResults: []float64{5}, // Count below min
					},
				}, nil)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("10.0.0.1"),
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Entries, 1)

				entry := res.Entries[0]
				assert.Equal(t, int64(5), entry.Count)
				assert.False(t, entry.Interesting, "count of 5 should not be interesting (below min of 10)")
			},
		},
		{
			name: "success - count above max not interesting",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Len(1),
				).Return(client.Result[components.ValueCountsResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.ValueCountsResponse{
						AndCountResults: []float64{200}, // Count above max
					},
				}, nil)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("10.0.0.2"),
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Entries, 1)

				entry := res.Entries[0]
				assert.Equal(t, int64(200), entry.Count)
				assert.False(t, entry.Interesting, "count of 200 should not be interesting (above max of 100)")
			},
		},
		{
			name: "client structured error",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				detail := "Invalid organization ID"
				status := int64(403)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
					Detail: &detail,
					Status: &status,
				})
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Any(),
				).Return(client.Result[components.ValueCountsResponse]{}, structuredErr)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("192.168.1.1"),
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.NotNil(t, err)
				var cencliErr client.ClientStructuredError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "Invalid organization ID")
				require.Contains(t, err.Error(), "403")
			},
		},
		{
			name: "client generic error",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "Bad Request",
					StatusCode: 400,
					Body:       "Invalid request body",
				})
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Any(),
				).Return(client.Result[components.ValueCountsResponse]{}, genericErr)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("192.168.1.1"),
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
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
				mockClient := mocks.NewMockClient(ctrl)
				unknownErr := client.NewClientError(errors.New("network timeout"))
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Any(),
				).Return(client.Result[components.ValueCountsResponse]{}, unknownErr)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("192.168.1.1"),
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.NotNil(t, err)
				var cencliErr client.ClientError
				require.ErrorAs(t, err, &cencliErr)
				require.Contains(t, err.Error(), "network timeout")
			},
		},
		{
			name: "context canceled via client error",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Any(),
				).Return(client.Result[components.ValueCountsResponse]{}, client.NewClientError(context.Canceled))
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("192.168.1.1"),
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "the operation's context was cancelled before it completed")
			},
		},
		{
			name: "with orgID",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.Some("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
					mo.None[string](),
					gomock.Len(1),
				).Return(client.Result[components.ValueCountsResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.ValueCountsResponse{
						AndCountResults: []float64{50},
					},
				}, nil)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("192.168.1.1"),
				},
			},
			orgID:     mo.Some(identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))),
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Entries, 1)
			},
		},
		{
			name: "boundary - count exactly at min is interesting",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Len(1),
				).Return(client.Result[components.ValueCountsResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.ValueCountsResponse{
						AndCountResults: []float64{10}, // Exactly at min
					},
				}, nil)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("192.168.1.1"),
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Entries, 1)

				entry := res.Entries[0]
				assert.Equal(t, int64(10), entry.Count)
				assert.True(t, entry.Interesting, "count exactly at min should be interesting")
			},
		},
		{
			name: "boundary - count exactly at max is interesting",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Len(1),
				).Return(client.Result[components.ValueCountsResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.ValueCountsResponse{
						AndCountResults: []float64{100}, // Exactly at max
					},
				}, nil)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("192.168.1.1"),
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Entries, 1)

				entry := res.Entries[0]
				assert.Equal(t, int64(100), entry.Count)
				assert.True(t, entry.Interesting, "count exactly at max should be interesting")
			},
		},
		{
			name: "multiple fields - entries sorted by count descending",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				// Host with IP and services will generate multiple rules
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Any(),
				).Return(client.Result[components.ValueCountsResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.ValueCountsResponse{
						// Returns counts in unsorted order
						AndCountResults: []float64{25, 100, 50},
					},
				}, nil)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("192.168.1.1"),
					Services: []components.Service{
						{Port: intPtr(80), Protocol: strPtr("HTTP")},
					},
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotEmpty(t, res.Entries)

				// Verify entries are sorted by count descending
				for i := 0; i < len(res.Entries)-1; i++ {
					if res.Entries[i].Count == res.Entries[i+1].Count {
						// If counts are equal, should be sorted by query
						assert.LessOrEqual(t, res.Entries[i].Query, res.Entries[i+1].Query,
							"entries with equal count should be sorted by query")
					} else {
						assert.GreaterOrEqual(t, res.Entries[i].Count, res.Entries[i+1].Count,
							"entries should be sorted by count descending")
					}
				}
			},
		},
		{
			name: "zero count filtered out",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Len(1),
				).Return(client.Result[components.ValueCountsResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.ValueCountsResponse{
						AndCountResults: []float64{0},
					},
				}, nil)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("192.168.1.1"),
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Entries, 0)
			},
		},
		{
			name: "filters applied - location fields filtered out",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				// Only expect 1 count for IP, not for location fields (they're filtered)
				mockClient.EXPECT().GetValueCounts(
					gomock.Any(),
					mo.None[string](),
					mo.None[string](),
					gomock.Len(1), // Only IP, location fields are filtered
				).Return(client.Result[components.ValueCountsResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.ValueCountsResponse{
						AndCountResults: []float64{50},
					},
				}, nil)
				return mockClient
			},
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("192.168.1.1"),
					Location: &components.Location{
						City:    strPtr("San Francisco"),
						Country: strPtr("United States"),
					},
				},
			},
			rarityMin: 10,
			rarityMax: 100,
			assert: func(t *testing.T, res InvestigateHostResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Entries, 1, "location fields should be filtered out")

				entry := res.Entries[0]
				assert.Equal(t, int64(50), entry.Count)
				assert.Contains(t, entry.Query, "host.ip=", "only IP should remain after filtering")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := tc.client(ctrl)
			svc := New(mockClient)

			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}

			res, err := svc.InvestigateHost(ctx, tc.orgID, tc.host, tc.rarityMin, tc.rarityMax)
			tc.assert(t, res, err)
		})
	}
}
