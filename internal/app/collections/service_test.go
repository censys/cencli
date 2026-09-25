package collections

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
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

const testCollectionID = "550e8400-e29b-41d4-a716-446655440000"

func strPtr(s string) *string { return &s }

func okMeta() client.Metadata {
	return client.Metadata{
		Request:  &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.censys.io"}},
		Response: &http.Response{StatusCode: 200},
		Latency:  100 * time.Millisecond,
	}
}

func collectionID() identifiers.CollectionID {
	return identifiers.NewCollectionID(uuid.MustParse(testCollectionID))
}

func sdkCollection(name string) components.Collection {
	return components.Collection{
		ID:          name + "-id",
		Name:        name,
		Description: name + " description",
		Query:       "host.services.protocol=SSH",
		Status:      components.CollectionStatusActive,
		TotalAssets: 42,
		CreateTime:  time.Unix(0, 0).UTC(),
	}
}

func collectionPage(names []string, next string) client.Result[components.ListCollectionsResponseV1] {
	items := make([]components.Collection, 0, len(names))
	for _, n := range names {
		items = append(items, sdkCollection(n))
	}
	return client.Result[components.ListCollectionsResponseV1]{
		Metadata: okMeta(),
		Data:     &components.ListCollectionsResponseV1{Collections: items, NextPageToken: next},
	}
}

func TestCollectionsService_ListCollections(t *testing.T) {
	testCases := []struct {
		name   string
		client func(ctrl *gomock.Controller) client.Client
		params ListParams
		assert func(t *testing.T, res ListResult, err cenclierrors.CencliError)
	}{
		{
			name: "success - single page, no filters",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{}).
					Return(collectionPage([]string{"alpha", "beta"}, ""), nil)
				return m
			},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Collections, 2)
				require.Equal(t, "alpha", res.Collections[0].Name)
				require.Equal(t, "active", res.Collections[0].Status)
				require.Equal(t, int64(42), res.Collections[0].TotalAssets)
				require.NotNil(t, res.Meta)
				require.False(t, res.HasMore)
			},
		},
		{
			name:   "comma-separated statuses are sent as a list",
			params: ListParams{Statuses: []string{"active", "paused"}},
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{Statuses: []string{"active", "paused"}}).
					Return(collectionPage([]string{"alpha"}, ""), nil)
				return m
			},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Collections, 1)
			},
		},
		{
			name:   "org ID and page size are passed through",
			params: ListParams{OrgID: mo.Some(identifiers.NewOrganizationID(uuid.MustParse(testCollectionID))), PageSize: mo.Some[uint64](10)},
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{
					OrgID:    mo.Some(testCollectionID),
					PageSize: mo.Some[int64](10),
				}).Return(collectionPage([]string{"alpha"}, ""), nil)
				return m
			},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
			},
		},
		{
			name:   "stops at max pages",
			params: ListParams{MaxPages: mo.Some[uint64](1)},
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				m.EXPECT().ListCollections(gomock.Any(), gomock.Any()).
					Return(collectionPage([]string{"alpha"}, "next"), nil).Times(1)
				return m
			},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Collections, 1)
				require.True(t, res.HasMore)
			},
		},
		{
			name: "error on page 2 is a partial error",
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				gomock.InOrder(
					m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{}).
						Return(collectionPage([]string{"alpha"}, "next"), nil),
					m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{PageToken: mo.Some("next")}).
						Return(client.Result[components.ListCollectionsResponseV1]{}, client.NewClientError(errors.New("boom"))),
				)
				return m
			},
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Len(t, res.Collections, 1)
				require.NotNil(t, res.PartialError)
			},
		},
		{
			name:   "unknown status fails before any request",
			params: ListParams{Statuses: []string{"Active"}},
			client: func(ctrl *gomock.Controller) client.Client { return mocks.NewMockClient(ctrl) },
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid status 'Active'")
				require.Contains(t, err.Error(), "populating, active, paused, archived")
			},
		},
		{
			name:   "zero page size fails before any request",
			params: ListParams{PageSize: mo.Some[uint64](0)},
			client: func(ctrl *gomock.Controller) client.Client { return mocks.NewMockClient(ctrl) },
			assert: func(t *testing.T, res ListResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "page size must be greater than 0")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			svc := New(tc.client(ctrl))
			res, err := svc.ListCollections(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

func TestCollectionsService_GetCollection(t *testing.T) {
	t.Run("success maps every field", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		sdk := sdkCollection("alpha")
		reason := components.StatusReasonNotEnoughCredits
		sdk.StatusReason = &reason
		sdk.CreatedBy = strPtr("user-1")
		sdk.AddedAssets24Hours = 3
		sdk.RemovedAssets24Hours = 1
		m.EXPECT().GetCollection(gomock.Any(), mo.None[string](), testCollectionID).
			Return(client.Result[components.Collection]{Metadata: okMeta(), Data: &sdk}, nil)

		res, err := New(m).GetCollection(context.Background(), GetParams{CollectionID: collectionID()})
		require.NoError(t, err)
		require.Equal(t, Collection{
			ID:                   "alpha-id",
			Name:                 "alpha",
			Description:          "alpha description",
			Query:                "host.services.protocol=SSH",
			Status:               "active",
			StatusReason:         strPtr("not_enough_credits"),
			TotalAssets:          42,
			AddedAssets24Hours:   3,
			RemovedAssets24Hours: 1,
			CreatedBy:            strPtr("user-1"),
			CreateTime:           time.Unix(0, 0).UTC(),
		}, res.Collection)
		require.NotNil(t, res.Meta)
	})

	t.Run("client error is returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		m.EXPECT().GetCollection(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(client.Result[components.Collection]{}, client.NewClientError(errors.New("boom")))
		_, err := New(m).GetCollection(context.Background(), GetParams{CollectionID: collectionID()})
		require.Error(t, err)
	})
}

