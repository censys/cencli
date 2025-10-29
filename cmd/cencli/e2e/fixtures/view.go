package fixtures

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/golden"
	"github.com/censys/cencli/cmd/cencli/e2e/fixtures/templates"
)

var viewFixtures = []Fixture{
	{
		Name:      "help",
		Args:      []string{"--help"},
		ExitCode:  0,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			assertGoldenFile(t, golden.ViewHelpStdout, stdout, 0)
		},
	},
	{
		Name:      "invalid asset type",
		Args:      []string{"invalid"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Asset ID]", lines[0])
			rest := strings.Join(lines[2:], "\n")
			assertGoldenFile(t, golden.ViewHelpStdout, []byte(rest), 2)
		},
	},
	// ================================================
	// Host fixtures
	// ================================================
	{
		Name:      "host-basic",
		Args:      []string{"1.1.1.1"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			assert.Len(t, v, 1)
			assert.Equal(t, "1.1.1.1", v[0]["ip"])
			assertHas200(t, stderr)
		},
	},
	{
		Name:      "host-at-time",
		Args:      []string{"1.1.1.1", "--at-time", "2025-09-15T14:30:00Z"},
		ExitCode:  0,
		Timeout:   8 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			require.Len(t, v, 1)
			assert.Equal(t, "1.1.1.1", v[0]["ip"])
			assertHas200(t, stderr)
		},
	},
	{
		Name:      "host-short",
		Args:      []string{"1.1.1.1", "--short"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			assertHas200(t, stderr)
			assertRenderedTemplate(t, templates.HostTemplate, stdout)
		},
	},
	{
		Name:      "host-multiple",
		Args:      []string{"1.1.1.1,8.8.8.8"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			require.Len(t, v, 2)
			assert.Equal(t, "1.1.1.1", v[0]["ip"])
			assert.Equal(t, "8.8.8.8", v[1]["ip"])
			assertHas200(t, stderr)
		},
	},
	// invalid at time
	{
		Name:      "host-invalid-at-time",
		Args:      []string{"1.1.1.1", "--at-time", "2025-09-15T14:3Z"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[Invalid Timestamp]", lines[0])
			rest := strings.Join(lines[2:], "\n")
			assertGoldenFile(t, golden.ViewHelpStdout, []byte(rest), 2)
		},
	},
	// ================================================
	// Certificate fixtures
	// ================================================
	{
		Name:      "certificate-basic",
		Args:      []string{"3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			require.Len(t, v, 1)
			assert.Equal(t, "3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf", v[0]["fingerprint_sha256"])
			assertHas200(t, stderr)
		},
	},
	{
		Name:      "certificate-short",
		Args:      []string{"3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf", "--short"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			assertHas200(t, stderr)
			assertRenderedTemplate(t, templates.CertificateTemplate, stdout)
		},
	},
	{
		Name:      "certificate-multiple",
		Args:      []string{"3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf,0043fed3f8df7cc9a9bddb3136e06e4ef3fba94c7a8e5b55567bda68d8adde5a"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			require.Len(t, v, 2)
			assert.Equal(t, "3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf", v[0]["fingerprint_sha256"])
			assert.Equal(t, "0043fed3f8df7cc9a9bddb3136e06e4ef3fba94c7a8e5b55567bda68d8adde5a", v[1]["fingerprint_sha256"])
			assertHas200(t, stderr)
		},
	},
	// at time not supported
	{
		Name:      "certificate-at-time",
		Args:      []string{"3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf", "--at-time", "2025-09-15T14:30:00Z"},
		ExitCode:  2,
		Timeout:   1 * time.Second,
		NeedsAuth: false,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			lines := strings.Split(string(stderr), "\n")
			assert.Greater(t, len(lines), 3)
			assert.Equal(t, "[At-Time Not Supported]", lines[0])
			rest := strings.Join(lines[2:], "\n")
			assertGoldenFile(t, golden.ViewHelpStdout, []byte(rest), 2)
		},
	},
	// ================================================
	// Web property fixtures
	// ================================================
	{
		Name:      "web-property-basic",
		Args:      []string{"platform.censys.io:80"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			require.Len(t, v, 1)
			assert.Equal(t, "platform.censys.io", v[0]["hostname"])
			assert.Equal(t, float64(80), v[0]["port"])
			assertHas200(t, stderr)
		},
	},
	{
		Name:      "web-property-at-time",
		Args:      []string{"platform.censys.io:80", "--at-time", "2025-09-15T14:30:00Z"},
		ExitCode:  0,
		Timeout:   8 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			require.Len(t, v, 1)
			assert.Equal(t, "platform.censys.io", v[0]["hostname"])
			assert.Equal(t, float64(80), v[0]["port"])
			assertHas200(t, stderr)
		},
	},
	{
		Name:      "web-property-short",
		Args:      []string{"platform.censys.io:80", "--short"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			assertRenderedTemplate(t, templates.WebPropertyTemplate, stdout)
			assertHas200(t, stderr)
		},
	},
	{
		Name:      "web-property-multiple",
		Args:      []string{"platform.censys.io:80,google.com:443"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			require.Len(t, v, 2)
			assert.Equal(t, "platform.censys.io", v[0]["hostname"])
			assert.Equal(t, float64(80), v[0]["port"])
			assert.Equal(t, "google.com", v[1]["hostname"])
			assert.Equal(t, float64(443), v[1]["port"])
			assertHas200(t, stderr)
		},
	},
	{
		Name:      "web-property-default-port",
		Args:      []string{"platform.censys.io"},
		ExitCode:  0,
		Timeout:   5 * time.Second,
		NeedsAuth: true,
		Assert: func(t *testing.T, _ string, stdout, stderr []byte) {
			v := unmarshalJSONAny[[]map[string]any](t, stdout)
			require.Len(t, v, 1)
			assert.Equal(t, "platform.censys.io", v[0]["hostname"])
			assert.Equal(t, float64(443), v[0]["port"])
			assertHas200(t, stderr)
		},
	},
}
