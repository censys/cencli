package tags

import (
	"slices"
	"time"

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

// validAssignmentsOrderBy is the set of accepted --order-by values for
// assignments, which sort by creation time only — hence separate from validOrderBy.
var validAssignmentsOrderBy = []string{
	string(operations.V3TagsListAssignmentsQueryParamOrderByCreateTimeAsc),
	string(operations.V3TagsListAssignmentsQueryParamOrderByCreateTimeDesc),
}

// validAssetType is the set of accepted --asset-type values, from the SDK enum.
var validAssetType = []string{
	string(operations.AssetTypeHost),
	string(operations.AssetTypeWebProperty),
	string(operations.AssetTypeCertificate),
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

// validateAssignmentsOrderBy checks order-by against the assignments set.
func validateAssignmentsOrderBy(orderBy mo.Option[string]) cenclierrors.CencliError {
	return validateEnumFilter("order-by", orderBy, validAssignmentsOrderBy)
}

// validateAssetType checks an optional asset-type value against the accepted set.
func validateAssetType(assetType mo.Option[string]) cenclierrors.CencliError {
	return validateEnumFilter("asset-type", assetType, validAssetType)
}

// validateTimeWindow rejects an impossible created-before/created-after pair,
// which the API answers with an empty page — easily mistaken for "nothing matched".
func validateTimeWindow(before, after mo.Option[time.Time]) cenclierrors.CencliError {
	if !before.IsPresent() || !after.IsPresent() {
		return nil
	}
	if before.MustGet().Before(after.MustGet()) {
		return NewInvalidTimeWindowError("created-before must be after created-after")
	}
	return nil
}

// validatePaginationParams rejects pagination values that would fetch nothing.
func validatePaginationParams(pageSize, maxPages mo.Option[uint64]) cenclierrors.CencliError {
	if pageSize.IsPresent() && pageSize.MustGet() == 0 {
		return NewInvalidPaginationParamsError("page size must be greater than 0")
	}
	if maxPages.IsPresent() && maxPages.MustGet() == 0 {
		return NewInvalidPaginationParamsError("max pages must be greater than 0")
	}
	return nil
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
