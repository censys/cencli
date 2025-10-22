package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures"
	"github.com/censys/cencli/cmd/cencli/e2e/lib"
)

// The smoke test is a collection of very basic tests that help ensure
// the binary can be built and run on multiple platforms and architectures.

const (
	// If not set to true, the smoke test will be skipped.
	smokeTestEnableEnvVar = "CENCLI_ENABLE_SMOKE_TEST"
)

var (
	// expectedConfigFiles are the files that are expected to be
	// created in the config directory
	expectedConfigFiles = []string{
		"config.yaml",
		"cencli.db",
	}
	// expectedConfigDirs are the directories that are expected to be
	// created in the config directory
	expectedConfigDirs = []string{
		"templates",
	}
)

var (
	// smokeTestFixtures is the list of fixtures that will be run for the smoke test.
	smokeTestFixtures = map[string][]fixtures.Fixture{}

	// dataDir is the directory where files added by cencli will be stored
	dataDir string
)

func init() {
	// only run the root fixture for now
	smokeTestFixtures["root"] = fixtures.RootFixtures

	// get the cencli data directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("failed to get home directory: %w", err))
	}
	dataDir = filepath.Join(homeDir, ".config", "cencli")
}

func TestSmoke(t *testing.T) {
	if os.Getenv(smokeTestEnableEnvVar) != "true" {
		t.Skip("Smoke test is disabled. Set " + smokeTestEnableEnvVar + " to 'true' to run these tests")
		return
	}

	binaryPath := lib.FindBinary()
	require.NotEmpty(t, binaryPath, "Binary not built; run 'make censys' first")

	for command, fixtures := range smokeTestFixtures {
		t.Run(command, func(t *testing.T) {
			for _, fixture := range fixtures {
				t.Run(fixture.Name, func(t *testing.T) {
					// ensure the data directory is empty
					ensureDataDirEmpty(t)
					// run the fixture test with an empty data directory
					// so that the default data directory is used and the expected files and directories are created
					runFixtureTest(t, binaryPath, "", command, fixture)
					// ensure the data directory contains the expected files and directories
					ensureDataDirContents(t)
					// delete the data directory
					require.NoError(t, os.RemoveAll(dataDir))
				})
			}
		})
	}
}

// ensureDataDirEmpty asserts that the data directory is empty.
// This does not assert failure if the directory does not exist.
func ensureDataDirEmpty(t *testing.T) {
	info, err := os.Stat(dataDir)
	if err != nil {
		assert.True(t, os.IsNotExist(err), "failed to check if data directory exists: %s", err)
		return
	}
	require.True(t, info.IsDir(), "data directory is not a directory: %s", dataDir)
	f, err := os.ReadDir(dataDir)
	require.NoError(t, err, "failed to read data directory: %s", dataDir)
	require.Empty(t, f, "data directory is not empty: %s", dataDir)
}

// ensureDataDirContents asserts that the data directory contains
// the expected files and directories.
func ensureDataDirContents(t *testing.T) {
	// ensure the expected files exist
	for _, file := range expectedConfigFiles {
		filePath := filepath.Join(dataDir, file)
		require.FileExists(t, filePath)
		content, err := os.ReadFile(filePath)
		require.NoError(t, err, "failed to read file: %s", filePath)
		require.NotEmpty(t, content, "file is empty: %s", filePath)
	}
	// ensure the expected directories exist
	for _, dir := range expectedConfigDirs {
		dirPath := filepath.Join(dataDir, dir)
		require.DirExists(t, dirPath)
		entries, err := os.ReadDir(dirPath)
		require.NoError(t, err, "failed to read directory: %s", dirPath)
		require.NotEmpty(t, entries, "directory is empty: %s", dirPath)
	}
}
