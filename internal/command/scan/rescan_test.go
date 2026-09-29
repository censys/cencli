package scan

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	scanmocks "github.com/censys/cencli/gen/app/scan/mocks"
	appscan "github.com/censys/cencli/internal/app/scan"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/formatter"
)

func newRescan(c *command.Context) command.Command { return NewRescanCommand(c) }

var exampleWebProperty = assets.WebPropertyID{Hostname: "example.com", Port: 443}

// noCalls is a service that must not be reached: the command fails before it.
func noCalls(ctrl *gomock.Controller) appscan.Service {
	m := scanmocks.NewMockScanService(ctrl)
	m.EXPECT().Rescan(gomock.Any(), gomock.Any()).Times(0)
	m.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)
	m.EXPECT().Wait(gomock.Any(), gomock.Any()).Times(0)
	return m
}

func TestRescanCommand_Rejections(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		env      env
		exitCode int
		contains string
	}{
		{name: "host", args: []string{"8.8.8.8"}, env: patWithStoredOrg, exitCode: 2, contains: `"8.8.8.8" is a host`},
		{name: "certificate", args: []string{"3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf"}, env: patWithStoredOrg, exitCode: 2, contains: "is a certificate"},
		{name: "unrecognised asset", args: []string{"not an asset"}, env: patWithStoredOrg, exitCode: 2, contains: "unable to infer asset type"},
		{name: "two web properties", args: []string{"a.example.com:443,b.example.com:443"}, env: patWithStoredOrg, exitCode: 2},
		{name: "timeout without wait", args: []string{"example.com:443", "--timeout", "5m"}, env: patWithStoredOrg, exitCode: 2, contains: "--timeout only applies while polling"},
		{name: "negative timeout", args: []string{"example.com:443", "--wait", "--timeout", "-1m"}, env: patWithStoredOrg, exitCode: 2, contains: "must not be negative"},
		{name: "pat with no organization", args: []string{"example.com:443"}, env: env{cred: patCredential}, exitCode: 2},
		{name: "oauth session rejects --org-id", args: []string{"example.com:443", "--org-id", testFlagOrg}, env: env{cred: oauthOrgCredential}, exitCode: 2},
		{name: "oauth free account needs an organization", args: []string{"example.com:443"}, env: env{cred: oauthFreeAccount}, exitCode: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			_, _, err := runScanCommand(t, newRescan, noCalls(ctrl), tc.env, tc.args)
			require.Error(t, err)
			assert.Equal(t, tc.exitCode, formatter.ExitCode(err))
			if tc.contains != "" {
				assert.Contains(t, err.Error(), tc.contains)
			}
		})
	}
}

func TestRescanCommand_OrganizationResolution(t *testing.T) {
	testCases := []struct {
		name   string
		env    env
		args   []string
		expect string
	}{
		{name: "pat --org-id wins over the stored org", env: patWithStoredOrg, args: []string{"--org-id", testFlagOrg}, expect: testFlagOrg},
		{name: "pat falls back to the stored org", env: patWithStoredOrg, expect: testStoredOrg},
		{name: "oauth session uses its bound org", env: env{cred: oauthOrgCredential}, expect: testBoundOrg},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := scanmocks.NewMockScanService(ctrl)
			m.EXPECT().Rescan(gomock.Any(), appscan.RescanParams{OrgID: orgID(tc.expect), WebProperty: exampleWebProperty}).
				Return(scanResult(trackedScan(false)), nil)

			_, _, err := runScanCommand(t, newRescan, m, tc.env, append([]string{"example.com:443"}, tc.args...))
			require.NoError(t, err)
		})
	}
}

