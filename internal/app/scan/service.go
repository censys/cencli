package scan

import (
	"context"
	"time"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/datetime"
)

//go:generate mockgen -destination=../../../gen/app/scan/mocks/scanservice_mock.go -package=mocks -mock_names Service=MockScanService . Service

// Service requests live rescans and reports on tracked scans.
type Service interface {
	Rescan(ctx context.Context, params RescanParams) (Result, cenclierrors.CencliError)
	Get(ctx context.Context, params GetParams) (Result, cenclierrors.CencliError)
	Wait(ctx context.Context, params WaitParams) (Result, cenclierrors.CencliError)
}

type scanService struct {
	client client.Client
	// sleep paces the Wait poll loop; a field so tests can substitute a fake clock.
	sleep func(ctx context.Context, d time.Duration) error
}

func New(client client.Client) Service {
	return &scanService{client: client, sleep: datetime.SleepContext}
}
