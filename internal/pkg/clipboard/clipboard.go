package clipboard

import (
	"errors"
	"io"
	"os/exec"
	"runtime"
)

// Copy copies the given text string to the system clipboard.
// It works on macOS, Linux (xclip required), and Windows.
func Copy(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	case "windows":
		cmd = exec.Command("cmd", "/c", "clip")
	default:
		return errors.New("unsupported OS: " + runtime.GOOS)
	}

	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if _, err := io.WriteString(in, text); err != nil {
		closeErr := in.Close()
		if closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return err
	}

	// Close stdin to signal EOF to the command before waiting
	if err := in.Close(); err != nil {
		return err
	}

	return cmd.Wait()
}
