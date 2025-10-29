package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocationHelper_Name(t *testing.T) {
	helper := NewLocationHelper(false)
	assert.Equal(t, "render_location", helper.Name())
}

func TestLocationHelper_BasicUsage(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name: "city, province, country, country code (all different)",
			input: map[string]any{
				"city":         "New York",
				"province":     "New York State",
				"country":      "United States",
				"country_code": "US",
			},
			expected: "New York, New York State, United States, (US)",
		},
		{
			name: "city and province same name (Madrid, Madrid)",
			input: map[string]any{
				"city":         "Madrid",
				"province":     "Madrid",
				"country":      "Spain",
				"country_code": "ES",
			},
			expected: "Madrid, Spain, (ES)",
		},
		{
			name: "city and province same name case-insensitive",
			input: map[string]any{
				"city":         "madrid",
				"province":     "Madrid",
				"country":      "Spain",
				"country_code": "ES",
			},
			expected: "madrid, Spain, (ES)",
		},
		{
			name: "only country and country code",
			input: map[string]any{
				"country":      "United States",
				"country_code": "US",
			},
			expected: "United States, (US)",
		},
		{
			name: "city and country, no province",
			input: map[string]any{
				"city":         "London",
				"country":      "United Kingdom",
				"country_code": "GB",
			},
			expected: "London, United Kingdom, (GB)",
		},
		{
			name: "province and country, no city",
			input: map[string]any{
				"province":     "California",
				"country":      "United States",
				"country_code": "US",
			},
			expected: "California, United States, (US)",
		},
		{
			name: "only country, no country code",
			input: map[string]any{
				"country": "Germany",
			},
			expected: "Germany",
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			expected: "",
		},
		{
			name: "empty strings",
			input: map[string]any{
				"city":         "",
				"province":     "",
				"country":      "",
				"country_code": "",
			},
			expected: "",
		},
		{
			name: "city, province, country (no country code)",
			input: map[string]any{
				"city":     "Paris",
				"province": "Île-de-France",
				"country":  "France",
			},
			expected: "Paris, Île-de-France, France",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewLocationHelper(false)
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLocationHelper_WithColor(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		colored  bool
		expected string
	}{
		{
			name: "colored output (colors disabled in test env)",
			input: map[string]any{
				"city":         "Madrid",
				"province":     "Madrid",
				"country":      "Spain",
				"country_code": "ES",
			},
			colored:  true,
			expected: "Madrid, Spain, (ES)",
		},
		{
			name: "no colored output",
			input: map[string]any{
				"city":         "Madrid",
				"province":     "Madrid",
				"country":      "Spain",
				"country_code": "ES",
			},
			colored:  false,
			expected: "Madrid, Spain, (ES)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewLocationHelper(tc.colored)
			fn := helper.Function().(func(any) string)
			result := fn(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLocationHelper_Interface(t *testing.T) {
	helper := NewLocationHelper(false)
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}
