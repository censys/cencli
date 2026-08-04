package tags

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

// bulkOrgUUID stands in for an --org-id override on a bulk submit.
var bulkOrgUUID = uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")

// Fixed timestamps for the bulk-delete time filters, ordered so before > after
// (the only combination the service accepts).
var (
	bulkAfterTime  = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bulkBeforeTime = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
)

// deleteOperationResult is operationResult's bulk_delete twin: TotalCount stays
// zero because the API only sets it once a bulk delete completes.
func deleteOperationResult(status components.TagOperationStatus) client.Result[components.TagOperation] {
	return client.Result[components.TagOperation]{
		Metadata: okMeta(),
		Data: &components.TagOperation{
			ID:      testOpUUID,
			TagID:   testTagUUID,
			TagName: "my-tag",
			Type:    components.TagOperationTypeBulkDelete,
			Status:  status,
		},
	}
}

// bulkNameLookup expects the one ListTags call that resolves a tag name.
func bulkNameLookup(m *mocks.MockClient, name string) {
	m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{
		Name: mo.Some(name), PageSize: mo.Some(int64(1)),
	}).Return(client.Result[components.TagsList]{
		Metadata: okMeta(),
		Data:     &components.TagsList{Tags: []components.Tag{{ID: testTagUUID, Name: name}}, TotalSize: 1},
	}, nil)
}

