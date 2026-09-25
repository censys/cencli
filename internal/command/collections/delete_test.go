package collections

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	collectionsmocks "github.com/censys/cencli/gen/app/collections/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	appcollections "github.com/censys/cencli/internal/app/collections"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/ui/form"
)

type deleteSeams struct {
	confirm    func(ctx context.Context, message string) (bool, error)
	stdinIsTTY func() bool
}

func runDeleteCommand(t *testing.T, svc appcollections.Service, seams deleteSeams, args []string) (stdout, stderr string, err error) {
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
	cmdContext := command.NewCommandContext(cfg, mockStore, command.WithCollectionsService(svc))
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

func deleteSuccessService(ctrl *gomock.Controller) appcollections.Service {
	m := collectionsmocks.NewMockCollectionsService(ctrl)
	m.EXPECT().DeleteCollection(gomock.Any(), gomock.Any()).
		Return(appcollections.DeleteResult{Meta: okMeta(), CollectionID: testCollectionID}, nil)
	return m
}

func deleteNoCallService(ctrl *gomock.Controller) appcollections.Service {
	return collectionsmocks.NewMockCollectionsService(ctrl)
}

func TestCollectionsDeleteCommand(t *testing.T) {
	isTTY := func() bool { return true }
	notTTY := func() bool { return false }
	yes := func(context.Context, string) (bool, error) { return true, nil }
	no := func(context.Context, string) (bool, error) { return false, nil }
	aborted := func(context.Context, string) (bool, error) { return false, form.ErrUserAborted }
	mustNotPrompt := func(t *testing.T) func(context.Context, string) (bool, error) {
		return func(context.Context, string) (bool, error) {
			t.Fatal("confirm must not be called")
			return false, nil
		}
	}

	testCases := []struct {
		name    string
		service func(ctrl *gomock.Controller) appcollections.Service
		args    []string
		seams   deleteSeams
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name:    "--yes skips the prompt",
			service: deleteSuccessService,
			args:    []string{testCollectionID, "--yes"},
			seams:   deleteSeams{confirm: mustNotPrompt(t), stdinIsTTY: notTTY},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "deleted")
			},
		},
		{
			name:    "confirmed prompt deletes",
			service: deleteSuccessService,
			args:    []string{testCollectionID},
			seams:   deleteSeams{confirm: yes, stdinIsTTY: isTTY},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "deleted")
			},
		},
		{
			name:    "declined confirmation sends no request",
			service: deleteNoCallService,
			args:    []string{testCollectionID},
			seams:   deleteSeams{confirm: no, stdinIsTTY: isTTY},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "Deletion aborted.")
			},
		},
		{
			name:    "aborted prompt is an interrupted error",
			service: deleteNoCallService,
			args:    []string{testCollectionID},
			seams:   deleteSeams{confirm: aborted, stdinIsTTY: isTTY},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name:    "non-interactive without --yes is refused",
			service: deleteNoCallService,
			args:    []string{testCollectionID},
			seams:   deleteSeams{confirm: mustNotPrompt(t), stdinIsTTY: notTTY},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "confirmation required")
			},
		},
		{
			name:    "invalid ID fails before the prompt",
			service: deleteNoCallService,
			args:    []string{"nope"},
			seams:   deleteSeams{confirm: mustNotPrompt(t), stdinIsTTY: isTTY},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "is not a valid UUID")
			},
		},
		{
			name:    "json output",
			service: deleteSuccessService,
			args:    []string{testCollectionID, "--yes", "--output-format", "json"},
			seams:   deleteSeams{stdinIsTTY: notTTY},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"collection": "`+testCollectionID+`"`)
				require.Contains(t, stdout, `"deleted": true`)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stdout, stderr, err := runDeleteCommand(t, tc.service(ctrl), tc.seams, tc.args)
			tc.assert(t, stdout, stderr, err)
		})
	}
}
