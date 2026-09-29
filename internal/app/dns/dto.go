package dns

import (
	"time"

	"github.com/samber/mo"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

// Params bundles inputs for a DNS resolution lookup.
// Using a struct prevents parameter drift and keeps the API extensible.
type Params struct {
	OrgID    mo.Option[identifiers.OrganizationID]
	FromTime time.Time
	ToTime   time.Time
	// RecordTypes holds the record types as the user gave them; the service
	// validates and normalizes them. Empty means all supported types.
	RecordTypes []string
	PageSize    mo.Option[uint64]
	MaxPages    mo.Option[uint64]
	// Domain narrows an IP timeline lookup (IPResolutionRanges) to one domain
	// name that resolved to the IP. The other three lookups ignore it.
	Domain mo.Option[assets.DomainName]
}

// NameResolutionsResult holds one aggregated row per record of a name.
type NameResolutionsResult struct {
	Meta         *responsemeta.ResponseMeta
	Records      []*components.DNSResolutionRecord
	TotalRecords int64
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

// NameResolutionRangesResult holds one row per observed time range of a name's records.
type NameResolutionRangesResult struct {
	Meta         *responsemeta.ResponseMeta
	Records      []*components.DNSResolutionRangeRecord
	TotalRecords int64
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

// IPResolutionsResult holds one aggregated row per domain that resolved to an IP.
type IPResolutionsResult struct {
	Meta         *responsemeta.ResponseMeta
	Records      []*components.DNSIPResolutionRecord
	TotalRecords int64
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

// IPResolutionRangesResult holds one row per observed time range of the domains that resolved to an IP.
type IPResolutionRangesResult struct {
	Meta         *responsemeta.ResponseMeta
	Records      []*components.DNSIPResolutionRangeRecord
	TotalRecords int64
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}
