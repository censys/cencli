package tags

import (
	"bytes"
	"errors"
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

// errBoom stands in for an arbitrary failure surfacing from the service.
var errBoom = errors.New("boom")

func runOperationsListCommand(t *testing.T, svc apptags.Service, args []string) (stdout, stderr string, err error) {
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
	rootCmd, buildErr := command.RootCommandToCobra(NewOperationsListCommand(cmdContext))
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

func operation(id, status string) apptags.TagOperation {
	return apptags.TagOperation{
		ID:              id,
		TagID:           "a6217129-be72-4b02-a42c-9c431574e524",
		TagName:         "my-tag",
		Type:            "bulk_create",
		Status:          status,
		TotalCount:      100,
		ProcessedCount:  40,
		SuccessfulCount: 38,
		CreatedAt:       time.Unix(0, 0).UTC(),
	}
}

// operationsNoCallService asserts the service is never reached, proving a bad
// input was rejected at the command boundary.
func operationsNoCallService(ctrl *gomock.Controller) apptags.Service {
	m := tagsmocks.NewMockTagsService(ctrl)
	m.EXPECT().ListOperations(gomock.Any(), gomock.Any()).Times(0)
	return m
}

func TestTagsOperationsListCommand(t *testing.T) {
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
				m.EXPECT().ListOperations(gomock.Any(), gomock.Any()).Return(
					apptags.OperationsResult{
						Meta:       okMeta(),
						Operations: []apptags.TagOperation{operation("op-1", "running"), operation("op-2", "succeeded")},
						TotalSize:  2,
					}, nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Operations (2)")
				require.Contains(t, stdout, "op-1")
				require.Contains(t, stdout, "bulk_create")
				require.Contains(t, stdout, "40/100")
			},
		},
		{
			name: "short output reports the API total when truncated",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListOperations(gomock.Any(), gomock.Any()).Return(
					apptags.OperationsResult{
						Meta:       okMeta(),
						Operations: []apptags.TagOperation{operation("op-1", "running")},
						TotalSize:  9,
					}, nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Operations (1 of 9)")
			},
		},
		{
			name: "empty result renders a friendly message",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListOperations(gomock.Any(), gomock.Any()).Return(
					apptags.OperationsResult{Meta: okMeta()}, nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "No operations found.")
			},
		},
		{
			name: "no tag argument leaves the tag unset for org-wide listing",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListOperations(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ any, params apptags.OperationsParams) (apptags.OperationsResult, cenclierrors.CencliError) {
						require.False(t, params.TagID.IsPresent())
						return apptags.OperationsResult{Meta: okMeta()}, nil
					})
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "tag argument and filters are threaded to the service",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListOperations(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ any, params apptags.OperationsParams) (apptags.OperationsResult, cenclierrors.CencliError) {
						require.True(t, params.TagID.IsPresent())
						require.Equal(t, "my-tag", params.TagID.MustGet().String())
						require.Equal(t, mo.Some("bulk_delete"), params.Type)
						require.Equal(t, mo.Some("failed"), params.Status)
						require.Equal(t, mo.Some("create_time_asc"), params.OrderBy)
						require.Equal(t, mo.Some(uint64(25)), params.PageSize)
						return apptags.OperationsResult{Meta: okMeta()}, nil
					})
				return m
			},
			args: []string{
				"my-tag", "--type", "bulk_delete", "--status", "failed",
				"--order-by", "create_time_asc", "--page-size", "25",
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "blank filters are omitted rather than sent empty",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListOperations(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ any, params apptags.OperationsParams) (apptags.OperationsResult, cenclierrors.CencliError) {
						require.False(t, params.Type.IsPresent())
						require.False(t, params.Status.IsPresent())
						return apptags.OperationsResult{Meta: okMeta()}, nil
					})
				return m
			},
			args: []string{"--type", "  ", "--status", ""},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "json output renders the operations",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListOperations(gomock.Any(), gomock.Any()).Return(
					apptags.OperationsResult{
						Meta:       okMeta(),
						Operations: []apptags.TagOperation{operation("op-1", "succeeded")},
						TotalSize:  1,
					}, nil)
				return m
			},
			args: []string{"--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"id": "op-1"`)
				require.Contains(t, stdout, `"status": "succeeded"`)
				// Absent optional fields must not surface as empty strings.
				require.NotContains(t, stdout, `"error_message"`)
				require.NotContains(t, stdout, `"query"`)
			},
		},
		{
			name: "partial error is printed after the data",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListOperations(gomock.Any(), gomock.Any()).Return(
					apptags.OperationsResult{
						Meta:         okMeta(),
						Operations:   []apptags.TagOperation{operation("op-1", "succeeded")},
						TotalSize:    1,
						PartialError: cenclierrors.ToPartialError(cenclierrors.NewCencliError(errBoom)),
					}, nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "op-1")
				require.Contains(t, stderr, "boom")
			},
		},
		{
			name:    "empty tag argument is rejected before the service",
			service: operationsNoCallService,
			args:    []string{"   "},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "tag name or ID is required")
			},
		},
		{
			name:    "too many arguments are rejected",
			service: operationsNoCallService,
			args:    []string{"my-tag", "extra"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name:    "max-pages of zero is rejected before the service",
			service: operationsNoCallService,
			args:    []string{"--max-pages", "0"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "must be -1 or >= 1")
			},
		},
		{
			name:    "page-size above the API maximum is rejected before the service",
			service: operationsNoCallService,
			args:    []string{"--page-size", "1001"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "1000")
			},
		},
		{
			name: "service error is returned",
			service: func(ctrl *gomock.Controller) apptags.Service {
				m := tagsmocks.NewMockTagsService(ctrl)
				m.EXPECT().ListOperations(gomock.Any(), gomock.Any()).Return(
					apptags.OperationsResult{}, cenclierrors.NewCencliError(errBoom))
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

			stdout, stderr, err := runOperationsListCommand(t, tc.service(ctrl), tc.args)
			tc.assert(t, stdout, stderr, err)
		})
	}
}
