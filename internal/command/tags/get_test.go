package tags

import (
	"bytes"
	"context"
	"errors"
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

func runGetCommand(t *testing.T, svc apptags.Service, args []string) (stdout, stderr string, err error) {
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
	rootCmd, buildErr := command.RootCommandToCobra(NewGetCommand(cmdContext))
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

func TestTagsGetCommand(t *testing.T) {
	testCases := []struct {
		name    string
		service func(ctrl *gomock.Controller) apptags.Service
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "success - short output",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().GetTag(gomock.Any(), gomock.Any()).Return(
					apptags.GetResult{Meta: okMeta(), Tag: tag("alpha")},
					nil,
				)
				return m
			},
			args: []string{"alpha"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "alpha")
				require.Contains(t, stdout, "Name:")
				require.Contains(t, stdout, "Privacy:")
			},
		},
		{
			name: "success - json output",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().GetTag(gomock.Any(), gomock.Any()).Return(
					apptags.GetResult{Meta: okMeta(), Tag: tag("alpha")},
					nil,
				)
				return m
			},
			args: []string{"alpha", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"name": "alpha"`)
			},
		},
		{
			name: "tag identifier threaded to service",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().GetTag(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params apptags.GetParams) (apptags.GetResult, cenclierrors.CencliError) {
						require.Equal(t, "my-tag", params.TagID.String())
						require.True(t, params.TagID.UID().IsAbsent())
						return apptags.GetResult{Meta: okMeta(), Tag: tag("my-tag")}, nil
					},
				)
				return m
			},
			args: []string{"my-tag"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "my-tag")
			},
		},
		{
			name: "error - missing arg",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return tagsmocks.NewMockTagsService(ctrl) // not called
			},
			args: nil,
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "error - too many args",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return tagsmocks.NewMockTagsService(ctrl) // not called
			},
			args: []string{"a", "b"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "error - empty tag id",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return tagsmocks.NewMockTagsService(ctrl) // not called
			},
			args: []string{"   "},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "required")
			},
		},
		{
			name: "error - service failure surfaced",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().GetTag(gomock.Any(), gomock.Any()).Return(
					apptags.GetResult{},
					cenclierrors.NewCencliError(errors.New("tag not found")),
				)
				return m
			},
			args: []string{"missing"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "tag not found")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			stdout, stderr, err := runGetCommand(t, tc.service(ctrl), tc.args)
			tc.assert(t, stdout, stderr, err)
		})
	}
}

func TestTagsGetCommand_AssetCount(t *testing.T) {
	t.Run("count is rendered without asking for it", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		count := int64(7)
		counted := tag("alpha")
		counted.AssetCount = &count

		m := tagsmocks.NewMockTagsService(ctrl)
		m.EXPECT().GetTag(gomock.Any(), gomock.Any()).Return(
			apptags.GetResult{Meta: okMeta(), Tag: counted}, nil)

		stdout, _, err := runGetCommand(t, m, []string{"alpha"})
		require.NoError(t, err)
		require.Contains(t, stdout, "Assets:")
		require.Contains(t, stdout, "7")
	})

	t.Run("the count is serialized in json", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		count := int64(0)
		counted := tag("alpha")
		counted.AssetCount = &count

		m := tagsmocks.NewMockTagsService(ctrl)
		m.EXPECT().GetTag(gomock.Any(), gomock.Any()).Return(
			apptags.GetResult{Meta: okMeta(), Tag: counted}, nil)

		// A zero count must survive omitempty as an explicit 0, or a script
		// cannot tell "no assets" from "not counted".
		stdout, _, err := runGetCommand(t, m, []string{"alpha", "--output-format", "json"})
		require.NoError(t, err)
		require.Contains(t, stdout, `"asset_count": 0`)
	})

	t.Run("--asset-count is no longer a flag", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := tagsmocks.NewMockTagsService(ctrl)
		m.EXPECT().GetTag(gomock.Any(), gomock.Any()).Times(0)

		_, _, err := runGetCommand(t, m, []string{"alpha", "--asset-count"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown flag")
	})

	t.Run("count failure still renders the tag and reports the error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := tagsmocks.NewMockTagsService(ctrl)
		m.EXPECT().GetTag(gomock.Any(), gomock.Any()).Return(
			apptags.GetResult{
				Meta:         okMeta(),
				Tag:          tag("alpha"),
				PartialError: cenclierrors.NewCencliError(errors.New("permission denied")),
			}, nil)

		// A failed count must not fail the command: the tag is still on stdout
		// and the exit code stays 0.
		stdout, stderr, err := runGetCommand(t, m, []string{"alpha"})
		require.NoError(t, err)
		require.Contains(t, stdout, "alpha")
		require.NotContains(t, stdout, "Assets:")
		require.Contains(t, stderr, "permission denied")
	})
}
