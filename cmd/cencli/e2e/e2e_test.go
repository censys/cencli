package e2e

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures"
	"github.com/censys/cencli/cmd/cencli/e2e/lib"
	"github.com/stretchr/testify/require"
)

const (
	// If not set to true, the E2E tests will be skipped.
	enableE2ETestsEnvVar = "CENCLI_ENABLE_E2E_TESTS"
)

func TestE2E(t *testing.T) {
	if os.Getenv(enableE2ETestsEnvVar) != "true" {
		t.Skip("E2E tests are disabled. Set " + enableE2ETestsEnvVar + " to 'true' to run these tests")
		return
	}

	binaryPath := lib.FindBinary()
	require.NotEmpty(t, binaryPath, "Binary not built; run 'make censys' first")

	for command, fixtures := range fixtures.Fixtures() {
		t.Run(command, func(t *testing.T) {
			for _, fixture := range fixtures {
				t.Run(fixture.Name, func(t *testing.T) {
					dataDir := t.TempDir()
					runFixtureTest(t, binaryPath, dataDir, true, command, fixture)
				})
			}
		})
	}
}

func runFixtureTest(
	t *testing.T,
	binaryPath string,
	dataDir string,
	dataDirEnvOverride bool,
	command string,
	fixture fixtures.Fixture,
) {
	// Run the setup command if it is present
	if fixture.Setup.IsPresent() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		cmd := exec.CommandContext(ctx, binaryPath)
		cmd.Env = append(os.Environ(), lib.E2EEnvVars(dataDir)...)
		result := lib.RunCommand(cmd)
		require.NoError(t, result.Error, "failed to run setup command for fixture %s", fixture.Name)
		require.Equal(t, 0, result.ExitCode, "unexpected exit code, stderr: %s", string(result.Stderr))
		fixture.Setup.MustGet()(t, dataDir)
	}

	if fixture.NeedsAuth {
		err := lib.ConfigureAuth(dataDir, binaryPath)
		require.NoError(t, err, "Failed to configure auth for fixture %s", fixture.Name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), fixture.Timeout)
	defer cancel()

	// Special case: "root" means no subcommand, just run the binary with args
	var cmdArgs []string
	if command == "root" {
		cmdArgs = fixture.Args
	} else {
		cmdArgs = append([]string{command}, fixture.Args...)
	}

	cmd := exec.CommandContext(ctx, binaryPath, cmdArgs...)
	var env []string
	if dataDirEnvOverride {
		env = lib.E2EEnvVars(dataDir)
	} else {
		env = lib.E2EEnvVars("")
	}
	cmd.Env = append(os.Environ(), env...)

	result := lib.RunCommand(cmd)
	require.NoError(t, result.Error, "failed to run command for fixture %s", fixture.Name)

	require.Equal(t, fixture.ExitCode, result.ExitCode, "unexpected exit code, stderr: %s", string(result.Stderr))
	fixture.Assert(t, dataDir, result.Stdout, result.Stderr)
}
