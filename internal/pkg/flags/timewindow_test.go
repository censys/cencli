package flags

import (
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTimeWindow(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	tests := []struct {
		name        string
		startOpt    mo.Option[time.Time]
		endOpt      mo.Option[time.Time]
		duration    mo.Option[time.Duration]
		wantStart   time.Time
		wantEnd     time.Time
		wantErr     bool
		errContains string
	}{
		{name: "start and end", startOpt: mo.Some(start), endOpt: mo.Some(end), duration: mo.Some(day), wantStart: start, wantEnd: end},
		{name: "start only", startOpt: mo.Some(start), endOpt: mo.None[time.Time](), duration: mo.Some(2 * day), wantStart: start, wantEnd: start.Add(2 * day)},
		{name: "end only", startOpt: mo.None[time.Time](), endOpt: mo.Some(end), duration: mo.Some(2 * day), wantStart: end.Add(-2 * day), wantEnd: end},
		{
			name:        "end before start",
			startOpt:    mo.Some(end),
			endOpt:      mo.Some(start),
			duration:    mo.None[time.Duration](),
			wantErr:     true,
			errContains: "invalid time window: end time must be after start time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd, err := ResolveTimeWindow(tt.startOpt, tt.endOpt, tt.duration)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Equal(t, "Invalid Time Window", err.Title())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStart, gotStart)
			assert.Equal(t, tt.wantEnd, gotEnd)
		})
	}
}

func TestResolveTimeWindow_DefaultsToLastSevenDays(t *testing.T) {
	gotStart, gotEnd, err := ResolveTimeWindow(mo.None[time.Time](), mo.None[time.Time](), mo.None[time.Duration]())
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().UTC(), gotEnd, time.Minute)
	assert.Equal(t, 7*24*time.Hour, gotEnd.Sub(gotStart))
}
