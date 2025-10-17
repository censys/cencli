package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchConfig(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		assert    func(t *testing.T, cfg *Config)
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "valid_search_config",
			content: `search:
  page-size: 50
  max-pages: 5
`,
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, int64(50), cfg.Search.PageSize)
				assert.Equal(t, int64(5), cfg.Search.MaxPages)
			},
		},
		{
			name:    "default_search_config",
			content: "",
			assert: func(t *testing.T, cfg *Config) {
				// Should use defaults from defaultConfig
				assert.Equal(t, int64(100), cfg.Search.PageSize)
				assert.Equal(t, int64(1), cfg.Search.MaxPages)
			},
		},
		{
			name: "page_size_only",
			content: `search:
  page-size: 200
`,
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, int64(200), cfg.Search.PageSize)
				assert.Equal(t, int64(1), cfg.Search.MaxPages) // Should use default
			},
		},
		{
			name: "max_pages_only",
			content: `search:
  max-pages: 10
`,
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, int64(100), cfg.Search.PageSize) // Should use default
				assert.Equal(t, int64(10), cfg.Search.MaxPages)
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
