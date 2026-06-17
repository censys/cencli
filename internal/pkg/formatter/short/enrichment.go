package short

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/pkg/censyscopy"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
)

// EnrichedHosts renders enriched hosts in short format.
func EnrichedHosts(hosts []*assets.EnrichedHost) string {
	b := NewBlock()

	for i, host := range hosts {
		if i > 0 {
			b.Newline()
		}
		b.SeparatorWithLabel(fmt.Sprintf("Host #%d", i+1))
		b.Write(renderEnrichedHostShort(host))
	}

	return b.String()
}

// renderEnrichedHostShort renders a single enriched host.
func renderEnrichedHostShort(host *assets.EnrichedHost) string {
	var out strings.Builder

	out.WriteString(enrichmentHeader(host))
	out.WriteString(enrichmentMetadata(host))
	out.WriteString(hostLabels(host.Labels))
	out.WriteString(enrichmentDNS(host.DNS))
	out.WriteString(enrichmentReputation(host.Greynoise, host.Reputation))
	out.WriteString(enrichmentClassifications(host.Network, host.Privacy))
	out.WriteString(enrichmentServices(host.Services, host.ServiceCount))
	out.WriteString(enrichmentMallory(host.ThirdParty))

	return out.String()
}

// enrichmentHeader renders IP and platform link.
func enrichmentHeader(host *assets.EnrichedHost) string {
	ip := Val(host.IP, "")
	link := censyscopy.CensysHostLookupLink(ip)

	line := NewLine(WithLabelStyle(styles.GlobalStyles.Signature))

	if formatter.StdoutIsTTY() {
		underlinedIP := styles.GlobalStyles.Signature.Underline(true).Render(ip)
		line.Write("IP", link.Render(underlinedIP))
	} else {
		line.Write("IP", ip)
		line.Write("Platform URL", link.String())
	}

	return line.String()
}

// enrichmentMetadata renders ASN, WHOIS org, and location.
func enrichmentMetadata(host *assets.EnrichedHost) string {
	var out strings.Builder

	// Autonomous system
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
		if org := Val(host.Whois.Organization.Name, ""); org != "" {
			line := NewLine()
			line.Write("WHOIS Org", org)
			out.WriteString(line.String())
		}
	}

	// Location
	if loc := host.Location; loc != nil {
		parts := make([]string, 0, 3)
		if city := Val(loc.City, ""); city != "" {
			parts = append(parts, city)
		}
		if region := Val(loc.Province, ""); region != "" {
			parts = append(parts, region)
		}
		country := Val(loc.Country, "")
		if code := Val(loc.CountryCode, ""); code != "" {
			country = strings.TrimSpace(fmt.Sprintf("%s (%s)", country, code))
		}
		if country != "" {
			parts = append(parts, country)
		}
		if len(parts) > 0 {
			line := NewLine()
			line.Write("Location", strings.Join(parts, ", "))
			out.WriteString(line.String())
		}
	}

	return out.String()
}

