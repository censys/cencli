package helpers

import (
	"fmt"
	"reflect"
	"strings"
)

type joinHelper struct{}

var _ HandlebarsHelper = &joinHelper{}

// NewJoinHelper creates a helper that joins a list of strings with a delimiter.
func NewJoinHelper() HandlebarsHelper {
	return &joinHelper{}
}

func (h *joinHelper) Name() string {
	return "join"
}

func (h *joinHelper) Function() any {
	return func(list any, delimiter string) string {
		if list == nil {
			return ""
		}

		val := reflect.ValueOf(list)
		switch val.Kind() {
		case reflect.Slice, reflect.Array:
			if val.Len() == 0 {
				return ""
			}
			parts := make([]string, val.Len())
			for i := 0; i < val.Len(); i++ {
				parts[i] = fmt.Sprint(val.Index(i).Interface())
			}
			return strings.Join(parts, delimiter)
		default:
			return fmt.Sprint(list)
		}
	}
}
