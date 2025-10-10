package search

import (
	"github.com/censys/censys-sdk-go/models/components"
	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

// Result is the response from the search service.
type Result struct {
	Meta      *responsemeta.ResponseMeta
	Hits      []assets.Asset
	TotalHits int64
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

// Params bundles inputs for performing a search query.
// Using a struct prevents parameter drift and keeps the API extensible.
type Params struct {
	OrgID        mo.Option[identifiers.OrganizationID]
	CollectionID mo.Option[identifiers.CollectionID]
	Query        string
	Fields       []string
	PageSize     mo.Option[uint64]
	MaxPages     mo.Option[uint64]
}

func parseHits(hits []components.SearchQueryHit) []assets.Asset {
	parsedHits := make([]assets.Asset, 0, len(hits))
	for _, hit := range hits {
		if cert := hit.GetCertificateV1(); cert != nil {
			asset := assets.NewCertificate(cert.GetResource())
			parsedHits = append(parsedHits, &asset)
		} else if host := hit.GetHostV1(); host != nil {
			asset := assets.NewHostWithMatchedServices(host.GetResource(), host.GetMatchedServices())
			parsedHits = append(parsedHits, &asset)
		} else if webProperty := hit.GetWebpropertyV1(); webProperty != nil {
			asset := assets.NewWebProperty(webProperty.GetResource())
			parsedHits = append(parsedHits, &asset)
		}
	}
	return parsedHits
}
