package enrich

import (
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

// HostFailure records a single host IP that failed to enrich, with its error.
type HostFailure struct {
	HostID assets.HostID
	Err    cenclierrors.CencliError
}

// Result is the outcome of enriching one or more host IPs.
type Result struct {
	Meta *responsemeta.ResponseMeta
	// Hosts holds successful enrichments in input order. It is populated only in
	// non-streaming mode; in streaming mode results are emitted as they arrive
	// and this slice is empty (mirrors the view service).
	Hosts []*assets.EnrichedHost
	// Failures holds per-IP failures in input order.
	Failures []HostFailure
	// PartialError summarizes the run for the existing stderr-reporting path:
	// nil when every IP succeeded, the daily-limit error when a 429 short-circuited
	// the run, otherwise a summary when some hosts succeeded and some failed.
	PartialError cenclierrors.CencliError
}
