package helpers

import (
	"strings"

	"github.com/censys/cencli/internal/pkg/styles"
)

// locationHelper renders location information with smart deduplication.
// When city and province have the same name, only one is displayed.
// All location parts are rendered in blue when colored output is enabled.
type locationHelper struct {
	colored bool
}

var _ HandlebarsHelper = &locationHelper{}

// NewLocationHelper creates a helper that renders location information.
// When colored is true, all location parts are rendered in blue.
func NewLocationHelper(colored bool) HandlebarsHelper {
	return &locationHelper{colored: colored}
}

func (h *locationHelper) Name() string {
	return "render_location"
}

func (h *locationHelper) Function() any {
	return func(v any) string {
		if v == nil {
			return ""
		}

		// Try to handle as map (common in template data)
		locationMap, ok := v.(map[string]any)
		if !ok {
			return ""
		}

		var parts []string

		// Get location fields
		city, hasCity := locationMap["city"].(string)
		province, hasProvince := locationMap["province"].(string)
		country, hasCountry := locationMap["country"].(string)
		countryCode, hasCountryCode := locationMap["country_code"].(string)

		// Add city if present
		if hasCity && city != "" {
			parts = append(parts, h.colorize(city))
		}

		// Add province only if it's different from city
		if hasProvince && province != "" {
			// Skip province if it's the same as city (case-insensitive)
			if !hasCity || !strings.EqualFold(city, province) {
				parts = append(parts, h.colorize(province))
			}
		}

		// Add country if present
		if hasCountry && country != "" {
			parts = append(parts, h.colorize(country))
		}

		// Add country code in parentheses if present
		if hasCountryCode && countryCode != "" {
			parts = append(parts, "("+h.colorize(countryCode)+")")
		}

		return strings.Join(parts, ", ")
	}
}

// colorize applies blue color to text if colored output is enabled
func (h *locationHelper) colorize(text string) string {
	if h.colored {
		return styles.NewStyle(styles.ColorBlue).Render(text)
	}
	return text
}
