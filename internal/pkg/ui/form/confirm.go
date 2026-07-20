package form

import (
	"context"

	"github.com/charmbracelet/huh"
)

// Confirm prompts the user with a yes/no question and returns their answer.
// It reuses the shared form wrapper, so it inherits the default theme, terminal
// restoration, and Ctrl-C handling (returning ErrUserAborted on interrupt).
//
// The caller is responsible for gating this on an interactive terminal; in a
// non-TTY context the underlying form has nothing to read and the caller should
// avoid prompting at all.
func Confirm(ctx context.Context, message string) (bool, error) {
	var confirmed bool
	f := NewForm(
		huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(message).
					Affirmative("Yes").
					Negative("No").
					Value(&confirmed),
			),
		),
	)
	if err := f.RunWithContext(ctx); err != nil {
		return false, err
	}
	return confirmed, nil
}
