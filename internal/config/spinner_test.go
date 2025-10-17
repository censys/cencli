package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpinnerConfig(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		setup     func()
		assert    func(t *testing.T, cfg *Config)
		assertErr func(t *testing.T, err error)
	}{
		{
			name:    "default_spinner_config",
			content: "",
			assert: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.Spinner.Disabled)
				assert.Equal(t, uint64(5), cfg.Spinner.StartStopwatchAfterSeconds)
			},
		},
		{
			name: "spinner_disabled",
			content: `spinner:
  disabled: true
`,
			assert: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.Spinner.Disabled)
				assert.Equal(t, uint64(5), cfg.Spinner.StartStopwatchAfterSeconds)
			},
		},
		{
			name: "custom_stopwatch_time",
			content: `spinner:
  start-stopwatch-after: 10
`,
			assert: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.Spinner.Disabled)
				assert.Equal(t, uint64(10), cfg.Spinner.StartStopwatchAfterSeconds)
			},
		},
		{
			name: "both_spinner_fields_set",
			content: `spinner:
  disabled: true
  start-stopwatch-after: 15
`,
			assert: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.Spinner.Disabled)
				assert.Equal(t, uint64(15), cfg.Spinner.StartStopwatchAfterSeconds)
			},
		},
		{
			name: "zero_stopwatch_time",
			content: `spinner:
  start-stopwatch-after: 0
`,
			assert: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.Spinner.Disabled)
				assert.Equal(t, uint64(0), cfg.Spinner.StartStopwatchAfterSeconds)
			},
		},
		{
			name: "large_stopwatch_time",
			content: `spinner:
  start-stopwatch-after: 999
`,
			assert: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.Spinner.Disabled)
				assert.Equal(t, uint64(999), cfg.Spinner.StartStopwatchAfterSeconds)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, cleanup := setupConfigTest(t)
			defer cleanup()

			if tt.content != "" {
				writeConfigFile(t, tempDir, tt.content)
			}

			if tt.setup != nil {
				tt.setup()
			}

			cfg, err := New(tempDir)
			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				tt.assert(t, cfg)
			}
		})
	}
}

func TestSpinnerConfig_WrittenToYAML(t *testing.T) {
	tempDir, cleanup := setupConfigTest(t)
	defer cleanup()

	cfg, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify the config file contains the spinner configuration
	configPath := filepath.Join(tempDir, "config.yaml")
	fileContent, readErr := os.ReadFile(configPath)
	require.NoError(t, readErr)
	configStr := string(fileContent)

	// Check that spinner section exists with correct fields
	assert.Contains(t, configStr, "spinner:")
	assert.Contains(t, configStr, "disabled:")
	assert.Contains(t, configStr, "start-stopwatch-after:")

	// Verify default values are written
	assert.Contains(t, configStr, "disabled: false")
	assert.Contains(t, configStr, "start-stopwatch-after: 5")

	// Verify doc comments are present
	assert.Contains(t, configStr, "# Disable spinner during operations")
	assert.Contains(t, configStr, "# Show stopwatch in the spinner after this many seconds")
}

func TestSpinnerConfig_EnvironmentOverride(t *testing.T) {
	tests := []struct {
		name    string
		content string
		envVars map[string]string
		assert  func(t *testing.T, cfg *Config)
	}{
		{
			name: "env_override_disabled",
			content: `spinner:
  disabled: false
`,
			envVars: map[string]string{
				"CENCLI_SPINNER_DISABLED": "true",
			},
			assert: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.Spinner.Disabled)
			},
		},
		{
			name: "env_override_stopwatch_time",
			content: `spinner:
  start-stopwatch-after: 5
`,
			envVars: map[string]string{
				"CENCLI_SPINNER_START_STOPWATCH_AFTER": "20",
			},
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, uint64(20), cfg.Spinner.StartStopwatchAfterSeconds)
			},
		},
		{
			name:    "env_override_both_fields",
			content: "",
			envVars: map[string]string{
				"CENCLI_SPINNER_DISABLED":              "true",
				"CENCLI_SPINNER_START_STOPWATCH_AFTER": "30",
			},
			assert: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.Spinner.Disabled)
				assert.Equal(t, uint64(30), cfg.Spinner.StartStopwatchAfterSeconds)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, cleanup := setupConfigTest(t)
			defer cleanup()

			// Set environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			if tt.content != "" {
				writeConfigFile(t, tempDir, tt.content)
			}

			cfg, err := New(tempDir)
			require.NoError(t, err)
			require.NotNil(t, cfg)
			tt.assert(t, cfg)
		})
	}
}
