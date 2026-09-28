package rescan

import (
	"context"
	"fmt"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/samber/mo"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

const pollInterval = 5 * time.Second

//go:generate mockgen -destination=../../../gen/app/rescan/mocks/rescanservice_mock.go -package=mocks github.com/censys/cencli/internal/app/rescan Service

// Service initiates a live rescan and polls until it completes.
type Service interface {
	Rescan(ctx context.Context, params Params) (Result, cenclierrors.CencliError)
}

type rescanService struct {
	client client.Client
}

func New(c client.Client) Service {
	return &rescanService{client: c}
}

func (s *rescanService) Rescan(ctx context.Context, params Params) (Result, cenclierrors.CencliError) {
	orgID := mo.None[string]()
	if params.OrgID.IsPresent() {
		orgID = mo.Some(params.OrgID.MustGet().String())
	}

	target, err := buildTarget(params)
	if err != nil {
		return Result{}, err
	}

	progress.ReportMessage(ctx, progress.StageFetch, "Initiating live rescan...")
	createResult, cerr := s.client.CreateTrackedScan(ctx, orgID, target)
	if cerr != nil {
		return Result{}, cerr
	}

	scan := createResult.Data
	if scan == nil || scan.TrackedScanID == nil {
		return Result{}, cenclierrors.NewCencliError(fmt.Errorf("rescan created but returned no scan ID"))
	}

	scanID := *scan.TrackedScanID
	meta := responsemeta.NewResponseMeta(createResult.Metadata.Request, createResult.Metadata.Response, createResult.Metadata.Latency, createResult.Metadata.Attempts)

	progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Scan started (ID: %s). Waiting for completion...", scanID))

	// Poll until completed or context cancelled.
	for {
		if completed := scan.GetCompleted(); completed != nil && *completed {
			return Result{Meta: meta, Scan: scan}, nil
		}

		select {
		case <-ctx.Done():
			return Result{}, cenclierrors.ParseContextError(ctx.Err())
		case <-time.After(pollInterval):
		}

		progress.ReportMessage(ctx, progress.StageFetch, fmt.Sprintf("Scan %s in progress...", scanID))

		pollResult, cerr := s.client.GetTrackedScan(ctx, orgID, scanID)
		if cerr != nil {
			return Result{}, cerr
		}
		if pollResult.Data != nil {
			scan = pollResult.Data
		}
	}
}

func buildTarget(params Params) (components.ScansRescanInputBodyTarget, cenclierrors.CencliError) {
	switch params.TargetType {
	case TargetTypeService:
		return components.CreateScansRescanInputBodyTargetOne(components.One{
			ServiceID: components.TargetServiceID{
				IP:                params.IP,
				Port:              params.Port,
				Protocol:          params.Protocol,
				TransportProtocol: params.TransportProtocol,
			},
		}), nil
	case TargetTypeWebOrigin:
		return components.CreateScansRescanInputBodyTargetTwo(components.Two{
			WebOrigin: components.TargetWebOrigin{
				Hostname: params.Hostname,
				Port:     params.Port,
			},
		}), nil
	default:
		return components.ScansRescanInputBodyTarget{}, cenclierrors.NewCencliError(fmt.Errorf("unknown target type"))
	}
}
