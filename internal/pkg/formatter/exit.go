package formatter

import (
	"errors"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// ExitCode maps an error to a conventional CLI exit code.
// 0: success
// 2: usage/config/input error (print usage)
// 124: timeout
// 130: interrupted (canceled)
// 1: general error
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	// Check for interruption first (context.Canceled or ErrInterrupted)
	if cenclierrors.IsInterrupted(err) {
		return 130
	}
	// Context-derived errors
	if cenclierrors.IsDeadlineExceeded(err) {
		return 124
	}
	// Domain errors
	var ce cenclierrors.CencliError
	if errors.As(err, &ce) {
		if ce.ShouldPrintUsage() {
			return 2
		}
		return 1
	}
	return 1
}
