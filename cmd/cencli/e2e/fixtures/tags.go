package fixtures

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
	"github.com/censys/cencli/internal/app/tags"
)

var tagsFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "help with no args",
		Args:      []string{},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsHelpStdout, stdout, 0)
		},
	},
	// ========== list subcommand ==========
	{
		Name:      "list help",
		Args:      []string{"list", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsListHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "list invalid max-pages",
		Args:      []string{"list", "--max-pages", "0"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "max-pages")
		},
	},
	{
		Name:      "list basic",
		Args:      []string{"list", "--output-format", "json"},
		ExitCode:  0,
		Timeout:   10 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			data := unmarshalJSONAny[[]tags.Tag](t, stdout)
			for _, tag := range data {
				assert.NotEmpty(t, tag.ID)
				assert.NotEmpty(t, tag.Name)
				assert.NotEmpty(t, tag.Privacy)
			}
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
			assertGoldenFile(t, golden.TagsGetHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "get missing arg",
		Args:      []string{"get"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "accepts 1 arg")
		},
	},
	{
		// Exercises the live GetTag endpoint + API-error translation without
		// depending on org-specific tag data. A random UUID reliably 404s.
		Name:      "get not found",
		Args:      []string{"get", "00000000-0000-4000-8000-000000000000"},
		ExitCode:  1,
		Timeout:   10 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "Not Found")
		},
	},
	// ========== create subcommand ==========
	// No live create fixture: create is a non-idempotent write with no
	// deterministic teardown, and --privacy validation runs after auth. Both are
	// covered by unit tests (internal/app/tags, internal/command/tags).
	{
		Name:      "create help",
		Args:      []string{"create", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsCreateHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "create missing arg",
		Args:      []string{"create"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "accepts 1 arg")
		},
	},
}
