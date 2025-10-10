package lib

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

// CommandResult captures the output and exit status of a command execution.
type CommandResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Error    error
}

// RunCommand executes a command and captures its stdout, stderr, exit code, and any errors.
func RunCommand(cmd *exec.Cmd) CommandResult {
	result := CommandResult{}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Get the exit code from the ExitError
			result.ExitCode = exitErr.ExitCode()
			// Exit code -1 can occur in debug mode or with signals, but it's still a valid scenario
			// We only set an error if this was truly unexpected (non-ExitError cases)
		} else {
			// Non-ExitError means something went wrong with command execution itself
			result.Error = fmt.Errorf("failed to run command: %w", err)
		}
	}
	// If err == nil, ExitCode remains 0 and Error remains nil (success case)

	return result
}
