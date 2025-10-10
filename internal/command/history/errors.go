package history

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

type InvalidTimeWindowError interface {
	cenclierrors.CencliError
}

type invalidTimeWindowError struct {
	reason string
}

func newInvalidTimeWindowError(reason string) InvalidTimeWindowError {
	return &invalidTimeWindowError{reason: reason}
}

func (e *invalidTimeWindowError) Error() string {
	return fmt.Sprintf("invalid time window: %s", e.reason)
}

func (e *invalidTimeWindowError) Title() string {
	return "Invalid Time Window"
}

func (e *invalidTimeWindowError) ShouldPrintUsage() bool {
	return true
}
