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
