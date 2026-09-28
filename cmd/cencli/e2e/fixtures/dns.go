package fixtures

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
)

var dnsFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.DNSHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "invalid-port",
		Args:      []string{"censys.com:443"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Asset ID]", lines[0])
			assert.Contains(t, lines[1], "remove the port")
			rest := strings.Join(lines[2:], "\n")
			// dns's Long description is 13 lines (unlike view's 2), so this
			// strips the same number of lines from the golden file to land on
			// "Usage:" for comparison against the error's own Usage block.
			assertGoldenFile(t, golden.DNSHelpStdout, []byte(rest), 13)
		},
	},
	{
		Name:      "name-json",
		Args:      []string{"censys.com", "--output-format", "json"},
		ExitCode:  0,
		Timeout:   12 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			v := unmarshalJSONAny[[]struct {
				RecordType string `json:"record_type"`
			}](t, stdout)
			assert.NotEmpty(t, v)
		},
	},
	{
		Name:      "ip-short",
		Args:      []string{"8.8.8.8", "-p", "1"},
		ExitCode:  0,
		Timeout:   12 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			assert.Contains(t, string(stdout), "Domains resolving to 8.8.8.8")
		},
	},
}
