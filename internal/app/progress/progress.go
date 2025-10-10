package progress

import (
	"context"
	"errors"
	"sync"
)

// Stage identifies the conceptual phase of an operation emitting progress events.
type Stage string

const (
	// StagePrepare indicates the preparation phase before main work begins.
	// Used for validation, configuration loading, and setup tasks.
	StagePrepare Stage = "prepare"

	// StageFetch indicates network or I/O operations to retrieve data.
	// Used for API calls, database queries, and file reads.
	StageFetch Stage = "fetch"

	// StageProcess indicates computation or transformation of data.
	// Used for parsing, filtering, aggregation, and business logic.
	StageProcess Stage = "process"

	// StageRender indicates formatting and output of results.
	// Used for template rendering, serialization, and display preparation.
	StageRender Stage = "render"
)

// Event conveys progress for a long-running operation.
type Event struct {
	Stage   Stage
	Message string
	Done    bool
	Err     error
}

// Publisher emits progress events to interested listeners.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
	Close(err error)
}

var ErrPublisherClosed = errors.New("progress publisher closed")

const defaultBufferSize = 1

func NewChannelPublisher(buffer int) (Publisher, <-chan Event) {
	if buffer <= 0 {
		buffer = defaultBufferSize
	}
	ch := make(chan Event, buffer)
	return &channelPublisher{ch: ch}, ch
}

type channelPublisher struct {
	ch     chan Event
	once   sync.Once
	mu     sync.Mutex
	closed bool
}

func (p *channelPublisher) Publish(ctx context.Context, event Event) error {
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if closed {
		return ErrPublisherClosed
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.ch <- event:
		return nil
	}
}

func (p *channelPublisher) Close(err error) {
	p.once.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()

		final := Event{Done: true, Err: err}
		select {
		case p.ch <- final:
		default:
		}
		close(p.ch)
	})
}

type publisherContextKey struct{}

func WithPublisher(ctx context.Context, pub Publisher) context.Context {
	if pub == nil {
		return ctx
	}
	return context.WithValue(ctx, publisherContextKey{}, pub)
}

func FromContext(ctx context.Context) (Publisher, bool) {
	pub, ok := ctx.Value(publisherContextKey{}).(Publisher)
	return pub, ok
}

// Publish emits a progress event to the context-bound publisher.
// Returns an error if publishing fails (context canceled, publisher closed, etc.).
// Most callers should use the Report* helpers which ignore errors.
func Publish(ctx context.Context, event Event) error {
	pub, ok := FromContext(ctx)
	if !ok {
		return nil
	}
	return pub.Publish(ctx, event)
}

// ReportStage emits a stage transition event.
// Errors are silently ignored per the package error handling policy.
func ReportStage(ctx context.Context, stage Stage) {
	_ = Publish(ctx, Event{Stage: stage})
}

// ReportMessage emits a stage event with a descriptive message.
// Errors are silently ignored per the package error handling policy.
func ReportMessage(ctx context.Context, stage Stage, message string) {
	_ = Publish(ctx, Event{Stage: stage, Message: message})
}

// ReportError emits a stage event with an error message.
// This is useful for communicating non-fatal errors during batching or pagination
// so users can decide whether to cancel or continue.
// Errors are silently ignored per the package error handling policy.
func ReportError(ctx context.Context, stage Stage, err error) {
	if err != nil {
		_ = Publish(ctx, Event{Stage: stage, Message: err.Error(), Err: err})
	}
}
