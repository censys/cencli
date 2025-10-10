package formatter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"

	"github.com/censys/cencli/internal/pkg/styles"
)

func PrintYAML(v any, colored bool) error {
	serializer := newYamlSerializer()
	output, err := serializer.serialize(v, colored)
	if err != nil {
		return err
	}
	fmt.Fprint(Stdout, output)
	return nil
}

type yamlColors struct {
	Key       lipgloss.Style
	String    lipgloss.Style
	Number    lipgloss.Style
	Bool      lipgloss.Style
	Null      lipgloss.Style
	Anchor    lipgloss.Style
	Comment   lipgloss.Style
	Delimiter lipgloss.Style
}

func defaultYamlColors() *yamlColors {
	return &yamlColors{
		Key:       styles.NewStyle(styles.ColorAqua).Bold(true),
		String:    styles.NewStyle(styles.ColorOrange),
		Number:    styles.NewStyle(styles.ColorSage),
		Bool:      styles.NewStyle(styles.ColorSage),
		Null:      styles.NewStyle(styles.ColorTeal),
		Anchor:    styles.NewStyle(styles.ColorOffWhite),
		Comment:   styles.NewStyle(styles.ColorTeal),
		Delimiter: styles.NewStyle(styles.ColorOffWhite),
	}
}

type yamlSerializer struct {
	colors *yamlColors
}

func newYamlSerializer() *yamlSerializer {
	return &yamlSerializer{
		colors: defaultYamlColors(),
	}
}

func (s *yamlSerializer) serialize(v any, colored bool) (string, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}

	yamlContent := string(b)
	if !colored {
		return yamlContent, nil
	}

	return s.colorizeYAML(yamlContent), nil
}

// colorizeYAML applies colors to YAML content using regex patterns
func (s *yamlSerializer) colorizeYAML(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	var colorizedLines []string

	for _, line := range lines {
		colorizedLine := s.colorizeLine(line)
		colorizedLines = append(colorizedLines, colorizedLine)
	}

	return strings.Join(colorizedLines, "\n")
}

// colorizeLine applies colors to a single line of YAML
func (s *yamlSerializer) colorizeLine(line string) string {
	// Skip empty lines
	if strings.TrimSpace(line) == "" {
		return line
	}

	// Handle comments
	if commentMatch := regexp.MustCompile(`^(\s*)(#.*)$`).FindStringSubmatch(line); commentMatch != nil {
		return commentMatch[1] + s.colors.Comment.Render(commentMatch[2])
	}

	// Handle key-value pairs
	keyValueRegex := regexp.MustCompile(`^(\s*)([^:\s]+)(\s*:\s*)(.*)$`)
	if matches := keyValueRegex.FindStringSubmatch(line); matches != nil {
		indent := matches[1]
		key := matches[2]
		delimiter := matches[3]
		value := matches[4]

		coloredKey := s.colors.Key.Render(key)
		coloredDelimiter := s.colors.Delimiter.Render(delimiter)
		coloredValue := s.colorizeValue(value)

		return indent + coloredKey + coloredDelimiter + coloredValue
	}

	// Handle list items
	listRegex := regexp.MustCompile(`^(\s*)(- )(.*)$`)
	if matches := listRegex.FindStringSubmatch(line); matches != nil {
		indent := matches[1]
		bullet := matches[2]
		value := matches[3]

		coloredBullet := s.colors.Delimiter.Render(bullet)

		// Check if the value part is a key-value pair
		keyValueRegex := regexp.MustCompile(`^([^:\s]+)(\s*:\s*)(.*)$`)
		if kvMatches := keyValueRegex.FindStringSubmatch(value); kvMatches != nil {
			key := kvMatches[1]
			delimiter := kvMatches[2]
			val := kvMatches[3]

			coloredKey := s.colors.Key.Render(key)
			coloredDelimiter := s.colors.Delimiter.Render(delimiter)
			coloredVal := s.colorizeValue(val)

			return indent + coloredBullet + coloredKey + coloredDelimiter + coloredVal
		}

		coloredValue := s.colorizeValue(value)
		return indent + coloredBullet + coloredValue
	}

	// Fallback: try to colorize the whole line as a value
	return s.colorizeValue(line)
}

// colorizeValue applies appropriate colors to YAML values
func (s *yamlSerializer) colorizeValue(value string) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return value
	}

	// Handle null values
	if regexp.MustCompile(`^(null|~|)$`).MatchString(value) {
		return s.colors.Null.Render(value)
	}

	// Handle boolean values
	if regexp.MustCompile(`^(true|false|yes|no|on|off)$`).MatchString(strings.ToLower(value)) {
		return s.colors.Bool.Render(value)
	}

	// Handle numbers (integers and floats)
	if regexp.MustCompile(`^-?(\d+\.?\d*|\.\d+)([eE][-+]?\d+)?$`).MatchString(value) {
		return s.colors.Number.Render(value)
	}

	// Handle anchors and references
	if regexp.MustCompile(`^[&*][a-zA-Z0-9_-]+$`).MatchString(value) {
		return s.colors.Anchor.Render(value)
	}

	// Handle quoted strings
	if regexp.MustCompile(`^(['"].*['"])$`).MatchString(value) {
		return s.colors.String.Render(value)
	}

	// Handle unquoted strings
	return s.colors.String.Render(value)
}
