package short

import (
	"fmt"
	"strings"
)

// FormatNumber formats an int64 with comma separators.
func FormatNumber(n int64) string {
	if n < 0 {
		return "-" + FormatNumber(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	s := fmt.Sprintf("%d", n)
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}
