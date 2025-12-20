package fixtures

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
)

var censeyeFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.CenseyeHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "basic",
		Args:      []string{"145.131.8.169", "--rarity-min", "2", "--rarity-max", "125"},
		ExitCode:  0,
		Timeout:   12 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			assert.Contains(t, string(stdout), "=== CensEye Results for 145.131.8.169 ===")
			assert.Contains(t, string(stdout), "Count")
			assert.Contains(t, string(stdout), "Query")
			assert.Contains(t, string(stdout), "within [2,125]")
		},
	},
	{
		Name:      "output-short-default",
		Args:      []string{"145.131.8.169", "--rarity-min", "2", "--rarity-max", "125"},
		ExitCode:  0,
		Timeout:   12 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			// Default output is short format (table), just verify it has content
			assert.Greater(t, len(stdout), 0)
			assert.Contains(t, string(stdout), "=== CensEye Results for 145.131.8.169 ===")
		},
	},
	{
		Name:      "output-json",
		Args:      []string{"145.131.8.169", "--output-format", "json"},
		ExitCode:  0,
		Timeout:   12 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			v := unmarshalJSONAny[[]struct {
				Count       int    `json:"count"`
				Query       string `json:"query"`
				Interesting bool   `json:"interesting"`
			}](t, stdout)
			assert.Greater(t, len(v), 1)
		},
	},
	{
		Name:      "output-template-unsupported",
		Args:      []string{"145.131.8.169", "--output-format", "template"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			// Should fail with error about unsupported output format
			assert.Contains(t, string(stderr), "template")
			assert.Contains(t, string(stderr), "not supported")
		},
	},
}
