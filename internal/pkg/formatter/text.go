package formatter

import (
	"reflect"
	"strconv"
	"time"
)

// TruncateEnd returns s truncated to max characters with an ellipsis suffix if needed.
func TruncateEnd(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// FormatShortTime renders a timestamp in a compact, human-friendly format.
func FormatShortTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// Int64String returns the base-10 string representation of v.
func Int64String(v int64) string {
	return strconv.FormatInt(v, 10)
}

// CountItems returns the number of items if v is a slice/array; otherwise 1.
func CountItems(v any) int {
	rv := reflect.ValueOf(v)
	if rv.IsValid() {
		k := rv.Kind()
		if k == reflect.Slice || k == reflect.Array {
			return rv.Len()
		}
	}
	return 1
}
