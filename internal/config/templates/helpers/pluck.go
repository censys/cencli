package helpers

import (
	"reflect"
)

type pluckHelper struct{}

var _ HandlebarsHelper = &pluckHelper{}

// NewPluckHelper creates a helper that extracts values from an array of objects/maps by key
// and filters out falsy values (nil, empty string, false, 0).
func NewPluckHelper() HandlebarsHelper {
	return &pluckHelper{}
}

func (h *pluckHelper) Name() string {
	return "pluck"
}

func (h *pluckHelper) Function() any {
	return func(array any, key string) any {
		if array == nil {
			return []any{}
		}

		val := reflect.ValueOf(array)
		switch val.Kind() {
		case reflect.Slice, reflect.Array:
			if val.Len() == 0 {
				return []any{}
			}

			result := make([]any, 0, val.Len())
			for i := 0; i < val.Len(); i++ {
				item := val.Index(i)
				var extracted any

				// Handle different types of items
				switch item.Kind() {
				case reflect.Map:
					// For maps, use the key directly
					if mapValue, ok := item.Interface().(map[string]any); ok && mapValue != nil {
						extracted = mapValue[key]
					}
				case reflect.Struct:
					// For structs, try to get field by name
					if field := item.FieldByName(key); field.IsValid() {
						extracted = field.Interface()
					}
				case reflect.Ptr:
					// Handle pointers to structs/maps
					if !item.IsNil() {
						elem := item.Elem()
						switch elem.Kind() {
						case reflect.Map:
							if mapValue, ok := elem.Interface().(map[string]any); ok && mapValue != nil {
								extracted = mapValue[key]
							}
						case reflect.Struct:
							if field := elem.FieldByName(key); field.IsValid() {
								extracted = field.Interface()
							}
						}
					}
				default:
					// For other types, try to treat as map if possible
					if item.CanInterface() {
						if mapValue, ok := item.Interface().(map[string]any); ok {
							extracted = mapValue[key]
						}
					}
				}

				// Only include non-falsy values
				if !isFalsy(extracted) {
					result = append(result, extracted)
				}
			}
			return result
		default:
			// If not an array/slice, return empty array
			return []any{}
		}
	}
}

// isFalsy determines if a value is falsy (nil, empty string, false, 0)
func isFalsy(value any) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case bool:
		return !v
	case string:
		return v == ""
	default:
		// For numeric types, check if they're zero
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return rv.Int() == 0
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return rv.Uint() == 0
		case reflect.Float32, reflect.Float64:
			return rv.Float() == 0
		case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan, reflect.Func:
			return rv.IsNil() || rv.Len() == 0
		case reflect.Ptr, reflect.Interface:
			return rv.IsNil()
		}
	}
	return false
}
