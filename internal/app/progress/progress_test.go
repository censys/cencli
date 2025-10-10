package progress

import (
	"context"
	"testing"
	"time"
)

// TestPublisher_NoDeadlockWithMultipleEvents verifies that rapid event publishing
// doesn't cause deadlocks even with a small buffer size.
func TestPublisher_NoDeadlockWithMultipleEvents(t *testing.T) {
	pub, events := NewChannelPublisher(1)

	// Start consumer
	done := make(chan struct{})
	go func() {
		defer close(done)
		for event := range events {
			if event.Done {
				break
			}
			// Simulate slow consumer
			time.Sleep(10 * time.Millisecond)
		}
	}()

	ctx := context.Background()

	// Send multiple events rapidly (more than buffer size)
	completeChan := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func(idx int) {
			err := pub.Publish(ctx, Event{
				Stage:   StageFetch,
				Message: "test",
			})
			completeChan <- err
		}(i)
	}

	// Wait for all publishes to complete (with timeout to detect deadlock)
	timeout := time.After(5 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case err := <-completeChan:
			if err != nil {
				t.Fatalf("publish %d failed: %v", i, err)
			}
		case <-timeout:
			t.Fatal("deadlock detected: publish operations did not complete within timeout")
		}
	}

	pub.Close(nil)
	<-done
}

// TestPublisher_NoDeadlockWithoutContextDeadline verifies behavior without context timeout.
func TestPublisher_NoDeadlockWithoutContextDeadline(t *testing.T) {
	pub, events := NewChannelPublisher(1)

	// Start consumer goroutine
	done := make(chan struct{})
	eventCount := 0
	go func() {
		defer close(done)
		for event := range events {
			if event.Done {
				break
			}
			eventCount++
			time.Sleep(50 * time.Millisecond) // Slow consumer
		}
	}()

	// Context without deadline
	ctx := context.Background()

	// Send events faster than consumer can handle
	sendDone := make(chan struct{})
	go func() {
		defer close(sendDone)
		for i := 0; i < 5; i++ {
			if err := pub.Publish(ctx, Event{Stage: StageFetch}); err != nil {
				t.Errorf("publish failed: %v", err)
				return
			}
		}
	}()

	// Ensure sends complete within reasonable time
	select {
	case <-sendDone:
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock detected: publishes blocked indefinitely without context deadline")
	}

	pub.Close(nil)
	<-done

	if eventCount != 5 {
		t.Errorf("expected 5 events consumed, got %d", eventCount)
	}
}
