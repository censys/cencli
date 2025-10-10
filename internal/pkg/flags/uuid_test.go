package flags

import (
	"testing"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	uuidFlagName  = "test-uuid-flag"
	uuidFlagShort = "u"
)

func TestUUIDFlag(t *testing.T) {
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := []struct {
		name          string
		required      bool
		defaultValue  mo.Option[uuid.UUID]
		args          []string
		expectedValue mo.Option[uuid.UUID]
		expectError   bool
		expectedError error
	}{
		{
			name:          "valid uuid - required",
			required:      true,
			defaultValue:  mo.None[uuid.UUID](),
			args:          []string{"--" + uuidFlagName, "123e4567-e89b-12d3-a456-426614174000"},
			expectedValue: mo.Some(validUUID),
			expectError:   false,
		},
		{
			name:          "valid uuid - required - shorthand",
			required:      true,
			defaultValue:  mo.None[uuid.UUID](),
			args:          []string{"-" + uuidFlagShort, "123e4567-e89b-12d3-a456-426614174000"},
			expectedValue: mo.Some(validUUID),
			expectError:   false,
		},
		{
			name:          "valid uuid - optional",
			required:      false,
			defaultValue:  mo.None[uuid.UUID](),
			args:          []string{"--" + uuidFlagName, "123e4567-e89b-12d3-a456-426614174000"},
			expectedValue: mo.Some(validUUID),
			expectError:   false,
		},
		{
			name:          "optional flag not set - with default value",
			required:      false,
			defaultValue:  mo.Some(validUUID),
			args:          []string{},
			expectedValue: mo.Some(validUUID),
			expectError:   false,
		},
		{
			name:          "optional flag not set - no default value",
			required:      false,
			defaultValue:  mo.None[uuid.UUID](),
			args:          []string{},
			expectedValue: mo.None[uuid.UUID](),
			expectError:   false,
		},
		{
			name:          "required flag not set",
			required:      true,
			defaultValue:  mo.None[uuid.UUID](),
			args:          []string{},
			expectedValue: mo.None[uuid.UUID](),
			expectError:   true,
			expectedError: NewRequiredFlagNotSetError(uuidFlagName),
		},
		{
			name:          "invalid uuid",
			required:      false,
			defaultValue:  mo.None[uuid.UUID](),
			args:          []string{"--" + uuidFlagName, "invalid"},
			expectedValue: mo.None[uuid.UUID](),
			expectError:   true,
			expectedError: NewInvalidUUIDFlagError(uuidFlagName, "invalid"),
		},
		{
			name:          "valid uuid with spaces",
			required:      false,
			defaultValue:  mo.None[uuid.UUID](),
			args:          []string{"--" + uuidFlagName, "  123e4567-e89b-12d3-a456-426614174000  "},
			expectedValue: mo.Some(validUUID),
			expectError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			flag := NewUUIDFlag(cmd.Flags(), tc.required, uuidFlagName, uuidFlagShort, tc.defaultValue, "A UUID Flag")
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
