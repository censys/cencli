package tags

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/app/streaming"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

func strPtr(s string) *string { return &s }

func okMeta() client.Metadata {
	return client.Metadata{
		Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
		Response: &http.Response{StatusCode: 200},
		Latency:  100 * time.Millisecond,
	}
}

func tagPage(names []string, total int64, nextToken string) client.Result[components.TagsList] {
	tags := make([]components.Tag, 0, len(names))
	for _, n := range names {
		tags = append(tags, components.Tag{ID: n + "-id", Name: n, Privacy: components.TagPrivacyShared})
	}
	list := &components.TagsList{Tags: tags, TotalSize: total}
	if nextToken != "" {
		list.NextPageToken = strPtr(nextToken)
	}
	return client.Result[components.TagsList]{Metadata: okMeta(), Data: list}
}

func TestTagsService_ListTags(t *testing.T) {
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params ListParams
		ctx    func() context.Context
		assert func(t *testing.T, res ListResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - single page, no filters",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{}).
					Return(tagPage([]string{"alpha", "beta"}, 2, ""), nil)
				return m
			},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Tags, 2)
				require.Equal(t, int64(2), res.TotalSize)
				require.Equal(t, "alpha", res.Tags[0].Name)
				require.Equal(t, "shared", res.Tags[0].Privacy)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "success - filters and org threaded to client",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{
					OrgID:     mo.Some("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
					PageSize:  mo.Some(int64(50)),
					OrderBy:   mo.Some("name_desc"),
					Name:      mo.Some("my-tag"),
					CreatedBy: mo.Some("creator-id"),
					Privacy:   mo.Some("shared"),
				}).Return(tagPage([]string{"my-tag"}, 1, ""), nil)
				return m
			},
			params: ListParams{
				OrgID:     mo.Some(identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))),
				Privacy:   mo.Some("shared"),
				Name:      mo.Some("my-tag"),
				CreatedBy: mo.Some("creator-id"),
				OrderBy:   mo.Some("name_desc"),
				PageSize:  mo.Some(uint64(50)),
			},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Tags, 1)
				require.Equal(t, "my-tag", res.Tags[0].Name)
			},
		},
		{
			name: "pagination - multiple pages collected",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{PageSize: mo.Some(int64(2))}).
						Return(tagPage([]string{"a", "b"}, 5, "token1"), nil),
					m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{PageSize: mo.Some(int64(2)), PageToken: mo.Some("token1")}).
						Return(tagPage([]string{"c", "d"}, 5, "token2"), nil),
					m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{PageSize: mo.Some(int64(2)), PageToken: mo.Some("token2")}).
						Return(tagPage([]string{"e"}, 5, ""), nil),
				)
				return m
			},
			params: ListParams{PageSize: mo.Some(uint64(2))},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Tags, 5)
				require.Equal(t, int64(5), res.TotalSize)
			},
		},
		{
			name: "pagination - limited by max-pages",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{PageSize: mo.Some(int64(2))}).
						Return(tagPage([]string{"a", "b"}, 10, "token1"), nil),
					m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{PageSize: mo.Some(int64(2)), PageToken: mo.Some("token1")}).
						Return(tagPage([]string{"c", "d"}, 10, "token2"), nil),
					// third page must NOT be called
				)
				return m
			},
			params: ListParams{PageSize: mo.Some(uint64(2)), MaxPages: mo.Some(uint64(2))},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Tags, 4)
			},
		},
		{
			name: "first-page error returned immediately",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				detail := "Invalid request"
				status := int64(400)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status})
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).
					Return(client.Result[components.TagsList]{}, structuredErr)
				return m
			},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Nil(t, res.Tags)
				require.Contains(t, err.Error(), "Invalid request")
			},
		},
		{
			name: "later-page error yields partial result",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{PageSize: mo.Some(int64(2))}).
						Return(tagPage([]string{"a", "b"}, 5, "token1"), nil),
					m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{PageSize: mo.Some(int64(2)), PageToken: mo.Some("token1")}).
						Return(client.Result[components.TagsList]{}, client.NewClientError(errors.New("network error"))),
				)
				return m
			},
			params: ListParams{PageSize: mo.Some(uint64(2))},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Tags, 2)
				require.NotNil(t, res.PartialError)
				require.Contains(t, res.PartialError.Error(), "network error")
			},
		},
		{
			name: "empty result",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).
					Return(tagPage(nil, 0, ""), nil)
				return m
			},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Empty(t, res.Tags)
				require.Equal(t, int64(0), res.TotalSize)
			},
		},
		{
			name: "invalid page size",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl) // no client call expected
			},
			params: ListParams{PageSize: mo.Some(uint64(0))},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "page size")
			},
		},
		{
			name: "invalid max pages",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl) // no client call expected
			},
			params: ListParams{MaxPages: mo.Some(uint64(0))},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "max pages")
			},
		},
		{
			name: "invalid order-by rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl) // no client call expected
			},
			params: ListParams{OrderBy: mo.Some("bogus")},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "order-by")
				require.Contains(t, err.Error(), "name_asc")
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "invalid privacy rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl) // no client call expected
			},
			params: ListParams{Privacy: mo.Some("bogus")},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "privacy")
				require.Contains(t, err.Error(), "private")
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "valid order-by and privacy pass through",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{
					OrderBy: mo.Some("created_at_desc"),
					Privacy: mo.Some("private"),
				}).Return(tagPage([]string{"a"}, 1, ""), nil)
				return m
			},
			params: ListParams{OrderBy: mo.Some("created_at_desc"), Privacy: mo.Some("private")},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Tags, 1)
			},
		},
		{
			name: "context cancellation propagates",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl) // no client call expected
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.ErrorIs(t, err, context.Canceled)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))

			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}

			res, err := svc.ListTags(ctx, tc.params)
			tc.assert(t, res, err)
		})
	}
}

