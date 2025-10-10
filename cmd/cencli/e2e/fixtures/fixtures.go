package fixtures

import (
	"testing"
	"time"
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
	// AssertSchema is the function that asserts the expected output.
	Assert func(t *testing.T, stdout, stderr []byte)
}

// Fixtures returns all the fixtures for the e2ev2 tests.
func Fixtures() map[string][]Fixture {
	return map[string][]Fixture{
		"view":      viewFixtures,
		"aggregate": aggregateFixtures,
		"search":    searchFixtures,
		"censeye":   censeyeFixtures,
		"history":   historyFixtures,
	}
}
