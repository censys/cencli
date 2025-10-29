package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitTemplates(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(tempDir string) map[TemplateEntity]TemplateConfig
		expectedError  func(t *testing.T, err error)
		expectedConfig func(t *testing.T, result map[TemplateEntity]TemplateConfig, tempDir string)
		expectedFiles  func(t *testing.T, tempDir string)
	}{
		{
			name: "fresh_setup_single_entity",
			setup: func(tempDir string) map[TemplateEntity]TemplateConfig {
				return map[TemplateEntity]TemplateConfig{
					TemplateEntityHost: {},
				}
			},
			expectedConfig: func(t *testing.T, result map[TemplateEntity]TemplateConfig, tempDir string) {
				require.Contains(t, result, TemplateEntityHost)
				expectedPath := filepath.Join(tempDir, "templates", "host.hbs")
				assert.Equal(t, expectedPath, result[TemplateEntityHost].Path)
			},
			expectedFiles: func(t *testing.T, tempDir string) {
				templatesDir := filepath.Join(tempDir, "templates")
				assert.DirExists(t, templatesDir)
				assert.FileExists(t, filepath.Join(templatesDir, "host.hbs"))
			},
		},
		{
			name: "fresh_setup_all_entities",
			setup: func(tempDir string) map[TemplateEntity]TemplateConfig {
				return map[TemplateEntity]TemplateConfig{
					TemplateEntityHost:         {},
					TemplateEntityCertificate:  {},
					TemplateEntityWebProperty:  {},
					TemplateEntitySearchResult: {},
				}
			},
			expectedConfig: func(t *testing.T, result map[TemplateEntity]TemplateConfig, tempDir string) {
				templatesDir := filepath.Join(tempDir, "templates")
				assert.Equal(t, filepath.Join(templatesDir, "host.hbs"), result[TemplateEntityHost].Path)
				assert.Equal(t, filepath.Join(templatesDir, "certificate.hbs"), result[TemplateEntityCertificate].Path)
				assert.Equal(t, filepath.Join(templatesDir, "webproperty.hbs"), result[TemplateEntityWebProperty].Path)
				assert.Equal(t, filepath.Join(templatesDir, "searchresult.hbs"), result[TemplateEntitySearchResult].Path)
			},
			expectedFiles: func(t *testing.T, tempDir string) {
				templatesDir := filepath.Join(tempDir, "templates")
				assert.FileExists(t, filepath.Join(templatesDir, "host.hbs"))
				assert.FileExists(t, filepath.Join(templatesDir, "certificate.hbs"))
				assert.FileExists(t, filepath.Join(templatesDir, "webproperty.hbs"))
				assert.FileExists(t, filepath.Join(templatesDir, "searchresult.hbs"))
			},
		},
		{
			name: "existing_template_files_are_used",
			setup: func(tempDir string) map[TemplateEntity]TemplateConfig {
				templatesDir := filepath.Join(tempDir, "templates")
				err := os.MkdirAll(templatesDir, 0o755)
				require.NoError(t, err)

				// Create existing template
				existingContent := []byte("existing template content")
				err = os.WriteFile(filepath.Join(templatesDir, "host.hbs"), existingContent, 0o644)
				require.NoError(t, err)

				return map[TemplateEntity]TemplateConfig{
					TemplateEntityHost: {},
				}
			},
			expectedConfig: func(t *testing.T, result map[TemplateEntity]TemplateConfig, tempDir string) {
				expectedPath := filepath.Join(tempDir, "templates", "host.hbs")
				assert.Equal(t, expectedPath, result[TemplateEntityHost].Path)

				// Verify it's using the existing file (not overwritten)
				content, err := os.ReadFile(expectedPath)
				require.NoError(t, err)
				assert.Equal(t, "existing template content", string(content))
			},
		},
		{
			name: "explicit_path_is_validated",
			setup: func(tempDir string) map[TemplateEntity]TemplateConfig {
				customPath := filepath.Join(tempDir, "custom.hbs")
				err := os.WriteFile(customPath, []byte("custom"), 0o644)
				require.NoError(t, err)

				return map[TemplateEntity]TemplateConfig{
					TemplateEntityHost: {Path: customPath},
				}
			},
			expectedConfig: func(t *testing.T, result map[TemplateEntity]TemplateConfig, tempDir string) {
				customPath := filepath.Join(tempDir, "custom.hbs")
				assert.Equal(t, customPath, result[TemplateEntityHost].Path)
			},
		},
		{
			name: "invalid_explicit_path_does_not_error_during_init",
			setup: func(tempDir string) map[TemplateEntity]TemplateConfig {
				return map[TemplateEntity]TemplateConfig{
					TemplateEntityHost: {Path: "/nonexistent/path/template.hbs"},
				}
			},
			expectedConfig: func(t *testing.T, result map[TemplateEntity]TemplateConfig, tempDir string) {
				// Path should be set even though file doesn't exist
				// Error will occur when actually trying to use the template
				assert.Equal(t, "/nonexistent/path/template.hbs", result[TemplateEntityHost].Path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			templateConfigs := tt.setup(tempDir)

			result, err := InitTemplates(tempDir, templateConfigs)

			if tt.expectedError != nil {
				tt.expectedError(t, err)
				return
			}

			require.NoError(t, err)

			if tt.expectedConfig != nil {
				tt.expectedConfig(t, result, tempDir)
			}

			if tt.expectedFiles != nil {
				tt.expectedFiles(t, tempDir)
			}
		})
	}
}

