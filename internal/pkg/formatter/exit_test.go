package formatter

import (
	"context"
	"errors"
	"testing"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

type usageErr struct{ msg string }

func (e usageErr) Title() string          { return "Usage" }
func (e usageErr) Error() string          { return e.msg }
func (e usageErr) ShouldPrintUsage() bool { return true }

type generalErr struct{ msg string }

func (e generalErr) Title() string          { return "General" }
func (e generalErr) Error() string          { return e.msg }
func (e generalErr) ShouldPrintUsage() bool { return false }

var (
	_ cenclierrors.CencliError = usageErr{}
	_ cenclierrors.CencliError = generalErr{}
)

func TestExitCode_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"deadline", context.DeadlineExceeded, 124},
		{"canceled", context.Canceled, 130},
		{"usage", usageErr{"bad args"}, 2},
		{"general", generalErr{"boom"}, 1},
		{"wrapped cencli", cenclierrors.NewCencliError(errors.New("oops")), 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Fatalf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}
