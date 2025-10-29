package helpers

import (
	"github.com/censys/cencli/internal/pkg/censyscopy"
	"github.com/censys/cencli/internal/pkg/styles"
)

// clickableIPHelper renders an IP address as a clickable link to the Censys platform.
// When render is true (TTY output), the IP is rendered as a terminal hyperlink.
// When render is false (non-TTY), the IP and URL are both displayed.
type clickableIPHelper struct {
	render  bool
	colored bool
}

var _ HandlebarsHelper = &clickableIPHelper{}

// NewClickableIPHelper creates a helper that renders an IP address as a clickable link.
// When render is true, uses terminal hyperlinks (OSC 8).
// When render is false, displays both IP and URL.
func NewClickableIPHelper(render bool, colored bool) HandlebarsHelper {
	return &clickableIPHelper{render: render, colored: colored}
}

func (h *clickableIPHelper) Name() string {
	return "clickable_ip"
}

func (h *clickableIPHelper) Function() any {
	return func(ip string) string {
		if ip == "" {
			return ""
		}

		link := censyscopy.CensysHostLookupLink(ip)

		if h.render {
			// For TTY: render as clickable link with the IP as anchor text
			// Apply orange color to the IP
			coloredIP := ip
			if h.colored {
				coloredIP = styles.NewStyle(styles.ColorOrange).Render(ip)
			}
			return link.Render(coloredIP)
		}

		// For non-TTY: display both IP and URL
		// Apply orange color to the IP
		coloredIP := ip
		if h.colored {
			coloredIP = styles.NewStyle(styles.ColorOrange).Render(ip)
		}
		return coloredIP + "\nPlatform URL: " + link.String()
	}
}
