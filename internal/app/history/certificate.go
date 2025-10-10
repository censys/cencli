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

func (s *historyService) GetCertificateHistory(
	ctx context.Context,
	orgID mo.Option[identifiers.OrganizationID],
	certificateID assets.CertificateID,
	fromTime time.Time,
	toTime time.Time,
) (CertificateHistoryResult, cenclierrors.CencliError) {
	start := time.Now()
	// convert orgID and certificateID
	orgIDStr := utilconvert.OptionalString(orgID)
	certIDStr := certificateID.String()

	var allRanges []*components.HostObservationRange
	var lastMeta *responsemeta.ResponseMeta
	var firstError cenclierrors.CencliError

	pages := uint64(0)
	pageToken := mo.None[string]()
	totalObservations := 0

	// Format date range for display
	dateRange := fmt.Sprintf("%s to %s", fromTime.Format("2006-01-02"), toTime.Format("2006-01-02"))

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
				return CertificateHistoryResult{
					Meta:         lastMeta,
					Ranges:       allRanges,
					PartialError: cenclierrors.ToPartialError(contextErr),
				}, nil
			}
			return CertificateHistoryResult{}, contextErr
		}

		pages++
		// Update progress with detailed pagination and observation count
		if pages == 1 {
			progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Fetching certificate observations for %s (%s)...", certIDStr, dateRange))
		} else {
			progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Fetching certificate observations for %s (%s, page %d, %d observations so far)...", certIDStr, dateRange, pages, totalObservations))
		}

		// fetch observations page
		res, err := s.client.GetHostObservationsWithCertificate(
			ctx,
			orgIDStr,
			certIDStr,
			mo.Some(fromTime),
			mo.Some(toTime),
			mo.None[int](),    // port
			mo.None[string](), // protocol
			mo.None[int64](),  // pageSize (use default)
			pageToken,
		)
		if err != nil {
			// If this is the first page, return the error immediately
			if pages == 1 {
				return CertificateHistoryResult{}, err
			}
			// Otherwise, record the error, report it, and return partial results
			firstError = err
			progress.ReportError(ctx, progress.StageFetch, err)
			break
		}

		// store metadata from the last successful request
		lastMeta = responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts)

		ranges := res.Data.GetRanges()

		// append ranges to result
		for i := range ranges {
			allRanges = append(allRanges, &ranges[i])
		}
		totalObservations = len(allRanges)

		// check if there's a next page
		nextToken := res.Data.GetNextPageToken()
		if nextToken == nil {
			// no more pages
			break
		}

		pageToken = mo.Some(*nextToken)
	}

	// set correct values for latency and page count
	latency := time.Since(start)
	if lastMeta != nil {
		lastMeta.Latency = latency
		lastMeta.PageCount = pages
	}

	return CertificateHistoryResult{
		Meta:         lastMeta,
		Ranges:       allRanges,
		PartialError: cenclierrors.ToPartialError(firstError),
	}, nil
}
