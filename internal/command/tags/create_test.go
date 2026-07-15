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

func runCreateCommand(t *testing.T, svc apptags.Service, args []string) (stdout, stderr string, err error) {
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
	rootCmd, buildErr := command.RootCommandToCobra(NewCreateCommand(cmdContext))
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

func TestTagsCreateCommand(t *testing.T) {
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
				m.EXPECT().CreateTag(gomock.Any(), gomock.Any()).Return(
					apptags.CreateResult{Meta: okMeta(), Tag: tag("alpha")},
					nil,
				)
				return m
			},
			args: []string{"alpha"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "alpha")
				require.Contains(t, stdout, "Name:")
				require.Contains(t, stdout, "Tag Created")
			},
		},
		{
			name: "success - json output",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CreateTag(gomock.Any(), gomock.Any()).Return(
					apptags.CreateResult{Meta: okMeta(), Tag: tag("alpha")},
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
			name: "default privacy is private; name threaded to service",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CreateTag(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params apptags.CreateParams) (apptags.CreateResult, cenclierrors.CencliError) {
						require.Equal(t, "my-tag", params.Name)
						require.Equal(t, "private", params.Privacy)
						require.True(t, params.Description.IsAbsent())
						return apptags.CreateResult{Meta: okMeta(), Tag: tag("my-tag")}, nil
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
			name: "privacy and description flags threaded to service",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CreateTag(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params apptags.CreateParams) (apptags.CreateResult, cenclierrors.CencliError) {
						require.Equal(t, "my-tag", params.Name)
						require.Equal(t, "shared", params.Privacy)
						require.True(t, params.Description.IsPresent())
						require.Equal(t, "some notes", params.Description.MustGet())
						return apptags.CreateResult{Meta: okMeta(), Tag: tag("my-tag")}, nil
					},
				)
				return m
			},
			args: []string{"my-tag", "--privacy", "shared", "--description", "some notes"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "my-tag")
			},
		},
		{
			name: "name is trimmed before reaching the service",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CreateTag(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params apptags.CreateParams) (apptags.CreateResult, cenclierrors.CencliError) {
						require.Equal(t, "my-tag", params.Name)
						return apptags.CreateResult{Meta: okMeta(), Tag: tag("my-tag")}, nil
					},
				)
				return m
			},
			args: []string{"  my-tag  "},
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
			name: "error - service failure surfaced",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().CreateTag(gomock.Any(), gomock.Any()).Return(
					apptags.CreateResult{},
					cenclierrors.NewCencliError(errors.New("tag already exists")),
				)
				return m
			},
			args: []string{"dupe"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "tag already exists")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			stdout, stderr, err := runCreateCommand(t, tc.service(ctrl), tc.args)
			tc.assert(t, stdout, stderr, err)
		})
	}
}
