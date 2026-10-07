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

// domainFlagEmptyError signals that --domain was explicitly set to an empty
// (or all-whitespace) value. Unlike an absent flag, this must be rejected: an
// absent --domain leaves an IP timeline unfiltered on purpose, but a blank
// value (for example an unset shell variable passed through) would silently
// widen the lookup to every domain instead.
type domainFlagEmptyError struct{}

// NewDomainFlagEmptyError creates the error shown when --domain is explicitly
// set to an empty value.
func NewDomainFlagEmptyError() cenclierrors.CencliError { return &domainFlagEmptyError{} }

func (e *domainFlagEmptyError) Error() string { return "--domain needs a domain name" }

func (e *domainFlagEmptyError) Title() string { return "Invalid Flag Value" }

func (e *domainFlagEmptyError) ShouldPrintUsage() bool { return true }

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

// recordTypeReasonError appends a reason to a record-type error's message,
// explaining why the record type was rejected even though it is valid for
// some of the inputs.
type recordTypeReasonError struct {
	err    cenclierrors.CencliError
	reason string
}

// withRecordTypeReason wraps err (from dns.ValidateRecordTypes) with reason,
// appended in parentheses after its message.
func withRecordTypeReason(err cenclierrors.CencliError, reason string) cenclierrors.CencliError {
	return &recordTypeReasonError{err: err, reason: reason}
}

func (e *recordTypeReasonError) Error() string {
	return fmt.Sprintf("%s (%s)", e.err.Error(), e.reason)
}

func (e *recordTypeReasonError) Title() string { return e.err.Title() }

func (e *recordTypeReasonError) ShouldPrintUsage() bool { return e.err.ShouldPrintUsage() }

func (e *recordTypeReasonError) Unwrap() error { return e.err }
