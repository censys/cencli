package tags

import (
	"fmt"
	"strings"
	"time"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// invalidPaginationParamsError signals that a pagination parameter (page size or
// max pages) was given an invalid value.
type invalidPaginationParamsError struct {
	reason string
}

// NewInvalidPaginationParamsError creates an invalid-pagination-params error.
func NewInvalidPaginationParamsError(reason string) cenclierrors.CencliError {
	return &invalidPaginationParamsError{reason: reason}
}

func (e *invalidPaginationParamsError) Error() string { return e.reason }

func (e *invalidPaginationParamsError) Title() string { return "Invalid Pagination Parameters" }

func (e *invalidPaginationParamsError) ShouldPrintUsage() bool { return true }

// invalidEnumFilterError signals that a filter flag with a fixed set of accepted
// values (e.g. --order-by, --privacy) was given an unsupported value.
type invalidEnumFilterError struct {
	filter    string
	provided  string
	supported []string
}

// NewInvalidEnumFilterError creates an invalid-enum-filter error naming the flag,
// the rejected value, and the accepted set.
func NewInvalidEnumFilterError(filter, provided string, supported []string) cenclierrors.CencliError {
	return &invalidEnumFilterError{filter: filter, provided: provided, supported: supported}
}

func (e *invalidEnumFilterError) Error() string {
	return fmt.Sprintf("invalid %s '%s'; supported values: %s", e.filter, e.provided, strings.Join(e.supported, ", "))
}

func (e *invalidEnumFilterError) Title() string { return "Invalid Filter Value" }

func (e *invalidEnumFilterError) ShouldPrintUsage() bool { return true }

// invalidTimeWindowError signals that a pair of time filters cannot match
// anything, e.g. created-before earlier than created-after.
type invalidTimeWindowError struct {
	reason string
}

// NewInvalidTimeWindowError creates an invalid-time-window error.
func NewInvalidTimeWindowError(reason string) cenclierrors.CencliError {
	return &invalidTimeWindowError{reason: reason}
}

func (e *invalidTimeWindowError) Error() string { return e.reason }

func (e *invalidTimeWindowError) Title() string { return "Invalid Time Window" }

func (e *invalidTimeWindowError) ShouldPrintUsage() bool { return true }

// emptyTagIDError signals that a tag command was given an empty tag identifier
// (name or UUID). Rejected before any lookup so an empty name can never be sent
// to ListTags — which would otherwise match no filter and resolve to an
// arbitrary tag.
type emptyTagIDError struct{}

// NewEmptyTagIDError creates an empty-tag-identifier error.
func NewEmptyTagIDError() cenclierrors.CencliError { return &emptyTagIDError{} }

func (e *emptyTagIDError) Error() string { return "a tag name or ID is required" }

func (e *emptyTagIDError) Title() string { return "Invalid Tag" }

func (e *emptyTagIDError) ShouldPrintUsage() bool { return true }

// invalidOperationIDError signals that an operation identifier is not a UUID.
// The endpoint declares operation_id as format:uuid, so anything else is a
// guaranteed 422 — rejected at the boundary instead of after a round trip.
type invalidOperationIDError struct {
	provided string
}

// NewInvalidOperationIDError creates an invalid-operation-ID error.
func NewInvalidOperationIDError(provided string) cenclierrors.CencliError {
	return &invalidOperationIDError{provided: provided}
}

func (e *invalidOperationIDError) Error() string {
	if e.provided == "" {
		return "an operation ID is required"
	}
	return fmt.Sprintf("operation ID %q is not a valid UUID", e.provided)
}

func (e *invalidOperationIDError) Title() string { return "Invalid Operation ID" }

func (e *invalidOperationIDError) ShouldPrintUsage() bool { return true }

// operationWaitTimeoutError signals that --wait gave up before the operation
// reached a terminal status. The job keeps running server-side.
type operationWaitTimeoutError struct {
	operationID string
	status      string
	timeout     time.Duration
}

// NewOperationWaitTimeoutError creates a wait-timeout error carrying the last
// status seen before giving up.
func NewOperationWaitTimeoutError(operationID, status string, timeout time.Duration) cenclierrors.CencliError {
	return &operationWaitTimeoutError{operationID: operationID, status: status, timeout: timeout}
}

func (e *operationWaitTimeoutError) Error() string {
	return fmt.Sprintf(
		"timed out after %s waiting for operation %s; it is still %s and continues server-side",
		e.timeout, e.operationID, e.status)
}

func (e *operationWaitTimeoutError) Title() string { return "Timeout" }

func (e *operationWaitTimeoutError) ShouldPrintUsage() bool { return false }

// tagNotFoundError signals that a tag name could not be resolved to an existing
// tag during a name→UUID lookup.
type tagNotFoundError struct {
	name string
}

// NewTagNotFoundError creates a tag-not-found error naming the unresolved tag.
func NewTagNotFoundError(name string) cenclierrors.CencliError {
	return &tagNotFoundError{name: name}
}

func (e *tagNotFoundError) Error() string {
	return fmt.Sprintf("tag %q not found", e.name)
}

func (e *tagNotFoundError) Title() string { return "Tag Not Found" }

func (e *tagNotFoundError) ShouldPrintUsage() bool { return false }

// invalidTagNameError signals that a tag name was empty or whitespace-only.
type invalidTagNameError struct{}

// NewInvalidTagNameError creates an invalid-tag-name error.
func NewInvalidTagNameError() cenclierrors.CencliError { return &invalidTagNameError{} }

func (e *invalidTagNameError) Error() string { return "tag name must not be empty" }

func (e *invalidTagNameError) Title() string { return "Invalid Tag Name" }

func (e *invalidTagNameError) ShouldPrintUsage() bool { return true }

// noAssetsError signals that an assignment command was given no assets to act
// on (a service-level guard; the command layer normally rejects it earlier).
type noAssetsError struct{}

// NewNoAssetsError creates a no-assets error.
func NewNoAssetsError() cenclierrors.CencliError { return &noAssetsError{} }

func (e *noAssetsError) Error() string { return "at least one asset is required" }

func (e *noAssetsError) Title() string { return "No Assets Provided" }

func (e *noAssetsError) ShouldPrintUsage() bool { return true }

// assignPartialError summarizes an assign run where some assets failed.
type assignPartialError struct {
	failed int
	total  int
}

func newAssignPartialError(failed, total int) cenclierrors.CencliError {
	return &assignPartialError{failed: failed, total: total}
}

func (e *assignPartialError) Error() string {
	return fmt.Sprintf("%d of %d asset(s) failed to assign", e.failed, e.total)
}

func (e *assignPartialError) Title() string { return "Some Assets Failed to Assign" }

func (e *assignPartialError) ShouldPrintUsage() bool { return false }

// unassignPartialError summarizes an unassign run where some assets failed.
type unassignPartialError struct {
	failed int
	total  int
}

func newUnassignPartialError(failed, total int) cenclierrors.CencliError {
	return &unassignPartialError{failed: failed, total: total}
}

func (e *unassignPartialError) Error() string {
	return fmt.Sprintf("%d of %d asset(s) failed to unassign", e.failed, e.total)
}

func (e *unassignPartialError) Title() string { return "Some Assets Failed to Unassign" }

func (e *unassignPartialError) ShouldPrintUsage() bool { return false }

// assetNotAssignedError signals that an asset has no assignment to the tag, so
// there is nothing to remove. Recorded as a per-asset failure during unassign.
type assetNotAssignedError struct {
	assetID string
}

// NewAssetNotAssignedError creates an asset-not-assigned error naming the asset.
func NewAssetNotAssignedError(assetID string) cenclierrors.CencliError {
	return &assetNotAssignedError{assetID: assetID}
}

func (e *assetNotAssignedError) Error() string {
	return fmt.Sprintf("asset %q is not assigned to this tag", e.assetID)
}

func (e *assetNotAssignedError) Title() string { return "Asset Not Assigned" }

func (e *assetNotAssignedError) ShouldPrintUsage() bool { return false }