func TestCollectionsService_CreateCollection(t *testing.T) {
	testCases := []struct {
		name   string
		params CreateParams
		client func(ctrl *gomock.Controller) client.Client
		assert func(t *testing.T, res CreateResult, err cenclierrors.CencliError)
	}{
		{
			name:   "success with description",
			params: CreateParams{Name: "alpha", Query: "host.services.protocol=SSH", Description: mo.Some("desc")},
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				c := sdkCollection("alpha")
				m.EXPECT().CreateCollection(gomock.Any(), client.CreateCollectionRequest{
					Name: "alpha", Query: "host.services.protocol=SSH", Description: mo.Some("desc"),
				}).Return(client.Result[components.Collection]{Metadata: okMeta(), Data: &c}, nil)
				return m
			},
			assert: func(t *testing.T, res CreateResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.Equal(t, "alpha-id", res.Collection.ID)
			},
		},
		{
			name:   "success without description",
			params: CreateParams{Name: "alpha", Query: "host.services.protocol=SSH"},
			client: func(ctrl *gomock.Controller) client.Client {
				m := mocks.NewMockClient(ctrl)
				c := sdkCollection("alpha")
				m.EXPECT().CreateCollection(gomock.Any(), client.CreateCollectionRequest{
					Name: "alpha", Query: "host.services.protocol=SSH",
				}).Return(client.Result[components.Collection]{Metadata: okMeta(), Data: &c}, nil)
				return m
			},
			assert: func(t *testing.T, res CreateResult, err cenclierrors.CencliError) {
				require.NoError(t, err)
			},
		},
		{
			name:   "blank name fails before any request",
			params: CreateParams{Name: "   ", Query: "x"},
			client: func(ctrl *gomock.Controller) client.Client { return mocks.NewMockClient(ctrl) },
			assert: func(t *testing.T, res CreateResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "collection name must not be empty")
			},
		},
		{
			name:   "blank query fails before any request",
			params: CreateParams{Name: "alpha", Query: "   "},
			client: func(ctrl *gomock.Controller) client.Client { return mocks.NewMockClient(ctrl) },
			assert: func(t *testing.T, res CreateResult, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "--query must not be empty")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			res, err := New(tc.client(ctrl)).CreateCollection(context.Background(), tc.params)
			tc.assert(t, res, err)
		})
	}
}

