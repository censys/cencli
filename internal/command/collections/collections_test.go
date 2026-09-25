package collections

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	collectionsmocks "github.com/censys/cencli/gen/app/collections/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	appcollections "github.com/censys/cencli/internal/app/collections"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
)

const testCollectionID = "550e8400-e29b-41d4-a716-446655440000"

func okMeta() *responsemeta.ResponseMeta {
	return &responsemeta.ResponseMeta{
		Method:  "GET",
		URL:     "https://api.censys.io/v3/collections",
		Status:  200,
		Latency: 100 * time.Millisecond,
	}
}

func collection(name string) appcollections.Collection {
	return appcollections.Collection{
		ID:          testCollectionID,
		Name:        name,
		Query:       "host.services.protocol=SSH",
		Status:      "active",
		TotalAssets: 42,
		CreateTime:  time.Unix(0, 0).UTC(),
	}
}

// runCommand executes cmd (built by build from a context that holds svc) with args.
func runCommand(
	t *testing.T,
	svc appcollections.Service,
	build func(*command.Context) command.Command,
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
	cmdContext := command.NewCommandContext(cfg, mockStore, command.WithCollectionsService(svc))
	rootCmd, buildErr := command.RootCommandToCobra(build(cmdContext))
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

func TestRequireCollectionID(t *testing.T) {
	testCases := []struct {
		name    string
		raw     string
		want    string
		wantErr string
	}{
		{name: "valid", raw: testCollectionID, want: testCollectionID},
		{name: "trims spaces and accepts upper case", raw: " 550E8400-E29B-41D4-A716-446655440000 ", want: testCollectionID},
		{name: "not a uuid", raw: "not-a-uuid", wantErr: `invalid collection ID "not-a-uuid"`},
		{name: "blank", raw: "   ", wantErr: "a collection ID is required"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := requireCollectionID(tc.raw)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got.String())
		})
	}
}

func TestCollectionsListCommand(t *testing.T) {
	build := func(c *command.Context) command.Command { return NewListCommand(c) }
	testCases := []struct {
		name    string
		service func(ctrl *gomock.Controller) appcollections.Service
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "success - short output",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), gomock.Any()).Return(
					appcollections.ListResult{Meta: okMeta(), Collections: []appcollections.Collection{collection("alpha")}}, nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Collections (1)")
				require.Contains(t, stdout, "alpha")
				require.Contains(t, stdout, "active")
			},
		},
		{
			name: "success - empty list",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), gomock.Any()).Return(appcollections.ListResult{Meta: okMeta()}, nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "No collections found.")
			},
		},
		{
			name: "success - json output",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), gomock.Any()).Return(
					appcollections.ListResult{Meta: okMeta(), Collections: []appcollections.Collection{collection("alpha")}}, nil)
				return m
			},
			args: []string{"--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, `"name": "alpha"`)
				require.Contains(t, stdout, `"total_assets": 42`)
			},
		},
		{
			name: "status flags are passed as a list",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params appcollections.ListParams) (appcollections.ListResult, cenclierrors.CencliError) {
						require.Equal(t, []string{"active", "paused"}, params.Statuses)
						return appcollections.ListResult{Meta: okMeta()}, nil
					},
				)
				return m
			},
			args: []string{"--status", "active, paused"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "blank status entries are dropped",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params appcollections.ListParams) (appcollections.ListResult, cenclierrors.CencliError) {
						require.Equal(t, []string{"active", "paused"}, params.Statuses)
						return appcollections.ListResult{Meta: okMeta()}, nil
					},
				)
				return m
			},
			args: []string{"--status", "active,,paused"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "max-pages 0 is rejected",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				return collectionsmocks.NewMockCollectionsService(ctrl)
			},
			args: []string{"--max-pages", "0"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "max-pages")
			},
		},
		{
			name: "max-pages -1 warns and fetches all",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params appcollections.ListParams) (appcollections.ListResult, cenclierrors.CencliError) {
						require.True(t, params.MaxPages.IsAbsent())
						return appcollections.ListResult{Meta: okMeta()}, nil
					},
				)
				return m
			},
			args: []string{"--max-pages", "-1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "fetching all pages")
			},
		},
		{
			name: "truncated results print a stderr note",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), gomock.Any()).Return(
					appcollections.ListResult{
						Meta:        okMeta(),
						Collections: []appcollections.Collection{collection("alpha")},
						HasMore:     true,
					}, nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "More collections are available")
			},
		},
		{
			name: "no more pages prints no stderr note",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), gomock.Any()).Return(
					appcollections.ListResult{
						Meta:        okMeta(),
						Collections: []appcollections.Collection{collection("alpha")},
						HasMore:     false,
					}, nil)
				return m
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.NotContains(t, stderr, "More collections are available")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stdout, stderr, err := runCommand(t, tc.service(ctrl), build, tc.args)
			tc.assert(t, stdout, stderr, err)
		})
	}
}
