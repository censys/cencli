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

func (s *historyService) GetWebPropertyHistory(
	ctx context.Context,
	orgID mo.Option[identifiers.OrganizationID],
	webPropertyID assets.WebPropertyID,
	fromTime time.Time,
	toTime time.Time,
) (WebPropertyHistoryResult, cenclierrors.CencliError) {
	start := time.Now()
	// convert orgID and webPropertyID
	orgIDStr := utilconvert.OptionalString(orgID)
	webPropIDStr := webPropertyID.String()

	var allSnapshots []*WebPropertySnapshot
	var lastMeta *responsemeta.ResponseMeta
	var firstError cenclierrors.CencliError

	totalRequests := uint64(0)

	// Calculate total days by truncating to date boundaries
	// This ensures the count matches the actual loop iterations
	fromDate := time.Date(fromTime.Year(), fromTime.Month(), fromTime.Day(), 0, 0, 0, 0, time.UTC)
	toDate := time.Date(toTime.Year(), toTime.Month(), toTime.Day(), 0, 0, 0, 0, time.UTC)
	totalDays := int(toDate.Sub(fromDate).Hours()/24) + 1

	// Walk through each day in the time range
	current := fromTime
	for current.Before(toTime) || current.Equal(toTime) {
		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			contextErr := cenclierrors.ParseContextError(err)

			// Return partial results with context error
			if totalRequests > 0 {
				latency := time.Since(start)
				if lastMeta != nil {
					lastMeta.Latency = latency
					lastMeta.PageCount = totalRequests
				}
				return WebPropertyHistoryResult{
					Meta:         lastMeta,
					Snapshots:    allSnapshots,
					PartialError: cenclierrors.ToPartialError(contextErr),
				}, nil
			}
			return WebPropertyHistoryResult{}, contextErr
		}

		totalRequests++
		// Update progress with day-by-day info showing current date and progress
		currentDate := current.Format("2006-01-02")
		progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Fetching web property history for %s (day %d/%d: %s)...", webPropIDStr, totalRequests, totalDays, currentDate))

		// fetch web property at this specific time
		res, err := s.client.GetWebProperties(
			ctx,
			orgIDStr,
			[]string{webPropIDStr},
			mo.Some(current),
		)

		if err != nil {
			// If this is the first request, return the error immediately
			if totalRequests == 1 {
				return WebPropertyHistoryResult{}, err
			}
			// Otherwise, record the error and return partial results
			// Note: We don't break here because for web properties, errors often mean
			// the property didn't exist at that time, so we continue to the next day
			if firstError == nil {
				firstError = err
				// Report the first error so users are aware something went wrong
				progress.ReportError(ctx, progress.StageFetch, err)
			}
			allSnapshots = append(allSnapshots, &WebPropertySnapshot{
				Time:   current,
				Data:   nil,
				Exists: false,
			})
		} else {
			// store metadata from the last successful request
			lastMeta = responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts)

			// Check if we got any results
			exists := false
			var webProp *components.Webproperty
			if res.Data != nil && len(*res.Data) > 0 {
				webProp = &(*res.Data)[0]
				// Check if the web property has meaningful data beyond hostname/port
				exists = webPropertyHasMeaningfulData(webProp)
			}

			allSnapshots = append(allSnapshots, &WebPropertySnapshot{
				Time:   current,
				Data:   webProp,
				Exists: exists,
			})
		}

		// Move to next day
		current = current.AddDate(0, 0, 1)
	}

	// set correct values for latency and request count
	latency := time.Since(start)
	if lastMeta != nil {
		lastMeta.Latency = latency
		lastMeta.PageCount = totalRequests
	}

	return WebPropertyHistoryResult{
		Meta:         lastMeta,
		Snapshots:    allSnapshots,
		PartialError: cenclierrors.ToPartialError(firstError),
	}, nil
}

// webPropertyHasMeaningfulData returns true if the web property has any non-zero field
// other than Hostname and Port, indicating it actually existed at that time
func webPropertyHasMeaningfulData(webProp *components.Webproperty) bool {
	if webProp == nil {
		return false
	}

	// Check various fields that indicate the web property has actual data
	// Beyond just hostname and port
	if len(webProp.Endpoints) > 0 {
		return true
	}
	if webProp.Cert != nil {
		return true
	}
	if webProp.TLS != nil {
		return true
	}
	if webProp.Jarm != nil {
		return true
	}
	if len(webProp.Software) > 0 {
		return true
	}
	if len(webProp.Hardware) > 0 {
		return true
	}
	if len(webProp.OperatingSystems) > 0 {
		return true
	}
	if len(webProp.Vulns) > 0 {
		return true
	}
	if len(webProp.Exposures) > 0 {
		return true
	}
	if len(webProp.Misconfigs) > 0 {
		return true
	}
	if len(webProp.Threats) > 0 {
		return true
	}
	if len(webProp.Labels) > 0 {
		return true
	}

	return false
}
