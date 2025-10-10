package command

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestArgCountError(t *testing.T) {
	t.Run("Error method", func(t *testing.T) {
		baseErr := errors.New("test error")
		err := NewArgCountError(baseErr)
		// The actual error includes styling, so just check it contains the message
		assert.Contains(t, err.Error(), "test error")
	})

	t.Run("Title method", func(t *testing.T) {
		baseErr := errors.New("test error")
		err := NewArgCountError(baseErr)
		assert.Equal(t, "Incorrect Number of Arguments", err.Title())
	})

	t.Run("ShouldPrintUsage", func(t *testing.T) {
		baseErr := errors.New("test error")
		err := NewArgCountError(baseErr)
		assert.True(t, err.ShouldPrintUsage())
	})
}

func TestExactArgs(t *testing.T) {
	tests := []struct {
		name        string
		n           int
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "exact match",
			n:           2,
			args:        []string{"arg1", "arg2"},
			expectError: false,
		},
		{
			name:        "too few args",
			n:           3,
			args:        []string{"arg1", "arg2"},
			expectError: true,
			errorMsg:    "accepts 3 arg(s), received 2",
		},
		{
			name:        "too many args",
			n:           1,
			args:        []string{"arg1", "arg2"},
			expectError: true,
			errorMsg:    "accepts 1 arg(s), received 2",
		},
		{
			name:        "zero args expected and received",
			n:           0,
			args:        []string{},
			expectError: false,
		},
		{
			name:        "zero args expected but received some",
			n:           0,
			args:        []string{"arg1"},
			expectError: true,
			errorMsg:    "accepts 0 arg(s), received 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := ExactArgs(tt.n)
			cmd := &cobra.Command{Use: "test"}

			err := validator(cmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
				var argErr ArgCountError
				if errors.As(err, &argErr) {
					assert.Contains(t, argErr.Error(), tt.errorMsg)
					assert.True(t, argErr.ShouldPrintUsage())
				} else {
					t.Fatalf("Expected ArgCountError, got %T", err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRangeArgs(t *testing.T) {
	tests := []struct {
		name        string
		min         int
		max         int
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "within range",
			min:         1,
			max:         3,
			args:        []string{"arg1", "arg2"},
			expectError: false,
		},
		{
			name:        "at minimum",
			min:         2,
			max:         4,
			args:        []string{"arg1", "arg2"},
			expectError: false,
		},
		{
			name:        "at maximum",
			min:         1,
			max:         3,
			args:        []string{"arg1", "arg2", "arg3"},
			expectError: false,
		},
		{
			name:        "below minimum",
			min:         2,
			max:         4,
			args:        []string{"arg1"},
			expectError: true,
			errorMsg:    "accepts between 2 and 4 arg(s), received 1",
		},
		{
			name:        "above maximum",
			min:         1,
			max:         2,
			args:        []string{"arg1", "arg2", "arg3"},
			expectError: true,
			errorMsg:    "accepts between 1 and 2 arg(s), received 3",
		},
		{
			name:        "zero to many",
			min:         0,
			max:         5,
			args:        []string{},
			expectError: false,
		},
		{
			name:        "exactly one (min=max)",
			min:         1,
			max:         1,
			args:        []string{"arg1"},
			expectError: false,
		},
		{
			name:        "exactly one but got two (min=max)",
			min:         1,
			max:         1,
			args:        []string{"arg1", "arg2"},
			expectError: true,
			errorMsg:    "accepts between 1 and 1 arg(s), received 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := RangeArgs(tt.min, tt.max)
			cmd := &cobra.Command{Use: "test"}

			err := validator(cmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
				var argErr ArgCountError
				if errors.As(err, &argErr) {
					assert.Contains(t, argErr.Error(), tt.errorMsg)
					assert.True(t, argErr.ShouldPrintUsage())
				} else {
					t.Fatalf("Expected ArgCountError, got %T", err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPositionalArgs(t *testing.T) {
	t.Run("ExactArgs returns PositionalArgs", func(t *testing.T) {
		validator := ExactArgs(2)
		assert.NotNil(t, validator)
		// Verify it's a function that can be called
		cmd := &cobra.Command{Use: "test"}
		err := validator(cmd, []string{"arg1", "arg2"})
		assert.NoError(t, err)
	})

	t.Run("RangeArgs returns PositionalArgs", func(t *testing.T) {
		validator := RangeArgs(1, 3)
		assert.NotNil(t, validator)
		// Verify it's a function that can be called
		cmd := &cobra.Command{Use: "test"}
		err := validator(cmd, []string{"arg1", "arg2"})
		assert.NoError(t, err)
	})
}
