package tags

import (
	"errors"
	"time"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

// ListParams bundles inputs for listing tags. Filters left empty are omitted
// from the request. Using a struct keeps the service API extensible as later
// tickets add commands.
type ListParams struct {
	OrgID     mo.Option[identifiers.OrganizationID]
	Privacy   mo.Option[string]
	Name      mo.Option[string]
	CreatedBy mo.Option[string]
	OrderBy   mo.Option[string]
	PageSize  mo.Option[uint64]
	MaxPages  mo.Option[uint64]
}

// GetParams bundles inputs for retrieving a single tag by name or UUID.
type GetParams struct {
	OrgID mo.Option[identifiers.OrganizationID]
	TagID identifiers.TagID
}

// CreateParams bundles inputs for creating a tag.
type CreateParams struct {
	OrgID       mo.Option[identifiers.OrganizationID]
	Name        string
	Description mo.Option[string]
	Privacy     string
}

// UpdateParams bundles inputs for updating a tag by name or UUID. Clearing the
// description is expressed as Description = mo.Some("").
type UpdateParams struct {
	OrgID       mo.Option[identifiers.OrganizationID]
	TagID       identifiers.TagID
	Name        mo.Option[string]
	Description mo.Option[string]
	Privacy     mo.Option[string]
}

// DeleteParams bundles inputs for deleting a tag by name or UUID.
type DeleteParams struct {
	OrgID mo.Option[identifiers.OrganizationID]
	TagID identifiers.TagID
}

// AssignParams bundles inputs for assigning a tag (by name or UUID) to explicit
// assets. AssetIDs are validated and deduplicated by the command layer.
type AssignParams struct {
	OrgID    mo.Option[identifiers.OrganizationID]
	TagID    identifiers.TagID
	AssetIDs []string
}

// UnassignParams bundles inputs for unassigning a tag (by name or UUID) from
// explicit assets. AssetIDs are validated and deduplicated by the command layer.
type UnassignParams struct {
	OrgID    mo.Option[identifiers.OrganizationID]
	TagID    identifiers.TagID
	AssetIDs []string
}

// BulkAssignParams bundles inputs for assigning a tag (by name or UUID) to every
// asset matching a CenQL query. An absent MaxAssets leaves the cap to the plan's
// tag asset limit.
type BulkAssignParams struct {
	OrgID     mo.Option[identifiers.OrganizationID]
	TagID     identifiers.TagID
	Query     string
	MaxAssets mo.Option[int64]
}

// BulkAssignResult is the outcome of submitting a bulk assignment. The endpoint
// answers 202 with the operation tracking the job, not the assignments, so the
// caller polls the operation to learn how it ended.
type BulkAssignResult struct {
	Meta      *responsemeta.ResponseMeta
	Operation TagOperation
}

// BulkUnassignParams bundles inputs for removing a tag (by name or UUID) from
// assignments selected by filter rather than by asset. With neither timestamp
// present every assignment of the tag is removed.
type BulkUnassignParams struct {
	OrgID         mo.Option[identifiers.OrganizationID]
	TagID         identifiers.TagID
	CreatedBefore mo.Option[time.Time]
	CreatedAfter  mo.Option[time.Time]
}

// BulkUnassignResult is the outcome of submitting a bulk unassignment. Like its
// create counterpart the endpoint answers 202 with the operation tracking the
// job, so the caller polls the operation to learn how it ended.
type BulkUnassignResult struct {
	Meta      *responsemeta.ResponseMeta
	Operation TagOperation
}

// AssignmentsParams bundles inputs for listing a tag's assignments. Filters left
// empty are omitted from the request.
type AssignmentsParams struct {
	OrgID         mo.Option[identifiers.OrganizationID]
	TagID         identifiers.TagID
	AssetID       mo.Option[string]
	AssetType     mo.Option[string]
	CreatedBy     mo.Option[string]
	CreatedBefore mo.Option[time.Time]
	CreatedAfter  mo.Option[time.Time]
	OrderBy       mo.Option[string]
	PageSize      mo.Option[uint64]
	MaxPages      mo.Option[uint64]
}

// OperationsParams bundles inputs for listing bulk tag operations. An absent
// TagID lists operations across every tag in the organization.
type OperationsParams struct {
	OrgID    mo.Option[identifiers.OrganizationID]
	TagID    mo.Option[identifiers.TagID]
	Type     mo.Option[string]
	Status   mo.Option[string]
	OrderBy  mo.Option[string]
	PageSize mo.Option[uint64]
	MaxPages mo.Option[uint64]
}

// GetOperationParams bundles inputs for retrieving one bulk tag operation. The
// endpoint is keyed by UUID, so a tag name is resolved first.
type GetOperationParams struct {
	OrgID       mo.Option[identifiers.OrganizationID]
	TagID       identifiers.TagID
	OperationID string
}

// CancelOperationParams bundles inputs for cancelling one bulk tag operation.
// Both path parameters are UUID-only, so a tag name is resolved first.
type CancelOperationParams struct {
	OrgID       mo.Option[identifiers.OrganizationID]
	TagID       identifiers.TagID
	OperationID string
}

// CancelOperationResult is the outcome of requesting a cancellation. The
// operation comes back as it stood when the request was accepted, which may
// still be a non-terminal status while the job winds down.
type CancelOperationResult struct {
	Meta      *responsemeta.ResponseMeta
	Operation TagOperation
}

// WaitParams bundles inputs for polling an operation until it finishes. An
// absent Timeout polls until a terminal status or context cancellation.
type WaitParams struct {
	OrgID       mo.Option[identifiers.OrganizationID]
	TagID       identifiers.TagID
	OperationID string
	Timeout     mo.Option[time.Duration]
}

// TagOperation is the domain representation of an asynchronous bulk tag job.
// The optional fields stay pointers so absent values are omitted from json and
// yaml output rather than rendered as empty strings.
type TagOperation struct {
	ID      string `json:"id" yaml:"id"`
	TagID   string `json:"tag_id" yaml:"tag_id"`
	TagName string `json:"tag_name" yaml:"tag_name"`
	Type    string `json:"type" yaml:"type"`
	Status  string `json:"status" yaml:"status"`
	// Query is only set for bulk_create operations.
	Query *string `json:"query,omitempty" yaml:"query,omitempty"`
	// TotalCount is approximate at start for bulk_create, and set at completion
	// for bulk_delete.
	TotalCount      int64 `json:"total_count" yaml:"total_count"`
	ProcessedCount  int64 `json:"processed_count" yaml:"processed_count"`
	SuccessfulCount int64 `json:"successful_count" yaml:"successful_count"`
	// StatusMessage is set once the operation finishes; it mirrors ErrorMessage
	// on failure and explains the cap on limit_reached.
	StatusMessage *string    `json:"status_message,omitempty" yaml:"status_message,omitempty"`
	ErrorMessage  *string    `json:"error_message,omitempty" yaml:"error_message,omitempty"`
	CreatedAt     time.Time  `json:"created_at" yaml:"created_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty" yaml:"ended_at,omitempty"`
}

// OperationsResult is the outcome of listing bulk tag operations.
type OperationsResult struct {
	Meta       *responsemeta.ResponseMeta
	Operations []TagOperation
	// TotalSize is the API's count of operations matching the filters.
	TotalSize int64
	// PartialError summarizes an error hit after the first successful page.
	PartialError cenclierrors.CencliError
}

// GetOperationResult is the outcome of retrieving or waiting on one operation.
// A terminal status is not an error here: the operation is returned for every
// outcome and the command layer owns the exit-code policy, which keeps the data
// payload intact for a failed job.
type GetOperationResult struct {
	Meta      *responsemeta.ResponseMeta
	Operation TagOperation
}

// Tag is the domain representation of a Censys tag, decoupled from the SDK type.
type Tag struct {
	ID          string    `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	Description *string   `json:"description,omitempty" yaml:"description,omitempty"`
	Privacy     string    `json:"privacy" yaml:"privacy"`
	CreatedBy   string    `json:"created_by" yaml:"created_by"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" yaml:"updated_at"`
	// AssetCount is only populated by `get`, which counts the assignments in a
	// second request; it stays nil elsewhere, and on a get whose count failed.
	AssetCount *int64 `json:"asset_count,omitempty" yaml:"asset_count,omitempty"`
}

// GetResult is the outcome of retrieving a single tag.
type GetResult struct {
	Meta *responsemeta.ResponseMeta
	Tag  Tag
	// PartialError is set when the tag was fetched but the opted-in asset count
	// could not be; the tag itself is still valid.
	PartialError cenclierrors.CencliError
}

// CreateResult is the outcome of creating a tag.
type CreateResult struct {
	Meta *responsemeta.ResponseMeta
	Tag  Tag
}

// UpdateResult is the outcome of updating a tag.
type UpdateResult struct {
	Meta *responsemeta.ResponseMeta
	Tag  Tag
}

// DeleteResult is the outcome of deleting a tag. The endpoint returns no tag
// body, so only the identifier the caller supplied is echoed back for rendering.
type DeleteResult struct {
	Meta  *responsemeta.ResponseMeta
	TagID string
}

// Assignment is the domain representation of a tag↔asset assignment.
type Assignment struct {
	ID          string    `json:"id" yaml:"id"`
	TagID       string    `json:"tag_id" yaml:"tag_id"`
	AssetID     string    `json:"asset_id" yaml:"asset_id"`
	AssetType   string    `json:"asset_type" yaml:"asset_type"`
	PlatformRef string    `json:"platform_ref" yaml:"platform_ref"`
	CreatedBy   string    `json:"created_by" yaml:"created_by"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
}

// AssignmentFailure records an asset that could not be assigned. Err keeps the
// full error; Detail and Status are the one-line summary a per-asset report
// shows, since a run of many assets cannot spend a problem document on each.
type AssignmentFailure struct {
	AssetID string
	Err     cenclierrors.CencliError
	Detail  string
	Status  mo.Option[int64]
}

// summarizedError is the part of an API error worth quoting per asset. The
// client's structured error implements it; our own typed per-asset errors do
// not, and fall back to their (already one-line) message.
type summarizedError interface {
	Detail() mo.Option[string]
	StatusCode() mo.Option[int64]
}

// newAssignmentFailure records a failed asset, reducing its error to the
// one-line form a per-asset report can display.
func newAssignmentFailure(assetID string, err cenclierrors.CencliError) AssignmentFailure {
	failure := AssignmentFailure{AssetID: assetID, Err: err, Detail: err.Error()}

	var summarized summarizedError
	if errors.As(err, &summarized) {
		if detail := summarized.Detail(); detail.IsPresent() {
			failure.Detail = detail.MustGet()
		}
		failure.Status = summarized.StatusCode()
	}
	return failure
}

// perAssetOutcome decides what a continue-on-error run reports alongside its
// per-asset results; Assign and Unassign share it so the two cannot drift. A run
// that got nowhere has nothing to render, so its error is fatal. Otherwise the
// results speak for themselves, and only a mixed run or an interrupt adds a
// partial error - an all-failed run is left for the command to exit non-zero on.
func perAssetOutcome(
	succeeded, failed int,
	firstErr cenclierrors.CencliError,
	summarize func() cenclierrors.CencliError,
) (partial, fatal cenclierrors.CencliError) {
	switch {
	case succeeded == 0 && failed == 0:
		return nil, firstErr
	case succeeded > 0 && failed > 0:
		return cenclierrors.ToPartialError(summarize()), nil
	case failed == 0 && firstErr != nil:
		return cenclierrors.ToPartialError(firstErr), nil
	default:
		return nil, nil
	}
}

// AssignResult is the outcome of assigning a tag to explicit assets. TagID
// echoes the caller's identifier; PartialError is set when some assets
// succeeded and others failed. Assignments and Failures are in input order.
type AssignResult struct {
	Meta         *responsemeta.ResponseMeta
	TagID        string
	Assignments  []Assignment
	Failures     []AssignmentFailure
	PartialError cenclierrors.CencliError
}

// UnassignResult is the outcome of unassigning a tag from explicit assets. TagID
// echoes the caller's identifier; Unassigned holds the removed assignments (from
// the per-asset lookup, so asset type is available for rendering) and Failures the
// assets that were not assigned or whose removal failed, both in input order.
type UnassignResult struct {
	Meta         *responsemeta.ResponseMeta
	TagID        string
	Unassigned   []Assignment
	Failures     []AssignmentFailure
	PartialError cenclierrors.CencliError
}

// AssignmentsResult is the outcome of listing a tag's assignments.
type AssignmentsResult struct {
	Meta *responsemeta.ResponseMeta
	// Empty in streaming mode, where each assignment is emitted as it arrives.
	Assignments []Assignment
	// TotalSize is the API's count of assignments matching the filters.
	TotalSize int64
	// PartialError summarizes an error hit after the first successful page.
	PartialError cenclierrors.CencliError
}

// ListResult is the outcome of listing tags.
type ListResult struct {
	Meta *responsemeta.ResponseMeta
	// Tags holds the tags fetched across pages, in server order.
	Tags []Tag
	// TotalSize is the total number of tags visible to the caller (from the API).
	TotalSize int64
	// PartialError summarizes an error encountered after the first successful
	// page; when present the result carries partial data.
	PartialError cenclierrors.CencliError
}