func TestTagsService_BulkAssign(t *testing.T) {
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params BulkAssignParams
		assert func(t *testing.T, res BulkAssignResult, err cenclierrors.CencliError)
	}{
		{
			name: "UUID tag submits without a lookup and returns the operation",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				// The durable guard: a UUID must cost zero lookup requests.
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().BulkCreateTagAssignments(gomock.Any(), client.BulkCreateTagAssignmentsRequest{
					TagID: testTagUUID,
					Query: "host.services.port: 22",
				}).Return(operationResult(components.TagOperationStatusPending, 0), nil)
				return m
			},
			params: BulkAssignParams{
				TagID: identifiers.NewTagID(testTagUUID),
				Query: "host.services.port: 22",
			},
			assert: func(t *testing.T, res BulkAssignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, testOpUUID, res.Operation.ID)
				require.Equal(t, "pending", res.Operation.Status)
				require.Equal(t, "bulk_create", res.Operation.Type)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "name is resolved to a UUID before submitting",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				bulkNameLookup(m, "my-tag")
				m.EXPECT().BulkCreateTagAssignments(gomock.Any(), client.BulkCreateTagAssignmentsRequest{
					TagID: testTagUUID,
					Query: "host.ip: 1.1.1.1",
				}).Return(operationResult(components.TagOperationStatusRunning, 0), nil)
				return m
			},
			params: BulkAssignParams{
				TagID: identifiers.NewTagID("my-tag"),
				Query: "host.ip: 1.1.1.1",
			},
			assert: func(t *testing.T, res BulkAssignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, testOpUUID, res.Operation.ID)
			},
		},
		{
			name: "max assets is sent when set",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkCreateTagAssignments(gomock.Any(), client.BulkCreateTagAssignmentsRequest{
					TagID:     testTagUUID,
					Query:     "host.ip: 1.1.1.1",
					MaxAssets: mo.Some(int64(500)),
				}).Return(operationResult(components.TagOperationStatusPending, 0), nil)
				return m
			},
			params: BulkAssignParams{
				TagID:     identifiers.NewTagID(testTagUUID),
				Query:     "host.ip: 1.1.1.1",
				MaxAssets: mo.Some(int64(500)),
			},
			assert: func(t *testing.T, res BulkAssignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
			},
		},
		{
			name: "zero max assets is passed through as no explicit cap",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkCreateTagAssignments(gomock.Any(), client.BulkCreateTagAssignmentsRequest{
					TagID:     testTagUUID,
					Query:     "host.ip: 1.1.1.1",
					MaxAssets: mo.Some(int64(0)),
				}).Return(operationResult(components.TagOperationStatusPending, 0), nil)
				return m
			},
			params: BulkAssignParams{
				TagID:     identifiers.NewTagID(testTagUUID),
				Query:     "host.ip: 1.1.1.1",
				MaxAssets: mo.Some(int64(0)),
			},
			assert: func(t *testing.T, res BulkAssignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
			},
		},
		{
			name: "query is trimmed before it is sent",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkCreateTagAssignments(gomock.Any(), client.BulkCreateTagAssignmentsRequest{
					TagID: testTagUUID,
					Query: "host.ip: 1.1.1.1",
				}).Return(operationResult(components.TagOperationStatusPending, 0), nil)
				return m
			},
			params: BulkAssignParams{
				TagID: identifiers.NewTagID(testTagUUID),
				Query: "  host.ip: 1.1.1.1  ",
			},
			assert: func(t *testing.T, res BulkAssignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
			},
		},
		{
			name: "org id is threaded through to the request",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkCreateTagAssignments(gomock.Any(), client.BulkCreateTagAssignmentsRequest{
					OrgID: mo.Some(bulkOrgUUID.String()),
					TagID: testTagUUID,
					Query: "host.ip: 1.1.1.1",
				}).Return(operationResult(components.TagOperationStatusPending, 0), nil)
				return m
			},
			params: BulkAssignParams{
				OrgID: mo.Some(identifiers.NewOrganizationID(bulkOrgUUID)),
				TagID: identifiers.NewTagID(testTagUUID),
				Query: "host.ip: 1.1.1.1",
			},
			assert: func(t *testing.T, res BulkAssignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
			},
		},
		{
			name: "empty query is rejected without touching the API",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkCreateTagAssignments(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: BulkAssignParams{
				TagID: identifiers.NewTagID(testTagUUID),
				Query: "   ",
			},
			assert: func(t *testing.T, _ BulkAssignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--query must not be empty")
			},
		},
		{
			name: "unknown tag name fails before submitting",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Return(client.Result[components.TagsList]{
					Metadata: okMeta(),
					Data:     &components.TagsList{Tags: []components.Tag{}, TotalSize: 0},
				}, nil)
				m.EXPECT().BulkCreateTagAssignments(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: BulkAssignParams{
				TagID: identifiers.NewTagID("ghost"),
				Query: "host.ip: 1.1.1.1",
			},
			assert: func(t *testing.T, _ BulkAssignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "ghost")
			},
		},
		{
			name: "client error is returned as-is",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkCreateTagAssignments(gomock.Any(), gomock.Any()).Return(
					client.Result[components.TagOperation]{},
					clientStructuredError("Permission denied", 403),
				)
				expectNoTagOfThatName(m)
				return m
			},
			params: BulkAssignParams{
				TagID: identifiers.NewTagID(testTagUUID),
				Query: "host.ip: 1.1.1.1",
			},
			assert: func(t *testing.T, _ BulkAssignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "Permission denied")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := &tagsService{client: tc.client(ctrl)}
			res, err := svc.BulkAssign(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

func TestTagsService_BulkUnassign(t *testing.T) {
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params BulkUnassignParams
		assert func(t *testing.T, res BulkUnassignResult, err cenclierrors.CencliError)
	}{
		{
			// No filters means every assignment: the request must carry neither
			// timestamp, since a stray one would silently narrow the wipe.
			name: "no filters submits an unfiltered removal without a lookup",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().BulkDeleteTagAssignments(gomock.Any(), client.BulkDeleteTagAssignmentsRequest{
					TagID: testTagUUID,
				}).Return(deleteOperationResult(components.TagOperationStatusPending), nil)
				return m
			},
			params: BulkUnassignParams{TagID: identifiers.NewTagID(testTagUUID)},
			assert: func(t *testing.T, res BulkUnassignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, testOpUUID, res.Operation.ID)
				require.Equal(t, "pending", res.Operation.Status)
				require.Equal(t, "bulk_delete", res.Operation.Type)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "name is resolved to a UUID before submitting",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				bulkNameLookup(m, "my-tag")
				m.EXPECT().BulkDeleteTagAssignments(gomock.Any(), client.BulkDeleteTagAssignmentsRequest{
					TagID: testTagUUID,
				}).Return(deleteOperationResult(components.TagOperationStatusRunning), nil)
				return m
			},
			params: BulkUnassignParams{TagID: identifiers.NewTagID("my-tag")},
			assert: func(t *testing.T, res BulkUnassignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, testOpUUID, res.Operation.ID)
			},
		},
		{
			name: "created-before alone is sent alone",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkDeleteTagAssignments(gomock.Any(), client.BulkDeleteTagAssignmentsRequest{
					TagID:         testTagUUID,
					CreatedBefore: mo.Some(bulkBeforeTime),
				}).Return(deleteOperationResult(components.TagOperationStatusPending), nil)
				return m
			},
			params: BulkUnassignParams{
				TagID:         identifiers.NewTagID(testTagUUID),
				CreatedBefore: mo.Some(bulkBeforeTime),
			},
			assert: func(t *testing.T, _ BulkUnassignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
			},
		},
		{
			name: "created-after alone is sent alone",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkDeleteTagAssignments(gomock.Any(), client.BulkDeleteTagAssignmentsRequest{
					TagID:        testTagUUID,
					CreatedAfter: mo.Some(bulkAfterTime),
				}).Return(deleteOperationResult(components.TagOperationStatusPending), nil)
				return m
			},
			params: BulkUnassignParams{
				TagID:        identifiers.NewTagID(testTagUUID),
				CreatedAfter: mo.Some(bulkAfterTime),
			},
			assert: func(t *testing.T, _ BulkUnassignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
			},
		},
		{
			name: "both filters bound the window",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkDeleteTagAssignments(gomock.Any(), client.BulkDeleteTagAssignmentsRequest{
					TagID:         testTagUUID,
					CreatedBefore: mo.Some(bulkBeforeTime),
					CreatedAfter:  mo.Some(bulkAfterTime),
				}).Return(deleteOperationResult(components.TagOperationStatusPending), nil)
				return m
			},
			params: BulkUnassignParams{
				TagID:         identifiers.NewTagID(testTagUUID),
				CreatedBefore: mo.Some(bulkBeforeTime),
				CreatedAfter:  mo.Some(bulkAfterTime),
			},
			assert: func(t *testing.T, _ BulkUnassignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
			},
		},
		{
			name: "org id is threaded through to the request",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkDeleteTagAssignments(gomock.Any(), client.BulkDeleteTagAssignmentsRequest{
					OrgID: mo.Some(bulkOrgUUID.String()),
					TagID: testTagUUID,
				}).Return(deleteOperationResult(components.TagOperationStatusPending), nil)
				return m
			},
			params: BulkUnassignParams{
				OrgID: mo.Some(identifiers.NewOrganizationID(bulkOrgUUID)),
				TagID: identifiers.NewTagID(testTagUUID),
			},
			assert: func(t *testing.T, _ BulkUnassignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
			},
		},
		{
			// An inverted window matches nothing, which would look like a wipe that
			// found no assignments rather than a mistake.
			name: "impossible window is rejected without touching the API",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkDeleteTagAssignments(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: BulkUnassignParams{
				TagID:         identifiers.NewTagID(testTagUUID),
				CreatedBefore: mo.Some(bulkAfterTime),
				CreatedAfter:  mo.Some(bulkBeforeTime),
			},
			assert: func(t *testing.T, _ BulkUnassignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "created-before must be after created-after")
			},
		},
		{
			name: "unknown tag name fails before submitting",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Return(client.Result[components.TagsList]{
					Metadata: okMeta(),
					Data:     &components.TagsList{Tags: []components.Tag{}, TotalSize: 0},
				}, nil)
				m.EXPECT().BulkDeleteTagAssignments(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: BulkUnassignParams{TagID: identifiers.NewTagID("ghost")},
			assert: func(t *testing.T, _ BulkUnassignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "ghost")
			},
		},
		{
			name: "client error is returned as-is",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().BulkDeleteTagAssignments(gomock.Any(), gomock.Any()).Return(
					client.Result[components.TagOperation]{},
					clientStructuredError("Permission denied", 403),
				)
				expectNoTagOfThatName(m)
				return m
			},
			params: BulkUnassignParams{TagID: identifiers.NewTagID(testTagUUID)},
			assert: func(t *testing.T, _ BulkUnassignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "Permission denied")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := &tagsService{client: tc.client(ctrl)}
			res, err := svc.BulkUnassign(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}
