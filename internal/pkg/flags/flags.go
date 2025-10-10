package flags

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// StringFlag represents a string-based command-line flag.
// Provides validation on retrieval via Value().
type StringFlag interface {
	// Value returns the current value of the flag.
	// If the flag is marked as required but not provided,
	// it returns an error of type RequiredFlagNotSetError.
	Value() (string, cenclierrors.CencliError)
}

type stringFlag struct {
	name     string
	raw      *string
	parent   *pflag.FlagSet
	required bool
}

var _ StringFlag = (*stringFlag)(nil)

// NewStringFlag instantiates a new string flag on a given flag set.
// Returns a StringFlag that can be used to get the value of the flag.
// Note that flags can be required or have default values, but it doesn't
// make sense to have both. Misuse will panic to surface programmer error early.
// name: long flag name (e.g., "organization-id")
// short: shorthand letter (or empty)
// defaultValue: default string value (ignored if required)
// desc: user-facing description shown in help
func NewStringFlag(flags *pflag.FlagSet, required bool, name, short, defaultValue, desc string) *stringFlag {
	if required && defaultValue != "" {
		panic("flags: required string flag cannot also have a default value: --" + name)
	}
	return &stringFlag{
		name:     name,
		raw:      flags.StringP(name, short, defaultValue, desc),
		required: required,
		parent:   flags,
	}
}

func (f *stringFlag) Value() (string, cenclierrors.CencliError) {
	if (!f.parent.Changed(f.name) || *f.raw == "") && f.required {
		return "", NewRequiredFlagNotSetError(f.name)
	}
	return *f.raw, nil
}

func (f *stringFlag) wasProvided() bool {
	return f.parent.Changed(f.name)
}

func (f *stringFlag) trimSpace() {
	*f.raw = strings.TrimSpace(*f.raw)
}

// BoolFlag represents a boolean-based command-line flag.
type BoolFlag interface {
	// Value returns the current value of the flag.
	// If the flag is not provided, it will return the default value.
	Value() (bool, cenclierrors.CencliError)
}

var _ BoolFlag = (*boolFlag)(nil)

type boolFlag struct {
	name   string
	raw    *bool
	parent *pflag.FlagSet
}

// NewBoolFlag instantiates a new boolean flag on a given flag set.
// Returns a BoolFlag that can be used to get the value of the flag.
// name: long flag name
// short: shorthand letter (or empty)
// defaultValue: default boolean value
// desc: user-facing description shown in help
func NewBoolFlag(flags *pflag.FlagSet, name string, short string, defaultValue bool, desc string) *boolFlag {
	return &boolFlag{
		name:   name,
		raw:    flags.BoolP(name, short, defaultValue, desc),
		parent: flags,
	}
}

func (f *boolFlag) Value() (bool, cenclierrors.CencliError) {
	return *f.raw, nil
}

type RequiredFlagNotSetError interface {
	cenclierrors.CencliError
}

type requiredFlagNotSetError struct {
	flagName string
}

var _ cenclierrors.CencliError = &requiredFlagNotSetError{}

func NewRequiredFlagNotSetError(flagName string) RequiredFlagNotSetError {
	return &requiredFlagNotSetError{flagName: flagName}
}

func (e *requiredFlagNotSetError) Error() string {
	return "--" + e.flagName + " is required"
}

func (e *requiredFlagNotSetError) Title() string {
	return "Required Flag Not Set"
}

func (e *requiredFlagNotSetError) ShouldPrintUsage() bool {
	return true
}
