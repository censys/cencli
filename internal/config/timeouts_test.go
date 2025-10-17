package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutConfig(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		assert    func(t *testing.T, cfg *Config)
		assertErr func(t *testing.T, err error)
	}{
		{
			name:    "valid_duration",
			content: "timeouts.http: 30s\n",
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 30*time.Second, cfg.Timeouts.HTTP)
			},
		},
		{
			name:    "integer_duration_rejected",
			content: "timeouts.http: 30\n",
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "missing unit in duration")
			},
		},
		{
			name:    "invalid_duration_string",
			content: "timeouts.http: \"30\"\n",
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "missing unit in duration")
			},
		},
		{
			name:    "negative_duration",
			content: "timeouts.http: -30s\n",
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "value cannot be negative")
			},
		},
		{
			name:    "zero_duration",
			content: "timeouts.http: 0s\n",
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, time.Duration(0), cfg.Timeouts.HTTP)
			},
		},
		{
			name:    "invalid_duration_format",
			content: "timeouts.http: invalid_duration\n",
			assertErr: func(t *testing.T, err error) {
				var invalidConfigErr InvalidConfigError
				assert.ErrorAs(t, err, &invalidConfigErr)
				assert.Contains(t, err.Error(), "failed to unmarshal config")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, cleanup := setupConfigTest(t)
			defer cleanup()

			writeConfigFile(t, tempDir, tt.content)

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

func TestTimeoutConfig_EnvironmentOverride(t *testing.T) {
	tempDir, cleanup := setupConfigTest(t)
	defer cleanup()

	writeConfigFile(t, tempDir, "timeouts.http: 30s\n")

	// Set environment variable
	t.Setenv("CENCLI_TIMEOUTS_HTTP", "60s")

	cfg, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 60*time.Second, cfg.Timeouts.HTTP)
}
