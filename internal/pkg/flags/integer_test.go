package flags

import (
	"testing"

	"github.com/samber/mo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegerFlag_InvalidMinMax(t *testing.T) {
	cmd := &cobra.Command{}
	f := NewIntegerFlag(cmd.Flags(), false, "num", "n", mo.None[int64](), "number", mo.Some[int64](10), mo.Some[int64](20))

	// Below min
	cmd.SetArgs([]string{"--num", "5"})
	cmd.Run = func(cmd *cobra.Command, args []string) {
		v, err := f.Value()
		assert.Error(t, err)
		assert.False(t, v.IsPresent())
	}
	require.NoError(t, cmd.Execute())

	// Above max
	cmd = &cobra.Command{}
	f = NewIntegerFlag(cmd.Flags(), false, "num", "n", mo.None[int64](), "number", mo.Some[int64](10), mo.Some[int64](20))
	cmd.SetArgs([]string{"--num", "25"})
	cmd.Run = func(cmd *cobra.Command, args []string) {
		v, err := f.Value()
		assert.Error(t, err)
		assert.False(t, v.IsPresent())
	}
	require.NoError(t, cmd.Execute())

	// Valid in range
	cmd = &cobra.Command{}
	f = NewIntegerFlag(cmd.Flags(), false, "num", "n", mo.None[int64](), "number", mo.Some[int64](10), mo.Some[int64](20))
	cmd.SetArgs([]string{"--num", "15"})
	cmd.Run = func(cmd *cobra.Command, args []string) {
		v, err := f.Value()
		assert.NoError(t, err)
		assert.True(t, v.IsPresent())
		assert.Equal(t, int64(15), v.MustGet())
	}
	require.NoError(t, cmd.Execute())
}

const (
	intFlagName  = "test-integer-flag"
	intFlagShort = "i"
)

