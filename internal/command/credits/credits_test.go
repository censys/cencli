package credits

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

	creditsmocks "github.com/censys/cencli/gen/app/credits/mocks"
	clientmocks "github.com/censys/cencli/gen/client/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/app/credits"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
)

func TestCreditsCommand(t *testing.T) {
	testCases := []struct {
		name    string
		store   func(ctrl *gomock.Controller) store.Store
		service func(ctrl *gomock.Controller) credits.Service
		client  func(ctrl *gomock.Controller) *clientmocks.MockClient
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		// Success cases - free user credits
		{
			name: "success - free user credits with --free-user flag",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).Return(credits.UserCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Data: credits.UserCreditDetails{
						Balance:  500,
						ResetsAt: mo.Some(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)),
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"--free-user"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "500")
			},
		},
		{
			name: "success - free user credits with -u short flag",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).Return(credits.UserCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 80*time.Millisecond, 1),
					Data: credits.UserCreditDetails{
						Balance: 1000,
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"-u"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "1000")
			},
		},
		{
			name: "success - default to free user when no stored org ID",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).Return(credits.UserCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 90*time.Millisecond, 1),
					Data: credits.UserCreditDetails{
						Balance: 250,
					},
				}, nil)
				return mockSvc
			},
			client: func(ctrl *gomock.Controller) *clientmocks.MockClient {
				mockClient := clientmocks.NewMockClient(ctrl)
				mockClient.EXPECT().HasOrgID().Return(false)
				return mockClient
			},
			args: []string{},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "250")
			},
		},

		// Success cases - organization credits
		{
			name: "success - org credits with --org-id flag",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).Return(credits.OrganizationCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 120*time.Millisecond, 1),
					Data: credits.OrganizationCreditDetails{
						Balance:           5000,
						CreditExpirations: []credits.CreditExpiration{},
						AutoReplenishConfig: credits.AutoReplenishConfig{
							Enabled: false,
						},
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "5000")
			},
		},
		{
			name: "success - org credits with -o short flag",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("12345678-1234-1234-1234-123456789abc"),
				).Return(credits.OrganizationCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Data: credits.OrganizationCreditDetails{
						Balance:           10000,
						CreditExpirations: []credits.CreditExpiration{},
						AutoReplenishConfig: credits.AutoReplenishConfig{
							Enabled: false,
						},
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"-o", "12345678-1234-1234-1234-123456789abc"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "10000")
			},
		},
		{
			name: "success - org credits with stored org ID (no flags)",
			store: func(ctrl *gomock.Controller) store.Store {
				mockStore := storemocks.NewMockStore(ctrl)
				// Mock the store to return the stored org ID
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
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				// When using stored org ID, the command retrieves and passes the actual stored ID
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("58857aac-4b76-46ec-a567-0e02b2c3d479"),
				).Return(credits.OrganizationCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 150*time.Millisecond, 1),
					Data: credits.OrganizationCreditDetails{
						Balance:           7500,
						CreditExpirations: []credits.CreditExpiration{},
						AutoReplenishConfig: credits.AutoReplenishConfig{
							Enabled: false,
						},
					},
				}, nil)
				return mockSvc
			},
			client: func(ctrl *gomock.Controller) *clientmocks.MockClient {
				mockClient := clientmocks.NewMockClient(ctrl)
				mockClient.EXPECT().HasOrgID().Return(true)
				return mockClient
			},
			args: []string{},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "7500")
			},
		},
		{
			name: "success - org credits with credit expirations",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).Return(credits.OrganizationCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Data: credits.OrganizationCreditDetails{
						Balance: 3000,
						CreditExpirations: []credits.CreditExpiration{
							{
								Balance:        1500,
								CreationDate:   mo.Some(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
								ExpirationDate: mo.Some(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
							},
						},
						AutoReplenishConfig: credits.AutoReplenishConfig{
							Enabled: false,
						},
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "3000")
				require.Contains(t, stdout, `"credit_expirations"`)
				require.Contains(t, stdout, "1500")
			},
		},
		{
			name: "success - org credits with auto-replenish enabled",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).Return(credits.OrganizationCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Data: credits.OrganizationCreditDetails{
						Balance:           2000,
						CreditExpirations: []credits.CreditExpiration{},
						AutoReplenishConfig: credits.AutoReplenishConfig{
							Enabled:   true,
							Threshold: mo.Some(int64(100)),
							Amount:    mo.Some(int64(500)),
						},
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "2000")
				require.Contains(t, stdout, `"auto_replenish_config"`)
				require.Contains(t, stdout, `"enabled": true`)
				require.Contains(t, stdout, `"threshold"`)
				require.Contains(t, stdout, "100")
			},
		},

		// Error cases - flag validation
		{
			name: "error - conflicting flags (--org-id and --free-user)",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				return creditsmocks.NewMockCreditsService(ctrl)
			},
			client: nil,
			args:   []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479", "--free-user"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Equal(t, flags.NewConflictingFlagsError("org-id", "free-user"), err)
			},
		},
		{
			name: "error - conflicting flags short form (-o and -u)",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				return creditsmocks.NewMockCreditsService(ctrl)
			},
			client: nil,
			args:   []string{"-o", "f47ac10b-58cc-4372-a567-0e02b2c3d479", "-u"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cannot use --org-id and --free-user flags together")
			},
		},
		{
			name: "error - invalid org ID format",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				return creditsmocks.NewMockCreditsService(ctrl)
			},
			client: nil,
			args:   []string{"--org-id", "invalid-uuid"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid uuid")
			},
		},
		{
			name: "error - too many arguments",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				return creditsmocks.NewMockCreditsService(ctrl)
			},
			client: nil,
			args:   []string{"extra-arg"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "accepts 0 arg(s), received 1")
			},
		},

		// Error cases - service errors
		{
			name: "error - user credits service returns error",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).Return(credits.UserCreditDetailsResult{}, cenclierrors.NewCencliError(errors.New("unauthorized")))
				return mockSvc
			},
			client: nil,
			args:   []string{"--free-user"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unauthorized")
			},
		},
		{
			name: "error - org credits service returns error",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).Return(credits.OrganizationCreditDetailsResult{}, cenclierrors.NewCencliError(errors.New("organization not found")))
				return mockSvc
			},
			client: nil,
			args:   []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "organization not found")
			},
		},

		// Edge cases
		{
			name: "success - zero balance user credits",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetUserCreditDetails(
					gomock.Any(),
				).Return(credits.UserCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 50*time.Millisecond, 1),
					Data: credits.UserCreditDetails{
						Balance: 0,
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"--free-user"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, `: 0`)
			},
		},
		{
			name: "success - zero balance org credits",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
				).Return(credits.OrganizationCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Data: credits.OrganizationCreditDetails{
						Balance:           0,
						CreditExpirations: []credits.CreditExpiration{},
						AutoReplenishConfig: credits.AutoReplenishConfig{
							Enabled: false,
						},
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, `: 0`)
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

			creditsSvc := tc.service(ctrl)
			opts := []command.ContextOpts{command.WithCreditsService(creditsSvc)}

			// Set up mock client if provided (for HasOrgID tests)
			if tc.client != nil {
				mockClient := tc.client(ctrl)
				opts = append(opts, func(c *command.Context) {
					c.SetCensysClient(mockClient)
				})
			}

			cmdContext := command.NewCommandContext(cfg, tc.store(ctrl), opts...)
			rootCmd, err := command.RootCommandToCobra(NewCreditsCommand(cmdContext))
			require.NoError(t, err)

			rootCmd.SetArgs(tc.args)
			execErr := rootCmd.Execute()
			tc.assert(t, stdout.String(), stderr.String(), cenclierrors.NewCencliError(execErr))
		})
	}
}