// TestTagsService_ListTags_Progress verifies the running collected count is
// reported across pages.
func TestTagsService_ListTags_Progress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockClient(ctrl)
	gomock.InOrder(
		m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{PageSize: mo.Some(int64(2))}).
			Return(tagPage([]string{"a", "b"}, 5, "token1"), nil),
		m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{PageSize: mo.Some(int64(2)), PageToken: mo.Some("token1")}).
			Return(tagPage([]string{"c", "d"}, 5, "token2"), nil),
		m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{PageSize: mo.Some(int64(2)), PageToken: mo.Some("token2")}).
			Return(tagPage([]string{"e"}, 5, ""), nil),
	)

	pub, events := progress.NewChannelPublisher(64)
	ctx := progress.WithPublisher(context.Background(), pub)

	svc := New(m)
	res, err := svc.ListTags(ctx, ListParams{PageSize: mo.Some(uint64(2))})
	require.NoError(t, err)
	require.Len(t, res.Tags, 5)

	pub.Close(nil)
	var msgs []string
	for ev := range events {
		if ev.Message != "" {
			msgs = append(msgs, ev.Message)
		}
	}
	// Progress is reported from the second page onward; the collected count must
	// be non-zero and increasing (never stuck at 0 as in the pre-TAGS-1 bug).
	require.NotEmpty(t, msgs)
	require.Contains(t, msgs[len(msgs)-1], "collected")
}

