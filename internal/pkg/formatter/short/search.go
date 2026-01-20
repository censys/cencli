package short

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/domain/assets"
)

// FIXME: make this perfect

// SearchHits renders search hits in short format.
// Renders hits in the order received, adding numbered separators with asset type.
func SearchHits(hits []assets.Asset) string {
	if len(hits) == 0 {
		return ""
	}

	b := NewBlock()

	for i, hit := range hits {
		if i > 0 {
			b.Newline()
		}

		// Add separator with hit number and type
		assetTypeName := formatAssetTypeName(hit.AssetType())
		b.SeparatorWithLabel(fmt.Sprintf("Hit #%d (%s)", i+1, assetTypeName))

		// Render the hit based on its type (without their own separators)
		switch h := hit.(type) {
		case *assets.Host:
			b.Write(renderHostShort(h))
		case *assets.Certificate:
			b.Write(renderCertificateShort(h))
		case *assets.WebProperty:
			b.Write(renderWebPropertyShort(h))
		}
	}

	return b.String()
}

// formatAssetTypeName returns a display name for the asset type
func formatAssetTypeName(assetType assets.AssetType) string {
	switch assetType {
	case assets.AssetTypeHost:
		return "host"
	case assets.AssetTypeCertificate:
		return "certificate"
	case assets.AssetTypeWebProperty:
		return "web property"
	default:
		return string(assetType)
	}
}
