package formatter

import (
	"encoding/json"
	"io"

	"github.com/censys/cencli/internal/pkg/styles"
	jsoncolor "github.com/neilotoole/jsoncolor"
)

// PrintJSON prints v as pretty JSON, optionally colored.
// Uses the standard library for marshaling (to support omitzero),
// then colorizes the output if requested.
func PrintJSON(v any, colored bool) error {
	return writeJSON(Stdout, v, colored, true)
}

// writeJSON writes v as JSON to w, optionally colored and pretty-printed.
// Uses the standard library for marshaling (to support omitzero),
// then colorizes the output if requested.
func writeJSON(w io.Writer, v any, colored, pretty bool) error {
	var data []byte
	var err error
	if pretty {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}

	if colored {
		// Re-encode the raw JSON with colors
		enc := jsoncolor.NewEncoder(w)
		enc.SetColors(jsonColors())
		if pretty {
			enc.SetIndent("", "  ")
		}
		return enc.Encode(json.RawMessage(data))
	}

	_, err = w.Write(append(data, '\n'))
	return err
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
// Uses the standard library for marshaling (to support omitzero),
// then colorizes the output if requested.
func WriteNDJSONItem(w io.Writer, item any, colored bool) error {
	return writeJSON(w, item, colored, false)
}