func TestRescanCommand(t *testing.T) {
	testCases := []struct {
		name     string
		service  func(ctrl *gomock.Controller) appscan.Service
		args     []string
		exitCode int
		assert   func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "no wait - prints the scan and how to track it",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Rescan(gomock.Any(), gomock.Any()).Return(scanResult(trackedScan(false, "")), nil)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			args: []string{"https://example.com"},
			assert: func(t *testing.T, stdout, stderr string, _ error) {
				assert.Contains(t, stdout, testScanID)
				assert.Contains(t, stdout, "example.com:443 (web_origin)")
				assert.Contains(t, stdout, "pending")
				assert.Contains(t, stderr, "Track with: censys scan get "+testScanID+" --wait")
			},
		},
		{
			name: "no wait - json payload",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Rescan(gomock.Any(), gomock.Any()).Return(scanResult(trackedScan(false, appscan.TaskStatusScanning)), nil)
				return m
			},
			args: []string{"example.com:443", "--output-format", "json"},
			assert: func(t *testing.T, stdout, _ string, _ error) {
				var got appscan.TrackedScan
				require.NoError(t, json.Unmarshal([]byte(stdout), &got))
				assert.Equal(t, testScanID, got.ID)
				assert.False(t, got.Completed)
				require.NotNil(t, got.Target.WebOrigin)
			},
		},
		{
			name: "wait - success prints only the final scan",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				gomock.InOrder(
					m.EXPECT().Rescan(gomock.Any(), gomock.Any()).Return(scanResult(trackedScan(false)), nil),
					m.EXPECT().Wait(gomock.Any(), appscan.WaitParams{OrgID: orgID(testStoredOrg), ScanID: testScanID, Timeout: mo15m()}).
						Return(scanResult(trackedScan(true, appscan.TaskStatusCompleted, appscan.TaskStatusIgnored)), nil),
				)
				return m
			},
			args: []string{"example.com:443", "--wait", "--output-format", "json"},
			assert: func(t *testing.T, stdout, stderr string, _ error) {
				var got appscan.TrackedScan
				require.NoError(t, json.Unmarshal([]byte(stdout), &got), "stdout must hold exactly one document")
				assert.True(t, got.Completed)
				assert.NotContains(t, stderr, "Track with")
			},
		},
		{
			name: "wait - a scan with no results exits 1 after printing it",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Rescan(gomock.Any(), gomock.Any()).Return(scanResult(trackedScan(false)), nil)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Return(scanResult(trackedScan(true, appscan.TaskStatusRejected, appscan.TaskStatusTimedOut)), nil)
				return m
			},
			args:     []string{"example.com:443", "--wait", "--output-format", "json"},
			exitCode: 1,
			assert: func(t *testing.T, stdout, _ string, err error) {
				var got appscan.TrackedScan
				require.NoError(t, json.Unmarshal([]byte(stdout), &got))
				assert.True(t, got.Completed)
				assert.Contains(t, err.Error(), "completed without results (tasks: rejected, timed_out)")
			},
		},
		{
			name: "wait - timeout prints the last scan, a hint, and exits 124",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Rescan(gomock.Any(), gomock.Any()).Return(scanResult(trackedScan(false)), nil)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Return(scanResult(trackedScan(false, appscan.TaskStatusScanning)), appscan.NewScanWaitTimeoutError(testScanID, time.Minute))
				return m
			},
			args:     []string{"example.com:443", "--wait", "--timeout", "1m"},
			exitCode: 124,
			assert: func(t *testing.T, stdout, stderr string, _ error) {
				assert.Contains(t, stdout, "scanning")
				assert.Contains(t, stderr, "Track with: censys scan get "+testScanID+" --wait")
			},
		},
		{
			name: "wait - interrupt prints the last scan, a note, and exits 130",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Rescan(gomock.Any(), gomock.Any()).Return(scanResult(trackedScan(false)), nil)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Return(scanResult(trackedScan(false, appscan.TaskStatusScanning)), cenclierrors.NewInterruptedError())
				return m
			},
			args:     []string{"example.com:443", "--wait"},
			exitCode: 130,
			assert: func(t *testing.T, stdout, stderr string, _ error) {
				assert.Contains(t, stdout, testScanID)
				assert.Contains(t, stderr, "the scan continues server-side")
				assert.Contains(t, stderr, "Track with")
			},
		},
		{
			name: "wait - a first poll that fails still prints the accepted scan",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Rescan(gomock.Any(), gomock.Any()).Return(scanResult(trackedScan(false)), nil)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Return(appscan.Result{}, cenclierrors.NewCencliError(assertErr("poll failed")))
				return m
			},
			args:     []string{"example.com:443", "--wait", "--output-format", "json"},
			exitCode: 1,
			assert: func(t *testing.T, stdout, stderr string, _ error) {
				var got appscan.TrackedScan
				require.NoError(t, json.Unmarshal([]byte(stdout), &got))
				assert.Equal(t, testScanID, got.ID)
				assert.Contains(t, stderr, "Track with: censys scan get "+testScanID+" --wait")
			},
		},
		{
			name: "uncertain rescan failure is reported, never retried",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Rescan(gomock.Any(), gomock.Any()).
					Return(appscan.Result{}, appscan.NewRescanUncertainError(cenclierrors.NewCencliError(assertErr("502 Bad Gateway")))).Times(1)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			args:     []string{"example.com:443", "--wait"},
			exitCode: 1,
			assert: func(t *testing.T, stdout, _ string, err error) {
				assert.Empty(t, stdout)
				assert.Contains(t, err.Error(), "may have been charged")
			},
		},
		{
			name: "interrupted rescan warns it may have been charged",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Rescan(gomock.Any(), gomock.Any()).Return(appscan.Result{}, cenclierrors.NewInterruptedError())
				return m
			},
			args:     []string{"example.com:443"},
			exitCode: 130,
			assert: func(t *testing.T, _, stderr string, _ error) {
				assert.Contains(t, stderr, "may still have been accepted and charged")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stdout, stderr, err := runScanCommand(t, newRescan, tc.service(ctrl), patWithStoredOrg, tc.args)
			assert.Equal(t, tc.exitCode, formatter.ExitCode(err), "err: %v", err)
			tc.assert(t, stdout, stderr, err)
		})
	}
}
