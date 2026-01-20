package command

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
)

// OutputSupport defines what output formats a command supports and its default.
type OutputSupport struct {
	Default OutputType
}

// OutputType is the type of output a command supports.
type OutputType int

const (
	// OutputTypeData is the output type for commands that output buffered raw data (json, yaml, tree)
	OutputTypeData OutputType = iota
	// OutputTypeShort is the output type for commands that output a short view (i.e. a custom rendering)
	OutputTypeShort
	// OutputTypeTemplate is the output type for commands that output a template view (i.e. a handlebars template)
	OutputTypeTemplate
)

func validateOutputFormat(format formatter.OutputFormat, cmd Command) cenclierrors.CencliError {
	supportedTypes := cmd.SupportedOutputTypes()

	// Build list of supported formats for this command
	var supportedFormats []string
	for _, t := range supportedTypes {
		switch t {
		case OutputTypeData:
			supportedFormats = append(supportedFormats,
				formatter.OutputFormatJSON.String(),
				formatter.OutputFormatYAML.String(),
				formatter.OutputFormatTree.String(),
			)
		case OutputTypeShort:
			supportedFormats = append(supportedFormats, formatter.OutputFormatShort.String())
		case OutputTypeTemplate:
			supportedFormats = append(supportedFormats, formatter.OutputFormatTemplate.String())
		}
	}

	var requestedOutputType OutputType
	switch format {
	case formatter.OutputFormatJSON, formatter.OutputFormatYAML, formatter.OutputFormatTree:
		requestedOutputType = OutputTypeData
	case formatter.OutputFormatShort:
		requestedOutputType = OutputTypeShort
	case formatter.OutputFormatTemplate:
		requestedOutputType = OutputTypeTemplate
	default:
		// Invalid format - show only formats supported by this command
		return newInvalidOutputFormatError(format.String(), supportedFormats)
	}

	// Check if command supports this type
	if slices.Contains(supportedTypes, requestedOutputType) {
		return nil
	}

	// Valid format but not supported by this command
	return newUnsupportedOutputFormatError(format.String(), supportedFormats)
}

func getOutputFormatValue(cobraCmd *cobra.Command, cmd Command, valueFromConfig formatter.OutputFormat) formatter.OutputFormat {
	flag := cobraCmd.Flag(formatter.OutputFormatFlagName)

	if flag == nil {
		return valueFromConfig
	}

	if flag.Changed {
		return formatter.OutputFormat(flag.Value.String())
	}

	// User didn't explicitly set the output format, so apply the command's default if it has one
	defaultOutputType := cmd.DefaultOutputType()
	switch defaultOutputType {
	case OutputTypeData:
		return valueFromConfig
	case OutputTypeShort:
		return formatter.OutputFormatShort
	case OutputTypeTemplate:
		return formatter.OutputFormatTemplate
	default:
		return valueFromConfig
	}
}

// applyOutputFormatDefaultsRecursive applies output format defaults to a command
// and all its subcommands recursively. This must be called after the command has
// been added to its parent so inherited flags are available.
func applyOutputFormatDefaultsRecursive(cobraCmd *cobra.Command, cmd Command) error {
	// Apply to this command first
	if err := applyOutputFormatDefaults(cobraCmd, cmd); err != nil {
		return err
	}

	// Recursively apply to all subcommands
	for _, subCmd := range cobraCmd.Commands() {
		if err := applyOutputFormatDefaultsRecursive(subCmd, nil); err != nil {
			return err
		}
	}

	return nil
}

// applyOutputFormatDefaults adjusts the --output-format flag's default value
// based on the command's DefaultOutputType(). For commands with a custom default,
// we add a local flag that shadows the inherited one to avoid affecting siblings.
func applyOutputFormatDefaults(cobraCmd *cobra.Command, cmd Command) error {
	if cmd == nil {
		return nil
	}

	var defaultFormat formatter.OutputFormat
	switch cmd.DefaultOutputType() {
	case OutputTypeData:
		// no specialdefault output type, so we don't need to add a local flag
		return nil
	case OutputTypeShort:
		defaultFormat = formatter.OutputFormatShort
	case OutputTypeTemplate:
		defaultFormat = formatter.OutputFormatTemplate
	}

	// Check if the flag already exists (to avoid redefinition)
	if cobraCmd.PersistentFlags().Lookup(formatter.OutputFormatFlagName) != nil {
		return nil
	}

	// Add a local persistent flag that shadows the inherited --output-format flag.
	// This allows this command and its subcommands to have a different default
	// without affecting sibling commands.
	//
	// NOTE: This creates a Cobra quirk where the flag appears as "local" rather than
	// "inherited". We work around this by manually moving --output-format to the "Global Flags:" section
	// when displaying flags in the help text.
	cobraCmd.PersistentFlags().StringP(formatter.OutputFormatFlagName, "O", defaultFormat.String(),
		fmt.Sprintf("output format (%s)", strings.Join(formatter.AvailableOutputFormats(), "|")))

	// Bind the local flag to Viper. When Viper resolves a key with multiple bindings,
	// the last binding wins. This means the local (most specific) flag will take
	// precedence over the inherited flag from the root command.
	if err := viper.BindPFlag(formatter.OutputFormatFlagName,
		cobraCmd.PersistentFlags().Lookup(formatter.OutputFormatFlagName)); err != nil {
		return fmt.Errorf("failed to bind local output-format flag: %w", err)
	}

	return nil
}

