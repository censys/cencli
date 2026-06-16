package enrich

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/samber/mo"
	"golang.org/x/sync/errgroup"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/app/streaming"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

const (
	// maxConcurrentEnrichments bounds the number of in-flight single-IP enrichment requests
	maxConcurrentEnrichments = 10
)

//go:generate mockgen -destination=../../../gen/app/enrich/mocks/enrichservice_mock.go -package=mocks -mock_names Service=MockEnrichService . Service

// Service provides host enrichment capabilities. The endpoint is single-IP, so
// the service fans out concurrent requests.
type Service interface {
	EnrichHosts(ctx context.Context, orgID mo.Option[identifiers.OrganizationID], hostIDs []assets.HostID) (Result, cenclierrors.CencliError)
}

type enrichService struct {
	client client.Client
}

func New(client client.Client) Service {
	return &enrichService{client: client}
}

// enrichOutcome carries the result of a single host's enrichment from a worker
// goroutine to the single collector.
type enrichOutcome struct {
	index int
	host  assets.HostID
	data  *assets.EnrichedHost
	meta  *responsemeta.ResponseMeta
	err   cenclierrors.CencliError
}

func (s *enrichService) EnrichHosts(
	ctx context.Context,
	orgID mo.Option[identifiers.OrganizationID],
	hostIDs []assets.HostID,
) (Result, cenclierrors.CencliError) {
	start := time.Now()
	orgIDStr := utilconvert.OptionalString(orgID)
	streamingMode := streaming.IsStreaming(ctx)
	total := len(hostIDs)

	if total == 0 {
		return Result{}, nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, gctx := errgroup.WithContext(runCtx)
	g.SetLimit(maxConcurrentEnrichments)

	outCh := make(chan enrichOutcome, maxConcurrentEnrichments)

	var dailyLimitOnce sync.Once
	var dailyLimitErr cenclierrors.CencliError

	go func() {
		for i, host := range hostIDs {
			g.Go(func() error {
				if err := gctx.Err(); err != nil {
					return err
				}

				res, cerr := s.client.EnrichHost(gctx, orgIDStr, host.String())
				if cerr != nil {
					if isDailyLimit(cerr) {
						dailyLimitOnce.Do(func() { dailyLimitErr = newDailyLimitError(cerr) })
						return cerr
					}
					if gctx.Err() == nil {
						select {
						case outCh <- enrichOutcome{index: i, host: host, err: cerr}:
						case <-ctx.Done():
						}
					}
					return nil
				}

				enriched := assets.NewEnrichedHost(*res.Data)
				meta := responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts)
				select {
				case outCh <- enrichOutcome{index: i, host: host, data: &enriched, meta: meta}:
				case <-ctx.Done():
				}
				return nil
			})
		}
		_ = g.Wait()
		close(outCh)
	}()

	ordered := make([]*assets.EnrichedHost, total)
	failuresByIdx := make([]*HostFailure, total)
	var repMeta *responsemeta.ResponseMeta
	var successCount, failureCount int
	var emitErr error

	for o := range outCh {
		if o.err != nil {
			failuresByIdx[o.index] = &HostFailure{HostID: o.host, Err: o.err}
			failureCount++
			continue
		}
		successCount++
		if repMeta == nil {
			repMeta = o.meta
		}
		progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Enriched %d/%d host(s)...", successCount, total))

		if streamingMode {
			if _, emitErr = streaming.EmitOrCollect(ctx, o.data, nil); emitErr != nil {
				cancel()
				break
			}
		} else {
			ordered[o.index] = o.data
		}
	}
	// Drain any buffered outcomes after an early break so the workers (and the
	// dispatcher's close) can wind down cleanly.
	for range outCh {
	}

	if repMeta != nil {
		repMeta.Latency = time.Since(start)
		repMeta.PageCount = uint64(successCount)
	}

	hosts := compactHosts(ordered)
	failures := compactFailures(failuresByIdx)

	// Nothing succeeded: return a hard error
	if successCount == 0 {
		switch {
		case dailyLimitErr != nil:
			return Result{}, dailyLimitErr
		case emitErr != nil:
			return Result{}, cenclierrors.NewCencliError(emitErr)
		case len(failures) > 0:
			return Result{}, failures[0].Err
		case ctx.Err() != nil:
			return Result{}, cenclierrors.ParseContextError(ctx.Err())
		default:
			return Result{Meta: repMeta}, nil
		}
	}

	// Partial success: some IPs succeeded. Surface a summary through PartialError
	// so the existing stderr-reporting path fires; the detailed per-IP list is
	// carried in Failures.
	var partial cenclierrors.CencliError
	switch {
	case dailyLimitErr != nil:
		partial = dailyLimitErr
	case emitErr != nil:
		partial = cenclierrors.NewCencliError(emitErr)
	case failureCount > 0:
		partial = newPartialFailureError(failureCount, total)
	}

	return Result{
		Meta:         repMeta,
		Hosts:        hosts,
		Failures:     failures,
		PartialError: cenclierrors.ToPartialError(partial),
	}, nil
}

// isDailyLimit reports whether a client error is a 429. The client retries
// 429/5xx internally, so a 429 returned to us means the daily limit was hit.
func isDailyLimit(err client.ClientError) bool {
	if sc := err.StatusCode(); sc.IsPresent() {
		return sc.MustGet() == http.StatusTooManyRequests
	}
	return false
}

func compactHosts(in []*assets.EnrichedHost) []*assets.EnrichedHost {
	out := make([]*assets.EnrichedHost, 0, len(in))
	for _, h := range in {
		if h != nil {
			out = append(out, h)
		}
	}
	return out
}

func compactFailures(in []*HostFailure) []HostFailure {
	out := make([]HostFailure, 0, len(in))
	for _, f := range in {
		if f != nil {
			out = append(out, *f)
		}
	}
	return out
}
