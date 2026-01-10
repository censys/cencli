package formatter

import (
	"encoding/json"
	"io"

	"github.com/censys/cencli/internal/pkg/styles"
	jsoncolor "github.com/neilotoole/jsoncolor"
)

// PrintJSON prints v as pretty JSON, optionally colored.
func PrintJSON(v any, colored bool) error {
	enc := newEncoderForWriter(Stdout, colored, true)
	return enc.Encode(v)
}

// jsonEncoder is a type that can encode JSON.
type jsonEncoder interface {
	Encode(v any) error
}

// jsonColors defines the color scheme for jsoncolor.
// This attempts to map the domain color scheme to what JQ uses.
func jsonColors() *jsoncolor.Colors {
	res := jsoncolor.DefaultColors()
	res.Null = styles.ANSIPrefix(styles.NewStyle(styles.ColorTeal))            // jq = gray
	res.Bool = styles.ANSIPrefix(styles.NewStyle(styles.ColorSage))            // jq = white
	res.Number = styles.ANSIPrefix(styles.NewStyle(styles.ColorSage))          // jq = white
	res.String = styles.ANSIPrefix(styles.NewStyle(styles.ColorOrange))        // jq = green
	res.Key = styles.ANSIPrefix(styles.NewStyle(styles.ColorAqua))             // jq = purple
	res.Bytes = styles.ANSIPrefix(styles.NewStyle(styles.ColorOrange))         // jq = green base64
	res.Time = styles.ANSIPrefix(styles.NewStyle(styles.ColorOrange))          // jq = green string
	res.Punc = styles.ANSIPrefix(styles.NewStyle(styles.ColorOffWhite))        // jq = white
	res.TextMarshaler = styles.ANSIPrefix(styles.NewStyle(styles.ColorOrange)) // jq = white
	return res
}

// WriteNDJSONItem encodes a single item as NDJSON (one line of JSON) to the provided writer.
// This enables true streaming output where each item is written immediately.
// The item is encoded without pretty-printing (compact JSON on a single line).
func WriteNDJSONItem(w io.Writer, item any, colored bool) error {
	enc := newEncoderForWriter(w, colored, false)
	return enc.Encode(item)
}

// newEncoderForWriter creates a JSON encoder for a specific writer.
// If colored is true, output will include ANSI color codes.
// If pretty is true, output will be indented.
func newEncoderForWriter(w io.Writer, colored, pretty bool) jsonEncoder {
	if colored {
		enc := jsoncolor.NewEncoder(w)
		enc.SetColors(jsonColors())
		if pretty {
			enc.SetIndent("", "  ")
		}
		return enc
	}
	enc := json.NewEncoder(w)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc
}
