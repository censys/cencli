package short

import (
	"strings"
	"testing"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/pkg/domain/assets"
)

func TestWebProperties(t *testing.T) {
	testCases := []struct {
		webProperty    *assets.WebProperty
		name           string
		expectedOutput string
	}{
		{
			name: "web property",
			webProperty: &assets.WebProperty{
				Webproperty: components.Webproperty{
					Hostname: strPtr("example.com"),
					Port:     intPtr(443),
					Cert: &components.Certificate{
						FingerprintSha256: strPtr("ABC123"),
						Names:             []string{"example.com", "www.example.com"},
						Parsed: &components.CertificateParsed{
							IssuerDn:  strPtr("CN=Test Issuer"),
							SubjectDn: strPtr("CN=example.com"),
						},
					},
					Labels: []components.Label{
						{Value: strPtr("production")},
						{Value: strPtr("external")},
					},
					Software: []components.Attribute{
						{
							Vendor:  strPtr("nginx"),
							Product: strPtr("nginx_webserver"),
							Version: strPtr("1.2.3"),
						},
					},
					Hardware: []components.Attribute{
						{
							Vendor:  strPtr("supermicro"),
							Product: strPtr("x10dri"),
							Version: strPtr("rev2"),
						},
					},
					Endpoints: []components.EndpointScanState{
						{
							Path:         strPtr("/"),
							EndpointType: strPtr("HTTP"),
							IP:           strPtr("203.0.113.10"),
							ScanTime:     strPtr("2025-11-14T02:49:38Z"),

							HTTP: &components.HTTP{
								StatusCode:   intPtr(403),
								StatusReason: strPtr("Forbidden"),
								HTMLTitle:    strPtr("Just a moment..."),
								Headers: map[string]components.HTTPRepeatedHeaders{
									"Server": {
										Headers: []string{"cloudflare"},
									},
									"Location": {
										Headers: []string{"https://example.com/redirect"},
									},
								},
							},
						},
					},
				},
			},
			expectedOutput: `
--------------------- Web Property #1 ----------------------
Hostname: example.com:443
Platform URL: https://platform.censys.io/web/example.com:443

Certificate:
  Fingerprint (SHA256): ABC123
  Issuer: CN=Test Issuer
  Subject: CN=example.com
  Names: example.com, www.example.com

Labels: production, external

Software: Nginx Webserver 1.2.3

Hardware: Supermicro X10dri rev2

Endpoints: (1)
  - / (HTTP)
      IP: 203.0.113.10
      Type: HTTP
      Status: 403 Forbidden → https://example.com/redirect
      Server: cloudflare
      HTML Title: Just a moment...
      Scan Time: 2025-11-14T02:49:38Z
			`,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			actual := WebProperties([]*assets.WebProperty{tt.webProperty})
			actualTrimmed := strings.TrimSpace(actual)
			expectedTrimmed := strings.TrimSpace(tt.expectedOutput)
			require.Equal(t, expectedTrimmed, actualTrimmed)
		})
	}
}
