package lib

import (
	"os"
	"path/filepath"
	"runtime"
)

// FindBinary searches for the censys binary in common relative paths from the test directory.
// It returns the absolute path if found, or an empty string if not found.
func FindBinary() string {
	binaryName := "censys"
	if runtime.GOOS == "windows" {
		binaryName = "censys.exe"
	}

	paths := []string{
		filepath.Join("bin", binaryName),
		filepath.Join("..", "bin", binaryName),
		filepath.Join("..", "..", "bin", binaryName),
		filepath.Join("..", "..", "..", "bin", binaryName),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			absPath, _ := filepath.Abs(p)
			return absPath
		}
	}
	return ""
}