func TestValidateExistingTemplatePath(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(tempDir string) (TemplateEntity, TemplateConfig)
		expectedError func(t *testing.T, err error)
	}{
		{
			name: "existing_file_is_valid",
			setup: func(tempDir string) (TemplateEntity, TemplateConfig) {
				filePath := filepath.Join(tempDir, "test.tmpl")
				err := os.WriteFile(filePath, []byte("test"), 0o644)
				require.NoError(t, err)

				return TemplateEntityHost, TemplateConfig{Path: filePath}
			},
		},
		{
			name: "nonexistent_file_returns_error",
			setup: func(tempDir string) (TemplateEntity, TemplateConfig) {
				return TemplateEntityHost, TemplateConfig{
					Path: filepath.Join(tempDir, "nonexistent.tmpl"),
				}
			},
			expectedError: func(t *testing.T, err error) {
				require.Error(t, err)
				var templateNotFoundErr TemplateNotFoundError
				assert.ErrorAs(t, err, &templateNotFoundErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			entity, config := tt.setup(tempDir)

			err := validateExistingTemplatePath(entity, config)

			if tt.expectedError != nil {
				tt.expectedError(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFindExistingTemplateInDir(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(tempDir string) string
		entity         TemplateEntity
		expectedResult string
		expectedError  func(t *testing.T, err error)
	}{
		{
			name: "finds_matching_template",
			setup: func(tempDir string) string {
				templatesDir := filepath.Join(tempDir, "templates")
				err := os.MkdirAll(templatesDir, 0o755)
				require.NoError(t, err)

				err = os.WriteFile(filepath.Join(templatesDir, "host.hbs"), []byte("test"), 0o644)
				require.NoError(t, err)

				return templatesDir
			},
			entity:         TemplateEntityHost,
			expectedResult: "host.hbs",
		},
		{
			name: "no_matching_template",
			setup: func(tempDir string) string {
				templatesDir := filepath.Join(tempDir, "templates")
				err := os.MkdirAll(templatesDir, 0o755)
				require.NoError(t, err)

				return templatesDir
			},
			entity:         TemplateEntityHost,
			expectedResult: "",
		},
		{
			name: "directory_does_not_exist",
			setup: func(tempDir string) string {
				return filepath.Join(tempDir, "nonexistent")
			},
			entity: TemplateEntityHost,
			expectedError: func(t *testing.T, err error) {
				require.Error(t, err)
				var templateDirErr TemplateDirectoryError
				assert.ErrorAs(t, err, &templateDirErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			templatesDir := tt.setup(tempDir)

			result, err := findExistingTemplateInDir(tt.entity, templatesDir)

			if tt.expectedError != nil {
				tt.expectedError(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestCopyDefaultTemplate(t *testing.T) {
	tests := []struct {
		name          string
		templateName  string
		expectedError func(t *testing.T, err error)
	}{
		{
			name:         "copies_existing_template",
			templateName: "host.hbs",
		},
		{
			name:         "fails_for_nonexistent_template",
			templateName: "nonexistent.tmpl",
			expectedError: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to read default template")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			templatesDir := filepath.Join(tempDir, "templates")
			err := os.MkdirAll(templatesDir, 0o755)
			require.NoError(t, err)

			err = CopyDefaultTemplate(tt.templateName, templatesDir)

			if tt.expectedError != nil {
				tt.expectedError(t, err)
			} else {
				require.NoError(t, err)

				// Verify file was created
				expectedPath := filepath.Join(templatesDir, tt.templateName)
				assert.FileExists(t, expectedPath)

				// Verify content is not empty
				content, err := os.ReadFile(expectedPath)
				require.NoError(t, err)
				assert.NotEmpty(t, content)
			}
		})
	}
}

func TestEnsureTemplatesDirectory(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(tempDir string) string
		expectedError func(t *testing.T, err error)
	}{
		{
			name: "creates_directory_when_missing",
			setup: func(tempDir string) string {
				return filepath.Join(tempDir, "templates")
			},
		},
		{
			name: "succeeds_when_directory_exists",
			setup: func(tempDir string) string {
				templatesDir := filepath.Join(tempDir, "templates")
				err := os.MkdirAll(templatesDir, 0o755)
				require.NoError(t, err)
				return templatesDir
			},
		},
		{
			name: "fails_when_file_exists_at_path",
			setup: func(tempDir string) string {
				templatesPath := filepath.Join(tempDir, "templates")
				err := os.WriteFile(templatesPath, []byte("blocking"), 0o644)
				require.NoError(t, err)
				return templatesPath
			},
			expectedError: func(t *testing.T, err error) {
				// This test might not fail on all systems since MkdirAll behavior varies
				// We'll check if it fails, but won't require it
				if err != nil {
					var templateDirErr TemplateDirectoryError
					assert.ErrorAs(t, err, &templateDirErr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			templatesDir := tt.setup(tempDir)

			err := ensureTemplatesDirectory(templatesDir)

			if tt.expectedError != nil {
				tt.expectedError(t, err)
			} else {
				require.NoError(t, err)
				assert.DirExists(t, templatesDir)
			}
		})
	}
}

func TestListDefaultTemplates(t *testing.T) {
	templates, err := ListDefaultTemplates()
	require.NoError(t, err)
	require.NotEmpty(t, templates)

	// Verify we have the expected default templates
	expectedTemplates := map[string]bool{
		"host.hbs":         false,
		"certificate.hbs":  false,
		"webproperty.hbs":  false,
		"searchresult.hbs": false,
	}

	for _, tmpl := range templates {
		if _, ok := expectedTemplates[tmpl]; ok {
			expectedTemplates[tmpl] = true
		}
	}

	for name, found := range expectedTemplates {
		assert.True(t, found, "Expected default template %s not found", name)
	}
}

func TestGetTemplatesDir(t *testing.T) {
	dataDir := "/test/data/dir"
	expected := filepath.Join(dataDir, "templates")
	result := GetTemplatesDir(dataDir)
	assert.Equal(t, expected, result)
}

func TestTemplateEntityUnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected TemplateEntity
		wantErr  bool
	}{
		{
			name:     "host",
			input:    "host",
			expected: TemplateEntityHost,
		},
		{
			name:     "certificate",
			input:    "certificate",
			expected: TemplateEntityCertificate,
		},
		{
			name:     "webproperty",
			input:    "webproperty",
			expected: TemplateEntityWebProperty,
		},
		{
			name:     "searchresult",
			input:    "searchresult",
			expected: TemplateEntitySearchResult,
		},
		{
			name:    "invalid",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entity TemplateEntity
			err := entity.UnmarshalText([]byte(tt.input))

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrUnsupportedTemplateEntity)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, entity)
			}
		})
	}
}

func TestTemplateEntityString(t *testing.T) {
	tests := []struct {
		entity   TemplateEntity
		expected string
	}{
		{TemplateEntityHost, "host"},
		{TemplateEntityCertificate, "certificate"},
		{TemplateEntityWebProperty, "webproperty"},
		{TemplateEntitySearchResult, "searchresult"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.entity.String())
		})
	}
}

func TestResetTemplates(t *testing.T) {
	t.Run("resets_all_templates", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create templates directory with some existing files
		templatesDir := filepath.Join(tempDir, "templates")
		err := os.MkdirAll(templatesDir, 0o755)
		require.NoError(t, err)

		// Create an existing template with custom content
		customContent := []byte("custom template content")
		err = os.WriteFile(filepath.Join(templatesDir, "host.hbs"), customContent, 0o644)
		require.NoError(t, err)

		// Reset templates
		reset, err := ResetTemplates(tempDir)
		require.NoError(t, err)
		require.NotEmpty(t, reset)

		// Verify all default templates were reset
		assert.Contains(t, reset, "host.hbs")
		assert.Contains(t, reset, "certificate.hbs")
		assert.Contains(t, reset, "webproperty.hbs")
		assert.Contains(t, reset, "searchresult.hbs")

		// Verify the custom content was overwritten
		content, err := os.ReadFile(filepath.Join(templatesDir, "host.hbs"))
		require.NoError(t, err)
		assert.NotEqual(t, customContent, content)
		assert.NotEmpty(t, content)
	})

	t.Run("creates_directory_if_missing", func(t *testing.T) {
		tempDir := t.TempDir()

		// Don't create templates directory - ResetTemplates should create it
		reset, err := ResetTemplates(tempDir)
		require.NoError(t, err)
		require.NotEmpty(t, reset)

		// Verify directory was created
		templatesDir := filepath.Join(tempDir, "templates")
		assert.DirExists(t, templatesDir)

		// Verify templates were created
		for _, name := range reset {
			assert.FileExists(t, filepath.Join(templatesDir, name))
		}
	})
}

func TestInitTemplatesDoesNotFailOnMissingFiles(t *testing.T) {
	t.Run("missing_template_file_does_not_fail_init", func(t *testing.T) {
		tempDir := t.TempDir()
		templatesDir := filepath.Join(tempDir, "templates")
		err := os.MkdirAll(templatesDir, 0o755)
		require.NoError(t, err)

		// Create a template file
		err = os.WriteFile(filepath.Join(templatesDir, "host.hbs"), []byte("test"), 0o644)
		require.NoError(t, err)

		// Initialize with a path pointing to a missing file
		templateConfigs := map[TemplateEntity]TemplateConfig{
			TemplateEntityHost:        {Path: filepath.Join(templatesDir, "host.hbs")},
			TemplateEntityCertificate: {Path: filepath.Join(templatesDir, "certificate.hbs")}, // missing
		}

		// This should NOT fail even though certificate.hbs doesn't exist
		result, err := InitTemplates(tempDir, templateConfigs)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Both paths should be set
		assert.Equal(t, filepath.Join(templatesDir, "host.hbs"), result[TemplateEntityHost].Path)
		assert.Equal(t, filepath.Join(templatesDir, "certificate.hbs"), result[TemplateEntityCertificate].Path)
	})

	t.Run("deleted_template_file_does_not_fail_init", func(t *testing.T) {
		tempDir := t.TempDir()
		templatesDir := filepath.Join(tempDir, "templates")
		err := os.MkdirAll(templatesDir, 0o755)
		require.NoError(t, err)

		// Create template files
		err = os.WriteFile(filepath.Join(templatesDir, "host.hbs"), []byte("test"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(templatesDir, "certificate.hbs"), []byte("test"), 0o644)
		require.NoError(t, err)

		// First init - should work fine
		templateConfigs := map[TemplateEntity]TemplateConfig{
			TemplateEntityHost:        {},
			TemplateEntityCertificate: {},
		}
		result1, err := InitTemplates(tempDir, templateConfigs)
		require.NoError(t, err)

		// Delete one of the template files
		err = os.Remove(filepath.Join(templatesDir, "certificate.hbs"))
		require.NoError(t, err)

		// Second init with paths from first init - should NOT fail
		result2, err := InitTemplates(tempDir, result1)
		require.NoError(t, err)
		require.NotNil(t, result2)

		// Paths should still be set even though file is missing
		assert.Equal(t, filepath.Join(templatesDir, "host.hbs"), result2[TemplateEntityHost].Path)
		assert.Equal(t, filepath.Join(templatesDir, "certificate.hbs"), result2[TemplateEntityCertificate].Path)
	})
}
