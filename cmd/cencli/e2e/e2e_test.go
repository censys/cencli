package e2e

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures"
	"github.com/censys/cencli/cmd/cencli/e2e/lib"
	"github.com/stretchr/testify/require"
)

const (
	// enableE2ETestsEnvVar will actually run the tests if true
	enableE2ETestsEnvVar = "CENCLI_ENABLE_E2E_TESTS"
)

func TestE2E(t *testing.T) {
	if os.Getenv(enableE2ETestsEnvVar) != "true" {
		t.Skip("E2E tests are disabled. Set " + enableE2ETestsEnvVar + " to 'true' to run tests")
		return
	}

	binaryPath := lib.FindBinary()
	require.NotEmpty(t, binaryPath, "Binary not built; run 'make censys' first")

	for command, fixtures := range fixtures.Fixtures() {
		t.Run(command, func(t *testing.T) {
			for _, fixture := range fixtures {
				t.Run(fixture.Name, func(t *testing.T) {
					dataDir := t.TempDir()
					runFixtureTest(t, binaryPath, dataDir, command, fixture)
				})
			}
		})
	}
}

func runFixtureTest(
	t *testing.T,
	binaryPath string,
	dataDir string,
	command string,
	fixture fixtures.Fixture,
) {
	if fixture.NeedsAuth {
		err := lib.ConfigureAuth(dataDir, binaryPath)
		require.NoError(t, err, "Failed to configure auth for fixture %s", fixture.Name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), fixture.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		binaryPath,
		append([]string{command}, fixture.Args...)...,
	)
	cmd.Env = append(os.Environ(), lib.E2EEnvVars(dataDir)...)

	result := lib.RunCommand(cmd)
	require.NoError(t, result.Error, "failed to run command for fixture %s", fixture.Name)

	require.Equal(t, fixture.ExitCode, result.ExitCode, "unexpected exit code, stderr: %s", string(result.Stderr))
	fixture.Assert(t, result.Stdout, result.Stderr)
}
