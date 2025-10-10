package spinner

import (
	"io"
	"testing"
	"time"
)

func TestStart_NoTTY_NoOp(t *testing.T) {
	t.Setenv("NO_TTY", "1")
	done := make(chan struct{})
	stop := Start(done, false)
	// Should be a no-op and safe to call twice
	stop()
	stop()
}

func TestStop_Idempotent(t *testing.T) {
	t.Setenv("NO_TTY", "1")
	done := make(chan struct{})
	// disabled returns no-op
	stop := Start(done, true)
	stop()
	// Now enable; still no TTY so returns no-op
	stop = Start(done, false)
	close(done)
	// Allow any goroutine to process (even though no spinner started)
	time.Sleep(10 * time.Millisecond)
	stop()
}

func TestStartWithContext_StartsAndStops(t *testing.T) {
	t.Setenv("NO_TTY", "1")
	done := make(chan struct{})
	stop := Start(done, false)
	// Simulate operation finishing quickly
	close(done)
	time.Sleep(5 * time.Millisecond)
	// Should be safe even if already stopped via context
	stop()
}

func TestStartWithContext_InternalRunner(t *testing.T) {
	// Bypass TTY gating by using internal spinner starter directly
	start, stop, _ := newSpinner(io.Discard, WithMessage("Testing..."))
	done := make(chan struct{})
	start(done)
	// End quickly
	close(done)
	time.Sleep(10 * time.Millisecond)
	// Idempotent stop
	stop()
	stop()
}
