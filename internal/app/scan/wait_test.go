package scan

import (
	"context"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
)

func TestScanService_Wait(t *testing.T) {
	waitParams := WaitParams{OrgID: testOrgID, ScanID: testScanID}
	running := func() client.Result[components.TrackedScan] {
		return trackedScanResult(false, task(components.TrackedScanTaskStatusScanning), task(components.TrackedScanTaskStatusScanned))
	}

	t.Run("returns immediately when already completed", func(t *testing.T) {
		m := mocks.NewMockClient(gomock.NewController(t))
		m.EXPECT().GetTrackedScan(gomock.Any(), testOrgUUID, testScanID).
			Return(trackedScanResult(true, task(components.TrackedScanTaskStatusCompleted)), nil).Times(1)

		var delays []time.Duration
		res, err := newTestService(m, &delays).Wait(context.Background(), waitParams)

		require.NoError(t, err)
		assert.True(t, res.Scan.Completed)
		assert.Empty(t, delays, "a completed scan must not sleep")
	})

	t.Run("a completed but failed scan is still a result, not an error", func(t *testing.T) {
		m := mocks.NewMockClient(gomock.NewController(t))
		m.EXPECT().GetTrackedScan(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(trackedScanResult(true, task(components.TrackedScanTaskStatusRejected)), nil)

		var delays []time.Duration
		res, err := newTestService(m, &delays).Wait(context.Background(), waitParams)

		require.NoError(t, err)
		assert.False(t, res.Scan.Succeeded())
	})

	t.Run("polls with backoff until completed", func(t *testing.T) {
		m := mocks.NewMockClient(gomock.NewController(t))
		calls := []any{}
		for range 5 {
			calls = append(calls, m.EXPECT().GetTrackedScan(gomock.Any(), gomock.Any(), gomock.Any()).Return(running(), nil))
		}
		calls = append(calls, m.EXPECT().GetTrackedScan(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(trackedScanResult(true, task(components.TrackedScanTaskStatusCompleted)), nil))
		gomock.InOrder(calls...)

		var delays []time.Duration
		res, err := newTestService(m, &delays).Wait(context.Background(), waitParams)

		require.NoError(t, err)
		assert.True(t, res.Scan.Completed)
		assert.Equal(t, []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second, 30 * time.Second, 30 * time.Second}, delays)
	})

	t.Run("expired timeout returns the last scan with a 124-class error", func(t *testing.T) {
		m := mocks.NewMockClient(gomock.NewController(t))
		m.EXPECT().GetTrackedScan(gomock.Any(), gomock.Any(), gomock.Any()).Return(running(), nil).AnyTimes()

		svc := &scanService{client: m, sleep: func(ctx context.Context, _ time.Duration) error {
			<-ctx.Done()
			return ctx.Err()
		}}
		res, err := svc.Wait(context.Background(), WaitParams{OrgID: testOrgID, ScanID: testScanID, Timeout: mo.Some(10 * time.Millisecond)})

		require.Error(t, err)
		assert.True(t, cenclierrors.IsDeadlineExceeded(err))
		assert.False(t, cenclierrors.IsInterrupted(err))
		assert.Contains(t, err.Error(), testScanID)
		require.NotNil(t, res.Scan, "the last polled scan survives the timeout")
		assert.False(t, res.Scan.Completed)
		assert.NotNil(t, res.Meta)
	})

	t.Run("cancellation returns the last scan with an interrupted error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		m := mocks.NewMockClient(gomock.NewController(t))
		m.EXPECT().GetTrackedScan(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(context.Context, string, string) (client.Result[components.TrackedScan], client.ClientError) {
				cancel()
				return running(), nil
			})

		var delays []time.Duration
		res, err := newTestService(m, &delays).Wait(ctx, waitParams)

		require.Error(t, err)
		assert.True(t, cenclierrors.IsInterrupted(err))
		require.NotNil(t, res.Scan)
		assert.Equal(t, testScanID, res.Scan.ID)
	})

	t.Run("caller cancellation outranks an expired timeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		m := mocks.NewMockClient(gomock.NewController(t))
		m.EXPECT().GetTrackedScan(gomock.Any(), gomock.Any(), gomock.Any()).Return(running(), nil)

		svc := &scanService{client: m, sleep: func(pollCtx context.Context, _ time.Duration) error {
			<-pollCtx.Done()
			cancel()
			return pollCtx.Err()
		}}
		_, err := svc.Wait(ctx, WaitParams{OrgID: testOrgID, ScanID: testScanID, Timeout: mo.Some(time.Millisecond)})

		require.Error(t, err)
		assert.True(t, cenclierrors.IsInterrupted(err))
	})

	t.Run("a failed poll after a good one returns the last scan and the error", func(t *testing.T) {
		m := mocks.NewMockClient(gomock.NewController(t))
		gomock.InOrder(
			m.EXPECT().GetTrackedScan(gomock.Any(), gomock.Any(), gomock.Any()).Return(running(), nil),
			m.EXPECT().GetTrackedScan(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(client.Result[components.TrackedScan]{}, structuredErr(500)),
		)

		var delays []time.Duration
		res, err := newTestService(m, &delays).Wait(context.Background(), waitParams)

		require.Error(t, err)
		require.NotNil(t, res.Scan)
		assert.False(t, res.Scan.Completed)
	})

	t.Run("a failed first poll returns no scan", func(t *testing.T) {
		m := mocks.NewMockClient(gomock.NewController(t))
		m.EXPECT().GetTrackedScan(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(client.Result[components.TrackedScan]{}, structuredErr(404))

		var delays []time.Duration
		res, err := newTestService(m, &delays).Wait(context.Background(), waitParams)

		require.Error(t, err)
		assert.Nil(t, res.Scan)
	})

	t.Run("invalid input is refused without a request", func(t *testing.T) {
		m := mocks.NewMockClient(gomock.NewController(t))
		var delays []time.Duration
		svc := newTestService(m, &delays)

		_, err := svc.Wait(context.Background(), WaitParams{ScanID: testScanID})
		require.Error(t, err)
		_, err = svc.Wait(context.Background(), WaitParams{OrgID: testOrgID, ScanID: "nope"})
		require.Error(t, err)
		assert.Equal(t, "Invalid Scan ID", err.Title())
	})
}