func TestTagsService_GetTag(t *testing.T) {
	orgUUID := uuid.New()
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params GetParams
		assert func(t *testing.T, res GetResult, err cenclierrors.CencliError)
	}{
		{
			name: "success by name - maps SDK tag to DTO",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				desc := "a description"
				sdkTag := &components.Tag{
					ID:          "tag-id",
					Name:        "my-tag",
					Description: &desc,
					Privacy:     components.TagPrivacyPrivate,
					CreatedBy:   "creator",
				}
				m.EXPECT().GetTag(gomock.Any(), mo.None[string](), "my-tag").
					Return(client.Result[components.Tag]{Metadata: okMeta(), Data: sdkTag}, nil)
				// Every successful get also counts assignments; this case is about
				// the mapping, so the count itself is left unasserted.
				m.EXPECT().ListTagAssignments(gomock.Any(), gomock.Any()).
					Return(assignmentsPage(nil, 0, ""), nil)
				return m
			},
			params: GetParams{TagID: identifiers.NewTagID("my-tag")},
			assert: func(t *testing.T, res GetResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, "tag-id", res.Tag.ID)
				require.Equal(t, "my-tag", res.Tag.Name)
				require.Equal(t, "private", res.Tag.Privacy)
				require.NotNil(t, res.Tag.Description)
				require.Equal(t, "a description", *res.Tag.Description)
				require.Equal(t, "creator", res.Tag.CreatedBy)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "org id threaded to client",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().GetTag(gomock.Any(), mo.Some(orgUUID.String()), orgUUID.String()).
					Return(client.Result[components.Tag]{Metadata: okMeta(), Data: &components.Tag{ID: "id", Name: "n", Privacy: components.TagPrivacyShared}}, nil)
				// The org ID must reach the count request too, not just the get.
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					OrgID:    mo.Some(orgUUID.String()),
					TagID:    "id",
					PageSize: mo.Some(int64(1)),
				}).Return(assignmentsPage(nil, 0, ""), nil)
				return m
			},
			params: GetParams{
				OrgID: mo.Some(identifiers.NewOrganizationID(orgUUID)),
				TagID: identifiers.NewTagID(orgUUID.String()),
			},
			assert: func(t *testing.T, res GetResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, "n", res.Tag.Name)
			},
		},
		{
			name: "client error propagates",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				detail := "Tag not found"
				status := int64(404)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status})
				m.EXPECT().GetTag(gomock.Any(), mo.None[string](), "missing").
					Return(client.Result[components.Tag]{}, structuredErr)
				return m
			},
			params: GetParams{TagID: identifiers.NewTagID("missing")},
			assert: func(t *testing.T, res GetResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.Tag.ID)
				require.Contains(t, err.Error(), "Tag not found")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))
			res, err := svc.GetTag(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

func TestTagsService_CreateTag(t *testing.T) {
	orgUUID := uuid.New()
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params CreateParams
		assert func(t *testing.T, res CreateResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - maps SDK tag to DTO",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				desc := "a description"
				sdkTag := &components.Tag{
					ID:          "tag-id",
					Name:        "my-tag",
					Description: &desc,
					Privacy:     components.TagPrivacyPrivate,
					CreatedBy:   "creator",
				}
				m.EXPECT().CreateTag(gomock.Any(), client.CreateTagRequest{
					Name:        "my-tag",
					Description: mo.Some("a description"),
					Privacy:     "private",
				}).Return(client.Result[components.Tag]{Metadata: okMeta(), Data: sdkTag}, nil)
				return m
			},
			params: CreateParams{
				Name:        "my-tag",
				Description: mo.Some("a description"),
				Privacy:     "private",
			},
			assert: func(t *testing.T, res CreateResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, "tag-id", res.Tag.ID)
				require.Equal(t, "my-tag", res.Tag.Name)
				require.Equal(t, "private", res.Tag.Privacy)
				require.NotNil(t, res.Tag.Description)
				require.Equal(t, "a description", *res.Tag.Description)
				require.Equal(t, "creator", res.Tag.CreatedBy)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "org id and shared privacy threaded to client",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().CreateTag(gomock.Any(), client.CreateTagRequest{
					OrgID:   mo.Some(orgUUID.String()),
					Name:    "shared-tag",
					Privacy: "shared",
				}).Return(client.Result[components.Tag]{Metadata: okMeta(), Data: &components.Tag{ID: "id", Name: "shared-tag", Privacy: components.TagPrivacyShared}}, nil)
				return m
			},
			params: CreateParams{
				OrgID:   mo.Some(identifiers.NewOrganizationID(orgUUID)),
				Name:    "shared-tag",
				Privacy: "shared",
			},
			assert: func(t *testing.T, res CreateResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, "shared-tag", res.Tag.Name)
				require.Equal(t, "shared", res.Tag.Privacy)
			},
		},
		{
			name: "empty name rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl) // no client call expected
			},
			params: CreateParams{Name: "", Privacy: "private"},
			assert: func(t *testing.T, res CreateResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.Tag.ID)
				require.Contains(t, err.Error(), "name")
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "whitespace-only name rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl) // no client call expected
			},
			params: CreateParams{Name: "   ", Privacy: "private"},
			assert: func(t *testing.T, res CreateResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.Tag.ID)
				require.Contains(t, err.Error(), "name")
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "invalid privacy rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl) // no client call expected
			},
			params: CreateParams{Name: "my-tag", Privacy: "bogus"},
			assert: func(t *testing.T, res CreateResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.Tag.ID)
				require.Contains(t, err.Error(), "privacy")
				require.Contains(t, err.Error(), "private")
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "client error propagates",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				detail := "Tag already exists"
				status := int64(409)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status})
				m.EXPECT().CreateTag(gomock.Any(), client.CreateTagRequest{Name: "dupe", Privacy: "private"}).
					Return(client.Result[components.Tag]{}, structuredErr)
				return m
			},
			params: CreateParams{Name: "dupe", Privacy: "private"},
			assert: func(t *testing.T, res CreateResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.Tag.ID)
				require.Contains(t, err.Error(), "Tag already exists")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))
			res, err := svc.CreateTag(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

func TestTagsService_UpdateTag(t *testing.T) {
	orgUUID := uuid.New()
	tagUUID := uuid.New()
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params UpdateParams
		assert func(t *testing.T, res UpdateResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - UUID input skips resolution and maps SDK tag to DTO",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				// A UUID needs no name resolution.
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				desc := "new description"
				sdkTag := &components.Tag{
					ID:          "tag-id",
					Name:        "renamed",
					Description: &desc,
					Privacy:     components.TagPrivacyShared,
					CreatedBy:   "creator",
				}
				m.EXPECT().UpdateTag(gomock.Any(), client.UpdateTagRequest{
					TagID:       tagUUID.String(),
					Name:        mo.Some("renamed"),
					Description: mo.Some("new description"),
					Privacy:     mo.Some("shared"),
				}).Return(client.Result[components.Tag]{Metadata: okMeta(), Data: sdkTag}, nil)
				return m
			},
			params: UpdateParams{
				TagID:       identifiers.NewTagID(tagUUID.String()),
				Name:        mo.Some("renamed"),
				Description: mo.Some("new description"),
				Privacy:     mo.Some("shared"),
			},
			assert: func(t *testing.T, res UpdateResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, "tag-id", res.Tag.ID)
				require.Equal(t, "renamed", res.Tag.Name)
				require.Equal(t, "shared", res.Tag.Privacy)
				require.NotNil(t, res.Tag.Description)
				require.Equal(t, "new description", *res.Tag.Description)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "name resolved to UUID before the write",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{
					Name:     mo.Some("my-tag"),
					PageSize: mo.Some(int64(1)),
				}).Return(client.Result[components.TagsList]{Data: &components.TagsList{
					Tags: []components.Tag{{ID: "resolved-id", Name: "my-tag"}},
				}}, nil)
				m.EXPECT().UpdateTag(gomock.Any(), client.UpdateTagRequest{
					TagID:   "resolved-id",
					Privacy: mo.Some("shared"),
				}).Return(client.Result[components.Tag]{Metadata: okMeta(), Data: &components.Tag{ID: "resolved-id", Name: "my-tag", Privacy: components.TagPrivacyShared}}, nil)
				return m
			},
			params: UpdateParams{TagID: identifiers.NewTagID("my-tag"), Privacy: mo.Some("shared")},
			assert: func(t *testing.T, res UpdateResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, "resolved-id", res.Tag.ID)
			},
		},
		{
			name: "name not found returns typed error, no write",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).
					Return(client.Result[components.TagsList]{Data: &components.TagsList{}}, nil)
				return m
			},
			params: UpdateParams{TagID: identifiers.NewTagID("ghost"), Privacy: mo.Some("shared")},
			assert: func(t *testing.T, res UpdateResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.Tag.ID)
				require.Contains(t, err.Error(), "not found")
				require.False(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "partial update - only privacy, org and raw UUID threaded to client",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().UpdateTag(gomock.Any(), client.UpdateTagRequest{
					OrgID:   mo.Some(orgUUID.String()),
					TagID:   tagUUID.String(),
					Privacy: mo.Some("private"),
				}).Return(client.Result[components.Tag]{Metadata: okMeta(), Data: &components.Tag{ID: "id", Name: "n", Privacy: components.TagPrivacyPrivate}}, nil)
				return m
			},
			params: UpdateParams{
				OrgID:   mo.Some(identifiers.NewOrganizationID(orgUUID)),
				TagID:   identifiers.NewTagID(tagUUID.String()),
				Privacy: mo.Some("private"),
			},
			assert: func(t *testing.T, res UpdateResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, "private", res.Tag.Privacy)
			},
		},
		{
			name: "clear description sent as empty string",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().UpdateTag(gomock.Any(), client.UpdateTagRequest{
					TagID:       tagUUID.String(),
					Description: mo.Some(""),
				}).Return(client.Result[components.Tag]{Metadata: okMeta(), Data: &components.Tag{ID: "id", Name: "my-tag", Privacy: components.TagPrivacyPrivate}}, nil)
				return m
			},
			params: UpdateParams{
				TagID:       identifiers.NewTagID(tagUUID.String()),
				Description: mo.Some(""),
			},
			assert: func(t *testing.T, res UpdateResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, "id", res.Tag.ID)
			},
		},
		{
			name: "invalid privacy rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl) // no client call expected
			},
			params: UpdateParams{TagID: identifiers.NewTagID("my-tag"), Privacy: mo.Some("bogus")},
			assert: func(t *testing.T, res UpdateResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.Tag.ID)
				require.Contains(t, err.Error(), "privacy")
				require.Contains(t, err.Error(), "private")
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "client error propagates",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				detail := "Tag not found"
				status := int64(404)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status})
				m.EXPECT().UpdateTag(gomock.Any(), client.UpdateTagRequest{TagID: tagUUID.String(), Name: mo.Some("x")}).
					Return(client.Result[components.Tag]{}, structuredErr)
				return m
			},
			params: UpdateParams{TagID: identifiers.NewTagID(tagUUID.String()), Name: mo.Some("x")},
			assert: func(t *testing.T, res UpdateResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.Tag.ID)
				require.Contains(t, err.Error(), "Tag not found")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))
			res, err := svc.UpdateTag(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

func assignmentResult(id, assetID string) client.Result[components.TagAssignment] {
	return client.Result[components.TagAssignment]{
		Metadata: okMeta(),
		Data: &components.TagAssignment{
			ID:          id,
			TagID:       "tag-id",
			AssetID:     assetID,
			AssetType:   components.TagAssignmentAssetTypeHost,
			PlatformRef: "https://platform.censys.io/hosts/" + assetID,
		},
	}
}

func clientStructuredError(detail string, status int64) client.ClientError {
	return client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status})
}

