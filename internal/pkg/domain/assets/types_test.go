package assets

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

func TestAssetClassifier_ComprehensiveWorkflow(t *testing.T) {
	testCases := []struct {
		name               string
		rawAssets          []string
		expectedAssetType  AssetType
		expectedError      cenclierrors.CencliError
		expectedHostIDs    []string
		expectedCertIDs    []string
		expectedWebPropIDs []string
		expectedUnknowns   []string
	}{
		// Single asset type scenarios - Hosts
		{
			name:              "single IPv4 host",
			rawAssets:         []string{"192.168.1.100"},
			expectedAssetType: AssetTypeHost,
			expectedError:     nil,
			expectedHostIDs:   []string{"192.168.1.100"},
		},
		{
			name:              "multiple IPv4 hosts",
			rawAssets:         []string{"10.0.0.1", "172.16.0.1", "203.0.113.42"},
			expectedAssetType: AssetTypeHost,
			expectedError:     nil,
			expectedHostIDs:   []string{"10.0.0.1", "172.16.0.1", "203.0.113.42"},
		},
		{
			name:              "single IPv6 host",
			rawAssets:         []string{"2001:db8:85a3::8a2e:370:7334"},
			expectedAssetType: AssetTypeHost,
			expectedError:     nil,
			expectedHostIDs:   []string{"2001:db8:85a3::8a2e:370:7334"},
		},
		{
			name:              "mixed IPv4 and IPv6 hosts",
			rawAssets:         []string{"127.0.0.1", "::1", "fe80::1"},
			expectedAssetType: AssetTypeHost,
			expectedError:     nil,
			expectedHostIDs:   []string{"127.0.0.1", "::1", "fe80::1"},
		},

		// Single asset type scenarios - Certificates
		{
			name:              "single certificate fingerprint lowercase",
			rawAssets:         []string{"a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456"},
			expectedAssetType: AssetTypeCertificate,
			expectedError:     nil,
			expectedCertIDs:   []string{"a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456"},
		},
		{
			name:              "single certificate fingerprint uppercase",
			rawAssets:         []string{"A1B2C3D4E5F6789012345678901234567890ABCDEF1234567890ABCDEF123456"},
			expectedAssetType: AssetTypeCertificate,
			expectedError:     nil,
			expectedCertIDs:   []string{"A1B2C3D4E5F6789012345678901234567890ABCDEF1234567890ABCDEF123456"},
		},
		{
			name:              "multiple certificate fingerprints",
			rawAssets:         []string{"deadbeefcafebabe1234567890abcdef1234567890abcdef1234567890abcdef", "feedfacedeadbeef1234567890abcdef1234567890abcdef1234567890abcdef"},
			expectedAssetType: AssetTypeCertificate,
			expectedError:     nil,
			expectedCertIDs:   []string{"deadbeefcafebabe1234567890abcdef1234567890abcdef1234567890abcdef", "feedfacedeadbeef1234567890abcdef1234567890abcdef1234567890abcdef"},
		},

		// Single asset type scenarios - Web Properties
		{
			name:               "single web property with hostname",
			rawAssets:          []string{"api.example.com:443"},
			expectedAssetType:  AssetTypeWebProperty,
			expectedError:      nil,
			expectedWebPropIDs: []string{"api.example.com:443"},
		},
		{
			name:               "single web property with IPv4",
			rawAssets:          []string{"192.168.1.1:8080"},
			expectedAssetType:  AssetTypeWebProperty,
			expectedError:      nil,
			expectedWebPropIDs: []string{"192.168.1.1:8080"},
		},
		{
			name:               "single web property with IPv6",
			rawAssets:          []string{"[2001:db8::1]:9000"},
			expectedAssetType:  AssetTypeWebProperty,
			expectedError:      nil,
			expectedWebPropIDs: []string{"2001:db8::1:9000"},
		},
		{
			name:               "multiple web properties",
			rawAssets:          []string{"search.censys.io:443", "app.censys.io:80", "cdn.example.org:8443"},
			expectedAssetType:  AssetTypeWebProperty,
			expectedError:      nil,
			expectedWebPropIDs: []string{"search.censys.io:443", "app.censys.io:80", "cdn.example.org:8443"},
		},

		// Error scenarios - Mixed asset types
		{
			name:              "mixed hosts and certificates",
			rawAssets:         []string{"8.8.8.8", "deadbeefcafebabe1234567890abcdef1234567890abcdef1234567890abcdef"},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewMixedAssetTypesError(AssetTypeHost, AssetTypeCertificate),
		},
		{
			name:              "mixed hosts and web properties",
			rawAssets:         []string{"1.1.1.1", "example.com:443"},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewMixedAssetTypesError(AssetTypeHost, AssetTypeWebProperty),
		},
		{
			name:              "mixed certificates and web properties",
			rawAssets:         []string{"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "test.example.com:8080"},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewMixedAssetTypesError(AssetTypeCertificate, AssetTypeWebProperty),
		},
		{
			name:              "all three asset types mixed",
			rawAssets:         []string{"10.0.0.1", "feedface1234567890abcdef1234567890abcdef1234567890abcdef12345678", "api.test.com:443"},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewMixedAssetTypesError(AssetTypeHost, AssetTypeCertificate),
		},

		// Error scenarios - Invalid assets
		{
			name:              "single invalid asset",
			rawAssets:         []string{"not-a-valid-asset"},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewInvalidAssetIDError("not-a-valid-asset", "unable to infer asset type"),
			expectedUnknowns:  []string{"not-a-valid-asset"},
		},
		{
			name:              "invalid IP address",
			rawAssets:         []string{"999.999.999.999"},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewInvalidAssetIDError("999.999.999.999", "unable to infer asset type"),
			expectedUnknowns:  []string{"999.999.999.999"},
		},
		{
			name:              "invalid certificate fingerprint - too short",
			rawAssets:         []string{"abcdef123456"},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewInvalidAssetIDError("abcdef123456", "unable to infer asset type"),
			expectedUnknowns:  []string{"abcdef123456"},
		},
		{
			name:              "invalid certificate fingerprint - non-hex",
			rawAssets:         []string{"gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg"},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewInvalidAssetIDError("gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg", "unable to infer asset type"),
			expectedUnknowns:  []string{"gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg"},
		},
		{
			name:               "invalid web property - no port",
			rawAssets:          []string{"example.com"},
			expectedAssetType:  AssetTypeWebProperty,
			expectedWebPropIDs: []string{"example.com:443"},
		},
		{
			name:              "invalid web property - bad port",
			rawAssets:         []string{"example.com:abc"},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewInvalidAssetIDError("example.com:abc", "unable to infer asset type"),
			expectedUnknowns:  []string{"example.com:abc"},
		},

		// Error scenarios - No assets
		{
			name:              "no assets provided",
			rawAssets:         []string{},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewNoAssetsError(),
		},
		{
			name:              "only empty strings",
			rawAssets:         []string{"", "  ", "\t"},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewNoAssetsError(),
		},

		// Edge cases - Whitespace and duplicates
		{
			name:              "hosts with whitespace",
			rawAssets:         []string{" 8.8.8.8 ", "\t1.1.1.1\n", "  9.9.9.9"},
			expectedAssetType: AssetTypeHost,
			expectedError:     nil,
			expectedHostIDs:   []string{"8.8.8.8", "1.1.1.1", "9.9.9.9"},
		},
		{
			name:              "certificates with whitespace",
			rawAssets:         []string{" deadbeefcafebabe1234567890abcdef1234567890abcdef1234567890abcdef ", "\tfeedface1234567890abcdef1234567890abcdef1234567890abcdef12345678\n"},
			expectedAssetType: AssetTypeCertificate,
			expectedError:     nil,
			expectedCertIDs:   []string{"deadbeefcafebabe1234567890abcdef1234567890abcdef1234567890abcdef", "feedface1234567890abcdef1234567890abcdef1234567890abcdef12345678"},
		},
		{
			name:               "web properties with whitespace",
			rawAssets:          []string{" example.com:443 ", "\ttest.org:8080\n"},
			expectedAssetType:  AssetTypeWebProperty,
			expectedError:      nil,
			expectedWebPropIDs: []string{"example.com:443", "test.org:8080"},
		},
		{
			name:              "duplicate hosts",
			rawAssets:         []string{"8.8.8.8", "8.8.8.8", "1.1.1.1"},
			expectedAssetType: AssetTypeHost,
			expectedError:     nil,
			expectedHostIDs:   []string{"8.8.8.8", "1.1.1.1"}, // Duplicates should be deduplicated
		},
		{
			name:              "mixed valid and empty assets",
			rawAssets:         []string{"", "192.168.1.1", "", "10.0.0.1", "  "},
			expectedAssetType: AssetTypeHost,
			expectedError:     nil,
			expectedHostIDs:   []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:              "mixed valid and invalid assets",
			rawAssets:         []string{"8.8.8.8", "invalid-asset", "1.1.1.1"},
			expectedAssetType: AssetTypeUnknown,
			expectedError:     NewInvalidAssetIDError("invalid-asset", "unable to infer asset type"),
			expectedUnknowns:  []string{"invalid-asset"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Create classifier with raw assets
			classifier := NewAssetClassifier(tc.rawAssets...)

			// Step 2: Get asset type and handle errors
			assetType, err := classifier.AssetType()
			require.Equal(t, tc.expectedError, err, "AssetType() error mismatch")
			require.Equal(t, tc.expectedAssetType, assetType, "AssetType() result mismatch")

			// Step 3: If no error, call the appropriate ID function based on asset type
			if err == nil {
				switch assetType {
				case AssetTypeHost:
					hostIDs := classifier.HostIDs()
					hostStrings := make([]string, len(hostIDs))
					for i, h := range hostIDs {
						hostStrings[i] = h.String()
					}
					require.ElementsMatch(t, tc.expectedHostIDs, hostStrings, "HostIDs() mismatch")

					// Verify other ID functions return empty
					require.Empty(t, classifier.CertificateIDs(), "CertificateIDs() should be empty for host assets")
					require.Empty(t, classifier.WebPropertyIDs(), "WebPropertyIDs() should be empty for host assets")

				case AssetTypeCertificate:
					certIDs := classifier.CertificateIDs()
					certStrings := make([]string, len(certIDs))
					for i, c := range certIDs {
						certStrings[i] = c.String()
					}
					require.ElementsMatch(t, tc.expectedCertIDs, certStrings, "CertificateIDs() mismatch")

					// Verify other ID functions return empty
					require.Empty(t, classifier.HostIDs(), "HostIDs() should be empty for certificate assets")
					require.Empty(t, classifier.WebPropertyIDs(), "WebPropertyIDs() should be empty for certificate assets")

				case AssetTypeWebProperty:
					webPropIDs := classifier.WebPropertyIDs()
					webPropStrings := make([]string, len(webPropIDs))
					for i, w := range webPropIDs {
						webPropStrings[i] = w.String()
					}
					require.ElementsMatch(t, tc.expectedWebPropIDs, webPropStrings, "WebPropertyIDs() mismatch")

					// Verify other ID functions return empty
					require.Empty(t, classifier.HostIDs(), "HostIDs() should be empty for web property assets")
					require.Empty(t, classifier.CertificateIDs(), "CertificateIDs() should be empty for web property assets")
				}
			}

			// Step 4: Always check unknown assets (for error cases)
			if tc.expectedUnknowns != nil {
				unknowns := classifier.UnknownAssets()
				require.ElementsMatch(t, tc.expectedUnknowns, unknowns, "UnknownAssets() mismatch")
			}
		})
	}
}
