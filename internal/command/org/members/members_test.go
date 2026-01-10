package members

import (
	"bytes"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	orgmocks "github.com/censys/cencli/gen/app/organizations/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/app/organizations"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
)

func TestOrgMembersCommand(t *testing.T) {
	testCases := []struct {
		name    string
		store   func(ctrl *gomock.Controller) store.Store
		service func(ctrl *gomock.Controller) organizations.Service
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		// Success cases
		{
			name: "success - list members with --org-id flag",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) organizations.Service {
				mockSvc := orgmocks.NewMockOrganizationsService(ctrl)
				createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				lastLogin := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
				mockSvc.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")),
					mo.None[uint](),
					mo.None[uint](),
				).Return(organizations.OrganizationMembersResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Data: organizations.OrganizationMembers{
						Members: []organizations.OrganizationMember{
							{
								ID:              uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
								CreatedAt:       mo.Some(createdAt),
								Email:           mo.Some("user1@example.com"),
								FirstName:       mo.Some("John"),
								LastName:        mo.Some("Doe"),
								Roles:           []string{"admin", "viewer"},
								LatestLoginTime: mo.Some(lastLogin),
							},
							{
								ID:        uuid.MustParse("b2c3d4e5-f6a7-8901-bcde-f12345678901"),
								Email:     mo.Some("user2@example.com"),
								FirstName: mo.Some("Jane"),
								LastName:  mo.Some("Smith"),
								Roles:     []string{"viewer"},
							},
						},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"members"`)
				require.Contains(t, stdout, "user1@example.com")
				require.Contains(t, stdout, "user2@example.com")
				require.Contains(t, stdout, "John")
				require.Contains(t, stdout, "admin")
			},
		},
		{
			name: "success - list members with stored org ID",
			store: func(ctrl *gomock.Controller) store.Store {
				mockStore := storemocks.NewMockStore(ctrl)
				mockStore.EXPECT().GetLastUsedGlobalByName(
					gomock.Any(),
					"org-id",
				).Return(&store.ValueForGlobal{
					ID:    1,
					Name:  "org-id",
					Value: "58857aac-4b76-46ec-a567-0e02b2c3d479",
				}, nil)
				return mockStore
			},
			service: func(ctrl *gomock.Controller) organizations.Service {
				mockSvc := orgmocks.NewMockOrganizationsService(ctrl)
				mockSvc.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					identifiers.NewOrganizationID(uuid.MustParse("58857aac-4b76-46ec-a567-0e02b2c3d479")),
					mo.None[uint](),
					mo.None[uint](),
				).Return(organizations.OrganizationMembersResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Data: organizations.OrganizationMembers{
						Members: []organizations.OrganizationMember{
							{
								ID:    uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
								Email: mo.Some("user@example.com"),
								Roles: []string{"admin"},
							},
						},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "user@example.com")
			},
		},
		{
			name: "success - empty members list",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) organizations.Service {
				mockSvc := orgmocks.NewMockOrganizationsService(ctrl)
				mockSvc.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")),
					mo.None[uint](),
					mo.None[uint](),
				).Return(organizations.OrganizationMembersResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 50*time.Millisecond, 1),
					Data: organizations.OrganizationMembers{
						Members: []organizations.OrganizationMember{},
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"members"`)
			},
		},

		// Error cases
		{
			name: "error - no org ID configured",
			store: func(ctrl *gomock.Controller) store.Store {
				mockStore := storemocks.NewMockStore(ctrl)
				mockStore.EXPECT().GetLastUsedGlobalByName(
					gomock.Any(),
					"org-id",
				).Return(nil, store.ErrGlobalNotFound)
				return mockStore
			},
			service: func(ctrl *gomock.Controller) organizations.Service {
				return orgmocks.NewMockOrganizationsService(ctrl)
			},
			args: []string{"--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no organization ID configured")
			},
		},
		{
			name: "error - invalid org ID format",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) organizations.Service {
				return orgmocks.NewMockOrganizationsService(ctrl)
			},
			args: []string{"--org-id", "invalid-uuid", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid uuid")
			},
		},
		{
			name: "error - service returns error",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) organizations.Service {
				mockSvc := orgmocks.NewMockOrganizationsService(ctrl)
				mockSvc.EXPECT().ListOrganizationMembers(
					gomock.Any(),
					identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")),
					mo.None[uint](),
					mo.None[uint](),
				).Return(organizations.OrganizationMembersResult{}, cenclierrors.NewCencliError(errors.New("organization not found")))
				return mockSvc
			},
			args: []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "organization not found")
			},
		},
		{
			name: "error - too many arguments",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) organizations.Service {
				return orgmocks.NewMockOrganizationsService(ctrl)
			},
			args: []string{"extra-arg", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "accepts 0 arg(s), received 1")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			viper.Reset()
			cfg, err := config.New(tempDir)
			require.NoError(t, err)

			var stdout, stderr bytes.Buffer
			formatter.Stdout = &stdout
			formatter.Stderr = &stderr

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			orgSvc := tc.service(ctrl)
			opts := []command.ContextOpts{command.WithOrganizationsService(orgSvc)}

			cmdContext := command.NewCommandContext(cfg, tc.store(ctrl), opts...)
			rootCmd, err := command.RootCommandToCobra(NewMembersCommand(cmdContext))
			require.NoError(t, err)

			rootCmd.SetArgs(tc.args)
			execErr := rootCmd.Execute()
			tc.assert(t, stdout.String(), stderr.String(), cenclierrors.NewCencliError(execErr))
		})
	}
}
