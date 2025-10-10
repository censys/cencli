package datetime

import (
	"fmt"
	"time"
)

// supportedLayouts is a list of datetime formats we'll try
var supportedLayouts = []struct {
	layout      string
	hasTimezone bool
}{
	// Layouts without timezone info (will use defaultTZ)
	{"2006-01-02 15:04:05", false},
	{"2006-01-02", false},
	{"01/02/2006 15:04:05", false},
	{"01/02/2006", false},
	{"2006/01/02", false},

	// Layouts with timezone info (will preserve parsed timezone)
	{"2006-01-02 15:04:05 -0700", true},
	{"2006-01-02 15:04:05 -07:00", true},
}

// Parse tries to parse a string into a time.Time.
// If the string is in RFC3339 format, it will be returned as-is.
// If the string does not specify a timezone, the default timezone will be used.
// If the string only specifies a date, the time will be set to 00:00:00.
func Parse(input string, defaultTZ TimeZone) (time.Time, error) {
	// First, try RFC3339 or other layouts that include timezone info.
	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return t, nil
	}

	// Try parsing with known layouts
	for _, layout := range supportedLayouts {
		if t, err := time.Parse(layout.layout, input); err == nil {
			// If layout has timezone info, return as-is
			if layout.hasTimezone {
				return t, nil
			}

			// If layout didn't have tz info, interpret in defaultTZ
			loc := defaultTZ.location()
			year, month, day := t.Date()
			hour, min, sec := t.Clock()
			return time.Date(year, month, day, hour, min, sec, t.Nanosecond(), loc), nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse time string: %q", input)
}
