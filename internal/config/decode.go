package config

import (
	"fmt"
	"reflect"
	"time"

	"github.com/go-viper/mapstructure/v2"
)

// rejectNumericDurationHookFunc disallows numeric values for time.Duration fields,
// forcing users to include an explicit unit (e.g., "30s", "2m").
func rejectNumericDurationHookFunc() mapstructure.DecodeHookFuncType {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if to == reflect.TypeOf(time.Duration(0)) {
			// Allow duration-to-duration conversions (already validated)
			if from == reflect.TypeOf(time.Duration(0)) {
				return data, nil
			}
			switch from.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64:
				return nil, fmt.Errorf("missing unit in duration")
			}
		}
		return data, nil
	}
}

// rejectNegativeDurationHookFunc disallows negative values for time.Duration fields.
func rejectNegativeDurationHookFunc() mapstructure.DecodeHookFuncType {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		// Only validate when target type is time.Duration
		if to != reflect.TypeOf(time.Duration(0)) {
			return data, nil
		}

		// Check if the source is already a time.Duration (from viper or flags)
		if from == reflect.TypeOf(time.Duration(0)) {
			if val, ok := data.(time.Duration); ok && val < 0 {
				return nil, fmt.Errorf("value cannot be negative: %s", val)
			}
		}

		// Check if the source is a string that would be parsed as a duration
		if from.Kind() == reflect.String {
			if str, ok := data.(string); ok {
				// Try to parse it as a duration to validate
				duration, err := time.ParseDuration(str)
				if err != nil {
					// Let the StringToTimeDurationHookFunc handle the error
					return data, nil
				}
				if duration < 0 {
					return nil, fmt.Errorf("value cannot be negative: %s", duration)
				}
			}
		}

		return data, nil
	}
}

// validateUint64HookFunc validates that values being decoded to uint64 are not negative.
// This prevents negative values from wrapping around to large positive numbers.
func validateUint64HookFunc() mapstructure.DecodeHookFuncType {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if to.Kind() == reflect.Uint64 {
			switch from.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				// Check if the signed integer is negative
				val := reflect.ValueOf(data)
				if val.Int() < 0 {
					return nil, fmt.Errorf("value cannot be negative")
				}
			case reflect.Float32, reflect.Float64:
				// Check if the float is negative
				val := reflect.ValueOf(data)
				if val.Float() < 0 {
					return nil, fmt.Errorf("value cannot be negative")
				}
			}
		}
		return data, nil
	}
}
