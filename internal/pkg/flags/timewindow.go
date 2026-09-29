package flags

import (
	"time"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// ResolveTimeWindow determines the start and end times from the --start, --end,
// and --duration flags. With neither start nor end, the window ends now.
func ResolveTimeWindow(
	startOpt mo.Option[time.Time],
	endOpt mo.Option[time.Time],
	durationOpt mo.Option[time.Duration],
) (time.Time, time.Time, cenclierrors.CencliError) {
	var start, end time.Time
	var duration time.Duration

	hasDuration := durationOpt.IsPresent()
	if hasDuration {
		duration = durationOpt.MustGet()
	} else {
		duration = 7 * 24 * time.Hour // default to 7 days
	}

	hasStart := startOpt.IsPresent()
	hasEnd := endOpt.IsPresent()

	if hasStart {
		start = startOpt.MustGet()
	}
	if hasEnd {
		end = endOpt.MustGet()
	}

	switch {
	case hasStart && hasEnd:
		// both start and end are set, use them as is; duration is unused.
		if end.Before(start) {
			return time.Time{}, time.Time{}, NewInvalidTimeWindowError("end time must be after start time")
		}
		return start, end, nil
	case hasStart:
		// only start is set, calculate end
		if hasDuration && duration <= 0 {
			return time.Time{}, time.Time{}, NewInvalidTimeWindowError("duration must be greater than 0")
		}
		return start, start.Add(duration), nil
	case hasEnd:
		// only end is set, calculate start
		if hasDuration && duration <= 0 {
			return time.Time{}, time.Time{}, NewInvalidTimeWindowError("duration must be greater than 0")
		}
		return end.Add(-duration), end, nil
	default:
		// neither is set, use now as end and calculate start
		if hasDuration && duration <= 0 {
			return time.Time{}, time.Time{}, NewInvalidTimeWindowError("duration must be greater than 0")
		}
		end = time.Now().UTC()
		return end.Add(-duration), end, nil
	}
}
