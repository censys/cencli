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

func runUpdateCommand(t *testing.T, svc apptags.Service, args []string) (stdout, stderr string, err error) {
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
	rootCmd, buildErr := command.RootCommandToCobra(NewUpdateCommand(cmdContext))
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

func TestTagsUpdateCommand(t *testing.T) {
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
				m.EXPECT().UpdateTag(gomock.Any(), gomock.Any()).Return(
					apptags.UpdateResult{Meta: okMeta(), Tag: tag("alpha")},
					nil,
				)
				return m
			},
			args: []string{"alpha", "--privacy", "shared"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "alpha")
				require.Contains(t, stdout, "Name:")
				require.Contains(t, stdout, "Tag Updated")
			},
		},
		{
			name: "success - json output",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().UpdateTag(gomock.Any(), gomock.Any()).Return(
					apptags.UpdateResult{Meta: okMeta(), Tag: tag("alpha")},
					nil,
				)
				return m
			},
			args: []string{"alpha", "--name", "renamed", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"name": "alpha"`)
			},
		},
		{
			name: "mutation flags threaded to service; tag id passed raw",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().UpdateTag(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params apptags.UpdateParams) (apptags.UpdateResult, cenclierrors.CencliError) {
						require.Equal(t, "my-tag", params.TagID.String())
						require.Equal(t, "renamed", params.Name.MustGet())
						require.Equal(t, "shared", params.Privacy.MustGet())
						require.Equal(t, "some notes", params.Description.MustGet())
						return apptags.UpdateResult{Meta: okMeta(), Tag: tag("renamed")}, nil
					},
				)
				return m
			},
			args: []string{"my-tag", "--name", "renamed", "--privacy", "shared", "--description", "some notes"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "renamed")
			},
		},
		{
			name: "clear-description sends an empty description",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().UpdateTag(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params apptags.UpdateParams) (apptags.UpdateResult, cenclierrors.CencliError) {
						require.True(t, params.Description.IsPresent())
						require.Equal(t, "", params.Description.MustGet())
						require.True(t, params.Name.IsAbsent())
						require.True(t, params.Privacy.IsAbsent())
						return apptags.UpdateResult{Meta: okMeta(), Tag: tag("my-tag")}, nil
					},
				)
				return m
			},
			args: []string{"my-tag", "--clear-description"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "my-tag")
			},
		},
		{
			name: "error - nothing to update",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return tagsmocks.NewMockTagsService(ctrl) // not called
			},
			args: []string{"my-tag"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no fields to update")
			},
		},
		{
			name: "error - description conflict",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return tagsmocks.NewMockTagsService(ctrl) // not called
			},
			args: []string{"my-tag", "--description", "foo", "--clear-description"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cannot be used together")
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
			name: "error - service failure surfaced",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().UpdateTag(gomock.Any(), gomock.Any()).Return(
					apptags.UpdateResult{},
					cenclierrors.NewCencliError(errors.New("tag not found")),
				)
				return m
			},
			args: []string{"missing", "--privacy", "shared"},
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
			stdout, stderr, err := runUpdateCommand(t, tc.service(ctrl), tc.args)
			tc.assert(t, stdout, stderr, err)
		})
	}
}
