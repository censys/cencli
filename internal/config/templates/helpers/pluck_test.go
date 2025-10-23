package helpers

import (
	"os"
	"path/filepath"
	"testing"

	handlebars "github.com/aymerick/raymond"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluckHelper_Name(t *testing.T) {
	helper := NewPluckHelper()
	assert.Equal(t, "pluck", helper.Name())
}

func TestPluckHelper_Interface(t *testing.T) {
	helper := NewPluckHelper()
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}

func TestPluckHelper_BasicMaps(t *testing.T) {
	tests := []struct {
		name     string
		array    any
		key      string
		expected []any
	}{
		{
			name:     "pluck names from user maps",
			array:    []map[string]any{{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}, {"name": "Charlie", "age": 35}},
			key:      "name",
			expected: []any{"Alice", "Bob", "Charlie"},
		},
		{
			name:     "pluck ages from user maps",
			array:    []map[string]any{{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}, {"name": "Charlie", "age": 35}},
			key:      "age",
			expected: []any{30, 25, 35},
		},
		{
			name:     "pluck non-existent key",
			array:    []map[string]any{{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}},
			key:      "email",
			expected: []any{},
		},
		{
			name:     "pluck with mixed valid/invalid keys",
			array:    []map[string]any{{"name": "Alice", "email": "alice@test.com"}, {"name": "Bob"}, {"name": "Charlie", "email": "charlie@test.com"}},
			key:      "email",
			expected: []any{"alice@test.com", "charlie@test.com"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewPluckHelper()
			fn := helper.Function().(func(any, string) any)
			result := fn(tc.array, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPluckHelper_EmptyAndNil(t *testing.T) {
	tests := []struct {
		name     string
		array    any
		key      string
		expected []any
	}{
		{
			name:     "nil array",
			array:    nil,
			key:      "name",
			expected: []any{},
		},
		{
			name:     "empty slice",
			array:    []map[string]any{},
			key:      "name",
			expected: []any{},
		},
		{
			name:     "empty array",
			array:    [0]map[string]any{},
			key:      "name",
			expected: []any{},
		},
		{
			name:     "non-array input",
			array:    "not an array",
			key:      "name",
			expected: []any{},
		},
		{
			name:     "non-array input number",
			array:    42,
			key:      "name",
			expected: []any{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewPluckHelper()
			fn := helper.Function().(func(any, string) any)
			result := fn(tc.array, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPluckHelper_FalsyFiltering(t *testing.T) {
	tests := []struct {
		name     string
		array    any
		key      string
		expected []any
	}{
		{
			name: "filter out nil values",
			array: []map[string]any{
				{"name": "Alice", "email": "alice@test.com"},
				{"name": "Bob", "email": nil},
				{"name": "Charlie", "email": "charlie@test.com"},
			},
			key:      "email",
			expected: []any{"alice@test.com", "charlie@test.com"},
		},
		{
			name: "filter out empty strings",
			array: []map[string]any{
				{"name": "Alice", "email": "alice@test.com"},
				{"name": "Bob", "email": ""},
				{"name": "Charlie", "email": "charlie@test.com"},
			},
			key:      "email",
			expected: []any{"alice@test.com", "charlie@test.com"},
		},
		{
			name: "filter out false booleans",
			array: []map[string]any{
				{"name": "Alice", "active": true},
				{"name": "Bob", "active": false},
				{"name": "Charlie", "active": true},
			},
			key:      "active",
			expected: []any{true, true},
		},
		{
			name: "filter out zero numbers",
			array: []map[string]any{
				{"name": "Alice", "count": 5},
				{"name": "Bob", "count": 0},
				{"name": "Charlie", "count": 3},
			},
			key:      "count",
			expected: []any{5, 3},
		},
		{
			name: "filter out zero float",
			array: []map[string]any{
				{"name": "Alice", "score": 95.5},
				{"name": "Bob", "score": 0.0},
				{"name": "Charlie", "score": 87.2},
			},
			key:      "score",
			expected: []any{95.5, 87.2},
		},
		{
			name: "mixed falsy values",
			array: []map[string]any{
				{"value": "valid"},
				{"value": ""},
				{"value": nil},
				{"value": 0},
				{"value": false},
				{"value": "another valid"},
			},
			key:      "value",
			expected: []any{"valid", "another valid"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewPluckHelper()
			fn := helper.Function().(func(any, string) any)
			result := fn(tc.array, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPluckHelper_Structs(t *testing.T) {
	type User struct {
		Name  string
		Age   int
		Email string
	}

	tests := []struct {
		name     string
		array    any
		key      string
		expected []any
	}{
		{
			name:     "pluck from struct fields",
			array:    []User{{"Alice", 30, "alice@test.com"}, {"Bob", 25, "bob@test.com"}},
			key:      "Name",
			expected: []any{"Alice", "Bob"},
		},
		{
			name:     "pluck ages from struct fields",
			array:    []User{{"Alice", 30, "alice@test.com"}, {"Bob", 25, "bob@test.com"}},
			key:      "Age",
			expected: []any{30, 25},
		},
		{
			name:     "pluck emails from struct fields",
			array:    []User{{"Alice", 30, "alice@test.com"}, {"Bob", 25, "bob@test.com"}},
			key:      "Email",
			expected: []any{"alice@test.com", "bob@test.com"},
		},
		{
			name:     "pluck non-existent field",
			array:    []User{{"Alice", 30, "alice@test.com"}, {"Bob", 25, "bob@test.com"}},
			key:      "Phone",
			expected: []any{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewPluckHelper()
			fn := helper.Function().(func(any, string) any)
			result := fn(tc.array, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPluckHelper_Pointers(t *testing.T) {
	type User struct {
		Name  string
		Age   int
		Email string
	}

	users := []*User{
		{"Alice", 30, "alice@test.com"},
		{"Bob", 25, "bob@test.com"},
		nil, // nil pointer should be filtered out
	}

	tests := []struct {
		name     string
		array    any
		key      string
		expected []any
	}{
		{
			name:     "pluck from pointer structs",
			array:    users,
			key:      "Name",
			expected: []any{"Alice", "Bob"},
		},
		{
			name:     "pluck ages from pointer structs",
			array:    users,
			key:      "Age",
			expected: []any{30, 25},
		},
		{
			name:     "pluck with nil pointers",
			array:    []*User{{"Alice", 30, "alice@test.com"}, nil, {"Bob", 25, "bob@test.com"}},
			key:      "Name",
			expected: []any{"Alice", "Bob"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewPluckHelper()
			fn := helper.Function().(func(any, string) any)
			result := fn(tc.array, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPluckHelper_MixedTypes(t *testing.T) {
	tests := []struct {
		name     string
		array    any
		key      string
		expected []any
	}{
		{
			name: "mixed map and struct types",
			array: []any{
				map[string]any{"name": "Alice", "age": 30},
				map[string]any{"name": "Bob", "age": 25},
			},
			key:      "name",
			expected: []any{"Alice", "Bob"},
		},
		{
			name: "empty nested structures",
			array: []any{
				map[string]any{"data": map[string]any{}},
				map[string]any{"data": map[string]any{"value": "test"}},
			},
			key:      "data",
			expected: []any{map[string]any{"value": "test"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewPluckHelper()
			fn := helper.Function().(func(any, string) any)
			result := fn(tc.array, tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPluckHelper_InTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templatePath func(t *testing.T, tempDir string) string
		data         func() any
		expected     string
	}{
		{
			name: "pluck names from users",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Users: {{join (pluck users "name") ", "}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"users": []map[string]any{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
						{"name": "Charlie", "age": 35},
					},
				}
			},
			expected: "Users: Alice, Bob, Charlie",
		},
		{
			name: "pluck emails filtering empty ones",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `Emails: {{join (pluck users "email") ", "}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"users": []map[string]any{
						{"name": "Alice", "email": "alice@test.com"},
						{"name": "Bob", "email": ""},
						{"name": "Charlie", "email": "charlie@test.com"},
						{"name": "David", "email": nil},
					},
				}
			},
			expected: "Emails: alice@test.com, charlie@test.com",
		},
		{
			name: "pluck with conditional",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#if (pluck items "value")}}Values: {{join (pluck items "value") ", "}}{{else}}No values{{/if}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"items": []map[string]any{
						{"value": "valid1"},
						{"value": ""},
						{"value": "valid2"},
						{"value": nil},
						{"value": 0},
					},
				}
			},
			expected: "Values: valid1, valid2",
		},
		{
			name: "pluck from nested structures",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{#each projects}}Project: {{name}}, Tags: {{join (pluck tags "name") ", "}}
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"projects": []map[string]any{
						{
							"name": "API",
							"tags": []map[string]any{
								{"name": "backend", "priority": 1},
								{"name": "golang", "priority": 2},
							},
						},
						{
							"name": "Frontend",
							"tags": []map[string]any{
								{"name": "react", "priority": 1},
								{"name": "", "priority": 2},
							},
						},
					},
				}
			},
			expected: `Project: API, Tags: backend, golang
Project: Frontend, Tags: react
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tempDir := t.TempDir()

			// Register helpers using RegisterHelpers which tracks already registered helpers
			pluckHelper := NewPluckHelper()
			joinHelper := NewJoinHelper()
			RegisterHelpers(pluckHelper, joinHelper)

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
