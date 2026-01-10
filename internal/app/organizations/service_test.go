package organizations

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"
	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

func TestOrganizationsService_GetOrganizationDetails(t *testing.T) {
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		orgID  uuid.UUID
		ctx    func() context.Context
		assert func(t *testing.T, res OrganizationDetailsResult, err cenclierrors.CencliError)
	}{
		{
			name:  "success - basic organization details",
			orgID: uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				mockClient.EXPECT().GetOrganizationDetails(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
				).Return(client.Result[components.OrganizationDetails]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationDetails{
						UID:       "f47ac10b-58cc-4372-a567-0e02b2c3d479",
						CreatedAt: &createdAt,
						Name:      "Test Organization",
					},
				}, nil)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationDetailsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Equal(t, 100*time.Millisecond, res.Meta.Latency)
				require.Equal(t, uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"), res.Data.ID)
				require.Equal(t, "Test Organization", res.Data.Name)
				require.True(t, res.Data.CreatedAt.IsPresent())
				require.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), res.Data.CreatedAt.MustGet())
			},
		},
		{
			name:  "error - organization not found",
			orgID: uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				detail := "Organization not found"
				status := int64(404)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
					Detail: &detail,
					Status: &status,
				})
				mockClient.EXPECT().GetOrganizationDetails(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
				).Return(client.Result[components.OrganizationDetails]{}, structuredErr)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationDetailsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, OrganizationDetailsResult{}, res)

				var structuredErr client.ClientStructuredError
				require.True(t, errors.As(err, &structuredErr))
				require.True(t, structuredErr.StatusCode().IsPresent())
				require.Equal(t, int64(404), structuredErr.StatusCode().MustGet())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := tc.client(ctrl)
			svc := New(c)

			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}

			res, err := svc.GetOrganizationDetails(ctx, identifiers.NewOrganizationID(tc.orgID))
			tc.assert(t, res, err)
		})
	}
}

