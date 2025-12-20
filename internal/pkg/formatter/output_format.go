package formatter

import (
	"encoding"
	"errors"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	OutputFormatFlagName  = "output-format"
	outputFormatFlagShort = "O"
)

type OutputFormat string

const (
	OutputFormatJSON     OutputFormat = "json"
	OutputFormatYAML     OutputFormat = "yaml"
	OutputFormatNDJSON   OutputFormat = "ndjson"
	OutputFormatTree     OutputFormat = "tree"
	OutputFormatShort    OutputFormat = "short"
	OutputFormatTemplate OutputFormat = "template"
)

// ErrInvalidOutputFormat is returned when the provided output format is unsupported.
var ErrInvalidOutputFormat = errors.New("invalid output format")

func (o OutputFormat) String() string {
	return string(o)
}

var _ encoding.TextUnmarshaler = (*OutputFormat)(nil)

func (o *OutputFormat) UnmarshalText(text []byte) error {
	s := string(text)
	switch s {
	case OutputFormatJSON.String():
		*o = OutputFormatJSON
	case OutputFormatYAML.String():
		*o = OutputFormatYAML
	case OutputFormatNDJSON.String():
		*o = OutputFormatNDJSON
	case OutputFormatTree.String():
		*o = OutputFormatTree
	case OutputFormatShort.String():
		*o = OutputFormatShort
	case OutputFormatTemplate.String():
		*o = OutputFormatTemplate
	default:
		return fmt.Errorf("%w: %s", ErrInvalidOutputFormat, s)
	}
	return nil
}

func AvailableOutputFormats() []string {
	return []string{
		OutputFormatJSON.String(),
		OutputFormatYAML.String(),
		OutputFormatNDJSON.String(),
		OutputFormatTree.String(),
		OutputFormatShort.String(),
		OutputFormatTemplate.String(),
	}
}

func BindOutputFormat(persistentFlags *pflag.FlagSet, defaultValue OutputFormat) error {
	// Bind the global --output-format flag (no short form)
	persistentFlags.StringP(OutputFormatFlagName, outputFormatFlagShort, defaultValue.String(), "output format (json|yaml|ndjson|tree|short|template)")
	return viper.BindPFlag(OutputFormatFlagName, persistentFlags.Lookup(OutputFormatFlagName))
}
