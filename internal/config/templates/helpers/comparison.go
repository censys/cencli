package helpers

import (
	"reflect"
)

type comparisonHelper struct {
	name string
	op   string
}

var _ HandlebarsHelper = &comparisonHelper{}

// NewLessThanHelper creates a helper that returns true if a < b.
func NewLessThanHelper() HandlebarsHelper {
	return &comparisonHelper{name: "lt", op: "lt"}
}

// NewGreaterThanHelper creates a helper that returns true if a > b.
func NewGreaterThanHelper() HandlebarsHelper {
	return &comparisonHelper{name: "gt", op: "gt"}
}

// NewEqualHelper creates a helper that returns true if a == b.
func NewEqualHelper() HandlebarsHelper {
	return &comparisonHelper{name: "eq", op: "eq"}
}

func (h *comparisonHelper) Name() string {
	return h.name
}

func (h *comparisonHelper) Function() any {
	return func(a, b any) bool {
		return compare(a, b, h.op)
	}
}

func compare(a, b any, op string) bool {
	if a == nil || b == nil {
		if op == "eq" {
			return a == b
		}
		return false
	}

	aVal := reflect.ValueOf(a)
	bVal := reflect.ValueOf(b)

	// Try to compare as numbers
	aFloat, aOk := toFloat64(aVal)
	bFloat, bOk := toFloat64(bVal)

	if aOk && bOk {
		switch op {
		case "lt":
			return aFloat < bFloat
		case "gt":
			return aFloat > bFloat
		case "eq":
			return aFloat == bFloat
		}
	}

	// Try to compare as strings
	aStr, aOk := toString(aVal)
	bStr, bOk := toString(bVal)

	if aOk && bOk {
		switch op {
		case "lt":
			return aStr < bStr
		case "gt":
			return aStr > bStr
		case "eq":
			return aStr == bStr
		}
	}

	// Fall back to equality check using reflect.DeepEqual
	if op == "eq" {
		return reflect.DeepEqual(a, b)
	}

	return false
}

func toFloat64(v reflect.Value) (float64, bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	default:
		return 0, false
	}
}

func toString(v reflect.Value) (string, bool) {
	if v.Kind() == reflect.String {
		return v.String(), true
	}
	return "", false
}
