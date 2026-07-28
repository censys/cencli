package tags

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

const (
	testTagUUID = "a6217129-be72-4b02-a42c-9c431574e524"
	testOpUUID  = "d421a231-eb5e-4927-a0be-8aa749eb731c"
)

func operationsPage(ids []string, status components.TagOperationStatus, total int64, nextToken string,
) client.Result[components.TagOperationsList] {
	ops := make([]components.TagOperation, 0, len(ids))
	for _, id := range ids {
		ops = append(ops, components.TagOperation{
			ID:     id,
			TagID:  testTagUUID,
			Type:   components.TagOperationTypeBulkCreate,
			Status: status,
		})
	}
	list := &components.TagOperationsList{Operations: ops, TotalSize: total}
	if nextToken != "" {
		list.NextPageToken = strPtr(nextToken)
	}
	return client.Result[components.TagOperationsList]{Metadata: okMeta(), Data: list}
}

func operationResult(status components.TagOperationStatus, processed int64) client.Result[components.TagOperation] {
	return client.Result[components.TagOperation]{
		Metadata: okMeta(),
		Data: &components.TagOperation{
			ID:             testOpUUID,
			TagID:          testTagUUID,
			TagName:        "my-tag",
			Type:           components.TagOperationTypeBulkCreate,
			Status:         status,
			ProcessedCount: processed,
			TotalCount:     100,
		},
	}
}

