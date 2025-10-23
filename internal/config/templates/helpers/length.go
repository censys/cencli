package helpers

import (
	"fmt"
	"reflect"
)

type lengthHelper struct{}

var _ HandlebarsHelper = &lengthHelper{}

func NewLengthHelper() HandlebarsHelper {
	return &lengthHelper{}
}

func (h *lengthHelper) Name() string {
	return "length"
}

func (h *lengthHelper) Function() any {
	return func(v any) string {
		if v == nil {
			return "0"
		}
		val := reflect.ValueOf(v)
		switch val.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
			return fmt.Sprintf("%d", val.Len())
		default:
			return "0"
		}
	}
}
