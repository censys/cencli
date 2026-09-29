package command

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestWriteDetailField(t *testing.T) {
	var out strings.Builder
	WriteDetailField(&out, "ID", "abc")
	WriteDetailField(&out, "Completed", "yes")

	assert.Equal(t, "  ID:           abc\n  Completed:    yes\n", ansi.ReplaceAllString(out.String(), ""))
}
