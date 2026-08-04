package tags

import (
	"bytes"
	"context"
	"testing"

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

func runOperationsCancelCommand(
	t *testing.T,
	svc apptags.Service,
	args []string,
) (stdout, stderr string, err error) {
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
	cmd := NewOperationsCancelCommand(cmdContext)
	rootCmd, buildErr := command.RootCommandToCobra(cmd)
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

// cancelled builds the service result for an accepted cancellation.
func cancelled(status string) apptags.CancelOperationResult {
	return apptags.CancelOperationResult{Meta: okMeta(), Operation: deleteOperation(status)}
}

// cancelNoCallService asserts the cancel never reaches the service, proving the
// input was rejected (or the prompt declined) at the command boundary.
func cancelNoCallService(ctrl *gomock.Controller) apptags.Service {
	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().CancelOperation(gomock.Any(), gomock.Any()).Times(0)
	return m
}

func TestTagsOperationsCancelCommand(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		service func(t *testing.T, ctrl *gomock.Controller) apptags.Service
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "cancels and renders the operation",
			args: []string{"my-tag", testOperationID},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CancelOperation(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.CancelOperationParams) (apptags.CancelOperationResult, cenclierrors.CencliError) {
						require.Equal(t, "my-tag", p.TagID.String())
						require.Equal(t, testOperationID, p.OperationID)
						return cancelled("cancelled"), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				// A cancelled status is the goal here, so it must not become an error.
				require.NoError(t, err)
				require.Contains(t, stdout, "Tag Operation")
				require.Contains(t, stdout, testOperationID)
				require.Contains(t, stdout, "cancelled")
			},
		},
		{
			// The API answers with the operation as it stood when the request was
			// accepted, so a job still winding down is a success, not a failure.
			name: "a still-running operation after cancelling still exits zero",
			args: []string{"my-tag", testOperationID},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CancelOperation(gomock.Any(), gomock.Any()).Return(cancelled("running"), nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "running")
			},
		},
		{
			// Unlike a --wait, reading back a failed job here is not this command's
			// verdict: the cancellation request itself succeeded.
			name: "a failed operation after cancelling still exits zero",
			args: []string{"my-tag", testOperationID},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CancelOperation(gomock.Any(), gomock.Any()).Return(cancelled("failed"), nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "failed")
			},
		},
		{
			// Cancelling only stops further processing, so it runs unprompted -
			// including with nothing on stdin, where a prompt could not be answered.
			name: "cancels without prompting",
			args: []string{"my-tag", testOperationID},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CancelOperation(gomock.Any(), gomock.Any()).Return(cancelled("cancelled"), nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "cancelled")
			},
		},
		{
			// --yes belongs to the flows that still confirm; keeping a dead flag
			// here would imply this one prompts.
			name: "--yes is not a flag on cancel",
			args: []string{"my-tag", testOperationID, "--yes"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return cancelNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unknown flag")
			},
		},
		{
			name: "non-UUID operation ID is rejected before the service",
			args: []string{"my-tag", "not-a-uuid"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return cancelNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "not-a-uuid")
			},
		},
		{
			name: "empty tag is rejected before the service",
			args: []string{"  ", testOperationID},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return cancelNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "tag name or ID is required")
			},
		},
		{
			name: "missing the operation argument is rejected",
			args: []string{"my-tag"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				return cancelNoCallService(ctrl)
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "org id is threaded to the service",
			args: []string{
				"my-tag", testOperationID,
				"--org-id", "11111111-1111-1111-1111-111111111111",
			},
			service: func(t *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CancelOperation(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.CancelOperationParams) (apptags.CancelOperationResult, cenclierrors.CencliError) {
						require.True(t, p.OrgID.IsPresent())
						return cancelled("cancelled"), nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "json output renders the operation payload",
			args: []string{"my-tag", testOperationID, "--output-format", "json"},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CancelOperation(gomock.Any(), gomock.Any()).Return(cancelled("cancelled"), nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"status": "cancelled"`)
				require.Contains(t, stdout, `"type": "bulk_delete"`)
			},
		},
		{
			// A finished job cannot be cancelled; that error must surface.
			name: "service error is returned",
			args: []string{"my-tag", testOperationID},
			service: func(_ *testing.T, ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CancelOperation(gomock.Any(), gomock.Any()).Return(
					apptags.CancelOperationResult{}, cenclierrors.NewCencliError(errBoom))
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			stdout, stderr, err := runOperationsCancelCommand(t, tc.service(t, ctrl), tc.args)
			tc.assert(t, stdout, stderr, err)
		})
	}
}
