package dns

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// domainFlagMisuseError signals that --domain was given for a lookup that is
// not an IP timeline, where the API does not support the filter.
type domainFlagMisuseError struct{}

// NewDomainFlagMisuseError creates the error shown when --domain is set without an IP lookup and --timeline.
func NewDomainFlagMisuseError() cenclierrors.CencliError { return &domainFlagMisuseError{} }

func (e *domainFlagMisuseError) Error() string {
	return "--domain applies only to an IP lookup with --timeline"
}

func (e *domainFlagMisuseError) Title() string { return "Conflicting Flags" }

func (e *domainFlagMisuseError) ShouldPrintUsage() bool { return true }

// inputError names the input whose lookup failed, for a command with several
// inputs. It keeps the wrapped error's title and usage behavior.
type inputError struct {
	input string
	err   cenclierrors.CencliError
}

// newInputError creates an error that prefixes err's message with input.
func newInputError(input string, err cenclierrors.CencliError) cenclierrors.CencliError {
	return &inputError{input: input, err: err}
}

func (e *inputError) Error() string { return fmt.Sprintf("%s: %s", e.input, e.err.Error()) }

func (e *inputError) Title() string { return e.err.Title() }

func (e *inputError) ShouldPrintUsage() bool { return e.err.ShouldPrintUsage() }

func (e *inputError) Unwrap() error { return e.err }
