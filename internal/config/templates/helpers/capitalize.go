package helpers

import (
	"fmt"
	"strings"
)

type capitalizeHelper struct{}

var _ HandlebarsHelper = &capitalizeHelper{}

// NewCapitalizeHelper creates a helper that capitalizes the first letter of a string,
// or all words depending on the mode argument.
func NewCapitalizeHelper() HandlebarsHelper {
	return &capitalizeHelper{}
}

func (h *capitalizeHelper) Name() string {
	return "capitalize"
}

func (h *capitalizeHelper) Function() any {
	return func(v any, mode string) string {
		str := fmt.Sprint(v)
		if len(str) == 0 {
			return ""
		}

		// If mode is empty, default to "first"
		if mode == "" {
			mode = "first"
		}

		switch mode {
		case "all":
			// Uppercase all letters
			return strings.ToUpper(str)
		case "first":
			fallthrough
		default:
			// Capitalize only the first letter of the string
			return strings.ToUpper(str[:1]) + str[1:]
		}
	}
}
