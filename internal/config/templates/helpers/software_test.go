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
		options  []string
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
			name: "software with confidence in template",
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
  - nginx
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
  - nginx nginx 1.20
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
