package collections

import (
	"slices"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// SupportedStatuses is the collection_statuses filter the list endpoint accepts.
var SupportedStatuses = []string{"populating", "active", "paused", "archived"}

// validateStatuses rejects any status outside SupportedStatuses. Values are
// case-sensitive, the same as the tags enum filters.
func validateStatuses(statuses []string) cenclierrors.CencliError {
	for _, s := range statuses {
		if !slices.Contains(SupportedStatuses, s) {
			return NewInvalidEnumFilterError("status", s, SupportedStatuses)
		}
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
