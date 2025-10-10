package command

import (
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// Essentially the same as cobra.PositionalArgs, but with its
// own error type.
type PositionalArgs func(cmd *cobra.Command, args []string) error

type ArgCountError interface {
	cenclierrors.CencliError
}

type argCountError struct {
	err error
}

func (e *argCountError) Error() string {
	return e.err.Error()
}

func (e *argCountError) Title() string {
	return "Incorrect Number of Arguments"
}

func (e *argCountError) ShouldPrintUsage() bool {
	return true
}

func NewArgCountError(err error) ArgCountError {
	return &argCountError{
		err: err,
	}
}

func ExactArgs(n int) PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(n)(cmd, args); err != nil {
			return NewArgCountError(err)
		}
		return nil
	}
}

func RangeArgs(min, max int) PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(min, max)(cmd, args); err != nil {
			return NewArgCountError(err)
		}
		return nil
	}
}