// assignmentsListResult builds a single-assignment list response (what an
// asset_id-filtered ListTagAssignments returns for an assigned asset).
func assignmentsListResult(id, assetID string) client.Result[components.TagAssignmentsList] {
	return client.Result[components.TagAssignmentsList]{
		Metadata: okMeta(),
		Data: &components.TagAssignmentsList{
			Assignments: []components.TagAssignment{{
				ID:          id,
				TagID:       "tag-id",
				AssetID:     assetID,
				AssetType:   components.TagAssignmentAssetTypeHost,
				PlatformRef: "https://platform.censys.io/hosts/" + assetID,
			}},
			TotalSize: 1,
		},
	}
}

// emptyAssignmentsListResult is the response for an asset with no assignment to
// the tag (nothing to unassign).
func emptyAssignmentsListResult() client.Result[components.TagAssignmentsList] {
	return client.Result[components.TagAssignmentsList]{
		Metadata: okMeta(),
		Data:     &components.TagAssignmentsList{TotalSize: 0},
	}
}

func TestTagsService_Assign(t *testing.T) {
	orgUUID := uuid.New()
	tagUUID := uuid.New()
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params AssignParams
		assert func(t *testing.T, res AssignResult, err cenclierrors.CencliError)
	}{
		{
			name: "all assets assigned; UUID input skips resolution",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().CreateTagAssignment(gomock.Any(), client.CreateTagAssignmentRequest{
					TagID: tagUUID.String(), AssetID: "8.8.8.8",
				}).Return(assignmentResult("a1", "8.8.8.8"), nil)
				m.EXPECT().CreateTagAssignment(gomock.Any(), client.CreateTagAssignmentRequest{
					TagID: tagUUID.String(), AssetID: "1.1.1.1",
				}).Return(assignmentResult("a2", "1.1.1.1"), nil)
				return m
			},
			params: AssignParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				AssetIDs: []string{"8.8.8.8", "1.1.1.1"},
			},
			assert: func(t *testing.T, res AssignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Assignments, 2)
				require.Empty(t, res.Failures)
				require.Nil(t, res.PartialError)
				require.Equal(t, tagUUID.String(), res.TagID)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "name resolved to UUID once, then assets assigned",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{
					Name:     mo.Some("my-tag"),
					PageSize: mo.Some(int64(1)),
				}).Return(client.Result[components.TagsList]{Data: &components.TagsList{
					Tags: []components.Tag{{ID: "resolved-id", Name: "my-tag"}},
				}}, nil)
				m.EXPECT().CreateTagAssignment(gomock.Any(), client.CreateTagAssignmentRequest{
					TagID: "resolved-id", AssetID: "8.8.8.8",
				}).Return(assignmentResult("a1", "8.8.8.8"), nil)
				return m
			},
			params: AssignParams{
				TagID:    identifiers.NewTagID("my-tag"),
				AssetIDs: []string{"8.8.8.8"},
			},
			assert: func(t *testing.T, res AssignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Assignments, 1)
				require.Equal(t, "my-tag", res.TagID)
			},
		},
		{
			name: "partial failure: one asset fails, the rest still assigned",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().CreateTagAssignment(gomock.Any(), client.CreateTagAssignmentRequest{
					TagID: tagUUID.String(), AssetID: "8.8.8.8",
				}).Return(assignmentResult("a1", "8.8.8.8"), nil)
				m.EXPECT().CreateTagAssignment(gomock.Any(), client.CreateTagAssignmentRequest{
					TagID: tagUUID.String(), AssetID: "1.1.1.1",
				}).Return(client.Result[components.TagAssignment]{}, clientStructuredError("Forbidden", 403))
				return m
			},
			params: AssignParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				AssetIDs: []string{"8.8.8.8", "1.1.1.1"},
			},
			assert: func(t *testing.T, res AssignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Assignments, 1)
				require.Len(t, res.Failures, 1)
				require.Equal(t, "1.1.1.1", res.Failures[0].AssetID)
				require.NotNil(t, res.PartialError)
				require.Contains(t, res.PartialError.Error(), "1 of 2")
			},
		},
		{
			name: "all assets fail: first error returned, no partial result",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().CreateTagAssignment(gomock.Any(), gomock.Any()).
					Return(client.Result[components.TagAssignment]{}, clientStructuredError("Permission denied", 403)).
					Times(2)
				return m
			},
			params: AssignParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				AssetIDs: []string{"8.8.8.8", "1.1.1.1"},
			},
			assert: func(t *testing.T, res AssignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.Assignments)
				require.Contains(t, err.Error(), "Permission denied")
			},
		},
		{
			name: "empty identifier rejected before any lookup or assignment",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().CreateTagAssignment(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: AssignParams{
				TagID:    identifiers.NewTagID("  "),
				AssetIDs: []string{"8.8.8.8"},
			},
			assert: func(t *testing.T, res AssignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "required")
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "no assets rejected before any call",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().CreateTagAssignment(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: AssignParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				AssetIDs: nil,
			},
			assert: func(t *testing.T, res AssignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "org id threaded through to the client",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().CreateTagAssignment(gomock.Any(), client.CreateTagAssignmentRequest{
					OrgID: mo.Some(orgUUID.String()), TagID: tagUUID.String(), AssetID: "8.8.8.8",
				}).Return(assignmentResult("a1", "8.8.8.8"), nil)
				return m
			},
			params: AssignParams{
				OrgID:    mo.Some(identifiers.NewOrganizationID(orgUUID)),
				TagID:    identifiers.NewTagID(tagUUID.String()),
				AssetIDs: []string{"8.8.8.8"},
			},
			assert: func(t *testing.T, res AssignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Assignments, 1)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))
			res, err := svc.Assign(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

// A cancellation mid-run, after some assets have already been assigned, must be
// surfaced as a PartialError rather than reported as a clean success.
func TestTagsService_Assign_CancellationSurfacesPartial(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tagUUID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())

	m := mocks.NewMockClient(ctrl)
	// First asset succeeds and cancels the context; the loop then stops before
	// the second asset, with one success and no recorded failure.
	m.EXPECT().CreateTagAssignment(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ client.CreateTagAssignmentRequest) (client.Result[components.TagAssignment], client.ClientError) {
			cancel()
			return assignmentResult("a1", "8.8.8.8"), nil
		})

	svc := New(m)
	res, err := svc.Assign(ctx, AssignParams{
		TagID:    identifiers.NewTagID(tagUUID.String()),
		AssetIDs: []string{"8.8.8.8", "1.1.1.1"},
	})

	require.NoError(t, err)
	require.Len(t, res.Assignments, 1)
	require.Empty(t, res.Failures)
	require.NotNil(t, res.PartialError)
}

