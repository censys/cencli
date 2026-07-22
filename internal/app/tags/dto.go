package tags

import (
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

// Tag is the domain representation of a Censys tag, decoupled from the SDK type.
type Tag struct {
	ID          string    `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	Description *string   `json:"description,omitempty" yaml:"description,omitempty"`
	Privacy     string    `json:"privacy" yaml:"privacy"`
	CreatedBy   string    `json:"created_by" yaml:"created_by"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" yaml:"updated_at"`
}

// GetResult is the outcome of retrieving a single tag.
type GetResult struct {
	Meta *responsemeta.ResponseMeta
	Tag  Tag
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

// AssignmentFailure records an asset that could not be assigned.
type AssignmentFailure struct {
	AssetID string
	Err     cenclierrors.CencliError
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
