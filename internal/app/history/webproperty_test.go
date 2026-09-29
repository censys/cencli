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
	"github.com/censys/cencli/internal/app/streaming"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

const testWebPropertyID = "example.com:443"

func endpointEvent(eventTime, path string) components.WebTimelineEventAsset {
	return components.WebTimelineEventAsset{Resource: components.WebTimelineEvent{
		EventTime:       strPtr(eventTime),
		EndpointScanned: &components.EndpointScanned{Scan: &components.EndpointScan{Path: strPtr(path)}},
	}}
}

func withBanner(e components.WebTimelineEventAsset, banner string) components.WebTimelineEventAsset {
	e.Resource.EndpointScanned.Scan.Banner = strPtr(banner)
	return e
}

func jarmEvent(eventTime string) components.WebTimelineEventAsset {
	return components.WebTimelineEventAsset{Resource: components.WebTimelineEvent{
		EventTime:   strPtr(eventTime),
		JarmScanned: &components.JarmScanned{},
	}}
}

func webTimelinePage(scannedTo time.Time, latency time.Duration, events ...components.WebTimelineEventAsset) client.Result[components.WebpropertyTimeline] {
	return client.Result[components.WebpropertyTimeline]{
		Data: &components.WebpropertyTimeline{Events: events, ScannedTo: scannedTo},
		Metadata: client.Metadata{
			Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
			Response: &http.Response{StatusCode: 200},
			Latency:  latency,
			Attempts: 1,
		},
	}
}

func eventTimes(events []*components.WebTimelineEvent) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, *e.EventTime)
	}
	return out
}

