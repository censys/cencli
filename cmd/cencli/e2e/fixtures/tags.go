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
}
