package formatter

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPrintByFormat(t *testing.T) {
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	tests := []struct {
		name           string
		format         OutputFormat
		data           interface{}
		expectedInJSON bool // whether output should be valid JSON
		contains       []string
		isYAML         bool
	}{
		{
			name:           "JSON format",
			format:         OutputFormatJSON,
			data:           testData,
			expectedInJSON: true,
			contains:       []string{"key1", "value1", "key2", "value2"},
		},
		{
			name:     "YAML format",
			format:   OutputFormatYAML,
			data:     testData,
			isYAML:   true,
			contains: []string{"key1: value1", "key2: value2"},
		},
		{
			name:           "NDJSON format single item",
			format:         OutputFormatNDJSON,
			data:           testData,
			expectedInJSON: true,
			contains:       []string{"key1", "value1"},
		},
		{
			name:   "NDJSON format array",
			format: OutputFormatNDJSON,
			data: []map[string]string{
				{"id": "1", "name": "first"},
				{"id": "2", "name": "second"},
			},
			contains: []string{`"id":"1"`, `"name":"first"`, `"id":"2"`, `"name":"second"`},
		},
		{
			name:           "Unknown format defaults to JSON",
			format:         OutputFormat("unknown"),
			data:           testData,
			expectedInJSON: true,
			contains:       []string{"key1", "value1"},
		},
		{
			name:           "Nil data",
			format:         OutputFormatJSON,
			data:           nil,
			expectedInJSON: true,
			contains:       []string{"null"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output by replacing Stdout
			var buf bytes.Buffer
			old := Stdout
			Stdout = &buf
			defer func() { Stdout = old }()

			err := PrintByFormat(tt.data, tt.format, false)
			require.NoError(t, err)

			output := buf.String()

			// Check expected content
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}

			// Validate JSON format if expected
			if tt.expectedInJSON {
				var result interface{}
				err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result)
				assert.NoError(t, err, "Output should be valid JSON")
			}

			// Validate YAML format if expected
			if tt.isYAML {
				var result interface{}
				err := yaml.Unmarshal([]byte(output), &result)
				assert.NoError(t, err, "Output should be valid YAML")
			}
		})
	}
}

func TestPrintln(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	old := Stdout
	Stdout = &buf
	defer func() { Stdout = old }()

	Println(Stdout, "test message")

	assert.Equal(t, "test message\n", buf.String())
}

func TestPrintError(t *testing.T) {
	// Capture stderr output
	var buf bytes.Buffer
	old := Stderr
	Stderr = &buf
	defer func() { Stderr = old }()

	PrintError(errors.New("test error"), nil)

	assert.Contains(t, buf.String(), "test error")
}

func TestPrintf(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	old := Stdout
	Stdout = &buf
	defer func() { Stdout = old }()

	Printf(Stdout, "Number: %d, String: %s", 42, "test")

	assert.Equal(t, "Number: 42, String: test", buf.String())
}
