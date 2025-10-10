package formatter

import (
	"encoding/json"

	"github.com/censys/cencli/internal/pkg/styles"
	jsoncolor "github.com/neilotoole/jsoncolor"
)

// PrintJSON prints v as pretty JSON, optionally colored.
func PrintJSON(v any, colored bool) error {
	enc := newEncoder(colored, true)
	return enc.Encode(v)
}

// PrintNDJSON prints one JSON object per line.
// If v is a slice, it prints each element on its own line.
// Otherwise, it prints v.
func PrintNDJSON(v any, colored bool) error {
	enc := newEncoder(colored, false)

	switch s := v.(type) {
	case []any:
		for _, item := range s {
			if err := enc.Encode(item); err != nil {
				return err
			}
		}
		return nil
	default:
		return enc.Encode(v)
	}
}

// encoder is a type that can encode JSON.
type jsonEncoder interface {
	Encode(v any) error
}

// newEncoder creates either a plain or color encoder.
// pretty controls whether SetIndent is applied.
func newEncoder(colored, pretty bool) jsonEncoder {
	if colored {
		enc := jsoncolor.NewEncoder(Stdout)
		enc.SetColors(jsonColors())
		if pretty {
			enc.SetIndent("", "  ")
		}
		return enc
	}
	enc := json.NewEncoder(Stdout)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc
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
