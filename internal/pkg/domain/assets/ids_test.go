package assets

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHostID(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantValue   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid ipv4",
			input:     "8.8.8.8",
			wantValue: "8.8.8.8",
			wantErr:   false,
		},
		{
			name:      "valid ipv4 with leading/trailing whitespace",
			input:     "  192.168.1.1  ",
			wantValue: "192.168.1.1",
			wantErr:   false,
		},
		{
			name:      "valid ipv6",
			input:     "2001:4860:4860::8888",
			wantValue: "2001:4860:4860::8888",
			wantErr:   false,
		},
		{
			name:      "valid ipv6 full form",
			input:     "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantValue: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantErr:   false,
		},
		{
			name:      "defanged ipv4 with brackets",
			input:     "8[.]8[.]8[.]8",
			wantValue: "8.8.8.8",
			wantErr:   false,
		},
		{
			name:      "defanged ipv4 with parentheses",
			input:     "192(.)168(.)1(.)1",
			wantValue: "192.168.1.1",
			wantErr:   false,
		},
		{
			name:      "defanged ipv4 with backslash",
			input:     "10\\.0\\.0\\.1",
			wantValue: "10.0.0.1",
			wantErr:   false,
		},
		{
			name:      "defanged ipv4 mixed patterns",
			input:     "172[.]16(.)0\\.1",
			wantValue: "172.16.0.1",
			wantErr:   false,
		},
		{
			name:      "defanged ipv4 with spaces",
			input:     "8[ . ]8[ . ]8[ . ]8",
			wantValue: "8.8.8.8",
			wantErr:   false,
		},
		{
			name:      "defanged ipv6 with brackets",
			input:     "2001[:]4860[:]4860[:][:]8888",
			wantValue: "2001:4860:4860::8888",
			wantErr:   false,
		},
		{
			name:        "invalid - not an ip",
			input:       "not-an-ip",
			wantErr:     true,
			errContains: "invalid host id",
		},
		{
			name:        "invalid - domain name",
			input:       "example.com",
			wantErr:     true,
			errContains: "invalid host id",
		},
		{
			name:        "invalid - empty string",
			input:       "",
			wantErr:     true,
			errContains: "invalid host id",
		},
		{
			name:        "invalid - malformed ip",
			input:       "256.256.256.256",
			wantErr:     true,
			errContains: "invalid host id",
		},
		{
			name:        "invalid - incomplete ip",
			input:       "192.168.1",
			wantErr:     true,
			errContains: "invalid host id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewHostID(tt.input)

			if tt.wantErr {
				require.Error(t, err, "expected error for input: %q", tt.input)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err, "unexpected error for input: %q", tt.input)
				assert.Equal(t, tt.wantValue, result.String(), "parsed value mismatch")
			}
		})
	}
}

func TestNewCertificateFingerprint(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantValue   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid sha256 fingerprint",
			input:     "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			wantValue: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			wantErr:   false,
		},
		{
			name:      "valid sha256 with uppercase",
			input:     "2CF24DBA5FB0A30E26E83B2AC5B9E29E1B161E5C1FA7425E73043362938B9824",
			wantValue: "2CF24DBA5FB0A30E26E83B2AC5B9E29E1B161E5C1FA7425E73043362938B9824",
			wantErr:   false,
		},
		{
			name:      "valid sha256 with leading/trailing whitespace",
			input:     "  2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824  ",
			wantValue: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			wantErr:   false,
		},
		{
			name:      "valid sha256 all zeros",
			input:     "0000000000000000000000000000000000000000000000000000000000000000",
			wantValue: "0000000000000000000000000000000000000000000000000000000000000000",
			wantErr:   false,
		},
		{
			name:        "invalid - too short",
			input:       "abcd",
			wantErr:     true,
			errContains: "invalid certificate fingerprint length",
		},
		{
			name:        "invalid - too long",
			input:       "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b98241",
			wantErr:     true,
			errContains: "invalid certificate fingerprint length",
		},
		{
			name:        "invalid - non-hex characters",
			input:       "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9XYZ",
			wantErr:     true,
			errContains: "invalid certificate fingerprint hex",
		},
		{
			name:        "invalid - empty string",
			input:       "",
			wantErr:     true,
			errContains: "invalid certificate fingerprint length",
		},
		{
			name:        "invalid - spaces in middle",
			input:       "2cf24dba 5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			wantErr:     true,
			errContains: "invalid certificate fingerprint length",
		},
		{
			name:        "invalid - special characters",
			input:       "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b98!@",
			wantErr:     true,
			errContains: "invalid certificate fingerprint hex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewCertificateFingerprint(tt.input)

			if tt.wantErr {
				require.Error(t, err, "expected error for input: %q", tt.input)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err, "unexpected error for input: %q", tt.input)
				assert.Equal(t, tt.wantValue, result.String(), "parsed value mismatch")
			}
		})
	}
}

