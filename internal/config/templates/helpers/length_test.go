package helpers

import (
	"os"
	"path/filepath"
	"testing"

	handlebars "github.com/aymerick/raymond"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLengthHelper_Slice(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: "0",
		},
		{
			name:     "slice with one element",
			input:    []string{"a"},
			expected: "1",
		},
		{
			name:     "slice with multiple elements",
			input:    []string{"a", "b", "c"},
			expected: "3",
		},
		{
			name:     "slice of integers",
			input:    []int{1, 2, 3, 4, 5},
			expected: "5",
		},
		{
			name:     "slice of interfaces",
			input:    []any{"a", 1, true, nil},
			expected: "4",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewLengthHelper()
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLengthHelper_Array(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "array of strings",
			input:    [3]string{"a", "b", "c"},
			expected: "3",
		},
		{
			name:     "array of integers",
			input:    [5]int{1, 2, 3, 4, 5},
			expected: "5",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewLengthHelper()
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLengthHelper_Map(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: "0",
		},
		{
			name:     "map with one element",
			input:    map[string]string{"a": "1"},
			expected: "1",
		},
		{
			name:     "map with multiple elements",
			input:    map[string]string{"a": "1", "b": "2", "c": "3"},
			expected: "3",
		},
		{
			name:     "map with interface values",
			input:    map[string]any{"a": 1, "b": "two", "c": true, "d": nil},
			expected: "4",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewLengthHelper()
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLengthHelper_String(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "0",
		},
		{
			name:     "single character",
			input:    "a",
			expected: "1",
		},
		{
			name:     "multiple characters",
			input:    "hello",
			expected: "5",
		},
		{
			name:     "string with spaces",
			input:    "hello world",
			expected: "11",
		},
		{
			name:     "string with unicode",
			input:    "hello 世界",
			expected: "12", // len() returns byte count, not character count
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewLengthHelper()
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLengthHelper_Nil(t *testing.T) {
	helper := NewLengthHelper()
	fn := helper.Function().(func(any) string)
	result := fn(nil)
	assert.Equal(t, "0", result)
}

func TestLengthHelper_UnsupportedTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "integer",
			input:    42,
			expected: "0",
		},
		{
			name:     "float",
			input:    3.14,
			expected: "0",
		},
		{
			name:     "boolean",
			input:    true,
			expected: "0",
		},
		{
			name:     "struct",
			input:    struct{ Name string }{Name: "test"},
			expected: "0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewLengthHelper()
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLengthHelper_Name(t *testing.T) {
	helper := NewLengthHelper()
	assert.Equal(t, "length", helper.Name())
}

func TestLengthHelper_InTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templatePath func(t *testing.T, tempDir string) string
		data         func() any
		expected     string
	}{
		{
			name: "simple length usage",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Total: {{length items}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"items": []string{"a", "b", "c"},
				}
			},
			expected: "Total: 3",
		},
		{
			name: "length in each loop",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#each teams}}
Team: {{name}} ({{length members}} members)
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				type Member struct {
					Name string `json:"name"`
				}
				type Team struct {
					Name    string   `json:"name"`
					Members []Member `json:"members"`
				}
				return map[string]any{
					"teams": []Team{
						{
							Name: "Backend",
							Members: []Member{
								{Name: "Alice"},
								{Name: "Bob"},
								{Name: "Charlie"},
							},
						},
						{
							Name: "Frontend",
							Members: []Member{
								{Name: "Diana"},
							},
						},
					},
				}
			},
			expected: `Team: Backend (3 members)
Team: Frontend (1 members)
`,
		},
		{
			name: "nested length usage",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#each teams}}
Team: {{name}} ({{length members}} members)
{{#each members}}
  - {{name}}: {{length tasks}} tasks
{{/each}}
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				type Task struct {
					Title string `json:"title"`
				}
				type Member struct {
					Name  string `json:"name"`
					Tasks []Task `json:"tasks"`
				}
				type Team struct {
					Name    string   `json:"name"`
					Members []Member `json:"members"`
				}
				return map[string]any{
					"teams": []Team{
						{
							Name: "Backend",
							Members: []Member{
								{
									Name: "Alice",
									Tasks: []Task{
										{Title: "API Design"},
										{Title: "Database Schema"},
										{Title: "Testing"},
									},
								},
								{
									Name: "Bob",
									Tasks: []Task{
										{Title: "Code Review"},
									},
								},
							},
						},
					},
				}
			},
			expected: `Team: Backend (2 members)
  - Alice: 3 tasks
  - Bob: 1 tasks
`,
		},
		{
			name: "length with string",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Description length: {{length description}} characters`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"description": "Backend development team",
				}
			},
			expected: "Description length: 24 characters",
		},
		{
			name: "length with map",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Config keys: {{length config}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"config": map[string]any{
						"environment": "production",
						"debug":       false,
						"timeout":     30,
						"features":    []string{"auth", "logging"},
					},
				}
			},
			expected: "Config keys: 4",
		},
		{
			name: "length with empty slice",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#each members}}
  - {{name}}: {{length tasks}} tasks
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				type Task struct {
					Title string `json:"title"`
				}
				type Member struct {
					Name  string `json:"name"`
					Tasks []Task `json:"tasks"`
				}
				return map[string]any{
					"members": []Member{
						{
							Name:  "Charlie",
							Tasks: []Task{},
						},
					},
				}
			},
			expected: `  - Charlie: 0 tasks
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tempDir := t.TempDir()

			// Register helper using RegisterHelpers which tracks already registered helpers
			helper := NewLengthHelper()
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

func TestLengthHelper_Interface(t *testing.T) {
	helper := NewLengthHelper()
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}