// clientStructuredError builds a structured client error carrying the given
// status code, the same as clientStructuredError in tags/service_test.go.
func clientStructuredError(detail string, status int64) client.ClientError {
	return client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{Detail: &detail, Status: &status})
}

func TestCollectionsService_CreateCollection_Limit(t *testing.T) {
	createParams := CreateParams{Name: "alpha", Query: "host.services.protocol=SSH"}

	t.Run("412 with a count", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		gomock.InOrder(
			m.EXPECT().CreateCollection(gomock.Any(), gomock.Any()).
				Return(client.Result[components.Collection]{}, clientStructuredError("Collection limit exceeded", 412)),
			m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{Statuses: []string{"populating", "active", "paused"}, PageSize: mo.Some[int64](100)}).
				Return(collectionPage([]string{"a", "b", "c"}, ""), nil),
		)

		_, err := New(m).CreateCollection(context.Background(), createParams)
		require.Error(t, err)
		require.Contains(t, err.Error(), "reached its collection limit (3 collections count toward it; archived collections do not)")

		var limitErr *collectionLimitError
		require.True(t, errors.As(err, &limitErr))
	})

	t.Run("412 across two pages", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		gomock.InOrder(
			m.EXPECT().CreateCollection(gomock.Any(), gomock.Any()).
				Return(client.Result[components.Collection]{}, clientStructuredError("Collection limit exceeded", 412)),
			m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{Statuses: []string{"populating", "active", "paused"}, PageSize: mo.Some[int64](100)}).
				Return(collectionPage([]string{"a", "b"}, "t"), nil),
			m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{Statuses: []string{"populating", "active", "paused"}, PageSize: mo.Some[int64](100), PageToken: mo.Some("t")}).
				Return(collectionPage([]string{"c"}, ""), nil),
		)

		_, err := New(m).CreateCollection(context.Background(), createParams)
		require.Error(t, err)
		require.Contains(t, err.Error(), "reached its collection limit (3 collections count toward it; archived collections do not)")
	})

	t.Run("412 whose count fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		gomock.InOrder(
			m.EXPECT().CreateCollection(gomock.Any(), gomock.Any()).
				Return(client.Result[components.Collection]{}, clientStructuredError("Collection limit exceeded", 412)),
			m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{Statuses: []string{"populating", "active", "paused"}, PageSize: mo.Some[int64](100)}).
				Return(client.Result[components.ListCollectionsResponseV1]{}, client.NewClientError(errors.New("boom"))),
		)

		_, err := New(m).CreateCollection(context.Background(), createParams)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "collections count toward it")
		require.Contains(t, err.Error(), "reached its collection limit (archived collections do not count toward it)")
	})

	t.Run("412 whose count fails on page 2 (partial)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		gomock.InOrder(
			m.EXPECT().CreateCollection(gomock.Any(), gomock.Any()).
				Return(client.Result[components.Collection]{}, clientStructuredError("Collection limit exceeded", 412)),
			m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{Statuses: []string{"populating", "active", "paused"}, PageSize: mo.Some[int64](100)}).
				Return(collectionPage([]string{"a"}, "t"), nil),
			m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{Statuses: []string{"populating", "active", "paused"}, PageSize: mo.Some[int64](100), PageToken: mo.Some("t")}).
				Return(client.Result[components.ListCollectionsResponseV1]{}, client.NewClientError(errors.New("boom"))),
		)

		_, err := New(m).CreateCollection(context.Background(), createParams)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "collections count toward it")
		require.Contains(t, err.Error(), "reached its collection limit (archived collections do not count toward it)")
	})

	t.Run("non-412 error passes through", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		wantErr := clientStructuredError("invalid query", 422)
		// No ListCollections expectation: gomock fails the test if it is called.
		m.EXPECT().CreateCollection(gomock.Any(), gomock.Any()).
			Return(client.Result[components.Collection]{}, wantErr)

		_, err := New(m).CreateCollection(context.Background(), createParams)
		require.Error(t, err)
		require.Equal(t, wantErr.Error(), err.Error())

		var limitErr *collectionLimitError
		require.False(t, errors.As(err, &limitErr))
	})

	t.Run("the org ID is passed to the count", func(t *testing.T) {
		orgID := identifiers.NewOrganizationID(uuid.MustParse(testCollectionID))
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		gomock.InOrder(
			m.EXPECT().CreateCollection(gomock.Any(), gomock.Any()).
				Return(client.Result[components.Collection]{}, clientStructuredError("Collection limit exceeded", 412)),
			m.EXPECT().ListCollections(gomock.Any(), client.ListCollectionsRequest{
				OrgID:    mo.Some(testCollectionID),
				Statuses: []string{"populating", "active", "paused"},
				PageSize: mo.Some[int64](100),
			}).Return(collectionPage([]string{"a"}, ""), nil),
		)

		_, err := New(m).CreateCollection(context.Background(), CreateParams{
			OrgID: mo.Some(orgID), Name: "alpha", Query: "host.services.protocol=SSH",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "1 collections count toward it")
	})
}

