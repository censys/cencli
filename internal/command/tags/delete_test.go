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
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/ui/form"
)

type deleteSeams struct {
	confirm    func(ctx context.Context, message string) (bool, error)
	stdinIsTTY func() bool
}

func runDeleteCommand(t *testing.T, svc apptags.Service, seams deleteSeams, args []string) (stdout, stderr string, err error) {
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
	cmd := NewDeleteCommand(cmdContext)
	if seams.confirm != nil {
		cmd.confirm = seams.confirm
	}
	if seams.stdinIsTTY != nil {
		cmd.stdinIsTTY = seams.stdinIsTTY
	}
	rootCmd, buildErr := command.RootCommandToCobra(cmd)
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

// deleteSuccessService returns a service that expects DeleteTag once and echoes
// the given identifier.
func deleteSuccessService(id string) func(ctrl *gomock.Controller) apptags.Service {
	return func(ctrl *gomock.Controller) apptags.Service {
		m := tagsmocks.NewMockTagsService(ctrl)
		m.EXPECT().DeleteTag(gomock.Any(), gomock.Any()).Return(
			apptags.DeleteResult{Meta: okMeta(), TagID: id},
			nil,
		)
		return m
	}
}

// deleteNoCallService returns a service that must not be called.
func deleteNoCallService(ctrl *gomock.Controller) apptags.Service {
	return tagsmocks.NewMockTagsService(ctrl)
}

func TestTagsDeleteCommand(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		isTTY         bool
		confirmResult bool
		confirmErr    error
		expectConfirm bool
		service       func(ctrl *gomock.Controller) apptags.Service
		assert        func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name:          "--yes skips the prompt and deletes",
			args:          []string{"alpha", "--yes"},
			isTTY:         true,
			expectConfirm: false,
			service:       deleteSuccessService("alpha"),
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "alpha")
				require.Contains(t, stdout, "deleted")
			},
		},
		{
			name:          "confirmation accepted deletes",
			args:          []string{"alpha"},
			isTTY:         true,
			confirmResult: true,
			expectConfirm: true,
			service:       deleteSuccessService("alpha"),
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "deleted")
			},
		},
		{
			name:          "confirmation declined aborts without deleting",
			args:          []string{"alpha"},
			isTTY:         true,
			confirmResult: false,
			expectConfirm: true,
			service:       deleteNoCallService,
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.NotContains(t, stdout, "deleted")
				require.Contains(t, stderr, "aborted")
			},
		},
		{
			name:          "non-interactive terminal without --yes errors",
			args:          []string{"alpha"},
			isTTY:         false,
			expectConfirm: false,
			service:       deleteNoCallService,
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "confirmation required")
			},
		},
		{
			name:          "user abort during prompt returns interrupted",
			args:          []string{"alpha"},
			isTTY:         true,
			confirmResult: false,
			confirmErr:    form.ErrUserAborted,
			expectConfirm: true,
			service:       deleteNoCallService,
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cancelled")
			},
		},
		{
			name:          "json output renders the deletion payload",
			args:          []string{"alpha", "--yes", "--output-format", "json"},
			isTTY:         true,
			expectConfirm: false,
			service:       deleteSuccessService("alpha"),
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"tag": "alpha"`)
				require.Contains(t, stdout, `"deleted": true`)
			},
		},
		{
			name:          "missing arg errors",
			args:          nil,
			isTTY:         true,
			expectConfirm: false,
			service:       deleteNoCallService,
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name:          "empty tag id errors before any prompt or service call",
			args:          []string{"   "},
			isTTY:         true,
			expectConfirm: false,
			service:       deleteNoCallService,
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "required")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			confirmCalled := false
			seams := deleteSeams{
				confirm: func(_ context.Context, _ string) (bool, error) {
					confirmCalled = true
					return tc.confirmResult, tc.confirmErr
				},
				stdinIsTTY: func() bool { return tc.isTTY },
			}

			stdout, stderr, err := runDeleteCommand(t, tc.service(ctrl), seams, tc.args)
			require.Equal(t, tc.expectConfirm, confirmCalled, "confirm invocation mismatch")
			tc.assert(t, stdout, stderr, err)
		})
	}
}
