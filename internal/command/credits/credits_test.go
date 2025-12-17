package credits

import (
	"bytes"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	creditsmocks "github.com/censys/cencli/gen/app/credits/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/app/credits"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
)

func TestCreditsCommand(t *testing.T) {
	testCases := []struct {
		name    string
		store   func(ctrl *gomock.Controller) store.Store
		service func(ctrl *gomock.Controller) credits.Service
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		// Success cases - free user credits
		{
			name: "success - default free user credits",
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
			args: []string{"--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "500")
			},
		},
		{
			name: "success - free user credits with balance and reset date",
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
						Balance:  1000,
						ResetsAt: mo.Some(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)),
					},
				}, nil)
				return mockSvc
			},
			args: []string{"--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"balance"`)
				require.Contains(t, stdout, "1000")
			},
		},

		// Error cases
		{
			name: "error - too many arguments",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) credits.Service {
				return creditsmocks.NewMockCreditsService(ctrl)
			},
			args: []string{"extra-arg", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "accepts 0 arg(s), received 1")
			},
		},
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
			args: []string{},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unauthorized")
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
			args: []string{"--output-format", "json"},
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

			cmdContext := command.NewCommandContext(cfg, tc.store(ctrl), opts...)
			rootCmd, err := command.RootCommandToCobra(NewCreditsCommand(cmdContext))
			require.NoError(t, err)

			rootCmd.SetArgs(tc.args)
			execErr := rootCmd.Execute()
			tc.assert(t, stdout.String(), stderr.String(), cenclierrors.NewCencliError(execErr))
		})
	}
}
