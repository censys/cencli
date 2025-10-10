package flags

import (
	"fmt"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/datetime"
)

// TimestampFlag represents a timestamp-based command-line flag.
type TimestampFlag interface {
	// Value returns the current value of the flag, using the
	// default timezone if no timezone can be inferred.
	// If the flag is marked as required but not provided,
	// it returns an error of type RequiredFlagNotSetError.
	// If the flag has an invalid RFC3339 timestamp, it returns an error of type InvalidRFC3339TimestampFlagError.
	// An optional value is returned to keep callers from having to use IsZero().
	Value(defaultTZ datetime.TimeZone) (mo.Option[time.Time], cenclierrors.CencliError)
	// AddAlias registers an additional flag name and optional shorthand that
	// map to the same underlying value. Useful for supporting synonyms like
	// "--at" and "-a" for an "--at-time" flag.
	AddAlias(name string, short string, desc string) TimestampFlag
}

type timestampFlag struct {
	*stringFlag
	defaultValue mo.Option[time.Time]
}

var _ TimestampFlag = (*timestampFlag)(nil)

// NewTimestampFlag instantiates a new timestamp flag on a given flag set.
// Returns a TimestampFlag that can be used to get the value of the flag.
// Note that flags can be required or have default values, but it doesn't
// make sense to have both. Misuse will panic to surface programmer error early.
func NewTimestampFlag(flags *pflag.FlagSet, required bool, name string, short string, defaultValue mo.Option[time.Time], desc string) *timestampFlag {
	if required && defaultValue.IsPresent() {
		panic("flags: required timestamp flag cannot also have a default value: --" + name)
	}
	defaultValueStr := ""
	if defaultValue.IsPresent() {
		defaultValueStr = defaultValue.MustGet().Format(time.RFC3339)
	}
	return &timestampFlag{
		stringFlag:   NewStringFlag(flags, required, name, short, defaultValueStr, desc),
		defaultValue: defaultValue,
	}
}

func (f *timestampFlag) Value(defaultTZ datetime.TimeZone) (mo.Option[time.Time], cenclierrors.CencliError) {
	f.trimSpace()
	strValue, err := f.stringFlag.Value()
	if err != nil {
		return mo.None[time.Time](), err
	}
	if !f.wasProvided() {
		return f.defaultValue, nil
	}
	timestamp, parseErr := datetime.Parse(strValue, defaultTZ)
	if parseErr != nil {
		return mo.None[time.Time](), NewInvalidTimestampFlagError(f.stringFlag.name, strValue)
	}
	return mo.Some(timestamp), nil
}

// AddAlias registers an additional flag that shares the same underlying value pointer.
// This allows commands to accept synonymous flags (e.g., --at-time and --at / -a).
func (f *timestampFlag) AddAlias(name string, short string, desc string) TimestampFlag {
	if f == nil || f.stringFlag == nil || f.stringFlag.parent == nil || f.stringFlag.raw == nil {
		return f
	}
	// Bind the alias to the same backing variable
	f.stringFlag.parent.StringVarP(f.stringFlag.raw, name, short, "", desc)
	return f
}

type InvalidTimestampFlagError interface {
	cenclierrors.CencliError
}

type invalidTimestampFlagError struct {
	flagName  string
	flagValue string
}

var _ cenclierrors.CencliError = &invalidTimestampFlagError{}

func NewInvalidTimestampFlagError(flagName, flagValue string) InvalidTimestampFlagError {
	return &invalidTimestampFlagError{flagName: flagName, flagValue: flagValue}
}

func (e *invalidTimestampFlagError) Error() string {
	return fmt.Sprintf("--%s was set with an invalid timestamp: %s", e.flagName, e.flagValue)
}

func (e *invalidTimestampFlagError) Title() string {
	return "Invalid Timestamp"
}

func (e *invalidTimestampFlagError) ShouldPrintUsage() bool {
	return true
}
