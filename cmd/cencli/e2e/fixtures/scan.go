package fixtures

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
)

// Every fixture here fails or finishes in PreRun, before any credential is
// needed, so none of them can spend the 10 credits a rescan costs.
var scanFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.ScanHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "help with no args",
		Args:      []string{},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.ScanHelpStdout, stdout, 0)
		},
	},
	// ========== rescan subcommand ==========
	{
		Name:      "rescan help",
		Args:      []string{"rescan", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.ScanRescanHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "rescan missing arg",
		Args:      []string{"rescan"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "accepts 1 arg")
		},
	},
	{
		Name:      "rescan rejects a host",
		Args:      []string{"rescan", "8.8.8.8"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "supports web properties (hostname:port) only")
			assert.Contains(t, string(stderr), "is a host")
		},
	},
	{
		Name:      "rescan rejects --timeout without --wait",
		Args:      []string{"rescan", "example.com:443", "--timeout", "5m"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "--timeout only applies while polling")
		},
	},
	// ========== get subcommand ==========
	{
		Name:      "get help",
		Args:      []string{"get", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.ScanGetHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "get rejects a non-UUID scan ID",
		Args:      []string{"get", "not-a-uuid"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), `scan ID "not-a-uuid" is not a valid UUID`)
		},
	},
}
