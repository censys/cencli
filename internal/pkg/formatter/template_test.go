package formatter

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintDataWithTemplate_Smoke(t *testing.T) {
	dir := t.TempDir()
	// Create a minimal template file
	tpl := filepath.Join(dir, "host.hbs")
	if err := os.WriteFile(tpl, []byte("IP: {{orange ip}}"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	// capture stdout
	var buf bytes.Buffer
	old := Stdout
	Stdout = &buf
	defer func() { Stdout = old }()

	data := map[string]any{"ip": "127.0.0.1"}
	if err := PrintDataWithTemplate(tpl, true, data); err != nil {
		t.Fatalf("PrintDataWithTemplate error: %v", err)
	}
	assert.Contains(t, buf.String(), "127.0.0.1")
}

func TestPrintTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templatePath func(t *testing.T, tempDir string) string
		data         func() any
		assert       func(t *testing.T, stdout string)
		assertErr    func(t *testing.T, err error)
	}{
		{
			name: "successful template rendering with map data",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "host.hbs")
				templateContent := `Host: {{name}}
IP: {{ip}}
Status: {{status}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)

				return templatePath
			},
			data: func() any {
				return map[string]interface{}{
					"name":   "example.com",
					"ip":     "192.168.1.1",
					"status": "active",
				}
			},
			assert: func(t *testing.T, stdout string) {
				expected := `Host: example.com
IP: 192.168.1.1
Status: active`
				assert.Equal(t, expected, stdout)
			},
		},
		{
			name: "successful template rendering with slice of structs",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "host.hbs")
				templateContent := `{{#each results}}
Host: {{name}} - {{ip}}
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)

				return templatePath
			},
			data: func() any {
				type Host struct {
					Name string `json:"name"`
					IP   string `json:"ip"`
				}
				return map[string]interface{}{
					"results": []Host{
						{Name: "host1.com", IP: "192.168.1.1"},
						{Name: "host2.com", IP: "192.168.1.2"},
					},
				}
			},
			assert: func(t *testing.T, stdout string) {
				expected := `Host: host1.com - 192.168.1.1
Host: host2.com - 192.168.1.2
`
				assert.Equal(t, expected, stdout)
			},
		},
		{
			name: "successful template rendering with complex nested data",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "host.hbs")
				templateContent := `{{#each hosts}}
Host: {{name}}
{{#if services}}
Services:
{{#each services}}
  - {{name}} on port {{port}}
{{/each}}
{{/if}}
---
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)

				return templatePath
			},
			data: func() any {
				type Service struct {
					Name string `json:"name"`
					Port int    `json:"port"`
				}
				type Host struct {
					Name     string    `json:"name"`
					Services []Service `json:"services"`
				}
				return map[string]interface{}{
					"hosts": []Host{
						{
							Name: "web.example.com",
							Services: []Service{
								{Name: "http", Port: 80},
								{Name: "https", Port: 443},
							},
						},
						{
							Name:     "db.example.com",
							Services: []Service{{Name: "mysql", Port: 3306}},
						},
					},
				}
			},
			assert: func(t *testing.T, stdout string) {
				expected := `Host: web.example.com
Services:
  - http on port 80
  - https on port 443
---
Host: db.example.com
Services:
  - mysql on port 3306
---
`
				assert.Equal(t, expected, stdout)
			},
		},
		{
			name: "error: template file not found",
			templatePath: func(t *testing.T, tempDir string) string {
				nonExistentPath := filepath.Join(tempDir, "nonexistent.hbs")
				return nonExistentPath
			},
			data: func() any {
				return map[string]interface{}{"test": "data"}
			},
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				var templateFailureErr TemplateFailureError
				assert.True(t, errors.As(err, &templateFailureErr))
				assert.Contains(t, err.Error(), "failed to render template")
				assert.Contains(t, err.Error(), "no such file or directory")
			},
		},
		{
			name: "error: invalid template syntax",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "invalid.hbs")
				// Invalid handlebars syntax - unclosed block
				templateContent := `{{#each items}}
Name: {{name}}
{{#if active}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)

				return templatePath
			},
			data: func() any {
				return map[string]interface{}{
					"items": []map[string]interface{}{
						{"name": "test", "active": true},
					},
				}
			},
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				var templateFailureErr TemplateFailureError
				assert.True(t, errors.As(err, &templateFailureErr))
				assert.Contains(t, err.Error(), "failed to render template")
			},
		},
		{
			name: "error: template file is empty",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "empty.hbs")
				err := os.WriteFile(templatePath, []byte(""), 0o644)
				require.NoError(t, err)

				return templatePath
			},
			data: func() any {
				return map[string]interface{}{"test": "data"}
			},
			assert: func(t *testing.T, stdout string) {
				// Empty template should render empty string
				assert.Equal(t, "", stdout)
			},
		},
		{
			name: "template rendering with nil data",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "host.hbs")
				templateContent := `No data available`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)

				return templatePath
			},
			data: func() any {
				return nil
			},
			assert: func(t *testing.T, stdout string) {
				assert.Equal(t, "No data available", stdout)
			},
		},
		{
			name: "slice of nested structs with multiple levels",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "host.hbs")
				templateContent := `{{#each servers}}
Server: {{name}} ({{region}})
{{#if databases}}
Databases:
{{#each databases}}
  - {{name}} ({{type}}) - {{status}}
  {{#each replicas}}
  {{#if @first}}
  Replicas:
  {{/if}}
    * {{host}}:{{port}} - {{lag}}ms
  {{/each}}
{{/each}}
{{/if}}
---
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)

				return templatePath
			},
			data: func() any {
				type Replica struct {
					Host string `json:"host"`
					Port int    `json:"port"`
					Lag  int    `json:"lag"`
				}
				type Database struct {
					Name     string    `json:"name"`
					Type     string    `json:"type"`
					Status   string    `json:"status"`
					Replicas []Replica `json:"replicas"`
				}
				type Server struct {
					Name      string     `json:"name"`
					Region    string     `json:"region"`
					Databases []Database `json:"databases"`
				}
				return map[string]interface{}{
					"servers": []Server{
						{
							Name:   "db-primary",
							Region: "us-east-1",
							Databases: []Database{
								{
									Name:   "users",
									Type:   "postgresql",
									Status: "healthy",
									Replicas: []Replica{
										{Host: "replica1.db.com", Port: 5432, Lag: 10},
										{Host: "replica2.db.com", Port: 5432, Lag: 15},
									},
								},
								{
									Name:     "cache",
									Type:     "redis",
									Status:   "healthy",
									Replicas: []Replica{},
								},
							},
						},
						{
							Name:   "db-secondary",
							Region: "us-west-2",
							Databases: []Database{
								{
									Name:   "analytics",
									Type:   "clickhouse",
									Status: "degraded",
									Replicas: []Replica{
										{Host: "analytics-replica.db.com", Port: 9000, Lag: 100},
									},
								},
							},
						},
					},
				}
			},
			assert: func(t *testing.T, stdout string) {
				expected := `Server: db-primary (us-east-1)
