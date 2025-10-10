package flags

import (
	"fmt"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// DurationFlag represents a duration-based command-line flag.
type DurationFlag interface {
	// Value returns the current value of the flag.
	// If the flag is marked as required but not provided,
	// it returns an error of type RequiredFlagNotSetError.
	// If the flag has an invalid duration, it returns an error of type InvalidDurationFlagError.
	// An optional value is returned to keep callers from having to compare to 0.
	Value() (mo.Option[time.Duration], cenclierrors.CencliError)
}

type durationFlag struct {
	*stringFlag
	defaultValue mo.Option[time.Duration]
}

// NewDurationFlag instantiates a new duration flag on a given flag set.
// Returns a DurationFlag that can be used to get the value of the flag.
// Note that flags can be required or have default values, but it doesn't
// make sense to have both. Misuse will panic to surface programmer error early.
func NewDurationFlag(flags *pflag.FlagSet, required bool, name string, short string, defaultValue mo.Option[time.Duration], desc string) *durationFlag {
	if required && defaultValue.IsPresent() {
		panic("flags: required duration flag cannot also have a default value: --" + name)
	}
	var defaultStr string
	if defaultValue.IsPresent() {
		defaultStr = defaultValue.MustGet().String()
	}

	return &durationFlag{
		stringFlag:   NewStringFlag(flags, required, name, short, defaultStr, desc),
		defaultValue: defaultValue,
	}
}

func (f *durationFlag) Value() (mo.Option[time.Duration], cenclierrors.CencliError) {
	f.trimSpace()
	strValue, err := f.stringFlag.Value()
	if err != nil {
		return mo.None[time.Duration](), err
	}

	if !f.wasProvided() {
		return f.defaultValue, nil
	}

	duration, parseErr := time.ParseDuration(strValue)
	if parseErr != nil {
		return mo.None[time.Duration](), NewInvalidDurationFlagError(f.name, strValue)
	}

	return mo.Some(duration), nil
}

type InvalidDurationFlagError interface {
	cenclierrors.CencliError
}

type invalidDurationFlagError struct {
	flagName  string
	flagValue string
}

var _ cenclierrors.CencliError = &invalidDurationFlagError{}

func NewInvalidDurationFlagError(flagName, flagValue string) InvalidDurationFlagError {
	return &invalidDurationFlagError{flagName: flagName, flagValue: flagValue}
}

func (e *invalidDurationFlagError) Error() string {
	return fmt.Sprintf("--%s was set with an invalid duration: %s", e.flagName, e.flagValue)
}

func (e *invalidDurationFlagError) Title() string {
	return "Invalid Duration"
}

func (e *invalidDurationFlagError) ShouldPrintUsage() bool {
	return true
}
