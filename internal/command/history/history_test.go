package history

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	historymocks "github.com/censys/cencli/gen/app/history/mocks"
	historyapp "github.com/censys/cencli/internal/app/history"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/censys-sdk-go/models/components"
)

func TestHistoryCommand(t *testing.T) {
	// Common test data
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC)
	eventTime1Str := "2025-01-02T12:00:00Z"
	eventTime2Str := "2025-01-05T12:00:00Z"

	testCases := []struct {
		name       string
		historySvc func(ctrl *gomock.Controller) historyapp.Service
		args       []string
		assert     func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "success - host history output",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				ms := historymocks.NewMockHistoryService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")

				events := []*components.HostTimelineEvent{
					{EventTime: &eventTime1Str},
					{EventTime: &eventTime2Str},
				}

				result := historyapp.HostHistoryResult{
					Meta:   &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Events: events,
				}

				ms.EXPECT().GetHostHistory(
					gomock.Any(),
					mo.None[identifiers.OrganizationID](),
					hostID,
					gomock.Any(),
					gomock.Any(),
				).Return(result, nil)

				return ms
			},
			args: []string{"8.8.8.8", "--start", "2025-01-01T00:00:00Z", "--end", "2025-01-08T00:00:00Z"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)

				var result historyapp.HostHistoryResult
				jsonErr := json.Unmarshal([]byte(stdout), &result.Events)
				require.NoError(t, jsonErr)
				require.Equal(t, 2, len(result.Events))
			},
		},
		{
			name: "success - certificate history output",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				ms := historymocks.NewMockHistoryService(ctrl)
				certID, _ := assets.NewCertificateFingerprint("a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456")

				ranges := []*components.HostObservationRange{
					{
						IP:                "1.1.1.1",
						Port:              443,
						TransportProtocol: "tcp",
						Protocols:         []string{"https"},
						StartTime:         startTime,
						EndTime:           endTime,
					},
				}

				result := historyapp.CertificateHistoryResult{
					Meta:   &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Ranges: ranges,
				}

				ms.EXPECT().GetCertificateHistory(
					gomock.Any(),
					mo.None[identifiers.OrganizationID](),
					certID,
					gomock.Any(),
					gomock.Any(),
				).Return(result, nil)

				return ms
			},
			args: []string{"a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456", "--start", "2025-01-01T00:00:00Z", "--end", "2025-01-08T00:00:00Z"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)

				var result historyapp.CertificateHistoryResult
				jsonErr := json.Unmarshal([]byte(stdout), &result.Ranges)
				require.NoError(t, jsonErr)
				require.Equal(t, 1, len(result.Ranges))
			},
		},
		{
			name: "success - web property history output",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				ms := historymocks.NewMockHistoryService(ctrl)
				webPropID, _ := assets.NewWebPropertyID("example.com:443", assets.DefaultWebPropertyPort)

				snapshots := []*historyapp.WebPropertySnapshot{
					{
						Time:   startTime,
						Data:   &components.Webproperty{},
						Exists: true,
					},
					{
						Time:   startTime.AddDate(0, 0, 1),
						Data:   &components.Webproperty{},
						Exists: true,
					},
				}

				result := historyapp.WebPropertyHistoryResult{
					Meta:      &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Snapshots: snapshots,
				}

				ms.EXPECT().GetWebPropertyHistory(
					gomock.Any(),
					mo.None[identifiers.OrganizationID](),
					webPropID,
					gomock.Any(),
					gomock.Any(),
				).Return(result, nil)

				return ms
			},
			args: []string{"example.com:443", "--start", "2025-01-01T00:00:00Z", "--end", "2025-01-08T00:00:00Z"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)

				var result historyapp.WebPropertyHistoryResult
				jsonErr := json.Unmarshal([]byte(stdout), &result.Snapshots)
				require.NoError(t, jsonErr)
				require.Equal(t, 2, len(result.Snapshots))
			},
		},
		{
			name: "success - certificate history with multiple ranges",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				ms := historymocks.NewMockHistoryService(ctrl)
				certID, _ := assets.NewCertificateFingerprint("a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456")

				ranges := []*components.HostObservationRange{
					{
						IP:                "1.1.1.1",
						Port:              443,
						TransportProtocol: "tcp",
						Protocols:         []string{"https"},
						StartTime:         startTime,
						EndTime:           endTime,
					},
					{
						IP:                "2.2.2.2",
						Port:              443,
						TransportProtocol: "tcp",
						Protocols:         []string{"https"},
						StartTime:         startTime.AddDate(0, 0, 1),
						EndTime:           endTime,
					},
				}

				result := historyapp.CertificateHistoryResult{
					Meta:   &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Ranges: ranges,
				}

				ms.EXPECT().GetCertificateHistory(
					gomock.Any(),
					mo.None[identifiers.OrganizationID](),
					certID,
					gomock.Any(),
					gomock.Any(),
				).Return(result, nil)

				return ms
			},
			args: []string{"a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456", "--start", "2025-01-01T00:00:00Z", "--end", "2025-01-08T00:00:00Z"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)

				var result historyapp.CertificateHistoryResult
				jsonErr := json.Unmarshal([]byte(stdout), &result.Ranges)
				require.NoError(t, jsonErr)
				require.Equal(t, 2, len(result.Ranges))
			},
		},
		{
			name: "success - with duration flag",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				ms := historymocks.NewMockHistoryService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")

				result := historyapp.HostHistoryResult{
					Meta:   &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Events: []*components.HostTimelineEvent{},
				}

				ms.EXPECT().GetHostHistory(
					gomock.Any(),
					mo.None[identifiers.OrganizationID](),
					hostID,
					gomock.Any(),
					gomock.Any(),
				).Return(result, nil)

				return ms
			},
			args: []string{"8.8.8.8", "--duration", "30d"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				var result historyapp.HostHistoryResult
				jsonErr := json.Unmarshal([]byte(stdout), &result.Events)
				require.NoError(t, jsonErr)
			},
		},
		{
			name: "success - with org-id",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				ms := historymocks.NewMockHistoryService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")
				uuidVal, _ := uuid.Parse("a0000000-0000-0000-0000-000000000000")
				orgID := identifiers.NewOrganizationID(uuidVal)

				result := historyapp.HostHistoryResult{
					Meta:   &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Events: []*components.HostTimelineEvent{},
				}

				ms.EXPECT().GetHostHistory(
					gomock.Any(),
					mo.Some(orgID),
					hostID,
					gomock.Any(),
					gomock.Any(),
				).Return(result, nil)

				return ms
			},
			args: []string{"8.8.8.8", "--org-id", "a0000000-0000-0000-0000-000000000000", "--duration", "7d"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				var result historyapp.HostHistoryResult
				jsonErr := json.Unmarshal([]byte(stdout), &result.Events)
				require.NoError(t, jsonErr)
			},
		},
		{
			name: "success - empty results",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				ms := historymocks.NewMockHistoryService(ctrl)
				hostID, _ := assets.NewHostID("8.8.8.8")

				result := historyapp.HostHistoryResult{
					Meta:   &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
					Events: []*components.HostTimelineEvent{},
				}

				ms.EXPECT().GetHostHistory(
					gomock.Any(),
					mo.None[identifiers.OrganizationID](),
					hostID,
					gomock.Any(),
					gomock.Any(),
				).Return(result, nil)

				return ms
			},
			args: []string{"8.8.8.8", "--start", "2025-01-01T00:00:00Z", "--end", "2025-01-08T00:00:00Z"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				var result historyapp.HostHistoryResult
				jsonErr := json.Unmarshal([]byte(stdout), &result.Events)
				require.NoError(t, jsonErr)
				require.Equal(t, 0, len(result.Events))
			},
		},
		{
			name: "error - no argument",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				return historymocks.NewMockHistoryService(ctrl)
			},
			args: []string{},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
			},
		},
		{
			name: "error - multiple assets",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				return historymocks.NewMockHistoryService(ctrl)
			},
			args: []string{"8.8.8.8,1.1.1.1", "--duration", "7d"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Equal(t, assets.NewTooManyAssetsError(2, 1), err)
			},
		},
		{
			name: "error - invalid asset",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				return historymocks.NewMockHistoryService(ctrl)
			},
			args: []string{"invalid-asset", "--duration", "7d"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "unable to infer asset type")
			},
		},
		{
			name: "error - invalid start time",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				return historymocks.NewMockHistoryService(ctrl)
			},
			args: []string{"8.8.8.8", "--start", "invalid-time"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid timestamp")
			},
		},
		{
			name: "error - invalid end time",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				return historymocks.NewMockHistoryService(ctrl)
			},
			args: []string{"8.8.8.8", "--end", "invalid-time"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid timestamp")
			},
		},
		{
			name: "error - invalid duration",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				return historymocks.NewMockHistoryService(ctrl)
			},
			args: []string{"8.8.8.8", "--duration", "invalid"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "error - end before start",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				return historymocks.NewMockHistoryService(ctrl)
			},
			args: []string{"8.8.8.8", "--start", "2025-01-08T00:00:00Z", "--end", "2025-01-01T00:00:00Z"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "end time must be after start time")
			},
		},
		{
			name: "help message",
			historySvc: func(ctrl *gomock.Controller) historyapp.Service {
				return historymocks.NewMockHistoryService(ctrl)
			},
			args: []string{"--help"},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				require.NoError(t, err)
				require.Contains(t, stdout, "Usage:")
				require.Contains(t, stdout, "history <asset>")
				require.Contains(t, stdout, "start")
				require.Contains(t, stdout, "end")
				require.Contains(t, stdout, "duration")
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
			historySvc := tc.historySvc(ctrl)
			cmdContext := command.NewCommandContext(cfg, nil, command.WithHistoryService(historySvc))
			rootCmd, err := command.RootCommandToCobra(NewHistoryCommand(cmdContext))
			require.NoError(t, err)

			rootCmd.SetArgs(tc.args)

			cmdErr := rootCmd.Execute()
			tc.assert(t, stdout.String(), stderr.String(), cmdErr)
		})
	}
}

