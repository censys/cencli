package enrich

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	enrichmocks "github.com/censys/cencli/gen/app/enrich/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/app/enrich"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
)

const testOrgID = "550e8400-e29b-41d4-a716-446655440001"

func strPtr(s string) *string { return &s }

func enrichedHost(ip string) *assets.EnrichedHost {
	eh := assets.NewEnrichedHost(components.HostEnrichment{IP: strPtr(ip)})
	return &eh
}

func okMeta() *responsemeta.ResponseMeta {
	return &responsemeta.ResponseMeta{
		Method:  "GET",
		URL:     "https://api.censys.io/v3/global/asset/enrichment/host/8.8.8.8",
		Status:  200,
		Latency: 100 * time.Millisecond,
	}
}

func TestEnrichCommand(t *testing.T) {
	testCases := []struct {
		name    string
		service func(ctrl *gomock.Controller) enrich.Service
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "success - single ip with org",
			service: func(ctrl *gomock.Controller) enrich.Service {
				mockSvc := enrichmocks.NewMockEnrichService(ctrl)
				mockSvc.EXPECT().EnrichHosts(gomock.Any(), gomock.Any(), gomock.Len(1)).Return(
					enrich.Result{Meta: okMeta(), Hosts: []*assets.EnrichedHost{enrichedHost("8.8.8.8")}},
					nil,
				)
				return mockSvc
			},
			args: []string{"--org-id", testOrgID, "8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "8.8.8.8")
			},
		},
		{
			name: "success - multiple ips with org",
			service: func(ctrl *gomock.Controller) enrich.Service {
				mockSvc := enrichmocks.NewMockEnrichService(ctrl)
				mockSvc.EXPECT().EnrichHosts(gomock.Any(), gomock.Any(), gomock.Len(2)).Return(
					enrich.Result{Meta: okMeta(), Hosts: []*assets.EnrichedHost{enrichedHost("8.8.8.8"), enrichedHost("1.1.1.1")}},
					nil,
				)
				return mockSvc
			},
			args: []string{"--org-id", testOrgID, "8.8.8.8,1.1.1.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "8.8.8.8")
				require.Contains(t, stdout, "1.1.1.1")
			},
		},
		{
			name: "error - missing org id",
			service: func(ctrl *gomock.Controller) enrich.Service {
				return enrichmocks.NewMockEnrichService(ctrl) // not called
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "organization")
			},
		},
		{
			name: "error - invalid ip",
			service: func(ctrl *gomock.Controller) enrich.Service {
				return enrichmocks.NewMockEnrichService(ctrl) // not called
			},
			args: []string{"--org-id", testOrgID, "not-an-ip"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "valid host IP")
			},
		},
		{
			name: "error - no hosts provided",
			service: func(ctrl *gomock.Controller) enrich.Service {
				return enrichmocks.NewMockEnrichService(ctrl) // not called
			},
			args: []string{"--org-id", testOrgID},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no host IPs provided")
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

			mockStore := storemocks.NewMockStore(ctrl)
			cmdContext := command.NewCommandContext(cfg, mockStore, command.WithEnrichService(tc.service(ctrl)))
			rootCmd, err := command.RootCommandToCobra(NewEnrichCommand(cmdContext))
			require.NoError(t, err)

			rootCmd.SetArgs(tc.args)
			cmdErr := rootCmd.Execute()
			tc.assert(t, stdout.String(), stderr.String(), cmdErr)
		})
	}
}

func TestEnrichCommand_PartialError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := storemocks.NewMockStore(ctrl)
	mockSvc := enrichmocks.NewMockEnrichService(ctrl)

	baseErr := cenclierrors.NewCencliError(errors.New("1 of 2 host(s) failed to enrich"))
	mockSvc.EXPECT().EnrichHosts(gomock.Any(), gomock.Any(), gomock.Len(2)).Return(
		enrich.Result{
			Meta:         okMeta(),
			Hosts:        []*assets.EnrichedHost{enrichedHost("8.8.8.8")},
			PartialError: cenclierrors.ToPartialError(baseErr),
		},
		nil,
	)

	tempDir := t.TempDir()
	viper.Reset()
	cfg, err := config.New(tempDir)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	formatter.Stdout = &stdout
	formatter.Stderr = &stderr

	cmdContext := command.NewCommandContext(cfg, mockStore, command.WithEnrichService(mockSvc))
	rootCmd, err := command.RootCommandToCobra(NewEnrichCommand(cmdContext))
	require.NoError(t, err)

	rootCmd.SetArgs([]string{"--org-id", testOrgID, "8.8.8.8,1.1.1.1"})
	cmdErr := rootCmd.Execute()

	require.NoError(t, cmdErr)
	require.Contains(t, stdout.String(), "8.8.8.8", "should render data to stdout")
	require.Contains(t, stderr.String(), "(partial data)", "should indicate partial results in stderr")
	require.Contains(t, stderr.String(), "1 of 2", "should print the partial error detail to stderr")
}
