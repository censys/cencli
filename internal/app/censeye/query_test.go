package censeye

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToCenqlQuery(t *testing.T) {
	testCases := []struct {
		name     string
		pairs    []fieldValuePair
		expected string
	}{
		{
			name:     "empty pairs",
			pairs:    []fieldValuePair{},
			expected: "",
		},
		{
			name: "single field-value pair",
			pairs: []fieldValuePair{
				{Field: "host.ip", Value: "8.8.8.8"},
			},
			expected: `host.ip="8.8.8.8"`,
		},
		{
			name: "single service field",
			pairs: []fieldValuePair{
				{Field: "host.services.protocol", Value: "https"},
			},
			expected: `host.services.protocol="https"`,
		},
		{
			name: "multiple service fields",
			pairs: []fieldValuePair{
				{Field: "host.services.protocol", Value: "https"},
				{Field: "host.services.port", Value: "443"},
			},
			expected: `host.services:(protocol="https" and port="443")`,
		},
		{
			name: "three service fields",
			pairs: []fieldValuePair{
				{Field: "host.services.endpoints.http.protocol", Value: "HTTP/1.1"},
				{Field: "host.services.endpoints.http.status_code", Value: "200"},
				{Field: "host.services.endpoints.http.status_reason", Value: "OK"},
			},
			expected: `host.services:(endpoints.http.protocol="HTTP/1.1" and endpoints.http.status_code="200" and endpoints.http.status_reason="OK")`,
		},
		{
			name: "field with special characters",
			pairs: []fieldValuePair{
				{Field: "host.services.cert.parsed.subject_dn", Value: "CN=example.com, O=Example Inc"},
			},
			expected: `host.services.cert.parsed.subject_dn="CN=example.com, O=Example Inc"`,
		},
		{
			name: "numeric value",
			pairs: []fieldValuePair{
				{Field: "host.services.port", Value: "443"},
			},
			expected: `host.services.port="443"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := toCenqlQuery(tc.pairs)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToSearchURL(t *testing.T) {
	testCases := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "simple query",
			query:    `host.ip="8.8.8.8"`,
			expected: `https://platform.censys.io/search?q=host.ip%3D%228.8.8.8%22`,
		},
		{
			name:     "query with spaces",
			query:    `host.services:(protocol="https" and port="443")`,
			expected: `https://platform.censys.io/search?q=host.services%3A%28protocol%3D%22https%22+and+port%3D%22443%22%29`,
		},
		{
			name:     "empty query",
			query:    "",
			expected: "https://platform.censys.io/search?q=",
		},
		{
			name:     "complex query",
			query:    `host.services:(tls.ja4s="t130200_1302_a56c5b993250" and protocol="https")`,
			expected: `https://platform.censys.io/search?q=host.services%3A%28tls.ja4s%3D%22t130200_1302_a56c5b993250%22+and+protocol%3D%22https%22%29`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := toSearchURL(tc.query)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToCenqlQuery_RoundTrip(t *testing.T) {
	// Test that we can generate a query and it's in the expected format
	pairs := []fieldValuePair{
		{Field: "host.services.tls.ja4s", Value: "t130200_1302_a56c5b993250"},
		{Field: "host.services.protocol", Value: "https"},
	}

	query := toCenqlQuery(pairs)
	assert.NotEmpty(t, query)
	assert.Contains(t, query, "host.services:")
	assert.Contains(t, query, "tls.ja4s=")
	assert.Contains(t, query, "protocol=")
	assert.Contains(t, query, "and")

	// Should be able to convert to URL
	url := toSearchURL(query)
	assert.Contains(t, url, "https://platform.censys.io/search?q=")
	// Extract just the query parameter part after "?q="
	queryPart := url[len("https://platform.censys.io/search?q="):]
	assert.NotContains(t, queryPart, " ") // Spaces should be replaced with +
	assert.NotContains(t, queryPart, `"`) // Quotes should be URL encoded
	assert.NotContains(t, queryPart, ":") // Colons should be URL encoded
}
