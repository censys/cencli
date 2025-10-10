package aggregate

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/sdkerrors"
	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/censys/cencli/gen/client/mocks"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
)

func TestAggregateService(t *testing.T) {
	testCases := []struct {
		name         string
		client       func(ctrl *gomock.Controller) client.Client
		collectionID mo.Option[identifiers.CollectionID]
		orgID        mo.Option[identifiers.OrganizationID]
		params       Params
		ctx          func() context.Context
		assert       func(t *testing.T, res Result, err cenclierrors.CencliError)
	}{
		{
			name:         "success - no org - basic parameters",
			collectionID: mo.None[identifiers.CollectionID](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Aggregate(
					gomock.Any(),
					mo.None[string](),
					"ip:1.1.1.1",
					"location.country",
					int64(10),
					mo.None[string](),
					mo.None[bool](),
				).Return(client.Result[components.SearchAggregateResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
					},
					Data: &components.SearchAggregateResponse{
						Buckets: []components.SearchAggregateResponseBucket{
							{Key: "US", Count: 100},
							{Key: "CA", Count: 50},
						},
						TotalCount: 150,
					},
				}, nil)
				return mockClient
			},
			orgID:  mo.None[identifiers.OrganizationID](),
			params: Params{Query: "ip:1.1.1.1", Field: "location.country", NumBuckets: 10, CountByLevel: mo.None[CountByLevel](), FilterByQuery: mo.None[bool]()},
			ctx:    nil,
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Equal(t, 100*time.Millisecond, res.Meta.Latency)
				require.Len(t, res.Buckets, 2)
				require.Equal(t, "US", res.Buckets[0].Key)
				require.Equal(t, uint64(100), res.Buckets[0].Count)
				require.Equal(t, "CA", res.Buckets[1].Key)
				require.Equal(t, uint64(50), res.Buckets[1].Count)
			},
		},
		{
			name:         "success - with org - all optional parameters",
			collectionID: mo.None[identifiers.CollectionID](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Aggregate(
					gomock.Any(),
					mo.Some("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
					"services.service_name:HTTP",
					"services.port",
					int64(25),
					mo.Some("service"),
					mo.Some(true),
				).Return(client.Result[components.SearchAggregateResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  200 * time.Millisecond,
					},
					Data: &components.SearchAggregateResponse{
						Buckets: []components.SearchAggregateResponseBucket{
							{Key: "80", Count: 1000},
							{Key: "443", Count: 800},
							{Key: "8080", Count: 200},
						},
						TotalCount: 2000,
					},
				}, nil)
				return mockClient
			},
			orgID:  mo.Some(identifiers.NewOrganizationID(uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"))),
			params: Params{Query: "services.service_name:HTTP", Field: "services.port", NumBuckets: 25, CountByLevel: mo.Some(CountByLevel("service")), FilterByQuery: mo.Some(true)},
			ctx:    nil,
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Equal(t, 200*time.Millisecond, res.Meta.Latency)
				require.Len(t, res.Buckets, 3)
				require.Equal(t, "80", res.Buckets[0].Key)
				require.Equal(t, uint64(1000), res.Buckets[0].Count)
				require.Equal(t, "443", res.Buckets[1].Key)
				require.Equal(t, uint64(800), res.Buckets[1].Count)
				require.Equal(t, "8080", res.Buckets[2].Key)
				require.Equal(t, uint64(200), res.Buckets[2].Count)
			},
		},
		{
			name:         "success - empty buckets",
			collectionID: mo.None[identifiers.CollectionID](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Aggregate(
					gomock.Any(),
					mo.None[string](),
					"nonexistent:field",
					"location.country",
					int64(5),
					mo.None[string](),
					mo.None[bool](),
				).Return(client.Result[components.SearchAggregateResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  50 * time.Millisecond,
					},
					Data: &components.SearchAggregateResponse{
						Buckets:    []components.SearchAggregateResponseBucket{},
						TotalCount: 0,
					},
				}, nil)
				return mockClient
			},
			orgID:  mo.None[identifiers.OrganizationID](),
			params: Params{Query: "nonexistent:field", Field: "location.country", NumBuckets: 5, CountByLevel: mo.None[CountByLevel](), FilterByQuery: mo.None[bool]()},
			ctx:    nil,
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Equal(t, 50*time.Millisecond, res.Meta.Latency)
				require.Len(t, res.Buckets, 0)
			},
		},
		{
			name:         "error - structured client error",
			collectionID: mo.None[identifiers.CollectionID](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				detail := "Invalid query syntax"
				status := int64(400)
				structuredErr := client.NewCensysClientStructuredError(&sdkerrors.ErrorModel{
					Detail: &detail,
					Status: &status,
				})
				mockClient.EXPECT().Aggregate(
					gomock.Any(),
					mo.None[string](),
					"invalid query syntax",
					"location.country",
					int64(10),
					mo.None[string](),
					mo.None[bool](),
				).Return(client.Result[components.SearchAggregateResponse]{}, structuredErr)
				return mockClient
			},
			orgID:  mo.None[identifiers.OrganizationID](),
			params: Params{Query: "invalid query syntax", Field: "location.country", NumBuckets: 10, CountByLevel: mo.None[CountByLevel](), FilterByQuery: mo.None[bool]()},
			ctx:    nil,
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, Result{}, res)

				var structuredErr client.ClientStructuredError
				require.True(t, errors.As(err, &structuredErr))
				require.True(t, structuredErr.StatusCode().IsPresent())
				require.Equal(t, int64(400), structuredErr.StatusCode().MustGet())
			},
		},
		{
			name:         "error - generic client error",
			collectionID: mo.None[identifiers.CollectionID](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				genericErr := client.NewCensysClientGenericError(&sdkerrors.SDKError{
					Message:    "Internal server error",
					StatusCode: 500,
					Body:       "Server temporarily unavailable",
				})
				mockClient.EXPECT().Aggregate(
					gomock.Any(),
					mo.None[string](),
					"ip:1.1.1.1",
					"location.country",
					int64(10),
					mo.None[string](),
					mo.None[bool](),
				).Return(client.Result[components.SearchAggregateResponse]{}, genericErr)
				return mockClient
			},
			orgID:  mo.None[identifiers.OrganizationID](),
			params: Params{Query: "ip:1.1.1.1", Field: "location.country", NumBuckets: 10, CountByLevel: mo.None[CountByLevel](), FilterByQuery: mo.None[bool]()},
			ctx:    nil,
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, Result{}, res)

				var genericErr client.ClientGenericError
				require.True(t, errors.As(err, &genericErr))
				require.Equal(t, int64(500), genericErr.StatusCode().MustGet())
			},
		},
		{
			name:         "error - unknown client error",
			collectionID: mo.None[identifiers.CollectionID](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				unknownErr := client.NewClientError(errors.New("network timeout"))
				mockClient.EXPECT().Aggregate(
					gomock.Any(),
					mo.None[string](),
					"ip:1.1.1.1",
					"location.country",
					int64(10),
					mo.None[string](),
					mo.None[bool](),
				).Return(client.Result[components.SearchAggregateResponse]{}, unknownErr)
				return mockClient
			},
			orgID:  mo.None[identifiers.OrganizationID](),
			params: Params{Query: "ip:1.1.1.1", Field: "location.country", NumBuckets: 10, CountByLevel: mo.None[CountByLevel](), FilterByQuery: mo.None[bool]()},
			ctx:    nil,
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, Result{}, res)

				var unknownErr client.ClientError
				require.True(t, errors.As(err, &unknownErr))
				require.Contains(t, err.Error(), "network timeout")
			},
		},
		{
			name:         "context cancellation propagation",
			collectionID: mo.None[identifiers.CollectionID](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Aggregate(
					gomock.Any(),
					mo.None[string](),
					"ip:1.1.1.1",
					"location.country",
					int64(10),
					mo.None[string](),
					mo.None[bool](),
				).DoAndReturn(func(ctx context.Context, orgID mo.Option[string], query, field string, numBuckets int64, countByLevel mo.Option[string], filterByQuery mo.Option[bool]) (client.Result[components.SearchAggregateResponse], client.ClientError) {
					// Verify context is passed through
					select {
					case <-ctx.Done():
						return client.Result[components.SearchAggregateResponse]{}, client.NewClientError(ctx.Err())
					default:
						// Context not cancelled yet, return success
						return client.Result[components.SearchAggregateResponse]{
							Metadata: client.Metadata{
								Request:  &http.Request{},
								Response: &http.Response{StatusCode: 200},
								Latency:  10 * time.Millisecond,
								Attempts: 1,
							},
							Data: &components.SearchAggregateResponse{
								Buckets: []components.SearchAggregateResponseBucket{
									{Key: "US", Count: 100},
								},
								TotalCount: 100,
							},
						}, nil
					}
				})
				return mockClient
			},
			orgID:  mo.None[identifiers.OrganizationID](),
			params: Params{Query: "ip:1.1.1.1", Field: "location.country", NumBuckets: 10, CountByLevel: mo.None[CountByLevel](), FilterByQuery: mo.None[bool]()},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				// Don't cancel immediately - let the call succeed to verify context is passed
				_ = cancel
				return ctx
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.NoError(t, err)
				require.NotNil(t, res.Meta)
				require.Len(t, res.Buckets, 1)
				require.Equal(t, "US", res.Buckets[0].Key)
				require.Equal(t, uint64(100), res.Buckets[0].Count)
			},
		},
		{
			name:         "context cancellation - cancelled context",
			collectionID: mo.None[identifiers.CollectionID](),
			client: func(ctrl *gomock.Controller) client.Client {
				mockClient := mocks.NewMockClient(ctrl)
				mockClient.EXPECT().Aggregate(
					gomock.Any(),
					mo.None[string](),
					"ip:1.1.1.1",
					"location.country",
					int64(10),
					mo.None[string](),
					mo.None[bool](),
				).DoAndReturn(func(ctx context.Context, orgID mo.Option[string], query, field string, numBuckets int64, countByLevel mo.Option[string], filterByQuery mo.Option[bool]) (client.Result[components.SearchAggregateResponse], client.ClientError) {
					// Verify context is cancelled
					select {
					case <-ctx.Done():
						return client.Result[components.SearchAggregateResponse]{}, client.NewClientError(ctx.Err())
					default:
						t.Error("Expected context to be cancelled")
						return client.Result[components.SearchAggregateResponse]{}, client.NewClientError(errors.New("context should have been cancelled"))
					}
				})
				return mockClient
			},
			orgID:  mo.None[identifiers.OrganizationID](),
			params: Params{Query: "ip:1.1.1.1", Field: "location.country", NumBuckets: 10, CountByLevel: mo.None[CountByLevel](), FilterByQuery: mo.None[bool]()},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx
			},
			assert: func(t *testing.T, res Result, err cenclierrors.CencliError) {
				require.Error(t, err)
				require.Equal(t, Result{}, res)
				require.Contains(t, err.Error(), "the operation's context was cancelled before it completed")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			client := tc.client(ctrl)
			svc := New(client)

			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx()
			}

			params := tc.params
			res, err := svc.Aggregate(ctx, Params{
				CollectionID:  tc.collectionID,
				OrgID:         tc.orgID,
				Query:         params.Query,
				Field:         params.Field,
				NumBuckets:    params.NumBuckets,
				CountByLevel:  params.CountByLevel,
				FilterByQuery: params.FilterByQuery,
			})
			tc.assert(t, res, err)
		})
	}
}

