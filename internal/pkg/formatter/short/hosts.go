package short

import (
	"fmt"
	"sort"
	"strings"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/pkg/censyscopy"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/styles"
)

// Hosts renders hosts in short format.
func Hosts(hosts []*assets.Host) string {
	b := NewBlock()

	for i, host := range hosts {
		if i > 0 {
			b.Newline()
		}
		b.SeparatorWithLabel(fmt.Sprintf("Host #%d", i+1))
		b.Write(renderHostShort(host))
	}

	return b.String()
}

// renderHostShort renders a single host.
func renderHostShort(host *assets.Host) string {
	var out strings.Builder

	// Header lines
	out.WriteString(hostHeader(host))

	// ASN / WHOIS / Location
	out.WriteString(hostMetadata(host))

	// Reverse / Forward DNS
	out.WriteString(hostDNS(host))

	// Operating system
	if os := host.OperatingSystem; os != nil {
		out.WriteString(fmt.Sprintf("\nOperating System: %s\n", formatAttribute(*os)))
	}

	// Services
	out.WriteString(renderServices(host.Services))

	return out.String()
}

// hostHeader renders IP and platform link.
func hostHeader(host *assets.Host) string {
	ip := Val(host.IP, "")

	// IP line (no clickable in short output)
	line := NewLine(WithLabelStyle(styles.GlobalStyles.Primary))
	line.Write("IP", ip)

	// Platform URL (plain)
	link := censyscopy.CensysHostLookupLink(ip).String()
	link = strings.TrimPrefix(link, "https://")

	platform := NewLine(WithLabelStyle(styles.GlobalStyles.Primary))
	platform.Write("Platform URL", link)

	return line.String() + platform.String()
}

// hostMetadata renders ASN, WHOIS org, and location.
func hostMetadata(host *assets.Host) string {
	var out strings.Builder

	// ASN
	if as := host.AutonomousSystem; as != nil {
		asn := Val(as.Asn, 0)
		asName := strings.ToUpper(Val(as.Name, ""))
		if asn != 0 || asName != "" {
			line := NewLine()
			line.Write("ASN", fmt.Sprintf("%d (%s)", asn, asName))
			out.WriteString(line.String())
		}
	}

	// WHOIS Org
	if host.Whois != nil && host.Whois.Organization != nil {
		org := Val(host.Whois.Organization.Name, "")
		if org != "" {
			line := NewLine()
			line.Write("WHOIS Org", org)
			out.WriteString(line.String())
		}
	}

	// Location
	if host.Location != nil {
		loc := host.Location
		var city, region, country, code string
		city = Val(loc.City, "")
		region = Val(loc.Province, "")
		country = Val(loc.Country, "")
		code = Val(loc.Continent, "")

		// If continent looks like a short code (e.g., US), use that; otherwise use country code if present.
		countryDisplay := country
		if code != "" {
			countryDisplay = fmt.Sprintf("%s (%s)", countryDisplay, code)
		}

		parts := make([]string, 0, 3)
		if city != "" {
			parts = append(parts, city)
		}
		if region != "" {
			parts = append(parts, region)
		}
		if countryDisplay != "" {
			parts = append(parts, countryDisplay)
		}

		if len(parts) > 0 {
			line := NewLine()
			line.Write("Location", strings.Join(parts, ", "))
			out.WriteString(line.String())
		}
		if loc.Coordinates != nil {
			lat := Val(loc.Coordinates.Latitude, 0.0)
			lon := Val(loc.Coordinates.Longitude, 0.0)
			line := NewLine()
			line.Write("Coordinates", fmt.Sprintf("%.4f°, %.4f°", lat, lon))
			out.WriteString(line.String())
		}
	}

	return out.String()
}

// hostDNS renders reverse and forward DNS sections (limited to 10 entries with ellipsis).
func hostDNS(host *assets.Host) string {
	if host.DNS == nil {
		return ""
	}

	var out strings.Builder

	// Reverse DNS
	if host.DNS.ReverseDNS != nil && len(host.DNS.ReverseDNS.Names) > 0 {
		names := host.DNS.ReverseDNS.Names
		out.WriteString(fmt.Sprintf("\nReverse DNS (%d):\n", len(names)))
		out.WriteString(renderListWithLimit(names, 10))
	}

	// Forward DNS (keys of map) - sorted for deterministic output
	if len(host.DNS.ForwardDNS) > 0 {
		keys := make([]string, 0, len(host.DNS.ForwardDNS))
		for k := range host.DNS.ForwardDNS {
			keys = append(keys, k)
		}
		// Sort for stable output
		sort.Strings(keys)

		out.WriteString(fmt.Sprintf("\nForward DNS (%d):\n", len(keys)))
		out.WriteString(renderListWithLimit(keys, 10))
	}

	return out.String()
}

// renderListWithLimit renders up to limit items, appending "..." if more remain.
func renderListWithLimit(items []string, limit int) string {
	var out strings.Builder
	for i, v := range items {
		if i >= limit {
			out.WriteString("  ...\n")
			break
		}
		out.WriteString(fmt.Sprintf("  - %s\n", styles.GlobalStyles.Tertiary.Render(v)))
	}
	return out.String()
}

// renderServices renders services section.
func renderServices(services []components.Service) string {
	var out strings.Builder

	count := len(services)
	out.WriteString(fmt.Sprintf("\nServices (%d):\n", count))
	if count == 0 {
		return out.String()
	}

	b := NewBlock()
	for _, svc := range services {
		proto := strings.ToUpper(Val(svc.Protocol, ""))
		if proto == "" {
			proto = "UNKNOWN"
		}
		port := Val(svc.Port, 0)
		transport := string(Val(svc.TransportProtocol, components.ServiceTransportProtocol("")))
		title := fmt.Sprintf("%s %d/%s", styles.GlobalStyles.Signature.Render(proto), port, transport)
		b.Item(title)

		// TLS cert summary from service cert
		if svc.Cert != nil && svc.Cert.Parsed != nil {
			subj := Val(svc.Cert.Parsed.SubjectDn, "")
			iss := Val(svc.Cert.Parsed.IssuerDn, "")
			if subj != "" || iss != "" {
				var certParts []string
				if subj != "" {
					certParts = append(certParts, fmt.Sprintf("Subject DN: %s", subj))
				}
				if iss != "" {
					certParts = append(certParts, fmt.Sprintf("Issuer DN: %s", iss))
				}
				b.ItemField("Cert", strings.Join(certParts, ", "))
			}
		}

		b.EndItem()
	}

	out.WriteString(b.String())
	return out.String()
}
