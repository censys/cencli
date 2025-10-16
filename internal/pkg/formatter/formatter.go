package formatter

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/censys/cencli/internal/pkg/term"
)

var (
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr
)

func StdoutIsTTY() bool {
	return term.IsTTY(Stdout)
}

func StderrIsTTY() bool {
	return term.IsTTY(Stderr)
}

func Printf(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, format, a...)
}

func Println(w io.Writer, a ...any) {
	fmt.Fprintln(w, a...)
}

// PrintByFormat prints data to stdout according to the provided output format.
// Falls back to JSON when format is unrecognized.
func PrintByFormat(data any, format OutputFormat, colored bool) error {
	switch format.String() {
	case OutputFormatJSON.String():
		return PrintJSON(data, colored)
	case OutputFormatNDJSON.String():
		return PrintNDJSON(asAnySlice(data), colored)
	case OutputFormatYAML.String():
		return PrintYAML(data, colored)
	case OutputFormatTree.String():
		return PrintTree(data, colored)
	default:
		return PrintJSON(data, colored)
	}
}

func asAnySlice(v any) []any {
	// If it's already []any, return as-is
	if s, ok := v.([]any); ok {
		return s
	}
	// For any slice type, reflect to []any
	rv := reflect.ValueOf(v)
	if rv.IsValid() && rv.Kind() == reflect.Slice {
		n := rv.Len()
		out := make([]any, n)
		for i := 0; i < n; i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out
	}
	// Fallback: single element slice
	return []any{v}
}
