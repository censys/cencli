package enrich

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// dailyLimitError wraps the underlying 429 with a friendlier title/message while
// preserving the API detail.
type dailyLimitError struct {
	underlying cenclierrors.CencliError
}

func newDailyLimitError(err cenclierrors.CencliError) cenclierrors.CencliError {
	return &dailyLimitError{underlying: err}
}

func (e *dailyLimitError) Error() string {
	return "the daily host enrichment limit has been reached; some hosts may not have been enriched\n\n" + e.underlying.Error()
}

func (e *dailyLimitError) Title() string { return "Daily Enrichment Limit Reached" }

func (e *dailyLimitError) ShouldPrintUsage() bool { return false }

// partialFailureError summarizes a run where some hosts enriched and some failed.
type partialFailureError struct {
	failed int
	total  int
}

func newPartialFailureError(failed, total int) cenclierrors.CencliError {
	return &partialFailureError{failed: failed, total: total}
}

func (e *partialFailureError) Error() string {
	return fmt.Sprintf("%d of %d host(s) failed to enrich", e.failed, e.total)
}

func (e *partialFailureError) Title() string { return "Some Hosts Failed to Enrich" }

func (e *partialFailureError) ShouldPrintUsage() bool { return false }