func TestTagsService_Unassign(t *testing.T) {
	orgUUID := uuid.New()
	tagUUID := uuid.New()
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params UnassignParams
		assert func(t *testing.T, res UnassignResult, err cenclierrors.CencliError)
	}{
		{
			name: "all assets unassigned; UUID input skips resolution",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					TagID: tagUUID.String(), AssetID: mo.Some("8.8.8.8"), PageSize: mo.Some(int64(1)),
				}).Return(assignmentsListResult("a1", "8.8.8.8"), nil)
				m.EXPECT().DeleteTagAssignment(gomock.Any(), gomock.Any(), tagUUID.String(), "a1").
					Return(okMeta(), nil)
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					TagID: tagUUID.String(), AssetID: mo.Some("1.1.1.1"), PageSize: mo.Some(int64(1)),
				}).Return(assignmentsListResult("a2", "1.1.1.1"), nil)
				m.EXPECT().DeleteTagAssignment(gomock.Any(), gomock.Any(), tagUUID.String(), "a2").
					Return(okMeta(), nil)
				return m
			},
			params: UnassignParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				AssetIDs: []string{"8.8.8.8", "1.1.1.1"},
			},
			assert: func(t *testing.T, res UnassignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Unassigned, 2)
				require.Empty(t, res.Failures)
				require.Nil(t, res.PartialError)
				require.Equal(t, tagUUID.String(), res.TagID)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "name resolved to UUID once, then assets unassigned",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{
					Name:     mo.Some("my-tag"),
					PageSize: mo.Some(int64(1)),
				}).Return(client.Result[components.TagsList]{Data: &components.TagsList{
					Tags: []components.Tag{{ID: "resolved-id", Name: "my-tag"}},
				}}, nil)
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					TagID: "resolved-id", AssetID: mo.Some("8.8.8.8"), PageSize: mo.Some(int64(1)),
				}).Return(assignmentsListResult("a1", "8.8.8.8"), nil)
				m.EXPECT().DeleteTagAssignment(gomock.Any(), gomock.Any(), "resolved-id", "a1").
					Return(okMeta(), nil)
				return m
			},
			params: UnassignParams{
				TagID:    identifiers.NewTagID("my-tag"),
				AssetIDs: []string{"8.8.8.8"},
			},
			assert: func(t *testing.T, res UnassignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Unassigned, 1)
				require.Equal(t, "my-tag", res.TagID)
			},
		},
		{
			name: "not assigned: asset with no assignment is a per-asset failure",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					TagID: tagUUID.String(), AssetID: mo.Some("8.8.8.8"), PageSize: mo.Some(int64(1)),
				}).Return(assignmentsListResult("a1", "8.8.8.8"), nil)
				m.EXPECT().DeleteTagAssignment(gomock.Any(), gomock.Any(), tagUUID.String(), "a1").
					Return(okMeta(), nil)
				// The second asset has no assignment; its removal is never attempted.
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					TagID: tagUUID.String(), AssetID: mo.Some("1.1.1.1"), PageSize: mo.Some(int64(1)),
				}).Return(emptyAssignmentsListResult(), nil)
				m.EXPECT().DeleteTagAssignment(gomock.Any(), gomock.Any(), gomock.Any(), "a2").Times(0)
				return m
			},
			params: UnassignParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				AssetIDs: []string{"8.8.8.8", "1.1.1.1"},
			},
			assert: func(t *testing.T, res UnassignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Unassigned, 1)
				require.Len(t, res.Failures, 1)
				require.Equal(t, "1.1.1.1", res.Failures[0].AssetID)
				require.Contains(t, res.Failures[0].Err.Error(), "not assigned")
				require.NotNil(t, res.PartialError)
				require.Contains(t, res.PartialError.Error(), "1 of 2")
			},
		},
		{
			name: "partial failure: one asset's removal fails, the rest still unassigned",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					TagID: tagUUID.String(), AssetID: mo.Some("8.8.8.8"), PageSize: mo.Some(int64(1)),
				}).Return(assignmentsListResult("a1", "8.8.8.8"), nil)
				m.EXPECT().DeleteTagAssignment(gomock.Any(), gomock.Any(), tagUUID.String(), "a1").
					Return(okMeta(), nil)
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					TagID: tagUUID.String(), AssetID: mo.Some("1.1.1.1"), PageSize: mo.Some(int64(1)),
				}).Return(assignmentsListResult("a2", "1.1.1.1"), nil)
				m.EXPECT().DeleteTagAssignment(gomock.Any(), gomock.Any(), tagUUID.String(), "a2").
					Return(client.Metadata{}, clientStructuredError("Forbidden", 403))
				return m
			},
			params: UnassignParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				AssetIDs: []string{"8.8.8.8", "1.1.1.1"},
			},
			assert: func(t *testing.T, res UnassignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Unassigned, 1)
				require.Len(t, res.Failures, 1)
				require.Equal(t, "1.1.1.1", res.Failures[0].AssetID)
				require.NotNil(t, res.PartialError)
				require.Contains(t, res.PartialError.Error(), "1 of 2")
			},
		},
		{
			name: "all assets fail: first error returned, no partial result",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagAssignments(gomock.Any(), gomock.Any()).
					Return(emptyAssignmentsListResult(), nil).Times(2)
				m.EXPECT().DeleteTagAssignment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: UnassignParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				AssetIDs: []string{"8.8.8.8", "1.1.1.1"},
			},
			assert: func(t *testing.T, res UnassignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.Unassigned)
				require.Contains(t, err.Error(), "not assigned")
			},
		},
		{
			name: "empty identifier rejected before any lookup or removal",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().ListTagAssignments(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().DeleteTagAssignment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: UnassignParams{
				TagID:    identifiers.NewTagID("  "),
				AssetIDs: []string{"8.8.8.8"},
			},
			assert: func(t *testing.T, res UnassignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "required")
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "no assets rejected before any call",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().ListTagAssignments(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().DeleteTagAssignment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: UnassignParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				AssetIDs: nil,
			},
			assert: func(t *testing.T, res UnassignResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "org id threaded through to the client",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					OrgID: mo.Some(orgUUID.String()), TagID: tagUUID.String(),
					AssetID: mo.Some("8.8.8.8"), PageSize: mo.Some(int64(1)),
				}).Return(assignmentsListResult("a1", "8.8.8.8"), nil)
				m.EXPECT().DeleteTagAssignment(gomock.Any(), mo.Some(orgUUID.String()), tagUUID.String(), "a1").
					Return(okMeta(), nil)
				return m
			},
			params: UnassignParams{
				OrgID:    mo.Some(identifiers.NewOrganizationID(orgUUID)),
				TagID:    identifiers.NewTagID(tagUUID.String()),
				AssetIDs: []string{"8.8.8.8"},
			},
			assert: func(t *testing.T, res UnassignResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Unassigned, 1)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))
			res, err := svc.Unassign(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

// A cancellation mid-run, after some assets have already been unassigned, must be
// surfaced as a PartialError rather than reported as a clean success.
func TestTagsService_Unassign_CancellationSurfacesPartial(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tagUUID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())

	m := mocks.NewMockClient(ctrl)
	// First asset is looked up and removed; the delete cancels the context, so the
	// loop stops before the second asset with one success and no recorded failure.
	m.EXPECT().ListTagAssignments(gomock.Any(), gomock.Any()).
		Return(assignmentsListResult("a1", "8.8.8.8"), nil)
	m.EXPECT().DeleteTagAssignment(gomock.Any(), gomock.Any(), gomock.Any(), "a1").
		DoAndReturn(func(_ context.Context, _ mo.Option[string], _, _ string) (client.Metadata, client.ClientError) {
			cancel()
			return okMeta(), nil
		})

	svc := New(m)
	res, err := svc.Unassign(ctx, UnassignParams{
		TagID:    identifiers.NewTagID(tagUUID.String()),
		AssetIDs: []string{"8.8.8.8", "1.1.1.1"},
	})

	require.NoError(t, err)
	require.Len(t, res.Unassigned, 1)
	require.Empty(t, res.Failures)
	require.NotNil(t, res.PartialError)
}

