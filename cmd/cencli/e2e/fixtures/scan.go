package fixtures

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
)

const (
	// scanIDEnvVar names an existing tracked scan for the free `scan get` fixture.
	scanIDEnvVar = "CENCLI_E2E_SCAN_ID"
	// enableRescanEnvVar must be "true" to run the live rescan, which costs 10
	// credits per run.
	enableRescanEnvVar = "CENCLI_E2E_ENABLE_RESCAN"
)

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
	// ========== live ==========
	{
		Name:      "get live scan",
		Args:      []string{"get", os.Getenv(scanIDEnvVar), "--output-format", "json"},
		ExitCode:  0,
		Timeout:   10 * time.Second,
		NeedsAuth: true,
		Skip: func() string {
			if os.Getenv(scanIDEnvVar) == "" {
				return "set " + scanIDEnvVar + " to a tracked scan ID to run"
			}
			return ""
		},
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			v := unmarshalJSONAny[trackedScan](t, stdout)
			assert.Equal(t, os.Getenv(scanIDEnvVar), v.ID)
		},
	},
	{
		Name:      "rescan live - costs 10 credits",
		Args:      []string{"rescan", "platform.censys.io:80", "--output-format", "json"},
		ExitCode:  0,
		Timeout:   30 * time.Second,
		NeedsAuth: true,
		Skip: func() string {
			if os.Getenv(enableRescanEnvVar) != "true" {
				return "costs 10 credits; set " + enableRescanEnvVar + "=true to run"
			}
			return ""
		},
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			v := unmarshalJSONAny[trackedScan](t, stdout)
			_, err := uuid.Parse(v.ID)
			assert.NoError(t, err, "tracked_scan_id should be a UUID")
			assert.Equal(t, "platform.censys.io", v.Target.WebOrigin.Hostname)
			assert.Equal(t, 80, v.Target.WebOrigin.Port)
		},
	},
}

type trackedScan struct {
	ID     string `json:"tracked_scan_id"`
	Target struct {
		WebOrigin struct {
			Hostname string `json:"hostname"`
			Port     int    `json:"port"`
		} `json:"web_origin"`
	} `json:"target"`
}
