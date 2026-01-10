package streaming

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChannelEmitter(t *testing.T) {
	t.Run("creates emitter with default buffer", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(0)
		require.NotNil(t, emitter)
		require.NotNil(t, ch)
		emitter.Close(nil)
	})

	t.Run("creates emitter with custom buffer", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(5)
		require.NotNil(t, emitter)
		require.NotNil(t, ch)
		emitter.Close(nil)
	})
}

func TestEmitter_Emit(t *testing.T) {
	t.Run("emits data successfully", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(1)
		ctx := context.Background()

		go func() {
			err := emitter.Emit(ctx, "test-data")
			assert.NoError(t, err)
			emitter.Close(nil)
		}()

		item := <-ch
		if item.Data != "test-data" {
			t.Errorf("expected data 'test-data', got %v", item.Data)
		}
		if item.Done {
			t.Error("expected Done to be false")
		}
		if item.Err != nil {
			t.Errorf("expected nil error, got %v", item.Err)
		}

		// Drain final item
		<-ch
	})

	t.Run("returns error when emitter is closed", func(t *testing.T) {
		emitter, _ := NewChannelEmitter(1)
		ctx := context.Background()

		emitter.Close(nil)

		err := emitter.Emit(ctx, "test-data")
		require.ErrorIs(t, err, ErrEmitterClosed)
	})

	t.Run("returns error when context is canceled and channel is blocked", func(t *testing.T) {
		// Use a buffer of 1 and fill it first so the next emit blocks
		emitter, ch := NewChannelEmitter(1)
		ctx, cancel := context.WithCancel(context.Background())

		// Fill the buffer
		_ = emitter.Emit(ctx, "first")

		// Now cancel the context
		cancel()

		// This emit should fail because buffer is full and context is canceled
		err := emitter.Emit(ctx, "second")
		require.ErrorIs(t, err, context.Canceled)

		// Drain and close
		<-ch
		emitter.Close(nil)
	})
}

func TestEmitter_Close(t *testing.T) {
	t.Run("sends done signal", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(1)

		emitter.Close(nil)

		item := <-ch
		require.True(t, item.Done)
		require.NoError(t, item.Err)
	})

	t.Run("sends error with done signal", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(1)
		testErr := errors.New("test error")

		emitter.Close(testErr)

		item := <-ch
		require.True(t, item.Done)
		require.ErrorIs(t, item.Err, testErr)
	})

	t.Run("close is idempotent", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(1)

		// Close multiple times should not panic
		emitter.Close(nil)
		emitter.Close(nil)
		emitter.Close(nil)

		// Should only receive one done signal
		item := <-ch
		require.True(t, item.Done)
		_, ok := <-ch
		require.False(t, ok)
	})

	t.Run("closes channel after done", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(1)

		emitter.Close(nil)

		<-ch // Drain done signal

		// Verify channel is closed
		_, ok := <-ch
		require.False(t, ok)
	})
}

func TestContext_WithEmitter(t *testing.T) {
	t.Run("attaches emitter to context", func(t *testing.T) {
		emitter, _ := NewChannelEmitter(1)
		ctx := context.Background()

		ctx = WithEmitter(ctx, emitter)

		retrieved, ok := FromContext(ctx)
		require.True(t, ok)
		require.Equal(t, emitter, retrieved)
		emitter.Close(nil)
	})

	t.Run("returns original context for nil emitter", func(t *testing.T) {
		ctx := context.Background()

		newCtx := WithEmitter(ctx, nil)

		require.Equal(t, ctx, newCtx)
	})
}

func TestContext_FromContext(t *testing.T) {
	t.Run("returns false when no emitter", func(t *testing.T) {
		ctx := context.Background()

		_, ok := FromContext(ctx)
		require.False(t, ok)
	})
}

func TestEmit(t *testing.T) {
	t.Run("emits data through context emitter", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(1)
		ctx := WithEmitter(context.Background(), emitter)

		go func() {
			err := Emit(ctx, "context-data")
			assert.NoError(t, err)
			emitter.Close(nil)
		}()

		item := <-ch
		require.Equal(t, "context-data", item.Data)

		<-ch // Drain done
	})

	t.Run("returns nil when no emitter in context", func(t *testing.T) {
		ctx := context.Background()

		err := Emit(ctx, "data")
		require.NoError(t, err)
	})
}