func TestOrganizationsService_ListOrganizationMembers(t *testing.T) {
	testCases := []struct {
		name     string
		client   func(ctrl *gomock.Controller) client.Client
		orgID    uuid.UUID
		pageSize mo.Option[uint]
		maxPages mo.Option[uint]
		ctx      func() context.Context
		assert   func(t *testing.T, res OrganizationMembersResult, err cenclierrors.CencliError)
	}{
		{
			name:     "success - single page with members",
			orgID:    uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			pageSize: mo.None[uint](),
			maxPages: mo.None[uint](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.None[int](),
					mo.None[string](),
				).Return(client.Result[components.OrganizationMembersList]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationMembersList{
						Members: []components.OrganizationMember{
							{
								UID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
								CreatedAt: &createdAt,
								Email:     "user1@example.com",
								FirstName: "John",
								LastName:  "Doe",
								Roles:     []string{"admin", "viewer"},
							},
							{
								UID:       "b2c3d4e5-f6a7-8901-bcde-f12345678901",
								Email:     "user2@example.com",
								FirstName: "Jane",
								LastName:  "Smith",
								Roles:     []string{"viewer"},
							},
						},
						Pagination: components.PaginationInfo{
							NextPageToken: nil,
							PageSize:      10,
						},
					},
				}, nil)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationMembersResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Equal(t, 100*time.Millisecond, res.Meta.Latency)
				require.Len(t, res.Data.Members, 2)

				// Check first member
				require.Equal(t, uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890"), res.Data.Members[0].ID)
				require.True(t, res.Data.Members[0].Email.IsPresent())
				require.Equal(t, "user1@example.com", res.Data.Members[0].Email.MustGet())
				require.True(t, res.Data.Members[0].FirstName.IsPresent())
				require.Equal(t, "John", res.Data.Members[0].FirstName.MustGet())
				require.True(t, res.Data.Members[0].LastName.IsPresent())
				require.Equal(t, "Doe", res.Data.Members[0].LastName.MustGet())
				require.Equal(t, []string{"admin", "viewer"}, res.Data.Members[0].Roles)

				// Check second member
				require.Equal(t, uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-f12345678901"), res.Data.Members[1].ID)
				require.Equal(t, "user2@example.com", res.Data.Members[1].Email.MustGet())
				require.Equal(t, []string{"viewer"}, res.Data.Members[1].Roles)
			},
		},
		{
			name:     "success - empty members list",
			orgID:    uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			pageSize: mo.None[uint](),
			maxPages: mo.None[uint](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.None[int](),
					mo.None[string](),
				).Return(client.Result[components.OrganizationMembersList]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  50 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationMembersList{
						Members: []components.OrganizationMember{},
						Pagination: components.PaginationInfo{
							NextPageToken: nil,
							PageSize:      10,
						},
					},
				}, nil)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationMembersResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Data.Members, 0)
			},
		},
		{
			name:     "success - multiple pages with maxPages",
			orgID:    uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			pageSize: mo.Some(uint(2)),
			maxPages: mo.Some(uint(2)),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// First page
				token1 := "token1"
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.Some(2),
					mo.None[string](),
				).Return(client.Result[components.OrganizationMembersList]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationMembersList{
						Members: []components.OrganizationMember{
							{
								UID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
								Email:     "user1@example.com",
								FirstName: "User",
								LastName:  "One",
								Roles:     []string{"admin"},
							},
							{
								UID:       "b2c3d4e5-f6a7-8901-bcde-f12345678901",
								Email:     "user2@example.com",
								FirstName: "User",
								LastName:  "Two",
								Roles:     []string{"viewer"},
							},
						},
						Pagination: components.PaginationInfo{
							NextPageToken: &token1,
							PageSize:      2,
						},
					},
				}, nil)

				// Second page
				token2 := "token2"
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.Some(2),
					mo.Some("token1"),
				).Return(client.Result[components.OrganizationMembersList]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationMembersList{
						Members: []components.OrganizationMember{
							{
								UID:       "c3d4e5f6-a7b8-9012-cdef-123456789012",
								Email:     "user3@example.com",
								FirstName: "User",
								LastName:  "Three",
								Roles:     []string{"editor"},
							},
							{
								UID:       "d4e5f6a7-b8c9-0123-def1-234567890123",
								Email:     "user4@example.com",
								FirstName: "User",
								LastName:  "Four",
								Roles:     []string{"viewer"},
							},
						},
						Pagination: components.PaginationInfo{
							NextPageToken: &token2, // Has more pages but we stop at maxPages
							PageSize:      2,
						},
					},
				}, nil)

				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationMembersResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Data.Members, 4)
				require.Equal(t, "user1@example.com", res.Data.Members[0].Email.MustGet())
				require.Equal(t, "user2@example.com", res.Data.Members[1].Email.MustGet())
				require.Equal(t, "user3@example.com", res.Data.Members[2].Email.MustGet())
				require.Equal(t, "user4@example.com", res.Data.Members[3].Email.MustGet())
			},
		},
		{
			name:     "success - pagination until no next token",
			orgID:    uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			pageSize: mo.Some(uint(1)),
			maxPages: mo.None[uint](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// First page
				token1 := "token1"
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.Some(1),
					mo.None[string](),
				).Return(client.Result[components.OrganizationMembersList]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  50 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationMembersList{
						Members: []components.OrganizationMember{
							{
								UID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
								Email:     "user1@example.com",
								FirstName: "User",
								LastName:  "One",
								Roles:     []string{"admin"},
							},
						},
						Pagination: components.PaginationInfo{
							NextPageToken: &token1,
							PageSize:      1,
						},
					},
				}, nil)

				// Second page (last page)
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.Some(1),
					mo.Some("token1"),
				).Return(client.Result[components.OrganizationMembersList]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  50 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationMembersList{
						Members: []components.OrganizationMember{
							{
								UID:       "b2c3d4e5-f6a7-8901-bcde-f12345678901",
								Email:     "user2@example.com",
								FirstName: "User",
								LastName:  "Two",
								Roles:     []string{"viewer"},
							},
						},
						Pagination: components.PaginationInfo{
							NextPageToken: nil, // No more pages
							PageSize:      1,
						},
					},
				}, nil)

				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationMembersResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Data.Members, 2)
				require.Equal(t, "user1@example.com", res.Data.Members[0].Email.MustGet())
				require.Equal(t, "user2@example.com", res.Data.Members[1].Email.MustGet())
			},
		},
		{
			name:     "success - empty string next token treated as no next page",
			orgID:    uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			pageSize: mo.None[uint](),
			maxPages: mo.None[uint](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				emptyToken := ""
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.None[int](),
					mo.None[string](),
				).Return(client.Result[components.OrganizationMembersList]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  50 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationMembersList{
						Members: []components.OrganizationMember{
							{
								UID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
								Email:     "user1@example.com",
								FirstName: "User",
								LastName:  "One",
								Roles:     []string{"admin"},
							},
						},
						Pagination: components.PaginationInfo{
							NextPageToken: &emptyToken, // Empty string should be treated as no next page
							PageSize:      10,
						},
					},
				}, nil)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationMembersResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Data.Members, 1)
			},
		},
		{
			name:     "error - first page fails",
			orgID:    uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			pageSize: mo.None[uint](),
			maxPages: mo.None[uint](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				detail := "Organization not found"
				status := int64(404)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
					Detail: &detail,
					Status: &status,
				})
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.None[int](),
					mo.None[string](),
				).Return(client.Result[components.OrganizationMembersList]{}, structuredErr)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationMembersResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, OrganizationMembersResult{}, res)

				var structuredErr client.ClientStructuredError
				require.True(t, errors.As(err, &structuredErr))
				require.True(t, structuredErr.StatusCode().IsPresent())
				require.Equal(t, int64(404), structuredErr.StatusCode().MustGet())
			},
		},
		{
			name:     "error - second page fails",
			orgID:    uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			pageSize: mo.Some(uint(1)),
			maxPages: mo.None[uint](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)

				// First page succeeds
				token1 := "token1"
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.Some(1),
					mo.None[string](),
				).Return(client.Result[components.OrganizationMembersList]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  50 * time.Millisecond,
						Attempts: 1,
					},
					Data: &components.OrganizationMembersList{
						Members: []components.OrganizationMember{
							{
								UID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
								Email:     "user1@example.com",
								FirstName: "User",
								LastName:  "One",
								Roles:     []string{"admin"},
							},
						},
						Pagination: components.PaginationInfo{
							NextPageToken: &token1,
							PageSize:      1,
						},
					},
				}, nil)

				// Second page fails
				genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "Internal server error",
					StatusCode: 500,
					Body:       "Server error",
				})
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.Some(1),
					mo.Some("token1"),
				).Return(client.Result[components.OrganizationMembersList]{}, genericErr)

				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationMembersResult, err cenclierrors.CencliError) {
				// Should return error, not partial results
				require.Error(t, err)
				require.Equal(t, OrganizationMembersResult{}, res)

				var genericErr client.ClientGenericError
				require.True(t, errors.As(err, &genericErr))
				require.Equal(t, int64(500), genericErr.StatusCode().MustGet())
			},
		},
		{
			name:     "error - generic client error",
			orgID:    uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			pageSize: mo.None[uint](),
			maxPages: mo.None[uint](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "Service unavailable",
					StatusCode: 503,
					Body:       "Try again later",
				})
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.None[int](),
					mo.None[string](),
				).Return(client.Result[components.OrganizationMembersList]{}, genericErr)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationMembersResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, OrganizationMembersResult{}, res)

				var genericErr client.ClientGenericError
				require.True(t, errors.As(err, &genericErr))
				require.Equal(t, int64(503), genericErr.StatusCode().MustGet())
			},
		},
		{
			name:     "error - unknown client error",
			orgID:    uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			pageSize: mo.None[uint](),
			maxPages: mo.None[uint](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				unknownErr := client.NewClientError(errors.New("network timeout"))
				mockClient.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					"f47ac10b-58cc-4372-a567-0e02b2c3d479",
					mo.None[int](),
					mo.None[string](),
				).Return(client.Result[components.OrganizationMembersList]{}, unknownErr)
				return mockClient
			},
			ctx: nil,
			assert: func(t *testing.T, res OrganizationMembersResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, OrganizationMembersResult{}, res)

				var unknownErr client.ClientError
				require.True(t, errors.As(err, &unknownErr))
				require.Contains(t, err.Error(), "network timeout")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := tc.client(ctrl)
			svc := New(c)

			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}

			res, err := svc.ListOrganizationMembers(ctx, identifiers.NewOrganizationID(tc.orgID), tc.pageSize, tc.maxPages)
			tc.assert(t, res, err)
		})
	}
}
