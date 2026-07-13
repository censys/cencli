package tags

import (
	"slices"

	"github.com/samber/mo"

	"github.com/censys/censys-sdk-go/models/operations"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// validOrderBy is the set of accepted --order-by values, sourced from the SDK's
// generated enum so it stays in sync with the API contract.
var validOrderBy = []string{
	string(operations.V3TagsListTagsQueryParamOrderByNameAsc),
	string(operations.V3TagsListTagsQueryParamOrderByNameDesc),
	string(operations.V3TagsListTagsQueryParamOrderByCreatedAtAsc),
	string(operations.V3TagsListTagsQueryParamOrderByCreatedAtDesc),
	string(operations.V3TagsListTagsQueryParamOrderByUpdatedAtAsc),
	string(operations.V3TagsListTagsQueryParamOrderByUpdatedAtDesc),
}

// validPrivacy is the set of accepted --privacy values, sourced from the SDK's
// generated enum.
var validPrivacy = []string{
	string(operations.PrivacyPrivate),
	string(operations.PrivacyShared),
}

// validateOrderBy checks an optional order-by value against the accepted set. An
// absent value is valid (the filter is omitted from the request).
func validateOrderBy(orderBy mo.Option[string]) cenclierrors.CencliError {
	return validateEnumFilter("order-by", orderBy, validOrderBy)
}

// validatePrivacy checks an optional privacy value against the accepted set.
func validatePrivacy(privacy mo.Option[string]) cenclierrors.CencliError {
	return validateEnumFilter("privacy", privacy, validPrivacy)
}

func validateEnumFilter(filter string, value mo.Option[string], supported []string) cenclierrors.CencliError {
	if !value.IsPresent() {
		return nil
	}
	if !slices.Contains(supported, value.MustGet()) {
		return NewInvalidEnumFilterError(filter, value.MustGet(), supported)
	}
	return nil
}
