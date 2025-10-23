package helpers

import (
	"testing"

	handlebars "github.com/aymerick/raymond"
	"github.com/stretchr/testify/assert"
)

func TestParseLookupOperation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected lookupOperation
	}{
		{"host operation", "host", lookupOperationHost},
		{"certificate operation", "certificate", lookupOperationCertificate},
		{"webproperty operation", "webproperty", lookupOperationWebProperty},
		{"invalid operation", "invalid", ""},
		{"empty string", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseLookupOperation(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLookupOperation_String(t *testing.T) {
	tests := []struct {
		name     string
		op       lookupOperation
		expected string
	}{
		{"host", lookupOperationHost, "host"},
		{"certificate", lookupOperationCertificate, "certificate"},
		{"webproperty", lookupOperationWebProperty, "webproperty"},
		{"empty", lookupOperation(""), ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.op.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLookupURLHelper_WithoutRender(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		id       string
		expected string
	}{
		{
			name:     "host lookup",
			op:       "host",
			id:       "192.168.1.1",
			expected: "https://platform.censys.io/hosts/192.168.1.1",
		},
		{
			name:     "certificate lookup",
			op:       "certificate",
			id:       "abc123def456",
			expected: "https://platform.censys.io/certificates/abc123def456",
		},
		{
			name:     "webproperty lookup",
			op:       "webproperty",
			id:       "example.com:443",
			expected: "https://platform.censys.io/webproperties/example.com:443",
		},
		{
			name:     "invalid operation",
			op:       "invalid",
			id:       "test",
			expected: "",
		},
		{
			name:     "empty operation",
			op:       "",
			id:       "test",
			expected: "",
		},
	}

	helper := NewLookupURLHelper(false)
	fn := helper.Function().(func(string, string) string)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := fn(tc.op, tc.id)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLookupURLHelper_WithRender(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		id       string
		contains []string
	}{
		{
			name: "host lookup rendered",
			op:   "host",
			id:   "8.8.8.8",
			contains: []string{
				"https://platform.censys.io/hosts/8.8.8.8",
			},
		},
		{
			name: "certificate lookup rendered",
			op:   "certificate",
			id:   "cert123",
			contains: []string{
				"https://platform.censys.io/certificates/cert123",
			},
		},
		{
			name: "webproperty lookup rendered",
			op:   "webproperty",
			id:   "test.com:80",
			contains: []string{
				"https://platform.censys.io/webproperties/test.com:80",
			},
		},
	}

	helper := NewLookupURLHelper(true)
	fn := helper.Function().(func(string, string) string)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := fn(tc.op, tc.id)
			// When rendered, it should contain the URL (may have terminal escape codes)
			for _, expected := range tc.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestLookupURLHelper_Name(t *testing.T) {
	helper := NewLookupURLHelper(false)
	assert.Equal(t, "platform_lookup_url", helper.Name())
}

func TestLookupURLHelper_Interface(t *testing.T) {
	helper := NewLookupURLHelper(false)
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}

func TestLookupURLHelper_InTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     any
		render   bool
		expected string
	}{
		{
			name:     "host lookup in template",
			template: `Host: {{platform_lookup_url "host" ip}}`,
			data:     map[string]any{"ip": "1.2.3.4"},
			render:   false,
			expected: "Host: https://platform.censys.io/hosts/1.2.3.4",
		},
		{
			name:     "certificate lookup in template",
			template: `Certificate: {{platform_lookup_url "certificate" certID}}`,
			data:     map[string]any{"certID": "abc123"},
			render:   false,
			expected: "Certificate: https://platform.censys.io/certificates/abc123",
		},
		{
			name:     "webproperty lookup in template",
			template: `Web Property: {{platform_lookup_url "webproperty" hostport}}`,
			data:     map[string]any{"hostport": "example.com:443"},
			render:   false,
			expected: "Web Property: https://platform.censys.io/webproperties/example.com:443",
		},
		{
			name:     "multiple lookups in loop",
			template: `{{#each hosts}}Host {{ip}}: {{platform_lookup_url "host" ip}}, {{/each}}`,
			data: map[string]any{
				"hosts": []map[string]any{
					{"ip": "10.0.0.1"},
					{"ip": "10.0.0.2"},
				},
			},
			render:   false,
			expected: "Host 10.0.0.1: https://platform.censys.io/hosts/10.0.0.1, Host 10.0.0.2: https://platform.censys.io/hosts/10.0.0.2, ",
		},
		{
			name:     "invalid operation returns empty",
			template: `{{platform_lookup_url "invalid" id}}`,
			data:     map[string]any{"id": "test"},
			render:   false,
			expected: "",
		},
		{
			name:     "conditional with lookup",
			template: `{{#if ip}}Link: {{platform_lookup_url "host" ip}}{{else}}No IP{{/if}}`,
			data:     map[string]any{"ip": "192.168.1.1"},
			render:   false,
			expected: "Link: https://platform.censys.io/hosts/192.168.1.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Register helper
			helper := NewLookupURLHelper(tc.render)
			RegisterHelpers(helper)

			// Render template
			result, err := handlebars.Render(tc.template, tc.data)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLookupURLHelper_InTemplate_WithRender(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     any
		contains string
	}{
		{
			name:     "rendered host lookup",
			template: `{{platform_lookup_url "host" ip}}`,
			data:     map[string]any{"ip": "8.8.8.8"},
			contains: "https://platform.censys.io/hosts/8.8.8.8",
		},
		{
			name:     "rendered certificate lookup",
			template: `{{platform_lookup_url "certificate" certID}}`,
			data:     map[string]any{"certID": "xyz789"},
			contains: "https://platform.censys.io/certificates/xyz789",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Register helper with render=true
			helper := NewLookupURLHelper(true)
			RegisterHelpers(helper)

			// Render template
			result, err := handlebars.Render(tc.template, tc.data)
			assert.NoError(t, err)
			// When rendered, should contain the URL (may have terminal escape codes)
			assert.Contains(t, result, tc.contains)
		})
	}
}
