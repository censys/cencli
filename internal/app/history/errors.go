package history

import (
	"fmt"
	"time"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// timelineStalledError signals that the timeline cursor stopped moving back
// before the window was exhausted, so older events may be missing.
type timelineStalledError struct {
	at time.Time
}

func newTimelineStalledError(at time.Time) cenclierrors.CencliError {
	return &timelineStalledError{at: at}
}

func (e *timelineStalledError) Error() string {
	return fmt.Sprintf("the timeline stopped advancing at %s, so older events in the window may be missing",
		e.at.UTC().Format(time.RFC3339))
}

func (e *timelineStalledError) Title() string { return "Incomplete History" }

func (e *timelineStalledError) ShouldPrintUsage() bool { return false }

// eventHistoryNotEnabledError signals that web property event history is
// behind a feature flag that is off for the caller's organization.
type eventHistoryNotEnabledError struct{}

func newEventHistoryNotEnabledError() cenclierrors.CencliError {
	return &eventHistoryNotEnabledError{}
}

func (e *eventHistoryNotEnabledError) Error() string {
	return "web property event history is not enabled for your organization; " +
		"drop --mode events to use the default snapshot history"
}

func (e *eventHistoryNotEnabledError) Title() string { return "Feature Not Enabled" }

func (e *eventHistoryNotEnabledError) ShouldPrintUsage() bool { return false }
