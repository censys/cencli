package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/app/streaming"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/censys-sdk-go/models/components"
)

func (s *historyService) GetWebPropertyHistory(
	ctx context.Context,
	orgID mo.Option[identifiers.OrganizationID],
	webPropertyID assets.WebPropertyID,
	fromTime time.Time,
	toTime time.Time,
) (WebPropertyHistoryResult, cenclierrors.CencliError) {
	start := time.Now()
	orgIDStr := utilconvert.OptionalString(orgID)
	webPropIDStr := webPropertyID.String()

	var allEvents []*components.WebTimelineEvent
	var lastMeta *responsemeta.ResponseMeta
	var firstError cenclierrors.CencliError

	currentToTime := toTime

	pages := uint64(0)
	// The backend can return the same event twice, within a page or across
	// pages, so events already seen on this page or the previous one are skipped.
	var prevPageKeys map[string]struct{}

	dateRange := fmt.Sprintf("%s to %s", fromTime.Format("2006-01-02T15:04:05Z"), toTime.Format("2006-01-02T15:04:05Z"))

	for {
		if err := ctx.Err(); err != nil {
			contextErr := cenclierrors.ParseContextError(err)

			// Return partial results with context error
			if pages > 0 || streaming.IsStreaming(ctx) {
				if lastMeta != nil {
					lastMeta.Latency = time.Since(start)
					lastMeta.PageCount = pages
				}
				return WebPropertyHistoryResult{
					Meta:         lastMeta,
					Events:       allEvents,
					PartialError: cenclierrors.ToPartialError(contextErr),
				}, nil
			}
			return WebPropertyHistoryResult{}, contextErr
		}

		pages++
		if pages == 1 {
			progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Fetching web property timeline for %s (%s)...", webPropIDStr, dateRange))
		} else {
			currentRangeEnd := currentToTime.Format("2006-01-02T15:04:05Z")
			progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Fetching web property timeline for %s (page %d, scanning back to %s)...", webPropIDStr, pages, currentRangeEnd))
		}

		res, err := s.client.WebPropertyTimeline(ctx, orgIDStr, webPropIDStr, fromTime, currentToTime)
		if err != nil {
			if pages == 1 {
				return WebPropertyHistoryResult{}, err
			}
			firstError = err
			progress.ReportError(ctx, progress.StageFetch, err)
			break
		}

		lastMeta = responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts)

		events := res.Data.GetEvents()
		pageKeys := make(map[string]struct{}, len(events))
		for i := range events {
			event := &events[i].Resource
			key := webTimelineEventKey(event)
			if _, seen := pageKeys[key]; seen {
				continue
			}
			pageKeys[key] = struct{}{}
			if _, seen := prevPageKeys[key]; seen {
				continue
			}
			var emitErr error
			allEvents, emitErr = streaming.EmitOrCollect(ctx, event, allEvents)
			if emitErr != nil {
				if lastMeta != nil {
					lastMeta.Latency = time.Since(start)
					lastMeta.PageCount = pages
				}
				return WebPropertyHistoryResult{
					Meta:         lastMeta,
					Events:       nil,
					PartialError: cenclierrors.ToPartialError(cenclierrors.NewCencliError(emitErr)),
				}, nil
			}
		}
		if len(events) > 0 {
			prevPageKeys = pageKeys
		}

		// Neither a short nor an empty page ends pagination; only the cursor does.
		scannedTo := res.Data.GetScannedTo()
		if scannedTo.IsZero() || !scannedTo.After(fromTime) {
			// zero time is the API's "exhausted" sentinel
			break
		}
		if !scannedTo.Before(currentToTime) {
			// the cursor did not move back; the next request would repeat this one
			firstError = newTimelineStalledError(currentToTime)
			break
		}

		currentToTime = scannedTo
	}

	if lastMeta != nil {
		lastMeta.Latency = time.Since(start)
		lastMeta.PageCount = pages
	}
	return WebPropertyHistoryResult{
		Meta:         lastMeta,
		Events:       allEvents,
		PartialError: partialFromLoop(firstError),
	}, nil
}

// partialFromLoop wraps a mid-pagination failure as partial data. A stalled
// cursor is reported as is: it can happen before any event was returned.
func partialFromLoop(err cenclierrors.CencliError) cenclierrors.PartialError {
	if _, stalled := err.(*timelineStalledError); stalled {
		return err
	}
	return cenclierrors.ToPartialError(err)
}

// webTimelineEventKey identifies an event for seam deduplication. An event that
// cannot be marshalled gets a unique key, so it is always kept.
func webTimelineEventKey(event *components.WebTimelineEvent) string {
	eventTime := ""
	if t := event.GetEventTime(); t != nil {
		eventTime = *t
	}
	b, err := json.Marshal(event)
	if err != nil {
		return fmt.Sprintf("%s|unhashable|%p", eventTime, event)
	}
	sum := sha256.Sum256(b)
	return eventTime + "|" + hex.EncodeToString(sum[:])
}
