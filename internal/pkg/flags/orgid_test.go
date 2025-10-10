package flags

import (
	"testing"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

func TestOrgIDFlag(t *testing.T) {
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	validOrgID := identifiers.NewOrganizationID(validUUID)

	tests := []struct {
		name          string
		args          []string
		expectedValue mo.Option[identifiers.OrganizationID]
		expectError   bool
		expectedError error
	}{
		{
			name:          "valid org id",
			args:          []string{"--org-id", "123e4567-e89b-12d3-a456-426614174000"},
			expectedValue: mo.Some(validOrgID),
			expectError:   false,
		},
		{
			name:          "invalid org id",
			args:          []string{"--org-id", "invalid"},
			expectedValue: mo.None[identifiers.OrganizationID](),
			expectError:   true,
			expectedError: NewInvalidUUIDFlagError("org-id", "invalid"),
		},
		{
			name:          "flag not set",
			args:          []string{},
			expectedValue: mo.None[identifiers.OrganizationID](),
			expectError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			orgFlag := NewOrgIDFlag(cmd.Flags(), "")
			cmd.SetArgs(tc.args)
			cmd.Run = func(cmd *cobra.Command, args []string) {
				value, err := orgFlag.Value()

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

func TestOrgIDFlag_Properties(t *testing.T) {
	cmd := &cobra.Command{}
	NewOrgIDFlag(cmd.Flags(), "")

	// Check that the flag exists and has correct properties
	flag := cmd.Flags().Lookup("org-id")
	require.NotNil(t, flag, "org-id flag should exist")
	assert.Equal(t, "override the configured organization ID", flag.Usage)
	assert.Equal(t, "o", flag.Shorthand)
	assert.Equal(t, "", flag.DefValue) // Should have no default value
}

func TestOrgIDFlag_ShortOverride(t *testing.T) {
	cmd := &cobra.Command{}
	orgFlag := NewOrgIDFlag(cmd.Flags(), "x")

	// Check that the short override works
	flag := cmd.Flags().Lookup("org-id")
	require.NotNil(t, flag, "org-id flag should exist")
	assert.Equal(t, "x", flag.Shorthand)

	// Test that the short override actually works
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	validOrgID := identifiers.NewOrganizationID(validUUID)

	cmd.SetArgs([]string{"-x", "123e4567-e89b-12d3-a456-426614174000"})
	cmd.Run = func(cmd *cobra.Command, args []string) {
		value, err := orgFlag.Value()
		require.NoError(t, err)
		assert.Equal(t, mo.Some(validOrgID), value)
	}
	require.NoError(t, cmd.Execute())
}
