package fixtures

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
)

var aggregateFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.AggregateHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "basic",
		Args:      []string{"host.services.protocol=SSH", "host.services.port", "-n", "5"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			assert.Contains(t, string(stdout), "host.services.protocol=SSH")
			assert.Contains(t, string(stdout), "host.services.port")
			assert.Contains(t, string(stdout), "22")
		},
	},
	{
		Name:      "basic-raw",
		Args:      []string{"host.services.protocol=SSH", "host.services.port", "-n", "5", "--raw"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			v := unmarshalJSONAny[[]struct {
				Key   string `json:"key"`
				Count int    `json:"count"`
			}](t, stdout)
			assert.Len(t, v, 5)
		},
	},
}
