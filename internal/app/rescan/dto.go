package rescan

import (
	"github.com/censys/censys-sdk-go/models/components"
	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

// TargetType selects which kind of rescan target to build.
type TargetType int

const (
	TargetTypeService TargetType = iota
	TargetTypeWebOrigin
)

// Params bundles inputs for initiating a live rescan.
type Params struct {
	OrgID      mo.Option[identifiers.OrganizationID]
	TargetType TargetType
	// service target fields
	IP                string
	Port              int
	Protocol          string
	TransportProtocol components.TargetTransportProtocol
	// web origin target fields
	Hostname string
}

// Result is the final scan state returned after completion.
type Result struct {
	Meta *responsemeta.ResponseMeta
	Scan *components.TrackedScan
}
