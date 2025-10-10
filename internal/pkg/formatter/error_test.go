package formatter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

type fakeCencliError struct{ title, msg string }

func (e fakeCencliError) Title() string          { return e.title }
func (e fakeCencliError) Error() string          { return e.msg }
func (e fakeCencliError) ShouldPrintUsage() bool { return false }

var _ cenclierrors.CencliError = fakeCencliError{}

// TestPrintError already covered in formatter_test.go

func TestPrintCencliError(t *testing.T) {
	var out bytes.Buffer
	Stderr = &out
	printCencliError(fakeCencliError{title: "T", msg: "detail"}, nil)
	s := out.String()
	if !strings.Contains(s, "T") || !strings.Contains(s, "detail") {
		t.Fatalf("unexpected output: %s", s)
	}
}
