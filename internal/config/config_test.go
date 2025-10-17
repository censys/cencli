package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/datetime"
	"github.com/censys/cencli/internal/pkg/formatter"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(tempDir string) error
		override  func() error
		assert    func(t *testing.T, cfg *Config, tempDir string)
		assertErr func(t *testing.T, err cenclierrors.CencliError)
	}{
		{
			name: "default_config_creation",
			assert: func(t *testing.T, cfg *Config, tempDir string) {
				assert.Equal(t, formatter.OutputFormatJSON, cfg.OutputFormat)
				configPath := filepath.Join(tempDir, "config.yaml")
				_, err := os.Stat(configPath)
				assert.NoError(t, err)
				assert.Equal(t, "json", viper.GetString("output-format"))
			},
		},
		{
			name: "existing_config_file",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte("output-format: yaml\n"), 0o644)
			},
			assert: func(t *testing.T, cfg *Config, tempDir string) {
				assert.Equal(t, formatter.OutputFormatYAML, cfg.OutputFormat)
				assert.Equal(t, "yaml", viper.GetString("output-format"))
			},
		},
		{
			name: "viper_overrides",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte("output-format: yaml\n"), 0o644)
			},
			override: func() error {
				viper.Set("output-format", "ndjson")
				return nil
			},
			assert: func(t *testing.T, cfg *Config, tempDir string) {
				assert.Equal(t, formatter.OutputFormatNDJSON, cfg.OutputFormat)
				assert.Equal(t, "ndjson", viper.GetString("output-format"))
				configPath := filepath.Join(tempDir, "config.yaml")
				fileContent, err := os.ReadFile(configPath)
				require.NoError(t, err)
				// After our fix, the updated config should be written back to the file
				assert.Contains(t, string(fileContent), "output-format: ndjson")
			},
		},
		{
			name: "invalid_output_format",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte("output-format: invalid\n"), 0o644)
			},
			assertErr: func(t *testing.T, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid output format")
			},
		},
		{
			name: "malformed_yaml_config",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte("output-format: json\n  invalid: indentation\n"), 0o644)
			},
			assertErr: func(t *testing.T, err cenclierrors.CencliError) {
				var invalidConfigErr InvalidConfigError
				assert.ErrorAs(t, err, &invalidConfigErr)
				assert.Contains(t, err.Error(), "failed to read config file")
			},
		},
		{
			name: "invalid_yaml_syntax",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte("output-format: json\n[invalid yaml"), 0o644)
			},
			assertErr: func(t *testing.T, err cenclierrors.CencliError) {
				var invalidConfigErr InvalidConfigError
				assert.ErrorAs(t, err, &invalidConfigErr)
				assert.Contains(t, err.Error(), "failed to read config file")
			},
		},
		{
			name: "viper_override_invalid_output_format",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte("output-format: json\n"), 0o644)
			},
			override: func() error {
				viper.Set("output-format", "invalid")
				return nil
			},
			assertErr: func(t *testing.T, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid output format")
			},
		},
		{
			name: "all_boolean_flags_true",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte(`no-color: true
spinner:
  disabled: true
quiet: true
debug: true
`), 0o644)
			},
			assert: func(t *testing.T, cfg *Config, tempDir string) {
				assert.True(t, cfg.NoColor)
				assert.True(t, cfg.Spinner.Disabled)
				assert.True(t, cfg.Quiet)
				assert.True(t, cfg.Debug)
			},
		},
		{
			name: "viper_override_boolean_flags",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte(`no-color: false
debug: false
`), 0o644)
			},
			override: func() error {
				viper.Set("no-color", true)
				viper.Set("debug", true)
				return nil
			},
			assert: func(t *testing.T, cfg *Config, tempDir string) {
				assert.True(t, cfg.NoColor)
				assert.True(t, cfg.Debug)
			},
		},
		{
			name: "empty_config_file",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte(""), 0o644)
			},
			assert: func(t *testing.T, cfg *Config, tempDir string) {
				// Empty config file should still load defaults
				assert.Equal(t, formatter.OutputFormatJSON, cfg.OutputFormat)
				assert.Equal(t, BackoffFixed, cfg.RetryStrategy.Backoff)
				assert.Equal(t, uint64(2), cfg.RetryStrategy.MaxAttempts)
			},
		},
		{
			name: "config_with_comments",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte(`# This is a comment
output-format: yaml # inline comment
# Another comment
`), 0o644)
			},
			assert: func(t *testing.T, cfg *Config, tempDir string) {
				assert.Equal(t, formatter.OutputFormatYAML, cfg.OutputFormat)
			},
		},
		{
			name: "valid_default_tz",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte("default-tz: America/New_York\n"), 0o644)
			},
			assert: func(t *testing.T, cfg *Config, tempDir string) {
				assert.Equal(t, datetime.TimeZoneAmericaNewYork, cfg.DefaultTZ)
			},
		},
		{
			name: "invalid_default_tz",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte("default-tz: Invalid/Timezone\n"), 0o644)
			},
			assertErr: func(t *testing.T, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid timezone")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalSettings := viper.AllSettings()
			defer func() {
				viper.Reset()
				for key, value := range originalSettings {
					viper.Set(key, value)
				}
			}()

			tempDir := t.TempDir()
			viper.Reset()

			if tt.setup != nil {
				err := tt.setup(tempDir)
				require.NoError(t, err)
			}

			if tt.override != nil {
				err := tt.override()
				require.NoError(t, err)
			}

			cfg, err := New(tempDir)
			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				tt.assert(t, cfg, tempDir)
			}
		})
	}
}

