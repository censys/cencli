package collections

import (
	"fmt"
	"strings"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// invalidPaginationParamsError signals that a pagination parameter (page size or
// max pages) would fetch nothing.
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
// values (e.g. --status) was given an unsupported value.
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

// invalidCollectionIDError signals that a collection identifier is not a UUID.
// The endpoint declares collection_uid as a UUID, so anything else is rejected
// at the boundary instead of after a round trip.
type invalidCollectionIDError struct {
	provided string
}

// NewInvalidCollectionIDError creates an invalid-collection-ID error.
func NewInvalidCollectionIDError(provided string) cenclierrors.CencliError {
	return &invalidCollectionIDError{provided: provided}
}

func (e *invalidCollectionIDError) Error() string {
	if e.provided == "" {
		return "a collection ID is required"
	}
	return fmt.Sprintf("invalid collection ID %q; expected a UUID", e.provided)
}

func (e *invalidCollectionIDError) Title() string { return "Invalid Collection ID" }

func (e *invalidCollectionIDError) ShouldPrintUsage() bool { return true }

// invalidCollectionNameError signals that a collection name was empty or
// whitespace-only.
type invalidCollectionNameError struct{}

// NewInvalidCollectionNameError creates an invalid-collection-name error.
func NewInvalidCollectionNameError() cenclierrors.CencliError { return &invalidCollectionNameError{} }

func (e *invalidCollectionNameError) Error() string { return "collection name must not be empty" }

func (e *invalidCollectionNameError) Title() string { return "Invalid Collection Name" }

func (e *invalidCollectionNameError) ShouldPrintUsage() bool { return true }

// emptyQueryError signals that a collection was given a blank CenQL query.
type emptyQueryError struct{}

// NewEmptyQueryError creates an empty-query error.
func NewEmptyQueryError() cenclierrors.CencliError { return &emptyQueryError{} }

func (e *emptyQueryError) Error() string { return "--query must not be empty" }

func (e *emptyQueryError) Title() string { return "Invalid Query" }

func (e *emptyQueryError) ShouldPrintUsage() bool { return true }

// missingCollectionError signals that the API answered a read of a collection
// with no body. Update uses it to refuse to send a replacement built from
// empty values, which would overwrite the collection.
type missingCollectionError struct {
	collectionID string
}

// NewMissingCollectionError creates a missing-collection error.
func NewMissingCollectionError(collectionID string) cenclierrors.CencliError {
	return &missingCollectionError{collectionID: collectionID}
}

func (e *missingCollectionError) Error() string {
	return fmt.Sprintf("the API returned no data for collection %q; the update was not sent", e.collectionID)
}

func (e *missingCollectionError) Title() string { return "Collection Not Returned" }

func (e *missingCollectionError) ShouldPrintUsage() bool { return false }
