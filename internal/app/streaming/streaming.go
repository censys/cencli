package streaming

import (
	"context"
	"errors"
	"sync"
)

// Item represents a single piece of data emitted through the streaming channel.
type Item struct {
	// Data is the actual payload being streamed.
	Data any
	// Err indicates an error that occurred during streaming (non-fatal).
	Err error
	// Done signals that streaming has completed.
	Done bool
}

// Emitter sends data items to a consumer.
type Emitter interface {
	// Emit sends a data item through the stream.
	// Returns an error if the context is canceled or the emitter is closed.
	Emit(ctx context.Context, data any) error
	// Close signals that no more items will be sent.
	// The optional error is passed to the consumer as the final error state.
	Close(err error)
}

// ErrEmitterClosed is returned when attempting to emit after the emitter is closed.
var ErrEmitterClosed = errors.New("streaming emitter closed")

const defaultBufferSize = 1

// NewChannelEmitter creates a new channel-based emitter.
// The buffer parameter controls the channel buffer size; values <= 0 use the default.
// Returns the emitter and a receive-only channel for consuming items.
func NewChannelEmitter(buffer int) (Emitter, <-chan Item) {
	if buffer <= 0 {
		buffer = defaultBufferSize
	}
	ch := make(chan Item, buffer)
	return &channelEmitter{ch: ch}, ch
}

type channelEmitter struct {
	ch     chan Item
	once   sync.Once
	mu     sync.Mutex
	closed bool
}

// Emit sends a data item through the channel.
func (e *channelEmitter) Emit(ctx context.Context, data any) error {
	e.mu.Lock()
	closed := e.closed
	e.mu.Unlock()
	if closed {
		return ErrEmitterClosed
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case e.ch <- Item{Data: data}:
		return nil
	}
}

// Close signals completion and closes the channel.
func (e *channelEmitter) Close(err error) {
	e.once.Do(func() {
		e.mu.Lock()
		e.closed = true
		e.mu.Unlock()

		final := Item{Done: true, Err: err}
		select {
		case e.ch <- final:
		default:
		}
		close(e.ch)
	})
}

type emitterContextKey struct{}

// WithEmitter attaches an emitter to the context.
// If the emitter is nil, the context is returned unchanged.
func WithEmitter(ctx context.Context, emitter Emitter) context.Context {
	if emitter == nil {
		return ctx
	}
	return context.WithValue(ctx, emitterContextKey{}, emitter)
}

// FromContext retrieves the emitter from the context, if present.
func FromContext(ctx context.Context) (Emitter, bool) {
	emitter, ok := ctx.Value(emitterContextKey{}).(Emitter)
	return emitter, ok
}

// Emit sends data through the context-bound emitter.
// Returns nil if no emitter is present in the context.
// Returns an error if emission fails (context canceled, emitter closed, etc.).
func Emit(ctx context.Context, data any) error {
	emitter, ok := FromContext(ctx)
	if !ok {
		return nil
	}
	return emitter.Emit(ctx, data)
}

// IsStreaming returns true if a streaming emitter is attached to the context.
func IsStreaming(ctx context.Context) bool {
	_, ok := FromContext(ctx)
	return ok
}

// EmitOrCollect either emits the item (if streaming) or appends it to the slice.
// Returns the updated slice and any error from emission.
// This helper reduces duplication in services that support both streaming and buffered output.
//
// Usage:
//
//	items, err := streaming.EmitOrCollect(ctx, item, items)
//	if err != nil {
//	    return partialResult, nil
//	}
func EmitOrCollect[T any](ctx context.Context, item T, slice []T) ([]T, error) {
	if IsStreaming(ctx) {
		if err := Emit(ctx, item); err != nil {
			return slice, err
		}
		return slice, nil
	}
	return append(slice, item), nil
}
