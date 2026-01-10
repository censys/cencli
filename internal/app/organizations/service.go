package organizations

import (
	"context"

	"github.com/samber/mo"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

//go:generate mockgen -destination=../../../gen/app/organizations/mocks/organizationservice_mock.go -package=mocks -mock_names Service=MockOrganizationsService . Service

// Service provides organization and member details capabilities.
type Service interface {
	// GetOrganizationDetails retrieves the details for an organization.
	GetOrganizationDetails(
		ctx context.Context,
		orgID identifiers.OrganizationID,
	) (OrganizationDetailsResult, cenclierrors.CencliError)
	// ListOrganizationMembers retrieves the members for an organization.
	// If no pagination is provided, the client will return all members.
	ListOrganizationMembers(
		ctx context.Context,
		orgID identifiers.OrganizationID,
		pageSize mo.Option[uint],
		maxPages mo.Option[uint],
	) (OrganizationMembersResult, cenclierrors.CencliError)
}

type organizationsService struct {
	client client.Client
}

func New(client client.Client) Service {
	return &organizationsService{client: client}
}

func (s *organizationsService) GetOrganizationDetails(
	ctx context.Context,
	orgID identifiers.OrganizationID,
) (OrganizationDetailsResult, cenclierrors.CencliError) {
	res, err := s.client.GetOrganizationDetails(ctx, orgID.String())
	if err != nil {
		return OrganizationDetailsResult{}, err
	}
	return OrganizationDetailsResult{
		Meta: responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts),
		Data: parseOrganizationDetails(res.Data),
	}, nil
}

func (s *organizationsService) ListOrganizationMembers(
	ctx context.Context,
	orgID identifiers.OrganizationID,
	pageSize mo.Option[uint],
	maxPages mo.Option[uint],
) (OrganizationMembersResult, cenclierrors.CencliError) {
	var allMembers []OrganizationMember
	var lastMeta *responsemeta.ResponseMeta
	var pagesProcessed uint
	pageToken := mo.None[string]()

	// Convert pageSize from uint to int for the client
	var clientPageSize mo.Option[int]
	if pageSize.IsPresent() {
		clientPageSize = mo.Some(int(pageSize.MustGet()))
	}

	for {
		// Check if we've reached maxPages
		if maxPages.IsPresent() && pagesProcessed >= maxPages.MustGet() {
			break
		}

		// Fetch a page of members
		res, err := s.client.ListOrganizationMembers(ctx, orgID.String(), clientPageSize, pageToken)
		if err != nil {
			// Return error immediately - no partial results for this endpoint
			return OrganizationMembersResult{}, err
		}

		// Store metadata from the last successful request
		if res.Metadata.Request != nil || res.Metadata.Response != nil {
			lastMeta = responsemeta.NewResponseMeta(
				res.Metadata.Request,
				res.Metadata.Response,
				res.Metadata.Latency,
				res.Metadata.Attempts,
			)
		}

		// Parse and append members from this page
		if res.Data != nil {
			pageMembers := parseOrganizationMembers(res.Data)
			allMembers = append(allMembers, pageMembers.Members...)
		}

		pagesProcessed++

		// Check if there's a next page
		if res.Data == nil || res.Data.Pagination.GetNextPageToken() == nil || *res.Data.Pagination.GetNextPageToken() == "" {
			break
		}

		// Set the next page token
		pageToken = mo.Some(*res.Data.Pagination.GetNextPageToken())
	}

	return OrganizationMembersResult{
		Meta: lastMeta,
		Data: OrganizationMembers{
			Members: allMembers,
		},
	}, nil
}
