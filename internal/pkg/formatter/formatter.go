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

func Printf(format string, a ...any) {
	fmt.Fprintf(Stdout, format, a...)
}

func Print(a ...any) {
	fmt.Fprint(Stdout, a...)
}

func Println(a ...any) {
	fmt.Fprintln(Stdout, a...)
}

func Printlnf(format string, a ...any) {
	fmt.Fprintf(Stdout, format, a...)
	fmt.Fprintln(Stdout)
}

// PrintByFormat prints data according to the provided output format.
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
