package datetime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSleepContext(t *testing.T) {
	t.Run("waits the full duration", func(t *testing.T) {
		assert.NoError(t, SleepContext(context.Background(), time.Millisecond))
	})

	t.Run("returns early when cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		start := time.Now()
		assert.ErrorIs(t, SleepContext(ctx, time.Hour), context.Canceled)
		assert.Less(t, time.Since(start), time.Second)
	})
}
