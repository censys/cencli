package convertutil

import (
	"github.com/samber/mo"
)

// OptionalString converts an option of a Stringer-like type into an option of string.
// If the option is empty, it returns mo.None[string]().
func OptionalString[T interface{ String() string }](opt mo.Option[T]) mo.Option[string] {
	if !opt.IsPresent() {
		return mo.None[string]()
	}
	return mo.Some(opt.MustGet().String())
}

// Stringify converts a slice of Stringer-like values into a slice of strings.
func Stringify[T interface{ String() string }](items []T) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.String())
	}
	return out
}