func TestTagsService_DeleteTag(t *testing.T) {
	orgUUID := uuid.New()
	tagUUID := uuid.New()
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params DeleteParams
		assert func(t *testing.T, res DeleteResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - UUID input skips resolution, returns metadata and echoes the identifier",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				// A UUID needs no name resolution.
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().DeleteTag(gomock.Any(), mo.None[string](), tagUUID.String()).
					Return(okMeta(), nil)
				return m
			},
			params: DeleteParams{TagID: identifiers.NewTagID(tagUUID.String())},
			assert: func(t *testing.T, res DeleteResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, tagUUID.String(), res.TagID)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "name resolved to UUID before deletion; original identifier echoed",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{
					Name:     mo.Some("my-tag"),
					PageSize: mo.Some(int64(1)),
				}).Return(client.Result[components.TagsList]{Data: &components.TagsList{
					Tags: []components.Tag{{ID: "resolved-id", Name: "my-tag"}},
				}}, nil)
				m.EXPECT().DeleteTag(gomock.Any(), mo.None[string](), "resolved-id").
					Return(okMeta(), nil)
				return m
			},
			params: DeleteParams{TagID: identifiers.NewTagID("my-tag")},
			assert: func(t *testing.T, res DeleteResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				// The render shows what the user typed, not the resolved UUID.
				require.Equal(t, "my-tag", res.TagID)
			},
		},
		{
			name: "name not found returns typed error, no deletion",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).
					Return(client.Result[components.TagsList]{Data: &components.TagsList{}}, nil)
				return m
			},
			params: DeleteParams{TagID: identifiers.NewTagID("ghost")},
			assert: func(t *testing.T, res DeleteResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.TagID)
				require.Contains(t, err.Error(), "not found")
				require.False(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "empty identifier rejected before any lookup or deletion",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				// Neither a resolve lookup nor a delete may fire for an empty id.
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().DeleteTag(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: DeleteParams{TagID: identifiers.NewTagID("  ")},
			assert: func(t *testing.T, res DeleteResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.TagID)
				require.Contains(t, err.Error(), "required")
				require.True(t, err.ShouldPrintUsage())
			},
		},
		{
			name: "org id threaded; raw identifier passed straight through",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().DeleteTag(gomock.Any(), mo.Some(orgUUID.String()), tagUUID.String()).
					Return(okMeta(), nil)
				return m
			},
			params: DeleteParams{
				OrgID: mo.Some(identifiers.NewOrganizationID(orgUUID)),
				TagID: identifiers.NewTagID(tagUUID.String()),
			},
			assert: func(t *testing.T, res DeleteResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, tagUUID.String(), res.TagID)
			},
		},
		{
			name: "client error propagates",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				detail := "Tag not found"
				status := int64(404)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status})
				m.EXPECT().DeleteTag(gomock.Any(), mo.None[string](), tagUUID.String()).
					Return(client.Metadata{}, structuredErr)
				return m
			},
			params: DeleteParams{TagID: identifiers.NewTagID(tagUUID.String())},
			assert: func(t *testing.T, res DeleteResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.TagID)
				require.Contains(t, err.Error(), "Tag not found")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))
			res, err := svc.DeleteTag(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

// assignmentsPage builds a multi-assignment page, optionally with a next-page token.
func assignmentsPage(assetIDs []string, total int64, nextToken string) client.Result[components.TagAssignmentsList] {
	assignments := make([]components.TagAssignment, 0, len(assetIDs))
	for i, a := range assetIDs {
		assignments = append(assignments, components.TagAssignment{
			ID:          fmt.Sprintf("assignment-%d", i),
			TagID:       "tag-id",
			AssetID:     a,
			AssetType:   components.TagAssignmentAssetTypeHost,
			PlatformRef: "https://platform.censys.io/hosts/" + a,
		})
	}
	list := &components.TagAssignmentsList{Assignments: assignments, TotalSize: total}
	if nextToken != "" {
		list.NextPageToken = strPtr(nextToken)
	}
	return client.Result[components.TagAssignmentsList]{Metadata: okMeta(), Data: list}
}

func TestTagsService_ListAssignments(t *testing.T) {
	orgUUID := uuid.New()
	tagUUID := uuid.New()
	before := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	after := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params AssignmentsParams
		ctx    func() context.Context
		assert func(t *testing.T, res AssignmentsResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - UUID input skips resolution and maps assignments",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).Times(0)
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					TagID: tagUUID.String(),
				}).Return(assignmentsPage([]string{"8.8.8.8", "1.1.1.1"}, 2, ""), nil)
				return m
			},
			params: AssignmentsParams{TagID: identifiers.NewTagID(tagUUID.String())},
			assert: func(t *testing.T, res AssignmentsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Assignments, 2)
				require.Equal(t, int64(2), res.TotalSize)
				require.Equal(t, "8.8.8.8", res.Assignments[0].AssetID)
				require.Equal(t, "host", res.Assignments[0].AssetType)
				require.Equal(t, "https://platform.censys.io/hosts/8.8.8.8", res.Assignments[0].PlatformRef)
				require.NotNil(t, res.Meta)
			},
		},
		{
			name: "name resolved to UUID before listing",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), client.ListTagsRequest{
					Name:     mo.Some("my-tag"),
					PageSize: mo.Some(int64(1)),
				}).Return(client.Result[components.TagsList]{Data: &components.TagsList{
					Tags: []components.Tag{{ID: "resolved-id", Name: "my-tag"}},
				}}, nil)
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					TagID: "resolved-id",
				}).Return(assignmentsPage([]string{"8.8.8.8"}, 1, ""), nil)
				return m
			},
			params: AssignmentsParams{TagID: identifiers.NewTagID("my-tag")},
			assert: func(t *testing.T, res AssignmentsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Assignments, 1)
			},
		},
		{
			name: "filters and org threaded to client",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					OrgID:         mo.Some(orgUUID.String()),
					TagID:         tagUUID.String(),
					AssetID:       mo.Some("8.8.8.8"),
					AssetType:     mo.Some("host"),
					CreatedBy:     mo.Some("creator-id"),
					CreatedBefore: mo.Some(before),
					CreatedAfter:  mo.Some(after),
					OrderBy:       mo.Some("create_time_asc"),
					PageSize:      mo.Some(int64(50)),
				}).Return(assignmentsPage([]string{"8.8.8.8"}, 1, ""), nil)
				return m
			},
			params: AssignmentsParams{
				OrgID:         mo.Some(identifiers.NewOrganizationID(orgUUID)),
				TagID:         identifiers.NewTagID(tagUUID.String()),
				AssetID:       mo.Some("8.8.8.8"),
				AssetType:     mo.Some("host"),
				CreatedBy:     mo.Some("creator-id"),
				CreatedBefore: mo.Some(before),
				CreatedAfter:  mo.Some(after),
				OrderBy:       mo.Some("create_time_asc"),
				PageSize:      mo.Some(uint64(50)),
			},
			assert: func(t *testing.T, res AssignmentsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Assignments, 1)
			},
		},
		{
			name: "pagination - multiple pages collected",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
						TagID: tagUUID.String(), PageSize: mo.Some(int64(2)),
					}).Return(assignmentsPage([]string{"8.8.8.8", "1.1.1.1"}, 3, "token1"), nil),
					m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
						TagID: tagUUID.String(), PageSize: mo.Some(int64(2)), PageToken: mo.Some("token1"),
					}).Return(assignmentsPage([]string{"9.9.9.9"}, 3, ""), nil),
				)
				return m
			},
			params: AssignmentsParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				PageSize: mo.Some(uint64(2)),
			},
			assert: func(t *testing.T, res AssignmentsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Assignments, 3)
				require.Equal(t, int64(3), res.TotalSize)
			},
		},
		{
			name: "pagination - limited by max-pages",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
					TagID: tagUUID.String(), PageSize: mo.Some(int64(2)),
				}).Return(assignmentsPage([]string{"8.8.8.8", "1.1.1.1"}, 10, "token1"), nil)
				// the second page must NOT be fetched
				return m
			},
			params: AssignmentsParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				PageSize: mo.Some(uint64(2)),
				MaxPages: mo.Some(uint64(1)),
			},
			assert: func(t *testing.T, res AssignmentsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Assignments, 2)
				require.Equal(t, int64(10), res.TotalSize)
			},
		},
		{
			name: "pagination stops when the server repeats a page token",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
						TagID: tagUUID.String(),
					}).Return(assignmentsPage([]string{"8.8.8.8"}, 2, "stuck"), nil),
					// The server echoes back the same token; the loop must not continue.
					m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
						TagID: tagUUID.String(), PageToken: mo.Some("stuck"),
					}).Return(assignmentsPage([]string{"1.1.1.1"}, 2, "stuck"), nil),
				)
				return m
			},
			params: AssignmentsParams{TagID: identifiers.NewTagID(tagUUID.String())},
			assert: func(t *testing.T, res AssignmentsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Assignments, 2)
			},
		},
		{
			name: "first page error returns hard",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTagAssignments(gomock.Any(), gomock.Any()).
					Return(client.Result[components.TagAssignmentsList]{}, clientStructuredError("Permission denied", 403))
				return m
			},
			params: AssignmentsParams{TagID: identifiers.NewTagID(tagUUID.String())},
			assert: func(t *testing.T, res AssignmentsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Empty(t, res.Assignments)
				require.Contains(t, err.Error(), "Permission denied")
			},
		},
		{
			name: "later page error returns partial results",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
						TagID: tagUUID.String(),
					}).Return(assignmentsPage([]string{"8.8.8.8"}, 5, "token1"), nil),
					m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
						TagID: tagUUID.String(), PageToken: mo.Some("token1"),
					}).Return(client.Result[components.TagAssignmentsList]{}, clientStructuredError("Server error", 500)),
				)
				return m
			},
			params: AssignmentsParams{TagID: identifiers.NewTagID(tagUUID.String())},
			assert: func(t *testing.T, res AssignmentsResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Assignments, 1)
				require.Error(t, res.PartialError)
				require.Contains(t, res.PartialError.Error(), "Server error")
			},
		},
		{
			name: "invalid order-by rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl)
			},
			params: AssignmentsParams{
				TagID:   identifiers.NewTagID(tagUUID.String()),
				OrderBy: mo.Some("name_asc"),
			},
			assert: func(t *testing.T, _ AssignmentsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "create_time_asc")
			},
		},
		{
			name: "invalid asset-type rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl)
			},
			params: AssignmentsParams{
				TagID:     identifiers.NewTagID(tagUUID.String()),
				AssetType: mo.Some("hosts"),
			},
			assert: func(t *testing.T, _ AssignmentsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "web_property")
			},
		},
		{
			name: "impossible time window rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl)
			},
			params: AssignmentsParams{
				TagID:         identifiers.NewTagID(tagUUID.String()),
				CreatedBefore: mo.Some(after),
				CreatedAfter:  mo.Some(before),
			},
			assert: func(t *testing.T, _ AssignmentsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "created-before must be after created-after")
			},
		},
		{
			name: "zero page size rejected before any request",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl)
			},
			params: AssignmentsParams{
				TagID:    identifiers.NewTagID(tagUUID.String()),
				PageSize: mo.Some(uint64(0)),
			},
			assert: func(t *testing.T, _ AssignmentsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "page size")
			},
		},
		{
			name: "empty identifier rejected before any lookup",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl)
			},
			params: AssignmentsParams{TagID: identifiers.NewTagID("  ")},
			assert: func(t *testing.T, _ AssignmentsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "tag name or ID is required")
			},
		},
		{
			name: "unresolvable name returns tag-not-found",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListTags(gomock.Any(), gomock.Any()).
					Return(client.Result[components.TagsList]{Data: &components.TagsList{}}, nil)
				m.EXPECT().ListTagAssignments(gomock.Any(), gomock.Any()).Times(0)
				return m
			},
			params: AssignmentsParams{TagID: identifiers.NewTagID("missing")},
			assert: func(t *testing.T, _ AssignmentsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), `tag "missing" not found`)
			},
		},
		{
			name: "context cancellation propagates",
			client: func(ctrl *gomock.Controller) client.Client {
				return mocks.NewMockClient(ctrl)
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			params: AssignmentsParams{TagID: identifiers.NewTagID(tagUUID.String())},
			assert: func(t *testing.T, _ AssignmentsResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.ErrorIs(t, err, context.Canceled)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := New(tc.client(ctrl))

			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}

			res, err := svc.ListAssignments(ctx, tc.params)
			tc.assert(t, res, err)
		})
	}
}