func TestGetWebPropertyHistory(t *testing.T) {
	fromTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	midTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	earlyTime := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	orgUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		orgID  mo.Option[identifiers.OrganizationID]
		ctx    func() context.Context
		stream bool
		assert func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, streamed []*components.WebTimelineEvent)
	}{
		{
			name: "success - single page ending at the fromTime bound",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
					Return(webTimelinePage(fromTime, 100*time.Millisecond,
						endpointEvent("2024-01-20T12:00:00Z", "/"),
						jarmEvent("2024-01-10T10:00:00Z"),
					), nil)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				assert.Equal(t, []string{"2024-01-20T12:00:00Z", "2024-01-10T10:00:00Z"}, eventTimes(res.Events))
				assert.NotNil(t, res.Events[0].EndpointScanned)
				assert.NotNil(t, res.Events[1].JarmScanned)
				assert.Equal(t, "GET", res.Meta.Method)
				assert.Equal(t, 200, res.Meta.Status)
				assert.Equal(t, uint64(1), res.Meta.PageCount)
				assert.Nil(t, res.PartialError)
			},
		},
		{
			name: "success - no events in the window",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
					Return(webTimelinePage(fromTime, 50*time.Millisecond), nil)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				assert.Empty(t, res.Events)
				assert.Equal(t, uint64(1), res.Meta.PageCount)
			},
		},
		{
			name: "success - a short page does not end pagination while the cursor is inside the window",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
						Return(webTimelinePage(midTime, 10*time.Millisecond, endpointEvent("2024-01-20T00:00:00Z", "/")), nil),
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, midTime).
						Return(webTimelinePage(fromTime, 10*time.Millisecond, endpointEvent("2024-01-10T00:00:00Z", "/")), nil),
				)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Equal(t, []string{"2024-01-20T00:00:00Z", "2024-01-10T00:00:00Z"}, eventTimes(res.Events))
				assert.Equal(t, uint64(2), res.Meta.PageCount)
			},
		},
		{
			name: "success - an empty page with the cursor inside the window keeps paging",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
						Return(webTimelinePage(midTime, 10*time.Millisecond), nil),
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, midTime).
						Return(webTimelinePage(fromTime, 10*time.Millisecond, endpointEvent("2024-01-10T00:00:00Z", "/")), nil),
				)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Equal(t, []string{"2024-01-10T00:00:00Z"}, eventTimes(res.Events))
				assert.Equal(t, uint64(2), res.Meta.PageCount)
			},
		},
		{
			name: "success - an event repeated across an empty page is returned once",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
						Return(webTimelinePage(midTime, 10*time.Millisecond, endpointEvent("2024-01-15T12:00:00Z", "/")), nil),
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, midTime).
						Return(webTimelinePage(earlyTime, 10*time.Millisecond), nil),
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, earlyTime).
						Return(webTimelinePage(fromTime, 10*time.Millisecond,
							endpointEvent("2024-01-15T12:00:00Z", "/"),
							endpointEvent("2024-01-02T00:00:00Z", "/"),
						), nil),
				)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Equal(t, []string{"2024-01-15T12:00:00Z", "2024-01-02T00:00:00Z"}, eventTimes(res.Events))
				assert.Equal(t, uint64(3), res.Meta.PageCount)
			},
		},
		{
			name: "success - stops when scannedTo is before the fromTime bound",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
					Return(webTimelinePage(fromTime.Add(-time.Hour), 10*time.Millisecond, endpointEvent("2024-01-20T00:00:00Z", "/")), nil)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Len(t, res.Events, 1)
				assert.Equal(t, uint64(1), res.Meta.PageCount)
			},
		},
		{
			name: "success - stops on the zero-time sentinel",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
					Return(webTimelinePage(time.Time{}, 10*time.Millisecond, endpointEvent("2024-01-20T00:00:00Z", "/")), nil)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Len(t, res.Events, 1)
				assert.Equal(t, uint64(1), res.Meta.PageCount)
			},
		},
		{
			name: "success - stops when the cursor does not move",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
						Return(webTimelinePage(midTime, 10*time.Millisecond, endpointEvent("2024-01-20T00:00:00Z", "/")), nil),
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, midTime).
						Return(webTimelinePage(midTime, 10*time.Millisecond, endpointEvent("2024-01-15T12:00:00Z", "/")), nil),
				)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Len(t, res.Events, 2)
				assert.Equal(t, uint64(2), res.Meta.PageCount)
				require.NotNil(t, res.PartialError)
				assert.Equal(t, "Incomplete History", res.PartialError.Title())
				assert.Contains(t, res.PartialError.Error(), "stopped advancing at 2024-01-15T12:00:00Z")
			},
		},
		{
			name: "partial - an empty first page whose cursor is newer than the start is reported, not repeated",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
					Return(webTimelinePage(toTime.Add(24*time.Hour), 10*time.Millisecond), nil).Times(1)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Empty(t, res.Events)
				assert.Equal(t, uint64(1), res.Meta.PageCount)
				require.NotNil(t, res.PartialError)
				assert.Equal(t, "Incomplete History", res.PartialError.Title())
				assert.NotContains(t, res.PartialError.Error(), "successfully retrieved")
			},
		},
		{
			name: "success - an event repeated within a page is returned once",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
					Return(webTimelinePage(fromTime, 10*time.Millisecond,
						endpointEvent("2024-01-20T00:00:00Z", "/"),
						endpointEvent("2024-01-20T00:00:00Z", "/"),
						endpointEvent("2024-01-10T00:00:00Z", "/"),
					), nil)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Equal(t, []string{"2024-01-20T00:00:00Z", "2024-01-10T00:00:00Z"}, eventTimes(res.Events))
			},
		},
		{
			name: "success - three pages",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
						Return(webTimelinePage(midTime, 10*time.Millisecond, endpointEvent("2024-01-25T00:00:00Z", "/")), nil),
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, midTime).
						Return(webTimelinePage(earlyTime, 10*time.Millisecond, endpointEvent("2024-01-10T00:00:00Z", "/")), nil),
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, earlyTime).
						Return(webTimelinePage(time.Time{}, 10*time.Millisecond, jarmEvent("2024-01-02T00:00:00Z")), nil),
				)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Equal(t, []string{"2024-01-25T00:00:00Z", "2024-01-10T00:00:00Z", "2024-01-02T00:00:00Z"}, eventTimes(res.Events))
				assert.Equal(t, uint64(3), res.Meta.PageCount)
			},
		},
		{
			name: "success - an event repeated at the page seam is returned once",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
						Return(webTimelinePage(midTime, 10*time.Millisecond,
							endpointEvent("2024-01-20T00:00:00Z", "/"),
							endpointEvent("2024-01-15T12:00:00Z", "/"),
						), nil),
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, midTime).
						Return(webTimelinePage(fromTime, 10*time.Millisecond,
							endpointEvent("2024-01-15T12:00:00Z", "/"),                        // seam duplicate
							endpointEvent("2024-01-15T12:00:00Z", "/login"),                   // same second, different endpoint
							jarmEvent("2024-01-15T12:00:00Z"),                                 // same second, different kind
							withBanner(endpointEvent("2024-01-15T12:00:00Z", "/"), "changed"), // same second and endpoint, different content
							endpointEvent("2024-01-10T00:00:00Z", "/"),
						), nil),
				)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				require.Len(t, res.Events, 6)
				assert.Equal(t, []string{
					"2024-01-20T00:00:00Z",
					"2024-01-15T12:00:00Z",
					"2024-01-15T12:00:00Z",
					"2024-01-15T12:00:00Z",
					"2024-01-15T12:00:00Z",
					"2024-01-10T00:00:00Z",
				}, eventTimes(res.Events))
				assert.Equal(t, "/login", *res.Events[2].EndpointScanned.Scan.Path)
				assert.NotNil(t, res.Events[3].JarmScanned)
				assert.Equal(t, "changed", *res.Events[4].EndpointScanned.Scan.Banner)
			},
		},
		{
			name:   "streaming - an event repeated at the page seam is emitted once",
			stream: true,
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
						Return(webTimelinePage(midTime, 10*time.Millisecond,
							endpointEvent("2024-01-20T00:00:00Z", "/"),
							endpointEvent("2024-01-15T12:00:00Z", "/"),
						), nil),
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, midTime).
						Return(webTimelinePage(fromTime, 10*time.Millisecond,
							endpointEvent("2024-01-15T12:00:00Z", "/"),
							endpointEvent("2024-01-10T00:00:00Z", "/"),
						), nil),
				)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, streamed []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Empty(t, res.Events) // streamed, not collected
				assert.Equal(t, []string{"2024-01-20T00:00:00Z", "2024-01-15T12:00:00Z", "2024-01-10T00:00:00Z"}, eventTimes(streamed))
			},
		},
		{
			name:  "success - with orgID",
			orgID: mo.Some(identifiers.NewOrganizationID(orgUUID)),
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.Some(orgUUID.String()), testWebPropertyID, fromTime, toTime).
					Return(webTimelinePage(fromTime, 10*time.Millisecond, endpointEvent("2024-01-20T00:00:00Z", "/")), nil)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Len(t, res.Events, 1)
			},
		},
		{
			name: "client structured error on first page",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				detail := "Web property not found"
				status := int64(404)
				m.EXPECT().WebPropertyTimeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.WebpropertyTimeline]{}, client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status}))
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.Error(t, err)
				var structuredErr client.ClientStructuredError
				require.ErrorAs(t, err, &structuredErr)
				assert.Equal(t, int64(404), structuredErr.StatusCode().MustGet())
				assert.Nil(t, res.Meta)
			},
		},
		{
			name: "client generic error on first page",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().WebPropertyTimeline(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.WebpropertyTimeline]{}, client.NewCensysClientGenericError(&sdkerrors.SDKError{Message: "Internal server error", StatusCode: 500}))
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "Internal server error")
			},
		},
		{
			name: "context cancelled before the first page",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl)
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.Error(t, err)
				require.ErrorIs(t, err, context.Canceled)
			},
		},
		{
			name: "error on second page - returns partial results",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
						Return(webTimelinePage(midTime, 10*time.Millisecond, endpointEvent("2024-01-20T00:00:00Z", "/")), nil),
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, midTime).
						Return(client.Result[components.WebpropertyTimeline]{}, client.NewCensysClientGenericError(&sdkerrors.SDKError{Message: "Internal server error", StatusCode: 500})),
				)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.Len(t, res.Events, 1, "first page is kept")
				require.NotNil(t, res.PartialError)
				assert.Contains(t, res.PartialError.Error(), "Internal server error")
				assert.Equal(t, uint64(2), res.Meta.PageCount)
			},
		},
		{
			name: "metadata reflects total latency across all pages",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				sleep := func(context.Context, mo.Option[string], string, time.Time, time.Time) {
					time.Sleep(50 * time.Millisecond)
				}
				gomock.InOrder(
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
						Return(webTimelinePage(midTime, 200*time.Millisecond, endpointEvent("2024-01-20T00:00:00Z", "/")), nil).Do(sleep),
					m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, midTime).
						Return(webTimelinePage(fromTime, 150*time.Millisecond, endpointEvent("2024-01-10T00:00:00Z", "/")), nil).Do(sleep),
				)
				return m
			},
			assert: func(t *testing.T, res WebPropertyHistoryResult, err cenclierrors.CencliError, _ []*components.WebTimelineEvent) {
				require.NoError(t, err)
				assert.GreaterOrEqual(t, res.Meta.Latency, 100*time.Millisecond, "total latency should be at least the sleep time")
				assert.Equal(t, uint64(2), res.Meta.PageCount)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))

			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}

			var emitter streaming.Emitter
			var items <-chan streaming.Item
			if tc.stream {
				emitter, items = streaming.NewChannelEmitter(64)
				ctx = streaming.WithEmitter(ctx, emitter)
			}

			res, err := svc.GetWebPropertyHistory(ctx, tc.orgID, mustWebPropertyID(testWebPropertyID), fromTime, toTime)

			var streamed []*components.WebTimelineEvent
			if emitter != nil {
				emitter.Close(nil)
				for item := range items {
					if item.Done {
						continue
					}
					streamed = append(streamed, item.Data.(*components.WebTimelineEvent))
				}
			}
			tc.assert(t, res, err, streamed)
		})
	}
}

