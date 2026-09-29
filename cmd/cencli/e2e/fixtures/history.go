package fixtures

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
)

// enableWebPropertyEventsEnvVar must be "true" to run the live --mode events
// fixture, which needs web property event history enabled for the org.
const enableWebPropertyEventsEnvVar = "CENCLI_E2E_ENABLE_WEBPROPERTY_EVENTS"

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
	{
		Name:      "webproperty-events",
		Args:      []string{"platform.censys.io:80", "--mode", "events", "--duration", "30d"},
		ExitCode:  0,
		Timeout:   20 * time.Second,
		NeedsAuth: true,
		Skip: func() string {
			if os.Getenv(enableWebPropertyEventsEnvVar) != "true" {
				return "needs web property event history enabled for the org; set " + enableWebPropertyEventsEnvVar + "=true to run"
			}
			return ""
		},
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			assertWebTimelineEvents(t, unmarshalJSONAny[[]webTimelineEvent](t, stdout))
		},
	},
	{
		Name:      "mode-on-host",
		Args:      []string{"8.8.8.8", "--mode", "events"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "--mode only applies to web properties")
		},
	},
	{
		Name:      "mode-invalid",
		Args:      []string{"platform.censys.io:80", "--mode", "daily"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), `invalid --mode "daily"`)
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

type webTimelineEvent struct {
	EventTime       string `json:"event_time"`
	EndpointScanned any    `json:"endpoint_scanned"`
	JarmScanned     any    `json:"jarm_scanned"`
}

func assertWebTimelineEvents(t *testing.T, events []webTimelineEvent) {
	t.Helper()
	assert.NotEmpty(t, events)
	for _, e := range events {
		assert.NotEmpty(t, e.EventTime)
		assert.True(t, e.EndpointScanned != nil || e.JarmScanned != nil, "event carries no scan: %+v", e)
	}
}
