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
	var parts []string

	// Get vendor and product (required fields)
	vendor, hasVendor := attr["vendor"].(string)
	product, hasProduct := attr["product"].(string)

	// Determine what to display
	if hasVendor && hasProduct {
		// If product starts with vendor, only show product
		if strings.HasPrefix(product, vendor) {
			parts = append(parts, product)
		} else {
			parts = append(parts, vendor)
			parts = append(parts, product)
		}
	} else if hasVendor {
		parts = append(parts, vendor)
	} else if hasProduct {
		parts = append(parts, product)
	}

	// Add version if available (always at the end)
	if version, ok := attr["version"].(string); ok && version != "" {
		parts = append(parts, version)
	}

	return strings.Join(parts, " ")
}

// formatAttributeStruct formats an attribute using reflection
func (h *softwareHelper) formatAttributeStruct(val reflect.Value) string {
	var parts []string

	// Try to get vendor field
	vendorField := val.FieldByName("Vendor")
	if !vendorField.IsValid() {
		vendorField = val.FieldByName("vendor")
	}
	if vendorField.IsValid() && vendorField.Kind() == reflect.String {
		if vendor := vendorField.String(); vendor != "" {
			parts = append(parts, vendor)
		}
	}

	// Try to get product field
	productField := val.FieldByName("Product")
	if !productField.IsValid() {
		productField = val.FieldByName("product")
	}
	if productField.IsValid() && productField.Kind() == reflect.String {
		if product := productField.String(); product != "" {
			// If we have both vendor and product, check if product starts with vendor
			if len(parts) > 0 {
				vendor := parts[0]
				if strings.HasPrefix(product, vendor) {
					// Remove vendor and only use product
					parts = []string{product}
				} else {
					parts = append(parts, product)
				}
			} else {
				parts = append(parts, product)
			}
		}
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
