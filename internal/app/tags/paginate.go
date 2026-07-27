package tags

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/app/streaming"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

// pageData is what the paginator needs from one API page, pulled out of an
// endpoint-specific list envelope by the caller's extract function.
type pageData[Item any] struct {
	Items         []Item
	TotalSize     int64
	NextPageToken string
}

// paginated is the outcome of a paginated fetch. Items is empty in streaming
// mode, where each item is emitted as it arrives instead of being collected.
type paginated[Item any] struct {
	Meta         *responsemeta.ResponseMeta
	Items        []Item
	TotalSize    int64
	PartialError cenclierrors.CencliError
}

// paginate walks a list endpoint page by page until it runs out of pages or hits
// maxPages (absent = all pages), emitting items in streaming mode and collecting
// them otherwise. label names the items in progress messages ("tags").
//
// A failure on the first page — or a cancellation before any page landed — is a
// hard error; later failures return the pages gathered so far with the error as
// PartialError.
func paginate[Page, Item any](
	ctx context.Context,
	maxPages mo.Option[uint64],
	label string,
	fetch func(pageToken mo.Option[string]) (client.Result[Page], client.ClientError),
	extract func(*Page) pageData[Item],
) (paginated[Item], cenclierrors.CencliError) {
	var items []Item
	// len(items) stays 0 while streaming, so progress needs its own counter.
	var collected int
	var totalSize int64
	var lastMeta *responsemeta.ResponseMeta
	var pagesProcessed uint64
	var firstError cenclierrors.CencliError
	pageToken := mo.None[string]()

	start := time.Now()

	// finalize stamps elapsed time and page count onto the last page's metadata,
	// which is what the command layer renders.
	finalize := func() {
		if lastMeta != nil {
			lastMeta.Latency = time.Since(start)
			lastMeta.PageCount = pagesProcessed
		}
	}

	for {
		if maxPages.IsPresent() && pagesProcessed >= maxPages.MustGet() {
			break
		}

		if err := ctx.Err(); err != nil {
			contextErr := cenclierrors.ParseContextError(err)
			if pagesProcessed > 0 {
				finalize()
				return paginated[Item]{
					Meta:         lastMeta,
					Items:        items,
					TotalSize:    totalSize,
					PartialError: cenclierrors.ToPartialError(contextErr),
				}, nil
			}
			return paginated[Item]{}, contextErr
		}

		reportPageProgress(ctx, label, pagesProcessed, collected, maxPages)

		result, err := fetch(pageToken)
		if err != nil {
			if pagesProcessed == 0 {
				return paginated[Item]{}, err
			}
			firstError = err
			progress.ReportError(ctx, progress.StageFetch, err)
			break
		}

		if result.Metadata.Request != nil || result.Metadata.Response != nil {
			lastMeta = responsemeta.NewResponseMeta(result.Metadata.Request, result.Metadata.Response, 0, result.Metadata.Attempts)
		}

		if result.Data == nil {
			pagesProcessed++
			break
		}

		page := extract(result.Data)
		for _, item := range page.Items {
			emitted, emitErr := streaming.EmitOrCollect(ctx, item, items)
			if emitErr != nil {
				// The consumer is gone; keep what was emitted and report why.
				finalize()
				return paginated[Item]{
					Meta:         lastMeta,
					Items:        items,
					TotalSize:    page.TotalSize,
					PartialError: cenclierrors.ToPartialError(cenclierrors.NewCencliError(emitErr)),
				}, nil
			}
			items = emitted
			collected++
		}
		totalSize = page.TotalSize
		pagesProcessed++

		if page.NextPageToken == "" || len(page.Items) == 0 {
			break
		}

		// A server echoing back the token it was given would otherwise loop
		// forever under --max-pages=-1, spending a request per turn.
		if pageToken.IsPresent() && page.NextPageToken == pageToken.MustGet() {
			break
		}

		if maxPages.IsPresent() && pagesProcessed >= maxPages.MustGet() {
			break
		}

		pageToken = mo.Some(page.NextPageToken)
	}

	finalize()

	return paginated[Item]{
		Meta:         lastMeta,
		Items:        items,
		TotalSize:    totalSize,
		PartialError: cenclierrors.ToPartialError(firstError),
	}, nil
}

// reportPageProgress reports progress from the second page onwards; the initial
// message comes from the command layer.
func reportPageProgress(ctx context.Context, label string, page uint64, collected int, maxPages mo.Option[uint64]) {
	if page == 0 {
		return
	}

	var msg string
	if maxPages.IsPresent() {
		msg = fmt.Sprintf("Fetching %s (page %d/%d, %d collected)...", label, page+1, maxPages.MustGet(), collected)
	} else {
		msg = fmt.Sprintf("Fetching %s (page %d, %d collected)...", label, page+1, collected)
	}

	progress.ReportMessage(ctx, progress.StageFetch, msg)
}
