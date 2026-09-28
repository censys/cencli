package dns

import (
	"fmt"
	"strings"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// invalidRecordTypeError signals a --record-type value that the lookup
// direction does not support.
type invalidRecordTypeError struct {
	provided  string
	supported []string
}

// NewInvalidRecordTypeError creates an error naming the rejected record type and the accepted set.
func NewInvalidRecordTypeError(provided string, supported []string) cenclierrors.CencliError {
	return &invalidRecordTypeError{provided: provided, supported: supported}
}

func (e *invalidRecordTypeError) Error() string {
	return fmt.Sprintf("invalid record type '%s'; supported values: %s", e.provided, strings.Join(e.supported, ", "))
}

func (e *invalidRecordTypeError) Title() string { return "Invalid Record Type" }

func (e *invalidRecordTypeError) ShouldPrintUsage() bool { return true }

// accessDeniedError explains a 403 from the DNS endpoints, which are limited
// to some plans. The generic API error would not say why access failed.
type accessDeniedError struct{}

// NewAccessDeniedError creates the error shown when the DNS endpoints return 403.
func NewAccessDeniedError() cenclierrors.CencliError {
	return &accessDeniedError{}
}

func (e *accessDeniedError) Error() string {
	return "Censys returned 403 Forbidden. Active DNS is available only on the Censys Search and Core plans. " +
		"Check your plan, or use --org-id to select an organization with access."
}

func (e *accessDeniedError) Title() string { return "Active DNS Not Available" }

func (e *accessDeniedError) ShouldPrintUsage() bool { return false }

// invalidPaginationParamsError signals pagination values that would fetch
// nothing.
type invalidPaginationParamsError struct {
	reason string
}

// NewInvalidPaginationParamsError creates an error for pagination values that would fetch nothing.
func NewInvalidPaginationParamsError(reason string) cenclierrors.CencliError {
	return &invalidPaginationParamsError{reason: reason}
}

func (e *invalidPaginationParamsError) Error() string { return e.reason }

func (e *invalidPaginationParamsError) Title() string { return "Invalid Pagination Parameters" }

func (e *invalidPaginationParamsError) ShouldPrintUsage() bool { return true }
