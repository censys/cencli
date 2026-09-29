package command

import (
	"fmt"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/flags"
)

// ParseWaitFlags reads the --wait/--timeout pair every command that can follow
// a long-running job shares. A zero timeout means "no limit", matching the
// global --timeout-http, and a negative one is rejected rather than expiring
// before the first poll.
func ParseWaitFlags(
	cmd *cobra.Command,
	waitFlag flags.BoolFlag,
	timeoutFlag flags.HumanDurationFlag,
) (bool, mo.Option[time.Duration], cenclierrors.CencliError) {
	none := mo.None[time.Duration]()

	wait, err := waitFlag.Value()
	if err != nil {
		return false, none, err
	}

	timeout, err := timeoutFlag.Value()
	if err != nil {
		return false, none, err
	}

	// A timeout only means something while polling; silently ignoring it would
	// make the flag look like it worked.
	if !wait && cmd.Flags().Changed("timeout") {
		return false, none, NewTimeoutWithoutWaitError()
	}

	if timeout.IsPresent() {
		switch d := timeout.MustGet(); {
		case d < 0:
			return false, none, NewInvalidWaitTimeoutError(d)
		case d == 0:
			timeout = none
		}
	}

	return wait, timeout, nil
}

// timeoutWithoutWaitError signals that --timeout was set without --wait, where
// it would have no effect.
type timeoutWithoutWaitError struct{}

func NewTimeoutWithoutWaitError() cenclierrors.CencliError { return &timeoutWithoutWaitError{} }

func (e *timeoutWithoutWaitError) Error() string {
	return "--timeout only applies while polling; add --wait or drop --timeout"
}

func (e *timeoutWithoutWaitError) Title() string { return "Conflicting Flags" }

func (e *timeoutWithoutWaitError) ShouldPrintUsage() bool { return true }

// invalidWaitTimeoutError signals a negative --timeout, which would give up
// before the first poll ever ran.
type invalidWaitTimeoutError struct {
	value time.Duration
}

func NewInvalidWaitTimeoutError(value time.Duration) cenclierrors.CencliError {
	return &invalidWaitTimeoutError{value: value}
}

func (e *invalidWaitTimeoutError) Error() string {
	return fmt.Sprintf("--timeout must not be negative (got %s); use 0 to wait without a time limit", e.value)
}

func (e *invalidWaitTimeoutError) Title() string { return "Invalid Timeout" }

func (e *invalidWaitTimeoutError) ShouldPrintUsage() bool { return true }
