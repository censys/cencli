package flags

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/input"
)

// FileFlag represents a file-based command-line flag.
// Provides validation on retrieval via Lines().
type FileFlag interface {
	// Value returns the value of the flag.
	// Errors if the file does not exist or is not readable.
	// Special case: a value of "-" is treated as STDIN and returned as-is without filesystem validation.
	Value() (string, cenclierrors.CencliError)
	// IsSet returns true if the flag is set.
	IsSet() bool
	// Lines returns the lines of the file.
	// Takes in a cobra command in case it needs to access its stdin reader.
	// The command is not used if the flag value is a real file.
	Lines(*cobra.Command) ([]string, cenclierrors.CencliError)
}

type fileFlag struct {
	*stringFlag
}

var _ FileFlag = (*fileFlag)(nil)

// NewFileFlag instantiates a new file flag on a given flag set.
// Returns a FileFlag that can be used to get the value of the flag.
// When calling Value, an error of type InvalidFileFlagError will be returned if the file does not exist or is not readable.
// Special case: a value of "-" is treated as STDIN and returned as-is without filesystem validation.
func NewFileFlag(flags *pflag.FlagSet, required bool, name string, short string, desc string) FileFlag {
	return &fileFlag{
		stringFlag: NewStringFlag(flags, required, name, short, "", desc),
	}
}

func (f *fileFlag) Value() (string, cenclierrors.CencliError) {
	f.trimSpace()
	value, err := f.stringFlag.Value()
	if err != nil {
		return "", err
	}
	if value == "" && !f.stringFlag.required {
		return "", nil
	}
	// Support '-' sentinel for STDIN without validating file existence
	if value == input.StdInSentinel {
		return value, nil
	}
	// First ensure the path exists and is not a directory
	info, statErr := os.Stat(value)
	if statErr != nil {
		return "", NewInvalidFileFlagError(f.stringFlag.name, value, statErr)
	}
	if info.IsDir() {
		return "", NewInvalidFileFlagError(f.stringFlag.name, value, fmt.Errorf("is a directory"))
	}
	// Then verify readability by attempting to open
	file, openErr := os.Open(value)
	if openErr != nil {
		return "", NewInvalidFileFlagError(f.stringFlag.name, value, openErr)
	}
	defer file.Close()
	return file.Name(), nil
}

func (f *fileFlag) IsSet() bool {
	return f.stringFlag.wasProvided()
}

func (f *fileFlag) Lines(cmd *cobra.Command) ([]string, cenclierrors.CencliError) {
	value, err := f.Value()
	if err != nil {
		return nil, err
	}
	if value == input.StdInSentinel {
		return input.ReadLinesFromStdin(cmd.InOrStdin())
	}
	return input.ReadLinesFromFile(value)
}

type InvalidFileFlagError interface {
	cenclierrors.CencliError
}

type invalidFileFlagError struct {
	flagName  string
	flagValue string
	reason    string
}

var _ InvalidFileFlagError = &invalidFileFlagError{}

func NewInvalidFileFlagError(flagName string, flagValue string, err error) InvalidFileFlagError {
	return &invalidFileFlagError{flagName: flagName, flagValue: flagValue, reason: err.Error()}
}

func (e *invalidFileFlagError) Error() string {
	return fmt.Sprintf("--%s was set with an invalid file: %s (%s)", e.flagName, e.flagValue, e.reason)
}

func (e *invalidFileFlagError) Title() string {
	return "Invalid File"
}

func (e *invalidFileFlagError) ShouldPrintUsage() bool {
	return true
}
