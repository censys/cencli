package helpers

import (
	"fmt"
	"reflect"
	"strings"
)

// softwareHelper renders software and hardware information in a consistent format.
// It can handle both arrays of attributes and single attribute objects.
// When a product name starts with the vendor name, only the product is displayed.
// Version information is always placed at the end.
// Returns empty string when neither vendor nor product are present.
type softwareHelper struct{}

var _ HandlebarsHelper = &softwareHelper{}

// NewSoftwareHelper creates a helper that renders software and hardware information.
// It formats the data consistently regardless of whether it's an array or single object.
// The helper intelligently handles vendor/product display and version placement.
func NewSoftwareHelper() HandlebarsHelper {
	return &softwareHelper{}
}

func (h *softwareHelper) Name() string {
	return "render_components"
}

func (h *softwareHelper) Function() any {
	return func(v any) string {
		if v == nil {
			return ""
		}

		// Check if it's an array/slice of attributes
		val := reflect.ValueOf(v)
		if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
			return h.renderSoftwareSlice(val)
		}

		// Check if it's a single attribute object (like hardware in host template)
		if val.Kind() == reflect.Struct || val.Kind() == reflect.Map || val.Kind() == reflect.Interface {
			return h.renderSingleAttribute(v)
		}

		// If it's a string or other simple type, treat it as a single value
		return fmt.Sprint(v)
	}
}

// renderSoftwareSlice renders a slice/array of software or hardware attributes
func (h *softwareHelper) renderSoftwareSlice(val reflect.Value) string {
	var parts []string

	for i := 0; i < val.Len(); i++ {
		item := val.Index(i).Interface()
		formatted := h.renderSingleAttribute(item)
		if formatted != "" {
			if i > 0 {
				parts = append(parts, ", "+formatted)
			} else {
				parts = append(parts, formatted)
			}
		}
	}

	return strings.Join(parts, "")
}

// renderSingleAttribute renders a single software or hardware attribute
func (h *softwareHelper) renderSingleAttribute(v any) string {
	// Handle different types of attribute objects

	// Try to handle as map first (common in template data)
	if attrMap, ok := v.(map[string]any); ok {
		return h.formatAttributeMap(attrMap)
	}

	// Try to handle as struct using reflection
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Struct {
		return h.formatAttributeStruct(val)
	}

	// Fallback to string representation
	return fmt.Sprint(v)
}

// formatAttributeMap formats an attribute from a map[string]any
func (h *softwareHelper) formatAttributeMap(attr map[string]any) string {
	// Get vendor and product (required fields)
	vendor, hasVendor := attr["vendor"].(string)
	product, hasProduct := attr["product"].(string)

	// Check if we have meaningful data
	if (!hasVendor || vendor == "") && (!hasProduct || product == "") {
		return "" // No meaningful data to display
	}

	var parts []string

	// Determine what to display
	if hasVendor && vendor != "" && hasProduct && product != "" {
		// Both vendor and product exist
		if strings.HasPrefix(product, vendor) {
			// If product starts with vendor, only show product
			parts = append(parts, formatComponentName(product))
		} else {
			// Show both vendor and product
			parts = append(parts, formatComponentName(vendor))
			parts = append(parts, formatComponentName(product))
		}
	} else if hasVendor && vendor != "" {
		// Only vendor
		parts = append(parts, formatComponentName(vendor))
	} else if hasProduct && product != "" {
		// Only product
		parts = append(parts, formatComponentName(product))
	}

	// Add version if available (always at the end)
	if version, ok := attr["version"].(string); ok && version != "" {
		parts = append(parts, version)
	}

	return strings.Join(parts, " ")
}

// formatAttributeStruct formats an attribute using reflection
func (h *softwareHelper) formatAttributeStruct(val reflect.Value) string {
	// Try to get vendor field
	vendorField := val.FieldByName("Vendor")
	if !vendorField.IsValid() {
		vendorField = val.FieldByName("vendor")
	}
	var vendor string
	var hasVendor bool
	if vendorField.IsValid() && vendorField.Kind() == reflect.String {
		vendor = vendorField.String()
		hasVendor = vendor != ""
	}

	// Try to get product field
	productField := val.FieldByName("Product")
	if !productField.IsValid() {
		productField = val.FieldByName("product")
	}
	var product string
	var hasProduct bool
	if productField.IsValid() && productField.Kind() == reflect.String {
		product = productField.String()
		hasProduct = product != ""
	}

	// Check if we have meaningful data
	if !hasVendor && !hasProduct {
		return "" // No meaningful data to display
	}

	var parts []string

	// Determine what to display
	if hasVendor && hasProduct {
		// Both vendor and product exist
		if strings.HasPrefix(product, vendor) {
			// If product starts with vendor, only show product
			parts = append(parts, formatComponentName(product))
		} else {
			// Show both vendor and product
			parts = append(parts, formatComponentName(vendor))
			parts = append(parts, formatComponentName(product))
		}
	} else if hasVendor {
		// Only vendor
		parts = append(parts, formatComponentName(vendor))
	} else if hasProduct {
		// Only product
		parts = append(parts, formatComponentName(product))
	}

	// Try to get version field (always at the end)
	versionField := val.FieldByName("Version")
	if !versionField.IsValid() {
		versionField = val.FieldByName("version")
	}
	if versionField.IsValid() && versionField.Kind() == reflect.String {
		if version := versionField.String(); version != "" {
			parts = append(parts, version)
		}
	}

	return strings.Join(parts, " ")
}

// formatComponentName formats a component name by replacing underscores with spaces
// and capitalizing the first letter of each word.
func formatComponentName(name string) string {
	if name == "" {
		return ""
	}

	// Replace underscores with spaces
	name = strings.ReplaceAll(name, "_", " ")

	// Split into words and capitalize each
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}
