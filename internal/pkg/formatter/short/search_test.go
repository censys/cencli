package short

import (
	"strings"
	"testing"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/pkg/domain/assets"
)

func TestSearchHits(t *testing.T) {
	testCases := []struct {
		name           string
		hits           []assets.Asset
		expectedOutput string
	}{
		{
			name: "mixed asset types in order",
			hits: []assets.Asset{
				// Two hosts
				&assets.Host{
					Host: components.Host{
						IP: strPtr("1.1.1.1"),
					},
				},
				&assets.Host{
					Host: components.Host{
						IP: strPtr("2.2.2.2"),
					},
				},
				// One certificate
				&assets.Certificate{
					Certificate: components.Certificate{
						FingerprintSha256: strPtr("abc123"),
						Parsed: &components.CertificateParsed{
							Subject: &components.DistinguishedName{
								CommonName: []string{"test.com"},
							},
						},
					},
				},
				// One web property
				&assets.WebProperty{
					Webproperty: components.Webproperty{
						Hostname: strPtr("example.com"),
						Port:     intPtr(443),
					},
				},
			},
			expectedOutput: `
---------------------- Hit #1 (host) -----------------------
IP: 1.1.1.1
Platform URL: https://platform.censys.io/hosts/1.1.1.1

Services (0):

---------------------- Hit #2 (host) -----------------------
IP: 2.2.2.2
Platform URL: https://platform.censys.io/hosts/2.2.2.2

Services (0):

------------------- Hit #3 (certificate) -------------------
Certificate: abc123
Platform URL: https://platform.censys.io/certificates/abc123

------------------ Hit #4 (web property) -------------------
Hostname: example.com:443
Platform URL: https://platform.censys.io/web/example.com:443
`,
		},
		{
			name: "single asset type",
			hits: []assets.Asset{
				&assets.Host{
					Host: components.Host{
						IP: strPtr("3.3.3.3"),
					},
				},
			},
			expectedOutput: `
---------------------- Hit #1 (host) -----------------------
IP: 3.3.3.3
Platform URL: https://platform.censys.io/hosts/3.3.3.3

Services (0):
`,
		},
		{
			name:           "empty hits",
			hits:           []assets.Asset{},
			expectedOutput: ``,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			actual := SearchHits(tt.hits)
			actualTrimmed := strings.TrimSpace(actual)
			expectedTrimmed := strings.TrimSpace(tt.expectedOutput)
			require.Equal(t, expectedTrimmed, actualTrimmed)
		})
	}
}
