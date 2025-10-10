package view

import (
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

type HostsResult struct {
	Meta  *responsemeta.ResponseMeta
	Hosts []*assets.Host
	// PartialError contains any error encountered after the first successful batch.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

type CertificatesResult struct {
	Meta         *responsemeta.ResponseMeta
	Certificates []*assets.Certificate
	// PartialError contains any error encountered after the first successful batch.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

type WebPropertiesResult struct {
	Meta          *responsemeta.ResponseMeta
	WebProperties []*assets.WebProperty
	// PartialError contains any error encountered after the first successful batch.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}
