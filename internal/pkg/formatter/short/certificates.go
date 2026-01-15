package short

import (
	"fmt"
	"strings"
	"time"

	"github.com/censys/cencli/internal/pkg/censyscopy"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
)

// Certificates renders certificates in short format
func Certificates(certificates []*assets.Certificate) string {
	b := NewBlock()

	for i, cert := range certificates {
		if i > 0 {
			b.Newline()
		}
		b.SeparatorWithLabel(fmt.Sprintf("Certificate #%d", i+1))
		b.Write(renderCertificateShort(cert))
	}

	return b.String()
}

// renderCertificateShort renders a single certificate
func renderCertificateShort(cert *assets.Certificate) string {
	var out strings.Builder

	// Header
	out.WriteString(certHeader(cert))

	// Issuer and Subject DN
	out.WriteString(certIssuer(cert))

	// Validity
	out.WriteString(certValidity(cert))

	// TODO: Add labels section

	// Subject Alternative Names
	out.WriteString(certSubjectAltNames(cert))

	// Validation Level
	out.WriteString(certMetadata(cert))

	return out.String()
}

// certHeader renders the certificate SHA256 fingerprint as the main header
func certHeader(cert *assets.Certificate) string {
	fingerprint := Val(cert.FingerprintSha256, "")
	link := censyscopy.CensysCertificateLookupLink(fingerprint)

	line := NewLine(
		WithLabelStyle(styles.GlobalStyles.Signature),
	)

	if formatter.StdoutIsTTY() {
		// Make fingerprint an underlined clickable link
		underlinedFP := styles.GlobalStyles.Signature.Underline(true).Render(fingerprint)
		line.Write("Certificate", link.Render(underlinedFP))
	} else {
		// Plain fingerprint with separate Platform URL line
		line.Write("Certificate", fingerprint)
		line.Write("Platform URL", link.String())
	}

	return line.String()
}

// certIssuer renders the issuer information
func certIssuer(cert *assets.Certificate) string {
	if cert.Parsed == nil || cert.Parsed.Issuer == nil {
		return ""
	}

	issuerLine := NewLine(WithLabelStyle(styles.GlobalStyles.Primary))
	issuerLine.Newline()
	issuerLine.Write("Issuer DN", Val(cert.Parsed.IssuerDn, ""))
	subjectLine := NewLine()
	subjectLine.Write("Subject DN", Val(cert.Parsed.SubjectDn, ""))
	return issuerLine.String() + subjectLine.String()
}

// certValidity renders the validity period
func certValidity(cert *assets.Certificate) string {
	if cert.Parsed == nil || cert.Parsed.ValidityPeriod == nil {
		return ""
	}

	notBefore := Val(cert.Parsed.ValidityPeriod.NotBefore, "")
	notAfter := Val(cert.Parsed.ValidityPeriod.NotAfter, "")

	if notBefore == "" && notAfter == "" {
		return ""
	}

	// Format dates nicely
	notBeforeFormatted := formatCertDate(notBefore)
	notAfterFormatted := formatCertDate(notAfter)

	validityStr := fmt.Sprintf("%s → %s", notBeforeFormatted, notAfterFormatted)
	line := NewLine()
	line.Write("Validity", validityStr)
	return line.String()
}

// formatCertDate formats a certificate date string into a human-readable format
func formatCertDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// Try parsing common certificate date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05 UTC",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format("Jan 02, 2006")
		}
	}

	// Return original if parsing fails
	return dateStr
}

// certSubjectAltNames renders the subject alternative names section
func certSubjectAltNames(cert *assets.Certificate) string {
	if cert.Parsed == nil || cert.Parsed.Extensions == nil || cert.Parsed.Extensions.SubjectAltName == nil {
		return ""
	}

	dnsNames := cert.Parsed.Extensions.SubjectAltName.DNSNames
	if len(dnsNames) == 0 {
		return ""
	}

	var out strings.Builder
	out.WriteString("\nSubject Alternative Names:\n")

	for _, name := range dnsNames {
		out.WriteString(fmt.Sprintf("  - %s\n", styles.GlobalStyles.Tertiary.Render(name)))
	}

	return out.String()
}

// certMetadata renders validation level and parse status
func certMetadata(cert *assets.Certificate) string {
	var out strings.Builder

	if cert.ValidationLevel != nil {
		out.WriteString("\n")
		line := NewLine()
		line.Write("Validation Level", string(*cert.ValidationLevel))
		out.WriteString(line.String())
	}

	return out.String()
}