// TestConfigEnvironmentVariables tests environment variable overrides
func TestConfigEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(tempDir string) error
		envVars   map[string]string
		assert    func(t *testing.T, cfg *Config, tempDir string)
		assertErr func(t *testing.T, err cenclierrors.CencliError)
	}{
		{
			name: "env_override_output_format",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte("output-format: json\n"), 0o644)
			},
			envVars: map[string]string{
				"CENCLI_OUTPUT_FORMAT": "yaml",
			},
			assert: func(t *testing.T, cfg *Config, tempDir string) {
				assert.Equal(t, formatter.OutputFormatYAML, cfg.OutputFormat)
			},
		},
		{
			name: "env_override_invalid_output_format",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte("output-format: json\n"), 0o644)
			},
			envVars: map[string]string{
				"CENCLI_OUTPUT_FORMAT": "invalid",
			},
			assertErr: func(t *testing.T, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid output format")
			},
		},
		{
			name: "env_override_boolean_flags",
			setup: func(tempDir string) error {
				configPath := filepath.Join(tempDir, "config.yaml")
				return os.WriteFile(configPath, []byte(`no-color: false
debug: false
`), 0o644)
			},
			envVars: map[string]string{
				"CENCLI_NO_COLOR": "true",
				"CENCLI_DEBUG":    "true",
			},
			assert: func(t *testing.T, cfg *Config, tempDir string) {
				assert.True(t, cfg.NoColor)
				assert.True(t, cfg.Debug)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalSettings := viper.AllSettings()
			defer func() {
				viper.Reset()
				for key, value := range originalSettings {
					viper.Set(key, value)
				}
			}()

			// Set environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			tempDir := t.TempDir()
			viper.Reset()

			if tt.setup != nil {
				err := tt.setup(tempDir)
				require.NoError(t, err)
			}

			cfg, err := New(tempDir)
			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				tt.assert(t, cfg, tempDir)
			}
		})
	}
}

// TestConfigWriteErrors tests scenarios where config file writing might fail
func TestConfigWriteErrors(t *testing.T) {
	t.Run("readonly_directory", func(t *testing.T) {
		originalSettings := viper.AllSettings()
		defer func() {
			viper.Reset()
			for key, value := range originalSettings {
				viper.Set(key, value)
			}
		}()

		tempDir := t.TempDir()
		viper.Reset()

		// Make directory read-only
		err := os.Chmod(tempDir, 0o444)
		require.NoError(t, err)
		defer func() { _ = os.Chmod(tempDir, 0o755) }() // Restore permissions for cleanup

		_, err = New(tempDir)
		var invalidConfigErr InvalidConfigError
		assert.ErrorAs(t, err, &invalidConfigErr)
		assert.Contains(t, err.Error(), "failed to write config file")
	})
}

