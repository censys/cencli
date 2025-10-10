package view

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

const (
	// maxHostsPerRequest is the maximum number of host IDs the API accepts in a single request.
	maxHostsPerRequest = 100
	// maxCertificatesPerRequest is the maximum number of certificate IDs the API accepts in a single request.
	maxCertificatesPerRequest = 1000
	// maxWebPropertiesPerRequest is the maximum number of web property IDs the API accepts in a single request.
	maxWebPropertiesPerRequest = 100
)

//go:generate mockgen -destination=../../../gen/app/view/mocks/viewservice_mock.go -package=mocks -mock_names Service=MockViewService . Service

// Service provides asset view capabilities.
// Currently, this is just a thin wrapper around the SDK, but leaves room for future enhancements.
type Service interface {
	GetHosts(ctx context.Context, orgID mo.Option[identifiers.OrganizationID], hostIDs []assets.HostID, atTime mo.Option[time.Time]) (HostsResult, cenclierrors.CencliError)
	GetCertificates(ctx context.Context, orgID mo.Option[identifiers.OrganizationID], certificateIDs []assets.CertificateID) (CertificatesResult, cenclierrors.CencliError)
	GetWebProperties(ctx context.Context, orgID mo.Option[identifiers.OrganizationID], webPropertyIDs []assets.WebPropertyID, atTime mo.Option[time.Time]) (WebPropertiesResult, cenclierrors.CencliError)
}

type viewService struct {
	client client.Client
}

func New(client client.Client) Service {
	return &viewService{client: client}
}

func (s *viewService) GetHosts(
	ctx context.Context,
	orgID mo.Option[identifiers.OrganizationID],
	hostIDs []assets.HostID,
	atTime mo.Option[time.Time],
) (HostsResult, cenclierrors.CencliError) {
	start := time.Now()
	orgIDStr := utilconvert.OptionalString(orgID)

	// Split IDs into batches based on API limits
	batches := splitSlice(hostIDs, maxHostsPerRequest)
	totalBatches := len(batches)

	var allHosts []*assets.Host
	var lastMeta *responsemeta.ResponseMeta
	var firstError cenclierrors.CencliError
	batchesProcessed := 0

	for batchNum, batch := range batches {
		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			contextErr := cenclierrors.ParseContextError(err)

			// Return partial results with context error
			if len(allHosts) > 0 {
				if lastMeta != nil {
					lastMeta.Latency = time.Since(start)
					lastMeta.PageCount = uint64(batchesProcessed)
				}
				return HostsResult{
					Meta:         lastMeta,
					Hosts:        allHosts,
					PartialError: cenclierrors.ToPartialError(contextErr),
				}, nil
			}
			return HostsResult{}, contextErr
		}

		// Report progress for batch fetches
		if totalBatches > 1 {
			message := fmt.Sprintf("Fetching hosts batch %d/%d (%d hosts)", batchNum+1, totalBatches, len(batch))
			if atTime.IsPresent() {
				message = fmt.Sprintf("%s at %s", message, atTime.MustGet().Format(time.RFC3339))
			}
			progress.ReportMessage(ctx, progress.StageFetch, message+"...")
		} else {
			message := fmt.Sprintf("Fetching %d host(s)", len(hostIDs))
			if atTime.IsPresent() {
				message = fmt.Sprintf("%s at %s", message, atTime.MustGet().Format(time.RFC3339))
			}
			progress.ReportMessage(ctx, progress.StageFetch, message+"...")
		}

		// convert ids and fetch
		strHostIDs := utilconvert.Stringify(batch)
		res, err := s.client.GetHosts(ctx, orgIDStr, strHostIDs, atTime)
		if err != nil {
			// If this is the first batch, return the error immediately
			if batchNum == 0 {
				return HostsResult{}, err
			}

			// Otherwise, record the error, report it, and return partial results
			firstError = err
			progress.ReportError(ctx, progress.StageFetch, err)
			break
		}

		// Store metadata from the last successful request
		lastMeta = responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts)

		// Convert and append to results
		for _, host := range *res.Data {
			domainHost := assets.NewHost(host)
			allHosts = append(allHosts, &domainHost)
		}

		batchesProcessed++
	}

	// Update metadata with total latency and batch count
	if lastMeta != nil {
		lastMeta.Latency = time.Since(start)
		lastMeta.PageCount = uint64(batchesProcessed)
	}

	return HostsResult{
		Meta:         lastMeta,
		Hosts:        allHosts,
		PartialError: cenclierrors.ToPartialError(firstError),
	}, nil
}

