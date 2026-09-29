package history

import (
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/censys-sdk-go/models/components"
)

type HostHistoryResult struct {
	Meta   *responsemeta.ResponseMeta
	Events []*components.HostTimelineEvent
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

type CertificateHistoryResult struct {
	Meta   *responsemeta.ResponseMeta
	Ranges []*components.HostObservationRange
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

type WebPropertyHistoryResult struct {
	Meta   *responsemeta.ResponseMeta
	Events []*components.WebTimelineEvent
	// PartialError contains any error encountered after the first successful page.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}