// TestParseBuckets tests the bucket parsing functionality separately
func TestParseBuckets(t *testing.T) {
	testCases := []struct {
		name     string
		input    []components.SearchAggregateResponseBucket
		expected []Bucket
	}{
		{
			name:     "empty buckets",
			input:    []components.SearchAggregateResponseBucket{},
			expected: []Bucket{},
		},
		{
			name: "single bucket",
			input: []components.SearchAggregateResponseBucket{
				{Key: "US", Count: 100},
			},
			expected: []Bucket{
				{Key: "US", Count: 100},
			},
		},
		{
			name: "multiple buckets",
			input: []components.SearchAggregateResponseBucket{
				{Key: "US", Count: 1000},
				{Key: "CA", Count: 500},
				{Key: "GB", Count: 250},
			},
			expected: []Bucket{
				{Key: "US", Count: 1000},
				{Key: "CA", Count: 500},
				{Key: "GB", Count: 250},
			},
		},
		{
			name: "zero count bucket",
			input: []components.SearchAggregateResponseBucket{
				{Key: "EMPTY", Count: 0},
			},
			expected: []Bucket{
				{Key: "EMPTY", Count: 0},
			},
		},
		{
			name: "large count values",
			input: []components.SearchAggregateResponseBucket{
				{Key: "LARGE", Count: 9223372036854775807}, // max int64
			},
			expected: []Bucket{
				{Key: "LARGE", Count: 9223372036854775807},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseBuckets(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestAggregateService_ParameterValidation tests that all parameters are correctly passed to the client
func TestAggregateService_ParameterValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockClient(ctrl)

	// Test with specific parameter values to ensure they're passed correctly
	expectedOrgID := mo.Some("12345678-1234-1234-1234-123456789abc")
	expectedQuery := "services.service_name:HTTP AND location.country:US"
	expectedField := "services.port"
	expectedNumBuckets := int64(50)
	expectedCountByLevel := mo.Some("protocol")
	expectedFilterByQuery := mo.Some(false)

	mockClient.EXPECT().Aggregate(
		gomock.Any(), // context
		expectedOrgID,
		expectedQuery,
		expectedField,
		expectedNumBuckets,
		expectedCountByLevel,
		expectedFilterByQuery,
	).Return(client.Result[components.SearchAggregateResponse]{
		Metadata: client.Metadata{
			Request:  &http.Request{},
			Response: &http.Response{StatusCode: 200},
			Latency:  100 * time.Millisecond,
		},
		Data: &components.SearchAggregateResponse{
			Buckets: []components.SearchAggregateResponseBucket{
				{Key: "80", Count: 100},
			},
			TotalCount: 100,
		},
	}, nil)

	svc := New(mockClient)

	// Call with the exact parameters we expect to be passed
	res, err := svc.Aggregate(
		context.Background(),
		Params{
			CollectionID: mo.None[identifiers.CollectionID](),
			OrgID:        mo.Some(identifiers.NewOrganizationID(uuid.MustParse("12345678-1234-1234-1234-123456789abc"))),
			Query:        expectedQuery,
			Field:        expectedField,
			NumBuckets:   expectedNumBuckets,
			CountByLevel: func() mo.Option[CountByLevel] {
				if expectedCountByLevel.IsPresent() {
					return mo.Some(CountByLevel(expectedCountByLevel.MustGet()))
				}
				return mo.None[CountByLevel]()
			}(),
			FilterByQuery: expectedFilterByQuery,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, res.Meta)
	require.Len(t, res.Buckets, 1)
	require.Equal(t, "80", res.Buckets[0].Key)
	require.Equal(t, uint64(100), res.Buckets[0].Count)
}

// TestAggregateService_CollectionAggregate tests that collection aggregation calls the correct client method
func TestAggregateService_CollectionAggregate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockClient(ctrl)

	// Test that when collectionID is provided, AggregateCollection is called
	collectionID := identifiers.NewCollectionID(uuid.MustParse("12345678-1234-1234-1234-123456789abc"))
	orgID := identifiers.NewOrganizationID(uuid.MustParse("87654321-4321-4321-4321-cba987654321"))

	mockClient.EXPECT().AggregateCollection(
		gomock.Any(),                                    // context
		"12345678-1234-1234-1234-123456789abc",          // collection ID as string
		mo.Some("87654321-4321-4321-4321-cba987654321"), // org ID as string
		"services.service_name:HTTP",
		"services.port",
		int64(20),
		mo.Some("protocol"),
		mo.Some(true),
	).Return(client.Result[components.SearchAggregateResponse]{
		Metadata: client.Metadata{
			Request:  &http.Request{},
			Response: &http.Response{StatusCode: 200},
			Latency:  150 * time.Millisecond,
		},
		Data: &components.SearchAggregateResponse{
			Buckets: []components.SearchAggregateResponseBucket{
				{Key: "80", Count: 500},
				{Key: "443", Count: 300},
			},
			TotalCount: 800,
		},
	}, nil)

	svc := New(mockClient)

	res, err := svc.Aggregate(
		context.Background(),
		Params{
			CollectionID:  mo.Some(collectionID),
			OrgID:         mo.Some(orgID),
			Query:         "services.service_name:HTTP",
			Field:         "services.port",
			NumBuckets:    20,
			CountByLevel:  mo.Some(CountByLevel("protocol")),
			FilterByQuery: mo.Some(true),
		},
	)

	require.NoError(t, err)
	require.NotNil(t, res.Meta)
	require.Equal(t, 150*time.Millisecond, res.Meta.Latency)
	require.Len(t, res.Buckets, 2)
	require.Equal(t, "80", res.Buckets[0].Key)
	require.Equal(t, uint64(500), res.Buckets[0].Count)
	require.Equal(t, "443", res.Buckets[1].Key)
	require.Equal(t, uint64(300), res.Buckets[1].Count)
}

// TestAggregateService_GlobalVsCollection tests that the service chooses the correct client method
func TestAggregateService_GlobalVsCollection(t *testing.T) {
	testCases := []struct {
		name         string
		collectionID mo.Option[identifiers.CollectionID]
		expectGlobal bool
	}{
		{
			name:         "no collection ID - uses global aggregate",
			collectionID: mo.None[identifiers.CollectionID](),
			expectGlobal: true,
		},
		{
			name:         "with collection ID - uses collection aggregate",
			collectionID: mo.Some(identifiers.NewCollectionID(uuid.MustParse("12345678-1234-1234-1234-123456789abc"))),
			expectGlobal: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mocks.NewMockClient(ctrl)

			if tc.expectGlobal {
				mockClient.EXPECT().Aggregate(
					gomock.Any(),
					mo.None[string](),
					"test query",
					"test.field",
					int64(10),
					mo.None[string](),
					mo.None[bool](),
				).Return(client.Result[components.SearchAggregateResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
					},
					Data: &components.SearchAggregateResponse{
						Buckets: []components.SearchAggregateResponseBucket{
							{Key: "test", Count: 1},
						},
						TotalCount: 1,
					},
				}, nil)
			} else {
				mockClient.EXPECT().AggregateCollection(
					gomock.Any(),
					"12345678-1234-1234-1234-123456789abc",
					mo.None[string](),
					"test query",
					"test.field",
					int64(10),
					mo.None[string](),
					mo.None[bool](),
				).Return(client.Result[components.SearchAggregateResponse]{
					Metadata: client.Metadata{
						Request:  &http.Request{},
						Response: &http.Response{StatusCode: 200},
						Latency:  100 * time.Millisecond,
					},
					Data: &components.SearchAggregateResponse{
						Buckets: []components.SearchAggregateResponseBucket{
							{Key: "test", Count: 1},
						},
						TotalCount: 1,
					},
				}, nil)
			}

			svc := New(mockClient)

			res, err := svc.Aggregate(
				context.Background(),
				Params{
					CollectionID:  tc.collectionID,
					OrgID:         mo.None[identifiers.OrganizationID](),
					Query:         "test query",
					Field:         "test.field",
					NumBuckets:    10,
					CountByLevel:  mo.None[CountByLevel](),
					FilterByQuery: mo.None[bool](),
				},
			)

			require.NoError(t, err)
			require.NotNil(t, res.Meta)
			require.Len(t, res.Buckets, 1)
			require.Equal(t, "test", res.Buckets[0].Key)
			require.Equal(t, uint64(1), res.Buckets[0].Count)
		})
	}
}
