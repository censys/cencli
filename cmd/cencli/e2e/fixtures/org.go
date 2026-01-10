package fixtures

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
	"github.com/censys/cencli/internal/app/credits"
	"github.com/censys/cencli/internal/app/organizations"
)

var orgFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.OrgHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "help with no args",
		Args:      []string{},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.OrgHelpStdout, stdout, 0)
		},
	},
	// ========== credits subcommand ==========
	{
		Name:      "credits help",
		Args:      []string{"credits", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.OrgCreditsHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "credits basic",
		Args:      []string{"credits", "--output-format", "json"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			data := unmarshalJSONAny[credits.OrganizationCreditDetails](t, stdout)
			assert.Greater(t, data.Balance, int64(0))
			assert.NotNil(t, data.AutoReplenishConfig)
			assert.NotNil(t, data.CreditExpirations)
			assert.Greater(t, len(data.CreditExpirations), 0)
			for _, creditExpiration := range data.CreditExpirations {
				assert.Greater(t, creditExpiration.Balance, int64(0))
				assert.NotNil(t, creditExpiration.CreationDate)
				assert.NotNil(t, creditExpiration.ExpirationDate)
			}
		},
	},
	// ========== members subcommand ==========
	{
		Name:      "members help",
		Args:      []string{"members", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.OrgMembersHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "members basic",
		Args:      []string{"members", "--output-format", "json"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			data := unmarshalJSONAny[organizations.OrganizationMembers](t, stdout)
			assert.Greater(t, len(data.Members), 0)
			for _, member := range data.Members {
				assert.NotEmpty(t, member.Email)
				assert.NotEmpty(t, member.Roles)
			}
		},
	},
	// ========== details subcommand ==========
	{
		Name:      "details help",
		Args:      []string{"details", "--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.OrgDetailsHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "details basic",
		Args:      []string{"details", "--output-format", "json"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			data := unmarshalJSONAny[organizations.OrganizationDetails](t, stdout)
			assert.NotEmpty(t, data.Name)
			assert.NotEmpty(t, data.CreatedAt)
			assert.NotEmpty(t, data.MemberCounts)
		},
	},
}
