package helpers

import (
	"fmt"
	"reflect"
	"strings"
)

type concatHelper struct{}

var _ HandlebarsHelper = &concatHelper{}

// NewConcatHelper creates a helper that concatenates multiple string arguments.
func NewConcatHelper() HandlebarsHelper {
	return &concatHelper{}
}

func (h *concatHelper) Name() string {
	return "concat"
}

func (h *concatHelper) Function() any {
	return func(items any) string {
		if items == nil {
			return ""
		}

		val := reflect.ValueOf(items)
		switch val.Kind() {
		case reflect.Slice, reflect.Array:
			if val.Len() == 0 {
				return ""
			}

			var parts []string
			for i := 0; i < val.Len(); i++ {
				item := val.Index(i)
				str := fmt.Sprint(item.Interface())
				if str != "" {
					parts = append(parts, str)
				}
			}
			return strings.Join(parts, "")
		default:
			// If not an array/slice, just convert to string and filter empty
			str := fmt.Sprint(items)
			if str == "" {
				return ""
			}
			return str
		}
	}
}
