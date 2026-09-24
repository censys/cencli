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

	t.Run("--yes skips the prompt", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		stdout, _, err := runDeleteCommand(t, deleteSuccessService(ctrl),
			deleteSeams{confirm: mustNotPrompt(t), stdinIsTTY: notTTY}, []string{testCollectionID, "--yes"})
		require.NoError(t, err)
		require.Contains(t, stdout, "deleted")
	})

	t.Run("confirmed prompt deletes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		stdout, _, err := runDeleteCommand(t, deleteSuccessService(ctrl),
			deleteSeams{confirm: yes, stdinIsTTY: isTTY}, []string{testCollectionID})
		require.NoError(t, err)
		require.Contains(t, stdout, "deleted")
	})

	t.Run("declined confirmation sends no request", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		_, stderr, err := runDeleteCommand(t, collectionsmocks.NewMockCollectionsService(ctrl),
			deleteSeams{confirm: no, stdinIsTTY: isTTY}, []string{testCollectionID})
		require.NoError(t, err)
		require.Contains(t, stderr, "Deletion aborted.")
	})

	t.Run("aborted prompt is an interrupted error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		_, _, err := runDeleteCommand(t, collectionsmocks.NewMockCollectionsService(ctrl),
			deleteSeams{confirm: aborted, stdinIsTTY: isTTY}, []string{testCollectionID})
		require.Error(t, err)
	})

	t.Run("non-interactive without --yes is refused", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		_, _, err := runDeleteCommand(t, collectionsmocks.NewMockCollectionsService(ctrl),
			deleteSeams{confirm: mustNotPrompt(t), stdinIsTTY: notTTY}, []string{testCollectionID})
		require.Error(t, err)
		require.Contains(t, err.Error(), "confirmation required")
	})

	t.Run("invalid ID fails before the prompt", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		_, _, err := runDeleteCommand(t, collectionsmocks.NewMockCollectionsService(ctrl),
			deleteSeams{confirm: mustNotPrompt(t), stdinIsTTY: isTTY}, []string{"nope"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid collection ID")
	})

	t.Run("json output", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		stdout, _, err := runDeleteCommand(t, deleteSuccessService(ctrl),
			deleteSeams{stdinIsTTY: notTTY}, []string{testCollectionID, "--yes", "--output-format", "json"})
		require.NoError(t, err)
		require.Contains(t, stdout, `"collection": "`+testCollectionID+`"`)
		require.Contains(t, stdout, `"deleted": true`)
	})
}