func TestIsStreaming(t *testing.T) {
	t.Run("returns true when emitter present", func(t *testing.T) {
		emitter, _ := NewChannelEmitter(1)
		ctx := WithEmitter(context.Background(), emitter)

		require.True(t, IsStreaming(ctx))
		emitter.Close(nil)
	})

	t.Run("returns false when no emitter", func(t *testing.T) {
		ctx := context.Background()

		require.False(t, IsStreaming(ctx))
	})
}

func TestEmitter_ConcurrentEmit(t *testing.T) {
	t.Run("handles concurrent emits safely", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(10)
		ctx := context.Background()

		const numGoroutines = 5
		const itemsPerGoroutine = 10

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := range numGoroutines {
			go func(id int) {
				defer wg.Done()
				for j := range itemsPerGoroutine {
					_ = emitter.Emit(ctx, id*100+j)
				}
			}(i)
		}

		// Consume items in a separate goroutine
		received := make(chan int, numGoroutines*itemsPerGoroutine+1)
		go func() {
			for item := range ch {
				if item.Done {
					break
				}
				if v, ok := item.Data.(int); ok {
					received <- v
				}
			}
			close(received)
		}()

		wg.Wait()
		emitter.Close(nil)

		count := 0
		for range received {
			count++
		}

		require.Equal(t, numGoroutines*itemsPerGoroutine, count)
	})
}

func TestEmitter_NoDeadlock(t *testing.T) {
	t.Run("does not deadlock with slow consumer", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(1)
		ctx := context.Background()

		done := make(chan struct{})
		go func() {
			defer close(done)
			for item := range ch {
				if item.Done {
					break
				}
				// Simulate slow consumer
				time.Sleep(10 * time.Millisecond)
			}
		}()

		// Send multiple items faster than consumer can process
		for i := range 5 {
			err := emitter.Emit(ctx, i)
			assert.NoError(t, err)
		}
		emitter.Close(nil)

		// Wait with timeout to detect deadlock
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			require.Fail(t, "deadlock detected: consumer did not finish")
		}
	})
}

func TestEmitOrCollect(t *testing.T) {
	t.Run("appends to slice when not streaming", func(t *testing.T) {
		ctx := context.Background() // No emitter attached

		slice := []string{"a", "b"}
		result, err := EmitOrCollect(ctx, "c", slice)

		require.NoError(t, err)
		require.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("emits and returns unchanged slice when streaming", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(1)
		ctx := WithEmitter(context.Background(), emitter)

		go func() {
			// Consume emitted item
			item := <-ch
			assert.Equal(t, "streamed", item.Data)
			emitter.Close(nil)
		}()

		slice := []string{"existing"}
		result, err := EmitOrCollect(ctx, "streamed", slice)

		require.NoError(t, err)
		// Slice should not grow when streaming
		require.Equal(t, []string{"existing"}, result)

		// Drain close signal
		<-ch
	})

	t.Run("returns error on emit failure", func(t *testing.T) {
		emitter, ch := NewChannelEmitter(1)
		ctx := WithEmitter(context.Background(), emitter)

		// Fill buffer so next emit blocks
		_ = emitter.Emit(ctx, "fill")

		// Cancel to cause emit failure
		ctx, cancel := context.WithCancel(ctx)
		cancel()

		slice := []int{1, 2}
		result, err := EmitOrCollect(ctx, 3, slice)

		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, []int{1, 2}, result) // Slice unchanged on error

		<-ch
		emitter.Close(nil)
	})

	t.Run("works with nil slice", func(t *testing.T) {
		ctx := context.Background()

		var slice []int
		result, err := EmitOrCollect(ctx, 42, slice)

		require.NoError(t, err)
		require.Equal(t, []int{42}, result)
	})

	t.Run("works with pointer types", func(t *testing.T) {
		ctx := context.Background()

		type Data struct{ Value int }
		slice := []*Data{{Value: 1}}
		result, err := EmitOrCollect(ctx, &Data{Value: 2}, slice)

		require.NoError(t, err)
		require.Len(t, result, 2)
		require.Equal(t, 1, result[0].Value)
		require.Equal(t, 2, result[1].Value)
	})
}
