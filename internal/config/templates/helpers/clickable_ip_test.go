package helpers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClickableIPHelper_Name(t *testing.T) {
	helper := NewClickableIPHelper(false, false)
	assert.Equal(t, "clickable_ip", helper.Name())
}

func TestClickableIPHelper_WithRender(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		render   bool
		colored  bool
		contains []string // strings that should be in the output
	}{
		{
			name:     "TTY mode - renders as hyperlink",
			ip:       "1.1.1.1",
			render:   true,
			colored:  false,
			contains: []string{"1.1.1.1", "\x1b]8;;", "platform.censys.io/hosts/1.1.1.1"},
		},
		{
			name:     "non-TTY mode - displays IP and URL",
			ip:       "1.1.1.1",
			render:   false,
			colored:  false,
			contains: []string{"1.1.1.1", "Platform URL:", "https://platform.censys.io/hosts/1.1.1.1"},
		},
		{
			name:     "TTY mode with color",
			ip:       "8.8.8.8",
			render:   true,
			colored:  true,
			contains: []string{"8.8.8.8", "\x1b]8;;", "platform.censys.io/hosts/8.8.8.8"},
		},
		{
			name:     "non-TTY mode with color",
			ip:       "8.8.8.8",
			render:   false,
			colored:  true,
			contains: []string{"8.8.8.8", "Platform URL:", "https://platform.censys.io/hosts/8.8.8.8"},
		},
		{
			name:     "empty IP",
			ip:       "",
			render:   true,
			colored:  false,
			contains: []string{},
		},
		{
			name:     "IPv6 address",
			ip:       "2001:4860:4860::8888",
			render:   true,
			colored:  false,
			contains: []string{"2001:4860:4860::8888", "\x1b]8;;", "platform.censys.io/hosts/2001:4860:4860::8888"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := NewClickableIPHelper(tc.render, tc.colored)
			fn := helper.Function().(func(string) string)
			result := fn(tc.ip)

			if tc.ip == "" {
				assert.Equal(t, "", result)
				return
			}

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected, "Expected output to contain: %s", expected)
			}

			// Verify that in non-TTY mode, we have both IP and URL on separate lines
			if !tc.render && tc.ip != "" {
				lines := strings.Split(result, "\n")
				assert.Len(t, lines, 2, "Expected 2 lines in non-TTY mode")
				assert.Contains(t, lines[1], "Platform URL:")
			}
		})
	}
}

func TestClickableIPHelper_Interface(t *testing.T) {
	helper := NewClickableIPHelper(false, false)
	var _ HandlebarsHelper = helper
	assert.NotNil(t, helper)
}
