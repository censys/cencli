package tape

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// defaultTheme is the VHS theme used for all tape recordings.
	defaultTheme = "iTerm2 Pastel Dark Background"
)

// Recorder manages VHS tape recordings and GIF generation.
type Recorder struct {
	vhsPath string
	cliPath string
}

// NewTapeRecorder creates a new Recorder for the given VHS and CLI binaries.
func NewTapeRecorder(
	vhsPath string,
	cliPath string,
	env map[string]string,
) (*Recorder, error) {
	err := ensureBinary(vhsPath)
	if err != nil {
		return nil, fmt.Errorf("%s not found in PATH: %w", vhsPath, err)
	}

	// If cliPath is an absolute path, temporarily add its directory to PATH
	// so VHS can execute it by name only
	cliName := cliPath
	if filepath.IsAbs(cliPath) {
		// Check if the file exists
		if _, err := os.Stat(cliPath); err != nil {
			return nil, fmt.Errorf("CLI binary not found at %s: %w", cliPath, err)
		}
		// Add the directory to PATH
		binDir := filepath.Dir(cliPath)
		currentPath := os.Getenv("PATH")
		newPath := binDir + string(filepath.ListSeparator) + currentPath
		if err := os.Setenv("PATH", newPath); err != nil {
			return nil, fmt.Errorf("failed to update PATH: %w", err)
		}
		// Use just the basename for the command
		cliName = filepath.Base(cliPath)
	} else {
		// For relative paths or names, check if it's in PATH
		err = ensureBinary(cliPath)
		if err != nil {
			return nil, fmt.Errorf("%s not found in PATH: %w", cliPath, err)
		}
	}

	for key, value := range env {
		err = os.Setenv(key, value)
		if err != nil {
			return nil, fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}
	e := &Recorder{
		vhsPath: vhsPath,
		cliPath: cliName,
	}
	return e, nil
}

// prependSetup adds VHS configuration commands to the beginning of a tape.
func (e *Recorder) prependSetup(tape *Tape) {
	tape.commands = strings.Join([]string{
		fmt.Sprintf(`Set Theme "%s"`, defaultTheme),
		fmt.Sprintf("Set FontSize %d", tape.config.FontSize),
		fmt.Sprintf("Set Width %d", tape.config.Width),
		fmt.Sprintf("Set Height %d", tape.config.Height),
		tape.commands,
	}, "\n")
}

// CreateTape generates a GIF from a Tape and saves it to outputDir.
func (e *Recorder) CreateTape(
	ctx context.Context,
	tape Tape,
	outputDir string,
) error {
	err := ensureDirectory(outputDir)
	if err != nil {
		return fmt.Errorf("failed to ensure output directory %s: %w", outputDir, err)
	}
	// add tape setup
	e.prependSetup(&tape)
	// make a temporary directory to store the tape
	tempDir, err := os.MkdirTemp(outputDir, "tape")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	tapePath := filepath.Join(tempDir, fmt.Sprintf("%s.tape", tape.Name))
	// write the tape to the temporary directory
	if err := os.WriteFile(tapePath, []byte(tape.commands), 0o644); err != nil {
		return fmt.Errorf("failed to write tape: %w", err)
	}
	// create the gif
	gifPath := filepath.Join(outputDir, fmt.Sprintf("%s.gif", tape.Name))
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(e.vhsPath, tapePath, "--output", gifPath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run VHS: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	return nil
}

type typeOption func(*typeOptions)

// WithSleepAfter adds a sleep delay after executing a command.
func WithSleepAfter(seconds int) typeOption {
	return func(o *typeOptions) {
		o.sleepAfter = seconds
	}
}

// WithClearAfter clears the terminal after executing a command.
func WithClearAfter() typeOption {
	return func(o *typeOptions) {
		o.clearAfter = true
	}
}

// Type generates VHS commands to type and execute a CLI command.
func (e *Recorder) Type(cmd string, options ...typeOption) string {
	o := &typeOptions{
		sleepAfter: 0,
		clearAfter: false,
	}
	for _, option := range options {
		option(o)
	}
	commands := []string{
		fmt.Sprintf("Type `%s %s`", e.cliPath, cmd),
		"Enter",
	}
	if o.sleepAfter > 0 {
		commands = append(commands, fmt.Sprintf("Sleep %ds", o.sleepAfter))
	}
	if o.clearAfter {
		commands = append(commands, "Type clear", "Enter")
	}
	return strings.Join(commands, "\n")
}

// Press generates VHS commands to press a key a specified number of times.
// Will sleep 50 milliseconds between each press.
func (e *Recorder) Press(key string, numTimes int) string {
	commands := make([]string, 0, numTimes)
	for range numTimes {
		commands = append(commands, fmt.Sprintf("Type %s\nSleep 50ms", key))
	}
	return strings.Join(commands, "\n")
}

// SpamPress generates VHS commands to press a key a specified number of times.
// Will sleep 5 milliseconds between each press.
func (e *Recorder) SpamPress(key string, numTimes int) string {
	commands := make([]string, 0, numTimes)
	for range numTimes {
		commands = append(commands, fmt.Sprintf("Type %s\nSleep 5ms", key))
	}
	return strings.Join(commands, "\n")
}

// Sleep generates VHS commands to sleep for a specified number of seconds.
func (e *Recorder) Sleep(seconds int) string {
	return fmt.Sprintf("Sleep %ds", seconds)
}

// TypeClear generates VHS commands to type and clear the terminal.
func (e *Recorder) Clear() string {
	return "Type clear\nEnter"
}