type outputFormatError struct {
	provided  string
	supported []string
}

type invalidOutputFormatError struct {
	outputFormatError
}

func newInvalidOutputFormatError(provided string, supported []string) *invalidOutputFormatError {
	return &invalidOutputFormatError{
		outputFormatError: outputFormatError{
			provided:  provided,
			supported: supported,
		},
	}
}

var _ cenclierrors.CencliError = &invalidOutputFormatError{}

func (e *invalidOutputFormatError) Error() string {
	return fmt.Sprintf("invalid output format '%q'; supported formats: %s", e.provided, strings.Join(e.supported, ", "))
}

func (e *invalidOutputFormatError) Title() string {
	return "Invalid Output Format"
}

func (e *invalidOutputFormatError) ShouldPrintUsage() bool {
	return true
}

type unsupportedOutputFormatError struct {
	outputFormatError
}

var _ cenclierrors.CencliError = &unsupportedOutputFormatError{}

func newUnsupportedOutputFormatError(provided string, supported []string) *unsupportedOutputFormatError {
	return &unsupportedOutputFormatError{
		outputFormatError: outputFormatError{
			provided:  provided,
			supported: supported,
		},
	}
}

func (e *unsupportedOutputFormatError) Error() string {
	return fmt.Sprintf("output format '%s' is not supported by this command -- supported formats: %s", e.provided, strings.Join(e.supported, ", "))
}

func (e *unsupportedOutputFormatError) Title() string {
	return "Unsupported Output Format"
}

func (e *unsupportedOutputFormatError) ShouldPrintUsage() bool {
	return true
}

// validateStreamingMode checks for conflicts between streaming mode and output format flags.
// Returns an error if:
// - streaming is enabled (via config or flag) AND output format flag is explicitly set
// - streaming flag is explicitly set but command doesn't support streaming
func validateStreamingMode(cobraCmd *cobra.Command, cmd Command, streamingFromConfig bool) cenclierrors.CencliError {
	streamingFlag := cobraCmd.Flag(config.StreamingFlagName)
	outputFormatFlag := cobraCmd.Flag(formatter.OutputFormatFlagName)

	// Determine if streaming is enabled (flag takes precedence over config)
	streamingEnabled := streamingFromConfig
	streamingFlagExplicit := streamingFlag != nil && streamingFlag.Changed
	if streamingFlagExplicit {
		streamingEnabled = streamingFlag.Value.String() == "true"
	}

	// Check for conflict: streaming enabled + explicit output format
	outputFormatExplicit := outputFormatFlag != nil && outputFormatFlag.Changed
	if streamingEnabled && outputFormatExplicit {
		return newStreamingConflictError()
	}

	// Check if streaming flag was explicitly set on a command that doesn't support it
	if streamingFlagExplicit && streamingEnabled && !cmd.SupportsStreaming() {
		return newStreamingNotSupportedError()
	}

	return nil
}

type streamingConflictError struct{}

var _ cenclierrors.CencliError = &streamingConflictError{}

func newStreamingConflictError() *streamingConflictError {
	return &streamingConflictError{}
}

func (e *streamingConflictError) Error() string {
	return fmt.Sprintf("--%s and --%s cannot be used together; streaming mode uses NDJSON output", config.StreamingFlagName, formatter.OutputFormatFlagName)
}

func (e *streamingConflictError) Title() string {
	return "Conflicting Flags"
}

func (e *streamingConflictError) ShouldPrintUsage() bool {
	return true
}

type streamingNotSupportedError struct{}

var _ cenclierrors.CencliError = &streamingNotSupportedError{}

func newStreamingNotSupportedError() *streamingNotSupportedError {
	return &streamingNotSupportedError{}
}

func (e *streamingNotSupportedError) Error() string {
	return "this command does not support streaming output"
}

func (e *streamingNotSupportedError) Title() string {
	return "Streaming Not Supported"
}

func (e *streamingNotSupportedError) ShouldPrintUsage() bool {
	return true
}
