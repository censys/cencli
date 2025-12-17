package fixtures

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
	"github.com/censys/cencli/internal/app/credits"
)

var creditsFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.CreditsHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "basic",
		Args:      []string{"--output-format", "json"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			data := unmarshalJSONAny[credits.UserCreditDetails](t, stdout)
			assert.Greater(t, data.Balance, int64(0))
			assert.NotNil(t, data.ResetsAt)
		},
	},
}
