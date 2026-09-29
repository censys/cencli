package scan

import (
	"context"
	"testing"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
)

func TestScanService_Get(t *testing.T) {
	testCases := []struct {
		name   string
		params GetParams
		client func(m *mocks.MockClient)
		assert func(t *testing.T, res Result, err cenclierrors.CencliError)
	}{
		{
			name:   "success - a rejected scan is a successful read",
			params: GetParams{OrgID: testOrgID, ScanID: testScanID},
			client: func(m *mocks.MockClient) {
				result := trackedScanResult(true, task(components.TrackedScanTaskStatusRejected))
				result.Data.Target = &components.TrackedScanScanTarget{
					ServiceID: &components.ServiceID{IP: strPtr("1.1.1.1")},
				}
				m.EXPECT().GetTrackedScan(gomock.Any(), testOrgUUID, testScanID).Return(result, nil)
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Scan)
				assert.True(t, res.Scan.Completed)
				assert.False(t, res.Scan.Succeeded())
				require.NotNil(t, res.Scan.Target.ServiceID, "non-web targets pass through")
				assert.Equal(t, "2026-09-29T14:02:11Z", *res.Scan.CreateTime)
			},
		},
		{
			name:   "missing organization is refused without a request",
			params: GetParams{ScanID: testScanID},
			assert: func(t *testing.T, _ Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name:   "non-UUID scan ID is refused without a request",
			params: GetParams{OrgID: testOrgID, ScanID: "not-a-uuid"},
			assert: func(t *testing.T, _ Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.True(t, err.ShouldPrintUsage())
				assert.Equal(t, "Invalid Scan ID", err.Title())
			},
		},
		{
			name:   "empty scan ID is refused without a request",
			params: GetParams{OrgID: testOrgID},
			assert: func(t *testing.T, _ Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "a scan ID is required")
			},
		},
		{
			name:   "client error",
			params: GetParams{OrgID: testOrgID, ScanID: testScanID},
			client: func(m *mocks.MockClient) {
				m.EXPECT().GetTrackedScan(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(client.Result[components.TrackedScan]{}, structuredErr(404))
			},
			assert: func(t *testing.T, _ Result, err cenclierrors.CencliError) {
				require.Error(t, err)
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
			res, err := New(m).Get(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

func TestMapTrackedScan_NilFields(t *testing.T) {
	scan := mapTrackedScan(&components.TrackedScan{Tasks: []components.TrackedScanTask{{}}})
	assert.Empty(t, scan.ID)
	assert.False(t, scan.Completed)
	assert.Nil(t, scan.CreateTime)
	assert.Nil(t, scan.Target)
	require.Len(t, scan.Tasks, 1)
	assert.Empty(t, scan.Tasks[0].Status)
}
