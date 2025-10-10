package flags

import (
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/pkg/datetime"
)

const (
	timestampFlagName  = "test-timestamp-flag"
	timestampFlagShort = "t"
)

func TestRFC3339TimestampFlag(t *testing.T) {
	defaultTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		required      bool
		defaultValue  mo.Option[time.Time]
		defaultTZ     datetime.TimeZone
		args          []string
		expectedValue mo.Option[time.Time]
		expectError   bool
		expectedError error
	}{
		{
			name:          "required flag not set",
			required:      true,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneUTC,
			args:          []string{},
			expectedValue: mo.None[time.Time](),
			expectError:   true,
			expectedError: NewRequiredFlagNotSetError(timestampFlagName),
		},
		{
			name:          "optional flag not set - no default",
			required:      false,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneUTC,
			args:          []string{},
			expectedValue: mo.None[time.Time](),
			expectError:   false,
		},
		{
			name:          "optional flag not set - with default",
			required:      false,
			defaultValue:  mo.Some(defaultTime),
			defaultTZ:     datetime.TimeZoneUTC,
			args:          []string{},
			expectedValue: mo.Some(defaultTime),
			expectError:   false,
		},
		{
			name:          "required flag set with valid RFC3339 timestamp",
			required:      true,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneUTC,
			args:          []string{"--" + timestampFlagName, "2023-12-25T15:30:45Z"},
			expectedValue: mo.Some(time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)),
			expectError:   false,
		},
		{
			name:          "optional flag set with valid RFC3339 timestamp",
			required:      false,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneUTC,
			args:          []string{"--" + timestampFlagName, "2024-01-01T00:00:00Z"},
			expectedValue: mo.Some(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			expectError:   false,
		},
		{
			name:          "flag set with short form",
			required:      false,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneUTC,
			args:          []string{"-" + timestampFlagShort, "2023-06-15T10:30:00Z"},
			expectedValue: mo.Some(time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)),
			expectError:   false,
		},
		{
			name:          "flag set overrides default",
			required:      false,
			defaultValue:  mo.Some(defaultTime),
			defaultTZ:     datetime.TimeZoneUTC,
			args:          []string{"--" + timestampFlagName, "2023-12-31T23:59:59Z"},
			expectedValue: mo.Some(time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)),
			expectError:   false,
		},
		{
			name:          "invalid RFC3339 timestamp",
			required:      false,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneUTC,
			args:          []string{"--" + timestampFlagName, "invalid-timestamp"},
			expectedValue: mo.None[time.Time](),
			expectError:   true,
			expectedError: NewInvalidTimestampFlagError(timestampFlagName, "invalid-timestamp"),
		},
		// Test various supported formats
		{
			name:          "ISO datetime with space - uses defaultTZ",
			required:      false,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneAmericaNewYork,
			args:          []string{"--" + timestampFlagName, "2024-03-15 14:30:45"},
			expectedValue: mo.Some(time.Date(2024, 3, 15, 14, 30, 45, 0, mustLoadLocation(datetime.TimeZoneAmericaNewYork))),
			expectError:   false,
		},
		{
			name:          "ISO date only - uses defaultTZ",
			required:      false,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneAsiaTokyo,
			args:          []string{"--" + timestampFlagName, "2024-03-15"},
			expectedValue: mo.Some(time.Date(2024, 3, 15, 0, 0, 0, 0, mustLoadLocation(datetime.TimeZoneAsiaTokyo))),
			expectError:   false,
		},
		{
			name:          "US datetime format - uses defaultTZ",
			required:      false,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneEuropeLondon,
			args:          []string{"--" + timestampFlagName, "03/15/2024 14:30:45"},
			expectedValue: mo.Some(time.Date(2024, 3, 15, 14, 30, 45, 0, mustLoadLocation(datetime.TimeZoneEuropeLondon))),
			expectError:   false,
		},
		{
			name:          "US date format - uses defaultTZ",
			required:      false,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneUTC,
			args:          []string{"--" + timestampFlagName, "12/25/2024"},
			expectedValue: mo.Some(time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)),
			expectError:   false,
		},
		{
			name:          "slash-separated date - uses defaultTZ",
			required:      false,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneAustraliaSydney,
			args:          []string{"--" + timestampFlagName, "2024/03/15"},
			expectedValue: mo.Some(time.Date(2024, 3, 15, 0, 0, 0, 0, mustLoadLocation(datetime.TimeZoneAustraliaSydney))),
			expectError:   false,
		},
		{
			name:          "datetime with numeric timezone - preserves timezone",
			required:      false,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneUTC, // should be ignored
			args:          []string{"--" + timestampFlagName, "2024-03-15 14:30:45 -0700"},
			expectedValue: mo.Some(time.Date(2024, 3, 15, 14, 30, 45, 0, time.FixedZone("", -7*3600))),
			expectError:   false,
		},
		{
			name:          "datetime with colon timezone - preserves timezone",
			required:      false,
			defaultValue:  mo.None[time.Time](),
			defaultTZ:     datetime.TimeZoneUTC, // should be ignored
			args:          []string{"--" + timestampFlagName, "2024-03-15 14:30:45 +05:30"},
			expectedValue: mo.Some(time.Date(2024, 3, 15, 14, 30, 45, 0, time.FixedZone("", 5*3600+30*60))),
			expectError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			flag := NewTimestampFlag(cmd.Flags(), tc.required, timestampFlagName, timestampFlagShort, tc.defaultValue, "A RFC3339 timestamp flag")
			cmd.SetArgs(tc.args)
			cmd.Run = func(cmd *cobra.Command, args []string) {
				value, err := flag.Value(tc.defaultTZ)
				if tc.expectError {
					assert.Error(t, err)
					if tc.expectedError != nil {
						assert.Equal(t, tc.expectedError, err)
					}
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.expectedValue.IsPresent(), value.IsPresent())
					if tc.expectedValue.IsPresent() {
						expected := tc.expectedValue.MustGet()
						actual := value.MustGet()
						assert.True(t, expected.Equal(actual),
							"Expected %v (%s), got %v (%s)",
							expected, expected.Location(),
							actual, actual.Location())
					}
				}
			}
			require.NoError(t, cmd.Execute())
		})
	}
}

// mustLoadLocation is a helper to load a timezone location
func mustLoadLocation(tz datetime.TimeZone) *time.Location {
	loc, err := time.LoadLocation(string(tz))
	if err != nil {
		panic(err)
	}
	return loc
}
