// Package command provides CLI command implementations and helpers.
package command

import (
	"context"
	"log/slog"
	"sync"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/pkg/ui/spinner"
)

// progressDisplay manages both spinner UI and debug logging for progress events.
type progressDisplay struct {
	handle         spinner.Handle
	disableSpinner bool
	logger         *slog.Logger
}

// newProgressDisplay creates a progress display that shows a spinner (if enabled) and logs events.
// The spinner is only shown when disableSpinner is false (interactive mode).
// Progress events are always logged at debug level regardless of spinner state.
func newProgressDisplay(ctx context.Context, log *slog.Logger, disableSpinner bool, initialMessage string) progressDisplay {
	handle := spinner.NewNoopHandle()
	if !disableSpinner && initialMessage != "" {
		handle = spinner.StartWithHandle(ctx.Done(), false, spinner.WithMessage(initialMessage))
	} else if !disableSpinner {
		handle = spinner.StartWithHandle(ctx.Done(), false)
	}
	return progressDisplay{handle: handle, disableSpinner: disableSpinner, logger: log}
}

// render displays a progress event via spinner (if enabled) and always logs it.
// Message priority: event.Message > stage name.
func (d progressDisplay) render(event progress.Event) {
	msg := event.Message
	if msg == "" {
		msg = string(event.Stage)
	}

	d.logger.Debug("progress", "stage", event.Stage, "message", msg)

	// Update spinner if enabled
	if !d.disableSpinner && msg != "" {
		d.handle.SetMessage(msg)
	}
}

// stop terminates the spinner display.
func (d progressDisplay) stop() { d.handle.Stop() }

// startProgress sets up progress reporting infrastructure.
// It creates a publisher, attaches it to the context, starts a goroutine to consume events,
// and returns a new context and a stop function.
func (c *Context) startProgress(
	ctx context.Context,
	logger *slog.Logger,
	initialMessage string,
) (context.Context, func(error)) {
	disableSpinner := c.config.Spinner.Disabled || c.config.Quiet

	pub, events := progress.NewChannelPublisher(0)
	display := newProgressDisplay(ctx, logger, disableSpinner, initialMessage)

	derived := progress.WithPublisher(ctx, pub)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for event := range events {
			if event.Done {
				break
			}
			display.render(event)
		}
	}()

	var once sync.Once
	stop := func(finalErr error) {
		once.Do(func() {
			pub.Close(finalErr)
			<-done
			display.stop()
		})
	}

	return derived, stop
}
