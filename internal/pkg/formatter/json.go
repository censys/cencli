package formatter

import (
	"encoding/json"

	"github.com/censys/cencli/internal/pkg/styles"
	jsoncolor "github.com/neilotoole/jsoncolor"
)

// PrintJSON prints v as pretty JSON, optionally colored.
// Uses the standard library for marshaling (to support omitzero),
// then colorizes the output if requested.
func PrintJSON(v any, colored bool) error {
	// Marshal with standard library first to support omitzero
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	if colored {
		// Re-encode the raw JSON with colors
		enc := jsoncolor.NewEncoder(Stdout)
		enc.SetColors(jsonColors())
		enc.SetIndent("", "  ")
		return enc.Encode(json.RawMessage(data))
	}

	_, err = Stdout.Write(append(data, '\n'))
	return err
}

// PrintNDJSON prints one JSON object per line.
// If v is a slice, it prints each element on its own line.
// Otherwise, it prints v.
// Uses the standard library for marshaling (to support omitzero).
func PrintNDJSON(v any, colored bool) error {
	encode := func(item any) error {
		// Marshal with standard library first to support omitzero
		data, err := json.Marshal(item)
		if err != nil {
			return err
		}

		if colored {
			enc := jsoncolor.NewEncoder(Stdout)
			enc.SetColors(jsonColors())
			return enc.Encode(json.RawMessage(data))
		}

		_, err = Stdout.Write(append(data, '\n'))
		return err
	}

	switch s := v.(type) {
	case []any:
		for _, item := range s {
			if err := encode(item); err != nil {
				return err
			}
		}
		return nil
	default:
		return encode(v)
	}
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