Databases:
  - users (postgresql) - healthy
  Replicas:
    * replica1.db.com:5432 - 10ms
    * replica2.db.com:5432 - 15ms
  - cache (redis) - healthy
---
Server: db-secondary (us-west-2)
Databases:
  - analytics (clickhouse) - degraded
  Replicas:
    * analytics-replica.db.com:9000 - 100ms
---
`
				assert.Equal(t, expected, stdout)
			},
		},
		{
			name: "slice of structs with optional nested arrays",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "host.hbs")
				templateContent := `{{#each applications}}
App: {{name}} v{{version}}
{{#if dependencies}}
Dependencies:
{{#each dependencies}}
  - {{name}}@{{version}}{{#if optional}} (optional){{/if}}
{{/each}}
{{else}}
No dependencies
{{/if}}
{{#if environments}}
Environments:
{{#each environments}}
  - {{name}}: {{url}}{{#if active}} [ACTIVE]{{/if}}
{{/each}}
{{/if}}
---
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)

				return templatePath
			},
			data: func() any {
				type Dependency struct {
					Name     string `json:"name"`
					Version  string `json:"version"`
					Optional bool   `json:"optional"`
				}
				type Environment struct {
					Name   string `json:"name"`
					URL    string `json:"url"`
					Active bool   `json:"active"`
				}
				type Application struct {
					Name         string        `json:"name"`
					Version      string        `json:"version"`
					Dependencies []Dependency  `json:"dependencies"`
					Environments []Environment `json:"environments"`
				}
				return map[string]interface{}{
					"applications": []Application{
						{
							Name:    "web-frontend",
							Version: "2.1.0",
							Dependencies: []Dependency{
								{Name: "react", Version: "18.2.0", Optional: false},
								{Name: "lodash", Version: "4.17.21", Optional: true},
							},
							Environments: []Environment{
								{Name: "staging", URL: "https://staging.example.com", Active: false},
								{Name: "production", URL: "https://example.com", Active: true},
							},
						},
						{
							Name:         "api-service",
							Version:      "1.5.2",
							Dependencies: []Dependency{},
							Environments: []Environment{
								{Name: "production", URL: "https://api.example.com", Active: true},
							},
						},
					},
				}
			},
			assert: func(t *testing.T, stdout string) {
				expected := `App: web-frontend v2.1.0
Dependencies:
  - react@18.2.0
  - lodash@4.17.21 (optional)
Environments:
  - staging: https://staging.example.com
  - production: https://example.com [ACTIVE]
---
App: api-service v1.5.2
No dependencies
Environments:
  - production: https://api.example.com [ACTIVE]
---
`
				assert.Equal(t, expected, stdout)
			},
		},
		{
			name: "deeply nested slice with conditional rendering",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "host.hbs")
				templateContent := `{{#each teams}}
Team: {{name}}
{{#each members}}
  - {{name}} ({{role}})
  {{#each tasks}}
    * {{title}} - {{status}}
  {{/each}}
{{/each}}
---
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)

				return templatePath
			},
			data: func() any {
				type Task struct {
					Title  string `json:"title"`
					Status string `json:"status"`
				}
				type Member struct {
					Name  string `json:"name"`
					Role  string `json:"role"`
					Tasks []Task `json:"tasks"`
				}
				type Team struct {
					Name    string   `json:"name"`
					Members []Member `json:"members"`
				}
				return map[string]interface{}{
					"teams": []Team{
						{
							Name: "Development",
							Members: []Member{
								{
									Name: "Alice",
									Role: "Lead",
									Tasks: []Task{
										{Title: "Code Review", Status: "done"},
										{Title: "Architecture", Status: "in-progress"},
									},
								},
								{
									Name:  "Bob",
									Role:  "Developer",
									Tasks: []Task{},
								},
							},
						},
						{
							Name: "QA",
							Members: []Member{
								{
									Name: "Carol",
									Role: "Tester",
									Tasks: []Task{
										{Title: "Test Plan", Status: "pending"},
									},
								},
							},
						},
					},
				}
			},
			assert: func(t *testing.T, stdout string) {
				expected := `Team: Development
  - Alice (Lead)
    * Code Review - done
    * Architecture - in-progress
  - Bob (Developer)
---
Team: QA
  - Carol (Tester)
    * Test Plan - pending
---
`
				assert.Equal(t, expected, stdout)
			},
		},
		{
			name: "error: template with malformed handlebars syntax",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "malformed.hbs")
				// Malformed template with mismatched braces and invalid helper syntax
				templateContent := `{{#each hosts}
Host: {{name}}
{{#invalid_helper_syntax}}
  Invalid: {{unclosed_variable
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)

				return templatePath
			},
			data: func() any {
				return map[string]interface{}{
					"hosts": []map[string]interface{}{
						{"name": "server1.example.com"},
					},
				}
			},
			assertErr: func(t *testing.T, err error) {
				require.Error(t, err)
				var templateFailureErr TemplateFailureError
				assert.True(t, errors.As(err, &templateFailureErr))
				assert.Contains(t, err.Error(), "failed to render template")
				assert.Contains(t, err.Error(), "malformed.hbs")
				// The error should contain handlebars parsing details
				assert.Contains(t, err.Error(), "Parse error")
				assert.Contains(t, err.Error(), "Lexer error")
			},
		},
		{
			name: "error: template accessing non-existent keys",
			templatePath: func(t *testing.T, tempDir string) string {
				templatePath := filepath.Join(tempDir, "missing_keys.hbs")
				// Template tries to access keys that don't exist in the data
				templateContent := `{{#each servers}}
Server: {{name}}
IP: {{ip_address}}
Port: {{port}}
{{#if services}}
Services:
{{#each services}}
  - {{service_name}} on {{service_port}}
  - Description: {{description}}
  - Owner: {{owner.name}} ({{owner.email}})
{{/each}}
{{/if}}
---
{{/each}}`
				err := os.WriteFile(templatePath, []byte(templateContent), 0o644)
				require.NoError(t, err)

				return templatePath
			},
			data: func() any {
				// Data is missing many of the keys referenced in the template
				return map[string]interface{}{
					"servers": []map[string]interface{}{
						{
							"name": "web-server-1",
							// Missing: ip_address, port
							"services": []map[string]interface{}{
								{
									"service_name": "nginx",
									// Missing: service_port, description, owner
								},
								{
									"service_name": "redis",
									"service_port": 6379,
									// Missing: description, owner
								},
							},
						},
						{
							"name":       "db-server-1",
							"ip_address": "192.168.1.100",
							// Missing: port, services is nil (not empty array)
						},
					},
				}
			},
			assert: func(t *testing.T, stdout string) {
				// Handlebars should render successfully but with empty values for missing keys
				expected := `Server: web-server-1
IP: 
Port: 
Services:
  - nginx on 
  - Description: 
  - Owner:  ()
  - redis on 6379
  - Description: 
  - Owner:  ()
---
Server: db-server-1
IP: 192.168.1.100
Port: 
---
`
				assert.Equal(t, expected, stdout)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tempDir := t.TempDir()

			// Capture stdout
			stdout := &bytes.Buffer{}
			originalStdout := Stdout
			Stdout = stdout
			defer func() { Stdout = originalStdout }()

			// Execute
			err := PrintDataWithTemplate(tc.templatePath(t, tempDir), false, tc.data())

			// Assert
			if tc.assertErr != nil {
				tc.assertErr(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.assert != nil {
				tc.assert(t, stdout.String())
			}
		})
	}
}

func TestTemplateFailureError(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		err           error
		expectedError string
		expectedTitle string
		expectedUsage bool
	}{
		{
			name:          "error with path",
			path:          "/path/to/template.hbs",
			err:           fmt.Errorf("file not found"),
			expectedError: "failed to render template at path '/path/to/template.hbs': file not found",
			expectedTitle: "Template Failure",
			expectedUsage: false,
		},
		{
			name:          "error without path",
			path:          "",
			err:           fmt.Errorf("parsing error"),
			expectedError: "failed to render template: parsing error",
			expectedTitle: "Template Failure",
			expectedUsage: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := newTemplateFailureError(tc.path, tc.err)

			assert.Equal(t, tc.expectedError, err.Error())
			assert.Equal(t, tc.expectedTitle, err.Title())
			assert.Equal(t, tc.expectedUsage, err.ShouldPrintUsage())

			// Test that it implements the interface
			var templateFailureErr TemplateFailureError
			assert.True(t, errors.As(err, &templateFailureErr))
		})
	}
}

func TestPrintDataWithTemplate_MissingFile(t *testing.T) {
	// Test that using a missing template file produces an error
	nonexistentPath := filepath.Join(t.TempDir(), "nonexistent.hbs")

	// capture stdout
	var buf bytes.Buffer
	old := Stdout
	Stdout = &buf
	defer func() { Stdout = old }()

	data := map[string]any{"ip": "127.0.0.1"}
	err := PrintDataWithTemplate(nonexistentPath, true, data)

	// Should error when trying to read the missing file
	require.Error(t, err)
	var templateFailureErr TemplateFailureError
	assert.ErrorAs(t, err, &templateFailureErr)
	assert.Contains(t, err.Error(), "failed to render template")
}
