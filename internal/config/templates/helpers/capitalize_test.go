package helpers

import (
	"testing"

	handlebars "github.com/aymerick/raymond"
	"github.com/stretchr/testify/assert"
)

func TestCapitalizeHelper(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		mode     string
		expected string
	}{
		// Tests with default mode (first)
		{"lowercase word", "hello", "", "Hello"},
		{"already capitalized", "Hello", "", "Hello"},
		{"all caps", "HELLO", "", "HELLO"},
		{"single letter", "a", "", "A"},
		{"empty string", "", "", ""},
		{"with spaces", "hello world", "", "Hello world"},
		{"number", 123, "", "123"},
		{"mixed case", "hELLO", "", "HELLO"},

		// Tests with explicit "first" mode
		{"first mode lowercase", "hello world", "first", "Hello world"},
		{"first mode already capitalized", "Hello world", "first", "Hello world"},

		// Tests with "all" mode
		{"all mode lowercase", "hello world", "all", "HELLO WORLD"},
		{"all mode multiple words", "the quick brown fox", "all", "THE QUICK BROWN FOX"},
		{"all mode already capitalized", "Hello World", "all", "HELLO WORLD"},
		{"all mode mixed case", "hELLO wORLD", "all", "HELLO WORLD"},
		{"all mode single word", "hello", "all", "HELLO"},
		{"all mode with extra spaces", "hello  world", "all", "HELLO  WORLD"},
		{"all mode empty string", "", "all", ""},
	}

	helper := NewCapitalizeHelper()
	fn := helper.Function().(func(any, string) string)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := fn(tc.input, tc.mode)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCapitalizeHelper_Name(t *testing.T) {
	helper := NewCapitalizeHelper()
	assert.Equal(t, "capitalize", helper.Name())
}

func TestCapitalizeHelper_Interface(t *testing.T) {
	helper := NewCapitalizeHelper()
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}

func TestCapitalizeHelper_InTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     any
		expected string
	}{
		{
			name:     "capitalize single field",
			template: `{{capitalize name "first"}}`,
			data:     map[string]any{"name": "john"},
			expected: "John",
		},
		{
			name:     "capitalize in sentence",
			template: `Hello, {{capitalize name "first"}}!`,
			data:     map[string]any{"name": "alice"},
			expected: "Hello, Alice!",
		},
		{
			name:     "capitalize multiple fields",
			template: `{{capitalize firstName "first"}} {{capitalize lastName "first"}}`,
			data:     map[string]any{"firstName": "jane", "lastName": "doe"},
			expected: "Jane Doe",
		},
		{
			name:     "capitalize in loop",
			template: `{{#each users}}{{capitalize name "first"}}, {{/each}}`,
			data: map[string]any{
				"users": []map[string]any{
					{"name": "alice"},
					{"name": "bob"},
					{"name": "charlie"},
				},
			},
			expected: "Alice, Bob, Charlie, ",
		},
		{
			name:     "capitalize with conditional",
			template: `{{#if name}}Name: {{capitalize name "first"}}{{else}}No name{{/if}}`,
			data:     map[string]any{"name": "test"},
			expected: "Name: Test",
		},
		{
			name:     "capitalize already capitalized",
			template: `{{capitalize title "first"}}`,
			data:     map[string]any{"title": "Manager"},
			expected: "Manager",
		},
		{
			name:     "capitalize with first mode",
			template: `{{capitalize title "first"}}`,
			data:     map[string]any{"title": "hello world"},
			expected: "Hello world",
		},
		{
			name:     "capitalize with all mode",
			template: `{{capitalize title "all"}}`,
			data:     map[string]any{"title": "hello world"},
			expected: "HELLO WORLD",
		},
		{
			name:     "capitalize all mode multiple words",
			template: `{{capitalize phrase "all"}}`,
			data:     map[string]any{"phrase": "the quick brown fox"},
			expected: "THE QUICK BROWN FOX",
		},
		{
			name:     "capitalize all mode in sentence",
			template: `Title: {{capitalize title "all"}}`,
			data:     map[string]any{"title": "senior software engineer"},
			expected: "Title: SENIOR SOFTWARE ENGINEER",
		},
		{
			name:     "capitalize all mode in loop",
			template: `{{#each titles}}{{capitalize . "all"}}, {{/each}}`,
			data: map[string]any{
				"titles": []string{
					"software engineer",
					"product manager",
					"data scientist",
				},
			},
			expected: "SOFTWARE ENGINEER, PRODUCT MANAGER, DATA SCIENTIST, ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Register helper
			RegisterHelpers(NewCapitalizeHelper())

			// Render template
			result, err := handlebars.Render(tc.template, tc.data)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}
