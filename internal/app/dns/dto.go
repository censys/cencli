package dns

import (
	"encoding/json"
	"fmt"
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
	Records      []*NameRecord
	TotalRecords int64
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

// NameResolutionRangesResult holds one row per observed time range of a name's records.
type NameResolutionRangesResult struct {
	Meta         *responsemeta.ResponseMeta
	Records      []*NameRangeRecord
	TotalRecords int64
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

// IPResolutionsResult holds one aggregated row per domain that resolved to an IP.
type IPResolutionsResult struct {
	Meta         *responsemeta.ResponseMeta
	Records      []*IPRecord
	TotalRecords int64
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

// IPResolutionRangesResult holds one row per observed time range of the domains that resolved to an IP.
type IPResolutionRangesResult struct {
	Meta         *responsemeta.ResponseMeta
	Records      []*IPRangeRecord
	TotalRecords int64
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

// NameRecord is one name lookup record with the input (the normalized name)
// it answers. Input is encoded next to the SDK record's fields, so records
// from several inputs stay distinguishable in one output list.
type NameRecord struct {
	Input string `json:"input"`
	*components.DNSResolutionRecord
}

func (r NameRecord) MarshalJSON() ([]byte, error) {
	return marshalWithInput(r.Input, r.DNSResolutionRecord)
}

// NameRangeRecord is one name timeline record with the input it answers.
type NameRangeRecord struct {
	Input string `json:"input"`
	*components.DNSResolutionRangeRecord
}

func (r NameRangeRecord) MarshalJSON() ([]byte, error) {
	return marshalWithInput(r.Input, r.DNSResolutionRangeRecord)
}

// IPRecord is one IP lookup record with the input (the normalized IP) it answers.
type IPRecord struct {
	Input string `json:"input"`
	*components.DNSIPResolutionRecord
}

func (r IPRecord) MarshalJSON() ([]byte, error) {
	return marshalWithInput(r.Input, r.DNSIPResolutionRecord)
}

// IPRangeRecord is one IP timeline record with the input it answers.
type IPRangeRecord struct {
	Input string `json:"input"`
	*components.DNSIPResolutionRangeRecord
}

func (r IPRangeRecord) MarshalJSON() ([]byte, error) {
	return marshalWithInput(r.Input, r.DNSIPResolutionRangeRecord)
}

// marshalWithInput encodes record as one JSON object with "input" first,
// followed by the record's own fields. The SDK record types define
// MarshalJSON, and a struct that embeds one inherits that method, so without
// this encoding/json would call the inherited method and drop Input.
func marshalWithInput(input string, record any) ([]byte, error) {
	encodedInput, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	out := append([]byte(`{"input":`), encodedInput...)
	switch {
	case string(body) == "null" || string(body) == "{}":
		return append(out, '}'), nil
	case len(body) < 2 || body[0] != '{':
		return nil, fmt.Errorf("dns record for %s is not a JSON object", input)
	default:
		out = append(out, ',')
		return append(out, body[1:]...), nil
	}
}
