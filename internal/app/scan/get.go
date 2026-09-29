package scan

import (
	"context"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

func (s *scanService) Get(
	ctx context.Context,
	params GetParams,
) (Result, cenclierrors.CencliError) {
	if params.OrgID.IsZero() {
		return Result{}, cenclierrors.NewNoOrgIDError()
	}
	scanID, err := RequireScanID(params.ScanID)
	if err != nil {
		return Result{}, err
	}

	res, getErr := s.client.GetTrackedScan(ctx, params.OrgID.String(), scanID)
	if getErr != nil {
		return Result{}, getErr
	}
	return newScanResult(res)
}

// newScanResult maps a client response onto the domain result. An empty
// envelope keeps its metadata but is an error.
func newScanResult(res client.Result[components.TrackedScan]) (Result, cenclierrors.CencliError) {
	meta := responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts)
	if res.Data == nil {
		return Result{Meta: meta}, &emptyScanError{}
	}
	scan := mapTrackedScan(res.Data)
	return Result{Meta: meta, Scan: &scan}, nil
}

func mapTrackedScan(s *components.TrackedScan) TrackedScan {
	out := TrackedScan{
		CreateTime: s.GetCreateTime(),
		Target:     s.GetTarget(),
		Tasks:      make([]Task, 0, len(s.GetTasks())),
	}
	if id := s.GetTrackedScanID(); id != nil {
		out.ID = *id
	}
	if completed := s.GetCompleted(); completed != nil {
		out.Completed = *completed
	}
	for _, t := range s.GetTasks() {
		task := Task{UpdateTime: t.GetUpdateTime()}
		if d := t.GetDescription(); d != nil {
			task.Description = *d
		}
		if st := t.GetStatus(); st != nil {
			task.Status = string(*st)
		}
		out.Tasks = append(out.Tasks, task)
	}
	return out
}
