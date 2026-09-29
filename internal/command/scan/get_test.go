package scan

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	scanmocks "github.com/censys/cencli/gen/app/scan/mocks"
	appscan "github.com/censys/cencli/internal/app/scan"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
)

func newGet(c *command.Context) command.Command { return NewGetCommand(c) }

func TestGetCommand_Rejections(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		env      env
		exitCode int
		contains string
	}{
		{name: "non-UUID scan ID", args: []string{"not-a-uuid"}, env: patWithStoredOrg, exitCode: 2, contains: "is not a valid UUID"},
		{name: "timeout without wait", args: []string{testScanID, "--timeout", "5m"}, env: patWithStoredOrg, exitCode: 2, contains: "--timeout only applies while polling"},
		{name: "pat with no organization", args: []string{testScanID}, env: env{cred: patCredential}, exitCode: 2},
		{name: "oauth session rejects --org-id", args: []string{testScanID, "--org-id", testFlagOrg}, env: env{cred: oauthOrgCredential}, exitCode: 2},
		{name: "oauth free account needs an organization", args: []string{testScanID}, env: env{cred: oauthFreeAccount}, exitCode: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			_, _, err := runScanCommand(t, newGet, noCalls(ctrl), tc.env, tc.args)
			require.Error(t, err)
			assert.Equal(t, tc.exitCode, formatter.ExitCode(err))
			if tc.contains != "" {
				assert.Contains(t, err.Error(), tc.contains)
			}
		})
	}
}

func TestGetCommand_OrganizationResolution(t *testing.T) {
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
			m.EXPECT().Get(gomock.Any(), appscan.GetParams{OrgID: orgID(tc.expect), ScanID: testScanID}).
				Return(scanResult(trackedScan(true, appscan.TaskStatusCompleted)), nil)

			_, _, err := runScanCommand(t, newGet, m, tc.env, append([]string{testScanID}, tc.args...))
			require.NoError(t, err)
		})
	}
}

