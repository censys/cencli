package censys

import (
	"context"
	"time"

	"github.com/samber/mo"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/operations"
)

//go:generate mockgen -destination=../../../../gen/client/mocks/accountmanagement_mock.go -package=mocks github.com/censys/cencli/internal/pkg/clients/censys AccountManagementClient
type AccountManagementClient interface {
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/accountmanagement#getorganizationcredits
	GetOrganizationCreditDetails(
		ctx context.Context,
		orgID string,
	) (Result[components.OrganizationCredits], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/accountmanagement#getusercredits
	GetUserCreditDetails(
		ctx context.Context,
	) (Result[components.UserCredits], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/accountmanagement#getorganizationdetails
	GetOrganizationDetails(
		ctx context.Context,
		orgID string,
	) (Result[components.OrganizationDetails], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/accountmanagement#listorganizationmembers
	ListOrganizationMembers(
		ctx context.Context,
		orgID string,
		pageSize mo.Option[int],
		pageToken mo.Option[string],
	) (Result[components.OrganizationMembersList], ClientError)
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
	orgID string,
) (Result[components.OrganizationCredits], ClientError) {
	start := time.Now()
	var res *operations.V3AccountmanagementOrgCreditsResponse
	err, attempts := a.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = a.censysSDK.client.AccountManagement.GetOrganizationCredits(ctx, operations.V3AccountmanagementOrgCreditsRequest{
			OrganizationID: orgID,
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

func (a *accountManagementSDK) GetOrganizationDetails(
	ctx context.Context,
	orgID string,
) (Result[components.OrganizationDetails], ClientError) {
	start := time.Now()
	var res *operations.V3AccountmanagementOrgDetailsResponse
	err, attempts := a.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = a.censysSDK.client.AccountManagement.GetOrganizationDetails(ctx, operations.V3AccountmanagementOrgDetailsRequest{
			OrganizationID:      orgID,
			IncludeMemberCounts: boolPtr(true),
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.OrganizationDetails]{}
		return zero, err
	}
	return Result[components.OrganizationDetails]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     res.GetResponseEnvelopeOrganizationDetails().GetResult(),
	}, nil
}

func (a *accountManagementSDK) ListOrganizationMembers(
	ctx context.Context,
	orgID string,
	pageSize mo.Option[int],
	pageToken mo.Option[string],
) (Result[components.OrganizationMembersList], ClientError) {
	start := time.Now()
	var res *operations.V3AccountmanagementListOrgMembersResponse
	err, attempts := a.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = a.censysSDK.client.AccountManagement.ListOrganizationMembers(ctx, operations.V3AccountmanagementListOrgMembersRequest{
			OrganizationID: orgID,
			PageSize:       pageSize.ToPointer(),
			PageToken:      pageToken.ToPointer(),
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.OrganizationMembersList]{}
		return zero, err
	}
	return Result[components.OrganizationMembersList]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     res.GetResponseEnvelopeOrganizationMembersList().GetResult(),
	}, nil
}

func boolPtr(b bool) *bool {
	return &b
}
