package helpers

import (
	"github.com/censys/cencli/internal/pkg/styles"
)

// osHelper renders operating system information with smart vendor/product handling.
// When a product name starts with the vendor name, only the product is displayed.
// Version information is always placed at the end.
type osHelper struct {
	colored bool
}

var _ HandlebarsHelper = &osHelper{}

// NewOSHelper creates a helper that renders operating system information.
// When colored is true, the output is rendered in blue.
func NewOSHelper(colored bool) HandlebarsHelper {
	return &osHelper{colored: colored}
}

func (h *osHelper) Name() string {
	return "render_os"
}

func (h *osHelper) Function() any {
	return func(v any) string {
		if v == nil {
			return ""
		}

		// Use the existing software helper to format the OS
		componentHelper := &softwareHelper{}
		formatted := componentHelper.renderSingleAttribute(v)

		// Apply blue color if enabled
		if h.colored && formatted != "" {
			return styles.NewStyle(styles.ColorBlue).Render(formatted)
		}

		return formatted
	}
}
