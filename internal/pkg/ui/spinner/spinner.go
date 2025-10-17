package spinner

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/term"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Start starts a spinner in stderr and returns an idempotent stop function.
// Callers must provide a channel to ensure the spinner eventually stops.
func Start(ctxDone <-chan struct{}, disabled bool, opts ...ComponentOption) (stop func()) {
	// if this is manually disabled by user config, return no-op
	if disabled {
		return func() {}
	}
	// if stderr is not a TTY, return no-op
	if !term.IsTTY(formatter.Stderr) {
		return func() {}
	}
	start, stop, _ := newSpinner(formatter.Stderr, opts...)
	start(ctxDone)
	return stop
}

type spinnerComponent struct {
	program     *tea.Program
	done        chan struct{}
	programDone chan struct{} // signals when the bubbletea program has finished
	out         io.Writer
	once        sync.Once
	design      spinner.Spinner
	message     string
	mu          sync.Mutex
}

type spinnerComponentOptions struct {
	design             spinner.Spinner
	message            string
	stopwatchEnabled   bool
	stopwatchStartSecs uint
}

// ComponentOption configures the spinner component behavior.
type ComponentOption func(*spinnerComponentOptions)

func WithDesign(design spinner.Spinner) ComponentOption {
	return func(o *spinnerComponentOptions) {
		o.design = design
	}
}

func WithMessage(message string) ComponentOption {
	return func(o *spinnerComponentOptions) {
		o.message = message
	}
}

func WithStopwatch(startAfterSeconds uint) ComponentOption {
	return func(o *spinnerComponentOptions) {
		o.stopwatchEnabled = true
		o.stopwatchStartSecs = startAfterSeconds
	}
}

// Handle allows callers to update the spinner's message and stop it.
type Handle interface {
	Stop()
	SetMessage(message string)
}

type noopHandle struct{}

func (noopHandle) Stop()             {}
func (noopHandle) SetMessage(string) {}

// NewNoopHandle returns a handle that performs no work.
func NewNoopHandle() Handle { return noopHandle{} }

// newSpinner creates a new spinner component with options.
// Returns start and stop functions. Stop is idempotent.
func newSpinner(out io.Writer, opts ...ComponentOption) (startWithContext func(done <-chan struct{}), stop func(), comp *spinnerComponent) {
	spinnerOpts := &spinnerComponentOptions{
		design:  spinner.Dot,
		message: "Loading...",
	}
	for _, opt := range opts {
		opt(spinnerOpts)
	}

	s := &spinnerComponent{
		done:        make(chan struct{}),
		programDone: make(chan struct{}),
		out:         out,
		design:      spinnerOpts.design,
		message:     spinnerOpts.message,
		once:        sync.Once{},
	}

	// Wrap StartWithContext to pass the stopwatch options
	startWrapper := func(done <-chan struct{}) {
		s.startWithContext(done, spinnerOpts.stopwatchEnabled, spinnerOpts.stopwatchStartSecs)
	}

	return startWrapper, s.Stop, s
}

// startWithContext stops the spinner when the context is done.
func (s *spinnerComponent) startWithContext(ctxDone <-chan struct{}, stopwatchEnabled bool, stopwatchStartSecs uint) {
	m := spinner.New()
	m.Spinner = s.design

	prog := tea.NewProgram(
		model{
			spinner:            m,
			done:               s.done,
			message:            s.message,
			stopwatchEnabled:   stopwatchEnabled,
			stopwatchStartSecs: stopwatchStartSecs,
			startTime:          time.Now(),
		},
		tea.WithoutSignalHandler(), // Let parent handle signals
		tea.WithOutput(s.out),
		tea.WithInput(nil), // Disable input to avoid capturing Ctrl-C
	)
	s.mu.Lock()
	s.program = prog
	s.mu.Unlock()

	go func() {
		defer close(s.programDone)
		_, _ = s.program.Run()
	}()

	go func() {
		<-ctxDone
		s.Stop()
	}()
}

func (s *spinnerComponent) Stop() {
	s.once.Do(func() {
		close(s.done)
		s.mu.Lock()
		prog := s.program
		s.mu.Unlock()
		if prog != nil {
			prog.Quit()
			// Wait for the bubbletea program to finish before clearing
			<-s.programDone
			// Clear the spinner line after quitting to prevent residual text
			// \r moves cursor to start of line, \033[K clears from cursor to end of line
			fmt.Fprint(s.out, "\r\033[K")
		}
	})
}

// SetMessage updates the spinner's message in a thread-safe way.
func (s *spinnerComponent) SetMessage(message string) {
	s.mu.Lock()
	prog := s.program
	s.mu.Unlock()
	if prog != nil {
		prog.Send(setMessageMsg{text: message})
	}
}

type model struct {
	spinner            spinner.Model
	message            string
	done               chan struct{}
	stopwatchEnabled   bool
	stopwatchStartSecs uint
	startTime          time.Time
	elapsedSeconds     uint
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick, waitForDone(m.done)}
	if m.stopwatchEnabled {
		cmds = append(cmds, stopwatchTick())
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case stopwatchTickMsg:
		if m.stopwatchEnabled {
			elapsed := time.Since(m.startTime)
			m.elapsedSeconds = uint(elapsed.Seconds())
			return m, stopwatchTick()
		}
		return m, nil
	case doneMsg:
		return m, tea.Quit
	case setMessageMsg:
		m.message = msg.text
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	msg := m.message
	if m.stopwatchEnabled && m.elapsedSeconds >= m.stopwatchStartSecs {
		msg = fmt.Sprintf("%s (time elapsed: %ds)", m.message, m.elapsedSeconds)
	}
	return fmt.Sprintf("\r%s %s", m.spinner.View(), msg)
}

type doneMsg struct{}

func waitForDone(done chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-done
		return doneMsg{}
	}
}

type setMessageMsg struct{ text string }

type stopwatchTickMsg struct{}

func stopwatchTick() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return stopwatchTickMsg{}
	})
}

// StartWithHandle starts a spinner and returns a handle that can update the message and stop the spinner.
func StartWithHandle(ctxDone <-chan struct{}, disabled bool, opts ...ComponentOption) Handle {
	if disabled {
		return noopHandle{}
	}
	if !term.IsTTY(formatter.Stderr) {
		return noopHandle{}
	}
	start, _, comp := newSpinner(formatter.Stderr, opts...)
	start(ctxDone)
	return comp
}
