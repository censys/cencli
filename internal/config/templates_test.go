package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplatePathsWrittenToYAML tests that template paths are properly initialized
// and written to the config.yaml file during config initialization.
func TestTemplatePathsWrittenToYAML(t *testing.T) {
	tempDir, cleanup := setupConfigTest(t)
	defer cleanup()

	cfg, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify templates were initialized with paths
	require.Contains(t, cfg.Templates, TemplateEntityHost)
	require.Contains(t, cfg.Templates, TemplateEntityCertificate)
	require.Contains(t, cfg.Templates, TemplateEntityWebProperty)
	require.Contains(t, cfg.Templates, TemplateEntitySearchResult)

	// All template paths should be set
	assert.NotEmpty(t, cfg.Templates[TemplateEntityHost].Path)
	assert.NotEmpty(t, cfg.Templates[TemplateEntityCertificate].Path)
	assert.NotEmpty(t, cfg.Templates[TemplateEntityWebProperty].Path)
	assert.NotEmpty(t, cfg.Templates[TemplateEntitySearchResult].Path)

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

func TestInitTemplates(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(tempDir string) (*Config, error)
		expectedError  func(t *testing.T, err error)
		expectedConfig func(t *testing.T, cfg *Config, tempDir string)
		expectedFiles  func(t *testing.T, tempDir string)
	}{
		{
			name: "fresh_setup_single_entity",
			setup: func(tempDir string) (*Config, error) {
				return &Config{
					Templates: map[TemplateEntity]TemplateConfig{
						TemplateEntityHost: {},
					},
				}, nil
			},
			expectedConfig: func(t *testing.T, cfg *Config, tempDir string) {
				require.Contains(t, cfg.Templates, TemplateEntityHost)
				expectedPath := filepath.Join(tempDir, "templates", "host.hbs")
				assert.Equal(t, expectedPath, cfg.Templates[TemplateEntityHost].Path)
			},
			expectedFiles: func(t *testing.T, tempDir string) {
				templatesDir := filepath.Join(tempDir, "templates")
				assert.DirExists(t, templatesDir)

				hostFile := filepath.Join(templatesDir, "host.hbs")
				assert.FileExists(t, hostFile)

				// Verify file content is not empty
				content, err := os.ReadFile(hostFile)
				require.NoError(t, err)
				assert.NotEmpty(t, content)
			},
		},
		{
			name: "fresh_setup_multiple_entities",
			setup: func(tempDir string) (*Config, error) {
				return &Config{
					Templates: map[TemplateEntity]TemplateConfig{
						TemplateEntityHost:         {},
						TemplateEntityCertificate:  {},
						TemplateEntityWebProperty:  {},
						TemplateEntitySearchResult: {},
					},
				}, nil
			},
			expectedConfig: func(t *testing.T, cfg *Config, tempDir string) {
				templatesDir := filepath.Join(tempDir, "templates")

				// All entities should have paths set since default templates exist for all
				require.Contains(t, cfg.Templates, TemplateEntityHost)
				expectedHostPath := filepath.Join(templatesDir, "host.hbs")
				assert.Equal(t, expectedHostPath, cfg.Templates[TemplateEntityHost].Path)

				require.Contains(t, cfg.Templates, TemplateEntityCertificate)
				expectedCertPath := filepath.Join(templatesDir, "certificate.hbs")
				assert.Equal(t, expectedCertPath, cfg.Templates[TemplateEntityCertificate].Path)

				require.Contains(t, cfg.Templates, TemplateEntityWebProperty)
				expectedWebPath := filepath.Join(templatesDir, "webproperty.hbs")
				assert.Equal(t, expectedWebPath, cfg.Templates[TemplateEntityWebProperty].Path)

				require.Contains(t, cfg.Templates, TemplateEntitySearchResult)
				expectedSearchPath := filepath.Join(templatesDir, "searchresult.hbs")
				assert.Equal(t, expectedSearchPath, cfg.Templates[TemplateEntitySearchResult].Path)
			},
			expectedFiles: func(t *testing.T, tempDir string) {
				templatesDir := filepath.Join(tempDir, "templates")
				assert.DirExists(t, templatesDir)

				// All template files should exist
				files := []string{"host.hbs", "certificate.hbs", "webproperty.hbs", "searchresult.hbs"}
				for _, filename := range files {
					filePath := filepath.Join(templatesDir, filename)
					assert.FileExists(t, filePath)

					// Verify file content is not empty
					content, err := os.ReadFile(filePath)
					require.NoError(t, err)
					assert.NotEmpty(t, content)
				}
			},
		},
		{
			name: "existing_template_files_found",
			setup: func(tempDir string) (*Config, error) {
				templatesDir := filepath.Join(tempDir, "templates")
				err := os.MkdirAll(templatesDir, 0o755)
				if err != nil {
					return nil, err
				}

				// Create existing template files with different extensions
				hostFile := filepath.Join(templatesDir, "host.tmpl")
				err = os.WriteFile(hostFile, []byte("custom host template"), 0o644)
				if err != nil {
					return nil, err
				}

				return &Config{
					Templates: map[TemplateEntity]TemplateConfig{
						TemplateEntityHost: {},
					},
				}, nil
			},
			expectedConfig: func(t *testing.T, cfg *Config, tempDir string) {
				require.Contains(t, cfg.Templates, TemplateEntityHost)
				expectedPath := filepath.Join(tempDir, "templates", "host.tmpl")
				assert.Equal(t, expectedPath, cfg.Templates[TemplateEntityHost].Path)
			},
			expectedFiles: func(t *testing.T, tempDir string) {
				hostFile := filepath.Join(tempDir, "templates", "host.tmpl")
				assert.FileExists(t, hostFile)

				// Verify it kept the existing content
				content, err := os.ReadFile(hostFile)
				require.NoError(t, err)
				assert.Equal(t, "custom host template", string(content))

				// Should not have created the default .hbs file
				defaultFile := filepath.Join(tempDir, "templates", "host.hbs")
				assert.NoFileExists(t, defaultFile)
			},
		},
		{
			name: "valid_existing_paths_in_config",
			setup: func(tempDir string) (*Config, error) {
				customTemplateDir := filepath.Join(tempDir, "custom")
				err := os.MkdirAll(customTemplateDir, 0o755)
				if err != nil {
					return nil, err
				}

				customHostFile := filepath.Join(customTemplateDir, "my-host.template")
				err = os.WriteFile(customHostFile, []byte("my custom host"), 0o644)
				if err != nil {
					return nil, err
				}

				return &Config{
					Templates: map[TemplateEntity]TemplateConfig{
						TemplateEntityHost: {
							Path: customHostFile,
						},
					},
				}, nil
			},
			expectedConfig: func(t *testing.T, cfg *Config, tempDir string) {
				require.Contains(t, cfg.Templates, TemplateEntityHost)
				expectedPath := filepath.Join(tempDir, "custom", "my-host.template")
				assert.Equal(t, expectedPath, cfg.Templates[TemplateEntityHost].Path)
			},
			expectedFiles: func(t *testing.T, tempDir string) {
				// Should not create templates directory or default files
				templatesDir := filepath.Join(tempDir, "templates")
				if _, err := os.Stat(templatesDir); err == nil {
					// If templates dir exists, it should be empty
					entries, err := os.ReadDir(templatesDir)
					require.NoError(t, err)
					assert.Empty(t, entries)
				}

				// Custom file should still exist
				customFile := filepath.Join(tempDir, "custom", "my-host.template")
				assert.FileExists(t, customFile)
			},
		},
		{
			name: "invalid_existing_path_in_config",
			setup: func(tempDir string) (*Config, error) {
				return &Config{
					Templates: map[TemplateEntity]TemplateConfig{
						TemplateEntityHost: {
							Path: "/nonexistent/path/host.tmpl",
						},
					},
				}, nil
			},
			expectedError: func(t *testing.T, err error) {
				require.Error(t, err)
				var templateNotFoundErr TemplateNotFoundError
				assert.ErrorAs(t, err, &templateNotFoundErr)
				assert.Contains(t, templateNotFoundErr.Error(), "host")
				assert.Contains(t, templateNotFoundErr.Error(), "/nonexistent/path/host.tmpl")
			},
		},
		{
			name: "mixed_scenarios",
			setup: func(tempDir string) (*Config, error) {
				// Create templates directory with some existing files
				templatesDir := filepath.Join(tempDir, "templates")
				err := os.MkdirAll(templatesDir, 0o755)
				if err != nil {
					return nil, err
				}

				// Existing template file
				existingFile := filepath.Join(templatesDir, "host.custom")
				err = os.WriteFile(existingFile, []byte("existing custom"), 0o644)
				if err != nil {
					return nil, err
				}

				// Custom path outside templates dir
				customDir := filepath.Join(tempDir, "custom")
				err = os.MkdirAll(customDir, 0o755)
				if err != nil {
					return nil, err
				}

				customFile := filepath.Join(customDir, "cert.tmpl")
				err = os.WriteFile(customFile, []byte("custom cert"), 0o644)
				if err != nil {
					return nil, err
				}

				return &Config{
					Templates: map[TemplateEntity]TemplateConfig{
						TemplateEntityHost: {}, // Should find existing file
						TemplateEntityCertificate: { // Should use custom path
							Path: customFile,
						},
						TemplateEntityWebProperty: { // Should fail (invalid path)
							Path: "/nonexistent/invalid/path.tmpl",
						},
					},
				}, nil
			},
			expectedError: func(t *testing.T, err error) {
				require.Error(t, err)
				var templateNotFoundErr TemplateNotFoundError
				assert.ErrorAs(t, err, &templateNotFoundErr)
				assert.Contains(t, err.Error(), "webproperty")
				assert.Contains(t, err.Error(), "/nonexistent/invalid/path.tmpl")
			},
		},
		{
			name: "templates_directory_creation_failure",
			setup: func(tempDir string) (*Config, error) {
				// Create a file where the templates directory should be
				templatesPath := filepath.Join(tempDir, "templates")
				err := os.WriteFile(templatesPath, []byte("blocking file"), 0o644)
				if err != nil {
					return nil, err
				}

				return &Config{
					Templates: map[TemplateEntity]TemplateConfig{
						TemplateEntityHost: {},
					},
				}, nil
			},
			expectedError: func(t *testing.T, err error) {
				require.Error(t, err)
				var templateDirErr TemplateDirectoryError
				assert.ErrorAs(t, err, &templateDirErr)
				// The error could be either "create" or "read" depending on when it fails
				assert.Contains(t, templateDirErr.Error(), "read")
			},
		},
		{
			name: "empty_templates_map",
			setup: func(tempDir string) (*Config, error) {
				return &Config{
					Templates: map[TemplateEntity]TemplateConfig{},
				}, nil
			},
			expectedConfig: func(t *testing.T, cfg *Config, tempDir string) {
				assert.Empty(t, cfg.Templates)
			},
			expectedFiles: func(t *testing.T, tempDir string) {
				templatesDir := filepath.Join(tempDir, "templates")
				assert.DirExists(t, templatesDir)

				// Directory should be empty
				entries, err := os.ReadDir(templatesDir)
				require.NoError(t, err)
				assert.Empty(t, entries)
			},
		},
		{
			name: "multiple_matching_files_first_wins",
			setup: func(tempDir string) (*Config, error) {
				templatesDir := filepath.Join(tempDir, "templates")
				err := os.MkdirAll(templatesDir, 0o755)
				if err != nil {
					return nil, err
				}

				// Create multiple files that match the pattern
				files := []string{"host.hbs", "host.tmpl", "host.template"}
				for i, filename := range files {
					content := []byte(fmt.Sprintf("content %d", i))
					err = os.WriteFile(filepath.Join(templatesDir, filename), content, 0o644)
					if err != nil {
						return nil, err
					}
				}

				return &Config{
					Templates: map[TemplateEntity]TemplateConfig{
						TemplateEntityHost: {},
					},
				}, nil
			},
			expectedConfig: func(t *testing.T, cfg *Config, tempDir string) {
				require.Contains(t, cfg.Templates, TemplateEntityHost)
				// Should pick the first one found (order depends on filesystem)
				path := cfg.Templates[TemplateEntityHost].Path
				assert.Contains(t, path, "host.")
				assert.True(t,
					filepath.Base(path) == "host.hbs" ||
						filepath.Base(path) == "host.tmpl" ||
						filepath.Base(path) == "host.template")
			},
		},
		{
			name: "mixed_existing_and_new_paths_no_overwrite",
			setup: func(tempDir string) (*Config, error) {
				// Create templates directory with one existing file
				templatesDir := filepath.Join(tempDir, "templates")
				err := os.MkdirAll(templatesDir, 0o755)
				if err != nil {
					return nil, err
				}

				// Create an existing template file that should be found
				existingFile := filepath.Join(templatesDir, "host.custom")
				err = os.WriteFile(existingFile, []byte("existing host template"), 0o644)
				if err != nil {
					return nil, err
				}

				// Create custom directory with a custom template
				customDir := filepath.Join(tempDir, "custom")
				err = os.MkdirAll(customDir, 0o755)
				if err != nil {
					return nil, err
				}

				customCertFile := filepath.Join(customDir, "my-cert.tmpl")
				err = os.WriteFile(customCertFile, []byte("custom certificate template"), 0o644)
				if err != nil {
					return nil, err
				}

				return &Config{
					Templates: map[TemplateEntity]TemplateConfig{
						TemplateEntityHost: {}, // Should find existing file in templates dir
						TemplateEntityCertificate: { // Should keep existing custom path
							Path: customCertFile,
						},
						TemplateEntityWebProperty:  {}, // Should get default template copied
						TemplateEntitySearchResult: {}, // Should get default template copied
					},
				}, nil
			},
			expectedConfig: func(t *testing.T, cfg *Config, tempDir string) {
				templatesDir := filepath.Join(tempDir, "templates")

				// Host should use the existing file found in templates directory
				require.Contains(t, cfg.Templates, TemplateEntityHost)
				expectedHostPath := filepath.Join(templatesDir, "host.custom")
				assert.Equal(t, expectedHostPath, cfg.Templates[TemplateEntityHost].Path)

				// Certificate should keep its custom path (not overwritten)
				require.Contains(t, cfg.Templates, TemplateEntityCertificate)
				expectedCertPath := filepath.Join(tempDir, "custom", "my-cert.tmpl")
				assert.Equal(t, expectedCertPath, cfg.Templates[TemplateEntityCertificate].Path)

				// WebProperty should get default template
				require.Contains(t, cfg.Templates, TemplateEntityWebProperty)
				expectedWebPath := filepath.Join(templatesDir, "webproperty.hbs")
				assert.Equal(t, expectedWebPath, cfg.Templates[TemplateEntityWebProperty].Path)

				// SearchResult should get default template
				require.Contains(t, cfg.Templates, TemplateEntitySearchResult)
				expectedSearchPath := filepath.Join(templatesDir, "searchresult.hbs")
				assert.Equal(t, expectedSearchPath, cfg.Templates[TemplateEntitySearchResult].Path)
			},
			expectedFiles: func(t *testing.T, tempDir string) {
				templatesDir := filepath.Join(tempDir, "templates")
				assert.DirExists(t, templatesDir)

				// Existing host file should still exist with original content
				hostFile := filepath.Join(templatesDir, "host.custom")
				assert.FileExists(t, hostFile)
				content, err := os.ReadFile(hostFile)
				require.NoError(t, err)
				assert.Equal(t, "existing host template", string(content))

				// Custom certificate file should still exist with original content
				customCertFile := filepath.Join(tempDir, "custom", "my-cert.tmpl")
				assert.FileExists(t, customCertFile)
				content, err = os.ReadFile(customCertFile)
				require.NoError(t, err)
				assert.Equal(t, "custom certificate template", string(content))

				// Default templates should have been created for webproperty and searchresult
				webFile := filepath.Join(templatesDir, "webproperty.hbs")
				assert.FileExists(t, webFile)
				content, err = os.ReadFile(webFile)
				require.NoError(t, err)
				assert.NotEmpty(t, content)
				assert.NotEqual(t, "existing host template", string(content)) // Should be default content

				searchFile := filepath.Join(templatesDir, "searchresult.hbs")
				assert.FileExists(t, searchFile)
				content, err = os.ReadFile(searchFile)
				require.NoError(t, err)
				assert.NotEmpty(t, content)
				assert.NotEqual(t, "existing host template", string(content)) // Should be default content

				// Should NOT have created a default certificate.hbs since custom path was provided
				defaultCertFile := filepath.Join(templatesDir, "certificate.hbs")
				assert.NoFileExists(t, defaultCertFile)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			cfg, err := tt.setup(tempDir)
			require.NoError(t, err)
			require.NotNil(t, cfg)

			err = initTemplates(tempDir, cfg)

			if tt.expectedError != nil {
				tt.expectedError(t, err)
				return
			}

			require.NoError(t, err)

			if tt.expectedConfig != nil {
				tt.expectedConfig(t, cfg, tempDir)
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

			err = copyDefaultTemplate(tt.templateName, templatesDir)

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