// TestTagsService_ListAssignments_Streaming verifies assignments are emitted
// instead of collected when a streaming emitter is attached.
func TestTagsService_ListAssignments_Streaming(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tagUUID := uuid.New()
	m := mocks.NewMockClient(ctrl)
	gomock.InOrder(
		m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
			TagID: tagUUID.String(), PageSize: mo.Some(int64(2)),
		}).Return(assignmentsPage([]string{"8.8.8.8", "1.1.1.1"}, 3, "token1"), nil),
		m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
			TagID: tagUUID.String(), PageSize: mo.Some(int64(2)), PageToken: mo.Some("token1"),
		}).Return(assignmentsPage([]string{"9.9.9.9"}, 3, ""), nil),
	)

	emitter, items := streaming.NewChannelEmitter(8)
	ctx := streaming.WithEmitter(context.Background(), emitter)

	res, err := New(m).ListAssignments(ctx, AssignmentsParams{
		TagID:    identifiers.NewTagID(tagUUID.String()),
		PageSize: mo.Some(uint64(2)),
	})
	require.NoError(t, err)
	require.Empty(t, res.Assignments, "streamed assignments must not also be collected")
	require.Equal(t, int64(3), res.TotalSize)

	emitter.Close(nil)
	var streamed []string
	for item := range items {
		if item.Done {
			break
		}
		assignment, ok := item.Data.(Assignment)
		require.True(t, ok)
		streamed = append(streamed, assignment.AssetID)
	}
	require.Equal(t, []string{"8.8.8.8", "1.1.1.1", "9.9.9.9"}, streamed)
}

