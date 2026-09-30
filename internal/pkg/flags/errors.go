package flags

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

type ConflictingFlagsError interface {
	cenclierrors.CencliError
}

type conflictingFlagsError struct {
	flag1 string
	flag2 string
}

var _ ConflictingFlagsError = &conflictingFlagsError{}

func NewConflictingFlagsError(flag1 string, flag2 string) ConflictingFlagsError {
	return &conflictingFlagsError{flag1: flag1, flag2: flag2}
}

func (e *conflictingFlagsError) Error() string {
	return fmt.Sprintf("cannot use --%s and --%s flags together", e.flag1, e.flag2)
}

func (e *conflictingFlagsError) Title() string {
	return "Conflicting Flags"
}

func (e *conflictingFlagsError) ShouldPrintUsage() bool {
	return true
}

// InvalidTimeWindowError represents a --start/--end/--duration combination
// that gives no valid window.
type InvalidTimeWindowError interface {
	cenclierrors.CencliError
}

type invalidTimeWindowError struct {
	reason string
}

var _ InvalidTimeWindowError = &invalidTimeWindowError{}

func NewInvalidTimeWindowError(reason string) InvalidTimeWindowError {
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
