package lib

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	AuthEnvVar  = "CENSYS_API_TOKEN"
	OrgIDEnvVar = "CENSYS_ORG_ID"
)

// E2EEnvVars returns the environment variables that are used for all e2e tests.
// It configures a custom data directory and disables color/spinner output for deterministic testing.
// If the data dir is empty, which will be the case for the smoke test,
// then that environment variable is not set.
func E2EEnvVars(dataDir string) []string {
	res := []string{
		"CENCLI_NO_COLOR=1",
		"CENCLI_NO_SPINNER=1",
	}
	if dataDir != "" {
		res = append(res, "CENCLI_DATA_DIR="+dataDir)
	}
	return res
}

// ConfigureAuth sets up authentication credentials for e2e tests by running config commands.
// It requires CENSYS_API_TOKEN and CENSYS_ORG_ID environment variables to be set.
// It also requires the binary path to be passed in.
func ConfigureAuth(dataDir string, binaryPath string) error {
	// ensure the auth environment variables are set
	pat := os.Getenv(AuthEnvVar)
	if pat == "" {
		return fmt.Errorf("%s is not set", AuthEnvVar)
	}
	orgID := os.Getenv(OrgIDEnvVar)
	if orgID == "" {
		return fmt.Errorf("%s is not set", OrgIDEnvVar)
	}
	env := E2EEnvVars(dataDir)
	// add auth
	cmd := exec.Command(binaryPath, "config", "auth", "add", "--value", pat, "--name", "test")
	cmd.Env = append(os.Environ(), env...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add auth: %w", err)
	}
	// add org id
	cmd = exec.Command(binaryPath, "config", "org-id", "add", "--value", orgID, "--name", "test")
	cmd.Env = append(os.Environ(), env...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add org id: %w", err)
	}
	return nil
}
