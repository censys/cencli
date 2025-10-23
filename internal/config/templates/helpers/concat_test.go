package helpers

import (
	"os"
	"path/filepath"
	"testing"

	handlebars "github.com/aymerick/raymond"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcatHelper_Name(t *testing.T) {
	helper := NewConcatHelper()
	assert.Equal(t, "concat", helper.Name())
}

func TestConcatHelper_Interface(t *testing.T) {
	helper := NewConcatHelper()
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}

func TestConcatHelper_BasicConcatenation(t *testing.T) {
	tests := []struct {
		name     string
		args     []any
		expected string
	}{
		{
			name:     "no arguments",
			args:     []any{},
			expected: "",
		},
		{
			name:     "single argument",
			args:     []any{"hello"},
			expected: "hello",
		},
		{
			name:     "two arguments",
			args:     []any{"hello", "world"},
			expected: "helloworld",
		},
		{
			name:     "three arguments",
			args:     []any{"foo", "bar", "baz"},
			expected: "foobarbaz",
		},
		{
			name:     "multiple words",
			args:     []any{"hello", "world", "test"},
			expected: "helloworldtest",
		},
		{
			name:     "with spaces",
			args:     []any{"hello ", "world ", "test"},
			expected: "hello world test",
		},
		{
			name:     "with special characters",
			args:     []any{"path/", "to/", "file.txt"},
			expected: "path/to/file.txt",
		},
		{
			name:     "numbers as strings",
			args:     []any{"version", "1", ".", "2", ".", "3"},
			expected: "version1.2.3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewConcatHelper()
			fn := helper.Function().(func(any) string)
			result := fn(tc.args)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConcatHelper_EmptyStringFiltering(t *testing.T) {
	tests := []struct {
		name     string
		args     []any
		expected string
	}{
		{
			name:     "empty strings filtered out",
			args:     []any{"", "hello", "", "world", ""},
			expected: "helloworld",
		},
		{
			name:     "all empty strings",
			args:     []any{"", "", ""},
			expected: "",
		},
		{
			name:     "empty string at start",
			args:     []any{"", "hello", "world"},
			expected: "helloworld",
		},
		{
			name:     "empty string in middle",
			args:     []any{"hello", "", "world"},
			expected: "helloworld",
		},
		{
			name:     "empty string at end",
			args:     []any{"hello", "world", ""},
			expected: "helloworld",
		},
		{
			name:     "mixed empty and valid",
			args:     []any{"", "a", "", "b", "", "c", ""},
			expected: "abc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewConcatHelper()
			fn := helper.Function().(func(any) string)
			result := fn(tc.args)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConcatHelper_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		args     []any
		expected string
	}{
		{
			name:     "unicode characters",
			args:     []any{"café", "münchen", "naïve"},
			expected: "cafémünchennaïve",
		},
		{
			name:     "very long strings",
			args:     []any{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
			expected: "abcdefghij",
		},
		{
			name:     "strings with newlines",
			args:     []any{"line1\n", "line2\n", "line3"},
			expected: "line1\nline2\nline3",
		},
		{
			name:     "strings with tabs",
			args:     []any{"col1\t", "col2\t", "col3"},
			expected: "col1\tcol2\tcol3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewConcatHelper()
			fn := helper.Function().(func(any) string)
			result := fn(tc.args)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConcatHelper_InTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templatePath func(t *testing.T, tempDir string) string
		data         func() any
		expected     string
	}{
		{
			name: "simple concat usage",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Path: {{concat pathParts}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"pathParts": []string{"home", "/", "user", "/", "documents"},
				}
			},
			expected: "Path: home/user/documents",
		},
		{
			name: "concat with variables",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `URL: {{concat urlParts}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"urlParts": []string{"https", "://", "example.com", ":", "443", "/api/v1"},
				}
			},
			expected: "URL: https://example.com:443/api/v1",
		},
		{
			name: "concat in each loop",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#each users}}Full Name: {{concat nameParts}}
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"users": []map[string]any{
						{"nameParts": []string{"John", " ", "Doe"}},
						{"nameParts": []string{"Jane", " ", "Smith"}},
					},
				}
			},
			expected: `Full Name: John Doe
Full Name: Jane Smith
`,
		},
		{
			name: "concat with conditional",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#if prefix}}Title: {{concat titleParts}}{{else}}Title: {{title}}{{/if}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"prefix":     "Important",
					"title":      "Breaking News",
					"titleParts": []string{"Important", " - ", "Breaking News"},
				}
			},
			expected: "Title: Important - Breaking News",
		},
		{
			name: "concat with empty values filtered",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Path: {{concat pathParts}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"pathParts": []string{"", "/", "home", "", "/", "user", "/"},
				}
			},
			expected: "Path: /home/user/",
		},
		{
			name: "concat for file extensions",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `File: {{concat fileParts}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"fileParts": []string{"document", ".", "txt"},
				}
			},
			expected: "File: document.txt",
		},
		{
			name: "concat for CSS classes",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Class: {{concat classParts}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"classParts": []string{"btn", " ", "btn-", "large", " ", "btn-", "primary"},
				}
			},
			expected: "Class: btn btn-large btn-primary",
		},
		{
			name: "concat for nested paths",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#each categories}}{{concat pathParts}}{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"categories": []map[string]any{
						{"pathParts": []string{"../", "electronics", "/"}},
						{"pathParts": []string{"../", "books", "/"}},
						{"pathParts": []string{"../", "clothing", "/"}},
					},
				}
			},
			expected: "../electronics/../books/../clothing/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tempDir := t.TempDir()

			// Register helpers using RegisterHelpers which tracks already registered helpers
			concatHelper := NewConcatHelper()
			RegisterHelpers(concatHelper)

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