func TestIntegerFlag(t *testing.T) {
	tests := []struct {
		name          string
		required      bool
		defaultValue  mo.Option[int64]
		minValue      mo.Option[int64]
		maxValue      mo.Option[int64]
		args          []string
		expectedValue mo.Option[int64]
		expectError   bool
		expectedError error
	}{
		{
			name:          "flag not set - no default value",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{},
			expectedValue: mo.None[int64](),
			expectError:   false,
		},
		{
			name:          "flag not set - with default value",
			required:      false,
			defaultValue:  mo.Some[int64](42),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{},
			expectedValue: mo.Some[int64](42),
			expectError:   false,
		},
		{
			name:          "flag set to positive integer",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{"--" + intFlagName, "123"},
			expectedValue: mo.Some[int64](123),
			expectError:   false,
		},
		{
			name:          "flag set to negative integer",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{"--" + intFlagName, "-456"},
			expectedValue: mo.Some[int64](-456),
			expectError:   false,
		},
		{
			name:          "flag set to zero",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{"--" + intFlagName, "0"},
			expectedValue: mo.Some[int64](0),
			expectError:   false,
		},
		{
			name:          "flag set to zero with default value",
			required:      false,
			defaultValue:  mo.Some[int64](42),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{"--" + intFlagName, "0"},
			expectedValue: mo.Some[int64](0),
			expectError:   false,
		},
		{
			name:          "flag set with short form",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{"-" + intFlagShort, "789"},
			expectedValue: mo.Some[int64](789),
			expectError:   false,
		},
		{
			name:          "flag set overrides default value",
			required:      false,
			defaultValue:  mo.Some[int64](42),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{"--" + intFlagName, "999"},
			expectedValue: mo.Some[int64](999),
			expectError:   false,
		},
		{
			name:          "flag set to maximum int64",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{"--" + intFlagName, "9223372036854775807"},
			expectedValue: mo.Some[int64](9223372036854775807),
			expectError:   false,
		},
		{
			name:          "flag set to minimum int64",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{"--" + intFlagName, "-9223372036854775808"},
			expectedValue: mo.Some[int64](-9223372036854775808),
			expectError:   false,
		},
		// Required flag tests
		{
			name:          "required flag not set",
			required:      true,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{},
			expectedValue: mo.None[int64](),
			expectError:   true,
			expectedError: NewRequiredFlagNotSetError(intFlagName),
		},
		{
			name:          "required flag set",
			required:      true,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.None[int64](),
			args:          []string{"--" + intFlagName, "100"},
			expectedValue: mo.Some[int64](100),
			expectError:   false,
		},
		// Min value validation tests
		{
			name:          "value below minimum",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.Some[int64](10),
			maxValue:      mo.None[int64](),
			args:          []string{"--" + intFlagName, "5"},
			expectedValue: mo.None[int64](),
			expectError:   true,
			expectedError: NewIntegerFlagInvalidValueError(intFlagName, 5, "must be >= 10"),
		},
		{
			name:          "value at minimum",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.Some[int64](10),
			maxValue:      mo.None[int64](),
			args:          []string{"--" + intFlagName, "10"},
			expectedValue: mo.Some[int64](10),
			expectError:   false,
		},
		{
			name:          "value above minimum",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.Some[int64](10),
			maxValue:      mo.None[int64](),
			args:          []string{"--" + intFlagName, "15"},
			expectedValue: mo.Some[int64](15),
			expectError:   false,
		},
		// Max value validation tests
		{
			name:          "value above maximum",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.Some[int64](100),
			args:          []string{"--" + intFlagName, "150"},
			expectedValue: mo.None[int64](),
			expectError:   true,
			expectedError: NewIntegerFlagInvalidValueError(intFlagName, 150, "must be <= 100"),
		},
		{
			name:          "value at maximum",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.Some[int64](100),
			args:          []string{"--" + intFlagName, "100"},
			expectedValue: mo.Some[int64](100),
			expectError:   false,
		},
		{
			name:          "value below maximum",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.None[int64](),
			maxValue:      mo.Some[int64](100),
			args:          []string{"--" + intFlagName, "95"},
			expectedValue: mo.Some[int64](95),
			expectError:   false,
		},
		// Min and max value validation tests
		{
			name:          "value within range",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.Some[int64](10),
			maxValue:      mo.Some[int64](100),
			args:          []string{"--" + intFlagName, "50"},
			expectedValue: mo.Some[int64](50),
			expectError:   false,
		},
		{
			name:          "value below range",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.Some[int64](10),
			maxValue:      mo.Some[int64](100),
			args:          []string{"--" + intFlagName, "5"},
			expectedValue: mo.None[int64](),
			expectError:   true,
			expectedError: NewIntegerFlagInvalidValueError(intFlagName, 5, "must be >= 10"),
		},
		{
			name:          "value above range",
			required:      false,
			defaultValue:  mo.None[int64](),
			minValue:      mo.Some[int64](10),
			maxValue:      mo.Some[int64](100),
			args:          []string{"--" + intFlagName, "150"},
			expectedValue: mo.None[int64](),
			expectError:   true,
			expectedError: NewIntegerFlagInvalidValueError(intFlagName, 150, "must be <= 100"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			flag := NewIntegerFlag(cmd.Flags(), tc.required, intFlagName, intFlagShort, tc.defaultValue, "An Integer Flag", tc.minValue, tc.maxValue)
			cmd.SetArgs(tc.args)
			cmd.Run = func(cmd *cobra.Command, args []string) {
				value, err := flag.Value()

				if tc.expectError {
					require.Error(t, err)
					if tc.expectedError != nil {
						assert.Equal(t, tc.expectedError, err)
					}
				} else {
					require.NoError(t, err)
				}

				assert.Equal(t, tc.expectedValue, value)
			}
			require.NoError(t, cmd.Execute())
		})
	}
}

func TestIntegerFlag_Properties(t *testing.T) {
	cmd := &cobra.Command{}
	NewIntegerFlag(cmd.Flags(), false, intFlagName, intFlagShort, mo.None[int64](), "An Integer Flag", mo.None[int64](), mo.None[int64]())

	// Check that the flag exists and has correct properties
	flag := cmd.Flags().Lookup(intFlagName)
	require.NotNil(t, flag, "integer flag should exist")
	assert.Equal(t, "An Integer Flag", flag.Usage)
	assert.Equal(t, intFlagShort, flag.Shorthand)
	assert.Equal(t, "0", flag.DefValue) // Should default to 0 when no default provided
}

