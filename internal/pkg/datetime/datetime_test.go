package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	// Helper to create time in a specific timezone
	makeTime := func(tz TimeZone, year int, month time.Month, day, hour, min, sec int) time.Time {
		return time.Date(year, month, day, hour, min, sec, 0, tz.location())
	}

	testCases := []struct {
		name        string
		defaultTZ   TimeZone
		input       string
		expected    time.Time
		errContains string
	}{
		// RFC3339 format (includes timezone info)
		{
			name:      "RFC3339 with Z",
			defaultTZ: TimeZoneAmericaNewYork,
			input:     "2024-03-15T14:30:45Z",
			expected:  time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC),
		},
		{
			name:      "RFC3339 with offset +05:30",
			defaultTZ: TimeZoneUTC,
			input:     "2024-03-15T14:30:45+05:30",
			expected:  time.Date(2024, 3, 15, 14, 30, 45, 0, time.FixedZone("", 5*3600+30*60)),
		},
		{
			name:      "RFC3339 with negative offset -08:00",
			defaultTZ: TimeZoneUTC,
			input:     "2024-03-15T14:30:45-08:00",
			expected:  time.Date(2024, 3, 15, 14, 30, 45, 0, time.FixedZone("", -8*3600)),
		},

		// Layout: "2006-01-02 15:04:05" (no timezone, uses defaultTZ)
		{
			name:      "datetime space separated UTC",
			defaultTZ: TimeZoneUTC,
			input:     "2024-03-15 14:30:45",
			expected:  makeTime(TimeZoneUTC, 2024, 3, 15, 14, 30, 45),
		},
		{
			name:      "datetime space separated America/New_York",
			defaultTZ: TimeZoneAmericaNewYork,
			input:     "2024-03-15 14:30:45",
			expected:  makeTime(TimeZoneAmericaNewYork, 2024, 3, 15, 14, 30, 45),
		},
		{
			name:      "datetime space separated Asia/Tokyo",
			defaultTZ: TimeZoneAsiaTokyo,
			input:     "2024-03-15 14:30:45",
			expected:  makeTime(TimeZoneAsiaTokyo, 2024, 3, 15, 14, 30, 45),
		},
		{
			name:      "datetime space separated Europe/London",
			defaultTZ: TimeZoneEuropeLondon,
			input:     "2024-03-15 14:30:45",
			expected:  makeTime(TimeZoneEuropeLondon, 2024, 3, 15, 14, 30, 45),
		},

		// Layout: "2006-01-02" (date only, no timezone, uses defaultTZ)
		{
			name:      "date only UTC",
			defaultTZ: TimeZoneUTC,
			input:     "2024-03-15",
			expected:  makeTime(TimeZoneUTC, 2024, 3, 15, 0, 0, 0),
		},
		{
			name:      "date only America/Chicago",
			defaultTZ: TimeZoneAmericaChicago,
			input:     "2024-03-15",
			expected:  makeTime(TimeZoneAmericaChicago, 2024, 3, 15, 0, 0, 0),
		},
		{
			name:      "date only Australia/Sydney",
			defaultTZ: TimeZoneAustraliaSydney,
			input:     "2024-03-15",
			expected:  makeTime(TimeZoneAustraliaSydney, 2024, 3, 15, 0, 0, 0),
		},

		// Layout: "01/02/2006 15:04:05" (US format with time)
		{
			name:      "US datetime format UTC",
			defaultTZ: TimeZoneUTC,
			input:     "03/15/2024 14:30:45",
			expected:  makeTime(TimeZoneUTC, 2024, 3, 15, 14, 30, 45),
		},
		{
			name:      "US datetime format America/Los_Angeles",
			defaultTZ: TimeZoneAmericaLosAngeles,
			input:     "03/15/2024 14:30:45",
			expected:  makeTime(TimeZoneAmericaLosAngeles, 2024, 3, 15, 14, 30, 45),
		},
		{
			name:      "US datetime format with midnight",
			defaultTZ: TimeZoneUTC,
			input:     "12/31/2023 00:00:00",
			expected:  makeTime(TimeZoneUTC, 2023, 12, 31, 0, 0, 0),
		},

		// Layout: "01/02/2006" (US date format)
		{
			name:      "US date format UTC",
			defaultTZ: TimeZoneUTC,
			input:     "03/15/2024",
			expected:  makeTime(TimeZoneUTC, 2024, 3, 15, 0, 0, 0),
		},
		{
			name:      "US date format Europe/Paris",
			defaultTZ: TimeZoneEuropeParis,
			input:     "12/31/2023",
			expected:  makeTime(TimeZoneEuropeParis, 2023, 12, 31, 0, 0, 0),
		},

		// Layout: "2006/01/02" (slash separated date)
		{
			name:      "slash date format UTC",
			defaultTZ: TimeZoneUTC,
			input:     "2024/03/15",
			expected:  makeTime(TimeZoneUTC, 2024, 3, 15, 0, 0, 0),
		},
		{
			name:      "slash date format Asia/Singapore",
			defaultTZ: TimeZoneAsiaSingapore,
			input:     "2024/03/15",
			expected:  makeTime(TimeZoneAsiaSingapore, 2024, 3, 15, 0, 0, 0),
		},

		// Layout: "2006-01-02 15:04:05 -0700" (datetime with numeric timezone)
		{
			name:      "datetime with numeric timezone -0700",
			defaultTZ: TimeZoneUTC,
			input:     "2024-03-15 14:30:45 -0700",
			expected:  time.Date(2024, 3, 15, 14, 30, 45, 0, time.FixedZone("", -7*3600)),
		},
		{
			name:      "datetime with numeric timezone +0000",
			defaultTZ: TimeZoneAmericaNewYork,
			input:     "2024-03-15 14:30:45 +0000",
			expected:  time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC),
		},
		{
			name:      "datetime with numeric timezone +0900",
			defaultTZ: TimeZoneUTC,
			input:     "2024-03-15 14:30:45 +0900",
			expected:  time.Date(2024, 3, 15, 14, 30, 45, 0, time.FixedZone("", 9*3600)),
		},

		// Layout: "2006-01-02 15:04:05 -07:00" (datetime with colon timezone)
		{
			name:      "datetime with colon timezone -07:00",
			defaultTZ: TimeZoneUTC,
			input:     "2024-03-15 14:30:45 -07:00",
			expected:  time.Date(2024, 3, 15, 14, 30, 45, 0, time.FixedZone("", -7*3600)),
		},
		{
			name:      "datetime with colon timezone +05:30",
			defaultTZ: TimeZoneUTC,
			input:     "2024-03-15 14:30:45 +05:30",
			expected:  time.Date(2024, 3, 15, 14, 30, 45, 0, time.FixedZone("", 5*3600+30*60)),
		},

		// Edge cases
		{
			name:      "date with single digit month and day",
			defaultTZ: TimeZoneUTC,
			input:     "01/05/2024",
			expected:  makeTime(TimeZoneUTC, 2024, 1, 5, 0, 0, 0),
		},
		{
			name:      "datetime with 23:59:59",
			defaultTZ: TimeZoneUTC,
			input:     "2024-12-31 23:59:59",
			expected:  makeTime(TimeZoneUTC, 2024, 12, 31, 23, 59, 59),
		},
		{
			name:      "leap year date",
			defaultTZ: TimeZoneUTC,
			input:     "2024-02-29",
			expected:  makeTime(TimeZoneUTC, 2024, 2, 29, 0, 0, 0),
		},

		// Error cases
		{
			name:        "invalid format",
			defaultTZ:   TimeZoneUTC,
			input:       "not a date",
			errContains: "could not parse time string",
		},
		{
			name:        "empty string",
			defaultTZ:   TimeZoneUTC,
			input:       "",
			errContains: "could not parse time string",
		},
		{
			name:        "invalid date values",
			defaultTZ:   TimeZoneUTC,
			input:       "2024-13-45",
			errContains: "could not parse time string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Parse(tc.input, tc.defaultTZ)
			if tc.errContains != "" {
				assert.ErrorContains(t, err, tc.errContains)
			} else {
				assert.NoError(t, err)
				assert.True(t, tc.expected.Equal(result),
					"Expected %v (%s), got %v (%s)",
					tc.expected, tc.expected.Location(),
					result, result.Location())
			}
		})
	}
}
