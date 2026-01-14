package short

import (
	"fmt"
	"strings"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/pkg/censyscopy"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
)

// WebProperties renders web properties in short format
func WebProperties(webProperties []*assets.WebProperty) string {
	b := NewBlock()

	for i, wp := range webProperties {
		if i > 0 {
			b.Newline()
		}
		b.SeparatorWithLabel(fmt.Sprintf("Web Property #%d", i+1))
		b.Write(renderWebPropertyShort(wp))
	}

	return b.String()
}

// renderWebPropertyShort renders a single web property in short format
func renderWebPropertyShort(wp *assets.WebProperty) string {
	var out strings.Builder

	// Header
	out.WriteString(wpHeader(wp))
	out.WriteString("\n")

	// Certificate section (if present)
	if wp.Cert != nil {
		out.WriteString(wpCertificate(wp.Cert))
	}

	// Labels section (if present)
	if len(wp.Labels) > 0 {
		out.WriteString(wpLabels(wp.Labels))
	}

	// Software section (if present)
	if len(wp.Software) > 0 {
		out.WriteString(wpComponents("Software", wp.Software))
	}

	// Hardware section (if present)
	if len(wp.Hardware) > 0 {
		out.WriteString(wpComponents("Hardware", wp.Hardware))
	}

	// Endpoints section
	if len(wp.Endpoints) > 0 {
		out.WriteString(wpEndpoints(wp.Endpoints))
	}

	return out.String()
}

// wpHeader renders the header section (hostname:port and platform URL)
func wpHeader(wp *assets.WebProperty) string {
	hostname := Val(wp.Hostname, "")
	port := Val(wp.Port, 0)
	hostPort := fmt.Sprintf("%s:%d", hostname, port)
	link := censyscopy.CensysWebPropertyLookupLink(hostPort)

	line := NewLine(
		WithLabelStyle(styles.GlobalStyles.Signature),
	)

	if formatter.StdoutIsTTY() {
		// Make hostname:port an underlined clickable link
		underlinedHostPort := styles.GlobalStyles.Signature.Underline(true).Render(hostPort)
		line.Write("Hostname", link.Render(underlinedHostPort))
	} else {
		// Plain hostname:port with separate Platform URL line
		line.Write("Hostname", hostPort)
		line.Write("Platform URL", link.String())
	}

	return line.String()
}

// wpCertificate renders the certificate section
func wpCertificate(cert *components.Certificate) string {
	var out strings.Builder

	b := NewBlock()
	b.Title("Certificate:")

	b.Field("Fingerprint (SHA256)", Val(cert.FingerprintSha256, ""))

	if cert.Parsed != nil {
		b.Field("Issuer", Val(cert.Parsed.IssuerDn, ""))
		b.Field("Subject", Val(cert.Parsed.SubjectDn, ""))
	}

	if len(cert.Names) > 0 {
		b.Field("Names", strings.Join(cert.Names, ", "))
	}

	out.WriteString(b.String())

	return out.String()
}

// wpLabels renders the labels section
func wpLabels(labels []components.Label) string {
	labelValues := renderLabels(labels, ", ")
	if labelValues == "" {
		return ""
	}

	line := NewLine()
	line.Write("Labels", labelValues)
	return line.String()
}

// wpComponents renders software or hardware components
func wpComponents(name string, components []components.Attribute) string {
	rendered := renderComponentList(components)
	if rendered == "" {
		return ""
	}

	line := NewLine()
	line.Write(name, rendered)
	return line.String()
}

// wpEndpoints renders the endpoints section
func wpEndpoints(endpoints []components.EndpointScanState) string {
	b := NewBlock()

	// Header line
	line := NewLine()
	line.Write("Endpoints", fmt.Sprintf("(%d)", len(endpoints)))
	b.Write(line.String())

	// Render each endpoint as a list item
	for _, endpoint := range endpoints {
		path := Val(endpoint.Path, "")
		endpointType := Val(endpoint.EndpointType, "")

		// Start item with styled title
		title := fmt.Sprintf("%s (%s)",
			styles.GlobalStyles.Signature.Render(path),
			styles.GlobalStyles.Tertiary.Render(endpointType))
		b.Item(title)

		// Add fields for this endpoint
		b.ItemField("IP", Val(endpoint.IP, ""))
		b.ItemField("Type", Val(endpoint.EndpointType, ""))

		// HTTP details
		if endpoint.HTTP != nil {
			http := endpoint.HTTP

			// Status with optional redirect
			if http.StatusCode != nil {
				statusLine := fmt.Sprintf("%d", *http.StatusCode)
				if http.StatusReason != nil {
					statusLine += " " + *http.StatusReason
				}

				// Check for Location header (redirect)
				if http.Headers != nil {
					if location, ok := http.Headers["Location"]; ok && len(location.Headers) > 0 {
						statusLine += " → " + location.Headers[0]
					}
				}

				b.ItemField("Status", statusLine)
			}

			// Server header
			if http.Headers != nil {
				if server, ok := http.Headers["Server"]; ok && len(server.Headers) > 0 {
					b.ItemField("Server", server.Headers[0])
				}
			}

			// HTML Title
			b.ItemField("HTML Title", Val(http.HTMLTitle, ""))
		}

		// Scan time
		b.ItemField("Scan Time", Val(endpoint.ScanTime, ""))

		b.EndItem()
	}

	return b.String()
}

// renderComponentList renders a list of software/hardware attributes
// Mimics the legacy render_components helper logic.
func renderComponentList(attrs []components.Attribute) string {
	parts := make([]string, 0, len(attrs))

	for _, attr := range attrs {
		formatted := formatAttribute(attr)
		if formatted != "" {
			parts = append(parts, formatted)
		}
	}

	return strings.Join(parts, ", ")
}

// formatAttribute formats a single attribute (vendor, product, version).
// Mimics the legacy render_components helper logic.
func formatAttribute(attr components.Attribute) string {
	vendor := Val(attr.Vendor, "")
	product := Val(attr.Product, "")
	version := Val(attr.Version, "")

	// Skip if no vendor or product
	if vendor == "" && product == "" {
		return ""
	}

	var parts []string

	// Determine what to display
	switch {
	case vendor != "" && product != "":
		// If product starts with vendor, only show product
		if strings.HasPrefix(product, vendor) {
			parts = append(parts, formatComponentName(product))
		} else {
			parts = append(parts, formatComponentName(vendor))
			parts = append(parts, formatComponentName(product))
		}
	case vendor != "":
		parts = append(parts, formatComponentName(vendor))
	case product != "":
		parts = append(parts, formatComponentName(product))
	}

	// Add version at the end
	if version != "" {
		parts = append(parts, version)
	}

	return strings.Join(parts, " ")
}

// formatComponentName formats a component name by replacing underscores with spaces
// and capitalizing the first letter of each word.
func formatComponentName(name string) string {
	if name == "" {
		return ""
	}

	// Replace underscores with spaces
	name = strings.ReplaceAll(name, "_", " ")

	// Split into words and capitalize each
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

// renderLabels extracts label values and joins them with a delimiter.
func renderLabels(labels []components.Label, delimiter string) string {
	values := make([]string, 0, len(labels))
	for _, label := range labels {
		if label.Value != nil {
			values = append(values, *label.Value)
		}
	}
	return strings.Join(values, delimiter)
}
