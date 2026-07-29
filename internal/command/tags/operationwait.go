package tags

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
)

// defaultWaitTimeout bounds --wait so a stalled job cannot hang a script
// indefinitely. The operation keeps running server-side either way.
const defaultWaitTimeout = 30 * time.Minute

// parseWaitFlags reads the --wait/--timeout pair every command that can follow an
// operation shares. A zero timeout means "no limit", matching the global
// --timeout-http, and a negative one is rejected rather than expiring before the
// first poll.
func parseWaitFlags(
	cmd *cobra.Command,
	waitFlag flags.BoolFlag,
	timeoutFlag flags.HumanDurationFlag,
) (bool, mo.Option[time.Duration], cenclierrors.CencliError) {
	none := mo.None[time.Duration]()

	wait, err := waitFlag.Value()
	if err != nil {
		return false, none, err
	}

	timeout, err := timeoutFlag.Value()
	if err != nil {
		return false, none, err
	}

	// A timeout only means something while polling; silently ignoring it would
	// make the flag look like it worked.
	if !wait && cmd.Flags().Changed("timeout") {
		return false, none, NewTimeoutWithoutWaitError()
	}

	if timeout.IsPresent() {
		switch d := timeout.MustGet(); {
		case d < 0:
			return false, none, NewInvalidWaitTimeoutError(d)
		case d == 0:
			timeout = none
		}
	}

	return wait, timeout, nil
}

// waitForOperation polls an operation to completion behind a spinner. Shared by
// every command that can wait on a bulk job.
func waitForOperation(
	ctx context.Context,
	base *command.BaseCommand,
	logger *slog.Logger,
	svc tags.Service,
	params tags.WaitParams,
) (tags.GetOperationResult, cenclierrors.CencliError) {
	var result tags.GetOperationResult
	err := base.WithProgress(ctx, logger, "Waiting for operation to finish...",
		func(pctx context.Context) cenclierrors.CencliError {
			var waitErr cenclierrors.CencliError
			result, waitErr = svc.WaitForOperation(pctx, params)
			return waitErr
		})
	return result, err
}

// followSubmittedOperation polls a job that was just submitted and returns the
// finished operation. A wait that ends early still points the user at the job,
// which keeps running server-side regardless. Shared by bulk assign and bulk
// unassign, which differ only in what they submitted.
func followSubmittedOperation(
	ctx context.Context,
	base *command.BaseCommand,
	logger *slog.Logger,
	svc tags.Service,
	params tags.WaitParams,
) (tags.TagOperation, cenclierrors.CencliError) {
	result, err := waitForOperation(ctx, base, logger, svc, params)
	if err != nil {
		quiet := base.Config().Quiet
		if cenclierrors.IsInterrupted(err) {
			printOperationStillRunningNote(quiet, params.TagID.String(), params.OperationID)
		} else {
			printOperationTrackHint(quiet, params.TagID.String(), params.OperationID)
		}
		return tags.TagOperation{}, err
	}
	return result.Operation, nil
}

// reportOperationTerminalStatus maps a finished operation onto the exit code. A
// capped run still succeeded, so it warns rather than failing. Only a wait calls
// this: reading a failed operation is itself a successful read.
func reportOperationTerminalStatus(op tags.TagOperation) cenclierrors.CencliError {
	switch op.Status {
	case statusFailed:
		return NewOperationFailedError(op)
	case statusCancelled:
		return NewOperationCancelledError(op)
	case statusLimitReached:
		msg := fmt.Sprintf(
			"Warning: operation %s stopped at its asset limit after %d of %d asset(s).",
			op.ID, op.SuccessfulCount, op.TotalCount)
		if op.StatusMessage != nil && *op.StatusMessage != "" {
			msg = fmt.Sprintf("%s %s", msg, *op.StatusMessage)
		}
		formatter.Println(formatter.Stderr, styles.GlobalStyles.Warning.Render(msg))
		return nil
	default:
		return nil
	}
}

// printOperationStillRunningNote reminds the user that interrupting the poll does
// not stop the job, and how to pick tracking back up.
func printOperationStillRunningNote(quiet bool, tagID, operationID string) {
	if quiet {
		return
	}
	formatter.Println(formatter.Stderr, styles.GlobalStyles.Warning.Render(
		"Stopped waiting; the operation continues server-side."))
	printOperationTrackHint(quiet, tagID, operationID)
}

// printOperationTrackHint tells the user how to follow an operation they did not
// wait on. It stays silent without an operation to name, so the hint is never a
// half-written command.
func printOperationTrackHint(quiet bool, tagID, operationID string) {
	if quiet || operationID == "" {
		return
	}
	formatter.Println(formatter.Stderr, fmt.Sprintf(
		"Track with: censys tags operations get %s %s --wait", shellArg(tagID), operationID))
}

// shellArg quotes a value only when pasting it back into a shell would otherwise
// split or mangle it. Tag names allow spaces, which would silently turn the
// hinted command into a different one.
func shellArg(value string) string {
	if strings.ContainsAny(value, " \t\n\"'\\$`*?&|;<>()[]{}#~!") {
		return fmt.Sprintf("%q", value)
	}
	return value
}
