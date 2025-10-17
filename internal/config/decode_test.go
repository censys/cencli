package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRejectNumericDurationHookFunc(t *testing.T) {
	hook := rejectNumericDurationHookFunc()
	durationType := reflect.TypeOf(time.Duration(0))

	tests := []struct {
		name        string
		fromType    reflect.Type
		toType      reflect.Type
		data        any
		expectError bool
		errorMsg    string
	}{
		{
			name:        "reject_int",
			fromType:    reflect.TypeOf(int(0)),
			toType:      durationType,
			data:        30,
			expectError: true,
			errorMsg:    "missing unit in duration",
		},
		{
			name:        "reject_int64",
			fromType:    reflect.TypeOf(int64(0)),
			toType:      durationType,
			data:        int64(30),
			expectError: true,
			errorMsg:    "missing unit in duration",
		},
		{
			name:        "reject_uint",
			fromType:    reflect.TypeOf(uint(0)),
			toType:      durationType,
			data:        uint(30),
			expectError: true,
			errorMsg:    "missing unit in duration",
		},
		{
			name:        "reject_float64",
			fromType:    reflect.TypeOf(float64(0)),
			toType:      durationType,
			data:        30.5,
			expectError: true,
			errorMsg:    "missing unit in duration",
		},
		{
			name:        "allow_string",
			fromType:    reflect.TypeOf(""),
			toType:      durationType,
			data:        "30s",
			expectError: false,
		},
		{
			name:        "allow_duration",
			fromType:    durationType,
			toType:      durationType,
			data:        30 * time.Second,
			expectError: false,
		},
		{
			name:        "ignore_non_duration_target",
			fromType:    reflect.TypeOf(int(0)),
			toType:      reflect.TypeOf(""),
			data:        30,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hook(tt.fromType, tt.toType, tt.data)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.data, result)
			}
		})
	}
}

func TestRejectNegativeDurationHookFunc(t *testing.T) {
	hook := rejectNegativeDurationHookFunc()
	durationType := reflect.TypeOf(time.Duration(0))

	tests := []struct {
		name        string
		fromType    reflect.Type
		toType      reflect.Type
		data        any
		expectError bool
		errorMsg    string
	}{
		{
			name:        "reject_negative_duration",
			fromType:    durationType,
			toType:      durationType,
			data:        -30 * time.Second,
			expectError: true,
			errorMsg:    "value cannot be negative",
		},
		{
			name:        "allow_positive_duration",
			fromType:    durationType,
			toType:      durationType,
			data:        30 * time.Second,
			expectError: false,
		},
		{
			name:        "allow_zero_duration",
			fromType:    durationType,
			toType:      durationType,
			data:        time.Duration(0),
			expectError: false,
		},
		{
			name:        "reject_negative_string",
			fromType:    reflect.TypeOf(""),
			toType:      durationType,
			data:        "-30s",
			expectError: true,
			errorMsg:    "value cannot be negative",
		},
		{
			name:        "allow_positive_string",
			fromType:    reflect.TypeOf(""),
			toType:      durationType,
			data:        "30s",
			expectError: false,
		},
		{
			name:        "allow_invalid_string",
			fromType:    reflect.TypeOf(""),
			toType:      durationType,
			data:        "invalid",
			expectError: false, // Let StringToTimeDurationHookFunc handle it
		},
		{
			name:        "ignore_non_duration_target",
			fromType:    durationType,
			toType:      reflect.TypeOf(""),
			data:        -30 * time.Second,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hook(tt.fromType, tt.toType, tt.data)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.data, result)
			}
		})
	}
}

func TestValidateUint64HookFunc(t *testing.T) {
	hook := validateUint64HookFunc()
	uint64Type := reflect.TypeOf(uint64(0))

	tests := []struct {
		name        string
		fromType    reflect.Type
		toType      reflect.Type
		data        any
		expectError bool
		errorMsg    string
	}{
		{
			name:        "reject_negative_int",
			fromType:    reflect.TypeOf(int(0)),
			toType:      uint64Type,
			data:        int(-1),
			expectError: true,
			errorMsg:    "value cannot be negative",
		},
		{
			name:        "reject_negative_int64",
			fromType:    reflect.TypeOf(int64(0)),
			toType:      uint64Type,
			data:        int64(-100),
			expectError: true,
			errorMsg:    "value cannot be negative",
		},
		{
			name:        "reject_negative_float64",
			fromType:    reflect.TypeOf(float64(0)),
			toType:      uint64Type,
			data:        float64(-1.5),
			expectError: true,
			errorMsg:    "value cannot be negative",
		},
		{
			name:        "allow_positive_int",
			fromType:    reflect.TypeOf(int(0)),
			toType:      uint64Type,
			data:        int(100),
			expectError: false,
		},
		{
			name:        "allow_zero_int",
			fromType:    reflect.TypeOf(int(0)),
			toType:      uint64Type,
			data:        int(0),
			expectError: false,
		},
		{
			name:        "allow_positive_float",
			fromType:    reflect.TypeOf(float64(0)),
			toType:      uint64Type,
			data:        float64(100.5),
			expectError: false,
		},
		{
			name:        "ignore_non_uint64_target",
			fromType:    reflect.TypeOf(int(0)),
			toType:      reflect.TypeOf(int(0)),
			data:        int(-1),
			expectError: false,
		},
		{
			name:        "allow_string",
			fromType:    reflect.TypeOf(""),
			toType:      uint64Type,
			data:        "100",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hook(tt.fromType, tt.toType, tt.data)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.data, result)
			}
		})
	}
}
