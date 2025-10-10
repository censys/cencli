package flags

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// StringSliceFlag represents a string slice-based command-line flag.
// Allows multiple values to be provided by specifying the flag multiple times.
// Provides validation on retrieval via Value().
type StringSliceFlag interface {
	// Value returns the current slice of values for the flag.
	// If the flag is marked as required but not provided,
	// it returns an error of type RequiredFlagNotSetError.
	Value() ([]string, cenclierrors.CencliError)
}

type stringSliceFlag struct {
	name     string
	raw      *[]string
	parent   *pflag.FlagSet
	required bool
}

var _ StringSliceFlag = (*stringSliceFlag)(nil)

// NewStringSliceFlag instantiates a new string slice flag on a given flag set.
// Returns a StringSliceFlag that can be used to get the slice of values.
// The flag can be specified multiple times and values will be accumulated.
// name: long flag name (e.g., "tags")
// short: shorthand letter (or empty)
// defaultValue: default slice of strings (ignored if required)
// desc: user-facing description shown in help
func NewStringSliceFlag(flags *pflag.FlagSet, required bool, name, short string, defaultValue []string, desc string) *stringSliceFlag {
	if required && defaultValue != nil && len(defaultValue) > 0 {
		panic("flags: required string slice flag cannot also have a default value: --" + name)
	}
	var defaultVal []string
	if !required && defaultValue != nil {
		defaultVal = make([]string, len(defaultValue))
		copy(defaultVal, defaultValue)
	}

	return &stringSliceFlag{
		name:     name,
		raw:      flags.StringSliceP(name, short, defaultVal, desc),
		required: required,
		parent:   flags,
	}
}

func (f *stringSliceFlag) Value() ([]string, cenclierrors.CencliError) {
	// Return a copy to prevent external modification
	result := make([]string, len(*f.raw))
	copy(result, *f.raw)

	// Trim whitespace from all values
	for i, val := range result {
		result[i] = strings.TrimSpace(val)
	}

	// Check if required flag is not provided or results in empty slice
	if f.required && (!f.wasProvided() || len(result) == 0) {
		return nil, NewRequiredFlagNotSetError(f.name)
	}

	return result, nil
}

func (f *stringSliceFlag) wasProvided() bool {
	return f.parent.Changed(f.name)
}
