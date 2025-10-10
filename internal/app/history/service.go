package history

import (
	"context"
	"time"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

//go:generate mockgen -destination=../../../gen/app/history/mocks/historyservice_mock.go -package=mocks -mock_names Service=MockHistoryService . Service

// Service provides history fetching capabilities.
type Service interface {
	GetHostHistory(
		ctx context.Context,
		orgID mo.Option[identifiers.OrganizationID],
		host assets.HostID,
		fromTime time.Time,
		toTime time.Time,
	) (HostHistoryResult, cenclierrors.CencliError)

	GetCertificateHistory(
		ctx context.Context,
		orgID mo.Option[identifiers.OrganizationID],
		certificateID assets.CertificateID,
		fromTime time.Time,
		toTime time.Time,
	) (CertificateHistoryResult, cenclierrors.CencliError)

	GetWebPropertyHistory(
		ctx context.Context,
		orgID mo.Option[identifiers.OrganizationID],
		webPropertyID assets.WebPropertyID,
		fromTime time.Time,
		toTime time.Time,
	) (WebPropertyHistoryResult, cenclierrors.CencliError)
}

type historyService struct {
	client client.Client
}

func New(client client.Client) Service {
	return &historyService{client: client}
}