func TestTagsService_ListOperations(t *testing.T) {
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params OperationsParams
		assert func(t *testing.T, res OperationsResult, err cenclierrors.CencliError)
	}{
		{
			name: "no tag lists org-wide without resolving anything",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				// Org-wide listing must never trigger a name lookup.
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().ListTagOperations(gomock.Any(), client.ListTagOperationsRequest{TagID: "-"}).
					Return(operationsPage([]string{"op-1"}, components.TagOperationStatusRunning, 1, ""), nil)
				return m
			},
			assert: func(t *testing.T, res OperationsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Operations, 1)
				require.Equal(t, "op-1", res.Operations[0].ID)
				require.Equal(t, "running", res.Operations[0].Status)
				require.Equal(t, "bulk_create", res.Operations[0].Type)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "UUID tag skips resolution",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				// The durable guard: a UUID must cost zero lookup requests.
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().ListTagOperations(gomock.Any(), client.ListTagOperationsRequest{TagID: testTagUUID}).
					Return(operationsPage([]string{"op-1"}, components.TagOperationStatusSucceeded, 1, ""), nil)
				return m
			},
			params: OperationsParams{TagID: mo.Some(identifiers.NewTagID(testTagUUID))},
			assert: func(t *testing.T, res OperationsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Operations, 1)
			},
		},
		{
			name: "name is resolved to a UUID before listing",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{
					Name: mo.Some("my-tag"), PageSize: mo.Some(int64(1)),
				}).Return(client.Result[components.TagsList]{
					Metadata: okMeta(),
					Data:     &components.TagsList{Tags: []components.Tag{{ID: testTagUUID, Name: "my-tag"}}, TotalSize: 1},
				}, nil)
				m.EXPECT().ListTagOperations(gomock.Any(), client.ListTagOperationsRequest{TagID: testTagUUID}).
					Return(operationsPage([]string{"op-1"}, components.TagOperationStatusSucceeded, 1, ""), nil)
				return m
			},
			params: OperationsParams{TagID: mo.Some(identifiers.NewTagID("my-tag"))},
			assert: func(t *testing.T, res OperationsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Operations, 1)
			},
		},
		{
			name: "filters and org are threaded to the client",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagOperations(gomock.Any(), client.ListTagOperationsRequest{
					OrgID:    mo.Some("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
					TagID:    "-",
					Type:     mo.Some("bulk_delete"),
					Status:   mo.Some("failed"),
					OrderBy:  mo.Some("create_time_asc"),
					PageSize: mo.Some(int64(50)),
				}).Return(operationsPage(nil, components.TagOperationStatusFailed, 0, ""), nil)
				return m
			},
			params: OperationsParams{
				OrgID:    mo.Some(identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))),
				Type:     mo.Some("bulk_delete"),
				Status:   mo.Some("failed"),
				OrderBy:  mo.Some("create_time_asc"),
				PageSize: mo.Some(uint64(50)),
			},
			assert: func(t *testing.T, res OperationsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Empty(t, res.Operations)
			},
		},
		{
			name: "pagination collects across pages",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().ListTagOperations(gomock.Any(), client.ListTagOperationsRequest{TagID: "-"}).
						Return(operationsPage([]string{"op-1"}, components.TagOperationStatusSucceeded, 2, "token1"), nil),
					m.EXPECT().ListTagOperations(gomock.Any(), client.ListTagOperationsRequest{
						TagID: "-", PageToken: mo.Some("token1"),
					}).Return(operationsPage([]string{"op-2"}, components.TagOperationStatusSucceeded, 2, ""), nil),
				)
				return m
			},
			params: OperationsParams{MaxPages: mo.None[uint64]()},
			assert: func(t *testing.T, res OperationsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Operations, 2)
				require.Equal(t, "op-2", res.Operations[1].ID)
			},
		},
		{
			name: "later page failure returns collected data with a partial error",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().ListTagOperations(gomock.Any(), gomock.Any()).
						Return(operationsPage([]string{"op-1"}, components.TagOperationStatusSucceeded, 2, "token1"), nil),
					m.EXPECT().ListTagOperations(gomock.Any(), gomock.Any()).
						Return(client.Result[components.TagOperationsList]{}, client.NewClientError(errors.New("boom"))),
				)
				return m
			},
			params: OperationsParams{MaxPages: mo.None[uint64]()},
			assert: func(t *testing.T, res OperationsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Operations, 1)
				require.NotNil(t, res.PartialError)
			},
		},
		{
			name: "invalid type is rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagOperations(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: OperationsParams{Type: mo.Some("bogus")},
			assert: func(t *testing.T, res OperationsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "invalid status is rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagOperations(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: OperationsParams{Status: mo.Some("in_progress")},
			assert: func(t *testing.T, res OperationsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "in_progress")
			},
		},
		{
			name: "invalid order-by is rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagOperations(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: OperationsParams{OrderBy: mo.Some("name_asc")},
			assert: func(t *testing.T, res OperationsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
			},
		},
		{
			name: "zero page size is rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagOperations(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: OperationsParams{PageSize: mo.Some(uint64(0))},
			assert: func(t *testing.T, res OperationsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))
			res, err := svc.ListOperations(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

func TestTagsService_GetOperation(t *testing.T) {
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params GetOperationParams
		assert func(t *testing.T, res GetOperationResult, err cenclierrors.CencliError)
	}{
		{
			name: "UUID tag skips resolution and maps the operation",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().GetTagOperation(gomock.Any(), mo.None[string](), testTagUUID, testOpUUID).
					Return(operationResult(components.TagOperationStatusSucceeded, 100), nil)
				return m
			},
			params: GetOperationParams{TagID: identifiers.NewTagID(testTagUUID), OperationID: testOpUUID},
			assert: func(t *testing.T, res GetOperationResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, testOpUUID, res.Operation.ID)
				require.Equal(t, "succeeded", res.Operation.Status)
				require.Equal(t, int64(100), res.Operation.ProcessedCount)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "name is resolved before the get",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{
					Name: mo.Some("my-tag"), PageSize: mo.Some(int64(1)),
				}).Return(client.Result[components.TagsList]{
					Metadata: okMeta(),
					Data:     &components.TagsList{Tags: []components.Tag{{ID: testTagUUID, Name: "my-tag"}}, TotalSize: 1},
				}, nil)
				m.EXPECT().GetTagOperation(gomock.Any(), mo.None[string](), testTagUUID, testOpUUID).
					Return(operationResult(components.TagOperationStatusRunning, 5), nil)
				return m
			},
			params: GetOperationParams{TagID: identifiers.NewTagID("my-tag"), OperationID: testOpUUID},
			assert: func(t *testing.T, res GetOperationResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, "running", res.Operation.Status)
			},
		},
		{
			name: "non-UUID operation ID is rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: GetOperationParams{TagID: identifiers.NewTagID(testTagUUID), OperationID: "not-a-uuid"},
			assert: func(t *testing.T, res GetOperationResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.True(t, err.ShouldPrintUsage())
				require.Contains(t, err.Error(), "not-a-uuid")
			},
		},
		{
			name: "empty tag is rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: GetOperationParams{TagID: identifiers.NewTagID(""), OperationID: testOpUUID},
			assert: func(t *testing.T, res GetOperationResult, err cenclierrors.CencliError) {
				require.Error(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))
			res, err := svc.GetOperation(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

// newWaitService builds a service whose poll loop records the delays it was
// asked to sleep for instead of actually waiting.
func newWaitService(c client.Client, delays *[]time.Duration) Service {
	return &tagsService{
		client: c,
		sleep: func(ctx context.Context, d time.Duration) error {
			*delays = append(*delays, d)
			return ctx.Err()
		},
	}
}

func TestTagsService_WaitForOperation(t *testing.T) {
	waitParams := WaitParams{TagID: identifiers.NewTagID(testTagUUID), OperationID: testOpUUID}

	// Every terminal status ends the poll and comes back as a result, never as
	// an error: the command layer owns what each outcome means.
	terminalStatuses := []components.TagOperationStatus{
		components.TagOperationStatusSucceeded,
		components.TagOperationStatusLimitReached,
		components.TagOperationStatusFailed,
		components.TagOperationStatusCancelled,
	}

	for _, status := range terminalStatuses {
		t.Run("returns immediately on "+string(status), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockClient(ctrl)
			m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), testTagUUID, testOpUUID).
				Return(operationResult(status, 100), nil).Times(1)

			var delays []time.Duration
			res, err := newWaitService(m, &delays).WaitForOperation(context.Background(), waitParams)

			require.NoError(t, err)
			require.Equal(t, string(status), res.Operation.Status)
			require.Empty(t, delays, "a terminal status must not sleep")
		})
	}

	t.Run("polls through pending and running to a terminal status", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockClient(ctrl)
		gomock.InOrder(
			m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), testTagUUID, testOpUUID).
				Return(operationResult(components.TagOperationStatusPending, 0), nil),
			m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), testTagUUID, testOpUUID).
				Return(operationResult(components.TagOperationStatusRunning, 50), nil),
			m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), testTagUUID, testOpUUID).
				Return(operationResult(components.TagOperationStatusSucceeded, 100), nil),
		)

		var delays []time.Duration
		res, err := newWaitService(m, &delays).WaitForOperation(context.Background(), waitParams)

		require.NoError(t, err)
		require.Equal(t, "succeeded", res.Operation.Status)
		require.Equal(t, []time.Duration{2 * time.Second, 4 * time.Second}, delays)
	})

	t.Run("backoff doubles and caps at the maximum", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockClient(ctrl)
		// Eight running polls, then done: long enough to reach and hold the cap.
		m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), testTagUUID, testOpUUID).
			Return(operationResult(components.TagOperationStatusRunning, 1), nil).Times(8)
		m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), testTagUUID, testOpUUID).
			Return(operationResult(components.TagOperationStatusSucceeded, 100), nil)

		var delays []time.Duration
		_, err := newWaitService(m, &delays).WaitForOperation(context.Background(), waitParams)

		require.NoError(t, err)
		require.Equal(t, []time.Duration{
			2 * time.Second, 4 * time.Second, 8 * time.Second,
			15 * time.Second, 15 * time.Second, 15 * time.Second,
			15 * time.Second, 15 * time.Second,
		}, delays)
	})

	t.Run("resolves the tag name once, not once per poll", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockClient(ctrl)
		m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(1).
			Return(client.Result[components.TagsList]{
				Metadata: okMeta(),
				Data:     &components.TagsList{Tags: []components.Tag{{ID: testTagUUID, Name: "my-tag"}}, TotalSize: 1},
			}, nil)
		gomock.InOrder(
			m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), testTagUUID, testOpUUID).
				Return(operationResult(components.TagOperationStatusRunning, 1), nil),
			m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), testTagUUID, testOpUUID).
				Return(operationResult(components.TagOperationStatusSucceeded, 100), nil),
		)

		var delays []time.Duration
		_, err := newWaitService(m, &delays).WaitForOperation(context.Background(), WaitParams{
			TagID: identifiers.NewTagID("my-tag"), OperationID: testOpUUID,
		})

		require.NoError(t, err)
	})

	t.Run("context cancelled mid-poll reports an interruption", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx, cancel := context.WithCancel(context.Background())

		m := mocks.NewMockClient(ctrl)
		m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), testTagUUID, testOpUUID).
			DoAndReturn(func(context.Context, mo.Option[string], string, string) (client.Result[components.TagOperation], client.ClientError) {
				// Cancel while "in flight", so the sleep that follows observes it.
				cancel()
				return operationResult(components.TagOperationStatusRunning, 1), nil
			})

		var delays []time.Duration
		_, err := newWaitService(m, &delays).WaitForOperation(ctx, waitParams)

		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
		require.True(t, cenclierrors.IsInterrupted(err))
	})

	t.Run("expired timeout reports a wait timeout carrying the last status", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockClient(ctrl)
		m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), testTagUUID, testOpUUID).
			Return(operationResult(components.TagOperationStatusRunning, 1), nil).AnyTimes()

		// A timeout this small is already expired by the first sleep.
		svc := &tagsService{
			client: m,
			sleep: func(ctx context.Context, _ time.Duration) error {
				<-ctx.Done()
				return ctx.Err()
			},
		}

		_, err := svc.WaitForOperation(context.Background(), WaitParams{
			TagID:       identifiers.NewTagID(testTagUUID),
			OperationID: testOpUUID,
			Timeout:     mo.Some(10 * time.Millisecond),
		})

		require.Error(t, err)
		require.False(t, err.ShouldPrintUsage())
		require.Equal(t, "Timeout", err.Title())
		require.Contains(t, err.Error(), "running")
		require.Contains(t, err.Error(), testOpUUID)
	})

	t.Run("caller cancellation outranks an expired timeout", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx, cancel := context.WithCancel(context.Background())

		m := mocks.NewMockClient(ctrl)
		m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), testTagUUID, testOpUUID).
			DoAndReturn(func(context.Context, mo.Option[string], string, string) (client.Result[components.TagOperation], client.ClientError) {
				cancel()
				return operationResult(components.TagOperationStatusRunning, 1), nil
			})

		var delays []time.Duration
		_, err := newWaitService(m, &delays).WaitForOperation(ctx, WaitParams{
			TagID:       identifiers.NewTagID(testTagUUID),
			OperationID: testOpUUID,
			Timeout:     mo.Some(time.Hour),
		})

		// Ctrl-C must read as an interruption, not as a timeout.
		require.Error(t, err)
		require.True(t, cenclierrors.IsInterrupted(err))
	})

	t.Run("non-UUID operation ID is rejected before polling", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockClient(ctrl)
		m.EXPECT().GetTagOperation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		var delays []time.Duration
		_, err := newWaitService(m, &delays).WaitForOperation(context.Background(), WaitParams{
			TagID: identifiers.NewTagID(testTagUUID), OperationID: "nope",
		})

		require.Error(t, err)
		require.True(t, err.ShouldPrintUsage())
	})
}
