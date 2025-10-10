package history

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/censys-sdk-go/models/components"
)

const (
	// maxEventsPerPage is the maximum number of timeline events returned by the API per request.
	// The host timeline API is not paginated in the traditional sense (no page tokens), so we must
	// manually paginate by adjusting the time window. When exactly 100 events are returned, there
	// may be more events available, requiring another request with an adjusted time boundary.
	maxEventsPerPage = 100
)

func (s *historyService) GetHostHistory(
	ctx context.Context,
	orgID mo.Option[identifiers.OrganizationID],
	host assets.HostID,
	fromTime time.Time,
	toTime time.Time,
) (HostHistoryResult, cenclierrors.CencliError) {
	start := time.Now()
	// convert orgID and hostID
	orgIDStr := utilconvert.OptionalString(orgID)
	hostIDStr := host.String()

	var allEvents []*components.HostTimelineEvent
	var lastMeta *responsemeta.ResponseMeta
	var firstError cenclierrors.CencliError

	// Start from toTime and work backwards to fromTime
	currentToTime := toTime

	pages := uint64(0)

	// Format date range for display
	dateRange := fmt.Sprintf("%s to %s", fromTime.Format("2006-01-02T15:04:05Z"), toTime.Format("2006-01-02T15:04:05Z"))

	for {
		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			contextErr := cenclierrors.ParseContextError(err)

			// Return partial results with context error
			if pages > 0 {
				latency := time.Since(start)
				if lastMeta != nil {
					lastMeta.Latency = latency
					lastMeta.PageCount = pages
				}
				return HostHistoryResult{
					Meta:         lastMeta,
					Events:       allEvents,
					PartialError: cenclierrors.ToPartialError(contextErr),
				}, nil
			}
			return HostHistoryResult{}, contextErr
		}

		pages++
		// Update progress with detailed pagination and date range info
		if pages == 1 {
			progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Fetching host timeline for %s (%s)...", hostIDStr, dateRange))
		} else {
			// Show current scanning position
			currentRangeEnd := currentToTime.Format("2006-01-02T15:04:05Z")
			progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Fetching host timeline for %s (page %d, scanning back to %s)...", hostIDStr, pages, currentRangeEnd))
		}

		// fetch timeline page
		res, err := s.client.HostTimeline(ctx, orgIDStr, hostIDStr, fromTime, currentToTime)
		if err != nil {
			// If this is the first page, return the error immediately
			if pages == 1 {
				return HostHistoryResult{}, err
			}
			// Otherwise, record the error, report it, and return partial results
			firstError = err
			progress.ReportError(ctx, progress.StageFetch, err)
			break
		}

		// store metadata from the last successful request
		lastMeta = responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts)

		events := res.Data.GetEvents()
		if len(events) == 0 {
			// no more events available
			break
		}

		// append events to result
		for i := range events {
			allEvents = append(allEvents, &events[i].Resource)
		}

		// if we got fewer than maxEventsPerPage events, we've reached the end
		if len(events) < maxEventsPerPage {
			break
		}

		// check if we've reached the fromTime boundary
		scannedTo := res.Data.GetScannedTo()
		if scannedTo.IsZero() || !scannedTo.After(fromTime) {
			// reached the start boundary
			break
		}

		// move the time window backwards for the next iteration
		currentToTime = scannedTo
	}
	// set correct values for latency and page count
	latency := time.Since(start)
	if lastMeta != nil {
		lastMeta.Latency = latency
		lastMeta.PageCount = pages
	}
	return HostHistoryResult{
		Meta:         lastMeta,
		Events:       allEvents,
		PartialError: cenclierrors.ToPartialError(firstError),
	}, nil
}
