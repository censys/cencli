package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/config/templates"
)

func TestTemplatePathsWrittenToYAML(t *testing.T) {
	tempDir := t.TempDir()

	cfg, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify templates were initialized with paths
	require.Contains(t, cfg.Templates, templates.TemplateEntityHost)
	require.Contains(t, cfg.Templates, templates.TemplateEntityCertificate)
	require.Contains(t, cfg.Templates, templates.TemplateEntityWebProperty)
	require.Contains(t, cfg.Templates, templates.TemplateEntitySearchResult)

	// All template paths should be set
	assert.NotEmpty(t, cfg.Templates[templates.TemplateEntityHost].Path)
	assert.NotEmpty(t, cfg.Templates[templates.TemplateEntityCertificate].Path)
	assert.NotEmpty(t, cfg.Templates[templates.TemplateEntityWebProperty].Path)
	assert.NotEmpty(t, cfg.Templates[templates.TemplateEntitySearchResult].Path)

	// Verify the config file contains the template paths
	configPath := filepath.Join(tempDir, "config.yaml")
	fileContent, readErr := os.ReadFile(configPath)
	require.NoError(t, readErr)
	configStr := string(fileContent)

	// Check that templates section exists and has paths
	assert.Contains(t, configStr, "templates:")
	assert.Contains(t, configStr, "host:")
	assert.Contains(t, configStr, "certificate:")
	assert.Contains(t, configStr, "webproperty:")
	assert.Contains(t, configStr, "searchresult:")
	assert.Contains(t, configStr, "path:")
	assert.Contains(t, configStr, "host.hbs")
	assert.Contains(t, configStr, "certificate.hbs")
	assert.Contains(t, configStr, "webproperty.hbs")
	assert.Contains(t, configStr, "searchresult.hbs")

	// Verify template files were created
	templatesDir := filepath.Join(tempDir, "templates")
	assert.DirExists(t, templatesDir)
	assert.FileExists(t, filepath.Join(templatesDir, "host.hbs"))
	assert.FileExists(t, filepath.Join(templatesDir, "certificate.hbs"))
	assert.FileExists(t, filepath.Join(templatesDir, "webproperty.hbs"))
	assert.FileExists(t, filepath.Join(templatesDir, "searchresult.hbs"))
}

// TestGetTemplate tests the Config.GetTemplate method for retrieving template configurations.
func TestGetTemplate(t *testing.T) {
	t.Run("returns_template_for_valid_entity", func(t *testing.T) {
		viper.Reset()
		defer viper.Reset()

		tempDir := t.TempDir()
		cfg, err := New(tempDir)
		require.NoError(t, err)

		tmpl, err := cfg.GetTemplate(templates.TemplateEntityHost)
		require.NoError(t, err)
		assert.NotEmpty(t, tmpl.Path)
		assert.Contains(t, tmpl.Path, "host.hbs")
	})

	t.Run("returns_error_for_unregistered_entity", func(t *testing.T) {
		viper.Reset()
		defer viper.Reset()

		tempDir := t.TempDir()
		cfg, err := New(tempDir)
		require.NoError(t, err)

		_, err = cfg.GetTemplate(templates.TemplateEntity("nonexistent"))
		require.Error(t, err)
		var notRegisteredErr templates.TemplateNotRegisteredError
		assert.ErrorAs(t, err, &notRegisteredErr)
	})
}

// TestConfigReload tests that template paths persist across config reloads.
func TestConfigReload(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	tempDir := t.TempDir()

	// Create initial config
	cfg1, err := New(tempDir)
	require.NoError(t, err)
	hostPath1 := cfg1.Templates[templates.TemplateEntityHost].Path

	// Reset viper to simulate a fresh load
	viper.Reset()

	// Reload config from same directory
	cfg2, err := New(tempDir)
	require.NoError(t, err)
	hostPath2 := cfg2.Templates[templates.TemplateEntityHost].Path

	// Paths should be identical
	assert.Equal(t, hostPath1, hostPath2)
	assert.FileExists(t, hostPath2)
}
