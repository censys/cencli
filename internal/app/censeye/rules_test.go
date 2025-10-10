package censeye

import (
	"regexp"
	"testing"

	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/censys-sdk-go/models/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestCompileFieldRules(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		prefix   string
		config   *censeyeConfig
		expected [][]fieldValuePair
	}{
		{
			name:   "simple string field",
			input:  `{"name": "test"}`,
			prefix: "host",
			config: &censeyeConfig{
				Filters:          []string{},
				RgxFilters:       []*regexp.Regexp{},
				KeyValuePrefixes: []string{},
				ExtractionRules:  []*extractionRule{},
			},
			expected: [][]fieldValuePair{
				{{Field: "host.name", Value: "test"}},
			},
		},
		{
			name:   "numeric field",
			input:  `{"port": 443}`,
			prefix: "host",
			config: &censeyeConfig{
				Filters:          []string{},
				RgxFilters:       []*regexp.Regexp{},
				KeyValuePrefixes: []string{},
				ExtractionRules:  []*extractionRule{},
			},
			expected: [][]fieldValuePair{
				{{Field: "host.port", Value: "443"}},
			},
		},
		{
			name:   "nested object",
			input:  `{"services": {"protocol": "https"}}`,
			prefix: "host",
			config: &censeyeConfig{
				Filters:          []string{},
				RgxFilters:       []*regexp.Regexp{},
				KeyValuePrefixes: []string{},
				ExtractionRules:  []*extractionRule{},
			},
			expected: [][]fieldValuePair{
				{{Field: "host.services.protocol", Value: "https"}},
			},
		},
		{
			name:   "array of strings",
			input:  `{"tags": ["web", "prod"]}`,
			prefix: "host",
			config: &censeyeConfig{
				Filters:          []string{},
				RgxFilters:       []*regexp.Regexp{},
				KeyValuePrefixes: []string{},
				ExtractionRules:  []*extractionRule{},
			},
			expected: [][]fieldValuePair{
				{{Field: "host.tags", Value: "web"}},
				{{Field: "host.tags", Value: "prod"}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := gjson.Parse(tc.input)
			var result [][]fieldValuePair
			err := compileFieldRules(input, tc.prefix, &result, tc.config)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsKeyValueObject(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		prefix   string
		config   *censeyeConfig
		expected bool
	}{
		{
			name:   "configured key-value prefix",
			input:  `{"key1": "value1", "key2": "value2"}`,
			prefix: "host.dns.forward_dns",
			config: &censeyeConfig{
				KeyValuePrefixes: []string{"host.dns.forward_dns"},
			},
			expected: true,
		},
		{
			name:   "not in configured prefixes",
			input:  `{"key1": "value1", "key2": "value2"}`,
			prefix: "host.other",
			config: &censeyeConfig{
				KeyValuePrefixes: []string{"host.dns.forward_dns"},
			},
			expected: false,
		},
		{
			name:   "headers object pattern",
			input:  `{"Server": {"headers": ["nginx"]}, "Date": {"headers": ["Mon"]}}`,
			prefix: "host.services.endpoints.http.headers",
			config: &censeyeConfig{
				KeyValuePrefixes: []string{},
			},
			expected: true,
		},
		{
			name:   "not a headers object",
			input:  `{"Server": {"value": "nginx"}, "Date": {"value": "Mon"}}`,
			prefix: "host.services.endpoints.http.headers",
			config: &censeyeConfig{
				KeyValuePrefixes: []string{},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := gjson.Parse(tc.input)
			result := isKeyValueObject(input, tc.prefix, tc.config)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCompileServiceRules(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		config   *censeyeConfig
		expected int // Number of rule combinations expected
	}{
		{
			name: "no services",
			input: `{
				"ip": "8.8.8.8"
			}`,
			config: &censeyeConfig{
				ExtractionRules: []*extractionRule{
					{Fields: []string{"protocol", "port"}},
				},
			},
			expected: 0,
		},
		{
			name: "single service with extraction rule",
			input: `{
				"services": [
					{
						"protocol": "https",
						"port": 443
					}
				]
			}`,
			config: &censeyeConfig{
				ExtractionRules: []*extractionRule{
					{Fields: []string{"protocol"}},
				},
			},
			expected: 1,
		},
		{
			name: "service with missing field",
			input: `{
				"services": [
					{
						"protocol": "https"
					}
				]
			}`,
			config: &censeyeConfig{
				ExtractionRules: []*extractionRule{
					{Fields: []string{"protocol", "port"}},
				},
			},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := gjson.Parse(tc.input)
			var result [][]fieldValuePair
			err := compileServiceRules(input, &result, tc.config)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, len(result))
		})
	}
}

func TestGenCombos(t *testing.T) {
	testCases := []struct {
		name         string
		fieldValues  map[string][]string
		fields       []string
		prefix       string
		expectedLen  int
		validateFunc func(t *testing.T, combos [][]fieldValuePair)
	}{
		{
			name: "single field single value",
			fieldValues: map[string][]string{
				"host.services.protocol": {"https"},
			},
			fields:      []string{"protocol"},
			prefix:      "host.services",
			expectedLen: 1,
			validateFunc: func(t *testing.T, combos [][]fieldValuePair) {
				assert.Equal(t, "host.services.protocol", combos[0][0].Field)
				assert.Equal(t, "https", combos[0][0].Value)
			},
		},
		{
			name: "single field multiple values",
			fieldValues: map[string][]string{
				"host.services.protocol": {"http", "https"},
			},
			fields:      []string{"protocol"},
			prefix:      "host.services",
			expectedLen: 2,
		},
		{
			name: "multiple fields cartesian product",
			fieldValues: map[string][]string{
				"host.services.protocol": {"http", "https"},
				"host.services.port":     {"80", "443"},
			},
			fields:      []string{"protocol", "port"},
			prefix:      "host.services",
			expectedLen: 4, // 2 x 2 = 4 combinations
		},
		{
			name:        "no fields",
			fieldValues: map[string][]string{},
			fields:      []string{},
			prefix:      "host.services",
			expectedLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := genCombos(tc.fieldValues, tc.fields, tc.prefix)
			assert.Equal(t, tc.expectedLen, len(result))
			if tc.validateFunc != nil {
				tc.validateFunc(t, result)
			}
		})
	}
}

func TestDeduplicateRules(t *testing.T) {
	testCases := []struct {
		name        string
		rules       [][]fieldValuePair
		expectedLen int
	}{
		{
			name: "no duplicates",
			rules: [][]fieldValuePair{
				{{Field: "host.ip", Value: "8.8.8.8"}},
				{{Field: "host.port", Value: "443"}},
			},
			expectedLen: 2,
		},
		{
			name: "exact duplicates",
			rules: [][]fieldValuePair{
				{{Field: "host.ip", Value: "8.8.8.8"}},
				{{Field: "host.ip", Value: "8.8.8.8"}},
				{{Field: "host.ip", Value: "8.8.8.8"}},
			},
			expectedLen: 1,
		},
		{
			name: "different order same content",
			rules: [][]fieldValuePair{
				{
					{Field: "host.protocol", Value: "https"},
					{Field: "host.port", Value: "443"},
				},
				{
					{Field: "host.port", Value: "443"},
					{Field: "host.protocol", Value: "https"},
				},
			},
			expectedLen: 1, // Should deduplicate based on content regardless of order
		},
		{
			name:        "empty input",
			rules:       [][]fieldValuePair{},
			expectedLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := deduplicateRules(tc.rules)
			assert.Equal(t, tc.expectedLen, len(result))
		})
	}
}

func TestCompileRules_Integration(t *testing.T) {
	// Test with a realistic host structure
	host := &assets.Host{
		Host: components.Host{
			IP: strPtr("8.8.8.8"),
			Services: []components.Service{
				{
					Port:     intPtr(80),
					Protocol: strPtr("http"),
				},
			},
		},
	}

	config := &censeyeConfig{
		Filters:          []string{},
		RgxFilters:       []*regexp.Regexp{},
		KeyValuePrefixes: []string{},
		ExtractionRules:  []*extractionRule{},
	}

	rules, err := compileRulesForHost(host, config)
	require.NoError(t, err)
	assert.Greater(t, len(rules), 0, "should extract at least some rules")

	// Verify structure
	for _, rule := range rules {
		assert.Greater(t, len(rule), 0, "each rule should have at least one field-value pair")
		for _, fv := range rule {
			assert.NotEmpty(t, fv.Field, "field should not be empty")
			// Value can be empty in some cases
		}
	}
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
