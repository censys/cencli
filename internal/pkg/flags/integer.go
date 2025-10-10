package flags

import (
	"fmt"

	"github.com/samber/mo"
	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

type IntegerFlag interface {
	// Value returns the current value of the flag.
	// If the flag is not provided, it will return the default value.
	// An optional value is returned to distinguish between 0 and not provided.
	Value() (mo.Option[int64], cenclierrors.CencliError)
}

type integerFlag struct {
	name         string
	raw          *int64
	parent       *pflag.FlagSet
	defaultValue mo.Option[int64]
	required     bool
	minValue     mo.Option[int64]
	maxValue     mo.Option[int64]
}

var _ IntegerFlag = (*integerFlag)(nil)

// NewIntegerFlag instantiates a new integer flag on a given flag set.
// Returns an IntegerFlag that can be used to get the value of the flag.
// required: whether the flag is required
// name: long flag name
// short: shorthand letter (or empty)
// defaultValue: default integer value (ignored if required)
// desc: user-facing description shown in help
// minValue: minimum allowed value (optional)
// maxValue: maximum allowed value (optional)
func NewIntegerFlag(
	flags *pflag.FlagSet,
	required bool,
	name string,
	short string,
	defaultValue mo.Option[int64],
	desc string,
	minValue mo.Option[int64],
	maxValue mo.Option[int64],
) *integerFlag {
	if required && defaultValue.IsPresent() {
		panic("flags: required integer flag cannot also have a default value: --" + name)
	}
	defaultValueInt := int64(0)
	if defaultValue.IsPresent() && !required {
		defaultValueInt = defaultValue.MustGet()
	}
	return &integerFlag{
		name:         name,
		raw:          flags.Int64P(name, short, defaultValueInt, desc),
		parent:       flags,
		defaultValue: defaultValue,
		required:     required,
		minValue:     minValue,
		maxValue:     maxValue,
	}
}

func (f *integerFlag) Value() (mo.Option[int64], cenclierrors.CencliError) {
	if !f.parent.Changed(f.name) {
		if f.required {
			return mo.None[int64](), NewRequiredFlagNotSetError(f.name)
		}
		if f.defaultValue.IsPresent() {
			return f.defaultValue, nil
		}
		return mo.None[int64](), nil
	}

	value := *f.raw

	// Validate minimum value
	if f.minValue.IsPresent() && value < f.minValue.MustGet() {
		return mo.None[int64](), NewIntegerFlagInvalidValueError(
			f.name,
			value,
			fmt.Sprintf("must be >= %d", f.minValue.MustGet()),
		)
	}

	// Validate maximum value
	if f.maxValue.IsPresent() && value > f.maxValue.MustGet() {
		return mo.None[int64](), NewIntegerFlagInvalidValueError(
			f.name,
			value,
			fmt.Sprintf("must be <= %d", f.maxValue.MustGet()),
		)
	}

	return mo.Some(value), nil
}

type IntegerFlagInvalidValueError interface {
	cenclierrors.CencliError
}

type integerFlagInvalidValueError struct {
	flagName string
	value    int64
	reason   string
}

var _ cenclierrors.CencliError = &integerFlagInvalidValueError{}

func NewIntegerFlagInvalidValueError(flagName string, value int64, reason string) IntegerFlagInvalidValueError {
	return &integerFlagInvalidValueError{
		flagName: flagName,
		value:    value,
		reason:   reason,
	}
}

func (e *integerFlagInvalidValueError) Error() string {
	return fmt.Sprintf("--%s was set with an invalid value: %d (%s)", e.flagName, e.value, e.reason)
}

func (e *integerFlagInvalidValueError) Title() string {
	return "Invalid Integer Flag Value"
}

func (e *integerFlagInvalidValueError) ShouldPrintUsage() bool {
	return true
}
