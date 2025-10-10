package term

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	// custom environment variable
	noTTYEnvVar = "NO_TTY"
	// https://man7.org/linux/man-pages/man7/term.7.html
	termEnvVar = "TERM"
	// https://invisible-island.net/ncurses/terminfo.src.html
	dumbTerm = "dumb"
	// Default terminal width when it cannot be determined
	defaultWidth = 80
)

// IsTTY returns true if the given writer is a terminal.
func IsTTY(w io.Writer) bool {
	if os.Getenv(noTTYEnvVar) == "1" {
		return false
	}
	if strings.ToLower(os.Getenv(termEnvVar)) == dumbTerm {
		return false
	}
	if f, ok := w.(interface{ Fd() uintptr }); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// GetWidth returns the width of the terminal, or defaultWidth if it cannot be determined.
func GetWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil && width > 0 {
		return width
	}
	return defaultWidth
}

// RenderLink creates a clickable terminal link (OSC 8 escape sequence).
// Not all terminals support this feature.
func RenderLink(url, text string) string {
	// Return OSC 8 hyperlink - visible width is just the text
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, text)
}
