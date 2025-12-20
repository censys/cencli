package short

import (
	"strings"
	"testing"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/pkg/domain/assets"
)

func TestCertificates(t *testing.T) {
	testCases := []struct {
		name           string
		certificate    *assets.Certificate
		expectedOutput string
	}{
		{
			name: "rich certificate",
			certificate: &assets.Certificate{
				Certificate: components.Certificate{
					FingerprintSha256: strPtr("abc123def456"),
					FingerprintSha1:   strPtr("abc123"),
					FingerprintMd5:    strPtr("def456"),
					ParseStatus:       components.ParseStatusSuccess.ToPointer(),
					ValidationLevel:   components.ValidationLevelDv.ToPointer(),
					Parsed: &components.CertificateParsed{
						Subject: &components.DistinguishedName{
							CommonName:   []string{"example.com"},
							Organization: []string{"Example Inc"},
						},
						Issuer: &components.DistinguishedName{
							CommonName:   []string{"Let's Encrypt Authority X3"},
							Organization: []string{"Let's Encrypt"},
						},
						ValidityPeriod: &components.ValidityPeriod{
							NotBefore: strPtr("2024-01-01T00:00:00Z"),
							NotAfter:  strPtr("2024-12-31T23:59:59Z"),
						},
						Extensions: &components.CertificateExtensions{
							SubjectAltName: &components.GeneralNames{
								DNSNames: []string{
									"example.com",
									"www.example.com",
									"api.example.com",
								},
							},
						},
					},
					Ct: &components.Ct{
						Entries: map[string]components.CtRecord{
							"google_xenon2023": {
								AddedToCtAt: strPtr("2024-01-01T12:00:00Z"),
							},
							"cloudflare_nimbus2023": {
								AddedToCtAt: strPtr("2024-01-01T12:05:00Z"),
							},
						},
					},
				},
			},
			expectedOutput: `
---------------------- Certificate #1 ----------------------
Certificate for: example.com
Issuer: Let's Encrypt (Let's Encrypt Authority X3)
Validity: 2024-01-01T00:00:00Z → 2024-12-31T23:59:59Z

Fingerprints:
  SHA256: abc123def456
  SHA1: abc123
  MD5: def456

Subject Alternative Names:
  - example.com
  - www.example.com
  - api.example.com

CT Log Entries:
  - cloudflare_nimbus2023 |  Added: 2024-01-01T12:05:00Z
  - google_xenon2023 |  Added: 2024-01-01T12:00:00Z

Validation Level: dv
Parse Status: success
`,
		},
		{
			name: "minimal certificate",
			certificate: &assets.Certificate{
				Certificate: components.Certificate{
					FingerprintSha256: strPtr("minimal123"),
					Parsed: &components.CertificateParsed{
						Subject: &components.DistinguishedName{
							CommonName: []string{"minimal.com"},
						},
					},
				},
			},
			expectedOutput: `
---------------------- Certificate #1 ----------------------
Certificate for: minimal.com

Fingerprints:
  SHA256: minimal123
`,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			actual := Certificates([]*assets.Certificate{tt.certificate})
			actualTrimmed := strings.TrimSpace(actual)
			expectedTrimmed := strings.TrimSpace(tt.expectedOutput)
			require.Equal(t, expectedTrimmed, actualTrimmed)
		})
	}
}
