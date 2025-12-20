package formatter

import (
	"fmt"
	"io"
	"os"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
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
//
// Note: NDJSON is not supported here - it requires streaming via WithStreamingOutput.
// Commands that support NDJSON must use OutputTypeStreaming and skip PrintData when streaming.
func PrintByFormat(data any, format OutputFormat, colored bool) cenclierrors.CencliError {
	switch format {
	case OutputFormatJSON:
		return cenclierrors.NewCencliError(PrintJSON(data, colored))
	case OutputFormatNDJSON:
		// NDJSON requires streaming - this should never be reached if commands are implemented correctly
		return cenclierrors.NewCencliError(fmt.Errorf("ndjson format requires streaming output"))
	case OutputFormatYAML:
		return cenclierrors.NewCencliError(PrintYAML(data, colored))
	case OutputFormatTree:
		return cenclierrors.NewCencliError(PrintTree(data, colored))
	case OutputFormatShort, OutputFormatTemplate:
		// these will be handled by the command
		return cenclierrors.NewCencliError(fmt.Errorf("output format %s not supported", format))
	default:
		return cenclierrors.NewCencliError(PrintJSON(data, colored))
	}
}
