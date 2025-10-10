package view

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"
	"github.com/samber/mo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	viewmocks "github.com/censys/cencli/gen/app/view/mocks"
	"github.com/censys/cencli/internal/app/view"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
)

func TestViewCommand(t *testing.T) {
	testCases := []struct {
		name    string
		store   func() store.Store
		service func(ctrl *gomock.Controller) view.Service
		setup   func(t *testing.T, args []string)
		stdin   string
		args    []string
		assert  func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name:  "host view - explicit ip",
			store: func() store.Store { s, _ := store.New(t.TempDir()); return s },
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{IP: strPtr("8.8.8.8")}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8", "--short"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "8.8.8.8")
			},
		},
		{
			name:  "web property view - explicit domain:port",
			store: func() store.Store { s, _ := store.New(t.TempDir()); return s },
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				wp, _ := assets.NewWebPropertyID("platform.censys.io:443", assets.DefaultWebPropertyPort)
				w := &assets.WebProperty{Webproperty: components.Webproperty{Hostname: strPtr("platform.censys.io"), Port: intPtr(443)}}
				result := view.WebPropertiesResult{
					Meta:          &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					WebProperties: []*assets.WebProperty{w},
				}
				ms.EXPECT().GetWebProperties(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.WebPropertyID{wp}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			args: []string{"platform.censys.io:443", "--short"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "platform.censys.io")
			},
		},
		{
			name:  "host view - short output",
			store: func() store.Store { s, _ := store.New(t.TempDir()); return s },
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{IP: strPtr("8.8.8.8")}}
				result := view.HostsResult{
					Meta:  &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8", "--short"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "8.8.8.8")
			},
		},
		{
			name:  "certificate view - short output",
			store: func() store.Store { s, _ := store.New(t.TempDir()); return s },
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				fp := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
				certID, _ := assets.NewCertificateFingerprint(fp)
				cert := &assets.Certificate{Certificate: components.Certificate{FingerprintSha256: strPtr(fp)}}
				result := view.CertificatesResult{
					Meta:         &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Certificates: []*assets.Certificate{cert},
				}
				ms.EXPECT().GetCertificates(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.CertificateID{certID}).Return(result, nil)
				return ms
			},
			args: []string{"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", "--short"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				// Expect truncated format first16…last4
				require.Contains(t, stdout, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
			},
		},
		{
			name:  "web property view - short output",
			store: func() store.Store { s, _ := store.New(t.TempDir()); return s },
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				wp, _ := assets.NewWebPropertyID("platform.censys.io:80", assets.DefaultWebPropertyPort)
				w := &assets.WebProperty{Webproperty: components.Webproperty{Hostname: strPtr("platform.censys.io"), Port: intPtr(80)}}
				result := view.WebPropertiesResult{
					Meta:          &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					WebProperties: []*assets.WebProperty{w},
				}
				ms.EXPECT().GetWebProperties(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.WebPropertyID{wp}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			args: []string{"platform.censys.io:80", "--short"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "platform.censys.io")
			},
		},
		{
			name:  "host view - stdin input file",
			store: func() store.Store { s, _ := store.New(t.TempDir()); return s },
			service: func(ctrl *gomock.Controller) view.Service {
				mc := viewmocks.NewMockViewService(ctrl)
				hostID1, _ := assets.NewHostID("8.8.8.8")
				hostID2, _ := assets.NewHostID("1.1.1.1")
				mc.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID1, hostID2}, mo.None[time.Time]()).
					Return(view.HostsResult{
						Meta: &responsemeta.ResponseMeta{
							Method:  "GET",
							URL:     "https://127.0.0.1",
							Status:  200,
							Latency: 100 * time.Millisecond,
						},
						Hosts: []*assets.Host{{Host: components.Host{IP: strPtr("8.8.8.8")}}, {Host: components.Host{IP: strPtr("1.1.1.1")}}},
					}, nil)
				return mc
			},
			stdin: "8.8.8.8\n1.1.1.1\n",
			args:  []string{"--input-file", "-"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				// status now prints as: "200 (OK) - ..."
				require.Contains(t, stderr, "200")
				require.Contains(t, stderr, "OK")
				require.Contains(t, stdout, "8.8.8.8")
				require.Contains(t, stdout, "1.1.1.1")
			},
		},
		{
			name: "help message",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			service: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			args: []string{"--help"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Usage:")
				require.Contains(t, stdout, "view <asset>")
			},
		},
		{
			name:  "host view",
			store: func() store.Store { s, _ := store.New(t.TempDir()); return s },
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{IP: strPtr("8.8.8.8")}}
				result := view.HostsResult{
					Meta: &responsemeta.ResponseMeta{
						Method:  "GET",
						URL:     "https://127.0.0.1",
						Status:  200,
						Latency: 100 * time.Millisecond,
					},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "200")
				require.Contains(t, stdout, "8.8.8.8")
			},
		},
		{
			name: "host view - multiple hosts",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				host1 := &assets.Host{Host: components.Host{IP: strPtr("8.8.8.8")}}
				host2 := &assets.Host{Host: components.Host{IP: strPtr("1.1.1.1")}}
				result := view.HostsResult{
					Meta: &responsemeta.ResponseMeta{
						Method:  "GET",
						URL:     "https://127.0.0.1",
						Status:  200,
						Latency: 100 * time.Millisecond,
					},
					Hosts: []*assets.Host{host1, host2},
				}
				// Use gomock.Any() for the hostIDs slice since order might vary
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8,1.1.1.1"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "200")
				require.Contains(t, stdout, "8.8.8.8")
				require.Contains(t, stdout, "1.1.1.1")
			},
		},
		{
			name: "certificate view",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				fingerprint := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
				certID, _ := assets.NewCertificateFingerprint(fingerprint)
				cert := &assets.Certificate{Certificate: components.Certificate{FingerprintSha256: strPtr(fingerprint)}}
				result := view.CertificatesResult{
					Meta: &responsemeta.ResponseMeta{
						Method:  "GET",
						URL:     "https://127.0.0.1",
						Status:  200,
						Latency: 150 * time.Millisecond,
					},
					Certificates: []*assets.Certificate{cert},
				}
				ms.EXPECT().GetCertificates(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.CertificateID{certID}).Return(result, nil)
				return ms
			},
			args: []string{"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "200")
				require.Contains(t, stdout, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
			},
		},
		{
			name: "web property view",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				webPropID, _ := assets.NewWebPropertyID("platform.censys.io:80", assets.DefaultWebPropertyPort)
				webProp := &assets.WebProperty{Webproperty: components.Webproperty{Hostname: strPtr("platform.censys.io"), Port: intPtr(80)}}
				result := view.WebPropertiesResult{
					Meta: &responsemeta.ResponseMeta{
						Method:  "GET",
						URL:     "https://127.0.0.1",
						Status:  200,
						Latency: 200 * time.Millisecond,
					},
					WebProperties: []*assets.WebProperty{webProp},
				}
				ms.EXPECT().GetWebProperties(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.WebPropertyID{webPropID}, mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			args: []string{"platform.censys.io:80"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "200")
				require.Contains(t, stdout, "platform.censys.io")
				require.Contains(t, stdout, "80")
			},
		},
		{
			name: "host view - structured error",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
					Detail:   strPtr("test-detail"),
					Title:    strPtr("test-title"),
					Status:   int64Ptr(400),
					Type:     strPtr("test-type"),
					Instance: strPtr("test-instance"),
				})
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(view.HostsResult{}, structuredErr)
				return ms
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr client.ClientStructuredError
				require.ErrorAs(t, err, &cencliErr)
				errStr := err.Error()
				require.Contains(t, errStr, "test-detail")
				require.Contains(t, errStr, "test-title")
				require.Contains(t, errStr, "400")
				require.Contains(t, errStr, "test-type")
				require.Contains(t, errStr, "test-instance")
			},
		},
		{
			name: "host view - generic error",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "test-message",
					StatusCode: 400,
					Body:       "test-body",
				})
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(view.HostsResult{}, genericErr)
				return ms
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr client.ClientGenericError
				require.ErrorAs(t, err, &cencliErr)
				errStr := err.Error()
				require.Contains(t, errStr, "test-message")
				require.Contains(t, errStr, "400")
				require.Contains(t, errStr, "test-body")
			},
		},
		{
			name: "host view - unknown error",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				unknownErr := client.NewClientError(errors.New("test-error"))
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(view.HostsResult{}, unknownErr)
				return ms
			},
			args: []string{"8.8.8.8"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr client.ClientError
				require.ErrorAs(t, err, &cencliErr)
				errStr := err.Error()
				require.Contains(t, errStr, "test-error")
			},
		},
		{
			name: "host view - bad org id",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			service: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			args: []string{"8.8.8.8", "--org-id", "bad-org-id"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr flags.InvalidUUIDFlagError
				require.ErrorAs(t, err, &cencliErr)
				errStr := err.Error()
				require.Contains(t, errStr, "invalid uuid")
			},
		},
		{
			name: "no argument",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			service: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			args: []string{},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr assets.NoAssetsError
				require.ErrorAs(t, err, &cencliErr)
				errStr := err.Error()
				require.Contains(t, errStr, "you must provide at least one asset")
			},
		},
		{
			name: "host view - mixed view types",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			service: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			args: []string{"8.8.8.8,2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var cencliErr assets.MixedAssetTypesError
				require.ErrorAs(t, err, &cencliErr)
				errStr := err.Error()
				require.Contains(t, errStr, "mixed asset types")
				require.Contains(t, errStr, "host, certificate")
			},
		},
		{
			name: "host view - input file",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			setup: func(t *testing.T, args []string) {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "assets.txt")
				err := os.WriteFile(filePath, []byte("8.8.8.8\n1.1.1.1\n"), 0o644)
				require.NoError(t, err)
				args[len(args)-1] = filepath.Join(tempDir, args[len(args)-1])
			},
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				host1 := &assets.Host{Host: components.Host{IP: strPtr("8.8.8.8")}}
				host2 := &assets.Host{Host: components.Host{IP: strPtr("1.1.1.1")}}
				result := view.HostsResult{
					Meta: &responsemeta.ResponseMeta{
						Method:  "GET",
						URL:     "https://127.0.0.1",
						Status:  200,
						Latency: 100 * time.Millisecond,
					},
					Hosts: []*assets.Host{host1, host2},
				}
				// Use gomock.Any() for the hostIDs slice since order might vary
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), gomock.Any(), mo.None[time.Time]()).Return(result, nil)
				return ms
			},
			args: []string{"--input-file", "assets.txt"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "200")
				require.Contains(t, stdout, "8.8.8.8")
				require.Contains(t, stdout, "1.1.1.1")
			},
		},
		{
			name: "certificate view - at-time",
			store: func() store.Store {
				s, _ := store.New(t.TempDir())
				return s
			},
			service: func(ctrl *gomock.Controller) view.Service {
				return viewmocks.NewMockViewService(ctrl)
			},
			args: []string{"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", "--at-time", "2025-09-15T14:30:00Z"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				var domErr AtTimeNotSupportedError
				require.ErrorAs(t, err, &domErr)
				require.Contains(t, err.Error(), "at-time is not supported for certificate assets")
			},
		},
		{
			name:  "host view - custom at time format",
			store: func() store.Store { s, _ := store.New(t.TempDir()); return s },
			service: func(ctrl *gomock.Controller) view.Service {
				ms := viewmocks.NewMockViewService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				host := &assets.Host{Host: components.Host{IP: strPtr("8.8.8.8")}}
				result := view.HostsResult{
					Meta: &responsemeta.ResponseMeta{
						Method:  "GET",
						URL:     "https://127.0.0.1",
						Status:  200,
						Latency: 100 * time.Millisecond,
					},
					Hosts: []*assets.Host{host},
				}
				ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.Some(time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC))).Return(result, nil)
				return ms
			},
			args: []string{"8.8.8.8", "--at-time", "2025-09-15"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stderr, "200")
				require.Contains(t, stdout, "8.8.8.8")
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

			if tc.setup != nil {
				tc.setup(t, tc.args)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			viewSvc := tc.service(ctrl)
			cmdContext := command.NewCommandContext(cfg, tc.store(), command.WithViewService(viewSvc))

			rootCmd, err := command.RootCommandToCobra(NewViewCommand(cmdContext))
			require.NoError(t, err)

			rootCmd.SetArgs(tc.args)
			if tc.stdin != "" {
				rootCmd.SetIn(bytes.NewBufferString(tc.stdin))
			}

			cmdErr := rootCmd.Execute()
			tc.assert(t, stdout.String(), stderr.String(), cmdErr)
		})
	}
}

