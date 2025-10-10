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

func TestGetHostHistory(t *testing.T) {
	// Test time boundaries
	fromTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	midTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name     string
		client   func(ctrl *gomock.Controller) client.Client
		orgID    mo.Option[identifiers.OrganizationID]
		host     assets.HostID
		fromTime time.Time
		toTime   time.Time
		ctx      func() context.Context
		assert   func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - single page with fewer than 100 events",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"8.8.8.8",
					fromTime,
					toTime,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events: []components.HostTimelineEventAsset{
							{Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-15T12:00:00Z")}},
							{Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-10T10:00:00Z")}},
						},
						ScannedTo: fromTime,
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
			host:     mustHostID("8.8.8.8"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Events, 2)
				assert.Equal(t, "2024-01-15T12:00:00Z", *res.Events[0].EventTime)
				assert.Equal(t, "2024-01-10T10:00:00Z", *res.Events[1].EventTime)
				assert.Equal(t, "GET", res.Meta.Method)
				assert.Equal(t, 200, res.Meta.Status)
				assert.Equal(t, uint64(1), res.Meta.PageCount)
			},
		},
		{
			name: "success - single page with no events",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"8.8.8.8",
					fromTime,
					toTime,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    []components.HostTimelineEventAsset{},
						ScannedTo: fromTime,
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
			host:     mustHostID("8.8.8.8"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Empty(t, res.Events)
				assert.Equal(t, uint64(1), res.Meta.PageCount)
			},
		},
		{
			name: "success - multiple pages with exactly 100 events per page",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// First page - exactly 100 events
				events1 := make([]components.HostTimelineEventAsset, 100)
				for i := 0; i < 100; i++ {
					events1[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-20T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"192.168.1.1",
					fromTime,
					toTime,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events1,
						ScannedTo: midTime,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  150 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				// Second page - 50 events (less than 100, so pagination stops)
				events2 := make([]components.HostTimelineEventAsset, 50)
				for i := 0; i < 50; i++ {
					events2[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-10T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"192.168.1.1",
					fromTime,
					midTime, // Uses scannedTo from previous page
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events2,
						ScannedTo: fromTime,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  120 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				return mockClient
			},
			host:     mustHostID("192.168.1.1"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Events, 150, "should have all events from both pages")
				assert.Equal(t, uint64(2), res.Meta.PageCount)
				// Verify latency is measured (non-zero) and reflects total time
				assert.Greater(t, res.Meta.Latency, time.Duration(0))
			},
		},
		{
			name: "success - pagination stops when scannedTo reaches fromTime boundary",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// First page - exactly 100 events
				events1 := make([]components.HostTimelineEventAsset, 100)
				for i := 0; i < 100; i++ {
					events1[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-20T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"10.0.0.1",
					fromTime,
					toTime,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events1,
						ScannedTo: fromTime.Add(-1 * time.Hour), // scannedTo is before fromTime
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
			host:     mustHostID("10.0.0.1"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Events, 100)
				assert.Equal(t, uint64(1), res.Meta.PageCount, "should stop after first page when boundary reached")
			},
		},
		{
			name: "success - pagination stops when scannedTo equals fromTime boundary",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				events := make([]components.HostTimelineEventAsset, 100)
				for i := 0; i < 100; i++ {
					events[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-15T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"10.0.0.2",
					fromTime,
					toTime,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events,
						ScannedTo: fromTime, // scannedTo equals fromTime
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
			host:     mustHostID("10.0.0.2"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Events, 100)
				assert.Equal(t, uint64(1), res.Meta.PageCount)
			},
		},
		{
			name: "success - pagination stops when scannedTo is zero",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				events := make([]components.HostTimelineEventAsset, 100)
				for i := 0; i < 100; i++ {
					events[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-15T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"10.0.0.3",
					fromTime,
					toTime,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events,
						ScannedTo: time.Time{}, // Zero time
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
			host:     mustHostID("10.0.0.3"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Events, 100)
				assert.Equal(t, uint64(1), res.Meta.PageCount, "should stop when scannedTo is zero")
			},
		},
		{
			name: "success - three pages of pagination",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				time1 := time.Date(2024, 1, 25, 0, 0, 0, 0, time.UTC)
				time2 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
				time3 := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)

				// Page 1
				events1 := make([]components.HostTimelineEventAsset, 100)
				for i := 0; i < 100; i++ {
					events1[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-25T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"172.16.0.1",
					fromTime,
					toTime,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events1,
						ScannedTo: time1,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				// Page 2
				events2 := make([]components.HostTimelineEventAsset, 100)
				for i := 0; i < 100; i++ {
					events2[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-15T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"172.16.0.1",
					fromTime,
					time1,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events2,
						ScannedTo: time2,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				// Page 3
				events3 := make([]components.HostTimelineEventAsset, 100)
				for i := 0; i < 100; i++ {
					events3[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-05T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"172.16.0.1",
					fromTime,
					time2,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events3,
						ScannedTo: time3,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
				}, nil)

				// Page 4 - final page with fewer events
				events4 := make([]components.HostTimelineEventAsset, 25)
				for i := 0; i < 25; i++ {
					events4[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-02T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"172.16.0.1",
					fromTime,
					time3,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events4,
						ScannedTo: fromTime,
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
			host:     mustHostID("172.16.0.1"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Events, 325, "should have all events from all pages")
				assert.Equal(t, uint64(4), res.Meta.PageCount)
			},
		},
		{
			name: "success - with orgID",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.Some("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
					"8.8.8.8",
					fromTime,
					toTime,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events: []components.HostTimelineEventAsset{
							{Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-15T12:00:00Z")}},
						},
						ScannedTo: fromTime,
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
			orgID:    mo.Some(identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))),
			host:     mustHostID("8.8.8.8"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Events, 1)
			},
		},
		{
			name: "client structured error",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				detail := "Host not found"
				status := int64(404)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
					Detail: &detail,
					Status: &status,
				})
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(client.Result[components.HostTimeline]{}, structuredErr)
				return mockClient
			},
			host:     mustHostID("8.8.8.8"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.NotNil(t, err)
				var cencliErr client.ClientStructuredError
				require.ErrorAs(t, err, &cencliErr)
				assert.Contains(t, err.Error(), "Host not found")
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
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(client.Result[components.HostTimeline]{}, genericErr)
				return mockClient
			},
			host:     mustHostID("8.8.8.8"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
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
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(client.Result[components.HostTimeline]{}, unknownErr)
				return mockClient
			},
			host:     mustHostID("8.8.8.8"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
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
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(client.Result[components.HostTimeline]{}, client.NewClientError(context.Canceled))
				return mockClient
			},
			host:     mustHostID("8.8.8.8"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "the operation's context was cancelled before it completed")
			},
		},
		{
			name: "error on second page - returns partial results",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// First page succeeds
				events1 := make([]components.HostTimelineEventAsset, 100)
				for i := 0; i < 100; i++ {
					events1[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-20T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"8.8.8.8",
					fromTime,
					toTime,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events1,
						ScannedTo: midTime,
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
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"8.8.8.8",
					fromTime,
					midTime,
				).Return(client.Result[components.HostTimeline]{}, genericErr)

				return mockClient
			},
			host:     mustHostID("8.8.8.8"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Events, 100, "should return partial results from first page")
				require.NotNil(t, res.PartialError, "should have partial error")
				assert.Contains(t, res.PartialError.Error(), "Internal server error")
			},
		},
		{
			name: "metadata reflects total latency across all pages",
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// First page
				events1 := make([]components.HostTimelineEventAsset, 100)
				for i := 0; i < 100; i++ {
					events1[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-20T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"8.8.8.8",
					fromTime,
					toTime,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events1,
						ScannedTo: midTime,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  200 * time.Millisecond,
						Attempts: 2,
					},
				}, nil).Do(func(ctx context.Context, orgID mo.Option[string], hostID string, fromTime, toTime time.Time) {
					time.Sleep(50 * time.Millisecond) // Simulate some actual time passing
				})

				// Second page
				events2 := make([]components.HostTimelineEventAsset, 50)
				for i := 0; i < 50; i++ {
					events2[i] = components.HostTimelineEventAsset{
						Resource: components.HostTimelineEvent{EventTime: strPtr("2024-01-10T00:00:00Z")},
					}
				}
				mockClient.EXPECT().HostTimeline(
					gomock.Any(),
					mo.None[string](),
					"8.8.8.8",
					fromTime,
					midTime,
				).Return(client.Result[components.HostTimeline]{
					Data: &components.HostTimeline{
						Events:    events2,
						ScannedTo: fromTime,
					},
					Metadata: client.Metadata{
						Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
						Response: &http.Response{StatusCode: 200},
						Latency:  150 * time.Millisecond,
						Attempts: 1,
					},
				}, nil).Do(func(ctx context.Context, orgID mo.Option[string], hostID string, fromTime, toTime time.Time) {
					time.Sleep(50 * time.Millisecond) // Simulate some actual time passing
				})

				return mockClient
			},
			host:     mustHostID("8.8.8.8"),
			fromTime: fromTime,
			toTime:   toTime,
			assert: func(t *testing.T, res HostHistoryResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Events, 150)
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

			res, err := svc.GetHostHistory(ctx, tc.orgID, tc.host, tc.fromTime, tc.toTime)
			tc.assert(t, res, err)
		})
	}
}

// Helper functions
func mustHostID(ip string) assets.HostID {
	hostID, err := assets.NewHostID(ip)
	if err != nil {
		panic(err)
	}
	return hostID
}

func strPtr(s string) *string {
	return &s
}
