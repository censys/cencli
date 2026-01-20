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

type Color = lipgloss.AdaptiveColor

// Color variables initialized from the active color scheme.
// These can be used throughout the codebase.
var (
	ColorOrange   Color
	ColorOffWhite Color
	ColorSage     Color
	ColorTeal     Color
	ColorAqua     Color
	ColorGold     Color
	ColorRed      Color
	ColorGray     Color

	// Additional basic colors
	ColorBlack = Color{Light: "#000000", Dark: "#000000"}
	ColorBlue  = Color{Light: "#3BBFDE", Dark: "#3BBFDE"}
)

// DefaultColorScheme returns the default Censys-themed color scheme.
// To use a custom color scheme, replace DefaultColorScheme() with your own implementation.
func DefaultColorScheme() ColorScheme {
	return CensysColorScheme{}
}

func init() {
	lipgloss.SetColorProfile(termenv.TrueColor)
	if isTestEnvironment() {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
	scheme := DefaultColorScheme()
	GlobalStyles = NewStyles(scheme)

	ColorOrange = scheme.Signature()
	ColorOffWhite = scheme.Primary()
	ColorSage = scheme.Secondary()
	ColorTeal = scheme.Tertiary()
	ColorAqua = scheme.Info()
	ColorGold = scheme.Warning()
	ColorRed = scheme.Danger()
	ColorGray = scheme.Comment()
}

// ColorScheme defines the interface for providing colors to the CLI.
type ColorScheme interface {
	// Signature color is used for branding and signatures
	Signature() lipgloss.AdaptiveColor
	// Primary color is used for main content
	Primary() lipgloss.AdaptiveColor
	// Secondary color is used for supporting content
	Secondary() lipgloss.AdaptiveColor
	// Tertiary color is used for additional accents
	Tertiary() lipgloss.AdaptiveColor
	// Info color is used for informational messages
	Info() lipgloss.AdaptiveColor
	// Warning color is used for warning messages
	Warning() lipgloss.AdaptiveColor
	// Danger color is used for error messages
	Danger() lipgloss.AdaptiveColor
	// Comment color is used for less important text
	Comment() lipgloss.AdaptiveColor
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

// NewStyles creates a new Styles instance from a ColorScheme.
func NewStyles(scheme ColorScheme) *Styles {
	return &Styles{
		Signature: lipgloss.NewStyle().Foreground(scheme.Signature()),
		Primary:   lipgloss.NewStyle().Foreground(scheme.Primary()),
		Secondary: lipgloss.NewStyle().Foreground(scheme.Secondary()),
		Tertiary:  lipgloss.NewStyle().Foreground(scheme.Tertiary()),
		Info:      lipgloss.NewStyle().Foreground(scheme.Info()),
		Warning:   lipgloss.NewStyle().Foreground(scheme.Warning()),
		Danger:    lipgloss.NewStyle().Foreground(scheme.Danger()),
		Comment:   lipgloss.NewStyle().Foreground(scheme.Comment()),
		Indent4:   lipgloss.NewStyle().PaddingLeft(4),
		Indent8:   lipgloss.NewStyle().PaddingLeft(8),
	}
}

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
	if strings.HasSuffix(os.Args[0], ".test") {
		return true
	}
	if len(os.Args) > 1 && os.Args[1] == "-test.run" {
		return true
	}
	return false
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