func TestPreRun_AtTimeNotSupportedForCertificate(t *testing.T) {
	tempDir := t.TempDir()
	viper.Reset()
	cfg, err := config.New(tempDir)
	require.NoError(t, err)
	cmdContext := command.NewCommandContext(cfg, mustStore(t))
	viewCmd := NewViewCommand(cmdContext)
	rootCmd, err := command.RootCommandToCobra(viewCmd)
	require.NoError(t, err)
	// set args: certificate with --at-time
	args := []string{"3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf", "--at-time", time.Now().UTC().Format(time.RFC3339)}
	rootCmd.SetArgs(args)
	cmdErr := rootCmd.Execute()
	require.Error(t, cmdErr)
	var domErr AtTimeNotSupportedError
	assert.True(t, errors.As(cmdErr, &domErr))
}

func mustStore(t *testing.T) store.Store {
	t.Helper()
	s, _ := store.New(t.TempDir())
	return s
}

func TestViewCommand_PartialError(t *testing.T) {
	t.Run("prints partial error to stderr after rendering data", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ms := viewmocks.NewMockViewService(ctrl)
		hostID, _ := assets.NewHostID("8.8.8.8")
		host := &assets.Host{Host: components.Host{IP: strPtr("8.8.8.8")}}

		// Service returns partial results with error wrapped in NewPartialError
		baseErr := client.NewClientError(&sdkerrors.SDKError{Message: "Batch 2 failed", StatusCode: 500})
		result := view.HostsResult{
			Meta:         &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
			Hosts:        []*assets.Host{host},
			PartialError: cenclierrors.ToPartialError(baseErr),
		}
		ms.EXPECT().GetHosts(gomock.Any(), mo.None[identifiers.OrganizationID](), []assets.HostID{hostID}, mo.None[time.Time]()).Return(result, nil)

		tempDir := t.TempDir()
		viper.Reset()
		cfg, err := config.New(tempDir)
		require.NoError(t, err)

		cmdContext := command.NewCommandContext(cfg, mustStore(t), command.WithViewService(ms))

		viewCmd := NewViewCommand(cmdContext)
		rootCmd, err := command.RootCommandToCobra(viewCmd)
		require.NoError(t, err)

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		rootCmd.SetOut(stdout)
		rootCmd.SetErr(stderr)
		formatter.Stdout = stdout
		formatter.Stderr = stderr

		rootCmd.SetArgs([]string{"8.8.8.8", "--short"})
		cmdErr := rootCmd.Execute()

		require.NoError(t, cmdErr)
		assert.Contains(t, stdout.String(), "8.8.8.8", "should render data to stdout")
		assert.Contains(t, stderr.String(), "(partial data)", "should indicate partial results in stderr")
		assert.Contains(t, stderr.String(), "Batch 2 failed", "should print partial error to stderr")
		assert.Contains(t, stderr.String(), "some data was successfully retrieved", "should include partial error message")
	})
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func int64Ptr(i int64) *int64 { return &i }
