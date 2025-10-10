package tape

import (
	"fmt"
	"os"
	"os/exec"
)

// ensureBinary checks if a binary exists in PATH.
func ensureBinary(binaryName string) error {
	_, err := exec.LookPath(binaryName)
	if err != nil {
		return fmt.Errorf("%s not found in PATH: %w", binaryName, err)
	}
	return nil
}

// ensureDirectory creates a directory if it doesn't exist.
func ensureDirectory(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		// if not exists, create it
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0o755)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	// make sure it's a directory
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}
