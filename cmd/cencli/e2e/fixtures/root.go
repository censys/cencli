package fixtures

import (
	"testing"
	"time"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
)

var RootFixtures = []Fixture{
	{
		Name:      "base",
		Args:      []string{},
		ExitCode:  0,
		Timeout:   5 * time.Second, // give more time since this will be run on all platforms
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.RootStdout, stdout, 0)
		},
	},
}
