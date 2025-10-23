package helpers

import (
	"os"
	"path/filepath"
	"testing"

	handlebars "github.com/aymerick/raymond"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/pkg/styles"
)

func TestColorHelper_WithColorEnabled(t *testing.T) {
	tests := []struct {
		name     string
		color    styles.Color
		input    any
		contains string
	}{
		{
			name:     "red color with string",
			color:    styles.ColorRed,
			input:    "error message",
			contains: "error message",
		},
		{
			name:     "blue color with string",
			color:    styles.ColorBlue,
			input:    "info message",
			contains: "info message",
		},
		{
			name:     "orange color with string",
			color:    styles.ColorOrange,
			input:    "warning message",
			contains: "warning message",
		},
		{
			name:     "yellow/gold color with string",
			color:    styles.ColorGold,
			input:    "alert message",
			contains: "alert message",
		},
		{
			name:     "color with integer",
			color:    styles.ColorRed,
			input:    42,
			contains: "42",
		},
		{
			name:     "color with float",
			color:    styles.ColorBlue,
			input:    3.14,
			contains: "3.14",
		},
		{
			name:     "color with boolean",
			color:    styles.ColorOrange,
			input:    true,
			contains: "true",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := newColorHelper("testcolor", tc.color, true)
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)

			// When color is enabled, result should contain ANSI codes
			// We can't test exact ANSI codes as they depend on the color profile,
			// but we can verify the content is there
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestColorHelper_WithColorDisabled(t *testing.T) {
	tests := []struct {
		name     string
		color    styles.Color
		input    any
		expected string
	}{
		{
			name:     "red color with string - no ANSI",
			color:    styles.ColorRed,
			input:    "error message",
			expected: "error message",
		},
		{
			name:     "blue color with string - no ANSI",
			color:    styles.ColorBlue,
			input:    "info message",
			expected: "info message",
		},
		{
			name:     "orange color with string - no ANSI",
			color:    styles.ColorOrange,
			input:    "warning message",
			expected: "warning message",
		},
		{
			name:     "yellow/gold color with string - no ANSI",
			color:    styles.ColorGold,
			input:    "alert message",
			expected: "alert message",
		},
		{
			name:     "color with integer - no ANSI",
			color:    styles.ColorRed,
			input:    42,
			expected: "42",
		},
		{
			name:     "color with float - no ANSI",
			color:    styles.ColorBlue,
			input:    3.14,
			expected: "3.14",
		},
		{
			name:     "color with boolean - no ANSI",
			color:    styles.ColorOrange,
			input:    true,
			expected: "true",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := newColorHelper("testcolor", tc.color, false)
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)

			// When color is disabled, result should be plain text
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestColorHelper_Name(t *testing.T) {
	tests := []struct {
		name         string
		helperName   string
		expectedName string
	}{
		{
			name:         "red helper name",
			helperName:   "red",
			expectedName: "red",
		},
		{
			name:         "blue helper name",
			helperName:   "blue",
			expectedName: "blue",
		},
		{
			name:         "orange helper name",
			helperName:   "orange",
			expectedName: "orange",
		},
		{
			name:         "yellow helper name",
			helperName:   "yellow",
			expectedName: "yellow",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := newColorHelper(tc.helperName, styles.ColorRed, true)
			assert.Equal(t, tc.expectedName, helper.Name())
		})
	}
}

func TestColorHelper_InTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templatePath func(t *testing.T, tempDir string) string
		data         func() any
		colored      bool
		assert       func(t *testing.T, result string)
	}{
		{
			name: "template with orange helper - colored",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `IP: {{orange ip}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{"ip": "127.0.0.1"}
			},
			colored: true,
			assert: func(t *testing.T, result string) {
				assert.Contains(t, result, "127.0.0.1")
				assert.Contains(t, result, "IP:")
			},
		},
		{
			name: "template with orange helper - uncolored",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `IP: {{orange ip}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{"ip": "127.0.0.1"}
			},
			colored: false,
			assert: func(t *testing.T, result string) {
				assert.Equal(t, "IP: 127.0.0.1", result)
			},
		},
		{
			name: "template with multiple color helpers",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "test.hbs")
				templateContent := `{{red status}}: {{blue name}} at {{orange ip}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)
				return templatePath
			},
			data: func() any {
				return map[string]any{
					"status": "ERROR",
					"name":   "server1",
					"ip":     "192.168.1.1",
				}
			},
			colored: false,
			assert: func(t *testing.T, result string) {
				assert.Equal(t, "ERROR: server1 at 192.168.1.1", result)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tempDir := t.TempDir()

			// Register helpers using RegisterHelpers which tracks already registered helpers
			colors := map[string]styles.Color{
				"red":    styles.ColorRed,
				"blue":   styles.ColorBlue,
				"orange": styles.ColorOrange,
				"yellow": styles.ColorGold,
			}

			var helpersToRegister []HandlebarsHelper
			for name, color := range colors {
				helpersToRegister = append(helpersToRegister, newColorHelper(name, color, tc.colored))
			}
			RegisterHelpers(helpersToRegister...)

			// Read and render template
			templateBytes, err := os.ReadFile(tc.templatePath(t, tempDir))
			require.NoError(t, err)

			result, err := handlebars.Render(string(templateBytes), tc.data())
			require.NoError(t, err)

			// Assert
			tc.assert(t, result)
		})
	}
}

func TestColorHelper_Interface(t *testing.T) {
	helper := newColorHelper("test", styles.ColorRed, true)
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}
