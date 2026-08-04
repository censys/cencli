package tags

import (
	"fmt"
	"time"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// Terminal operation statuses the command layer branches on for its exit code.
// They mirror the API's status enum; the service validates the full set.
const (
	statusSucceeded    = "succeeded"
	statusLimitReached = "limit_reached"
	statusFailed       = "failed"
	statusCancelled    = "cancelled"
)

// timeoutWithoutWaitError signals that --timeout was set without --wait, where
// it would have no effect.
type timeoutWithoutWaitError struct{}

func NewTimeoutWithoutWaitError() cenclierrors.CencliError { return &timeoutWithoutWaitError{} }

func (e *timeoutWithoutWaitError) Error() string {
	return "--timeout only applies while polling; add --wait or drop --timeout"
}

func (e *timeoutWithoutWaitError) Title() string { return "Conflicting Flags" }

func (e *timeoutWithoutWaitError) ShouldPrintUsage() bool { return true }

// invalidWaitTimeoutError signals a negative --timeout, which would give up
// before the first poll ever ran.
type invalidWaitTimeoutError struct {
	value time.Duration
}

func NewInvalidWaitTimeoutError(value time.Duration) cenclierrors.CencliError {
	return &invalidWaitTimeoutError{value: value}
}

func (e *invalidWaitTimeoutError) Error() string {
	return fmt.Sprintf("--timeout must not be negative (got %s); use 0 to wait without a time limit", e.value)
}

func (e *invalidWaitTimeoutError) Title() string { return "Invalid Timeout" }

func (e *invalidWaitTimeoutError) ShouldPrintUsage() bool { return true }

// assignModeConflictError signals that explicit assets and --query were given
// together. Bulk is never inferred, so the two input modes cannot be mixed.
type assignModeConflictError struct{}

func NewAssignModeConflictError() cenclierrors.CencliError { return &assignModeConflictError{} }

func (e *assignModeConflictError) Error() string {
	return "--query assigns by search results, so it cannot be combined with explicit assets or --input-file"
}

func (e *assignModeConflictError) Title() string { return "Conflicting Input Modes" }

func (e *assignModeConflictError) ShouldPrintUsage() bool { return true }

// unassignModeConflictError signals that explicit assets and the bulk filters
// were given together. As with assign, bulk is never inferred, so the two input
// modes cannot be mixed.
type unassignModeConflictError struct{}

func NewUnassignModeConflictError() cenclierrors.CencliError { return &unassignModeConflictError{} }

func (e *unassignModeConflictError) Error() string {
	return "--all and the time filters unassign by filter, so they cannot be combined with explicit assets or --input-file"
}

func (e *unassignModeConflictError) Title() string { return "Conflicting Input Modes" }

func (e *unassignModeConflictError) ShouldPrintUsage() bool { return true }

// allWithTimeFilterError signals that --all was narrowed by a time filter. --all
// means every assignment, so the combination contradicts itself rather than
// meaning either one.
type allWithTimeFilterError struct{}

func NewAllWithTimeFilterError() cenclierrors.CencliError { return &allWithTimeFilterError{} }

func (e *allWithTimeFilterError) Error() string {
	return "--all unassigns every assignment, so it cannot be combined with --created-before or --created-after"
}

func (e *allWithTimeFilterError) Title() string { return "Conflicting Flags" }

func (e *allWithTimeFilterError) ShouldPrintUsage() bool { return true }

// flagRequiresBulkError signals that a bulk-only flag was set without the flag
// that selects bulk mode, where it would have no effect. verb and trigger name
// the operation and its mode flag, since assign and unassign enter bulk mode
// differently.
type flagRequiresBulkError struct {
	flag    string
	verb    string
	trigger string
}

// NewFlagRequiresQueryError reports a bulk-only assign flag used without --query.
func NewFlagRequiresQueryError(flag string) cenclierrors.CencliError {
	return &flagRequiresBulkError{flag: flag, verb: "assignment", trigger: "query"}
}

// NewFlagRequiresAllError reports a bulk-only unassign flag used without --all
// or a time filter. It names --all as the fix, being the unfiltered form.
func NewFlagRequiresAllError(flag string) cenclierrors.CencliError {
	return &flagRequiresBulkError{flag: flag, verb: "unassignment", trigger: "all"}
}

func (e *flagRequiresBulkError) Error() string {
	return fmt.Sprintf("--%s only applies to a bulk %s; add --%s or drop --%s",
		e.flag, e.verb, e.trigger, e.flag)
}

func (e *flagRequiresBulkError) Title() string { return "Conflicting Flags" }

func (e *flagRequiresBulkError) ShouldPrintUsage() bool { return true }

// operationFailedError signals that a waited-on operation finished as failed.
// The operation itself was still rendered; this only drives the exit code.
type operationFailedError struct {
	operation tags.TagOperation
}

func NewOperationFailedError(op tags.TagOperation) cenclierrors.CencliError {
	return &operationFailedError{operation: op}
}

func (e *operationFailedError) Error() string {
	msg := fmt.Sprintf("operation %s failed after %d of %d asset(s)",
		e.operation.ID, e.operation.SuccessfulCount, e.operation.TotalCount)
	if detail := operationDetail(e.operation); detail != "" {
		msg = fmt.Sprintf("%s: %s", msg, detail)
	}
	return msg
}

func (e *operationFailedError) Title() string { return "Operation Failed" }

func (e *operationFailedError) ShouldPrintUsage() bool { return false }

// operationCancelledError signals that a waited-on operation was cancelled.
// Work already committed before the cancellation is kept by the API.
type operationCancelledError struct {
	operation tags.TagOperation
}

func NewOperationCancelledError(op tags.TagOperation) cenclierrors.CencliError {
	return &operationCancelledError{operation: op}
}

func (e *operationCancelledError) Error() string {
	msg := fmt.Sprintf("operation %s was cancelled after %d of %d asset(s)",
		e.operation.ID, e.operation.SuccessfulCount, e.operation.TotalCount)
	if detail := operationDetail(e.operation); detail != "" {
		msg = fmt.Sprintf("%s: %s", msg, detail)
	}
	return msg
}

func (e *operationCancelledError) Title() string { return "Operation Cancelled" }

func (e *operationCancelledError) ShouldPrintUsage() bool { return false }

// operationDetail picks the most specific explanation the API gave. On failure
// status_message mirrors error_message, so either one will do.
func operationDetail(op tags.TagOperation) string {
	if op.ErrorMessage != nil && *op.ErrorMessage != "" {
		return *op.ErrorMessage
	}
	if op.StatusMessage != nil && *op.StatusMessage != "" {
		return *op.StatusMessage
	}
	return ""
}

// nothingToUpdateError signals that `tags update` was invoked without any
// mutation flag, so there is nothing to change.
type nothingToUpdateError struct{}

func NewNothingToUpdateError() cenclierrors.CencliError { return &nothingToUpdateError{} }

func (e *nothingToUpdateError) Error() string {
	return "no fields to update; specify at least one of --name, --privacy, --description, --clear-description"
}

func (e *nothingToUpdateError) Title() string { return "Nothing To Update" }

func (e *nothingToUpdateError) ShouldPrintUsage() bool { return true }

// descriptionConflictError signals that --description and --clear-description
// were used together, which is contradictory.
type descriptionConflictError struct{}

func NewDescriptionConflictError() cenclierrors.CencliError { return &descriptionConflictError{} }

func (e *descriptionConflictError) Error() string {
	return "--description and --clear-description cannot be used together"
}

func (e *descriptionConflictError) Title() string { return "Conflicting Flags" }

func (e *descriptionConflictError) ShouldPrintUsage() bool { return true }

// confirmationRequiredError signals that a destructive command was invoked in a
// non-interactive terminal without --yes, so it cannot prompt for confirmation.
type confirmationRequiredError struct{}

func NewConfirmationRequiredError() cenclierrors.CencliError { return &confirmationRequiredError{} }

func (e *confirmationRequiredError) Error() string {
	return "confirmation required; re-run with --yes to skip the prompt in a non-interactive terminal"
}

func (e *confirmationRequiredError) Title() string { return "Confirmation Required" }

func (e *confirmationRequiredError) ShouldPrintUsage() bool { return true }

// allAssetsFailedError signals that no asset in an explicit assign or unassign
// succeeded. The per-asset outcomes were still rendered; this only drives the
// exit code, the way operationFailedError does after a wait. It counts failures
// against assets requested rather than saying "all", since an interrupted run
// records fewer failures than it was given assets.
type allAssetsFailedError struct {
	failed    int
	requested int
	// verb is the past participle: "assigned" or "unassigned".
	verb string
}

// NewAllAssetsFailedError reports that no asset in the run succeeded.
func NewAllAssetsFailedError(failed, requested int, verb string) cenclierrors.CencliError {
	return &allAssetsFailedError{failed: failed, requested: requested, verb: verb}
}

func (e *allAssetsFailedError) Error() string {
	return fmt.Sprintf("no asset was %s; %d of %d failed - see the per-asset results above",
		e.verb, e.failed, e.requested)
}

func (e *allAssetsFailedError) Title() string { return "All Assets Failed" }

func (e *allAssetsFailedError) ShouldPrintUsage() bool { return false }