func TestIntegerFlag_PropertiesWithDefault(t *testing.T) {
	cmd := &cobra.Command{}
	NewIntegerFlag(cmd.Flags(), false, intFlagName, intFlagShort, mo.Some[int64](42), "An Integer Flag with Default", mo.None[int64](), mo.None[int64]())

	// Check that the flag exists and has correct properties
	flag := cmd.Flags().Lookup(intFlagName)
	require.NotNil(t, flag, "integer flag should exist")
	assert.Equal(t, "An Integer Flag with Default", flag.Usage)
	assert.Equal(t, intFlagShort, flag.Shorthand)
	assert.Equal(t, "42", flag.DefValue) // Should show the default value
}

func TestIntegerFlag_EmptyShorthand(t *testing.T) {
	cmd := &cobra.Command{}
	flag := NewIntegerFlag(cmd.Flags(), false, intFlagName, "", mo.None[int64](), "An Integer Flag", mo.None[int64](), mo.None[int64]())

	// Check that the flag exists without shorthand
	pflag := cmd.Flags().Lookup(intFlagName)
	require.NotNil(t, pflag, "integer flag should exist")
	assert.Equal(t, "", pflag.Shorthand)

	// Test that it works with long form only
	cmd.SetArgs([]string{"--" + intFlagName, "123"})
	cmd.Run = func(cmd *cobra.Command, args []string) {
		value, err := flag.Value()
		require.NoError(t, err)
		assert.Equal(t, mo.Some[int64](123), value)
	}
	require.NoError(t, cmd.Execute())
}

func TestIntegerFlag_DistinguishZeroFromNotSet(t *testing.T) {
	t.Run("zero value set explicitly", func(t *testing.T) {
		cmd := &cobra.Command{}
		flag := NewIntegerFlag(cmd.Flags(), false, intFlagName, intFlagShort, mo.None[int64](), "An Integer Flag", mo.None[int64](), mo.None[int64]())
		cmd.SetArgs([]string{"--" + intFlagName, "0"})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			assert.True(t, value.IsPresent(), "value should be present when explicitly set to 0")
			assert.Equal(t, int64(0), value.MustGet())
		}
		require.NoError(t, cmd.Execute())
	})

	t.Run("flag not set at all", func(t *testing.T) {
		cmd := &cobra.Command{}
		flag := NewIntegerFlag(cmd.Flags(), false, intFlagName, intFlagShort, mo.None[int64](), "An Integer Flag", mo.None[int64](), mo.None[int64]())
		cmd.SetArgs([]string{})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			assert.False(t, value.IsPresent(), "value should not be present when not set")
		}
		require.NoError(t, cmd.Execute())
	})
}

func TestIntegerFlag_DefaultValueBehavior(t *testing.T) {
	t.Run("default value returned when flag not set", func(t *testing.T) {
		cmd := &cobra.Command{}
		defaultVal := int64(100)
		flag := NewIntegerFlag(cmd.Flags(), false, intFlagName, intFlagShort, mo.Some(defaultVal), "An Integer Flag", mo.None[int64](), mo.None[int64]())
		cmd.SetArgs([]string{})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			assert.True(t, value.IsPresent(), "default value should be present")
			assert.Equal(t, defaultVal, value.MustGet())
		}
		require.NoError(t, cmd.Execute())
	})

	t.Run("explicit value overrides default", func(t *testing.T) {
		cmd := &cobra.Command{}
		defaultVal := int64(100)
		explicitVal := int64(200)
		flag := NewIntegerFlag(cmd.Flags(), false, intFlagName, intFlagShort, mo.Some(defaultVal), "An Integer Flag", mo.None[int64](), mo.None[int64]())
		cmd.SetArgs([]string{"--" + intFlagName, "200"})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			assert.True(t, value.IsPresent(), "explicit value should be present")
			assert.Equal(t, explicitVal, value.MustGet())
		}
		require.NoError(t, cmd.Execute())
	})

	t.Run("explicit zero overrides non-zero default", func(t *testing.T) {
		cmd := &cobra.Command{}
		defaultVal := int64(100)
		flag := NewIntegerFlag(cmd.Flags(), false, intFlagName, intFlagShort, mo.Some(defaultVal), "An Integer Flag", mo.None[int64](), mo.None[int64]())
		cmd.SetArgs([]string{"--" + intFlagName, "0"})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			assert.True(t, value.IsPresent(), "explicit zero should be present")
			assert.Equal(t, int64(0), value.MustGet())
		}
		require.NoError(t, cmd.Execute())
	})
}
