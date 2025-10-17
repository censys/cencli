package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryStrategy(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		setup     func()
		assert    func(t *testing.T, cfg *Config)
		assertErr func(t *testing.T, err error)
	}{
		{
			name:    "custom_retry_strategy",
			content: "retry-strategy.backoff: fixed\n",
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, BackoffFixed, cfg.RetryStrategy.Backoff)
			},
		},
		{
			name:    "invalid_backoff_type",
			content: "retry-strategy.backoff: invalid\n",
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid backoff type")
			},
		},
		{
			name:    "retry_config_invalid_max_attempts",
			content: "retry-strategy.max-attempts: -1\n",
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "value cannot be negative")
			},
		},
		{
			name:    "retry_config_invalid_base_delay",
			content: "retry-strategy.base-delay: invalid\n",
			assertErr: func(t *testing.T, err error) {
				var invalidConfigErr InvalidConfigError
				assert.ErrorAs(t, err, &invalidConfigErr)
				assert.Contains(t, err.Error(), "failed to unmarshal config")
			},
		},
		{
			name:    "retry_config_numeric_base_delay",
			content: "retry-strategy.base-delay: 500\n",
			assertErr: func(t *testing.T, err error) {
				var invalidConfigErr InvalidConfigError
				assert.ErrorAs(t, err, &invalidConfigErr)
				assert.Contains(t, err.Error(), "missing unit in duration")
			},
		},
		{
			name:    "retry_config_numeric_max_delay",
			content: "retry-strategy.max-delay: 30000\n",
			assertErr: func(t *testing.T, err error) {
				var invalidConfigErr InvalidConfigError
				assert.ErrorAs(t, err, &invalidConfigErr)
				assert.Contains(t, err.Error(), "missing unit in duration")
			},
		},
		{
			name: "valid_retry_config_all_fields",
			content: `retry-strategy:
  max-attempts: 5
  base-delay: 1s
  max-delay: 60s
  backoff: exponential
`,
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, uint64(5), cfg.RetryStrategy.MaxAttempts)
				assert.Equal(t, 1*time.Second, cfg.RetryStrategy.BaseDelay)
				assert.Equal(t, 60*time.Second, cfg.RetryStrategy.MaxDelay)
				assert.Equal(t, BackoffExponential, cfg.RetryStrategy.Backoff)
			},
		},
		{
			name:    "valid_linear_backoff",
			content: "retry-strategy.backoff: linear\n",
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, BackoffLinear, cfg.RetryStrategy.Backoff)
			},
		},
		{
			name:    "valid_exponential_backoff",
			content: "retry-strategy.backoff: exponential\n",
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, BackoffExponential, cfg.RetryStrategy.Backoff)
			},
		},
		{
			name:    "viper_override_invalid_backoff",
			content: "retry-strategy.backoff: exponential\n",
			setup: func() {
				viper.Set("retry-strategy.backoff", "invalid")
			},
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid backoff type")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, cleanup := setupConfigTest(t)
			defer cleanup()

			writeConfigFile(t, tempDir, tt.content)

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

func TestBackoffType_UnmarshalText(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    BackoffType
		expectError bool
	}{
		{
			name:        "fixed",
			input:       "fixed",
			expected:    BackoffFixed,
			expectError: false,
		},
		{
			name:        "linear",
			input:       "linear",
			expected:    BackoffLinear,
			expectError: false,
		},
		{
			name:        "exponential",
			input:       "exponential",
			expected:    BackoffExponential,
			expectError: false,
		},
		{
			name:        "invalid",
			input:       "invalid",
			expectError: true,
		},
		{
			name:        "empty",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b BackoffType
			err := b.UnmarshalText([]byte(tt.input))
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidBackoffType)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, b)
			}
		})
	}
}
