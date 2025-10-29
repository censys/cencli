package helpers

import (
	"reflect"
	"strings"

	"github.com/censys/cencli/internal/pkg/styles"
)

// softwareListHelper renders a list of software or hardware objects as a comma-separated string.
// This is useful for displaying multiple software/hardware entries in a single line.
// When colored output is enabled, commas are rendered in white and components in blue.
// Entries with no vendor or product are automatically skipped.
type softwareListHelper struct {
	colored bool
}

var _ HandlebarsHelper = &softwareListHelper{}

// NewSoftwareListHelper creates a helper that renders a list of software or hardware objects.
// It formats each entry using the same logic as render_components and joins them with commas.
// When colored is true, commas are rendered in white and components in blue.
// Entries with no vendor or product are automatically skipped.
func NewSoftwareListHelper(colored bool) HandlebarsHelper {
	return &softwareListHelper{colored: colored}
}

func (h *softwareListHelper) Name() string {
	return "render_component_list"
}

func (h *softwareListHelper) Function() any {
	return func(v any) string {
		if v == nil {
			return ""
		}

		// Use the existing software helper to render individual components
		componentHelper := &softwareHelper{}

		// Check if it's an array/slice
		val := reflect.ValueOf(v)
		if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
			// If it's not a slice/array, just render it as a single component
			formatted := componentHelper.renderSingleAttribute(v)
			if h.colored && formatted != "" {
				return styles.NewStyle(styles.ColorBlue).Render(formatted)
			}
			return formatted
		}

		// Render each software entry and collect non-empty results
		var parts []string
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i).Interface()
			formatted := componentHelper.renderSingleAttribute(item)
			if formatted != "" {
				parts = append(parts, formatted)
			}
		}

		// Join with commas, using colors if enabled
		if h.colored && len(parts) > 0 {
			return h.joinWithColoredCommas(parts)
		}

		return strings.Join(parts, ", ")
	}
}

// joinWithColoredCommas joins the parts with white-colored commas and blue-colored components
func (h *softwareListHelper) joinWithColoredCommas(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return styles.NewStyle(styles.ColorBlue).Render(parts[0])
	}

	blueStyle := styles.NewStyle(styles.ColorBlue)
	var result strings.Builder
	for i, part := range parts {
		if i > 0 {
			// Add white comma separator
			result.WriteString("\033[97m, \033[0m")
		}
		// Render component in blue
		result.WriteString(blueStyle.Render(part))
	}
	return result.String()
}

// hasComponentsHelper checks if a list of software/hardware objects has any valid entries.
// Returns true if there's at least one entry with a vendor or product.
type hasComponentsHelper struct{}

var _ HandlebarsHelper = &hasComponentsHelper{}

// NewHasComponentsHelper creates a helper that checks if a component list has valid entries.
func NewHasComponentsHelper() HandlebarsHelper {
	return &hasComponentsHelper{}
}

func (h *hasComponentsHelper) Name() string {
	return "has_components"
}

func (h *hasComponentsHelper) Function() any {
	return func(v any) bool {
		if v == nil {
			return false
		}

		// Use the existing software helper to check individual components
		componentHelper := &softwareHelper{}

		// Check if it's an array/slice
		val := reflect.ValueOf(v)
		if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
			// If it's not a slice/array, check if it's a valid single component
			formatted := componentHelper.renderSingleAttribute(v)
			return formatted != ""
		}

		// Check if any entry in the slice has valid content
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i).Interface()
			formatted := componentHelper.renderSingleAttribute(item)
			if formatted != "" {
				return true
			}
		}

		return false
	}
}
