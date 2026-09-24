package collections

import (
	"time"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

// ListParams bundles inputs for listing collections. An empty Statuses leaves
// the status filter off the request.
type ListParams struct {
	OrgID    mo.Option[identifiers.OrganizationID]
	Statuses []string
	PageSize mo.Option[uint64]
	MaxPages mo.Option[uint64]
}

// GetParams bundles inputs for retrieving a single collection.
type GetParams struct {
	OrgID        mo.Option[identifiers.OrganizationID]
	CollectionID identifiers.CollectionID
}

// CreateParams bundles inputs for creating a collection.
type CreateParams struct {
	OrgID       mo.Option[identifiers.OrganizationID]
	Name        string
	Query       string
	Description mo.Option[string]
}

// UpdateParams bundles inputs for updating a collection. Absent fields keep
// their current value. Clearing the description is expressed as
// Description = mo.Some("").
type UpdateParams struct {
	OrgID        mo.Option[identifiers.OrganizationID]
	CollectionID identifiers.CollectionID
	Name         mo.Option[string]
	Query        mo.Option[string]
	Description  mo.Option[string]
}

// DeleteParams bundles inputs for deleting a collection.
type DeleteParams struct {
	OrgID        mo.Option[identifiers.OrganizationID]
	CollectionID identifiers.CollectionID
}

// Collection is the domain representation of a Censys collection, decoupled
// from the SDK type.
type Collection struct {
	ID                   string    `json:"id" yaml:"id"`
	Name                 string    `json:"name" yaml:"name"`
	Description          string    `json:"description" yaml:"description"`
	Query                string    `json:"query" yaml:"query"`
	Status               string    `json:"status" yaml:"status"`
	StatusReason         *string   `json:"status_reason,omitempty" yaml:"status_reason,omitempty"`
	TotalAssets          int64     `json:"total_assets" yaml:"total_assets"`
	AddedAssets24Hours   int64     `json:"added_assets_24_hours" yaml:"added_assets_24_hours"`
	RemovedAssets24Hours int64     `json:"removed_assets_24_hours" yaml:"removed_assets_24_hours"`
	CreatedBy            *string   `json:"created_by,omitempty" yaml:"created_by,omitempty"`
	CreateTime           time.Time `json:"create_time" yaml:"create_time"`
}

// ListResult is the outcome of listing collections.
type ListResult struct {
	Meta        *responsemeta.ResponseMeta
	Collections []Collection
	// TotalSize is always 0: the list endpoint does not report a total.
	TotalSize int64
	// PartialError summarizes an error hit after the first successful page.
	PartialError cenclierrors.CencliError
}

// GetResult is the outcome of retrieving a single collection.
type GetResult struct {
	Meta       *responsemeta.ResponseMeta
	Collection Collection
}

// CreateResult is the outcome of creating a collection.
type CreateResult struct {
	Meta       *responsemeta.ResponseMeta
	Collection Collection
}

// UpdateResult is the outcome of updating a collection.
type UpdateResult struct {
	Meta       *responsemeta.ResponseMeta
	Collection Collection
}

// DeleteResult is the outcome of deleting a collection. The endpoint has no
// body, so only the identifier is echoed.
type DeleteResult struct {
	Meta         *responsemeta.ResponseMeta
	CollectionID string
}
