package short

import (
	"strings"
	"testing"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/pkg/domain/assets"
)

func TestHosts(t *testing.T) {
	testCases := []struct {
		name           string
		host           *assets.Host
		expectedOutput string
	}{
		{
			name: "rich host",
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("1.1.1.1"),
					AutonomousSystem: &components.Routing{
						Asn:  intPtr(13335),
						Name: strPtr("Cloudflare"),
					},
					Whois: &components.Whois{
						Organization: &components.Organization{
							Name: strPtr("Cloudflare"),
						},
					},
					Location: &components.Location{
						City:      strPtr("Mountain View"),
						Province:  strPtr("California"),
						Country:   strPtr("United States"),
						Continent: strPtr("US"),
						Coordinates: &components.Coordinates{
							Latitude:  floatPtr(37.4056),
							Longitude: floatPtr(-122.0775),
						},
					},
					DNS: &components.HostDNS{
						ReverseDNS: &components.HostDNSReverseResolution{
							Names: []string{
								"one.one.one.one",
								"1dot1dot1dot1.cloudflare-dns.com",
							},
						},
						ForwardDNS: map[string]components.HostDNSForwardResolution{
							"example.com": {},
							"example.org": {},
							"example.net": {},
							"a1.com":      {},
							"a2.com":      {},
							"a3.com":      {},
							"a4.com":      {},
							"a5.com":      {},
							"a6.com":      {},
							"a7.com":      {},
							"a8.com":      {},
							"a9.com":      {},
						},
					},
					OperatingSystem: &components.Attribute{
						Vendor:  strPtr("linux"),
						Product: strPtr("ubuntu"),
						Version: strPtr("22.04"),
					},
					Services: []components.Service{
						{
							Protocol:          strPtr("DNS"),
							Port:              intPtr(53),
							TransportProtocol: components.ServiceTransportProtocolUDP.ToPointer(),
						},
						{
							Protocol:          strPtr("UNKNOWN"),
							Port:              intPtr(443),
							TransportProtocol: components.ServiceTransportProtocolQuic.ToPointer(),
						},
						{
							Protocol:          strPtr("HTTP"),
							Port:              intPtr(443),
							TransportProtocol: components.ServiceTransportProtocolTCP.ToPointer(),
							Cert: &components.Certificate{
								Parsed: &components.CertificateParsed{
									SubjectDn: strPtr("CN=dns.google"),
									IssuerDn:  strPtr("C=US, O=Google Trust Services, CN=WR2"),
								},
							},
						},
						{
							Protocol:          strPtr("UNKNOWN"),
							Port:              intPtr(853),
							TransportProtocol: components.ServiceTransportProtocolTCP.ToPointer(),
							Cert: &components.Certificate{
								Parsed: &components.CertificateParsed{
									SubjectDn: strPtr("CN=dns.google"),
									IssuerDn:  strPtr("C=US, O=Google Trust Services, CN=WR2"),
								},
							},
						},
					},
				},
			},
			expectedOutput: `
------------------------- Host #1 --------------------------
IP: 1.1.1.1
Platform URL: https://platform.censys.io/hosts/1.1.1.1
ASN: 13335 (CLOUDFLARE)
WHOIS Org: Cloudflare
Location: Mountain View, California, United States (US)
Coordinates: 37.4056°, -122.0775°

Reverse DNS (2):
  - one.one.one.one
  - 1dot1dot1dot1.cloudflare-dns.com

Forward DNS (12):
  - a1.com
  - a2.com
  - a3.com
  - a4.com
  - a5.com
  - a6.com
  - a7.com
  - a8.com
  - a9.com
  - example.com
  ...

Operating System: Linux Ubuntu 22.04

Services (4):
  - DNS 53/udp

  - UNKNOWN 443/quic

  - HTTP 443/tcp
      Cert: Subject DN: CN=dns.google, Issuer DN: C=US, O=Google Trust Services, CN=WR2

  - UNKNOWN 853/tcp
      Cert: Subject DN: CN=dns.google, Issuer DN: C=US, O=Google Trust Services, CN=WR2
`,
		},
		{
			name: "minimal host",
			host: &assets.Host{
				Host: components.Host{
					IP: strPtr("2.2.2.2"),
				},
			},
			expectedOutput: `
------------------------- Host #1 --------------------------
IP: 2.2.2.2
Platform URL: https://platform.censys.io/hosts/2.2.2.2

Services (0):
`,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			actual := Hosts([]*assets.Host{tt.host})
			actualTrimmed := strings.TrimSpace(actual)
			expectedTrimmed := strings.TrimSpace(tt.expectedOutput)
			require.Equal(t, expectedTrimmed, actualTrimmed)
		})
	}
}

func strPtr(s string) *string     { return &s }
func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }
