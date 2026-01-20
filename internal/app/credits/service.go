package credits

import (
	"context"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

//go:generate mockgen -destination=../../../gen/app/credits/mocks/creditservice_mock.go -package=mocks -mock_names Service=MockCreditsService . Service

// Service provides credit details capabilities.
type Service interface {
	// GetOrganizationCreditDetails retrieves the credit details for an organization.
	GetOrganizationCreditDetails(
		ctx context.Context,
		orgID identifiers.OrganizationID,
	) (OrganizationCreditDetailsResult, cenclierrors.CencliError)
	// GetUserCreditDetails retrieves the credit details for the current user.
	GetUserCreditDetails(
		ctx context.Context,
	) (UserCreditDetailsResult, cenclierrors.CencliError)
}

type creditsService struct {
	client client.Client
}

func New(client client.Client) Service {
	return &creditsService{client: client}
}

func (s *creditsService) GetOrganizationCreditDetails(
	ctx context.Context,
	orgID identifiers.OrganizationID,
) (OrganizationCreditDetailsResult, cenclierrors.CencliError) {
	res, err := s.client.GetOrganizationCreditDetails(ctx, orgID.String())
	if err != nil {
		return OrganizationCreditDetailsResult{}, err
	}
	return OrganizationCreditDetailsResult{
		Meta: responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts),
		Data: parseOrganizationCreditDetails(res.Data),
	}, nil
}

func (s *creditsService) GetUserCreditDetails(ctx context.Context) (UserCreditDetailsResult, cenclierrors.CencliError) {
	res, err := s.client.GetUserCreditDetails(ctx)
	if err != nil {
		return UserCreditDetailsResult{}, err
	}
	return UserCreditDetailsResult{
		Meta: responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts),
		Data: parseUserCreditDetails(res.Data),
	}, nil
}
