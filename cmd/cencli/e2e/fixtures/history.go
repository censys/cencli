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
		// The only live web property fixture, since the timeline costs credits.
		// 30d: the property is scanned about weekly, so 7d can be empty.
		Name:      "webproperty-basic",
		Args:      []string{"platform.censys.io:80", "--duration", "30d"},
		ExitCode:  0,
		Timeout:   20 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			assertWebTimelineEvents(t, unmarshalJSONAny[[]webTimelineEvent](t, stdout))
		},
	},
	// Output format tests
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
