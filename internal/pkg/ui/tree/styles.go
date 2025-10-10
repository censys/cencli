package tree

import (
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/charmbracelet/lipgloss"
)

type Styles struct {
	KeyStyle    lipgloss.Style
	StringStyle lipgloss.Style
	NumberStyle lipgloss.Style
	BoolStyle   lipgloss.Style
	NullStyle   lipgloss.Style
	ObjectStyle lipgloss.Style
	ArrayStyle  lipgloss.Style

	SelectedStyle lipgloss.Style
	HeaderStyle   lipgloss.Style
	HelpStyle     lipgloss.Style
	FooterStyle   lipgloss.Style

	ExpandedSymbol  string
	CollapsedSymbol string
	LeafSymbol      string
}

func defaultStyles() Styles {
	return Styles{
		KeyStyle:    lipgloss.NewStyle().Foreground(styles.ColorAqua),
		StringStyle: lipgloss.NewStyle().Foreground(styles.ColorOrange),
		NumberStyle: lipgloss.NewStyle().Foreground(styles.ColorSage),
		BoolStyle:   lipgloss.NewStyle().Foreground(styles.ColorSage),
		NullStyle:   lipgloss.NewStyle().Foreground(styles.ColorTeal),
		ObjectStyle: lipgloss.NewStyle().Foreground(styles.ColorGray),
		ArrayStyle:  lipgloss.NewStyle().Foreground(styles.ColorGray),

		SelectedStyle: lipgloss.NewStyle().Background(styles.ColorGold).Foreground(styles.ColorBlack),
		HeaderStyle:   lipgloss.NewStyle().Foreground(styles.ColorGray).Bold(true),
		HelpStyle:     lipgloss.NewStyle().Foreground(styles.ColorGray),
		FooterStyle:   lipgloss.NewStyle().Foreground(styles.ColorGray),

		ExpandedSymbol:  "▼ ",
		CollapsedSymbol: "▶ ",
		LeafSymbol:      "  ",
	}
}