func TestNewWebPropertyID(t *testing.T) {
	const defaultPort = 443
	tests := []struct {
		name         string
		input        string
		wantHostname string
		wantPort     int
		wantErr      bool
		errContains  string
	}{
		// === Valid Hostname Cases (with explicit port) ===
		{
			name:         "hostname with port - basic",
			input:        "example.com:8080",
			wantHostname: "example.com",
			wantPort:     8080,
		},
		{
			name:         "hostname with port - https",
			input:        "platform.censys.io:443",
			wantHostname: "platform.censys.io",
			wantPort:     443,
		},
		{
			name:         "hostname with port - http",
			input:        "example.com:80",
			wantHostname: "example.com",
			wantPort:     80,
		},
		{
			name:         "hostname with port - max valid port",
			input:        "example.com:65535",
			wantHostname: "example.com",
			wantPort:     65535,
		},
		{
			name:         "hostname with port - subdomain",
			input:        "api.v2.example.com:9000",
			wantHostname: "api.v2.example.com",
			wantPort:     9000,
		},

		// === Valid Hostname Cases (without explicit port - use default) ===
		{
			name:         "hostname without port - uses default",
			input:        "example.com",
			wantHostname: "example.com",
			wantPort:     defaultPort,
		},
		{
			name:         "hostname without port - multi-level subdomain",
			input:        "api.v2.staging.example.com",
			wantHostname: "api.v2.staging.example.com",
			wantPort:     defaultPort,
		},
		{
			name:         "hostname without port - with whitespace",
			input:        "  example.com  ",
			wantHostname: "example.com",
			wantPort:     defaultPort,
		},

		// === Valid Hostname Cases (with protocol prefix) ===
		{
			name:         "hostname with https:// prefix and port",
			input:        "https://platform.censys.io:443",
			wantHostname: "platform.censys.io",
			wantPort:     443,
		},
		{
			name:         "hostname with http:// prefix and port",
			input:        "http://example.com:80",
			wantHostname: "example.com",
			wantPort:     80,
		},
		{
			name:         "hostname with https:// prefix without port",
			input:        "https://example.com",
			wantHostname: "example.com",
			wantPort:     defaultPort,
		},
		{
			name:         "hostname with http:// prefix without port",
			input:        "http://api.example.com",
			wantHostname: "api.example.com",
			wantPort:     defaultPort,
		},

		// === Valid Defanged Hostname Cases ===
		{
			name:         "defanged hostname with brackets and port",
			input:        "example[.]com:443",
			wantHostname: "example.com",
			wantPort:     443,
		},
		{
			name:         "defanged hostname mixed patterns",
			input:        "sub[.]example(.)com:443",
			wantHostname: "sub.example.com",
			wantPort:     443,
		},
		{
			name:         "defanged hostname with protocol but no port",
			input:        "https://example[.]com",
			wantHostname: "example.com",
			wantPort:     defaultPort,
		},

		// === Valid IPv4 Cases (with explicit port) ===
		{
			name:         "ipv4 with port",
			input:        "192.168.1.1:443",
			wantHostname: "192.168.1.1",
			wantPort:     443,
		},
		{
			name:         "ipv4 with http prefix and port",
			input:        "http://8.8.8.8:80",
			wantHostname: "8.8.8.8",
			wantPort:     80,
		},

		// === Valid IPv4 Cases (without explicit port - use default) ===
		{
			name:         "ipv4 without port - uses default",
			input:        "192.168.1.1",
			wantHostname: "192.168.1.1",
			wantPort:     defaultPort,
		},
		{
			name:         "ipv4 without port - with http prefix",
			input:        "http://10.0.0.1",
			wantHostname: "10.0.0.1",
			wantPort:     defaultPort,
		},
		{
			name:         "ipv4 without port - with https prefix",
			input:        "https://8.8.8.8",
			wantHostname: "8.8.8.8",
			wantPort:     defaultPort,
		},

		// === Valid Defanged IPv4 Cases ===
		{
			name:         "defanged ipv4 with brackets and port",
			input:        "192[.]168[.]1[.]1:8080",
			wantHostname: "192.168.1.1",
			wantPort:     8080,
		},
		{
			name:         "defanged ipv4 with parentheses and port",
			input:        "10(.)0(.)0(.)1:443",
			wantHostname: "10.0.0.1",
			wantPort:     443,
		},
		{
			name:         "defanged ipv4 with http prefix and port",
			input:        "http://192[.]168[.]1[.]1:80",
			wantHostname: "192.168.1.1",
			wantPort:     80,
		},
		{
			name:         "defanged ipv4 without port",
			input:        "192[.]168[.]1[.]1",
			wantHostname: "192.168.1.1",
			wantPort:     defaultPort,
		},
		{
			name:         "defanged ipv4 with protocol but no port",
			input:        "https://10[.]0[.]0[.]1",
			wantHostname: "10.0.0.1",
			wantPort:     defaultPort,
		},

		// === Valid IPv6 Cases (with explicit port) ===
		{
			name:         "ipv6 with port - short form",
			input:        "[2001:4860:4860::8888]:443",
			wantHostname: "2001:4860:4860::8888",
			wantPort:     443,
		},
		{
			name:         "ipv6 with port - full form",
			input:        "[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:8080",
			wantHostname: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantPort:     8080,
		},
		{
			name:         "ipv6 with port - localhost",
			input:        "[::1]:9000",
			wantHostname: "::1",
			wantPort:     9000,
		},
		{
			name:         "ipv6 with port - all zeros",
			input:        "[::]:443",
			wantHostname: "::",
			wantPort:     443,
		},
		{
			name:         "ipv6 with http prefix and port",
			input:        "http://[2001:db8::1]:80",
			wantHostname: "2001:db8::1",
			wantPort:     80,
		},
		{
			name:         "ipv6 with https prefix and port",
			input:        "https://[fe80::1]:443",
			wantHostname: "fe80::1",
			wantPort:     443,
		},

		// === Valid IPv6 Cases (without explicit port - use default) ===
		{
			name:         "ipv6 without port - uses default",
			input:        "2001:4860:4860::8888",
			wantHostname: "2001:4860:4860::8888",
			wantPort:     defaultPort,
		},
		{
			name:         "ipv6 without port - localhost",
			input:        "::1",
			wantHostname: "::1",
			wantPort:     defaultPort,
		},
		{
			name:         "ipv6 without port - with http prefix",
			input:        "http://2001:db8::1",
			wantHostname: "2001:db8::1",
			wantPort:     defaultPort,
		},
		{
			name:         "ipv6 without port - with https prefix",
			input:        "https://fe80::1",
			wantHostname: "fe80::1",
			wantPort:     defaultPort,
		},
		{
			name:         "ipv6 without port - full form",
			input:        "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantHostname: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantPort:     defaultPort,
		},

		// === Invalid Cases - Empty/Missing Hostname ===
		{
			name:        "empty string",
			input:       "",
			wantErr:     true,
			errContains: "invalid webproperty",
		},
		{
			name:        "only whitespace",
			input:       "   ",
			wantErr:     true,
			errContains: "invalid webproperty",
		},
		{
			name:        "only port",
			input:       ":443",
			wantErr:     true,
			errContains: "missing hostname",
		},
		{
			name:        "only protocol",
			input:       "http://",
			wantErr:     true,
			errContains: "invalid webproperty",
		},
		{
			name:        "only protocol with colon",
			input:       "https://:",
			wantErr:     true,
			errContains: "missing hostname",
		},

		// === Invalid Cases - Bad Hostname (no period, not IP) ===
		{
			name:        "hostname without period - single word",
			input:       "localhost:443",
			wantErr:     true,
			errContains: "invalid hostname",
		},
		{
			name:        "hostname without period - just text",
			input:       "abc:443",
			wantErr:     true,
			errContains: "invalid hostname",
		},
		{
			name:        "hostname without period - no port",
			input:       "localhost",
			wantErr:     true,
			errContains: "invalid hostname",
		},
		{
			name:        "hostname without period - hyphenated",
			input:       "my-server:8080",
			wantErr:     true,
			errContains: "invalid hostname",
		},

		// === Invalid Cases - Bad Port ===
		{
			name:        "port zero",
			input:       "example.com:0",
			wantErr:     true,
			errContains: "invalid port",
		},
		{
			name:        "negative port",
			input:       "example.com:-1",
			wantErr:     true,
			errContains: "invalid port",
		},
		{
			name:        "port too high",
			input:       "example.com:65536",
			wantErr:     true,
			errContains: "invalid port",
		},
		{
			name:        "port too high - way over",
			input:       "example.com:99999",
			wantErr:     true,
			errContains: "invalid port",
		},
		{
			name:        "non-numeric port",
			input:       "example.com:abc",
			wantErr:     true,
			errContains: "invalid port",
		},
		{
			name:         "port with spaces",
			input:        "example.com: 443",
			wantHostname: "example.com",
			wantPort:     443,
		},
		{
			name:        "seemingly multiple ports - ambiguous",
			input:       "example.com:80:443",
			wantErr:     true,
			errContains: "invalid webproperty",
		},

		// === Invalid Cases - Malformed IPv6 ===
		{
			name:        "ipv6 without brackets and with port - ambiguous",
			input:       "2001:4860:4860::8888:443",
			wantErr:     true,
			errContains: "invalid webproperty",
		},
		{
			name:         "ipv6 short form with port",
			input:        "[2001::]:443",
			wantHostname: "2001::",
			wantPort:     443,
		},
		{
			name:        "ipv6 with single bracket",
			input:       "[2001:db8::1:443",
			wantErr:     true,
			errContains: "invalid webproperty",
		},

		// === Invalid Cases - Invalid IPv4 ===
		{
			name:        "invalid ipv4",
			input:       "192.168.1.256",
			wantErr:     true,
			errContains: "invalid webproperty",
		},

		// === Edge Cases ===
		{
			name:         "hostname with trailing colon but no port - uses default",
			input:        "example.com:",
			wantHostname: "example.com",
			wantPort:     defaultPort,
		},
		{
			name:         "ipv4 with trailing colon but no port - uses default",
			input:        "192.168.1.1:",
			wantHostname: "192.168.1.1",
			wantPort:     defaultPort,
		},
		{
			name:         "hostname with port 1",
			input:        "example.com:1",
			wantHostname: "example.com",
			wantPort:     1,
		},
		{
			name:         "lots of whitespace",
			input:        "   example.com:443   ",
			wantHostname: "example.com",
			wantPort:     443,
		},
		{
			name:         "mixed case hostname",
			input:        "Example.COM:443",
			wantHostname: "Example.COM",
			wantPort:     443,
		},
		{
			name:         "hostname with hyphen",
			input:        "my-api.example.com:443",
			wantHostname: "my-api.example.com",
			wantPort:     443,
		},
		{
			name:         "hostname with numbers",
			input:        "api1.example2.com:443",
			wantHostname: "api1.example2.com",
			wantPort:     443,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewWebPropertyID(tt.input, defaultPort)

			if tt.wantErr {
				require.Error(t, err, "expected error for input: %q", tt.input)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains, "error message should contain expected string")
				}
			} else {
				require.NoError(t, err, "unexpected error for input: %q", tt.input)
				assert.Equal(t, tt.wantHostname, result.Hostname, "hostname mismatch")
				assert.Equal(t, tt.wantPort, result.Port, "port mismatch")

				// Verify String() method produces correct output
				expectedString := fmt.Sprintf("%s:%d", tt.wantHostname, tt.wantPort)
				assert.Equal(t, expectedString, result.String(), "String() output should match hostname:port")
			}
		})
	}
}