func TestTagsService_GetTag_AssetCount(t *testing.T) {
	tagUUID := uuid.New()
	tagResult := client.Result[components.Tag]{
		Metadata: okMeta(),
		Data:     &components.Tag{ID: tagUUID.String(), Name: "my-tag", Privacy: components.TagPrivacyPrivate},
	}

	t.Run("counts assignments off the tag's own UUID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockClient(ctrl)
		m.EXPECT().GetTag(gomock.Any(), mo.None[string](), "my-tag").Return(tagResult, nil)
		// The count uses the UUID from the tag just fetched, so a get by name
		// still needs no separate resolve.
		m.EXPECT().ListTagAssignments(gomock.Any(), client.ListTagAssignmentsRequest{
			TagID:    tagUUID.String(),
			PageSize: mo.Some(int64(1)),
		}).Return(assignmentsPage([]string{"8.8.8.8"}, 7, ""), nil)

		res, err := New(m).GetTag(context.Background(), GetParams{
			TagID: identifiers.NewTagID("my-tag"),
		})
		require.NoError(t, err)
		require.NotNil(t, res.Tag.AssetCount)
		require.Equal(t, int64(7), *res.Tag.AssetCount)
		require.NoError(t, res.PartialError)
	})

	t.Run("a zero count is reported, not omitted", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockClient(ctrl)
		m.EXPECT().GetTag(gomock.Any(), mo.None[string](), "my-tag").Return(tagResult, nil)
		m.EXPECT().ListTagAssignments(gomock.Any(), gomock.Any()).
			Return(assignmentsPage(nil, 0, ""), nil)

		res, err := New(m).GetTag(context.Background(), GetParams{
			TagID: identifiers.NewTagID("my-tag"),
		})
		require.NoError(t, err)
		// Non-nil, so an untagged tag renders "Assets: 0" rather than dropping
		// the row and reading like the count was never taken.
		require.NotNil(t, res.Tag.AssetCount)
		require.Equal(t, int64(0), *res.Tag.AssetCount)
	})

	t.Run("count failure keeps the tag and reports a partial error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockClient(ctrl)
		m.EXPECT().GetTag(gomock.Any(), mo.None[string](), "my-tag").Return(tagResult, nil)
		m.EXPECT().ListTagAssignments(gomock.Any(), gomock.Any()).
			Return(client.Result[components.TagAssignmentsList]{}, clientStructuredError("Permission denied", 403))

		res, err := New(m).GetTag(context.Background(), GetParams{
			TagID: identifiers.NewTagID("my-tag"),
		})
		require.NoError(t, err)
		require.Equal(t, "my-tag", res.Tag.Name)
		require.Nil(t, res.Tag.AssetCount)
		require.Error(t, res.PartialError)
		require.Contains(t, res.PartialError.Error(), "Permission denied")
	})

	t.Run("a tag-less response is not counted", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockClient(ctrl)
		m.EXPECT().GetTag(gomock.Any(), mo.None[string](), "my-tag").
			Return(client.Result[components.Tag]{Metadata: okMeta()}, nil)
		// With no tag ID there is nothing to count against, so the assignments
		// endpoint must not be asked.
		m.EXPECT().ListTagAssignments(gomock.Any(), gomock.Any()).Times(0)

		res, err := New(m).GetTag(context.Background(), GetParams{
			TagID: identifiers.NewTagID("my-tag"),
		})
		require.NoError(t, err)
		require.Nil(t, res.Tag.AssetCount)
		require.NoError(t, res.PartialError)
	})
}
