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
		// depending on org-specific tag data. A random UUID never maps to a real
		// tag; the API masks resource existence and returns 403 Permission denied
		// (not 404), so that is the expected error here.
		Name:      "get empty tag id",
		Args:      []string{"get", ""},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "a tag name or ID is required")
		},
	},
	{
		Name:      "get forbidden for unknown id",
		Args:      []string{"get", "00000000-0000-4000-8000-000000000000"},
		ExitCode:  1,
		Timeout:   10 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "Permission denied")
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
	// ========== update subcommand ==========
	// No live update fixture: update is a non-idempotent write with no
	// deterministic fixture, and --privacy validation runs after auth. Covered
	// by unit tests (internal/app/tags, internal/command/tags).
	{
		Name:      "update help",
		Args:      []string{"update", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsUpdateHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "update missing arg",
		Args:      []string{"update"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "accepts 1 arg")
		},
	},
	{
		Name:      "update nothing to update",
		Args:      []string{"update", "my-tag"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "no fields to update")
		},
	},
	{
		Name:      "update description conflict",
		Args:      []string{"update", "my-tag", "--description", "foo", "--clear-description"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "cannot be used together")
		},
	},
	// ========== delete subcommand ==========
	{
		Name:      "delete help",
		Args:      []string{"delete", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsDeleteHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "delete missing arg",
		Args:      []string{"delete"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "accepts 1 arg")
		},
	},
	{
		// A non-interactive terminal (piped stdin) without --yes cannot prompt,
		// so the command refuses before any auth is required.
		Name:      "delete confirmation required in non-interactive terminal",
		Args:      []string{"delete", "some-tag"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "confirmation required")
		},
	},
	{
		// An empty tag id must be rejected up front, never resolved to an
		// arbitrary tag and deleted.
		Name:      "delete empty tag id",
		Args:      []string{"delete", ""},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "a tag name or ID is required")
		},
	},
	// ========== assign subcommand ==========
	// No live assign fixture: assign is a non-idempotent write with no
	// deterministic teardown. Covered by unit tests (internal/app/tags,
	// internal/command/tags).
	{
		Name:      "assign help",
		Args:      []string{"assign", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsAssignHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "assign missing arg",
		Args:      []string{"assign"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "requires at least 1 arg")
		},
	},
	{
		// A tag with no assets and no --input-file has nothing to act on.
		Name:      "assign no assets",
		Args:      []string{"assign", "my-tag"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "at least one asset")
		},
	},
	{
		// An unparseable asset is rejected before anything is sent.
		Name:      "assign unknown asset",
		Args:      []string{"assign", "my-tag", "not-an-asset"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "not-an-asset")
		},
	},
	{
		Name:      "assign empty tag id",
		Args:      []string{"assign", "", "8.8.8.8"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "a tag name or ID is required")
		},
	},
	// No live bulk-assign fixture either: a bulk job mutates at scale and cannot
	// be undone deterministically. These cover what the command rejects before
	// any request is sent.
	{
		// Bulk is never inferred, so the two input modes cannot be mixed.
		Name:      "assign query with explicit assets",
		Args:      []string{"assign", "my-tag", "8.8.8.8", "--query", "host.services.port: 22"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "cannot be combined with explicit assets")
		},
	},
	{
		Name:      "assign query with input file",
		Args:      []string{"assign", "my-tag", "--input-file", "-", "--query", "host.services.port: 22"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "cannot be combined with explicit assets")
		},
	},
	{
		// A blank query would match nothing; rejected before it can prompt.
		Name:      "assign empty query",
		Args:      []string{"assign", "my-tag", "--query", "   "},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "--query must not be empty")
		},
	},
	{
		Name:      "assign max assets without query",
		Args:      []string{"assign", "my-tag", "8.8.8.8", "--max-assets", "10"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "--max-assets only applies to a bulk assignment")
		},
	},
	{
		Name:      "assign wait without query",
		Args:      []string{"assign", "my-tag", "8.8.8.8", "--wait"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "--wait only applies to a bulk assignment")
		},
	},
	{
		Name:      "assign timeout without wait",
		Args:      []string{"assign", "my-tag", "--query", "host.services.port: 22", "--timeout", "5m"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "--timeout only applies while polling")
		},
	},
	{
		// e2e runs without a TTY, so a bulk assignment cannot prompt and must
		// refuse rather than submit silently.
		Name:      "assign query non-interactive without yes",
		Args:      []string{"assign", "my-tag", "--query", "host.services.port: 22"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "confirmation required")
		},
	},
	// ========== unassign subcommand ==========
	// No live unassign fixture: unassign is a non-idempotent write with no
	// deterministic teardown. Covered by unit tests (internal/app/tags,
	// internal/command/tags).
	{
		Name:      "unassign help",
		Args:      []string{"unassign", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsUnassignHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "unassign missing arg",
		Args:      []string{"unassign"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "requires at least 1 arg")
		},
	},
	{
		// A tag with no assets and no --input-file has nothing to act on.
		Name:      "unassign no assets",
		Args:      []string{"unassign", "my-tag"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "at least one asset")
		},
	},
	{
		// An unparseable asset is rejected before anything is sent.
		Name:      "unassign unknown asset",
		Args:      []string{"unassign", "my-tag", "not-an-asset"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "not-an-asset")
		},
	},
	{
		Name:      "unassign empty tag id",
		Args:      []string{"unassign", "", "8.8.8.8"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "a tag name or ID is required")
		},
	},
	{
		// Removing more than one asset prompts for confirmation; a non-interactive
		// run (e2e is non-TTY) without --yes must refuse rather than proceed.
		Name:      "unassign multi-asset requires confirmation",
		Args:      []string{"unassign", "my-tag", "8.8.8.8", "1.1.1.1"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "confirmation required")
		},
	},
	// ========== assignments subcommand ==========
	// No live assignments fixture: the org has no tag with a deterministic set of
	// assignments to assert against. Covered by unit tests (internal/app/tags,
	// internal/command/tags) and verified manually against the live API.
	{
		Name:      "assignments help",
		Args:      []string{"assignments", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsAssignmentsHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "assignments missing arg",
		Args:      []string{"assignments"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "accepts 1 arg")
		},
	},
	{
		Name:      "assignments empty tag id",
		Args:      []string{"assignments", ""},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "a tag name or ID is required")
		},
	},
	{
		// An unparseable --asset filter is rejected before anything is sent.
		Name:      "assignments unknown asset filter",
		Args:      []string{"assignments", "my-tag", "--asset", "not-an-asset"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "not-an-asset")
		},
	},
	{
		// The endpoint filters on one asset, so a list is a usage error.
		Name:      "assignments multiple asset filters",
		Args:      []string{"assignments", "my-tag", "--asset", "8.8.8.8,1.1.1.1"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "only 1")
		},
	},
	{
		// Page sizes above the documented maximum never reach the API.
		Name:      "assignments page-size above maximum",
		Args:      []string{"assignments", "my-tag", "--page-size", "1001"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "1000")
		},
	},
	{
		Name:      "assignments invalid max-pages",
		Args:      []string{"assignments", "my-tag", "--max-pages", "0"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "max-pages")
		},
	},
	{
		Name:      "assignments invalid timestamp",
		Args:      []string{"assignments", "my-tag", "--created-after", "yesterday"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "created-after")
		},
	},
	{
		// The API declares created_by as a UUID and 422s on anything else.
		Name:      "assignments non-uuid created-by",
		Args:      []string{"assignments", "my-tag", "--created-by", "not-a-uuid"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "not-a-uuid")
		},
	},
	{
		Name:      "list non-uuid created-by",
		Args:      []string{"list", "--created-by", "not-a-uuid"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "not-a-uuid")
		},
	},

	// ========== operations subcommand ==========
	{
		Name:      "operations help",
		Args:      []string{"operations", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsOperationsHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "operations list help",
		Args:      []string{"operations", "list", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsOperationsListHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "operations get help",
		Args:      []string{"operations", "get", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.TagsOperationsGetHelpStdout, stdout, 0)
		},
	},
	{
		// The parent lists nothing itself; subcommands do the work.
		Name:      "operations rejects a positional argument",
		Args:      []string{"operations", "my-tag"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "accepts 0 arg")
		},
	},
	{
		Name:      "operations list empty tag",
		Args:      []string{"operations", "list", "   "},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "tag name or ID is required")
		},
	},
	// No fixtures for invalid --status/--type/--order-by: those enums are checked
	// in the service layer, and PreRun resolves the service (requiring auth)
	// before Run reaches the validation, so without credentials they fail as
	// "not configured" instead. Same gap as list's --order-by/--privacy; covered
	// by the service unit tests in internal/app/tags.
	{
		Name:      "operations list invalid max-pages",
		Args:      []string{"operations", "list", "--max-pages", "0"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "max-pages")
		},
	},
	{
		// operation_id is format:uuid, so a bad one never reaches the API.
		Name:      "operations get non-uuid operation id",
		Args:      []string{"operations", "get", "my-tag", "not-a-uuid"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "not-a-uuid")
		},
	},
	{
		Name:      "operations get missing operation id",
		Args:      []string{"operations", "get", "my-tag"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "accepts 2 arg")
		},
	},
	{
		// A timeout that silently does nothing would be a dead flag.
		Name:      "operations get timeout without wait",
		Args:      []string{"operations", "get", "my-tag", "d421a231-eb5e-4927-a0be-8aa749eb731c", "--timeout", "5m"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assert.Contains(t, string(stderr), "--timeout only applies while polling")
		},
	},
	{
		// Read-only and safe to run live; the org may legitimately have none.
		Name:      "operations list basic",
		Args:      []string{"operations", "list", "--output-format", "json"},
		ExitCode:  0,
		Timeout:   10 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			data := unmarshalJSONAny[[]tags.TagOperation](t, stdout)
			for _, op := range data {
				assert.NotEmpty(t, op.ID)
				assert.NotEmpty(t, op.TagID)
				assert.Contains(t, []string{"bulk_create", "bulk_delete"}, op.Type)
				assert.Contains(t,
					[]string{"pending", "running", "succeeded", "limit_reached", "failed", "cancelled"},
					op.Status)
			}
		},
	},
}
