package fixtures

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
)

// testOrgID is a syntactically-valid org ID used by the no-auth validation
// fixtures so they exercise input validation rather than the missing-org path.
const testOrgID = "550e8400-e29b-41d4-a716-446655440001"

var enrichFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.EnrichHelpStdout, stdout, 0)
		},
	},
	// ================================================
	// Validation fixtures (no auth, deterministic)
	// ================================================
	{
		Name:      "invalid-ip",
		Args:      []string{"--org-id", testOrgID, "not-an-ip"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			require.NotEmpty(t, lines)
			assert.Equal(t, "[Invalid Host]", lines[0])
			assert.Contains(t, string(stderr), "valid host IP")
		},
	},
	{
		Name:      "no-hosts",
		Args:      []string{"--org-id", testOrgID},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			require.NotEmpty(t, lines)
			assert.Equal(t, "[No Hosts Provided]", lines[0])
		},
	},
	{
		// Enrichment requires an org; with no --org-id and no configured org the
		// command must fail before making any request.
		Name:      "missing-org",
		Args:      []string{"8.8.8.8"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			require.NotEmpty(t, lines)
			assert.Equal(t, "[No Organization ID]", lines[0])
		},
	},
	// ================================================
	// Live API fixtures (require auth + enrichment entitlement)
	// ================================================
	{
		Name:      "host-json-default",
		Args:      []string{"1.1.1.1"},
		ExitCode:  0,
		Timeout:   8 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			require.Len(t, v, 1)
			assert.Equal(t, "1.1.1.1", v[0]["ip"])
		},
	},
	{
		Name:      "host-short",
		Args:      []string{"1.1.1.1", "--output-format", "short"},
		ExitCode:  0,
		Timeout:   8 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			assert.Greater(t, len(stdout), 0)
			assert.Contains(t, string(stdout), "1.1.1.1")
		},
	},
	{
		Name:      "host-multiple",
		Args:      []string{"1.1.1.1,8.8.8.8"},
		ExitCode:  0,
		Timeout:   10 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			require.Len(t, v, 2)
			// Non-streaming output preserves input order.
			assert.Equal(t, "1.1.1.1", v[0]["ip"])
			assert.Equal(t, "8.8.8.8", v[1]["ip"])
		},
	},
	{
		Name:      "output-ndjson",
		Args:      []string{"1.1.1.1,8.8.8.8", "--streaming"},
		ExitCode:  0,
		Timeout:   10 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			// One JSON object per line; order is completion-order under streaming.
			lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
			assert.Equal(t, 2, len(lines))
		},
	},
}
