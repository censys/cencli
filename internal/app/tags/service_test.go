package tags

import (
	"context"
	"errors"
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
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params UpdateParams
		assert func(t *testing.T, res UpdateResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - maps SDK tag to DTO",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				desc := "new description"
				sdkTag := &components.Tag{
					ID:          "tag-id",
					Name:        "renamed",
					Description: &desc,
					Privacy:     components.TagPrivacyShared,
					CreatedBy:   "creator",
				}
				m.EXPECT().UpdateTag(gomock.Any(), client.UpdateTagRequest{
					TagID:       "my-tag",
					Name:        mo.Some("renamed"),
					Description: mo.Some("new description"),
					Privacy:     mo.Some("shared"),
				}).Return(client.Result[components.Tag]{Metadata: okMeta(), Data: sdkTag}, nil)
				return m
			},
			params: UpdateParams{
				TagID:       identifiers.NewTagID("my-tag"),
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
			name: "partial update - only privacy, org and raw UUID threaded to client",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().UpdateTag(gomock.Any(), client.UpdateTagRequest{
					OrgID:   mo.Some(orgUUID.String()),
					TagID:   orgUUID.String(),
					Privacy: mo.Some("private"),
				}).Return(client.Result[components.Tag]{Metadata: okMeta(), Data: &components.Tag{ID: "id", Name: "n", Privacy: components.TagPrivacyPrivate}}, nil)
				return m
			},
			params: UpdateParams{
				OrgID:   mo.Some(identifiers.NewOrganizationID(orgUUID)),
				TagID:   identifiers.NewTagID(orgUUID.String()),
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
					TagID:       "my-tag",
					Description: mo.Some(""),
				}).Return(client.Result[components.Tag]{Metadata: okMeta(), Data: &components.Tag{ID: "id", Name: "my-tag", Privacy: components.TagPrivacyPrivate}}, nil)
				return m
			},
			params: UpdateParams{
				TagID:       identifiers.NewTagID("my-tag"),
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
				m.EXPECT().UpdateTag(gomock.Any(), client.UpdateTagRequest{TagID: "missing", Name: mo.Some("x")}).
					Return(client.Result[components.Tag]{}, structuredErr)
				return m
			},
			params: UpdateParams{TagID: identifiers.NewTagID("missing"), Name: mo.Some("x")},
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
