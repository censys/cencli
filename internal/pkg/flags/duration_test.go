package flags

import (
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	durationFlagName  = "test-duration-flag"
	durationFlagShort = "d"
)

func TestDurationFlag(t *testing.T) {
	tests := []struct {
		name          string
		required      bool
		defaultValue  mo.Option[time.Duration]
		args          []string
		expectedValue mo.Option[time.Duration]
		expectError   bool
		expectedError error
	}{
		{
			name:          "required flag not set",
			required:      true,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{},
			expectedValue: mo.None[time.Duration](),
			expectError:   true,
			expectedError: NewRequiredFlagNotSetError(durationFlagName),
		},
		{
			name:          "optional flag not set - no default",
			required:      false,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{},
			expectedValue: mo.None[time.Duration](),
			expectError:   false,
		},
		{
			name:          "optional flag not set - with default",
			required:      false,
			defaultValue:  mo.Some(5 * time.Second),
			args:          []string{},
			expectedValue: mo.Some(5 * time.Second),
			expectError:   false,
		},
		{
			name:          "required flag set with valid duration",
			required:      true,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{"--" + durationFlagName, "10s"},
			expectedValue: mo.Some(10 * time.Second),
			expectError:   false,
		},
		{
			name:          "optional flag set with valid duration",
			required:      false,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{"--" + durationFlagName, "2m"},
			expectedValue: mo.Some(2 * time.Minute),
			expectError:   false,
		},
		{
			name:          "flag set with short form",
			required:      false,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{"-" + durationFlagShort, "1h"},
			expectedValue: mo.Some(1 * time.Hour),
			expectError:   false,
		},
		{
			name:          "flag set overrides default",
			required:      false,
			defaultValue:  mo.Some(5 * time.Second),
			args:          []string{"--" + durationFlagName, "30s"},
			expectedValue: mo.Some(30 * time.Second),
			expectError:   false,
		},
		{
			name:          "flag set with milliseconds",
			required:      false,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{"--" + durationFlagName, "500ms"},
			expectedValue: mo.Some(500 * time.Millisecond),
			expectError:   false,
		},
		{
			name:          "flag set with nanoseconds",
			required:      false,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{"--" + durationFlagName, "1000ns"},
			expectedValue: mo.Some(1000 * time.Nanosecond),
			expectError:   false,
		},
		{
			name:          "flag set with complex duration",
			required:      false,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{"--" + durationFlagName, "1h30m45s"},
			expectedValue: mo.Some(1*time.Hour + 30*time.Minute + 45*time.Second),
			expectError:   false,
		},
		{
			name:          "invalid duration - no unit",
			required:      false,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{"--" + durationFlagName, "123"},
			expectedValue: mo.None[time.Duration](),
			expectError:   true,
			expectedError: NewInvalidDurationFlagError(durationFlagName, "123"),
		},
		{
			name:          "invalid duration - unknown unit",
			required:      false,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{"--" + durationFlagName, "5x"},
			expectedValue: mo.None[time.Duration](),
			expectError:   true,
			expectedError: NewInvalidDurationFlagError(durationFlagName, "5x"),
		},
		{
			name:          "invalid duration - invalid format",
			required:      false,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{"--" + durationFlagName, "invalid"},
			expectedValue: mo.None[time.Duration](),
			expectError:   true,
			expectedError: NewInvalidDurationFlagError(durationFlagName, "invalid"),
		},
		{
			name:          "invalid duration - empty string",
			required:      false,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{"--" + durationFlagName, ""},
			expectedValue: mo.None[time.Duration](),
			expectError:   true,
			expectedError: NewInvalidDurationFlagError(durationFlagName, ""),
		},
		{
			name:          "invalid duration - required flag",
			required:      true,
			defaultValue:  mo.None[time.Duration](),
			args:          []string{"--" + durationFlagName, "bad"},
			expectedValue: mo.None[time.Duration](),
			expectError:   true,
			expectedError: NewInvalidDurationFlagError(durationFlagName, "bad"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			flag := NewDurationFlag(cmd.Flags(), tc.required, durationFlagName, durationFlagShort, tc.defaultValue, "A Duration Flag")
			cmd.SetArgs(tc.args)
			cmd.Run = func(cmd *cobra.Command, args []string) {
				value, err := flag.Value()

				if tc.expectError {
					require.Error(t, err)
					if tc.expectedError != nil {
						assert.Equal(t, tc.expectedError, err)
					}
					assert.Equal(t, tc.expectedValue, value)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tc.expectedValue, value)
				}
			}
			require.NoError(t, cmd.Execute())
		})
	}
}