func TestGetWebPropertyHistory_CancelledAfterFirstPage(t *testing.T) {
	fromTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	midTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	ctrl := gomock.NewController(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := mocks.NewMockClient(ctrl)
	m.EXPECT().WebPropertyTimeline(gomock.Any(), mo.None[string](), testWebPropertyID, fromTime, toTime).
		DoAndReturn(func(context.Context, mo.Option[string], string, time.Time, time.Time) (client.Result[components.WebpropertyTimeline], client.ClientError) {
			cancel()
			return webTimelinePage(midTime, 10*time.Millisecond, endpointEvent("2024-01-20T00:00:00Z", "/")), nil
		})

	res, err := New(m).GetWebPropertyHistory(ctx, mo.None[identifiers.OrganizationID](), mustWebPropertyID(testWebPropertyID), fromTime, toTime)

	require.NoError(t, err)
	assert.Len(t, res.Events, 1)
	require.NotNil(t, res.PartialError)
	require.ErrorIs(t, res.PartialError, context.Canceled)
}

func TestWebTimelineEventKey(t *testing.T) {
	testCases := []struct {
		name       string
		a, b       components.WebTimelineEvent
		expectSame bool
	}{
		{
			name:       "same endpoint event",
			a:          endpointEvent("2024-01-01T00:00:00Z", "/").Resource,
			b:          endpointEvent("2024-01-01T00:00:00Z", "/").Resource,
			expectSame: true,
		},
		{
			name: "different time",
			a:    endpointEvent("2024-01-01T00:00:00Z", "/").Resource,
			b:    endpointEvent("2024-01-01T00:00:01Z", "/").Resource,
		},
		{
			name: "different endpoint path",
			a:    endpointEvent("2024-01-01T00:00:00Z", "/").Resource,
			b:    endpointEvent("2024-01-01T00:00:00Z", "/admin").Resource,
		},
		{
			name: "endpoint vs jarm at the same time",
			a:    endpointEvent("2024-01-01T00:00:00Z", "/").Resource,
			b:    jarmEvent("2024-01-01T00:00:00Z").Resource,
		},
		{
			name: "same time, kind, and endpoint but different content",
			a:    endpointEvent("2024-01-01T00:00:00Z", "/").Resource,
			b:    withBanner(endpointEvent("2024-01-01T00:00:00Z", "/"), "changed").Resource,
		},
		{
			name:       "empty events",
			expectSame: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ka, kb := webTimelineEventKey(&tc.a), webTimelineEventKey(&tc.b)
			if tc.expectSame {
				assert.Equal(t, ka, kb)
			} else {
				assert.NotEqual(t, ka, kb)
			}
		})
	}
}

func mustWebPropertyID(id string) assets.WebPropertyID {
	webPropID, err := assets.NewWebPropertyID(id, assets.DefaultWebPropertyPort)
	if err != nil {
		panic(err)
	}
	return webPropID
}
