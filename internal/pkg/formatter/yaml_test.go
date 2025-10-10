package formatter

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
