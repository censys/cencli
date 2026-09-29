package scan

import (
	"context"
	"fmt"
	"net/http"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
)

func (s *scanService) Rescan(
	ctx context.Context,
	params RescanParams,
) (Result, cenclierrors.CencliError) {
	// The API has no free-wallet path for scans.
	if params.OrgID.IsZero() {
		return Result{}, cenclierrors.NewNoOrgIDError()
	}

	progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Requesting a rescan of %s...", params.WebProperty))
	res, err := s.client.CreateWebPropertyRescan(ctx, params.OrgID.String(), params.WebProperty.Hostname, params.WebProperty.Port)
	if err != nil {
		return Result{}, classifyRescanError(ctx, err)
	}
	result, mapErr := newScanResult(res)
	if mapErr != nil {
		// A success without a scan was most likely accepted and charged.
		return result, NewRescanUncertainError(mapErr)
	}
	return result, nil
}

// classifyRescanError marks a failure uncertain unless it proves the API
// rejected the request: a 5xx, a conflict or a transport error can follow an
// accepted scan.
func classifyRescanError(ctx context.Context, err client.ClientError) cenclierrors.CencliError {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return cenclierrors.ParseContextError(ctxErr)
	}
	if client.IsFeatureNotEnabled(err) {
		return NewRescanNotEnabledError()
	}
	status := err.StatusCode()
	if status.IsAbsent() || status.MustGet() >= 500 || status.MustGet() == http.StatusConflict {
		return NewRescanUncertainError(err)
	}
	return err
}
