package scan

import (
	"bytes"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	clientmocks "github.com/censys/cencli/gen/client/mocks"
	storemocks "github.com/censys/cencli/gen/store/mocks"
	appscan "github.com/censys/cencli/internal/app/scan"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/credential"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
)

const (
	testScanID    = "3f2b9c1e-0000-4000-8000-000000000001"
	testFlagOrg   = "00000000-0000-0000-0000-00000000000a"
	testStoredOrg = "00000000-0000-0000-0000-00000000000b"
	testBoundOrg  = "00000000-0000-0000-0000-00000000000c"
)

var (
	patCredential      = credential.Info{Kind: credential.KindPersonalAccessToken}
	oauthOrgCredential = credential.Info{Kind: credential.KindOAuth, OrgID: testBoundOrg, OrgName: "Censys"}
	oauthFreeAccount   = credential.Info{Kind: credential.KindOAuth}
)

func orgID(s string) identifiers.OrganizationID {
	return identifiers.NewOrganizationID(uuid.MustParse(s))
}

// env describes the credential a command runs under and the org-id global it
// finds stored ("" means none).
type env struct {
	cred      credential.Info
	storedOrg string
}

var patWithStoredOrg = env{cred: patCredential, storedOrg: testStoredOrg}

func runScanCommand(
	t *testing.T,
	newCmd func(*command.Context) command.Command,
	svc appscan.Service,
	e env,
	args []string,
) (stdout, stderr string, err error) {
	t.Helper()

	viper.Reset()
	cfg, cfgErr := config.New(t.TempDir())
	require.NoError(t, cfgErr)

	var outBuf, errBuf bytes.Buffer
	formatter.Stdout = &outBuf
	formatter.Stderr = &errBuf

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ds := storemocks.NewMockStore(ctrl)
	if e.storedOrg != "" {
		ds.EXPECT().GetLastUsedGlobalByName(gomock.Any(), config.OrgIDGlobalName).
			Return(&store.ValueForGlobal{Value: e.storedOrg}, nil).AnyTimes()
	} else {
		ds.EXPECT().GetLastUsedGlobalByName(gomock.Any(), config.OrgIDGlobalName).
			Return((*store.ValueForGlobal)(nil), store.ErrGlobalNotFound).AnyTimes()
	}

	cli := clientmocks.NewMockClient(ctrl)
	cli.EXPECT().CredentialInfo().Return(e.cred).AnyTimes()

	cmdContext := command.NewCommandContext(cfg, ds, command.WithScanService(svc))
	cmdContext.SetCensysClient(cli)
	rootCmd, buildErr := command.RootCommandToCobra(newCmd(cmdContext))
	require.NoError(t, buildErr)

	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), cmdErr
}

func okMeta() *responsemeta.ResponseMeta {
	return &responsemeta.ResponseMeta{Method: "GET", URL: "https://api.censys.io/v3/global/scans", Status: 200}
}

func trackedScan(completed bool, statuses ...string) *appscan.TrackedScan {
	createTime := "2026-09-29T14:02:11Z"
	host, port := "example.com", 443
	s := &appscan.TrackedScan{ID: testScanID, Completed: completed, CreateTime: &createTime}
	s.Target = &components.TrackedScanScanTarget{WebOrigin: &components.WebOrigin{Hostname: &host, Port: &port}}
	for _, st := range statuses {
		s.Tasks = append(s.Tasks, appscan.Task{Description: "HTTP endpoint scan", Status: st})
	}
	return s
}

func scanResult(s *appscan.TrackedScan) appscan.Result {
	return appscan.Result{Meta: okMeta(), Scan: s}
}

func mo15m() mo.Option[time.Duration] { return mo.Some(defaultWaitTimeout) }

type assertErr string

func (e assertErr) Error() string { return string(e) }
