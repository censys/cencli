package search

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	searchmocks "github.com/censys/cencli/gen/app/search/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/app/search"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
	"github.com/censys/censys-sdk-go/models/components"
)

func TestSearchCommand(t *testing.T) {
	testCases := []struct {
		name    string
		store   func(ctrl *gomock.Controller) store.Store
		service func(ctrl *gomock.Controller) search.Service
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		// Success cases - basic functionality
		{
			name: "success - no fields - no org - no collection - default pagination",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				mockSvc := searchmocks.NewMockSearchService(ctrl)
				mockSvc.EXPECT().Search(
					gomock.Any(),
					gomock.AssignableToTypeOf(search.Params{}),
				).Return(
					search.Result{
						Meta: &responsemeta.ResponseMeta{
							Method:  "POST",
							URL:     "https://api.censys.io/v1/search",
							Status:  200,
							Latency: 100 * time.Millisecond,
						},
						Hits: []assets.Asset{
							&assets.Host{
								Host: components.Host{
									IP: strPtr("127.0.0.1"),
								},
							},
						},
					}, nil)
				return mockSvc
			},
			args: []string{"host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "127.0.0.1")
			},
		},
		{
			name: "success - with matched services",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				mockSvc := searchmocks.NewMockSearchService(ctrl)
				mockSvc.EXPECT().Search(
					gomock.Any(),
					gomock.AssignableToTypeOf(search.Params{}),
				).Return(
					search.Result{
						Meta: &responsemeta.ResponseMeta{
							Method:  "POST",
							URL:     "https://api.censys.io/v1/search",
							Status:  200,
							Latency: 100 * time.Millisecond,
						},
						Hits: []assets.Asset{
							&assets.Host{
								Host: components.Host{
									IP: strPtr("127.0.0.1"),
								},
								MatchedServices: []components.MatchedService{
									{
										Port:              intPtr(22),
										Protocol:          strPtr("SSH"),
										TransportProtocol: strPtr(components.TransportProtocolTCP),
									},
								},
							},
						},
					}, nil)
				return mockSvc
			},
			args: []string{"host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "127.0.0.1")
				require.Contains(t, stdout, "SSH")
				require.Contains(t, stdout, "tcp")
				require.Contains(t, stdout, "22")
			},
		},
		{
			name: "success - with fields",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				mockSvc := searchmocks.NewMockSearchService(ctrl)
				mockSvc.EXPECT().Search(
					gomock.Any(),
					gomock.AssignableToTypeOf(search.Params{}),
				).Return(
					search.Result{
						Meta: &responsemeta.ResponseMeta{
							Method:  "POST",
							URL:     "https://api.censys.io/v1/search",
							Status:  200,
							Latency: 100 * time.Millisecond,
						},
						Hits: []assets.Asset{
							&assets.Host{
								Host: components.Host{
									IP: strPtr("127.0.0.1"),
								},
							},
						},
					}, nil)
				return mockSvc
			},
			args: []string{"--fields", "host.ip,host.location", "host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "127.0.0.1")
			},
		},
		{
			name: "success - with orgid",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				mockSvc := searchmocks.NewMockSearchService(ctrl)
				mockSvc.EXPECT().Search(
					gomock.Any(),
					gomock.AssignableToTypeOf(search.Params{}),
				).Return(
					search.Result{
						Meta: &responsemeta.ResponseMeta{
							Method:  "POST",
							URL:     "https://api.censys.io/v1/search",
							Status:  200,
							Latency: 100 * time.Millisecond,
						},
						Hits: []assets.Asset{
							&assets.Host{
								Host: components.Host{
									IP: strPtr("127.0.0.1"),
								},
							},
						},
					}, nil)
				return mockSvc
			},
			args: []string{"--org-id", "550e8400-e29b-41d4-a716-446655440001", "host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "127.0.0.1")
			},
		},
		{
			name: "success - with collection",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				mockSvc := searchmocks.NewMockSearchService(ctrl)
				mockSvc.EXPECT().Search(
					gomock.Any(),
					gomock.AssignableToTypeOf(search.Params{}),
				).Return(
					search.Result{
						Meta: &responsemeta.ResponseMeta{
							Method:  "POST",
							URL:     "https://api.censys.io/v1/search",
							Status:  200,
							Latency: 100 * time.Millisecond,
						},
						Hits: []assets.Asset{
							&assets.Host{
								Host: components.Host{
									IP: strPtr("127.0.0.1"),
								},
							},
						},
					}, nil)
				return mockSvc
			},
			args: []string{"--collection-id", "550e8400-e29b-41d4-a716-446655440000", "host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "127.0.0.1")
			},
		},
		{
			name: "success - with custom pagination",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				mockSvc := searchmocks.NewMockSearchService(ctrl)
				mockSvc.EXPECT().Search(
					gomock.Any(),
					gomock.AssignableToTypeOf(search.Params{}),
				).Return(
					search.Result{
						Meta: &responsemeta.ResponseMeta{
							Method:  "POST",
							URL:     "https://api.censys.io/v1/search",
							Status:  200,
							Latency: 100 * time.Millisecond,
						},
						Hits: []assets.Asset{
							&assets.Host{
								Host: components.Host{
									IP: strPtr("127.0.0.1"),
								},
							},
						},
					}, nil)
				return mockSvc
			},
			args: []string{"--page-size", "25", "--max-pages", "5", "host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "127.0.0.1")
			},
		},
		{
			name: "success - all flags combined",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				mockSvc := searchmocks.NewMockSearchService(ctrl)
				mockSvc.EXPECT().Search(
					gomock.Any(),
					gomock.AssignableToTypeOf(search.Params{}),
				).Return(
					search.Result{
						Meta: &responsemeta.ResponseMeta{
							Method:  "POST",
							URL:     "https://api.censys.io/v1/search",
							Status:  200,
							Latency: 100 * time.Millisecond,
						},
						Hits: []assets.Asset{
							&assets.Host{
								Host: components.Host{
									IP: strPtr("127.0.0.1"),
								},
							},
						},
					}, nil)
				return mockSvc
			},
			args: []string{
				"--org-id", "550e8400-e29b-41d4-a716-446655440001",
				"--collection-id", "550e8400-e29b-41d4-a716-446655440000",
				"--fields", "host.ip,host.location",
				"--page-size", "20",
				"--max-pages", "3",
				"host.ip: 127.0.0.1",
			},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "127.0.0.1")
			},
		},
		// Pagination validation error cases
		{
			name: "error - page size below minimum",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				return searchmocks.NewMockSearchService(ctrl)
			},
			args: []string{"--page-size", "0", "host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "must be >= 1")
			},
		},
		{
			name: "error - max pages below minimum",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				return searchmocks.NewMockSearchService(ctrl)
			},
			args: []string{"--max-pages", "0", "host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "must be -1 or >= 1")
			},
		},
		{
			name: "error - negative page size",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				return searchmocks.NewMockSearchService(ctrl)
			},
			args: []string{"--page-size", "-5", "host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "must be >= 1")
			},
		},
		{
			name: "error - negative max pages",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				return searchmocks.NewMockSearchService(ctrl)
			},
			args: []string{"--max-pages", "-2", "host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "must be -1 or >= 1")
			},
		},
		// Service layer error cases
		{
			name: "error - service search failure",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				mockSvc := searchmocks.NewMockSearchService(ctrl)
				mockSvc.EXPECT().Search(
					gomock.Any(),
					gomock.AssignableToTypeOf(search.Params{}),
				).Return(
					search.Result{},
					search.NewInvalidPaginationParamsError("service error"),
				)
				return mockSvc
			},
			args: []string{"host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "service error")
			},
		},
		// Flag validation error cases
		{
			name: "error - invalid collection id format",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				return searchmocks.NewMockSearchService(ctrl)
			},
			args: []string{"--collection-id", "invalid-uuid", "host.ip: 127.0.0.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid uuid")
			},
		},
		{
			name: "error - missing query argument",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				return searchmocks.NewMockSearchService(ctrl)
			},
			args: []string{},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
			},
		},
		{
			name: "error - too many query arguments",
			store: func(ctrl *gomock.Controller) store.Store {
				return storemocks.NewMockStore(ctrl)
			},
			service: func(ctrl *gomock.Controller) search.Service {
				return searchmocks.NewMockSearchService(ctrl)
			},
			args: []string{"query1", "query2"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "accepts 1 arg(s), received 2")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			viper.Reset()
			cfg, err := config.New(tempDir)
			require.NoError(t, err)

			var stdout, stderr bytes.Buffer
			formatter.Stdout = &stdout
			formatter.Stderr = &stderr

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			queryConverterSvc := tc.service(ctrl)
			cmdContext := command.NewCommandContext(cfg, tc.store(ctrl), command.WithSearchService(queryConverterSvc))
			rootCmd, err := command.RootCommandToCobra(NewSearchCommand(cmdContext))
			require.NoError(t, err)

			rootCmd.SetArgs(tc.args)
			cmdErr := rootCmd.Execute()
			tc.assert(t, stdout.String(), stderr.String(), cmdErr)
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func strPtr[T ~string](v T) *T {
	return &v
}

func TestSearchCommand_PartialError(t *testing.T) {
	t.Run("prints partial error to stderr after rendering data", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := storemocks.NewMockStore(ctrl)
		mockSvc := searchmocks.NewMockSearchService(ctrl)

		// Service returns partial results with error wrapped in NewPartialError
		baseErr := cenclierrors.NewCencliError(errors.New("Page 2 failed"))
		mockSvc.EXPECT().Search(
			gomock.Any(),
			gomock.AssignableToTypeOf(search.Params{}),
		).Return(
			search.Result{
				Meta: &responsemeta.ResponseMeta{
					Method:  "POST",
					URL:     "https://api.censys.io/v1/search",
					Status:  200,
					Latency: 100 * time.Millisecond,
				},
				Hits: []assets.Asset{
					&assets.Host{
						Host: components.Host{
							IP: strPtr("127.0.0.1"),
						},
					},
				},
				PartialError: cenclierrors.ToPartialError(baseErr),
			}, nil)

		tempDir := t.TempDir()
		viper.Reset()
		cfg, err := config.New(tempDir)
		require.NoError(t, err)

		cmdContext := command.NewCommandContext(cfg, mockStore, command.WithSearchService(mockSvc))

		searchCmd := NewSearchCommand(cmdContext)
		rootCmd, err := command.RootCommandToCobra(searchCmd)
		require.NoError(t, err)

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		rootCmd.SetOut(stdout)
		rootCmd.SetErr(stderr)
		formatter.Stdout = stdout
		formatter.Stderr = stderr

		rootCmd.SetArgs([]string{"host.ip: 127.0.0.1"})
		cmdErr := rootCmd.Execute()

		require.NoError(t, cmdErr)
		require.Contains(t, stdout.String(), "127.0.0.1", "should render data to stdout")
		require.Contains(t, stderr.String(), "(partial data)", "should indicate partial results in stderr")
		require.Contains(t, stderr.String(), "Page 2 failed", "should print partial error to stderr")
		require.Contains(t, stderr.String(), "some data was successfully retrieved", "should include partial error message")
	})
}
