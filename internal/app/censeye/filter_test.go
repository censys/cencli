package censeye

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyPrefixFilters(t *testing.T) {
	testCases := []struct {
		name        string
		rules       [][]fieldValuePair
		filters     []string
		expectedLen int
	}{
		{
			name: "no filters",
			rules: [][]fieldValuePair{
				{{Field: "host.ip", Value: "8.8.8.8"}},
				{{Field: "host.services.port", Value: "443"}},
			},
			filters:     []string{},
			expectedLen: 2,
		},
		{
			name: "exact match filter",
			rules: [][]fieldValuePair{
				{{Field: "host.ip", Value: "8.8.8.8"}},
				{{Field: "host.services.port", Value: "443"}},
			},
			filters:     []string{"host.ip"},
			expectedLen: 1,
		},
		{
			name: "prefix filter with dot",
			rules: [][]fieldValuePair{
				{{Field: "host.location.country", Value: "US"}},
				{{Field: "host.location.city", Value: "NYC"}},
				{{Field: "host.services.port", Value: "443"}},
			},
			filters:     []string{"host.location."},
			expectedLen: 1,
		},
		{
			name: "multiple filters",
			rules: [][]fieldValuePair{
				{{Field: "host.ip", Value: "8.8.8.8"}},
				{{Field: "host.location.country", Value: "US"}},
				{{Field: "host.services.port", Value: "443"}},
			},
			filters:     []string{"host.ip", "host.location."},
			expectedLen: 1,
		},
		{
			name: "filter on multi-field rule",
			rules: [][]fieldValuePair{
				{
					{Field: "host.services.protocol", Value: "https"},
					{Field: "host.services.port", Value: "443"},
				},
				{
					{Field: "host.services.protocol", Value: "http"},
					{Field: "host.services.scan_time", Value: "12345"},
				},
			},
			filters:     []string{"host.services.scan_time"},
			expectedLen: 1,
		},
		{
			name:        "empty input",
			rules:       [][]fieldValuePair{},
			filters:     []string{"host.ip"},
			expectedLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := applyPrefixFilters(tc.rules, tc.filters)
			assert.Equal(t, tc.expectedLen, len(result))
		})
	}
}

func TestApplyRegexFilters(t *testing.T) {
	testCases := []struct {
		name        string
		rules       [][]fieldValuePair
		regexStrs   []string
		expectedLen int
	}{
		{
			name: "no regex filters",
			rules: [][]fieldValuePair{
				{{Field: "host.ip", Value: "8.8.8.8"}},
			},
			regexStrs:   []string{},
			expectedLen: 1,
		},
		{
			name: "filter by exact query match",
			rules: [][]fieldValuePair{
				{{Field: "host.services.protocol", Value: "HTTP"}},
				{{Field: "host.services.protocol", Value: "HTTPS"}},
			},
			regexStrs:   []string{`^host\.services\.protocol="HTTP"$`},
			expectedLen: 1,
		},
		{
			name: "filter by pattern",
			rules: [][]fieldValuePair{
				{{Field: "host.services.endpoints.http.status_code", Value: "404"}},
				{{Field: "host.services.endpoints.http.status_code", Value: "200"}},
				{{Field: "host.services.endpoints.http.status_code", Value: "500"}},
			},
			regexStrs:   []string{`status_code="(200|404)"`},
			expectedLen: 1, // Only 500 should remain
		},
		{
			name: "multiple regex filters",
			rules: [][]fieldValuePair{
				{{Field: "host.services.protocol", Value: "HTTP"}},
				{{Field: "host.services.protocol", Value: "HTTPS"}},
				{{Field: "host.services.protocol", Value: "SSH"}},
			},
			regexStrs:   []string{`protocol="HTTP"$`, `protocol="SSH"$`},
			expectedLen: 1, // Only HTTPS should remain
		},
		{
			name:        "empty input",
			rules:       [][]fieldValuePair{},
			regexStrs:   []string{`test`},
			expectedLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Compile regex patterns
			regexes := make([]*regexp.Regexp, 0, len(tc.regexStrs))
			for _, pattern := range tc.regexStrs {
				rgx := regexp.MustCompile(pattern)
				regexes = append(regexes, rgx)
			}

			config := &censeyeConfig{
				RgxFilters: regexes,
			}

			result := applyRegexFilters(tc.rules, config)
			assert.Equal(t, tc.expectedLen, len(result))
		})
	}
}

func TestApplyFilters_Integration(t *testing.T) {
	testCases := []struct {
		name        string
		rules       [][]fieldValuePair
		config      *censeyeConfig
		expectedLen int
	}{
		{
			name: "combined prefix and regex filters",
			rules: [][]fieldValuePair{
				{{Field: "host.ip", Value: "8.8.8.8"}},
				{{Field: "host.location.country", Value: "US"}},
				{{Field: "host.services.protocol", Value: "HTTP"}},
				{{Field: "host.services.protocol", Value: "HTTPS"}},
			},
			config: &censeyeConfig{
				Filters:    []string{"host.location."},
				RgxFilters: []*regexp.Regexp{regexp.MustCompile(`protocol="HTTP"$`)},
			},
			expectedLen: 2, // host.ip and HTTPS remain
		},
		{
			name: "deduplicate after filtering",
			rules: [][]fieldValuePair{
				{{Field: "host.services.port", Value: "443"}},
				{{Field: "host.services.port", Value: "443"}},
				{{Field: "host.services.port", Value: "80"}},
			},
			config: &censeyeConfig{
				Filters:    []string{},
				RgxFilters: []*regexp.Regexp{},
			},
			expectedLen: 2, // Duplicates removed
		},
		{
			name: "all filters applied",
			rules: [][]fieldValuePair{
				{{Field: "host.ip", Value: "1.1.1.1"}},
				{{Field: "host.ip", Value: "1.1.1.1"}},
				{{Field: "host.location.country", Value: "US"}},
				{{Field: "host.services.protocol", Value: "HTTP"}},
			},
			config: &censeyeConfig{
				Filters:    []string{"host.location."},
				RgxFilters: []*regexp.Regexp{regexp.MustCompile(`protocol="HTTP"$`)},
			},
			expectedLen: 1, // Only unique host.ip remains
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := applyFilters(tc.rules, tc.config)
			assert.Equal(t, tc.expectedLen, len(result))
		})
	}
}
