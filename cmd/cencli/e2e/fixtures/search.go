package fixtures

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/templates"
)

var searchFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.SearchHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "basic - 2 pages",
		Args:      []string{"host.services: (protocol=SSH)", "-n", "2", "-p", "2"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			assert.Contains(t, string(stderr), "pages: 2")
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			assert.Len(t, v, 4)
		},
	},
	{
		Name:      "output-json-default",
		Args:      []string{"host.services: (protocol=SSH)", "-n", "2", "-p", "1"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			// Default is JSON output
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			assert.Len(t, v, 2)
		},
	},
	{
		Name:      "output-short",
		Args:      []string{"host.services: (protocol=SSH)", "-n", "2", "--output-format", "short"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			// Just verify short output exists (don't validate content)
			assert.Greater(t, len(stdout), 0)
		},
	},
	{
		Name:      "output-template",
		Args:      []string{"host.services: (protocol=SSH)", "-n", "2", "-p", "2", "--output-format", "template"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			assertRenderedTemplate(t, templates.SearchResultTemplate, stdout)
		},
	},
	{
		Name:      "output-yaml",
		Args:      []string{"host.services: (protocol=SSH)", "-n", "2", "--output-format", "yaml"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			// Verify YAML output format
			assert.Contains(t, string(stdout), "host:")
		},
	},
}