func (s *viewService) GetCertificates(
	ctx context.Context,
	orgID mo.Option[identifiers.OrganizationID],
	certificateIDs []assets.CertificateID,
) (CertificatesResult, cenclierrors.CencliError) {
	start := time.Now()
	orgIDStr := utilconvert.OptionalString(orgID)

	// Split IDs into batches based on API limits
	batches := splitSlice(certificateIDs, maxCertificatesPerRequest)
	totalBatches := len(batches)

	var allCertificates []*assets.Certificate
	var lastMeta *responsemeta.ResponseMeta
	var firstError cenclierrors.CencliError
	batchesProcessed := 0

	for batchNum, batch := range batches {
		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			contextErr := cenclierrors.ParseContextError(err)

			// Return partial results with context error
			if len(allCertificates) > 0 {
				if lastMeta != nil {
					lastMeta.Latency = time.Since(start)
					lastMeta.PageCount = uint64(batchesProcessed)
				}
				return CertificatesResult{
					Meta:         lastMeta,
					Certificates: allCertificates,
					PartialError: cenclierrors.ToPartialError(contextErr),
				}, nil
			}
			return CertificatesResult{}, contextErr
		}

		// Report progress for batch fetches
		if totalBatches > 1 {
			progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Fetching certificates batch %d/%d (%d certificates)...", batchNum+1, totalBatches, len(batch)))
		} else if len(certificateIDs) > 1 {
			progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Fetching %d certificates...", len(certificateIDs)))
		}

		// convert ids and fetch
		strCertificateIDs := utilconvert.Stringify(batch)
		res, err := s.client.GetCertificates(ctx, orgIDStr, strCertificateIDs)
		if err != nil {
			// If this is the first batch, return the error immediately
			if batchNum == 0 {
				return CertificatesResult{}, err
			}
			// Otherwise, record the error, report it, and return partial results
			firstError = err
			progress.ReportError(ctx, progress.StageFetch, err)
			break
		}

		// Store metadata from the last successful request
		lastMeta = responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts)

		// Convert and append to results
		for _, certificate := range *res.Data {
			domainCertificate := assets.NewCertificate(certificate)
			allCertificates = append(allCertificates, &domainCertificate)
		}

		batchesProcessed++
	}

	// Update metadata with total latency and batch count
	if lastMeta != nil {
		lastMeta.Latency = time.Since(start)
		lastMeta.PageCount = uint64(batchesProcessed)
	}

	return CertificatesResult{
		Meta:         lastMeta,
		Certificates: allCertificates,
		PartialError: cenclierrors.ToPartialError(firstError),
	}, nil
}

func (s *viewService) GetWebProperties(
	ctx context.Context,
	orgID mo.Option[identifiers.OrganizationID],
	webPropertyIDs []assets.WebPropertyID,
	atTime mo.Option[time.Time],
) (WebPropertiesResult, cenclierrors.CencliError) {
	start := time.Now()
	orgIDStr := utilconvert.OptionalString(orgID)

	// Split IDs into batches based on API limits
	batches := splitSlice(webPropertyIDs, maxWebPropertiesPerRequest)
	totalBatches := len(batches)

	var allWebProperties []*assets.WebProperty
	var lastMeta *responsemeta.ResponseMeta
	var firstError cenclierrors.CencliError
	batchesProcessed := 0

	for batchNum, batch := range batches {
		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			contextErr := cenclierrors.ParseContextError(err)

			// Return partial results with context error
			if len(allWebProperties) > 0 {
				if lastMeta != nil {
					lastMeta.Latency = time.Since(start)
					lastMeta.PageCount = uint64(batchesProcessed)
				}
				return WebPropertiesResult{
					Meta:          lastMeta,
					WebProperties: allWebProperties,
					PartialError:  cenclierrors.ToPartialError(contextErr),
				}, nil
			}
			return WebPropertiesResult{}, contextErr
		}

		// Report progress for batch fetches
		if totalBatches > 1 {
			message := fmt.Sprintf("Fetching web properties batch %d/%d (%d web properties)", batchNum+1, totalBatches, len(batch))
			if atTime.IsPresent() {
				message = fmt.Sprintf("%s at %s", message, atTime.MustGet().Format(time.RFC3339))
			}
			progress.ReportMessage(ctx, progress.StageFetch, message+"...")
		} else if len(webPropertyIDs) > 1 {
			message := fmt.Sprintf("Fetching %d web properties", len(webPropertyIDs))
			if atTime.IsPresent() {
				message = fmt.Sprintf("%s at %s", message, atTime.MustGet().Format(time.RFC3339))
			}
			progress.ReportMessage(ctx, progress.StageFetch, message+"...")
		}

		// convert ids and fetch
		strWebPropertyIDs := utilconvert.Stringify(batch)
		res, err := s.client.GetWebProperties(ctx, orgIDStr, strWebPropertyIDs, atTime)
		if err != nil {
			// If this is the first batch, return the error immediately
			if batchNum == 0 {
				return WebPropertiesResult{}, err
			}
			// Otherwise, record the error, report it, and return partial results
			firstError = err
			progress.ReportError(ctx, progress.StageFetch, err)
			break
		}

		// Store metadata from the last successful request
		lastMeta = responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts)

		// Convert and append to results
		for _, webProperty := range *res.Data {
			domainWebProperty := assets.NewWebProperty(webProperty)
			allWebProperties = append(allWebProperties, &domainWebProperty)
		}

		batchesProcessed++
	}

	// Update metadata with total latency and batch count
	if lastMeta != nil {
		lastMeta.Latency = time.Since(start)
		lastMeta.PageCount = uint64(batchesProcessed)
	}

	return WebPropertiesResult{
		Meta:          lastMeta,
		WebProperties: allWebProperties,
		PartialError:  cenclierrors.ToPartialError(firstError),
	}, nil
}

func splitSlice[T any](items []T, batchSize int) [][]T {
	if batchSize <= 0 {
		return nil
	}
	totalBatches := (len(items) + batchSize - 1) / batchSize
	batches := make([][]T, 0, totalBatches)
	for i := 0; i < len(items); i += batchSize {
		end := min(i+batchSize, len(items))
		batches = append(batches, items[i:end])
	}
	return batches
}
