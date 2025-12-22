package formatter

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dummyStruct is a test struct with omitempty tags to verify null values are dropped
type dummyStruct struct {
	Name   string        `json:"name"`
	Age    int           `json:"age"`
	Email  *string       `json:"email,omitempty"`
	Phone  *string       `json:"phone,omitempty"`
	Active bool          `json:"active"`
	Score  *int          `json:"score,omitempty"`
	Nested *nestedStruct `json:"nested,omitempty"`
}

type nestedStruct struct {
	Value    string  `json:"value"`
	Optional *string `json:"optional,omitempty"`
}

func TestPrintYAML_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		colored  bool
		expected string
	}{
		{
			name:    "simple object uncolored",
			input:   map[string]any{"name": "Alice", "age": 30},
			colored: false,
			expected: `age: 30
name: Alice
`,
		},
		{
			name:    "nested object uncolored",
			input:   map[string]any{"user": map[string]any{"name": "Bob", "active": true}, "count": 42},
			colored: false,
			expected: `count: 42
user:
    active: true
    name: Bob
`,
		},
		{
			name:    "array uncolored",
			input:   []string{"apple", "banana", "cherry"},
			colored: false,
			expected: `- apple
- banana
- cherry
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var buf bytes.Buffer
			old := Stdout
			Stdout = &buf
			defer func() { Stdout = old }()

			err := PrintYAML(tt.input, tt.colored)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestYamlSerializer_serialize(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		colored  bool
		expected string
	}{
		{
			name:    "boolean and null values uncolored",
			input:   map[string]any{"enabled": true, "disabled": false, "missing": nil},
			colored: false,
			expected: `disabled: false
enabled: true
missing: null
`,
		},
		{
			name:    "numbers uncolored",
			input:   map[string]any{"integer": 42, "float": 3.14, "negative": -10},
			colored: false,
			expected: `float: 3.14
integer: 42
negative: -10
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := newYamlSerializer()
			result, err := serializer.serialize(tt.input, tt.colored)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestYamlSerializer_serialize_DropsNullValuesWithOmitempty(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		colored  bool
		expected string
	}{
		{
			name: "drops null pointer fields with omitempty",
			input: dummyStruct{
				Name:   "Alice",
				Age:    30,
				Email:  nil, // Should be dropped
				Phone:  nil, // Should be dropped
				Active: true,
				Score:  nil, // Should be dropped
			},
			colored: false,
			expected: `active: true
age: 30
name: Alice
`,
		},
		{
			name: "keeps non-null pointer fields with omitempty",
			input: dummyStruct{
				Name:   "Bob",
				Age:    25,
				Email:  stringPtr("bob@example.com"),
				Phone:  nil, // Should be dropped
				Active: false,
				Score:  intPtr(100),
			},
			colored: false,
			expected: `active: false
age: 25
email: bob@example.com
name: Bob
score: 100
`,
		},
		{
			name: "drops nested struct with omitempty when nil",
			input: dummyStruct{
				Name:   "Charlie",
				Age:    35,
				Email:  stringPtr("charlie@example.com"),
				Active: false,
				Nested: nil, // Should be dropped
			},
			colored: false,
			expected: `active: false
age: 35
email: charlie@example.com
name: Charlie
`,
		},
		{
			name: "keeps nested struct with omitempty when not nil",
			input: dummyStruct{
				Name:   "David",
				Age:    40,
				Email:  stringPtr("david@example.com"),
				Active: false,
				Nested: &nestedStruct{
					Value:    "test",
					Optional: nil, // Should be dropped from nested struct
				},
			},
			colored: false,
			expected: `active: false
age: 40
email: david@example.com
name: David
nested:
    value: test
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := newYamlSerializer()
			result, err := serializer.serialize(tt.input, tt.colored)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, result)
			// Verify that "null" doesn't appear in the output
			assert.NotContains(t, result, "null")
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
