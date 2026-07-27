package tags

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/samber/mo"
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

func runAssignmentsCommand(t *testing.T, svc apptags.Service, args []string) (stdout, stderr string, err error) {
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
	rootCmd, buildErr := command.RootCommandToCobra(NewAssignmentsCommand(cmdContext))
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

func assignment(assetID string) apptags.Assignment {
	return apptags.Assignment{
		ID:          assetID + "-assignment",
		TagID:       "tag-id",
		AssetID:     assetID,
		AssetType:   "host",
		PlatformRef: "https://platform.censys.io/hosts/" + assetID,
		CreatedBy:   "creator",
		CreatedAt:   time.Unix(0, 0).UTC(),
	}
}

// assignmentsNoCallService asserts the service is never reached, proving a bad
// input was rejected at the command boundary.
func assignmentsNoCallService(ctrl *gomock.Controller) apptags.Service {
	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().ListAssignments(gomock.Any(), gomock.Any()).Times(0)
	return m
}

func TestTagsAssignmentsCommand(t *testing.T) {
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
				m.EXPECT().ListAssignments(gomock.Any(), gomock.Any()).Return(
					apptags.AssignmentsResult{
						Meta:        okMeta(),
						Assignments: []apptags.Assignment{assignment("8.8.8.8"), assignment("1.1.1.1")},
						TotalSize:   2,
					}, nil)
				return m
			},
			args: []string{"my-tag"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Assignments (2)")
				require.Contains(t, stdout, "8.8.8.8")
				require.Contains(t, stdout, "host")
			},
		},
		{
			name: "short output reports the API total when truncated",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListAssignments(gomock.Any(), gomock.Any()).Return(
					apptags.AssignmentsResult{
						Meta:        okMeta(),
						Assignments: []apptags.Assignment{assignment("8.8.8.8")},
						TotalSize:   9,
					}, nil)
				return m
			},
			args: []string{"my-tag"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Assignments (1 of 9)")
			},
		},
		{
			name: "empty result renders a friendly message",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListAssignments(gomock.Any(), gomock.Any()).Return(
					apptags.AssignmentsResult{Meta: okMeta()}, nil)
				return m
			},
			args: []string{"my-tag"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "No assignments found.")
			},
		},
		{
			name: "json output",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListAssignments(gomock.Any(), gomock.Any()).Return(
					apptags.AssignmentsResult{
						Meta:        okMeta(),
						Assignments: []apptags.Assignment{assignment("8.8.8.8")},
						TotalSize:   1,
					}, nil)
				return m
			},
			args: []string{"my-tag", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"asset_id"`)
				require.Contains(t, stdout, `"platform_ref"`)
			},
		},
		{
			name: "filters and pagination parsed into params",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListAssignments(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.AssignmentsParams) (apptags.AssignmentsResult, cenclierrors.CencliError) {
						require.Equal(t, "my-tag", p.TagID.String())
						require.Equal(t, mo.Some("8.8.8.8"), p.AssetID)
						require.Equal(t, mo.Some("host"), p.AssetType)
						require.Equal(t, mo.Some("creator-id"), p.CreatedBy)
						require.Equal(t, mo.Some("create_time_asc"), p.OrderBy)
						require.Equal(t, mo.Some(uint64(25)), p.PageSize)
						require.Equal(t, mo.Some(uint64(3)), p.MaxPages)
						require.True(t, p.CreatedAfter.IsPresent())
						require.Equal(t, 2025, p.CreatedAfter.MustGet().Year())
						return apptags.AssignmentsResult{Meta: okMeta()}, nil
					})
				return m
			},
			args: []string{
				"my-tag",
				"--asset", "8.8.8.8",
				"--asset-type", "host",
				"--created-by", "creator-id",
				"--created-after", "2025-01-01T00:00:00Z",
				"--order-by", "create_time_asc",
				"--page-size", "25",
				"--max-pages", "3",
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "all pages requested - max-pages left absent and usage warned",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListAssignments(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, p apptags.AssignmentsParams) (apptags.AssignmentsResult, cenclierrors.CencliError) {
						require.False(t, p.MaxPages.IsPresent())
						return apptags.AssignmentsResult{Meta: okMeta()}, nil
					})
				return m
			},
			args: []string{"my-tag", "--max-pages", "-1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "fetching all pages")
			},
		},
		{
			name: "partial error printed after the data",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListAssignments(gomock.Any(), gomock.Any()).Return(
					apptags.AssignmentsResult{
						Meta:         okMeta(),
						Assignments:  []apptags.Assignment{assignment("8.8.8.8")},
						TotalSize:    1,
						PartialError: cenclierrors.NewCencliError(context.Canceled),
					}, nil)
				return m
			},
			args: []string{"my-tag"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "8.8.8.8")
				require.NotEmpty(t, stderr)
			},
		},
		{
			name:    "empty tag identifier rejected before the service",
			service: assignmentsNoCallService,
			args:    []string{"   "},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "tag name or ID is required")
			},
		},
		{
			name:    "unparseable asset filter rejected before the service",
			service: assignmentsNoCallService,
			args:    []string{"my-tag", "--asset", "bogus%%"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to infer asset type")
			},
		},
		{
			name:    "multiple assets in the filter rejected before the service",
			service: assignmentsNoCallService,
			args:    []string{"my-tag", "--asset", "8.8.8.8,1.1.1.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "only 1")
			},
		},
		{
			name:    "page-size above the API maximum rejected before the service",
			service: assignmentsNoCallService,
			args:    []string{"my-tag", "--page-size", "1001"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "1000")
			},
		},
		{
			name:    "zero max-pages rejected before the service",
			service: assignmentsNoCallService,
			args:    []string{"my-tag", "--max-pages", "0"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "must be -1 or >= 1")
			},
		},
		{
			name:    "invalid timestamp rejected before the service",
			service: assignmentsNoCallService,
			args:    []string{"my-tag", "--created-after", "yesterday"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid timestamp")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			stdout, stderr, err := runAssignmentsCommand(t, tc.service(ctrl), tc.args)
			tc.assert(t, stdout, stderr, err)
		})
	}
}
