package scan

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// A rescan takes several minutes; polling faster only spends requests.
const (
	initialPollInterval = 5 * time.Second
	maxPollInterval     = 30 * time.Second
)

func (s *scanService) Wait(
	ctx context.Context,
	params WaitParams,
) (Result, cenclierrors.CencliError) {
	if params.OrgID.IsZero() {
		return Result{}, cenclierrors.NewNoOrgIDError()
	}
	scanID, err := RequireScanID(params.ScanID)
	if err != nil {
		return Result{}, err
	}
	orgIDStr := params.OrgID.String()

	pollCtx := ctx
	if params.Timeout.IsPresent() {
		var cancel context.CancelFunc
		pollCtx, cancel = context.WithTimeout(ctx, params.Timeout.MustGet())
		defer cancel()
	}

	interval := initialPollInterval
	var last Result

	for {
		res, getErr := s.client.GetTrackedScan(pollCtx, orgIDStr, scanID)
		if getErr != nil {
			if waitErr := waitContextError(ctx, pollCtx, scanID, params.Timeout); waitErr != nil {
				return last, waitErr
			}
			return last, getErr
		}

		current, mapErr := newScanResult(res)
		if mapErr != nil {
			return last, mapErr
		}
		last = current

		if last.Scan.Completed {
			return last, nil
		}

		reportScanProgress(pollCtx, *last.Scan)

		if sleepErr := s.sleep(pollCtx, interval); sleepErr != nil {
			if waitErr := waitContextError(ctx, pollCtx, scanID, params.Timeout); waitErr != nil {
				return last, waitErr
			}
			return last, cenclierrors.ParseContextError(sleepErr)
		}

		interval = min(interval*2, maxPollInterval)
	}
}

// waitContextError explains why polling stopped, preferring the caller's own
// cancellation over an expired wait timeout. It returns nil when neither
// context is done.
func waitContextError(ctx, pollCtx context.Context, scanID string, timeout mo.Option[time.Duration]) cenclierrors.CencliError {
	if parentErr := ctx.Err(); parentErr != nil {
		return cenclierrors.ParseContextError(parentErr)
	}
	if pollCtx.Err() != nil && timeout.IsPresent() {
		return NewScanWaitTimeoutError(scanID, timeout.MustGet())
	}
	return nil
}

func reportScanProgress(ctx context.Context, s TrackedScan) {
	if len(s.Tasks) == 0 {
		progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Waiting for scan %s to start...", s.ID))
		return
	}
	done := 0
	for _, t := range s.Tasks {
		if t.Status != "" && t.Status != TaskStatusScanning {
			done++
		}
	}
	progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Waiting for scan %s: %d/%d tasks finished...", s.ID, done, len(s.Tasks)))
}
