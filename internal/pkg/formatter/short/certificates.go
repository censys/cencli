package short

import (
	"fmt"
	"sort"
	"strings"

	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/styles"
)

// FIXME: make this perfect

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

	// Issuer
	out.WriteString(certIssuer(cert))

	// Validity
	out.WriteString(certValidity(cert))

	// Fingerprints
	out.WriteString(certFingerprints(cert))

	// Subject Alternative Names
	out.WriteString(certSubjectAltNames(cert))

	// CT Log Entries
	out.WriteString(certCTLogEntries(cert))

	// Validation Level and Parse Status
	out.WriteString(certMetadata(cert))

	return out.String()
}

// certHeader renders the certificate subject common name
func certHeader(cert *assets.Certificate) string {
	var subjectCN string
	if cert.Parsed != nil && cert.Parsed.Subject != nil && len(cert.Parsed.Subject.CommonName) > 0 {
		subjectCN = cert.Parsed.Subject.CommonName[0]
	}

	line := NewLine(
		WithLabelStyle(styles.GlobalStyles.Signature),
	)
	line.Write("Certificate for", subjectCN)
	return line.String()
}

// certIssuer renders the issuer information
func certIssuer(cert *assets.Certificate) string {
	if cert.Parsed == nil || cert.Parsed.Issuer == nil {
		return ""
	}

	var org, cn string
	if len(cert.Parsed.Issuer.Organization) > 0 {
		org = cert.Parsed.Issuer.Organization[0]
	}
	if len(cert.Parsed.Issuer.CommonName) > 0 {
		cn = cert.Parsed.Issuer.CommonName[0]
	}

	if org == "" && cn == "" {
		return ""
	}

	issuerStr := org
	if cn != "" {
		if org != "" {
			issuerStr = fmt.Sprintf("%s (%s)", org, cn)
		} else {
			issuerStr = cn
		}
	}

	line := NewLine()
	line.Write("Issuer", issuerStr)
	return line.String()
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

	validityStr := fmt.Sprintf("%s → %s", notBefore, notAfter)
	line := NewLine()
	line.Write("Validity", validityStr)
	return line.String()
}

// certFingerprints renders the fingerprints section
func certFingerprints(cert *assets.Certificate) string {
	var out strings.Builder
	out.WriteString("\nFingerprints:\n")

	b := NewBlock(WithValueStyle(styles.GlobalStyles.Warning))
	b.Field("SHA256", Val(cert.FingerprintSha256, ""))
	b.Field("SHA1", Val(cert.FingerprintSha1, ""))
	b.Field("MD5", Val(cert.FingerprintMd5, ""))

	out.WriteString(b.String())
	return out.String()
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

// certCTLogEntries renders the CT log entries section
func certCTLogEntries(cert *assets.Certificate) string {
	if cert.Ct == nil || len(cert.Ct.Entries) == 0 {
		return ""
	}

	var out strings.Builder
	out.WriteString("\nCT Log Entries:\n")

	// Sort keys for deterministic output
	keys := make([]string, 0, len(cert.Ct.Entries))
	for k := range cert.Ct.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, logName := range keys {
		entry := cert.Ct.Entries[logName]
		addedAt := Val(entry.AddedToCtAt, "")
		out.WriteString(fmt.Sprintf("  - %s |  Added: %s\n",
			styles.GlobalStyles.Tertiary.Render(logName),
			styles.GlobalStyles.Tertiary.Render(addedAt)))
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

	if cert.ParseStatus != nil {
		line := NewLine()
		line.Write("Parse Status", string(*cert.ParseStatus))
		out.WriteString(line.String())
	}

	return out.String()
}
