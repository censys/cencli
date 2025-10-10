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
