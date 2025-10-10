package formatter

import (
	"encoding"
	"errors"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	outputFormatKey   = "output-format"
	outputFormatShort = "O"
)

type OutputFormat string

const (
	OutputFormatJSON   OutputFormat = "json"
	OutputFormatYAML   OutputFormat = "yaml"
	OutputFormatNDJSON OutputFormat = "ndjson"
	OutputFormatTree   OutputFormat = "tree"
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
	default:
		return fmt.Errorf("%w: %s", ErrInvalidOutputFormat, s)
	}
	return nil
}

func BindOutputFormat(persistentFlags *pflag.FlagSet) error {
	// Bind the global --output-format flag (no short form)
	persistentFlags.StringP(outputFormatKey, outputFormatShort, OutputFormatJSON.String(), "output format (json|yaml|ndjson|tree)")
	return viper.BindPFlag(outputFormatKey, persistentFlags.Lookup(outputFormatKey))
}
