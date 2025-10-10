package search

import (
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

type InvalidPaginationParamsError interface {
	cenclierrors.CencliError
}

type invalidPaginationParamsError struct {
	reason string
}

var _ InvalidPaginationParamsError = &invalidPaginationParamsError{}

func NewInvalidPaginationParamsError(reason string) InvalidPaginationParamsError {
	return &invalidPaginationParamsError{reason: reason}
}

func (e *invalidPaginationParamsError) Error() string {
	return e.reason
}

func (e *invalidPaginationParamsError) Title() string {
	return "Invalid Pagination Params"
}

func (e *invalidPaginationParamsError) ShouldPrintUsage() bool {
	return true
}
