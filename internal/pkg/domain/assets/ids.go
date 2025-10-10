package assets

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/censys/cencli/internal/pkg/refang"
)

// DefaultWebPropertyPort is the default port that will
// be used if no port is specified in the input.
const DefaultWebPropertyPort = 443

// HostID represents a validated IP address (v4 or v6).
type HostID struct{ value string }

func (h HostID) String() string { return h.value }

// NewHostID parses an IP address into a HostID.
// Supports defanged IPs with [.] or (.) patterns.
func NewHostID(raw string) (HostID, error) {
	refanged := refang.RefangIP(raw)
	trimmed := strings.TrimSpace(refanged)
	if ip := net.ParseIP(trimmed); ip != nil {
		return HostID{value: trimmed}, nil
	}
	return HostID{}, fmt.Errorf("invalid host id: %q", raw)
}

// CertificateID represents a validated SHA-256 hex string (64 chars).
type CertificateID struct{ value string }

func (c CertificateID) String() string { return c.value }

// NewCertificateFingerprint parses a sha256 hex string into a CertificateID.
func NewCertificateFingerprint(raw string) (CertificateID, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) != 64 {
		return CertificateID{}, fmt.Errorf("invalid certificate fingerprint length: %d", len(trimmed))
	}
	if _, err := hex.DecodeString(trimmed); err != nil {
		return CertificateID{}, fmt.Errorf("invalid certificate fingerprint hex: %w", err)
	}
	return CertificateID{value: trimmed}, nil
}

// WebPropertyID represents a hostname:port tuple.
type WebPropertyID struct {
	Hostname string
	Port     int
}

func (w WebPropertyID) String() string { return fmt.Sprintf("%s:%d", w.Hostname, w.Port) }

// looksLikeIP returns true if the string appears to be an IP address (v4 or v6).
func looksLikeIP(s string) bool {
	// IPv6 contains colons
	if strings.Contains(s, ":") {
		return true
	}
	// IPv4: check if all parts are numeric
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

// NewWebPropertyID parses a hostname:port string into a WebPropertyID.
// Supports refanged URLs with http:// or https:// prefixes.
// Hostnames that are not IP addresses must have a period.
// Handles IPv6 addresses with or without brackets. For compressed IPv6 addresses
// (containing ::) that end with a numeric segment that could be a port (e.g., ::1:8080),
// an error is returned to avoid ambiguity. Use brackets to disambiguate.
// If no port is specified, defaultPort is used.
func NewWebPropertyID(raw string, defaultPort int) (WebPropertyID, error) {
	refanged := refang.RefangURL(raw)
	trimmed := strings.TrimSpace(refanged)
	trimmed = strings.TrimPrefix(strings.TrimPrefix(trimmed, "http://"), "https://")

	h, p, err := net.SplitHostPort(trimmed)
	if err != nil {
		// Add default port if missing
		if !strings.HasPrefix(trimmed, "[") && net.ParseIP(trimmed) != nil && strings.Contains(trimmed, ":") {
			// IPv6 without brackets
			colonCount := strings.Count(trimmed, ":")
			lastColon := strings.LastIndex(trimmed, ":")
			if strings.Contains(trimmed, "::") && colonCount >= 5 && lastColon >= 0 {
				lastSegment := trimmed[lastColon+1:]
				if port, err := strconv.Atoi(lastSegment); err == nil && len(lastSegment) <= 5 && port > 0 && port <= 65535 {
					return WebPropertyID{}, fmt.Errorf("invalid webproperty: %q: ambiguous IPv6 address (use brackets if port intended)", raw)
				}
			}
			trimmed = fmt.Sprintf("[%s]:%d", trimmed, defaultPort)
			h, p, err = net.SplitHostPort(trimmed)
		} else if !strings.HasPrefix(trimmed, "[") {
			trimmed = fmt.Sprintf("%s:%d", trimmed, defaultPort)
			h, p, err = net.SplitHostPort(trimmed)
		}
		if err != nil {
			return WebPropertyID{}, fmt.Errorf("invalid webproperty: %q: %w", raw, err)
		}
	}

	host := strings.TrimSpace(h)
	if host == "" {
		return WebPropertyID{}, fmt.Errorf("invalid webproperty: %q: missing hostname", raw)
	}

	// If it looks like an IP, validate it's a valid IP
	if looksLikeIP(host) {
		if net.ParseIP(host) == nil {
			return WebPropertyID{}, fmt.Errorf("invalid webproperty: %q: invalid IP address", raw)
		}
	} else if !strings.Contains(host, ".") {
		// Hostnames that are not IPs must have a period
		return WebPropertyID{}, fmt.Errorf("invalid webproperty: %q: invalid hostname", raw)
	}

	if p == "" {
		p = strconv.Itoa(defaultPort)
	}

	p = strings.TrimSpace(p)
	port, err := strconv.Atoi(p)
	if err != nil || port <= 0 || port > 65535 {
		return WebPropertyID{}, fmt.Errorf("invalid port: %q: %w", p, err)
	}
	return WebPropertyID{Hostname: host, Port: port}, nil
}