// enrichmentDNS renders reverse and forward DNS sections (limited to 10 entries).
func enrichmentDNS(dns *components.HostDNS) string {
	if dns == nil {
		return ""
	}

	var out strings.Builder

	if dns.ReverseDNS != nil && len(dns.ReverseDNS.Names) > 0 {
		names := dns.ReverseDNS.Names
		out.WriteString(fmt.Sprintf("\nReverse DNS (%d):\n", len(names)))
		out.WriteString(renderListWithLimit(names, 10))
	}

	if len(dns.ForwardDNS) > 0 {
		keys := make([]string, 0, len(dns.ForwardDNS))
		for k := range dns.ForwardDNS {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out.WriteString(fmt.Sprintf("\nForward DNS (%d):\n", len(keys)))
		out.WriteString(renderListWithLimit(keys, 10))
	}

	return out.String()
}

// enrichmentReputation renders greynoise and reputation signals.
func enrichmentReputation(gn *components.Greynoise, rep *components.Reputation) string {
	var out strings.Builder

	if rep != nil {
		level := string(Val(rep.ScoreLevel, components.ScoreLevel("")))
		if level != "" {
			line := NewLine()
			if rep.Score != nil {
				line.Write("Reputation", fmt.Sprintf("%s (score %.2f)", level, *rep.Score))
			} else {
				line.Write("Reputation", level)
			}
			out.WriteString(line.String())
		}
	}

	if gn != nil {
		classification := Val(gn.Classification, "")
		actor := Val(gn.Actor, "")
		if classification != "" || actor != "" {
			parts := make([]string, 0, 2)
			if classification != "" {
				parts = append(parts, classification)
			}
			if actor != "" {
				parts = append(parts, fmt.Sprintf("actor: %s", actor))
			}
			line := NewLine()
			line.Write("GreyNoise", strings.Join(parts, ", "))
			out.WriteString(line.String())
		}
	}

	return out.String()
}

// enrichmentClassifications renders network and privacy classification flags.
func enrichmentClassifications(network []components.NetworkClassification, privacy []components.Privacy) string {
	var out strings.Builder

	netFlags := make([]string, 0, 3)
	for _, n := range network {
		if Val(n.Hosting, false) {
			netFlags = appendUnique(netFlags, "hosting")
		}
		if Val(n.Mobile, false) {
			netFlags = appendUnique(netFlags, "mobile")
		}
		if Val(n.Satellite, false) {
			netFlags = appendUnique(netFlags, "satellite")
		}
	}
	if len(netFlags) > 0 {
		line := NewLine()
		line.Write("Network", strings.Join(netFlags, ", "))
		out.WriteString(line.String())
	}

	privFlags := make([]string, 0, 5)
	for _, p := range privacy {
		if Val(p.Tor, false) {
			privFlags = appendUnique(privFlags, "tor")
		}
		if Val(p.Vpn, false) {
			privFlags = appendUnique(privFlags, "vpn")
		}
		if Val(p.Proxy, false) {
			privFlags = appendUnique(privFlags, "proxy")
		}
		if Val(p.Relay, false) {
			privFlags = appendUnique(privFlags, "relay")
		}
		if Val(p.Anonymous, false) {
			privFlags = appendUnique(privFlags, "anonymous")
		}
	}
	if len(privFlags) > 0 {
		line := NewLine()
		line.Write("Privacy", strings.Join(privFlags, ", "))
		out.WriteString(line.String())
	}

	return out.String()
}

// enrichmentServices renders the trimmed service list.
func enrichmentServices(services []components.HostEnrichmentService, serviceCount *int) string {
	var out strings.Builder

	count := Val(serviceCount, len(services))
	out.WriteString(fmt.Sprintf("\nServices (%d):\n", count))
	if len(services) == 0 {
		return out.String()
	}

	b := NewBlock()
	for _, svc := range services {
		proto := strings.ToUpper(Val(svc.Protocol, ""))
		if proto == "" {
			proto = "UNKNOWN"
		}
		port := Val(svc.Port, 0)
		title := fmt.Sprintf("%s %d", styles.GlobalStyles.Signature.Render(proto), port)
		b.Item(title)

		if labels := labelValues(svc.Labels); labels != "" {
			b.ItemField("Labels", labels)
		}
		if threats := threatValues(svc.Threats); threats != "" {
			b.ItemField("Threats", threats)
		}

		b.EndItem()
	}

	out.WriteString(b.String())
	return out.String()
}

// enrichmentMallory renders a summary of MalloryAI verdicts.
func enrichmentMallory(tp *components.ThirdParty) string {
	if tp == nil || len(tp.Mallory) == 0 {
		return ""
	}

	var malicious, suspicious int64
	for _, m := range tp.Mallory {
		if m.VerdictSummary != nil {
			malicious += Val(m.VerdictSummary.Malicious, 0)
			suspicious += Val(m.VerdictSummary.Suspicious, 0)
		}
	}

	line := NewLine()
	line.Write("MalloryAI", fmt.Sprintf("%d record(s), %d malicious / %d suspicious verdict(s)", len(tp.Mallory), malicious, suspicious))
	return line.String()
}

func labelValues(labels []components.Label) string {
	values := make([]string, 0, len(labels))
	for _, label := range labels {
		if label.Value != nil {
			values = append(values, *label.Value)
		}
	}
	return strings.Join(values, ", ")
}

func threatValues(threats []components.Threat) string {
	values := make([]string, 0, len(threats))
	for _, threat := range threats {
		if name := Val(threat.Name, ""); name != "" {
			values = append(values, name)
		}
	}
	return strings.Join(values, ", ")
}

func appendUnique(items []string, value string) []string {
	if slices.Contains(items, value) {
		return items
	}
	return append(items, value)
}
