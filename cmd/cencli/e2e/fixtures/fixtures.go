package fixtures

import (
	"testing"
	"time"

	"github.com/samber/mo"
)

type Fixture struct {
	// Name is the name of the fixture.
	Name string
	// Args is the command arguments.
	Args []string
	// ExitCode is the expected exit code of the command.
	ExitCode int
	// Timeout is the maximum time to wait for the command to complete.
	Timeout time.Duration
	// NeedsAuth is set to configure a PAT and org ID before running the fixture.
	NeedsAuth bool
	// Setup is an optional function that is called after the binary is initially invoked
	// to setup the generated files and directories, but before the command is run.
	Setup mo.Option[func(t *testing.T, dataDir string)]
	// Assert is the function that asserts the expected output.
	Assert func(t *testing.T, dataDir string, stdout, stderr []byte)
}

// Fixtures returns all the fixtures for the e2ev2 tests.
func Fixtures() map[string][]Fixture {
	return map[string][]Fixture{
		"root":      RootFixtures,
		"view":      viewFixtures,
		"aggregate": aggregateFixtures,
		"search":    searchFixtures,
		"censeye":   censeyeFixtures,
		"history":   historyFixtures,
		"config":    ConfigFixtures,
	}
}
