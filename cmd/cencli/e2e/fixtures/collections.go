package fixtures

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
	"github.com/censys/cencli/internal/app/collections"
)

var collectionsFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.CollectionsHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "help with no args",
		Args:      []string{},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.CollectionsHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "list help",
		Args:      []string{"list", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.CollectionsListHelpStdout, stdout, 0)
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
		Name:      "list invalid page-size",
		Args:      []string{"list", "--page-size", "0"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "--page-size was set with an invalid value")
		},
	},
	{
		Name:      "list invalid org-id",
		Args:      []string{"list", "--org-id", "not-a-uuid"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "invalid uuid")
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
			data := unmarshalJSONAny[[]collections.Collection](t, stdout)
			for _, c := range data {
				assert.NotEmpty(t, c.ID)
			}
		},
	},
	{
		Name:      "get help",
		Args:      []string{"get", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.CollectionsGetHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "get invalid id",
		Args:      []string{"get", "not-a-uuid"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "is not a valid UUID")
		},
	},
	{
		Name:      "get unsupported output format",
		Args:      []string{"get", "550e8400-e29b-41d4-a716-446655440000", "-O", "template"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "output format 'template' is not supported")
		},
	},
	// Read-only live check: an unknown ID is a 404, not a crash or an empty result.
	{
		Name:      "get not found for unknown id",
		Args:      []string{"get", "00000000-0000-4000-8000-000000000000"},
		ExitCode:  1,
		Timeout:   10 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "resource not found")
		},
	},
	{
		Name:      "create help",
		Args:      []string{"create", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.CollectionsCreateHelpStdout, stdout, 0)
		},
	},
	// No live create, update, or delete fixture: each create uses a slot against
	// the plan's collection limit. The input checks below send no request.
	{
		Name:      "create missing --query",
		Args:      []string{"create", "web"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "--query is required")
		},
	},
	{
		Name:      "create blank name",
		Args:      []string{"create", "", "--query", "q"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "collection name must not be empty")
		},
	},
	{
		Name:      "update help",
		Args:      []string{"update", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.CollectionsUpdateHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "update with nothing to update",
		Args:      []string{"update", "550e8400-e29b-41d4-a716-446655440000"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "no fields to update")
		},
	},
	{
		Name:      "update description conflict",
		Args:      []string{"update", "550e8400-e29b-41d4-a716-446655440000", "--description", "d", "--clear-description"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "cannot be used together")
		},
	},
	{
		Name:      "update blank name",
		Args:      []string{"update", "550e8400-e29b-41d4-a716-446655440000", "--name", ""},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "--name cannot be blank")
		},
	},
	{
		Name:      "update invalid id",
		Args:      []string{"update", "not-a-uuid", "--name", "x"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "is not a valid UUID")
		},
	},
	{
		Name:      "delete help",
		Args:      []string{"delete", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.CollectionsDeleteHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "delete non-interactive without --yes",
		Args:      []string{"delete", "550e8400-e29b-41d4-a716-446655440000"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "confirmation required")
		},
	},
	{
		Name:      "delete invalid id",
		Args:      []string{"delete", "not-a-uuid", "--yes"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "is not a valid UUID")
		},
	},
}
