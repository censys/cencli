// Package browser provides utilities for opening URLs in the system's default browser.
package browser

import (
	"errors"
	"os/exec"
	"runtime"
)

// ErrUnsupportedOS is returned when the current operating system is not supported for browser operations.
var ErrUnsupportedOS = errors.New("unsupported operating system")

// Open opens the given URL in the system's default browser.
// It returns an error if the browser could not be launched or if the operating system is unsupported.
//
// Supported operating systems:
//   - darwin (macOS) - uses "open" command
//   - linux - uses "xdg-open" command
//   - windows - uses "cmd /c start" command
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return ErrUnsupportedOS
	}
	return cmd.Start()
}
