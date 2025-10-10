package flags

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sliceFlagName  = "test-slice-flag"
	sliceFlagShort = "t"
)

func TestStringSliceFlag(t *testing.T) {
	tests := []struct {
		name          string
		required      bool
		defaultValue  []string
		args          []string
		expectedValue []string
		expectError   bool
		expectedError error
	}{
		{
			name:          "required flag not set",
			required:      true,
			defaultValue:  nil,
			args:          []string{},
			expectedValue: nil,
			expectError:   true,
			expectedError: NewRequiredFlagNotSetError(sliceFlagName),
		},
		{
			name:          "required flag set to empty string (filtered out by pflag) - should error",
			required:      true,
			defaultValue:  nil,
			args:          []string{"--" + sliceFlagName, ""},
			expectedValue: nil,
			expectError:   true,
			expectedError: NewRequiredFlagNotSetError(sliceFlagName),
		},
		{
			name:          "required flag set to multiple empty strings (all filtered out) - should error",
			required:      true,
			defaultValue:  nil,
			args:          []string{"--" + sliceFlagName, "", "--" + sliceFlagName, "", "--" + sliceFlagName, ""},
			expectedValue: nil,
			expectError:   true,
			expectedError: NewRequiredFlagNotSetError(sliceFlagName),
		},
		{
			name:          "optional flag not set - no default",
			required:      false,
			defaultValue:  nil,
			args:          []string{},
			expectedValue: []string{},
			expectError:   false,
		},
		{
			name:          "optional flag not set - with default",
			required:      false,
			defaultValue:  []string{"default1", "default2"},
			args:          []string{},
			expectedValue: []string{"default1", "default2"},
			expectError:   false,
		},
		{
			name:          "single value provided",
			required:      false,
			defaultValue:  nil,
			args:          []string{"--" + sliceFlagName, "value1"},
			expectedValue: []string{"value1"},
			expectError:   false,
		},
		{
			name:          "multiple values provided via multiple flags",
			required:      false,
			defaultValue:  nil,
			args:          []string{"--" + sliceFlagName, "value1", "--" + sliceFlagName, "value2", "--" + sliceFlagName, "value3"},
			expectedValue: []string{"value1", "value2", "value3"},
			expectError:   false,
		},
		{
			name:          "multiple values provided via comma separation",
			required:      false,
			defaultValue:  nil,
			args:          []string{"--" + sliceFlagName, "value1,value2,value3"},
			expectedValue: []string{"value1", "value2", "value3"},
			expectError:   false,
		},
		{
			name:          "mixed multiple flags and comma separation",
			required:      false,
			defaultValue:  nil,
			args:          []string{"--" + sliceFlagName, "value1,value2", "--" + sliceFlagName, "value3"},
			expectedValue: []string{"value1", "value2", "value3"},
			expectError:   false,
		},
		{
			name:          "short form flag",
			required:      false,
			defaultValue:  nil,
			args:          []string{"-" + sliceFlagShort, "value1", "-" + sliceFlagShort, "value2"},
			expectedValue: []string{"value1", "value2"},
			expectError:   false,
		},
		{
			name:          "values with spaces are trimmed",
			required:      false,
			defaultValue:  nil,
			args:          []string{"--" + sliceFlagName, "  value1  ", "--" + sliceFlagName, "  value2  "},
			expectedValue: []string{"value1", "value2"},
			expectError:   false,
		},
		{
			name:          "empty values are filtered out by pflag",
			required:      false,
			defaultValue:  nil,
			args:          []string{"--" + sliceFlagName, "", "--" + sliceFlagName, "value2"},
			expectedValue: []string{"value2"},
			expectError:   false,
		},
		{
			name:          "flag overrides default values",
			required:      false,
			defaultValue:  []string{"default1", "default2"},
			args:          []string{"--" + sliceFlagName, "override1", "--" + sliceFlagName, "override2"},
			expectedValue: []string{"override1", "override2"},
			expectError:   false,
		},
		{
			name:          "special characters in values",
			required:      false,
			defaultValue:  nil,
			args:          []string{"--" + sliceFlagName, "value@#$", "--" + sliceFlagName, "value with spaces"},
			expectedValue: []string{"value@#$", "value with spaces"},
			expectError:   false,
		},
		{
			name:          "unicode values",
			required:      false,
			defaultValue:  nil,
			args:          []string{"--" + sliceFlagName, "测试", "--" + sliceFlagName, "🚀"},
			expectedValue: []string{"测试", "🚀"},
			expectError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			flag := NewStringSliceFlag(cmd.Flags(), tc.required, sliceFlagName, sliceFlagShort, tc.defaultValue, "A String Slice Flag")
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

func TestStringSliceFlag_Properties(t *testing.T) {
	cmd := &cobra.Command{}
	NewStringSliceFlag(cmd.Flags(), false, sliceFlagName, sliceFlagShort, nil, "A String Slice Flag")

	// Check that the flag exists and has correct properties
	flag := cmd.Flags().Lookup(sliceFlagName)
	require.NotNil(t, flag, "string slice flag should exist")
	assert.Equal(t, "A String Slice Flag", flag.Usage)
	assert.Equal(t, sliceFlagShort, flag.Shorthand)
	assert.Equal(t, "[]", flag.DefValue) // Should default to empty slice when no default provided
}

func TestStringSliceFlag_PropertiesWithDefault(t *testing.T) {
	cmd := &cobra.Command{}
	defaultValues := []string{"default1", "default2"}
	NewStringSliceFlag(cmd.Flags(), false, sliceFlagName, sliceFlagShort, defaultValues, "A String Slice Flag with Default")

	// Check that the flag exists and has correct properties
	flag := cmd.Flags().Lookup(sliceFlagName)
	require.NotNil(t, flag, "string slice flag should exist")
	assert.Equal(t, "A String Slice Flag with Default", flag.Usage)
	assert.Equal(t, sliceFlagShort, flag.Shorthand)
	assert.Equal(t, "[default1,default2]", flag.DefValue) // Should show the default values
}

func TestStringSliceFlag_EmptyShorthand(t *testing.T) {
	cmd := &cobra.Command{}
	flag := NewStringSliceFlag(cmd.Flags(), false, sliceFlagName, "", nil, "A String Slice Flag")

	// Check that the flag exists without shorthand
	pflag := cmd.Flags().Lookup(sliceFlagName)
	require.NotNil(t, pflag, "string slice flag should exist")
	assert.Equal(t, "", pflag.Shorthand)

	// Test that it works with long form only
	cmd.SetArgs([]string{"--" + sliceFlagName, "value1", "--" + sliceFlagName, "value2"})
	cmd.Run = func(cmd *cobra.Command, args []string) {
		value, err := flag.Value()
		require.NoError(t, err)
		assert.Equal(t, []string{"value1", "value2"}, value)
	}
	require.NoError(t, cmd.Execute())
}

func TestStringSliceFlag_RequiredValidation(t *testing.T) {
	t.Run("required flag with values", func(t *testing.T) {
		cmd := &cobra.Command{}
		flag := NewStringSliceFlag(cmd.Flags(), true, sliceFlagName, sliceFlagShort, nil, "A Required String Slice Flag")
		cmd.SetArgs([]string{"--" + sliceFlagName, "value1", "--" + sliceFlagName, "value2"})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			assert.Equal(t, []string{"value1", "value2"}, value)
		}
		require.NoError(t, cmd.Execute())
	})

	t.Run("required flag not provided", func(t *testing.T) {
		cmd := &cobra.Command{}
		flag := NewStringSliceFlag(cmd.Flags(), true, sliceFlagName, sliceFlagShort, nil, "A Required String Slice Flag")
		cmd.SetArgs([]string{})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.Error(t, err)
			assert.IsType(t, &requiredFlagNotSetError{}, err)
			assert.Equal(t, NewRequiredFlagNotSetError(sliceFlagName), err)
			assert.Nil(t, value)
		}
		require.NoError(t, cmd.Execute())
	})
}

func TestStringSliceFlag_DefaultValueBehavior(t *testing.T) {
	t.Run("default values returned when flag not set", func(t *testing.T) {
		cmd := &cobra.Command{}
		defaultValues := []string{"default1", "default2", "default3"}
		flag := NewStringSliceFlag(cmd.Flags(), false, sliceFlagName, sliceFlagShort, defaultValues, "A String Slice Flag")
		cmd.SetArgs([]string{})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			assert.Equal(t, defaultValues, value)
		}
		require.NoError(t, cmd.Execute())
	})

	t.Run("explicit values override defaults", func(t *testing.T) {
		cmd := &cobra.Command{}
		defaultValues := []string{"default1", "default2"}
		explicitValues := []string{"explicit1", "explicit2", "explicit3"}
		flag := NewStringSliceFlag(cmd.Flags(), false, sliceFlagName, sliceFlagShort, defaultValues, "A String Slice Flag")
		cmd.SetArgs([]string{"--" + sliceFlagName, "explicit1", "--" + sliceFlagName, "explicit2", "--" + sliceFlagName, "explicit3"})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			assert.Equal(t, explicitValues, value)
		}
		require.NoError(t, cmd.Execute())
	})

	t.Run("empty string overrides defaults (but gets filtered)", func(t *testing.T) {
		cmd := &cobra.Command{}
		defaultValues := []string{"default1", "default2"}
		flag := NewStringSliceFlag(cmd.Flags(), false, sliceFlagName, sliceFlagShort, defaultValues, "A String Slice Flag")
		cmd.SetArgs([]string{"--" + sliceFlagName, ""})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			assert.Equal(t, []string{}, value) // Empty string gets filtered out by pflag
		}
		require.NoError(t, cmd.Execute())
	})
}