// TestConfigDefaults tests that default values are correctly set
func TestConfigDefaults(t *testing.T) {
	originalSettings := viper.AllSettings()
	defer func() {
		viper.Reset()
		for key, value := range originalSettings {
			viper.Set(key, value)
		}
	}()

	tempDir := t.TempDir()
	viper.Reset()

	cfg, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Assert all default values
	assert.Equal(t, formatter.OutputFormatJSON, cfg.OutputFormat)
	assert.False(t, cfg.NoColor)
	assert.False(t, cfg.Spinner.Disabled)
	assert.Equal(t, uint64(5), cfg.Spinner.StartStopwatchAfterSeconds)
	assert.False(t, cfg.Quiet)
	assert.False(t, cfg.Debug)
	assert.Equal(t, uint64(2), cfg.RetryStrategy.MaxAttempts)
	assert.Equal(t, BackoffFixed, cfg.RetryStrategy.Backoff)

	// Verify config file was created with correct content
	configPath := filepath.Join(tempDir, "config.yaml")
	_, statErr := os.Stat(configPath)
	assert.NoError(t, statErr)
}

func TestConfig_DocComments_InitialCreation(t *testing.T) {
	// Setup: Reset viper and create temp dir
	originalSettings := viper.AllSettings()
	defer func() {
		viper.Reset()
		for key, value := range originalSettings {
			viper.Set(key, value)
		}
	}()

	tempDir := t.TempDir()
	viper.Reset()

	// Initialize config - this should create config.yaml with doc comments
	cfg, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Read the generated config file
	configPath := filepath.Join(tempDir, "config.yaml")
	content, readErr := os.ReadFile(configPath)
	require.NoError(t, readErr)

	yamlStr := string(content)
	t.Logf("Generated config.yaml:\n%s", yamlStr)

	// Verify that doc comments are present for top-level fields
	assert.Contains(t, yamlStr, "# Default output format (json|yaml|ndjson|tree)")
	assert.Contains(t, yamlStr, "# Disable ANSI colors and styles")
	assert.Contains(t, yamlStr, "# Disable spinner during operations")
	assert.Contains(t, yamlStr, "# Show stopwatch in the spinner after this many seconds")
	assert.Contains(t, yamlStr, "# Suppress non-essential output")
	assert.Contains(t, yamlStr, "# Per-request timeout for HTTP requests")

	// Verify nested struct comments
	assert.Contains(t, yamlStr, "# Backoff strategy (fixed|linear|exponential)")

	// Verify search config comments
	assert.Contains(t, yamlStr, "# Default number of results per page (must be >= 1)")
	assert.Contains(t, yamlStr, "# Number of pages to fetch (max is 100)")
}

func TestConfig_DocComments_LineFormat(t *testing.T) {
	originalSettings := viper.AllSettings()
	defer func() {
		viper.Reset()
		for key, value := range originalSettings {
			viper.Set(key, value)
		}
	}()

	tempDir := t.TempDir()
	viper.Reset()

	_, err := New(tempDir)
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, "config.yaml")
	content, readErr := os.ReadFile(configPath)
	require.NoError(t, readErr)

	lines := strings.Split(string(content), "\n")

	// Check that comments are inline (on the same line as the key)
	foundInlineComment := false
	for _, line := range lines {
		if strings.Contains(line, "output-format:") && strings.Contains(line, "#") {
			foundInlineComment = true
			// Verify format: "key: value  # comment"
			assert.Regexp(t, `output-format:\s+\S+\s+#\s+.+`, line)
			break
		}
	}
	assert.True(t, foundInlineComment, "Should have at least one inline comment")
}

// Helper functions for other tests

func setupConfigTest(t *testing.T) (tempDir string, cleanup func()) {
	t.Helper()

	// Save original viper settings
	originalSettings := viper.AllSettings()

	// Create temp directory
	tempDir = t.TempDir()

	// Reset viper
	viper.Reset()

	// Return cleanup function
	cleanup = func() {
		viper.Reset()
		for key, value := range originalSettings {
			viper.Set(key, value)
		}
	}

	return tempDir, cleanup
}

func writeConfigFile(t *testing.T, tempDir string, content string) {
	t.Helper()
	configPath := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)
}
