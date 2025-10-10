package search

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/mo"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

//go:generate mockgen -destination=../../../gen/app/search/mocks/searchservice_mock.go -package=mocks -mock_names Service=MockSearchService . Service

// Service provides asset search capabilities.
type Service interface {
	Search(ctx context.Context, params Params) (Result, cenclierrors.CencliError)
}

type searchService struct {
	client client.Client
}

func New(client client.Client) Service {
	return &searchService{client: client}
}

func (s *searchService) Search(
	ctx context.Context,
	params Params,
) (Result, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	// handle pagination invariants
	if params.PageSize.IsPresent() && params.PageSize.MustGet() == 0 {
		return Result{}, NewInvalidPaginationParamsError("page size must be greater than 0")
	}
	if params.MaxPages.IsPresent() && params.MaxPages.MustGet() == 0 {
		return Result{}, NewInvalidPaginationParamsError("max pages must be greater than 0")
	}

	var searchFn func(mo.Option[string]) (client.Result[components.SearchQueryResponse], cenclierrors.CencliError)
	if params.CollectionID.IsPresent() {
		searchFn = func(pageToken mo.Option[string]) (client.Result[components.SearchQueryResponse], cenclierrors.CencliError) {
			return s.client.SearchCollection(
				ctx,
				params.CollectionID.MustGet().String(),
				orgIDStr,
				params.Query,
				params.Fields,
				toInt64(params.PageSize),
				pageToken,
			)
		}
	} else {
		searchFn = func(pageToken mo.Option[string]) (client.Result[components.SearchQueryResponse], cenclierrors.CencliError) {
			return s.client.Search(
				ctx,
				orgIDStr,
				params.Query,
				params.Fields,
				toInt64(params.PageSize),
				pageToken,
			)
		}
	}

	return s.searchWithPagination(ctx, searchFn, params.MaxPages)
}

func (s *searchService) searchWithPagination(
	ctx context.Context,
	searchFn func(mo.Option[string]) (client.Result[components.SearchQueryResponse], cenclierrors.CencliError),
	maxPages mo.Option[uint64],
) (Result, cenclierrors.CencliError) {
	var allHits []assets.Asset
	var totalHits int64
	var lastMeta *responsemeta.ResponseMeta
	var pagesProcessed uint64
	var firstError cenclierrors.CencliError
	pageToken := mo.None[string]()

	start := time.Now()

	for {
		if maxPages.IsPresent() && pagesProcessed >= maxPages.MustGet() {
			break
		}

		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			contextErr := cenclierrors.ParseContextError(err)

			// Return partial results with context error
			if pagesProcessed > 0 {
				if lastMeta != nil {
					lastMeta.Latency = time.Since(start)
					lastMeta.PageCount = pagesProcessed
				}
				return Result{
					Meta:         lastMeta,
					Hits:         allHits,
					TotalHits:    totalHits,
					PartialError: cenclierrors.ToPartialError(contextErr),
				}, nil
			}
			return Result{}, contextErr
		}

		// Report progress for pagination
		s.reportSearchProgress(ctx, pagesProcessed, len(allHits), maxPages)

		result, err := searchFn(pageToken)
		if err != nil {
			// If this is the first page, return the error immediately
			if pagesProcessed == 0 {
				return Result{}, err
			}
			// Otherwise, record the error, report it, and return partial results
			firstError = err
			progress.ReportError(ctx, progress.StageFetch, err)
			break
		}

		if result.Metadata.Request != nil || result.Metadata.Response != nil {
			lastMeta = responsemeta.NewResponseMeta(result.Metadata.Request, result.Metadata.Response, 0, uint64(result.Metadata.Attempts))
		}

		if result.Data == nil {
			pagesProcessed++
			break
		}

		pageHits := parseHits(result.Data.Hits)
		allHits = append(allHits, pageHits...)
		totalHits = int64(result.Data.TotalHits)

		pagesProcessed++

		nextPageToken := result.Data.GetNextPageToken()
		if nextPageToken == "" || len(pageHits) == 0 {
			break
		}

		if maxPages.IsPresent() && pagesProcessed >= maxPages.MustGet() {
			break
		}

		pageToken = mo.Some(nextPageToken)
	}

	if lastMeta != nil {
		lastMeta.Latency = time.Since(start)
		lastMeta.PageCount = pagesProcessed
	}

	return Result{
		Meta:         lastMeta,
		Hits:         allHits,
		TotalHits:    totalHits,
		PartialError: cenclierrors.ToPartialError(firstError),
	}, nil
}

func toInt64(pageSize mo.Option[uint64]) mo.Option[int64] {
	res := mo.None[int64]()
	if pageSize.IsPresent() {
		res = mo.Some(int64(pageSize.MustGet()))
	}
	return res
}

func (s *searchService) reportSearchProgress(ctx context.Context, page uint64, hitsCollected int, maxPages mo.Option[uint64]) {
	if page == 0 {
		// First page, just show initial message
		return
	}

	var msg string
	if maxPages.IsPresent() {
		msg = fmt.Sprintf("Fetching search results (page %d/%d, %d hits collected)...", page+1, maxPages.MustGet(), hitsCollected)
	} else {
		msg = fmt.Sprintf("Fetching search results (page %d, %d hits collected)...", page+1, hitsCollected)
	}

	progress.ReportMessage(ctx, progress.StageFetch, msg)
}