func TestGetCommand(t *testing.T) {
	testCases := []struct {
		name     string
		service  func(ctrl *gomock.Controller) appscan.Service
		args     []string
		exitCode int
		assert   func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "reading a rejected scan is a successful read",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Return(scanResult(trackedScan(true, appscan.TaskStatusRejected)), nil)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			args: []string{testScanID},
			assert: func(t *testing.T, stdout, _ string, _ error) {
				assert.Contains(t, stdout, "rejected")
				assert.Contains(t, stdout, "yes")
			},
		},
		{
			name: "a non-web target is shown as returned",
			service: func(ctrl *gomock.Controller) appscan.Service {
				s := trackedScan(false, appscan.TaskStatusScanning)
				ip, port, protocol := "1.1.1.1", 443, "HTTP"
				s.Target = &components.TrackedScanScanTarget{ServiceID: &components.ServiceID{IP: &ip, Port: &port, Protocol: &protocol}}
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Return(scanResult(s), nil)
				return m
			},
			args: []string{testScanID},
			assert: func(t *testing.T, stdout, _ string, _ error) {
				assert.Contains(t, stdout, "1.1.1.1:443/HTTP (service_id)")
			},
		},
		{
			name: "json payload",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Return(scanResult(trackedScan(true, appscan.TaskStatusCompleted)), nil)
				return m
			},
			args: []string{testScanID, "--output-format", "json"},
			assert: func(t *testing.T, stdout, _ string, _ error) {
				var got appscan.TrackedScan
				require.NoError(t, json.Unmarshal([]byte(stdout), &got))
				assert.Equal(t, testScanID, got.ID)
				require.Len(t, got.Tasks, 1)
				assert.Equal(t, appscan.TaskStatusCompleted, got.Tasks[0].Status)
			},
		},
		{
			name: "wait - success",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().Wait(gomock.Any(), appscan.WaitParams{OrgID: orgID(testStoredOrg), ScanID: testScanID}).
					Return(scanResult(trackedScan(true, appscan.TaskStatusScanned)), nil)
				return m
			},
			args: []string{testScanID, "--wait", "--timeout", "0"},
			assert: func(t *testing.T, stdout, _ string, _ error) {
				assert.Contains(t, stdout, "scanned")
			},
		},
		{
			name: "wait - no results exits 1 after printing the scan",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).Return(scanResult(trackedScan(true)), nil)
				return m
			},
			args:     []string{testScanID, "--wait"},
			exitCode: 1,
			assert: func(t *testing.T, stdout, _ string, err error) {
				assert.Contains(t, stdout, "No tasks yet.")
				assert.Contains(t, err.Error(), "completed without running any tasks")
			},
		},
		{
			name: "wait - timeout prints the last scan and exits 124",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Return(scanResult(trackedScan(false, appscan.TaskStatusScanning)), appscan.NewScanWaitTimeoutError(testScanID, time.Minute))
				return m
			},
			args:     []string{testScanID, "--wait", "--timeout", "1m", "--output-format", "json"},
			exitCode: 124,
			assert: func(t *testing.T, stdout, stderr string, _ error) {
				var got appscan.TrackedScan
				require.NoError(t, json.Unmarshal([]byte(stdout), &got), "the payload survives the timeout")
				assert.False(t, got.Completed)
				assert.Contains(t, stderr, "Track with")
			},
		},
		{
			name: "wait - interrupt exits 130",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Return(scanResult(trackedScan(false)), cenclierrors.NewInterruptedError())
				return m
			},
			args:     []string{testScanID, "--wait"},
			exitCode: 130,
			assert: func(t *testing.T, _, stderr string, _ error) {
				assert.Contains(t, stderr, "the scan continues server-side")
			},
		},
		{
			name: "wait - a failed poll after a good one prints the last scan and a hint",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Return(scanResult(trackedScan(false, appscan.TaskStatusScanning)), cenclierrors.NewCencliError(assertErr("502 Bad Gateway")))
				return m
			},
			args:     []string{testScanID, "--wait"},
			exitCode: 1,
			assert: func(t *testing.T, stdout, stderr string, _ error) {
				assert.Contains(t, stdout, "scanning")
				assert.Contains(t, stderr, "Track with: censys scan get "+testScanID+" --wait")
			},
		},
		{
			name: "wait - an error before any poll prints nothing",
			service: func(ctrl *gomock.Controller) appscan.Service {
				m := scanmocks.NewMockScanService(ctrl)
				m.EXPECT().Wait(gomock.Any(), gomock.Any()).Return(appscan.Result{}, cenclierrors.NewCencliError(assertErr("not found")))
				return m
			},
			args:     []string{testScanID, "--wait"},
			exitCode: 1,
			assert: func(t *testing.T, stdout, stderr string, _ error) {
				assert.Empty(t, stdout)
				assert.NotContains(t, stderr, "Track with")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stdout, stderr, err := runScanCommand(t, newGet, tc.service(ctrl), patWithStoredOrg, tc.args)
			assert.Equal(t, tc.exitCode, formatter.ExitCode(err), "err: %v", err)
			tc.assert(t, stdout, stderr, err)
		})
	}
}

func TestTargetLabel(t *testing.T) {
	host, ip, port := "example.com", "2001:db8::1", 443
	testCases := []struct {
		name   string
		target *components.TrackedScanScanTarget
		expect string
	}{
		{name: "nil", expect: "-"},
		{name: "web origin", target: &components.TrackedScanScanTarget{WebOrigin: &components.WebOrigin{Hostname: &host, Port: &port}}, expect: "example.com:443 (web_origin)"},
		{name: "ipv6 host port", target: &components.TrackedScanScanTarget{HostPort: &components.TrackedScanScanTargetHostPort{IP: &ip, Port: &port}}, expect: "[2001:db8::1]:443 (host_port)"},
		{name: "hostname port without port", target: &components.TrackedScanScanTarget{HostnamePort: &components.TrackedScanScanTargetHostnamePort{Hostname: &host}}, expect: "example.com (hostname_port)"},
		{name: "empty target", target: &components.TrackedScanScanTarget{}, expect: "-"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, targetLabel(tc.target))
		})
	}
}