func TestStringSliceFlag_ValueImmutability(t *testing.T) {
	t.Run("returned slice is a copy", func(t *testing.T) {
		cmd := &cobra.Command{}
		flag := NewStringSliceFlag(cmd.Flags(), false, sliceFlagName, sliceFlagShort, nil, "A String Slice Flag")
		cmd.SetArgs([]string{"--" + sliceFlagName, "value1", "--" + sliceFlagName, "value2"})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value1, err := flag.Value()
			require.NoError(t, err)

			value2, err := flag.Value()
			require.NoError(t, err)

			// Modify the first returned slice
			value1[0] = "modified"

			// Second call should return unmodified values
			assert.Equal(t, []string{"value1", "value2"}, value2)
			assert.NotEqual(t, value1, value2)
		}
		require.NoError(t, cmd.Execute())
	})
}

func TestStringSliceFlag_EdgeCases(t *testing.T) {
	t.Run("very long values", func(t *testing.T) {
		cmd := &cobra.Command{}
		flag := NewStringSliceFlag(cmd.Flags(), false, sliceFlagName, sliceFlagShort, nil, "A String Slice Flag")
		longValue := strings.Repeat("a", 10000)
		cmd.SetArgs([]string{"--" + sliceFlagName, longValue})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			assert.Equal(t, []string{longValue}, value)
		}
		require.NoError(t, cmd.Execute())
	})

	t.Run("many values", func(t *testing.T) {
		cmd := &cobra.Command{}
		flag := NewStringSliceFlag(cmd.Flags(), false, sliceFlagName, sliceFlagShort, nil, "A String Slice Flag")

		// Create args with 100 values
		args := []string{}
		expectedValues := []string{}
		for i := 0; i < 100; i++ {
			value := "value" + string(rune('0'+i%10))
			args = append(args, "--"+sliceFlagName, value)
			expectedValues = append(expectedValues, value)
		}

		cmd.SetArgs(args)
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			assert.Equal(t, expectedValues, value)
		}
		require.NoError(t, cmd.Execute())
	})

	t.Run("values with newlines (pflag treats newlines as separators)", func(t *testing.T) {
		cmd := &cobra.Command{}
		flag := NewStringSliceFlag(cmd.Flags(), false, sliceFlagName, sliceFlagShort, nil, "A String Slice Flag")
		cmd.SetArgs([]string{"--" + sliceFlagName, "line1\nline2", "--" + sliceFlagName, "line3\nline4"})
		cmd.Run = func(cmd *cobra.Command, args []string) {
			value, err := flag.Value()
			require.NoError(t, err)
			// pflag treats newlines as separators, so line2 and line4 get cut off
			assert.Equal(t, []string{"line1", "line3"}, value)
		}
		require.NoError(t, cmd.Execute())
	})
}
