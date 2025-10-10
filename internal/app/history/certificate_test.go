package history

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"
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
)

func TestGetCertificateHistory(t *testing.T) {
	// Test time boundaries
	fromTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	// Valid SHA256 certificate fingerprints for testing
	cert1 := "fb444eb8e68437bae06232b9f5091bccff62a768ca09e92eb5c9c2cef1d9e5d5"
	cert2 := "3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf"

	testCases := []struct {
		name          string
		client        func(ctrl *gomock.Controller) client.Client
		orgID         mo.Option[identifiers.OrganizationID]
		certificateID assets.CertificateID
		fromTime      time.Time
		toTime        time.Time
		ctx           func() context.Context
		assert        func(t *testing.T, res CertificateHistoryResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - single page with results",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					mo.None[string](),
					cert1,
					mo.Some(fromTime),
					mo.Some(toTime),
					mo.None[int](),
					mo.None[string](),
					mo.None[int64](),
					mo.None[string](),
				).Return(client.Result[components.HostObservationResponse]{
					Data: &components.HostObservationResponse{
						Ranges: []components.HostObservationRange{
							{IP: "8.8.8.8"},
							{IP: "1.1.1.1"},
						},
						NextPageToken: nil,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)
				return mockClient
			},
			certificateID: mustCertificateID(cert1),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res CertificateHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Ranges, 2)
				assert.Equal(t, "8.8.8.8", res.Ranges[0].IP)
				assert.Equal(t, "1.1.1.1", res.Ranges[1].IP)
				assert.Equal(t, "GET", res.Meta.Method)
				assert.Equal(t, 200, res.Meta.Status)
				assert.Equal(t, uint64(1), res.Meta.PageCount)
			},
		},
		{
			name: "success - single page with no results",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					mo.None[string](),
					cert1,
					mo.Some(fromTime),
					mo.Some(toTime),
					mo.None[int](),
					mo.None[string](),
					mo.None[int64](),
					mo.None[string](),
				).Return(client.Result[components.HostObservationResponse]{
					Data: &components.HostObservationResponse{
						Ranges:        []components.HostObservationRange{},
						NextPageToken: nil,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  50 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)
				return mockClient
			},
			certificateID: mustCertificateID(cert1),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res CertificateHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Empty(t, res.Ranges)
				assert.Equal(t, uint64(1), res.Meta.PageCount)
			},
		},
		{
			name: "success - multiple pages",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// First page
				token1 := "page2token"
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					mo.None[string](),
					cert2,
					mo.Some(fromTime),
					mo.Some(toTime),
					mo.None[int](),
					mo.None[string](),
					mo.None[int64](),
					mo.None[string](),
				).Return(client.Result[components.HostObservationResponse]{
					Data: &components.HostObservationResponse{
						Ranges: []components.HostObservationRange{
							{IP: "192.168.1.1"},
							{IP: "192.168.1.2"},
						},
						NextPageToken: &token1,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  150 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				// Second page
				token2 := "page3token"
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					mo.None[string](),
					cert2,
					mo.Some(fromTime),
					mo.Some(toTime),
					mo.None[int](),
					mo.None[string](),
					mo.None[int64](),
					mo.Some(token1),
				).Return(client.Result[components.HostObservationResponse]{
					Data: &components.HostObservationResponse{
						Ranges: []components.HostObservationRange{
							{IP: "10.0.0.1"},
							{IP: "10.0.0.2"},
						},
						NextPageToken: &token2,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  120 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				// Third page (final)
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					mo.None[string](),
					cert2,
					mo.Some(fromTime),
					mo.Some(toTime),
					mo.None[int](),
					mo.None[string](),
					mo.None[int64](),
					mo.Some(token2),
				).Return(client.Result[components.HostObservationResponse]{
					Data: &components.HostObservationResponse{
						Ranges: []components.HostObservationRange{
							{IP: "172.16.0.1"},
						},
						NextPageToken: nil,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				return mockClient
			},
			certificateID: mustCertificateID(cert2),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res CertificateHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Ranges, 5, "should have all ranges from all pages")
				assert.Equal(t, "192.168.1.1", res.Ranges[0].IP)
				assert.Equal(t, "192.168.1.2", res.Ranges[1].IP)
				assert.Equal(t, "10.0.0.1", res.Ranges[2].IP)
				assert.Equal(t, "10.0.0.2", res.Ranges[3].IP)
				assert.Equal(t, "172.16.0.1", res.Ranges[4].IP)
				assert.Equal(t, uint64(3), res.Meta.PageCount)
			},
		},
		{
			name: "success - with orgID",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					mo.Some("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
					cert1,
					mo.Some(fromTime),
					mo.Some(toTime),
					mo.None[int](),
					mo.None[string](),
					mo.None[int64](),
					mo.None[string](),
				).Return(client.Result[components.HostObservationResponse]{
					Data: &components.HostObservationResponse{
						Ranges: []components.HostObservationRange{
							{IP: "8.8.8.8"},
						},
						NextPageToken: nil,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)
				return mockClient
			},
			orgID:         mo.Some(identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))),
			certificateID: mustCertificateID(cert1),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res CertificateHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Ranges, 1)
			},
		},
		{
			name: "client structured error",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				detail := "Certificate not found"
				status := int64(404)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
					Detail: &detail,
					Status: &status,
				})
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(client.Result[components.HostObservationResponse]{}, structuredErr)
				return mockClient
			},
			certificateID: mustCertificateID(cert1),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res CertificateHistoryResult, err cenclierrors.CencliError) {
				require.NotNil(t, err)
				var cencliErr client.ClientStructuredError
				require.ErrorAs(t, err, &cencliErr)
				assert.Contains(t, err.Error(), "Certificate not found")
				assert.Contains(t, err.Error(), "404")
			},
		},
		{
			name: "client generic error",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "Rate limit exceeded",
					StatusCode: 429,
					Body:       "Too many requests",
				})
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(client.Result[components.HostObservationResponse]{}, genericErr)
				return mockClient
			},
			certificateID: mustCertificateID(cert1),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res CertificateHistoryResult, err cenclierrors.CencliError) {
				require.NotNil(t, err)
				var cencliErr client.ClientGenericError
				require.ErrorAs(t, err, &cencliErr)
				assert.Contains(t, err.Error(), "Rate limit exceeded")
				assert.Contains(t, err.Error(), "429")
			},
		},
		{
			name: "client unknown error",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				unknownErr := client.NewClientError(errors.New("network timeout"))
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(client.Result[components.HostObservationResponse]{}, unknownErr)
				return mockClient
			},
			certificateID: mustCertificateID(cert1),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res CertificateHistoryResult, err cenclierrors.CencliError) {
				require.NotNil(t, err)
				var cencliErr client.ClientError
				require.ErrorAs(t, err, &cencliErr)
				assert.Contains(t, err.Error(), "network timeout")
			},
		},
		{
			name: "context canceled",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(client.Result[components.HostObservationResponse]{}, client.NewClientError(context.Canceled))
				return mockClient
			},
			certificateID: mustCertificateID(cert1),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res CertificateHistoryResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "the operation's context was cancelled before it completed")
			},
		},
		{
			name: "error on second page - returns partial results",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// First page succeeds
				token1 := "page2token"
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					mo.None[string](),
					cert1,
					mo.Some(fromTime),
					mo.Some(toTime),
					mo.None[int](),
					mo.None[string](),
					mo.None[int64](),
					mo.None[string](),
				).Return(client.Result[components.HostObservationResponse]{
					Data: &components.HostObservationResponse{
						Ranges: []components.HostObservationRange{
							{IP: "8.8.8.8"},
						},
						NextPageToken: &token1,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				// Second page fails
				genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "Internal server error",
					StatusCode: 500,
					Body:       "Server error",
				})
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					mo.None[string](),
					cert1,
					mo.Some(fromTime),
					mo.Some(toTime),
					mo.None[int](),
					mo.None[string](),
					mo.None[int64](),
					mo.Some(token1),
				).Return(client.Result[components.HostObservationResponse]{}, genericErr)

				return mockClient
			},
			certificateID: mustCertificateID(cert1),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res CertificateHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Ranges, 1, "should return partial results from first page")
				require.NotNil(t, res.PartialError, "should have partial error")
				assert.Contains(t, res.PartialError.Error(), "Internal server error")
			},
		},
		{
			name: "metadata reflects total latency across all pages",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// First page
				token1 := "page2token"
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					mo.None[string](),
					cert1,
					mo.Some(fromTime),
					mo.Some(toTime),
					mo.None[int](),
					mo.None[string](),
					mo.None[int64](),
					mo.None[string](),
				).Return(client.Result[components.HostObservationResponse]{
					Data: &components.HostObservationResponse{
						Ranges: []components.HostObservationRange{
							{IP: "8.8.8.8"},
						},
						NextPageToken: &token1,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  200 * time.Millisecond,
						Attempts: 2,
					},
				}, nil).Do(func(ctx context.Context, orgID mo.Option[string], certificateID string, startTime, endTime mo.Option[time.Time], port mo.Option[int], protocol mo.Option[string], pageSize mo.Option[int64], pageToken mo.Option[string]) {
					time.Sleep(50 * time.Millisecond) // Simulate some actual time passing
				})

				// Second page
				mockClient.EXPECT().GetHostObservationsWithCertificate(
					gomock.Any(),
					mo.None[string](),
					cert1,
					mo.Some(fromTime),
					mo.Some(toTime),
					mo.None[int](),
					mo.None[string](),
					mo.None[int64](),
					mo.Some(token1),
				).Return(client.Result[components.HostObservationResponse]{
					Data: &components.HostObservationResponse{
						Ranges: []components.HostObservationRange{
							{IP: "1.1.1.1"},
						},
						NextPageToken: nil,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  150 * time.Millisecond,
						Attempts: 1,
					},
				}, nil).Do(func(ctx context.Context, orgID mo.Option[string], certificateID string, startTime, endTime mo.Option[time.Time], port mo.Option[int], protocol mo.Option[string], pageSize mo.Option[int64], pageToken mo.Option[string]) {
					time.Sleep(50 * time.Millisecond) // Simulate some actual time passing
				})

				return mockClient
			},
			certificateID: mustCertificateID(cert1),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res CertificateHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Ranges, 2)
				// Latency should reflect total time, not individual request latencies
				assert.GreaterOrEqual(t, res.Meta.Latency, 100*time.Millisecond, "total latency should be at least the sleep time")
				assert.Equal(t, uint64(2), res.Meta.PageCount)
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

			res, err := svc.GetCertificateHistory(ctx, tc.orgID, tc.certificateID, tc.fromTime, tc.toTime)
			tc.assert(t, res, err)
		})
	}
}

func mustCertificateID(fingerprint string) assets.CertificateID {
	certID, err := assets.NewCertificateFingerprint(fingerprint)
	if err != nil {
		panic(err)
	}
	return certID
}
