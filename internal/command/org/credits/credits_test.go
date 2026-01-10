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
	appcredits "github.com/censys/cencli/internal/app/credits"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
)

func TestOrgCreditsCommand(t *testing.T) {
	testCases := []struct {
		name    string
		store   func(ctrl *gomock.Controller) store.Store
		service func(ctrl *gomock.Controller) appcredits.Service
		client  func(ctrl *gomock.Controller) *clientmocks.MockClient
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		// Success cases
		{
			name: "success - org credits with --org-id flag",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) appcredits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")),
				).Return(appcredits.OrganizationCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Data: appcredits.OrganizationCreditDetails{
						Balance:           5000,
						CreditExpirations: []appcredits.CreditExpiration{},
						AutoReplenishConfig: appcredits.AutoReplenishConfig{
							Enabled: false,
						},
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "5000")
			},
		},
		{
			name: "success - org credits with stored org ID",
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
			service: func(ctrl *gomock.Controller) appcredits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					identifiers.NewOrganizationID(uuid.MustParse("58857aac-4b76-46ec-a567-0e02b2c3d479")),
				).Return(appcredits.OrganizationCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 150*time.Millisecond, 1),
					Data: appcredits.OrganizationCreditDetails{
						Balance:           7500,
						CreditExpirations: []appcredits.CreditExpiration{},
						AutoReplenishConfig: appcredits.AutoReplenishConfig{
							Enabled: false,
						},
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"--output-format", "json"},
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
			service: func(ctrl *gomock.Controller) appcredits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")),
				).Return(appcredits.OrganizationCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Data: appcredits.OrganizationCreditDetails{
						Balance: 3000,
						CreditExpirations: []appcredits.CreditExpiration{
							{
								Balance:        1500,
								CreationDate:   mo.Some(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
								ExpirationDate: mo.Some(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
							},
						},
						AutoReplenishConfig: appcredits.AutoReplenishConfig{
							Enabled: false,
						},
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479", "--output-format", "json"},
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
			service: func(ctrl *gomock.Controller) appcredits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")),
				).Return(appcredits.OrganizationCreditDetailsResult{
					Meta: responsemeta.NewResponseMeta(&http.Request{}, &http.Response{StatusCode: 200}, 100*time.Millisecond, 1),
					Data: appcredits.OrganizationCreditDetails{
						Balance:           2000,
						CreditExpirations: []appcredits.CreditExpiration{},
						AutoReplenishConfig: appcredits.AutoReplenishConfig{
							Enabled:   true,
							Threshold: mo.Some(int64(100)),
							Amount:    mo.Some(int64(500)),
						},
					},
				}, nil)
				return mockSvc
			},
			client: nil,
			args:   []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "2000")
				require.Contains(t, stdout, `"auto_replenish_config"`)
				require.Contains(t, stdout, `"enabled": true`)
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
			service: func(ctrl *gomock.Controller) appcredits.Service {
				return creditsmocks.NewMockCreditsService(ctrl)
			},
			client: nil,
			args:   []string{"--output-format", "json"},
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
			service: func(ctrl *gomock.Controller) appcredits.Service {
				return creditsmocks.NewMockCreditsService(ctrl)
			},
			client: nil,
			args:   []string{"--org-id", "invalid-uuid", "--output-format", "json"},
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
			service: func(ctrl *gomock.Controller) appcredits.Service {
				mockSvc := creditsmocks.NewMockCreditsService(ctrl)
				mockSvc.EXPECT().GetOrganizationCreditDetails(
					gomock.Any(),
					identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")),
				).Return(appcredits.OrganizationCreditDetailsResult{}, cenclierrors.NewCencliError(errors.New("organization not found")))
				return mockSvc
			},
			client: nil,
			args:   []string{"--org-id", "f47ac10b-58cc-4372-a567-0e02b2c3d479"},
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
			service: func(ctrl *gomock.Controller) appcredits.Service {
				return creditsmocks.NewMockCreditsService(ctrl)
			},
			client: nil,
			args:   []string{"extra-arg", "--output-format", "json"},
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

			creditsSvc := tc.service(ctrl)
			opts := []command.ContextOpts{command.WithCreditsService(creditsSvc)}

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
