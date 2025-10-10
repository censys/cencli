package censys

import (
	"context"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/operations"
	"github.com/samber/mo"
)

//go:generate mockgen -destination=../../../../gen/client/mocks/collections_client_mock.go -package=mocks github.com/censys/cencli/internal/pkg/clients/censys CollectionsClient
type CollectionsClient interface {
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/collections#search
	SearchCollection(
		ctx context.Context,
		collectionID string,
		orgID mo.Option[string],
		query string,
		fields []string,
		pageSize mo.Option[int64],
		pageToken mo.Option[string],
	) (Result[components.SearchQueryResponse], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/collections#aggregate
	AggregateCollection(
		ctx context.Context,
		collectionID string,
		orgID mo.Option[string],
		query string,
		field string,
		numBuckets int64,
		countByLevel mo.Option[string],
		filterByQuery mo.Option[bool],
	) (Result[components.SearchAggregateResponse], ClientError)
}

type collectionsSDK struct {
	*censysSDK
}

var _ CollectionsClient = &collectionsSDK{}

func newCollectionsSDK(censysSDK *censysSDK) *collectionsSDK {
	return &collectionsSDK{
		censysSDK: censysSDK,
	}
}

func (c *collectionsSDK) SearchCollection(
	ctx context.Context,
	collectionID string,
	orgID mo.Option[string],
	query string,
	fields []string,
	pageSize mo.Option[int64],
	pageToken mo.Option[string],
) (Result[components.SearchQueryResponse], ClientError) {
	start := time.Now()
	var res *operations.V3CollectionsSearchQueryResponse
	err, attempts := c.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = c.censysSDK.client.Collections.Search(ctx, operations.V3CollectionsSearchQueryRequest{
			CollectionUID:  collectionID,
			OrganizationID: orgID.ToPointer(),
			SearchQueryInputBody: components.SearchQueryInputBody{
				Query:     query,
				Fields:    fields,
				PageSize:  pageSize.ToPointer(),
				PageToken: pageToken.ToPointer(),
			},
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.SearchQueryResponse]{}
		return zero, err
	}
	searchQueryResponse := res.GetResponseEnvelopeSearchQueryResponse().GetResult()
	return Result[components.SearchQueryResponse]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     searchQueryResponse,
	}, nil
}

func (c *collectionsSDK) AggregateCollection(
	ctx context.Context,
	collectionID string,
	orgID mo.Option[string],
	query string,
	field string,
	numBuckets int64,
	countByLevel mo.Option[string],
	filterByQuery mo.Option[bool],
) (Result[components.SearchAggregateResponse], ClientError) {
	start := time.Now()
	res, err := c.censysSDK.client.Collections.Aggregate(ctx, operations.V3CollectionsSearchAggregateRequest{
		CollectionUID:  collectionID,
		OrganizationID: orgID.ToPointer(),
		SearchAggregateInputBody: components.SearchAggregateInputBody{
			Query:           query,
			Field:           field,
			NumberOfBuckets: numBuckets,
			CountByLevel:    countByLevel.ToPointer(),
			FilterByQuery:   filterByQuery.ToPointer(),
		},
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.SearchAggregateResponse]{}
		return zero, NewClientError(err)
	}
	searchAggregateResponse := res.GetResponseEnvelopeSearchAggregateResponse().GetResult()
	return Result[components.SearchAggregateResponse]{
		Metadata: buildResponseMetadata(res, latency, 1),
		Data:     searchAggregateResponse,
	}, nil
}
