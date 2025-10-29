package helpers

import (
	"testing"

	handlebars "github.com/aymerick/raymond"
	"github.com/stretchr/testify/assert"
)

func TestLessThanHelper(t *testing.T) {
	tests := []struct {
		name     string
		a        any
		b        any
		expected bool
	}{
		{"int less than", 5, 10, true},
		{"int not less than", 10, 5, false},
		{"int equal", 5, 5, false},
		{"float less than", 3.14, 3.15, true},
		{"float not less than", 3.15, 3.14, false},
		{"string less than", "apple", "banana", true},
		{"string not less than", "banana", "apple", false},
		{"mixed int and float", 5, 5.1, true},
		{"nil values", nil, 5, false},
	}

	helper := NewLessThanHelper()
	fn := helper.Function().(func(any, any) bool)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := fn(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGreaterThanHelper(t *testing.T) {
	tests := []struct {
		name     string
		a        any
		b        any
		expected bool
	}{
		{"int greater than", 10, 5, true},
		{"int not greater than", 5, 10, false},
		{"int equal", 5, 5, false},
		{"float greater than", 3.15, 3.14, true},
		{"float not greater than", 3.14, 3.15, false},
		{"string greater than", "banana", "apple", true},
		{"string not greater than", "apple", "banana", false},
		{"mixed int and float", 5.1, 5, true},
		{"nil values", 5, nil, false},
	}

	helper := NewGreaterThanHelper()
	fn := helper.Function().(func(any, any) bool)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := fn(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEqualHelper(t *testing.T) {
	tests := []struct {
		name     string
		a        any
		b        any
		expected bool
	}{
		{"int equal", 5, 5, true},
		{"int not equal", 5, 10, false},
		{"float equal", 3.14, 3.14, true},
		{"float not equal", 3.14, 3.15, false},
		{"string equal", "hello", "hello", true},
		{"string not equal", "hello", "world", false},
		{"mixed int and float equal", 5, 5.0, true},
		{"bool equal", true, true, true},
		{"bool not equal", true, false, false},
		{"nil equal", nil, nil, true},
		{"nil not equal", nil, 5, false},
	}

	helper := NewEqualHelper()
	fn := helper.Function().(func(any, any) bool)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := fn(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestComparisonHelper_Name(t *testing.T) {
	assert.Equal(t, "lt", NewLessThanHelper().Name())
	assert.Equal(t, "gt", NewGreaterThanHelper().Name())
	assert.Equal(t, "eq", NewEqualHelper().Name())
}

func TestComparisonHelper_Interface(t *testing.T) {
	var _ HandlebarsHelper = NewLessThanHelper()
	var _ HandlebarsHelper = NewGreaterThanHelper()
	var _ HandlebarsHelper = NewEqualHelper()
}

func TestComparisonHelper_InTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     any
		expected string
	}{
		{
			name:     "lt helper with numbers",
			template: `{{#if (lt age 18)}}Minor{{else}}Adult{{/if}}`,
			data:     map[string]any{"age": 15},
			expected: "Minor",
		},
		{
			name:     "lt helper false case",
			template: `{{#if (lt age 18)}}Minor{{else}}Adult{{/if}}`,
			data:     map[string]any{"age": 21},
			expected: "Adult",
		},
		{
			name:     "gt helper with numbers",
			template: `{{#if (gt score 90)}}Excellent{{else}}Good{{/if}}`,
			data:     map[string]any{"score": 95},
			expected: "Excellent",
		},
		{
			name:     "gt helper false case",
			template: `{{#if (gt score 90)}}Excellent{{else}}Good{{/if}}`,
			data:     map[string]any{"score": 85},
			expected: "Good",
		},
		{
			name:     "eq helper with strings",
			template: `{{#if (eq status "active")}}Online{{else}}Offline{{/if}}`,
			data:     map[string]any{"status": "active"},
			expected: "Online",
		},
		{
			name:     "eq helper false case",
			template: `{{#if (eq status "active")}}Online{{else}}Offline{{/if}}`,
			data:     map[string]any{"status": "inactive"},
			expected: "Offline",
		},
		{
			name:     "multiple comparisons in loop",
			template: `{{#each items}}{{#if (gt price 100)}}Expensive: {{name}}{{else if (lt price 50)}}Cheap: {{name}}{{else}}Normal: {{name}}{{/if}}, {{/each}}`,
			data: map[string]any{
				"items": []map[string]any{
					{"name": "Widget", "price": 150},
					{"name": "Gadget", "price": 30},
					{"name": "Tool", "price": 75},
				},
			},
			expected: "Expensive: Widget, Cheap: Gadget, Normal: Tool, ",
		},
		{
			name:     "eq with numbers",
			template: `{{#if (eq count 0)}}Empty{{else}}Has items{{/if}}`,
			data:     map[string]any{"count": 0},
			expected: "Empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Register helpers
			RegisterHelpers(
				NewLessThanHelper(),
				NewGreaterThanHelper(),
				NewEqualHelper(),
			)

			// Render template
			result, err := renderTemplate(tc.template, tc.data)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func renderTemplate(templateStr string, data any) (string, error) {
	return handlebars.Render(templateStr, data)
}
