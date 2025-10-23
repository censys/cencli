package helpers

import (
	"os"
	"path/filepath"
	"testing"

	handlebars "github.com/aymerick/raymond"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinHelper_StringSlice(t *testing.T) {
	tests := []struct {
		name      string
		list      any
		delimiter string
		expected  string
	}{
		{
			name:      "empty slice",
			list:      []string{},
			delimiter: ", ",
			expected:  "",
		},
		{
			name:      "single element",
			list:      []string{"a"},
			delimiter: ", ",
			expected:  "a",
		},
		{
			name:      "multiple elements with comma",
			list:      []string{"a", "b", "c"},
			delimiter: ", ",
			expected:  "a, b, c",
		},
		{
			name:      "multiple elements with pipe",
			list:      []string{"apple", "banana", "cherry"},
			delimiter: " | ",
			expected:  "apple | banana | cherry",
		},
		{
			name:      "multiple elements with newline",
			list:      []string{"line1", "line2", "line3"},
			delimiter: "\n",
			expected:  "line1\nline2\nline3",
		},
		{
			name:      "multiple elements with dash",
			list:      []string{"one", "two", "three"},
			delimiter: "-",
			expected:  "one-two-three",
		},
		{
			name:      "multiple elements with empty delimiter",
			list:      []string{"a", "b", "c"},
			delimiter: "",
			expected:  "abc",
		},
		{
			name:      "elements with spaces",
			list:      []string{"hello world", "foo bar", "baz qux"},
			delimiter: " :: ",
			expected:  "hello world :: foo bar :: baz qux",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewJoinHelper()
			fn := helper.Function().(func(any, string) string)
			result := fn(tc.list, tc.delimiter)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJoinHelper_IntegerSlice(t *testing.T) {
	tests := []struct {
		name      string
		list      any
		delimiter string
		expected  string
	}{
		{
			name:      "integers with comma",
			list:      []int{1, 2, 3, 4, 5},
			delimiter: ", ",
			expected:  "1, 2, 3, 4, 5",
		},
		{
			name:      "integers with dash",
			list:      []int{10, 20, 30},
			delimiter: "-",
			expected:  "10-20-30",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewJoinHelper()
			fn := helper.Function().(func(any, string) string)
			result := fn(tc.list, tc.delimiter)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJoinHelper_MixedSlice(t *testing.T) {
	tests := []struct {
		name      string
		list      any
		delimiter string
		expected  string
	}{
		{
			name:      "mixed types",
			list:      []any{"hello", 42, true, 3.14},
			delimiter: ", ",
			expected:  "hello, 42, true, 3.14",
		},
		{
			name:      "mixed with nil",
			list:      []any{"a", nil, "b"},
			delimiter: "-",
			expected:  "a-<nil>-b",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewJoinHelper()
			fn := helper.Function().(func(any, string) string)
			result := fn(tc.list, tc.delimiter)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJoinHelper_Array(t *testing.T) {
	tests := []struct {
		name      string
		list      any
		delimiter string
		expected  string
	}{
		{
			name:      "array of strings",
			list:      [3]string{"x", "y", "z"},
			delimiter: ", ",
			expected:  "x, y, z",
		},
		{
			name:      "array of integers",
			list:      [4]int{100, 200, 300, 400},
			delimiter: " | ",
			expected:  "100 | 200 | 300 | 400",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewJoinHelper()
			fn := helper.Function().(func(any, string) string)
			result := fn(tc.list, tc.delimiter)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJoinHelper_Nil(t *testing.T) {
	helper := NewJoinHelper()
	fn := helper.Function().(func(any, string) string)
	result := fn(nil, ", ")
	assert.Equal(t, "", result)
}

func TestJoinHelper_NonSliceTypes(t *testing.T) {
	tests := []struct {
		name      string
		input     any
		delimiter string
		expected  string
	}{
		{
			name:      "string",
			input:     "hello",
			delimiter: ", ",
			expected:  "hello",
		},
		{
			name:      "integer",
			input:     42,
			delimiter: ", ",
			expected:  "42",
		},
		{
			name:      "boolean",
			input:     true,
			delimiter: ", ",
			expected:  "true",
		},
		{
			name:      "float",
			input:     3.14,
			delimiter: ", ",
			expected:  "3.14",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewJoinHelper()
			fn := helper.Function().(func(any, string) string)
			result := fn(tc.input, tc.delimiter)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJoinHelper_Name(t *testing.T) {
	helper := NewJoinHelper()
	assert.Equal(t, "join", helper.Name())
}

func TestJoinHelper_InTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templatePath func(t *testing.T, tempDir string) string
		data         func() any
		expected     string
	}{
		{
			name: "simple join usage",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Tags: {{join tags ", "}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"tags": []string{"go", "cli", "tool"},
				}
			},
			expected: "Tags: go, cli, tool",
		},
		{
			name: "join in each loop",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#each projects}}
Project: {{name}}
Languages: {{join languages ", "}}
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				type Project struct {
					Name      string   `json:"name"`
					Languages []string `json:"languages"`
				}
				return map[string]any{
					"projects": []Project{
						{
							Name:      "Backend API",
							Languages: []string{"Go", "Python", "SQL"},
						},
						{
							Name:      "Frontend",
							Languages: []string{"TypeScript", "CSS", "HTML"},
						},
					},
				}
			},
			expected: `Project: Backend API
Languages: Go, Python, SQL
Project: Frontend
Languages: TypeScript, CSS, HTML
`,
		},
		{
			name: "join with different delimiters",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Comma: {{join items ", "}}
Pipe: {{join items " | "}}
Dash: {{join items "-"}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"items": []string{"a", "b", "c"},
				}
			},
			expected: `Comma: a, b, c
Pipe: a | b | c
Dash: a-b-c`,
		},
		{
			name: "join with numbers",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Ports: {{join ports ", "}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"ports": []int{80, 443, 8080, 3000},
				}
			},
			expected: "Ports: 80, 443, 8080, 3000",
		},
		{
			name: "join empty list",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Tags: {{join tags ", "}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"tags": []string{},
				}
			},
			expected: "Tags: ",
		},
		{
			name: "join with conditional",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#if tags}}Tags: {{join tags ", "}}{{else}}No tags{{/if}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"tags": []string{"security", "network", "monitoring"},
				}
			},
			expected: "Tags: security, network, monitoring",
		},
		{
			name: "nested join usage",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#each teams}}
Team: {{name}}
Members: {{join members ", "}}
Skills: {{join skills " | "}}
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				type Team struct {
					Name    string   `json:"name"`
					Members []string `json:"members"`
					Skills  []string `json:"skills"`
				}
				return map[string]any{
					"teams": []Team{
						{
							Name:    "DevOps",
							Members: []string{"Alice", "Bob", "Charlie"},
							Skills:  []string{"Docker", "Kubernetes", "AWS"},
						},
						{
							Name:    "Security",
							Members: []string{"Diana"},
							Skills:  []string{"Penetration Testing", "Compliance"},
						},
					},
				}
			},
			expected: `Team: DevOps
Members: Alice, Bob, Charlie
Skills: Docker | Kubernetes | AWS
Team: Security
Members: Diana
Skills: Penetration Testing | Compliance
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tempDir := t.TempDir()

			// Register helpers using RegisterHelpers which tracks already registered helpers
			joinHelper := NewJoinHelper()
			RegisterHelpers(joinHelper)

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

func TestJoinHelper_Interface(t *testing.T) {
	helper := NewJoinHelper()
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}
