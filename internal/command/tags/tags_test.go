package tags

import (
	"bytes"
	"context"
	"errors"
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
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
)

func okMeta() *responsemeta.ResponseMeta {
	return &responsemeta.ResponseMeta{
		Method:  "GET",
		URL:     "https://api.censys.io/v3/tags",
		Status:  200,
		Latency: 100 * time.Millisecond,
	}
}

func tag(name string) apptags.Tag {
	return apptags.Tag{ID: name + "-id", Name: name, Privacy: "shared", CreatedBy: "creator", CreatedAt: time.Unix(0, 0).UTC()}
}

func runListCommand(t *testing.T, svc apptags.Service, args []string) (stdout, stderr string, err error) {
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
	rootCmd, buildErr := command.RootCommandToCobra(NewListCommand(cmdContext))
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

func TestTagsListCommand(t *testing.T) {
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
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Return(
					apptags.ListResult{Meta: okMeta(), Tags: []apptags.Tag{tag("alpha"), tag("beta")}, TotalSize: 2},
					nil,
				)
				return m
			},
			args: nil,
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "alpha")
				require.Contains(t, stdout, "beta")
				require.Contains(t, stdout, "Tags (2)")
			},
		},
		{
			name: "success - json output",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Return(
					apptags.ListResult{Meta: okMeta(), Tags: []apptags.Tag{tag("alpha")}, TotalSize: 1},
					nil,
				)
				return m
			},
			args: []string{"--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"name": "alpha"`)
			},
		},
		{
			name: "short output shows total when truncated by pagination",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Return(
					apptags.ListResult{Meta: okMeta(), Tags: []apptags.Tag{tag("alpha"), tag("beta")}, TotalSize: 42},
					nil,
				)
				return m
			},
			args: nil,
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Tags (2 of 42)")
			},
		},
		{
			name: "success - empty result",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Return(apptags.ListResult{Meta: okMeta()}, nil)
				return m
			},
			args: nil,
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "No tags found")
			},
		},
		{
			name: "filters and pagination threaded to service",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params apptags.ListParams) (apptags.ListResult, cenclierrors.CencliError) {
						require.Equal(t, "shared", params.Privacy.MustGet())
						require.Equal(t, "my-tag", params.Name.MustGet())
						require.Equal(t, "f47ac10b-58cc-4372-a567-0e02b2c3d479", params.CreatedBy.MustGet())
						require.Equal(t, "name_desc", params.OrderBy.MustGet())
						require.Equal(t, uint64(50), params.PageSize.MustGet())
						require.Equal(t, uint64(3), params.MaxPages.MustGet())
						return apptags.ListResult{Meta: okMeta(), Tags: []apptags.Tag{tag("my-tag")}, TotalSize: 1}, nil
					},
				)
				return m
			},
			args: []string{
				"--privacy", "shared",
				"--name", "my-tag",
				"--created-by", "f47ac10b-58cc-4372-a567-0e02b2c3d479",
				"--order-by", "name_desc",
				"--page-size", "50",
				"--max-pages", "3",
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "my-tag")
			},
		},
		{
			name: "error - unexpected positional arg",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return tagsmocks.NewMockTagsService(ctrl) // not called
			},
			args: []string{"unexpected"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "error - invalid max-pages",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return tagsmocks.NewMockTagsService(ctrl) // not called
			},
			args: []string{"--max-pages", "0"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "max-pages")
			},
		},
		{
			// The API declares created_by as a UUID and 422s on anything else, so
			// it is rejected here instead of costing a round trip.
			name: "error - non-UUID created-by",
			service: func(ctrl *gomock.Controller) apptags.Service {
				return tagsmocks.NewMockTagsService(ctrl) // not called
			},
			args: []string{"--created-by", "not-a-uuid"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid uuid")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			stdout, stderr, err := runListCommand(t, tc.service(ctrl), tc.args)
			tc.assert(t, stdout, stderr, err)
		})
	}
}

func TestTagsListCommand_PartialError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := tagsmocks.NewMockTagsService(ctrl)
	baseErr := cenclierrors.NewCencliError(errors.New("network error on page 2"))
	m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Return(
		apptags.ListResult{
			Meta:         okMeta(),
			Tags:         []apptags.Tag{tag("alpha")},
			TotalSize:    5,
			PartialError: cenclierrors.ToPartialError(baseErr),
		},
		nil,
	)

	stdout, stderr, err := runListCommand(t, m, nil)
	require.NoError(t, err)
	require.Contains(t, stdout, "alpha", "should render partial data to stdout")
	require.Contains(t, stderr, "network error on page 2", "should print the partial error to stderr")
}
