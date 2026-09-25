package collections

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	collectionsmocks "github.com/censys/cencli/gen/app/collections/mocks"
	clientmocks "github.com/censys/cencli/gen/client/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	appcollections "github.com/censys/cencli/internal/app/collections"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/credential"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
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
	return runCommandWith(t, svc, build, args, false, nil)
}

// runCommandWith is runCommand with an explicit quiet seam: quiet stands in for
// the global --quiet flag, which lives on the real root command and so is not
// registered when a subcommand is mounted alone. cli, when non-nil, is set on
// the context so credential-aware org resolution (see command.Context.ResolveOrgID)
// can be exercised; a nil cli leaves the context with no client, which reports
// credential.KindNone and so allows a manually chosen org, matching every
// existing test in this file.
func runCommandWith(
	t *testing.T,
	svc appcollections.Service,
	build func(*command.Context) command.Command,
	args []string,
	quiet bool,
	cli client.Client,
) (stdout, stderr string, err error) {
	t.Helper()

	tempDir := t.TempDir()
	viper.Reset()
	cfg, cfgErr := config.New(tempDir)
	require.NoError(t, cfgErr)
	if quiet {
		// PreRun re-reads the config from viper, so setting the struct field would
		// be overwritten; viper is also where the real --quiet flag lands.
		viper.Set("quiet", true)
	}

	var outBuf, errBuf bytes.Buffer
	formatter.Stdout = &outBuf
	formatter.Stderr = &errBuf

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := storemocks.NewMockStore(ctrl)
	// With no client set (credential.KindNone) or a personal-access-token
	// client, ResolveOrgID falls back to the stored org-id global when --org-id
	// is absent; report none stored so a missing flag resolves cleanly.
	mockStore.EXPECT().GetLastUsedGlobalByName(gomock.Any(), gomock.Any()).
		Return((*store.ValueForGlobal)(nil), store.ErrGlobalNotFound).AnyTimes()
	cmdContext := command.NewCommandContext(cfg, mockStore, command.WithCollectionsService(svc))
	if cli != nil {
		cmdContext.SetCensysClient(cli)
	}
	rootCmd, buildErr := command.RootCommandToCobra(build(cmdContext))
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

// boundCredentialClient returns a mock client reporting an OAuth-style
// credential bound to orgID, the same credential.Info shape
// internal/command/context_test.go uses to exercise ResolveOrgID's rejection
// path. Collections commands have no client-injection test mechanism of their
// own (nor does search, which G1 copies), so this reuses that mechanism here.
func boundCredentialClient(ctrl *gomock.Controller, orgID uuid.UUID) client.Client {
	cli := clientmocks.NewMockClient(ctrl)
	cli.EXPECT().CredentialInfo().Return(credential.Info{
		Kind: credential.KindOAuth, OrgID: orgID.String(), OrgName: "Censys",
	}).AnyTimes()
	return cli
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
		{name: "not a uuid", raw: "not-a-uuid", wantErr: `collection ID "not-a-uuid" is not a valid UUID`},
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

func TestShellQuote(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain string", in: "host.services.protocol=SSH", want: "'host.services.protocol=SSH'"},
		{name: "spaces", in: "a b c", want: "'a b c'"},
		{name: "shell variable", in: "$HOME", want: "'$HOME'"},
		{name: "backtick", in: "`whoami`", want: "'`whoami`'"},
		{name: "single quote", in: "it's", want: `'it'\''s'`},
		{name: "empty string", in: "", want: "''"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, shellQuote(tc.in))
		})
	}
}

func TestCollectionsListCommand(t *testing.T) {
	build := func(c *command.Context) command.Command { return NewListCommand(c) }
	sessionOrgID := uuid.New()
	otherOrgID := uuid.New()
	testCases := []struct {
		name    string
		service func(ctrl *gomock.Controller) appcollections.Service
		args    []string
		quiet   bool
		client  func(ctrl *gomock.Controller) client.Client
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
				require.Contains(t, stderr, "more collections are available")
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
				require.NotContains(t, stderr, "more collections are available")
			},
		},
		{
			name: "--quiet suppresses the truncation note",
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
			quiet: true,
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.NotContains(t, stderr, "more collections are available")
			},
		},
		{
			name: "org-bound credential rejects --org-id",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				// The service must not be called: rejection happens in PreRun.
				return collectionsmocks.NewMockCollectionsService(ctrl)
			},
			args:   []string{"--org-id", otherOrgID.String()},
			client: func(ctrl *gomock.Controller) client.Client { return boundCredentialClient(ctrl, sessionOrgID) },
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "only applies to personal access tokens")
			},
		},
		{
			name: "org-bound credential without --org-id passes the bound org to the service",
			service: func(ctrl *gomock.Controller) appcollections.Service {
				m := collectionsmocks.NewMockCollectionsService(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params appcollections.ListParams) (appcollections.ListResult, cenclierrors.CencliError) {
						require.True(t, params.OrgID.IsPresent())
						require.Equal(t, sessionOrgID.String(), params.OrgID.MustGet().String())
						return appcollections.ListResult{Meta: okMeta()}, nil
					},
				)
				return m
			},
			client: func(ctrl *gomock.Controller) client.Client { return boundCredentialClient(ctrl, sessionOrgID) },
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			var cli client.Client
			if tc.client != nil {
				cli = tc.client(ctrl)
			}
			stdout, stderr, err := runCommandWith(t, tc.service(ctrl), build, tc.args, tc.quiet, cli)
			tc.assert(t, stdout, stderr, err)
		})
	}
}
