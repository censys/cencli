package log

import (
	"io"
	"log/slog"
	"os"
)

// New returns a slog.Logger configured for either debug or info level.
// Output defaults to stderr if out is nil.
func New(debug bool, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stderr
	}
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	handler := slog.NewJSONHandler(out, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
