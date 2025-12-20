package formatter

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintJSON(t *testing.T) {
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
			expected: `{
  "age": 30,
  "name": "Alice"
}
`,
		},
		{
			name:    "simple array uncolored",
			input:   []string{"apple", "banana", "cherry"},
			colored: false,
			expected: `[
  "apple",
  "banana",
  "cherry"
]
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

			err := PrintJSON(tt.input, tt.colored)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestWriteNDJSONItem(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		colored  bool
		expected string
	}{
		{
			name:     "single object uncolored",
			input:    map[string]any{"name": "test"},
			colored:  false,
			expected: "{\"name\":\"test\"}\n",
		},
		{
			name:     "nested object uncolored",
			input:    map[string]any{"id": 1, "nested": map[string]any{"value": "x"}},
			colored:  false,
			expected: "{\"id\":1,\"nested\":{\"value\":\"x\"}}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteNDJSONItem(&buf, tt.input, tt.colored)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}
