package flags

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	flagName  = "test-string-flag"
	flagShort = "s"
)

func TestStringFlag(t *testing.T) {
	tests := []struct {
		name          string
		required      bool
		defaultValue  string
		args          []string
		expectedValue string
		err           error
	}{
		{
			name:          "required flag not set",
			required:      true,
			args:          []string{},
			expectedValue: "",
			err:           NewRequiredFlagNotSetError(flagName),
		},
		{
			name:          "required flag empty",
			required:      true,
			args:          []string{"--" + flagName, ""},
			expectedValue: "",
			err:           NewRequiredFlagNotSetError(flagName),
		},
		{
			name:          "optional flag set",
			required:      false,
			args:          []string{"--" + flagName, "value"},
			expectedValue: "value",
		},
		{
			name:          "optional flag not set",
			required:      false,
			args:          []string{},
			expectedValue: "",
		},
		{
			name:          "optional flag not set - default value",
			required:      false,
			defaultValue:  "value",
			args:          []string{},
			expectedValue: "value",
		},
		{
			name:          "optional flag set - default override",
			required:      false,
			defaultValue:  "value",
			args:          []string{"--" + flagName, "value2"},
			expectedValue: "value2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			flag := NewStringFlag(cmd.Flags(), tc.required, flagName, flagShort, tc.defaultValue, "A String Flag")
			cmd.SetArgs(tc.args)
			cmd.Run = func(cmd *cobra.Command, args []string) {
				value, err := flag.Value()
				assert.Equal(t, tc.expectedValue, value)
				if tc.err != nil {
					require.Error(t, err)
					assert.IsType(t, tc.err, err)
					assert.Equal(t, tc.err.Error(), err.Error())
				}
			}
			require.NoError(t, cmd.Execute())
		})
	}
}

func TestBoolFlag(t *testing.T) {
	tests := []struct {
		name          string
		defaultValue  bool
		args          []string
		expectedValue bool
		err           error
	}{
		{
			name:          "flag not set - default false",
			defaultValue:  false,
			args:          []string{},
			expectedValue: false,
		},
		{
			name:          "flag not set - default true",
			defaultValue:  true,
			args:          []string{},
			expectedValue: true,
		},
		{
			name:          "flag set to true with long form",
			defaultValue:  false,
			args:          []string{"--" + flagName + "=true"},
			expectedValue: true,
		},
		{
			name:          "flag set to false with long form",
			defaultValue:  true,
			args:          []string{"--" + flagName + "=false"},
			expectedValue: false,
		},
		{
			name:          "flag set without value (defaults to true)",
			defaultValue:  false,
			args:          []string{"--" + flagName},
			expectedValue: true,
		},
		{
			name:          "flag set with short form",
			defaultValue:  false,
			args:          []string{"-" + flagShort},
			expectedValue: true,
		},
		{
			name:          "flag set to true overrides default false",
			defaultValue:  false,
			args:          []string{"--" + flagName + "=true"},
			expectedValue: true,
		},
		{
			name:          "flag set to false overrides default true",
			defaultValue:  true,
			args:          []string{"--" + flagName + "=false"},
			expectedValue: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			flag := NewBoolFlag(cmd.Flags(), flagName, flagShort, tc.defaultValue, "A Bool Flag")
			cmd.SetArgs(tc.args)
			cmd.Run = func(cmd *cobra.Command, args []string) {
				value, err := flag.Value()
				assert.Equal(t, tc.expectedValue, value)
				if tc.err != nil {
					require.Error(t, err)
					assert.IsType(t, tc.err, err)
					assert.Equal(t, tc.err.Error(), err.Error())
				} else {
					assert.NoError(t, err)
				}
			}
			require.NoError(t, cmd.Execute())
		})
	}
}
