package pagination

import (
	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// ValidateParams rejects pagination values that would fetch nothing. It sets
// the rule every paginated list command shares.
func ValidateParams(pageSize, maxPages mo.Option[uint64]) cenclierrors.CencliError {
	if pageSize.IsPresent() && pageSize.MustGet() == 0 {
		return NewInvalidParamsError("page size must be greater than 0")
	}
	if maxPages.IsPresent() && maxPages.MustGet() == 0 {
		return NewInvalidParamsError("max pages must be greater than 0")
	}
	return nil
}

// OptionalInt64 narrows an unsigned page size to the signed type the client sends.
func OptionalInt64(v mo.Option[uint64]) mo.Option[int64] {
	if !v.IsPresent() {
		return mo.None[int64]()
	}
	return mo.Some(int64(v.MustGet()))
}

// invalidParamsError signals that a pagination parameter (page size or
// max pages) was given an invalid value.
type invalidParamsError struct {
	reason string
}

// NewInvalidParamsError creates an invalid-pagination-params error.
func NewInvalidParamsError(reason string) cenclierrors.CencliError {
	return &invalidParamsError{reason: reason}
}

func (e *invalidParamsError) Error() string { return e.reason }

func (e *invalidParamsError) Title() string { return "Invalid Pagination Parameters" }

func (e *invalidParamsError) ShouldPrintUsage() bool { return true }
