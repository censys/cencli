package flags

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// UUIDFlag represents a command-line flag that holds a UUID.
type UUIDFlag interface {
	// Value returns an optional value indicating the current value of the flag.
	// If the flag is marked as required but not provided,
	// it returns an error of type RequiredFlagNotSetError.
	// If the flag has an invalid UUID, it returns an error of type InvalidUUIDFlagError.
	// An optional value is returned to keep callers from having to compare to uuid.Nil.
	Value() (mo.Option[uuid.UUID], cenclierrors.CencliError)
}

type uuidFlag struct {
	*stringFlag
	defaultValue mo.Option[uuid.UUID]
}

var _ UUIDFlag = (*uuidFlag)(nil)

// NewUUIDFlag instantiates a new UUID flag on a given flag set.
// Returns a UUIDFlag that can be used to get the value of the flag.
// Note that flags can be required or have default values, but it doesn't
// make sense to have both. Misuse will panic to surface programmer error early.
// When calling Value, an error of type InvalidUUIDFlagError will be returned if the UUID is invalid.
func NewUUIDFlag(flags *pflag.FlagSet, required bool, name string, short string, defaultValue mo.Option[uuid.UUID], desc string) *uuidFlag {
	if required && defaultValue.IsPresent() {
		panic("flags: required UUID flag cannot also have a default value: --" + name)
	}
	defaultValueStr := ""
	if defaultValue.IsPresent() {
		defaultValueStr = defaultValue.MustGet().String()
	}
	return &uuidFlag{
		stringFlag:   NewStringFlag(flags, required, name, short, defaultValueStr, desc),
		defaultValue: defaultValue,
	}
}

func (f *uuidFlag) Value() (mo.Option[uuid.UUID], cenclierrors.CencliError) {
	f.trimSpace()
	value, err := f.stringFlag.Value()
	if err != nil {
		return mo.None[uuid.UUID](), err
	}
	if value == "" && !f.stringFlag.required {
		return f.defaultValue, nil
	}
	uid, parseErr := uuid.Parse(value)
	if parseErr != nil {
		return mo.None[uuid.UUID](), NewInvalidUUIDFlagError(f.stringFlag.name, value)
	}
	return mo.Some(uid), nil
}

type InvalidUUIDFlagError interface {
	cenclierrors.CencliError
}

type invalidUUIDFlagError struct {
	flagName  string
	flagValue string
}

var _ InvalidUUIDFlagError = &invalidUUIDFlagError{}

func NewInvalidUUIDFlagError(flagName string, flagValue string) InvalidUUIDFlagError {
	return &invalidUUIDFlagError{flagName: flagName, flagValue: flagValue}
}

func (e *invalidUUIDFlagError) Error() string {
	return fmt.Sprintf("--%s was set with an invalid uuid: %s", e.flagName, e.flagValue)
}

func (e *invalidUUIDFlagError) Title() string {
	return "Invalid UUID"
}

func (e *invalidUUIDFlagError) ShouldPrintUsage() bool {
	return true
}
