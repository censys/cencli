package fixtures

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
)

var dnsFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertGoldenFile(t, golden.DNSHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "invalid-port",
		Args:      []string{"censys.com:443"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Asset ID]", lines[0])
			assert.Contains(t, lines[1], "remove the port")
			rest := strings.Join(lines[2:], "\n")
			// dns's Long description is 17 lines (unlike view's 2), so this
			// strips the same number of lines from the golden file to land on
			// "Usage:" for comparison against the error's own Usage block.
			assertGoldenFile(t, golden.DNSHelpStdout, []byte(rest), 17)
		},
	},
	{
		Name:      "bracketed-ipv6",
		Args:      []string{"[2001:db8::1]"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Asset ID]", lines[0])
			assert.Contains(t, lines[1], "remove the brackets")
		},
	},
	{
		Name:      "cidr",
		Args:      []string{"8.8.8.8/32"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Asset ID]", lines[0])
			assert.Contains(t, lines[1], "a CIDR range is not supported")
		},
	},
	{
		Name:      "reject-cidr-defanged",
		Args:      []string{"8[.]8[.]8[.]8/32"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Asset ID]", lines[0])
			assert.Contains(t, lines[1], "a CIDR range is not supported")
		},
	},
	{
		Name:      "reject-cidr-defanged-slash",
		Args:      []string{"8.8.8.8[/]32"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Asset ID]", lines[0])
			assert.Contains(t, lines[1], "a CIDR range is not supported")
		},
	},
	{
		Name:      "reject-too-many-inputs",
		Args:      []string{tooManyDNSInputs()},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Too Many Assets]", lines[0])
			assert.Contains(t, lines[1], "101 assets provided, only 100 are supported")
		},
	},
	{
		Name:      "reject-list-with-invalid-member",
		Args:      []string{"censys.com,a..b.com"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Asset ID]", lines[0])
			assert.Contains(t, lines[1], "a..b.com")
			assert.Contains(t, lines[1], "empty label")
		},
	},
	{
		Name:      "email-in-name",
		Args:      []string{"user@censys.com"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Asset ID]", lines[0])
			assert.Contains(t, lines[1], "cannot contain '@'")
		},
	},
	{
		Name:      "empty-label",
		Args:      []string{"a..b.com"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Asset ID]", lines[0])
			assert.Contains(t, lines[1], "empty label")
		},
	},
	{
		Name:      "duration-negative",
		Args:      []string{"censys.com", "--duration", "-1h"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Time Window]", lines[0])
			assert.Contains(t, lines[1], "duration must be greater than 0")
		},
	},
	{
		Name:      "invalid-record-type",
		Args:      []string{"104.18.10.84", "-r", "MX"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Record Type]", lines[0])
			assert.Contains(t, lines[1], "invalid record type 'MX'")
		},
	},
	{
		Name:      "invalid-record-type-mixed-inputs",
		Args:      []string{"censys.com,104.18.10.84", "-r", "MX"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Record Type]", lines[0])
			assert.Contains(t, lines[1], "invalid record type 'MX'")
			assert.Contains(t, lines[1], "an IP address is among the inputs")
		},
	},
	{
		Name:      "domain-with-name-input",
		Args:      []string{"censys.com", "--timeline", "--domain", "x.com"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Conflicting Flags]", lines[0])
			assert.Contains(t, lines[1], "--domain applies only to an IP lookup with --timeline")
		},
	},
	{
		Name:      "domain-without-timeline",
		Args:      []string{"104.18.10.84", "--domain", "censys.com"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Conflicting Flags]", lines[0])
			assert.Contains(t, lines[1], "--domain applies only to an IP lookup with --timeline")
		},
	},
	{
		Name:      "reject-empty-domain",
		Args:      []string{"104.18.10.84", "--timeline", "--domain", ""},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Flag Value]", lines[0])
			assert.Contains(t, lines[1], "--domain needs a domain name")
		},
	},
	{
		Name:      "name-json",
		Args:      []string{"censys.com", "--output-format", "json"},
		ExitCode:  0,
		Timeout:   12 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			v := unmarshalJSONAny[[]struct {
				Input      string `json:"input"`
				RecordType string `json:"record_type"`
			}](t, stdout)
			assert.NotEmpty(t, v)
			for _, r := range v {
				assert.Equal(t, "censys.com", r.Input)
			}
		},
	},
	{
		Name:      "ip-short",
		Args:      []string{"8.8.8.8", "-p", "1"},
		ExitCode:  0,
		Timeout:   12 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, stdout, stderr []byte) {
			assertHas200(t, stderr)
			assert.Contains(t, string(stdout), "Domains resolving to 8.8.8.8")
		},
	},
}

// tooManyDNSInputs returns a comma list of 101 distinct names, one more than
// dns accepts in one command.
func tooManyDNSInputs() string {
	names := make([]string, 0, 101)
	for i := range 101 {
		names = append(names, fmt.Sprintf("host%d.example.com", i))
	}
	return strings.Join(names, ",")
}
