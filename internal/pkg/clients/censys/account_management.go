package censys

import (
	"context"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/operations"
	"github.com/google/uuid"
)

//go:generate mockgen -destination=../../../../gen/client/mocks/accountmanagement_mock.go -package=mocks github.com/censys/cencli/internal/pkg/clients/censys AccountManagementClient
type AccountManagementClient interface {
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/accountmanagement#getorganizationcredits
	GetOrganizationCreditDetails(
		ctx context.Context,
		orgID uuid.UUID,
	) (Result[components.OrganizationCredits], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/accountmanagement#getusercredits
	GetUserCreditDetails(
		ctx context.Context,
	) (Result[components.UserCredits], ClientError)
}

type accountManagementSDK struct {
	*censysSDK
}

var _ AccountManagementClient = &accountManagementSDK{}

func newAccountManagementSDK(censysSDK *censysSDK) *accountManagementSDK {
	return &accountManagementSDK{censysSDK: censysSDK}
}

func (a *accountManagementSDK) GetOrganizationCreditDetails(
	ctx context.Context,
	orgID uuid.UUID,
) (Result[components.OrganizationCredits], ClientError) {
	start := time.Now()
	var res *operations.V3AccountmanagementOrgCreditsResponse
	err, attempts := a.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = a.censysSDK.client.AccountManagement.GetOrganizationCredits(ctx, operations.V3AccountmanagementOrgCreditsRequest{
			OrganizationID: orgID.String(),
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.OrganizationCredits]{}
		return zero, err
	}
	return Result[components.OrganizationCredits]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     res.GetResponseEnvelopeOrganizationCredits().GetResult(),
	}, nil
}

func (a *accountManagementSDK) GetUserCreditDetails(
	ctx context.Context,
) (Result[components.UserCredits], ClientError) {
	start := time.Now()
	var res *operations.V3AccountmanagementUserCreditsResponse
	err, attempts := a.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = a.censysSDK.client.AccountManagement.GetUserCredits(ctx)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.UserCredits]{}
		return zero, err
	}
	return Result[components.UserCredits]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     res.GetResponseEnvelopeUserCredits().GetResult(),
	}, nil
}
