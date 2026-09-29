package scan

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	appscan "github.com/censys/cencli/internal/app/scan"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
)

// defaultWaitTimeout bounds --wait so a stalled scan cannot hang a script
// indefinitely. The scan keeps running server-side either way.
const defaultWaitTimeout = 15 * time.Minute

// waitForScan polls a scan to completion behind a spinner. The result carries
// the last state seen even when an error is returned.
func waitForScan(
	ctx context.Context,
	base *command.BaseCommand,
	logger *slog.Logger,
	svc appscan.Service,
	params appscan.WaitParams,
) (appscan.Result, cenclierrors.CencliError) {
	var result appscan.Result
	err := base.WithProgress(ctx, logger, "Waiting for scan to complete...",
		func(pctx context.Context) cenclierrors.CencliError {
			var waitErr cenclierrors.CencliError
			result, waitErr = svc.Wait(pctx, params)
			return waitErr
		})
	return result, err
}

// reportWaitOutcome maps the end of a wait onto the exit code. A wait that
// stopped early after seeing the scan says how to resume tracking; a completed
// scan fails only if it produced no results.
func reportWaitOutcome(quiet bool, scanID string, s *appscan.TrackedScan, err cenclierrors.CencliError) cenclierrors.CencliError {
	if err != nil {
		switch {
		case cenclierrors.IsInterrupted(err):
			printScanStillRunningNote(quiet, scanID)
		case cenclierrors.IsDeadlineExceeded(err), s != nil:
			printScanTrackHint(quiet, scanID)
		}
		return err
	}
	if s != nil && !s.Succeeded() {
		return NewScanFailedError(*s)
	}
	return nil
}

// printScanStillRunningNote reminds the user that interrupting the poll does
// not stop the scan, and how to pick tracking back up.
func printScanStillRunningNote(quiet bool, scanID string) {
	if quiet {
		return
	}
	formatter.Println(formatter.Stderr, styles.GlobalStyles.Warning.Render(
		"Stopped waiting; the scan continues server-side."))
	printScanTrackHint(quiet, scanID)
}

// printScanTrackHint tells the user how to resume following a scan. It stays
// silent without a scan ID, so the hint is never half a command.
func printScanTrackHint(quiet bool, scanID string) {
	if quiet || scanID == "" {
		return
	}
	formatter.Println(formatter.Stderr, fmt.Sprintf("Track with: censys scan get %s --wait", scanID))
}

// printRescanMayBeChargedNote warns that a rescan interrupted in flight may
// still have been accepted, leaving no scan ID to follow.
func printRescanMayBeChargedNote(quiet bool) {
	if quiet {
		return
	}
	formatter.Println(formatter.Stderr, styles.GlobalStyles.Warning.Render(
		"The rescan request was interrupted and may still have been accepted and charged 10 credits; "+
			"check `censys org credits` before trying again."))
}
