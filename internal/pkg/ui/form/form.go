package form

import (
	"context"
	"errors"
	"os"
	"os/signal"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

var ErrUserAborted = errors.New("user aborted")

type Form struct {
	form  *huh.Form
	theme *huh.Theme
}

func (f *Form) Init() tea.Cmd {
	return f.form.Init()
}

func (f *Form) View() string {
	return f.form.View()
}

func (f *Form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel, cmd := f.form.Update(msg)
	if huhForm, ok := newModel.(*huh.Form); ok {
		f.form = huhForm
	}
	if f.form.State == huh.StateCompleted {
		return f, tea.Quit
	}
	return f, cmd
}

type formOption func(*Form)

func WithAccessible(accessible bool) formOption {
	return func(f *Form) {
		f.form.WithAccessible(accessible)
	}
}

func NewForm(huhForm *huh.Form, opts ...formOption) *Form {
	f := &Form{
		form:  huhForm,
		theme: defaultTheme(),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// RunWithContext runs the form using the provided context and also
// listens for Ctrl-C to cancel
func (f *Form) RunWithContext(parent context.Context) error {
	ctx, cancel := signal.NotifyContext(parent, os.Interrupt)
	defer cancel()

	// Save the terminal state so we can restore it if context is cancelled
	// This prevents the terminal from being left in a broken state when
	// the context times out or is cancelled while the form is running
	var oldState *term.State
	if term.IsTerminal(int(os.Stdin.Fd())) {
		var err error
		oldState, err = term.GetState(int(os.Stdin.Fd()))
		if err != nil {
			// If we can't get the state, continue anyway - better to try than fail
			oldState = nil
		}
	}

	// Ensure terminal is restored on exit
	defer func() {
		if oldState != nil {
			// Restore the terminal to its original state
			// This is critical when context cancellation happens (timeout, Ctrl-C, etc.)
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
		}
	}()

	// Run the form in a goroutine to allow context cancellation
	// to interrupt even in accessible mode where stdin reads block
	errChan := make(chan error, 1)
	go func() {
		errChan <- f.form.WithTheme(f.theme).RunWithContext(ctx)
	}()

	// Wait for either form completion or context cancellation
	select {
	case err := <-errChan:
		if err != nil {
			if ctx.Err() != nil {
				return ErrUserAborted
			}
			if errors.Is(err, huh.ErrUserAborted) {
				return ErrUserAborted
			}
			return err
		}
		return nil
	case <-ctx.Done():
		// Context was cancelled (Ctrl-C, timeout, or parent cancellation)
		// The defer above will restore the terminal state
		// The form goroutine will eventually return but we exit immediately
		return ErrUserAborted
	}
}

// Run runs the form only erroring from explicit error returns, not from Ctrl-C
func (f *Form) Run() error {
	return f.RunWithContext(context.Background())
}
