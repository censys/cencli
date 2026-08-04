package tags

import (
	"bytes"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	tagsmocks "github.com/censys/cencli/gen/app/tags/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	apptags "github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
)

const testOperationID = "d421a231-eb5e-4927-a0be-8aa749eb731c"

func runOperationsGetCommand(t *testing.T, svc apptags.Service, args []string) (stdout, stderr string, err error) {
	t.Helper()

	tempDir := t.TempDir()
	viper.Reset()
	cfg, cfgErr := config.New(tempDir)
	require.NoError(t, cfgErr)

	var outBuf, errBuf bytes.Buffer
	formatter.Stdout = &outBuf
	formatter.Stderr = &errBuf

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := storemocks.NewMockStore(ctrl)
	cmdContext := command.NewCommandContext(cfg, mockStore, command.WithTagsService(svc))
	rootCmd, buildErr := command.RootCommandToCobra(NewOperationsGetCommand(cmdContext))
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

func finishedOperation(status string) apptags.TagOperation {
	ended := time.Unix(60, 0).UTC()
	op := operation(testOperationID, status)
	op.EndedAt = &ended
	op.ProcessedCount = 100
	op.SuccessfulCount = 90
	return op
}

// getOnly expects a plain read and no polling.
func getOnly(ctrl *gomock.Controller, op apptags.TagOperation) apptags.Service {
	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Times(0)
	m.EXPECT().GetOperation(gomock.Any(), gomock.Any()).Return(
		apptags.GetOperationResult{Meta: okMeta(), Operation: op}, nil)
	return m
}

// waitOnly expects polling and no plain read.
func waitOnly(ctrl *gomock.Controller, op apptags.TagOperation) apptags.Service {
	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().GetOperation(gomock.Any(), gomock.Any()).Times(0)
	m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Return(
		apptags.GetOperationResult{Meta: okMeta(), Operation: op}, nil)
	return m
}

func TestTagsOperationsGetCommand(t *testing.T) {
	testCases := []struct {
		name    string
		service func(ctrl *gomock.Controller) apptags.Service
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "success - short detail view",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return getOnly(ctrl, operation(testOperationID, "running"))
			},
			args: []string{"my-tag", testOperationID},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Tag Operation")
				require.Contains(t, stdout, testOperationID)
				require.Contains(t, stdout, "running")
				require.Contains(t, stdout, "40/100")
			},
		},
		{
			name: "json output renders the operation",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return getOnly(ctrl, operation(testOperationID, "succeeded"))
			},
			args: []string{"my-tag", testOperationID, "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"status": "succeeded"`)
			},
		},
		{
			name: "without --wait a failed operation is still a successful read",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return getOnly(ctrl, finishedOperation("failed"))
			},
			args: []string{"my-tag", testOperationID},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				// Reading the status is not the same as suffering the failure.
				require.NoError(t, err)
				require.Contains(t, stdout, "failed")
			},
		},
		{
			name: "--wait on a succeeded operation exits cleanly",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return waitOnly(ctrl, finishedOperation("succeeded"))
			},
			args: []string{"my-tag", testOperationID, "--wait"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "succeeded")
			},
		},
		{
			name: "--wait on a failed operation renders it then errors",
			service: func(ctrl *gomock.Controller) apptags.Service {
				op := finishedOperation("failed")
				msg := "quota exhausted"
				op.ErrorMessage = &msg
				return waitOnly(ctrl, op)
			},
			args: []string{"my-tag", testOperationID, "--wait"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				// The payload is still rendered; only the exit code carries the outcome.
				require.Contains(t, stdout, testOperationID)
				require.Contains(t, err.Error(), "quota exhausted")
			},
		},
		{
			name: "--wait on a cancelled operation errors",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return waitOnly(ctrl, finishedOperation("cancelled"))
			},
			args: []string{"my-tag", testOperationID, "--wait"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cancelled")
			},
		},
		{
			name: "--wait on a limit_reached operation warns but exits cleanly",
			service: func(ctrl *gomock.Controller) apptags.Service {
				op := finishedOperation("limit_reached")
				msg := "plan asset limit reached"
				op.StatusMessage = &msg
				return waitOnly(ctrl, op)
			},
			args: []string{"my-tag", testOperationID, "--wait"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				// A capped run still did real work, so it is not a failure.
				require.NoError(t, err)
				require.Contains(t, stderr, "asset limit")
				require.Contains(t, stderr, "plan asset limit reached")
			},
		},
		{
			name: "interrupted wait explains the job keeps running",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Return(
					apptags.GetOperationResult{}, cenclierrors.NewInterruptedError())
				return m
			},
			args: []string{"my-tag", testOperationID, "--wait"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, stderr, "continues server-side")
				require.Contains(t, stderr, "Track with: censys tags operations get my-tag "+testOperationID)
			},
		},
		{
			name: "--timeout is passed through when waiting",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ any, params apptags.WaitParams) (apptags.GetOperationResult, cenclierrors.CencliError) {
						require.True(t, params.Timeout.IsPresent())
						require.Equal(t, 5*time.Minute, params.Timeout.MustGet())
						require.Equal(t, testOperationID, params.OperationID)
						return apptags.GetOperationResult{Meta: okMeta(), Operation: finishedOperation("succeeded")}, nil
					})
				return m
			},
			args: []string{"my-tag", testOperationID, "--wait", "--timeout", "5m"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "--timeout 0 waits without a limit",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ any, params apptags.WaitParams) (apptags.GetOperationResult, cenclierrors.CencliError) {
						// Zero means unbounded, matching the global --timeout-http.
						require.False(t, params.Timeout.IsPresent())
						return apptags.GetOperationResult{Meta: okMeta(), Operation: finishedOperation("succeeded")}, nil
					})
				return m
			},
			args: []string{"my-tag", testOperationID, "--wait", "--timeout", "0"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "a negative --timeout is rejected before the service",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().GetOperation(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			args: []string{"my-tag", testOperationID, "--wait", "--timeout", "-5m"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--timeout must not be negative")
			},
		},
		{
			name: "--timeout without --wait is rejected before the service",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().GetOperation(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().WaitForOperation(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			args: []string{"my-tag", testOperationID, "--timeout", "5m"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--timeout only applies while polling")
			},
		},
		{
			name: "non-UUID operation ID is rejected before the service",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().GetOperation(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			args: []string{"my-tag", "not-a-uuid"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "not-a-uuid")
			},
		},
		{
			name: "empty tag is rejected before the service",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().GetOperation(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			args: []string{"  ", testOperationID},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "tag name or ID is required")
			},
		},
		{
			name: "missing the operation argument is rejected",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().GetOperation(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			args: []string{"my-tag"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "service error is returned",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().GetOperation(gomock.Any(), gomock.Any()).Return(
					apptags.GetOperationResult{}, cenclierrors.NewCencliError(errBoom))
				return m
			},
			args: []string{"my-tag", testOperationID},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			stdout, stderr, err := runOperationsGetCommand(t, tc.service(ctrl), tc.args)
			tc.assert(t, stdout, stderr, err)
		})
	}
}