func TestHistoryCommand_PartialError(t *testing.T) {
	t.Run("host history - prints partial error to stderr after rendering data", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ms := historymocks.NewMockHistoryService(ctrl)
		hostID, _ := assets.NewHostID("8.8.8.8")

		eventTime1Str := "2025-01-02T12:00:00Z"
		events := []*components.HostTimelineEvent{
			{EventTime: &eventTime1Str},
		}

		// Service returns partial results with error wrapped in NewPartialError
		baseErr := cenclierrors.NewCencliError(errors.New("Page 2 failed"))
		result := historyapp.HostHistoryResult{
			Meta:         &responsemeta.ResponseMeta{Method: "GET", URL: "https://127.0.0.1", Status: 200},
			Events:       events,
			PartialError: cenclierrors.ToPartialError(baseErr),
		}

		ms.EXPECT().GetHostHistory(
			gomock.Any(),
			mo.None[identifiers.OrganizationID](),
			hostID,
			gomock.Any(),
			gomock.Any(),
		).Return(result, nil)

		tempDir := t.TempDir()
		viper.Reset()
		cfg, err := config.New(tempDir)
		require.NoError(t, err)

		cmdContext := command.NewCommandContext(cfg, nil, command.WithHistoryService(ms))
		historyCmd := NewHistoryCommand(cmdContext)
		rootCmd, err := command.RootCommandToCobra(historyCmd)
		require.NoError(t, err)

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		rootCmd.SetOut(stdout)
		rootCmd.SetErr(stderr)
		formatter.Stdout = stdout
		formatter.Stderr = stderr

		rootCmd.SetArgs([]string{"8.8.8.8", "--start", "2025-01-01T00:00:00Z", "--end", "2025-01-08T00:00:00Z"})
		cmdErr := rootCmd.Execute()

		require.NoError(t, cmdErr)
		require.Contains(t, stdout.String(), "2025-01-02", "should render data to stdout")
		require.Contains(t, stderr.String(), "(partial data)", "should indicate partial results in stderr")
		require.Contains(t, stderr.String(), "Page 2 failed", "should print partial error to stderr")
		require.Contains(t, stderr.String(), "some data was successfully retrieved", "should include partial error message")
	})
}
