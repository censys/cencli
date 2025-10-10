package styles

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// GlobalStyles is the default palette used across CLI output.
// It is initialized during package init and can be overridden in tests.
var GlobalStyles *Styles

const (
	// https://no-color.org/.
	// Keep in mind this is not a configuration defined by this project,
	// but rather a common convention that this project follows.
	noColorEnvVar = "NO_COLOR"
	// https://force-color.org/.
	// Keep in mind this is not a configuration defined by this project,
	// but rather a common convention that this project follows.
	forceColorEnvVar = "FORCE_COLOR"
)

func init() {
	lipgloss.SetColorProfile(termenv.TrueColor)
	if isTestEnvironment() {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
	GlobalStyles = DefaultStyles()
}

// Styles describes the color palette and layout styles used by the CLI.
// These styles are consumed by formatter and command help rendering.
type Styles struct {
	Signature lipgloss.Style
	Primary   lipgloss.Style
	Secondary lipgloss.Style
	Tertiary  lipgloss.Style
	Info      lipgloss.Style
	Warning   lipgloss.Style
	Danger    lipgloss.Style
	Comment   lipgloss.Style
	Indent4   lipgloss.Style
	Indent8   lipgloss.Style
}

// DefaultStyles returns the default Censys-themed style palette.
func DefaultStyles() *Styles {
	return &Styles{
		Signature: lipgloss.NewStyle().Foreground(ColorOrange),
		Primary:   lipgloss.NewStyle().Foreground(ColorOffWhite),
		Secondary: lipgloss.NewStyle().Foreground(ColorSage),
		Tertiary:  lipgloss.NewStyle().Foreground(ColorTeal),
		Info:      lipgloss.NewStyle().Foreground(ColorAqua),
		Warning:   lipgloss.NewStyle().Foreground(ColorGold),
		Danger:    lipgloss.NewStyle().Foreground(ColorRed),
		Comment:   lipgloss.NewStyle().Foreground(ColorGray),
		Indent4:   lipgloss.NewStyle().PaddingLeft(4),
		Indent8:   lipgloss.NewStyle().PaddingLeft(8),
	}
}

type Color = lipgloss.AdaptiveColor

var (
	ColorOrange   = Color{Light: censysOrangeDarker, Dark: censysOrange}
	ColorOffWhite = Color{Light: black, Dark: censysOffWhite}
	ColorSage     = Color{Light: censysSageDarker, Dark: censysSage}
	ColorTeal     = Color{Light: censysTeal, Dark: censysTeal}
	ColorAqua     = Color{Light: censysAqua, Dark: censysAqua}
	ColorGold     = Color{Light: censysGoldDarker, Dark: censysGold}

	ColorRed   = Color{Light: red, Dark: red}
	ColorGray  = Color{Light: gray, Dark: gray}
	ColorBlack = Color{Light: black, Dark: black}
	ColorBlue  = Color{Light: blue, Dark: blue}
)

const (
	red   = "#dc322f"
	blue  = "#3BBFDE"
	gray  = "#808080"
	black = "#000000"

	censysOrange       = "#FFAD5B"
	censysOrangeDarker = "#ed9134"
	censysTeal         = "#387782"
	censysAqua         = "#38a7ab"
	censysGold         = "#BCB480"
	censysGoldDarker   = "#a39a5f"
	censysSage         = "#B6D5D4"
	censysSageDarker   = "#53b8b4"
	censysBlue         = "#3BBFDE"
	censysOffWhite     = "#FBFAF6"
)

// ColorDisabled returns true if colored output should be disabled.
// This function is not responsible for determining if output is a TTY.
// Callers should perform their own check and call DisableStyles if needed.
func ColorDisabled() bool {
	if os.Getenv(noColorEnvVar) == "1" || strings.ToLower(os.Getenv(noColorEnvVar)) == "true" {
		return true
	}
	if isTestEnvironment() {
		return true
	}
	return false
}

// ColorForced returns true if colored output should be forced.
func ColorForced() bool {
	return os.Getenv(forceColorEnvVar) == "1" || strings.ToLower(os.Getenv(forceColorEnvVar)) == "true"
}

func isTestEnvironment() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}

// DisableStyles disables ANSI styling by switching to an ASCII color profile.
// Should only be useful when the environment should allow colored output,
// but you want it to be explicitly disabled (i.e. if the output is not a TTY
// or if the user has explicitly disabled it).
func DisableStyles() {
	lipgloss.SetColorProfile(termenv.Ascii)
}

// EnableStyles enables ANSI styling by switching to a TrueColor color profile.
// Should only be used when the user explicitly forces color output.
func EnableStyles() {
	lipgloss.SetColorProfile(termenv.TrueColor)
}

// TemporarilyDisableStyles temporarily disables ANSI styling by switching to an ASCII color profile.
// Callers must defer the returned function to restore the original color profile.
func TemporarilyDisableStyles() func() {
	oldProfile := lipgloss.ColorProfile()
	DisableStyles()
	return func() {
		lipgloss.SetColorProfile(oldProfile)
	}
}

// ANSIPrefix returns the ANSI prefix for the given style.
func ANSIPrefix(st lipgloss.Style) []byte {
	marker := "§§§"               // sentinel that won't appear in the prefix
	s := st.Render(marker)        // prefix + marker + reset
	i := strings.Index(s, marker) // find start of content
	if i < 0 {
		return []byte{} // no colorization
	}
	return []byte(s[:i])
}

// NewStyle returns a new style with the given foreground color.
func NewStyle(color Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(color)
}
