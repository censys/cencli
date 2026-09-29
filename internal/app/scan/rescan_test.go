package scan

import (
	"context"
	"errors"
	"testing"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

func structuredErr(status int64) client.ClientError {
	return structuredErrWithDetail(status, "boom")
}

func structuredErrWithDetail(status int64, detail string) client.ClientError {
	return client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status})
}

func TestScanService_Rescan(t *testing.T) {
	webProperty := assets.WebPropertyID{Hostname: "example.com", Port: 443}

	testCases := []struct {
		name   string
		orgID  identifiers.OrganizationID
		ctx    func() context.Context
		client func(m *mocks.MockClient)
		assert func(t *testing.T, res Result, err cenclierrors.CencliError)
	}{
		{
			name:  "success maps the tracked scan",
			orgID: testOrgID,
			client: func(m *mocks.MockClient) {
				m.EXPECT().CreateWebPropertyRescan(gomock.Any(), testOrgUUID, "example.com", 443).
					Return(trackedScanResult(false, task(components.TrackedScanTaskStatusScanning)), nil).Times(1)
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.NotNil(t, res.Scan)
				assert.Equal(t, testScanID, res.Scan.ID)
				assert.False(t, res.Scan.Completed)
				require.Len(t, res.Scan.Tasks, 1)
				assert.Equal(t, TaskStatusScanning, res.Scan.Tasks[0].Status)
			},
		},
		{
			name:  "missing organization is refused without a request",
			orgID: identifiers.OrganizationID{},
			assert: func(t *testing.T, _ Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name:  "4xx is returned as the API error",
			orgID: testOrgID,
			client: func(m *mocks.MockClient) {
				m.EXPECT().CreateWebPropertyRescan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.TrackedScan]{}, structuredErr(422)).Times(1)
			},
			assert: func(t *testing.T, _ Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				var uncertain *rescanUncertainError
				assert.False(t, errors.As(err, &uncertain), "a 4xx proves the API rejected the request")
			},
		},
		{
			name:  "feature flag off is reported as not enabled",
			orgID: testOrgID,
			client: func(m *mocks.MockClient) {
				m.EXPECT().CreateWebPropertyRescan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.TrackedScan]{}, structuredErrWithDetail(409, "feature is not enabled")).Times(1)
			},
			assert: func(t *testing.T, _ Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				var uncertain *rescanUncertainError
				assert.False(t, errors.As(err, &uncertain), "the flag check runs before a scan is created")
				assert.Equal(t, "Feature Not Enabled", err.Title())
				assert.False(t, err.ShouldPrintUsage())
			},
		},
		{
			name:  "other conflict is reported as uncertain",
			orgID: testOrgID,
			client: func(m *mocks.MockClient) {
				m.EXPECT().CreateWebPropertyRescan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.TrackedScan]{}, structuredErrWithDetail(409, "resource already exists")).Times(1)
			},
			assert: func(t *testing.T, _ Result, err cenclierrors.CencliError) {
				var uncertain *rescanUncertainError
				require.ErrorAs(t, err, &uncertain)
			},
		},
		{
			name:  "5xx is reported as uncertain",
			orgID: testOrgID,
			client: func(m *mocks.MockClient) {
				m.EXPECT().CreateWebPropertyRescan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.TrackedScan]{}, structuredErr(502)).Times(1)
			},
			assert: func(t *testing.T, _ Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				var uncertain *rescanUncertainError
				require.ErrorAs(t, err, &uncertain)
				assert.Contains(t, err.Error(), "10 credits may have been charged")
				assert.Contains(t, err.Error(), "censys org credits")
			},
		},
		{
			name:  "transport error without a status is reported as uncertain",
			orgID: testOrgID,
			client: func(m *mocks.MockClient) {
				m.EXPECT().CreateWebPropertyRescan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.TrackedScan]{}, client.NewClientError(errors.New("connection reset by peer"))).Times(1)
			},
			assert: func(t *testing.T, _ Result, err cenclierrors.CencliError) {
				var uncertain *rescanUncertainError
				require.ErrorAs(t, err, &uncertain)
				assert.Contains(t, err.Error(), "connection reset by peer")
			},
		},
		{
			name:  "interruption during the request is reported as an interruption",
			orgID: testOrgID,
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			client: func(m *mocks.MockClient) {
				m.EXPECT().CreateWebPropertyRescan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.TrackedScan]{}, client.NewClientError(context.Canceled)).Times(1)
			},
			assert: func(t *testing.T, _ Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.True(t, cenclierrors.IsInterrupted(err))
			},
		},
		{
			name:  "empty envelope keeps the metadata and is reported as uncertain",
			orgID: testOrgID,
			client: func(m *mocks.MockClient) {
				res := trackedScanResult(false)
				res.Data = nil
				m.EXPECT().CreateWebPropertyRescan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(res, nil)
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				var uncertain *rescanUncertainError
				require.ErrorAs(t, err, &uncertain)
				assert.Contains(t, err.Error(), "10 credits may have been charged")
				assert.NotNil(t, res.Meta)
				assert.Nil(t, res.Scan)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks.NewMockClient(ctrl)
			if tc.client != nil {
				tc.client(m)
			}
			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}
			res, err := New(m).Rescan(ctx, RescanParams{OrgID: tc.orgID, WebProperty: webProperty})
			tc.assert(t, res, err)
		})
	}
}