func TestCollectionsService_DeleteCollection(t *testing.T) {
	t.Run("success echoes the ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		m.EXPECT().DeleteCollection(gomock.Any(), mo.None[string](), testCollectionID).Return(okMeta(), nil)
		res, err := New(m).DeleteCollection(context.Background(), DeleteParams{CollectionID: collectionID()})
		require.NoError(t, err)
		require.Equal(t, testCollectionID, res.CollectionID)
		require.NotNil(t, res.Meta)
	})

	t.Run("client error is returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		m.EXPECT().DeleteCollection(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(client.Metadata{}, client.NewClientError(errors.New("boom")))
		_, err := New(m).DeleteCollection(context.Background(), DeleteParams{CollectionID: collectionID()})
		require.Error(t, err)
	})
}

func TestCollectionsService_UpdateCollection(t *testing.T) {
	current := sdkCollection("alpha") // Name "alpha", Query "host.services.protocol=SSH", Description "alpha description"

	testCases := []struct {
		name        string
		params      UpdateParams
		wantRequest client.UpdateCollectionRequest
	}{
		{
			name:   "name only keeps query and description",
			params: UpdateParams{CollectionID: collectionID(), Name: mo.Some("renamed")},
			wantRequest: client.UpdateCollectionRequest{
				CollectionID: testCollectionID, Name: "renamed",
				Query: "host.services.protocol=SSH", Description: mo.Some("alpha description"),
			},
		},
		{
			name:   "query only keeps name and description",
			params: UpdateParams{CollectionID: collectionID(), Query: mo.Some("host.ip=1.1.1.1")},
			wantRequest: client.UpdateCollectionRequest{
				CollectionID: testCollectionID, Name: "alpha",
				Query: "host.ip=1.1.1.1", Description: mo.Some("alpha description"),
			},
		},
		{
			name:   "description only keeps name and query",
			params: UpdateParams{CollectionID: collectionID(), Description: mo.Some("new desc")},
			wantRequest: client.UpdateCollectionRequest{
				CollectionID: testCollectionID, Name: "alpha",
				Query: "host.services.protocol=SSH", Description: mo.Some("new desc"),
			},
		},
		{
			name:   "clear description sends an empty description",
			params: UpdateParams{CollectionID: collectionID(), Description: mo.Some("")},
			wantRequest: client.UpdateCollectionRequest{
				CollectionID: testCollectionID, Name: "alpha",
				Query: "host.services.protocol=SSH", Description: mo.Some(""),
			},
		},
		{
			name: "all fields replace all values",
			params: UpdateParams{
				CollectionID: collectionID(), Name: mo.Some("n"), Query: mo.Some("q"), Description: mo.Some("d"),
			},
			wantRequest: client.UpdateCollectionRequest{
				CollectionID: testCollectionID, Name: "n", Query: "q", Description: mo.Some("d"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks.NewMockClient(ctrl)
			got := current
			updated := sdkCollection("updated")
			gomock.InOrder(
				m.EXPECT().GetCollection(gomock.Any(), mo.None[string](), testCollectionID).
					Return(client.Result[components.Collection]{Metadata: okMeta(), Data: &got}, nil),
				m.EXPECT().UpdateCollection(gomock.Any(), tc.wantRequest).
					Return(client.Result[components.Collection]{Metadata: okMeta(), Data: &updated}, nil),
			)
			res, err := New(m).UpdateCollection(context.Background(), tc.params)
			require.NoError(t, err)
			require.Equal(t, "updated", res.Collection.Name)
			require.NotNil(t, res.Meta)
		})
	}

	t.Run("failed GET sends no update", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		m.EXPECT().GetCollection(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(client.Result[components.Collection]{}, client.NewClientError(errors.New("boom")))
		// No UpdateCollection expectation: gomock fails the test if it is called.
		_, err := New(m).UpdateCollection(context.Background(), UpdateParams{CollectionID: collectionID(), Name: mo.Some("x")})
		require.Error(t, err)
	})

	t.Run("GET returns nil data sends no update", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		m.EXPECT().GetCollection(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(client.Result[components.Collection]{Metadata: okMeta(), Data: nil}, nil)
		_, err := New(m).UpdateCollection(context.Background(), UpdateParams{CollectionID: collectionID(), Name: mo.Some("x")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "the update was not sent")
	})

	t.Run("blank current query blocks a name-only update", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		blank := current
		blank.Query = ""
		m.EXPECT().GetCollection(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(client.Result[components.Collection]{Metadata: okMeta(), Data: &blank}, nil)
		// No UpdateCollection expectation: gomock fails the test if it is called.
		_, err := New(m).UpdateCollection(context.Background(), UpdateParams{CollectionID: collectionID(), Name: mo.Some("renamed")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "the update was not sent")
	})

	t.Run("caller's query fills a blank current query", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		blank := current
		blank.Query = ""
		updated := sdkCollection("updated")
		gomock.InOrder(
			m.EXPECT().GetCollection(gomock.Any(), mo.None[string](), testCollectionID).
				Return(client.Result[components.Collection]{Metadata: okMeta(), Data: &blank}, nil),
			m.EXPECT().UpdateCollection(gomock.Any(), client.UpdateCollectionRequest{
				CollectionID: testCollectionID, Name: "alpha",
				Query: "q", Description: mo.Some("alpha description"),
			}).Return(client.Result[components.Collection]{Metadata: okMeta(), Data: &updated}, nil),
		)
		res, err := New(m).UpdateCollection(context.Background(), UpdateParams{CollectionID: collectionID(), Query: mo.Some("q")})
		require.NoError(t, err)
		require.Equal(t, "updated", res.Collection.Name)
	})

	t.Run("failed update is returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		m := mocks.NewMockClient(ctrl)
		got := current
		m.EXPECT().GetCollection(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(client.Result[components.Collection]{Metadata: okMeta(), Data: &got}, nil)
		m.EXPECT().UpdateCollection(gomock.Any(), gomock.Any()).
			Return(client.Result[components.Collection]{}, client.NewClientError(errors.New("boom")))
		_, err := New(m).UpdateCollection(context.Background(), UpdateParams{CollectionID: collectionID(), Name: mo.Some("x")})
		require.Error(t, err)
	})
}
