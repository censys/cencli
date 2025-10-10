package lib

import (
	"os"
	"path/filepath"
)

// FindBinary searches for the censys binary in common relative paths from the test directory.
// It returns the absolute path if found, or an empty string if not found.
func FindBinary() string {
	paths := []string{
		filepath.Join("bin", "censys"),
		filepath.Join("..", "bin", "censys"),
		filepath.Join("..", "..", "bin", "censys"),
		filepath.Join("..", "..", "..", "bin", "censys"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			absPath, _ := filepath.Abs(p)
			return absPath
		}
	}
	return ""
}
