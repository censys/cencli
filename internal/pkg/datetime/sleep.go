package datetime

import (
	"context"
	"time"
)

// SleepContext waits for d, returning early with ctx.Err() if the context is
// cancelled first.
func SleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
