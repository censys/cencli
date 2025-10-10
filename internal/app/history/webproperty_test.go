package history

import (
	"context"
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

func TestGetWebPropertyHistory(t *testing.T) {
	// Test time boundaries - short range for day-by-day iteration
	fromTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC) // 3 days total
	day1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name          string
		client        func(ctrl *gomock.Controller) client.Client
		orgID         mo.Option[identifiers.OrganizationID]
		webPropertyID assets.WebPropertyID
		fromTime      time.Time
		toTime        time.Time
		ctx           func() context.Context
		assert        func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - all days have data",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// Day 1
				mockClient.EXPECT().GetWebProperties(
					gomock.Any(),
					mo.None[string](),
					[]string{"example.com:443"},
					mo.Some(day1),
				).Return(client.Result[[]components.Webproperty]{
					Data: &[]components.Webproperty{
						{
							Hostname:  strPtr("example.com"),
							Port:      intPtr(443),
							Endpoints: []components.EndpointScanState{{}},
						},
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				// Day 2
				mockClient.EXPECT().GetWebProperties(
					gomock.Any(),
					mo.None[string](),
					[]string{"example.com:443"},
					mo.Some(day2),
				).Return(client.Result[[]components.Webproperty]{
					Data: &[]components.Webproperty{
						{
							Hostname: strPtr("example.com"),
							Port:     intPtr(443),
							Cert:     &components.Certificate{},
						},
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				// Day 3
				mockClient.EXPECT().GetWebProperties(
					gomock.Any(),
					mo.None[string](),
					[]string{"example.com:443"},
					mo.Some(day3),
				).Return(client.Result[[]components.Webproperty]{
					Data: &[]components.Webproperty{
						{
							Hostname: strPtr("example.com"),
							Port:     intPtr(443),
							TLS:      &components.TLS{},
						},
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				return mockClient
			},
			webPropertyID: mustWebPropertyID("example.com:443"),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Snapshots, 3)

				// All snapshots should exist
				assert.True(t, res.Snapshots[0].Exists)
				assert.True(t, res.Snapshots[1].Exists)
				assert.True(t, res.Snapshots[2].Exists)

				// Check times
				assert.Equal(t, day1, res.Snapshots[0].Time)
				assert.Equal(t, day2, res.Snapshots[1].Time)
				assert.Equal(t, day3, res.Snapshots[2].Time)

				assert.Equal(t, uint64(3), res.Meta.PageCount)
			},
		},
		{
			name: "success - some days have no data",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// Day 1 - has data
				mockClient.EXPECT().GetWebProperties(
					gomock.Any(),
					mo.None[string](),
					[]string{"test.com:80"},
					mo.Some(day1),
				).Return(client.Result[[]components.Webproperty]{
					Data: &[]components.Webproperty{
						{
							Hostname: strPtr("test.com"),
							Port:     intPtr(80),
							Software: []components.Attribute{{}},
						},
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				// Day 2 - only hostname/port (no meaningful data)
				mockClient.EXPECT().GetWebProperties(
					gomock.Any(),
					mo.None[string](),
					[]string{"test.com:80"},
					mo.Some(day2),
				).Return(client.Result[[]components.Webproperty]{
					Data: &[]components.Webproperty{
						{
							Hostname: strPtr("test.com"),
							Port:     intPtr(80),
						},
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				// Day 3 - error (doesn't exist)
				mockClient.EXPECT().GetWebProperties(
					gomock.Any(),
					mo.None[string](),
					[]string{"test.com:80"},
					mo.Some(day3),
				).Return(client.Result[[]components.Webproperty]{}, client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "Not found",
					StatusCode: 404,
				}))

				return mockClient
			},
			webPropertyID: mustWebPropertyID("test.com:80"),
			fromTime:      fromTime,
			toTime:        toTime,
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Snapshots, 3)

				// Day 1 exists
				assert.True(t, res.Snapshots[0].Exists)
				assert.NotNil(t, res.Snapshots[0].Data)

				// Day 2 doesn't exist (only hostname/port)
				assert.False(t, res.Snapshots[1].Exists)
				assert.NotNil(t, res.Snapshots[1].Data)

				// Day 3 doesn't exist (error)
				assert.False(t, res.Snapshots[2].Exists)
				assert.Nil(t, res.Snapshots[2].Data)
			},
		},
		{
			name: "success - single day",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				mockClient.EXPECT().GetWebProperties(
					gomock.Any(),
					mo.None[string](),
					[]string{"single.com:443"},
					mo.Some(day1),
				).Return(client.Result[[]components.Webproperty]{
					Data: &[]components.Webproperty{
						{
							Hostname: strPtr("single.com"),
							Port:     intPtr(443),
							Jarm:     &components.JarmScan{},
						},
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  50 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				return mockClient
			},
			webPropertyID: mustWebPropertyID("single.com:443"),
			fromTime:      day1,
			toTime:        day1,
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Snapshots, 1)
				assert.True(t, res.Snapshots[0].Exists)
				assert.Equal(t, day1, res.Snapshots[0].Time)
				assert.Equal(t, uint64(1), res.Meta.PageCount)
			},
		},
		{
			name: "success - with orgID",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				mockClient.EXPECT().GetWebProperties(
					gomock.Any(),
					mo.Some("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
					[]string{"org.com:443"},
					mo.Some(day1),
				).Return(client.Result[[]components.Webproperty]{
					Data: &[]components.Webproperty{
						{
							Hostname: strPtr("org.com"),
							Port:     intPtr(443),
							Hardware: []components.Attribute{{}},
						},
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				return mockClient
			},
			orgID:         mo.Some(identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))),
			webPropertyID: mustWebPropertyID("org.com:443"),
			fromTime:      day1,
			toTime:        day1,
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Snapshots, 1)
				assert.True(t, res.Snapshots[0].Exists)
			},
		},
		{
			name: "success - empty results",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				mockClient.EXPECT().GetWebProperties(
					gomock.Any(),
					mo.None[string](),
					[]string{"empty.com:443"},
					mo.Some(day1),
				).Return(client.Result[[]components.Webproperty]{
					Data: &[]components.Webproperty{},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  50 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				return mockClient
			},
			webPropertyID: mustWebPropertyID("empty.com:443"),
			fromTime:      day1,
			toTime:        day1,
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Snapshots, 1)
				assert.False(t, res.Snapshots[0].Exists)
				assert.Nil(t, res.Snapshots[0].Data)
			},
		},
		{
			name: "success - metadata reflects total latency",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// Day 1
				mockClient.EXPECT().GetWebProperties(
					gomock.Any(),
					mo.None[string](),
					[]string{"latency.com:443"},
					mo.Some(day1),
				).Return(client.Result[[]components.Webproperty]{
					Data: &[]components.Webproperty{
						{
							Hostname: strPtr("latency.com"),
							Port:     intPtr(443),
							Vulns:    []components.Vuln{{}},
						},
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil).Do(func(ctx context.Context, orgID mo.Option[string], webPropertyIDs []string, atTime mo.Option[time.Time]) {
					time.Sleep(30 * time.Millisecond)
				})

				// Day 2
				mockClient.EXPECT().GetWebProperties(
					gomock.Any(),
					mo.None[string](),
					[]string{"latency.com:443"},
					mo.Some(day2),
				).Return(client.Result[[]components.Webproperty]{
					Data: &[]components.Webproperty{
						{
							Hostname:  strPtr("latency.com"),
							Port:      intPtr(443),
							Exposures: []components.Risk{{}},
						},
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil).Do(func(ctx context.Context, orgID mo.Option[string], webPropertyIDs []string, atTime mo.Option[time.Time]) {
					time.Sleep(30 * time.Millisecond)
				})

				return mockClient
			},
			webPropertyID: mustWebPropertyID("latency.com:443"),
			fromTime:      day1,
			toTime:        day2,
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Snapshots, 2)
				// Latency should reflect total time
				assert.GreaterOrEqual(t, res.Meta.Latency, 60*time.Millisecond)
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

			res, err := svc.GetWebPropertyHistory(ctx, tc.orgID, tc.webPropertyID, tc.fromTime, tc.toTime)
			tc.assert(t, res, err)
		})
	}
}

// Helper function for web property IDs
func mustWebPropertyID(id string) assets.WebPropertyID {
	webPropID, err := assets.NewWebPropertyID(id, assets.DefaultWebPropertyPort)
	if err != nil {
		panic(err)
	}
	return webPropID
}

func intPtr(i int) *int {
	return &i
}

// TestGetWebPropertyHistory_PartialError tests partial error handling
func TestGetWebPropertyHistory_PartialError(t *testing.T) {
	fromTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	day1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	t.Run("error on second day returns partial results", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		webPropID := mustWebPropertyID("example.com:443")

		// Web property history iterates day by day from fromTime to toTime
		// First day succeeds - include meaningful data (endpoints) so Exists is true
		mockClient.EXPECT().GetWebProperties(
			gomock.Any(),
			mo.None[string](),
			[]string{"example.com:443"},
			mo.Some(day1),
		).Return(client.Result[[]components.Webproperty]{
			Data: &[]components.Webproperty{{
				Hostname:  strPtr("example.com"),
				Port:      intPtr(443),
				Endpoints: []components.EndpointScanState{{}},
			}},
			Metadata: client.Metadata{
				Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
				Response: &http.Response{StatusCode: 200},
				Latency:  100 * time.Millisecond,
			},
		}, nil)

		// Second day fails
		mockClient.EXPECT().GetWebProperties(
			gomock.Any(),
			mo.None[string](),
			[]string{"example.com:443"},
			mo.Some(day2),
		).Return(client.Result[[]components.Webproperty]{}, client.NewClientError(
			&sdkerrors.SDKError{Message: "Internal server error", StatusCode: 500, Body: "Server error"},
		))

		svc := New(mockClient)
		res, err := svc.GetWebPropertyHistory(context.Background(), mo.None[identifiers.OrganizationID](), webPropID, fromTime, toTime)

		require.NoError(t, err)
		require.NotNil(t, res.PartialError, "should have partial error")
		require.Contains(t, res.PartialError.Error(), "Internal server error")
		require.GreaterOrEqual(t, len(res.Snapshots), 1, "should return at least first day's snapshot")
		assert.True(t, res.Snapshots[0].Exists)
	})

	t.Run("context cancelled after first day returns partial results", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mocks.NewMockClient(ctrl)
		webPropID := mustWebPropertyID("example.com:443")

		ctx, cancel := context.WithCancel(context.Background())

		// First day succeeds, then cancel - include meaningful data (endpoints) so Exists is true
		mockClient.EXPECT().GetWebProperties(
			gomock.Any(),
			mo.None[string](),
			[]string{"example.com:443"},
			mo.Some(day1),
		).DoAndReturn(func(ctx context.Context, orgID mo.Option[string], ids []string, atTime mo.Option[time.Time]) (client.Result[[]components.Webproperty], client.ClientError) {
			defer cancel() // Cancel after first day
			return client.Result[[]components.Webproperty]{
				Data: &[]components.Webproperty{{
					Hostname:  strPtr("example.com"),
					Port:      intPtr(443),
					Endpoints: []components.EndpointScanState{{}},
				}},
				Metadata: client.Metadata{
					Request:  &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
					Response: &http.Response{StatusCode: 200},
					Latency:  100 * time.Millisecond,
				},
			}, nil
		})

		svc := New(mockClient)
		res, err := svc.GetWebPropertyHistory(ctx, mo.None[identifiers.OrganizationID](), webPropID, fromTime, toTime)

		require.NoError(t, err)
		require.NotNil(t, res.PartialError)
		require.ErrorIs(t, res.PartialError, context.Canceled)
		require.GreaterOrEqual(t, len(res.Snapshots), 1, "should return at least first day's snapshot")
	})
}
