package fixtures

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
)

var historyFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.HistoryHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "webproperty-basic",
		Args:      []string{"platform.censys.io:80", "--duration", "2d"},
		ExitCode:  0,
		Timeout:   12 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			v := unmarshalJSONAny[[]struct {
				Time   time.Time `json:"time"`
				Data   any       `json:"data"`
				Exists bool      `json:"exists"`
			}](t, stdout)
			assert.Greater(t, len(v), 1)
		},
	},
	// Output format tests
	{
		Name:      "output-json-default",
		Args:      []string{"platform.censys.io:80", "--duration", "2d"},
		ExitCode:  0,
		Timeout:   12 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			// Verify default JSON output
			v := unmarshalJSONAny[[]struct {
				Time   time.Time `json:"time"`
				Data   any       `json:"data"`
				Exists bool      `json:"exists"`
			}](t, stdout)
			assert.Greater(t, len(v), 1)
		},
	},
	{
		Name:      "output-short-unsupported",
		Args:      []string{"platform.censys.io:80", "--duration", "2d", "--output-format", "short"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			// Should fail with error about unsupported output format
			assert.Contains(t, string(stderr), "short")
			assert.Contains(t, string(stderr), "not supported")
		},
	},
	{
		Name:      "output-template-unsupported",
		Args:      []string{"platform.censys.io:80", "--duration", "2d", "--output-format", "template"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			// Should fail with error about unsupported output format
			assert.Contains(t, string(stderr), "template")
			assert.Contains(t, string(stderr), "not supported")
		},
	},
	// TODO: certificate and host history
}
