package tags

import (
	"fmt"
	"strings"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// invalidPaginationParamsError signals that a pagination parameter (page size or
// max pages) was given an invalid value.
type invalidPaginationParamsError struct {
	reason string
}

// NewInvalidPaginationParamsError creates an invalid-pagination-params error.
func NewInvalidPaginationParamsError(reason string) cenclierrors.CencliError {
	return &invalidPaginationParamsError{reason: reason}
}

func (e *invalidPaginationParamsError) Error() string { return e.reason }

func (e *invalidPaginationParamsError) Title() string { return "Invalid Pagination Parameters" }

func (e *invalidPaginationParamsError) ShouldPrintUsage() bool { return true }

// invalidEnumFilterError signals that a filter flag with a fixed set of accepted
// values (e.g. --order-by, --privacy) was given an unsupported value.
type invalidEnumFilterError struct {
	filter    string
	provided  string
	supported []string
}

// NewInvalidEnumFilterError creates an invalid-enum-filter error naming the flag,
// the rejected value, and the accepted set.
func NewInvalidEnumFilterError(filter, provided string, supported []string) cenclierrors.CencliError {
	return &invalidEnumFilterError{filter: filter, provided: provided, supported: supported}
}

func (e *invalidEnumFilterError) Error() string {
	return fmt.Sprintf("invalid %s '%s'; supported values: %s", e.filter, e.provided, strings.Join(e.supported, ", "))
}

func (e *invalidEnumFilterError) Title() string { return "Invalid Filter Value" }

func (e *invalidEnumFilterError) ShouldPrintUsage() bool { return true }

// emptyTagIDError signals that a tag command was given an empty tag identifier
// (name or UUID). Rejected before any lookup so an empty name can never be sent
// to ListTags — which would otherwise match no filter and resolve to an
// arbitrary tag.
type emptyTagIDError struct{}

// NewEmptyTagIDError creates an empty-tag-identifier error.
func NewEmptyTagIDError() cenclierrors.CencliError { return &emptyTagIDError{} }

func (e *emptyTagIDError) Error() string { return "a tag name or ID is required" }

func (e *emptyTagIDError) Title() string { return "Invalid Tag" }

func (e *emptyTagIDError) ShouldPrintUsage() bool { return true }

// tagNotFoundError signals that a tag name could not be resolved to an existing
// tag during a name→UUID lookup.
type tagNotFoundError struct {
	name string
}

// NewTagNotFoundError creates a tag-not-found error naming the unresolved tag.
func NewTagNotFoundError(name string) cenclierrors.CencliError {
	return &tagNotFoundError{name: name}
}

func (e *tagNotFoundError) Error() string {
	return fmt.Sprintf("tag %q not found", e.name)
}

func (e *tagNotFoundError) Title() string { return "Tag Not Found" }

func (e *tagNotFoundError) ShouldPrintUsage() bool { return false }

// invalidTagNameError signals that a tag name was empty or whitespace-only.
type invalidTagNameError struct{}

// NewInvalidTagNameError creates an invalid-tag-name error.
func NewInvalidTagNameError() cenclierrors.CencliError { return &invalidTagNameError{} }

func (e *invalidTagNameError) Error() string { return "tag name must not be empty" }

func (e *invalidTagNameError) Title() string { return "Invalid Tag Name" }

func (e *invalidTagNameError) ShouldPrintUsage() bool { return true }
