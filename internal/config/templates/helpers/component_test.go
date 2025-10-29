package helpers

import (
	"os"
	"path/filepath"
	"testing"

	handlebars "github.com/aymerick/raymond"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoftwareHelper_Name(t *testing.T) {
	helper := NewSoftwareHelper()
	assert.Equal(t, "render_components", helper.Name())
}

func TestSoftwareHelper_SliceOfMaps(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "empty slice",
			input:    []map[string]any{},
			expected: "",
		},
		{
			name: "single software entry",
			input: []map[string]any{
				{"vendor": "Apache", "product": "HTTP Server"},
			},
			expected: "Apache HTTP Server",
		},
		{
			name: "multiple software entries",
			input: []map[string]any{
				{"vendor": "Apache", "product": "HTTP Server"},
				{"vendor": "OpenSSL", "product": "OpenSSL"},
			},
			expected: "Apache HTTP Server, OpenSSL",
		},
		{
			name: "software with version",
			input: []map[string]any{
				{"vendor": "Apache", "product": "HTTP Server", "version": "2.4.41"},
			},
			expected: "Apache HTTP Server 2.4.41",
		},
		{
			name: "software where product starts with vendor",
			input: []map[string]any{
				{"vendor": "Apache", "product": "Apache HTTP Server"},
			},
			expected: "Apache HTTP Server",
		},
		{
			name: "software where product starts with vendor with version",
			input: []map[string]any{
				{"vendor": "Apache", "product": "Apache HTTP Server", "version": "2.4.41"},
			},
			expected: "Apache HTTP Server 2.4.41",
		},
		{
			name: "software only vendor",
			input: []map[string]any{
				{"vendor": "Apache"},
			},
			expected: "Apache",
		},
		{
			name: "software only product",
			input: []map[string]any{
				{"product": "HTTP Server"},
			},
			expected: "HTTP Server",
		},
		{
			name: "empty map (no vendor, no product)",
			input: []map[string]any{
				{},
			},
			expected: "",
		},
		{
			name: "empty strings for vendor and product",
			input: []map[string]any{
				{"vendor": "", "product": ""},
			},
			expected: "",
		},
		{
			name: "only version, no vendor or product",
			input: []map[string]any{
				{"version": "2.4.41"},
			},
			expected: "",
		},
		{
			name: "vendor is empty string, only product",
			input: []map[string]any{
				{"vendor": "", "product": "HTTP Server"},
			},
			expected: "HTTP Server",
		},
		{
			name: "product is empty string, only vendor",
			input: []map[string]any{
				{"vendor": "Apache", "product": ""},
			},
			expected: "Apache",
		},
		{
			name: "product with underscores",
			input: []map[string]any{
				{"vendor": "openbsd", "product": "openssh"},
			},
			expected: "Openbsd Openssh",
		},
		{
			name: "product with underscores and version",
			input: []map[string]any{
				{"vendor": "linux", "product": "linux_kernel", "version": "5.10"},
			},
			expected: "Linux Kernel 5.10",
		},
		{
			name: "vendor and product with underscores",
			input: []map[string]any{
				{"vendor": "honeywell", "product": "xl_web_ii_controller"},
			},
			expected: "Honeywell Xl Web Ii Controller",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewSoftwareHelper()
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSoftwareHelper_SingleMap(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "single hardware entry",
			input:    map[string]any{"vendor": "Intel", "product": "Xeon"},
			expected: "Intel Xeon",
		},
		{
			name:     "hardware with version",
			input:    map[string]any{"vendor": "Intel", "product": "Xeon", "version": "E5-2680"},
			expected: "Intel Xeon E5-2680",
		},
		{
			name:     "hardware where product starts with vendor",
			input:    map[string]any{"vendor": "Intel", "product": "Intel Xeon"},
			expected: "Intel Xeon",
		},
		{
			name:     "single empty map",
			input:    map[string]any{},
			expected: "",
		},
		{
			name:     "single map with empty strings",
			input:    map[string]any{"vendor": "", "product": "", "version": "1.0"},
			expected: "",
		},
		{
			name:     "single map with only version",
			input:    map[string]any{"version": "2.4.41"},
			expected: "",
		},
		{
			name:     "single map with empty vendor, valid product",
			input:    map[string]any{"vendor": "", "product": "HTTP Server"},
			expected: "HTTP Server",
		},
		{
			name:     "single map with empty product, valid vendor",
			input:    map[string]any{"vendor": "Apache", "product": ""},
			expected: "Apache",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewSoftwareHelper()
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSoftwareHelper_StructTypes(t *testing.T) {
	// Test with a struct that has capitalized field names (like SDK types)
	type Attribute struct {
		Vendor  string
		Product string
		Version string
	}

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "struct with vendor and product",
			input:    Attribute{Vendor: "Microsoft", Product: "Windows"},
			expected: "Microsoft Windows",
		},
		{
			name:     "struct with all fields",
			input:    Attribute{Vendor: "Microsoft", Product: "Windows", Version: "10"},
			expected: "Microsoft Windows 10",
		},
		{
			name:     "struct where product starts with vendor",
			input:    Attribute{Vendor: "Apache", Product: "Apache HTTP Server", Version: "2.4.41"},
			expected: "Apache HTTP Server 2.4.41",
		},
		{
			name:     "struct with empty vendor and product",
			input:    Attribute{Vendor: "", Product: "", Version: "1.0"},
			expected: "",
		},
		{
			name:     "struct with only version",
			input:    Attribute{Vendor: "", Product: "", Version: "2.4.41"},
			expected: "",
		},
		{
			name:     "struct with empty vendor, valid product",
			input:    Attribute{Vendor: "", Product: "HTTP Server", Version: "2.4"},
			expected: "HTTP Server 2.4",
		},
		{
			name:     "struct with empty product, valid vendor",
			input:    Attribute{Vendor: "Apache", Product: "", Version: "2.4.41"},
			expected: "Apache 2.4.41",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewSoftwareHelper()
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSoftwareHelper_NilAndEmpty(t *testing.T) {
	helper := NewSoftwareHelper()
	fn := helper.Function().(func(any) string)

	// Test nil
	result := fn(nil)
	assert.Equal(t, "", result)

	// Test empty slice
	result = fn([]map[string]any{})
	assert.Equal(t, "", result)

	// Test empty map
	result = fn(map[string]any{})
	assert.Equal(t, "", result)
}

func TestSoftwareHelper_InTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templatePath func(t *testing.T, tempDir string) string
		data         func() any
		expected     string
	}{
		{
			name: "software array in template",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Software: {{render_components software}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"software": []map[string]any{
						{"vendor": "Apache", "product": "HTTP Server", "version": "2.4.41"},
						{"vendor": "OpenSSL", "product": "OpenSSL", "version": "1.1.1"},
					},
				}
			},
			expected: "Software: Apache HTTP Server 2.4.41, OpenSSL 1.1.1",
		},
		{
			name: "hardware single object in template",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Hardware: {{render_components hardware}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"hardware": map[string]any{
						"vendor": "Intel", "product": "Xeon", "version": "E5-2680",
					},
				}
			},
			expected: "Hardware: Intel Xeon E5-2680",
		},
		{
			name: "software in template",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#each software}}
  - {{render_components this}}
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"software": []map[string]any{
						{"vendor": "Apache", "product": "HTTP Server"},
						{"vendor": "nginx", "product": "nginx"},
					},
				}
			},
			expected: `  - Apache HTTP Server
  - Nginx
`,
		},
		{
			name: "software where product starts with vendor in template",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#each software}}
  - {{render_components this}}
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"software": []map[string]any{
						{"vendor": "Apache", "product": "Apache HTTP Server", "version": "2.4.41"},
						{"vendor": "nginx", "product": "nginx nginx", "version": "1.20"},
					},
				}
			},
			expected: `  - Apache HTTP Server 2.4.41
  - Nginx Nginx 1.20
`,
		},
		{
			name: "mixed valid and invalid components in template",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#each components}}
  - {{render_components this}}
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"components": []map[string]any{
						{"vendor": "Apache", "product": "HTTP Server", "version": "2.4.41"},
						{},                            // Empty map
						{"vendor": "", "product": ""}, // Empty strings
						{"version": "1.0"},            // Only version
						{"vendor": "nginx", "product": "nginx", "version": "1.18"},
					},
				}
			},
			expected: `  - Apache HTTP Server 2.4.41
  - 
  - 
  - 
  - Nginx 1.18
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tempDir := t.TempDir()

			// Register helper
			helper := NewSoftwareHelper()
			RegisterHelpers(helper)

			// Read and render template
			templateBytes, err := os.ReadFile(tc.templatePath(t, tempDir))
			require.NoError(t, err)

			result, err := handlebars.Render(string(templateBytes), tc.data())
			require.NoError(t, err)

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSoftwareHelper_Interface(t *testing.T) {
	helper := NewSoftwareHelper()
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}

func TestSoftwareListHelper_Name(t *testing.T) {
	helper := NewSoftwareListHelper(false)
	assert.Equal(t, "render_component_list", helper.Name())
}

func TestSoftwareListHelper_BasicUsage(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "empty slice",
			input:    []map[string]any{},
			expected: "",
		},
		{
			name: "single software entry",
			input: []map[string]any{
				{"vendor": "Apache", "product": "HTTP Server", "version": "2.4.41"},
			},
			expected: "Apache HTTP Server 2.4.41",
		},
		{
			name: "multiple software entries",
			input: []map[string]any{
				{"vendor": "Apache", "product": "HTTP Server", "version": "2.4.41"},
				{"vendor": "OpenSSL", "product": "OpenSSL", "version": "1.1.1"},
				{"vendor": "PHP", "product": "PHP", "version": "7.4.3"},
			},
			expected: "Apache HTTP Server 2.4.41, OpenSSL 1.1.1, PHP 7.4.3",
		},
		{
			name: "software with product starting with vendor",
			input: []map[string]any{
				{"vendor": "nginx", "product": "nginx", "version": "1.20.1"},
				{"vendor": "Apache", "product": "Apache Tomcat", "version": "9.0"},
			},
			expected: "Nginx 1.20.1, Apache Tomcat 9.0",
		},
		{
			name: "mixed valid and empty entries",
			input: []map[string]any{
				{"vendor": "Apache", "product": "HTTP Server"},
				{},
				{"vendor": "OpenSSL", "product": "OpenSSL"},
			},
			expected: "Apache HTTP Server, OpenSSL",
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name: "single object (not array)",
			input: map[string]any{
				"vendor": "Apache", "product": "HTTP Server", "version": "2.4.41",
			},
			expected: "Apache HTTP Server 2.4.41",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewSoftwareListHelper(false)
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSoftwareListHelper_WithColoredCommas(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		colored  bool
		expected string
	}{
		{
			name: "colored commas (colors disabled in test env)",
			input: []map[string]any{
				{"vendor": "Apache", "product": "HTTP Server", "version": "2.4.41"},
				{"vendor": "OpenSSL", "product": "OpenSSL", "version": "1.1.1"},
			},
			colored:  true,
			expected: "Apache HTTP Server 2.4.41\033[97m, \033[0mOpenSSL 1.1.1",
		},
		{
			name: "no colored commas",
			input: []map[string]any{
				{"vendor": "Apache", "product": "HTTP Server", "version": "2.4.41"},
				{"vendor": "OpenSSL", "product": "OpenSSL", "version": "1.1.1"},
			},
			colored:  false,
			expected: "Apache HTTP Server 2.4.41, OpenSSL 1.1.1",
		},
		{
			name: "single entry with color enabled (colors disabled in test env)",
			input: []map[string]any{
				{"vendor": "Apache", "product": "HTTP Server", "version": "2.4.41"},
			},
			colored:  true,
			expected: "Apache HTTP Server 2.4.41",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewSoftwareListHelper(tc.colored)
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSoftwareListHelper_InTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templatePath func(t *testing.T, tempDir string) string
		data         func() any
		expected     string
	}{
		{
			name: "render component list in template",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Software: {{render_component_list software}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"software": []map[string]any{
						{"vendor": "Apache", "product": "HTTP Server", "version": "2.4.41"},
						{"vendor": "OpenSSL", "product": "OpenSSL", "version": "1.1.1"},
					},
				}
			},
			expected: "Software: Apache HTTP Server 2.4.41, OpenSSL 1.1.1",
		},
		{
			name: "render component list with conditional",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#if software}}Software: {{render_component_list software}}{{/if}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"software": []map[string]any{
						{"vendor": "nginx", "product": "nginx", "version": "1.20.1"},
					},
				}
			},
			expected: "Software: Nginx 1.20.1",
		},
		{
			name: "empty component list",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Software: {{render_component_list software}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"software": []map[string]any{},
				}
			},
			expected: "Software: ",
		},
		{
			name: "render hardware list in template",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Hardware: {{render_component_list hardware}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"hardware": []map[string]any{
						{"vendor": "Intel", "product": "Xeon", "version": "E5-2680"},
						{"vendor": "AMD", "product": "EPYC", "version": "7742"},
					},
				}
			},
			expected: "Hardware: Intel Xeon E5-2680, AMD EPYC 7742",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tempDir := t.TempDir()

			// Register helpers
			RegisterHelpers(NewSoftwareListHelper(false))

			// Read and render template
			templateBytes, err := os.ReadFile(tc.templatePath(t, tempDir))
			require.NoError(t, err)

			result, err := handlebars.Render(string(templateBytes), tc.data())
			require.NoError(t, err)

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSoftwareListHelper_Interface(t *testing.T) {
	helper := NewSoftwareListHelper(false)
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}

func TestHasComponentsHelper_Name(t *testing.T) {
	helper := NewHasComponentsHelper()
	assert.Equal(t, "has_components", helper.Name())
}

func TestHasComponentsHelper_BasicUsage(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: false,
		},
		{
			name:     "empty slice",
			input:    []map[string]any{},
			expected: false,
		},
		{
			name: "valid software entry",
			input: []map[string]any{
				{"vendor": "Apache", "product": "HTTP Server"},
			},
			expected: true,
		},
		{
			name: "empty map (no vendor, no product)",
			input: []map[string]any{
				{},
			},
			expected: false,
		},
		{
			name: "only type field (no vendor, no product)",
			input: []map[string]any{
				{"type": "some_type"},
			},
			expected: false,
		},
		{
			name: "mixed valid and invalid entries",
			input: []map[string]any{
				{},
				{"vendor": "Apache", "product": "HTTP Server"},
			},
			expected: true,
		},
		{
			name: "single valid object (not array)",
			input: map[string]any{
				"vendor": "Intel", "product": "Xeon",
			},
			expected: true,
		},
		{
			name: "single invalid object (not array)",
			input: map[string]any{
				"type": "some_type",
			},
			expected: false,
		},
		{
			name: "empty strings for vendor and product",
			input: []map[string]any{
				{"vendor": "", "product": ""},
			},
			expected: false,
		},
		{
			name: "only version, no vendor or product",
			input: []map[string]any{
				{"version": "2.4.41"},
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewHasComponentsHelper()
			fn := helper.Function().(func(any) bool)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasComponentsHelper_Interface(t *testing.T) {
	helper := NewHasComponentsHelper()
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}

func TestOSHelper_Name(t *testing.T) {
	helper := NewOSHelper(false)
	assert.Equal(t, "render_os", helper.Name())
}

func TestOSHelper_BasicUsage(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name: "vendor and product (different)",
			input: map[string]any{
				"vendor": "Microsoft", "product": "Windows", "version": "10",
			},
			expected: "Microsoft Windows 10",
		},
		{
			name: "product starts with vendor with underscores",
			input: map[string]any{
				"vendor": "Linux", "product": "Linux_kernel", "version": "5.10",
			},
			expected: "Linux Kernel 5.10",
		},
		{
			name: "product exactly equals vendor",
			input: map[string]any{
				"vendor": "Linux", "product": "Linux",
			},
			expected: "Linux",
		},
		{
			name: "only vendor",
			input: map[string]any{
				"vendor": "Linux",
			},
			expected: "Linux",
		},
		{
			name: "only product",
			input: map[string]any{
				"product": "Ubuntu",
			},
			expected: "Ubuntu",
		},
		{
			name: "vendor and product with version",
			input: map[string]any{
				"vendor": "Apple", "product": "macOS", "version": "13.0",
			},
			expected: "Apple MacOS 13.0",
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewOSHelper(false)
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestOSHelper_WithColor(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		colored  bool
		expected string
	}{
		{
			name: "colored output (colors disabled in test env)",
			input: map[string]any{
				"vendor": "Linux", "product": "Linux_kernel",
			},
			colored:  true,
			expected: "Linux Kernel",
		},
		{
			name: "no colored output",
			input: map[string]any{
				"vendor": "Linux", "product": "Linux_kernel",
			},
			colored:  false,
			expected: "Linux Kernel",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewOSHelper(tc.colored)
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestOSHelper_Interface(t *testing.T) {
	helper := NewOSHelper(false)
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}
