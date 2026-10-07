package collections

import (
	"slices"

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
