package censys

import (
	"context"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/operations"
	"github.com/samber/mo"
)

// ListCollectionsRequest bundles the query parameters for ListCollections.
// An empty Statuses leaves the status filter off the request.
type ListCollectionsRequest struct {
	OrgID     mo.Option[string]
	PageSize  mo.Option[int64]
	PageToken mo.Option[string]
	Statuses  []string
}

// CreateCollectionRequest bundles the fields for CreateCollection.
type CreateCollectionRequest struct {
	OrgID       mo.Option[string]
	Name        string
	Query       string
	Description mo.Option[string]
}

// UpdateCollectionRequest bundles the fields for UpdateCollection. The API
// replaces the whole collection, so Name and Query are always sent.
type UpdateCollectionRequest struct {
	OrgID        mo.Option[string]
	CollectionID string
	Name         string
	Query        string
	Description  mo.Option[string]
}

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
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/collections#list
	ListCollections(ctx context.Context, req ListCollectionsRequest) (Result[components.ListCollectionsResponseV1], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/collections#get
	GetCollection(
		ctx context.Context,
		orgID mo.Option[string],
		collectionID string,
	) (Result[components.Collection], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/collections#create
	CreateCollection(ctx context.Context, req CreateCollectionRequest) (Result[components.Collection], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/collections#update
	UpdateCollection(ctx context.Context, req UpdateCollectionRequest) (Result[components.Collection], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/collections#delete
	//
	// DeleteCollection returns only response metadata; the endpoint has no body.
	DeleteCollection(ctx context.Context, orgID mo.Option[string], collectionID string) (Metadata, ClientError)
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

func (c *collectionsSDK) ListCollections(
	ctx context.Context,
	req ListCollectionsRequest,
) (Result[components.ListCollectionsResponseV1], ClientError) {
	start := time.Now()
	var res *operations.V3CollectionsCrudListResponse
	err, attempts := c.executeWithRetry(ctx, func() ClientError {
		var err error
		sdkReq := operations.V3CollectionsCrudListRequest{
			OrganizationID: req.OrgID.ToPointer(),
			PageToken:      req.PageToken.ToPointer(),
			PageSize:       req.PageSize.ToPointer(),
		}
		for _, s := range req.Statuses {
			sdkReq.CollectionStatuses = append(sdkReq.CollectionStatuses, operations.CollectionStatuses(s))
		}
		res, err = c.censysSDK.client.Collections.List(ctx, sdkReq)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.ListCollectionsResponseV1]{}
		return zero, err
	}
	list := res.GetResponseEnvelopeListCollectionsResponseV1().GetResult()
	return Result[components.ListCollectionsResponseV1]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     list,
	}, nil
}

func (c *collectionsSDK) GetCollection(
	ctx context.Context,
	orgID mo.Option[string],
	collectionID string,
) (Result[components.Collection], ClientError) {
	start := time.Now()
	var res *operations.V3CollectionsCrudGetResponse
	err, attempts := c.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = c.censysSDK.client.Collections.Get(ctx, operations.V3CollectionsCrudGetRequest{
			OrganizationID: orgID.ToPointer(),
			CollectionUID:  collectionID,
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.Collection]{}
		return zero, err
	}
	collection := res.GetResponseEnvelopeCollection().GetResult()
	return Result[components.Collection]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     collection,
	}, nil
}

func (c *collectionsSDK) CreateCollection(
	ctx context.Context,
	req CreateCollectionRequest,
) (Result[components.Collection], ClientError) {
	start := time.Now()
	var res *operations.V3CollectionsCrudCreateResponse
	err, attempts := c.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = c.censysSDK.client.Collections.Create(ctx, operations.V3CollectionsCrudCreateRequest{
			OrganizationID: req.OrgID.ToPointer(),
			CrudCreateInputBody: &components.CrudCreateInputBody{
				Name:        req.Name,
				Query:       req.Query,
				Description: req.Description.ToPointer(),
			},
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.Collection]{}
		return zero, err
	}
	collection := res.GetResponseEnvelopeCollection().GetResult()
	return Result[components.Collection]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     collection,
	}, nil
}

func (c *collectionsSDK) UpdateCollection(
	ctx context.Context,
	req UpdateCollectionRequest,
) (Result[components.Collection], ClientError) {
	start := time.Now()
	var res *operations.V3CollectionsCrudUpdateResponse
	err, attempts := c.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = c.censysSDK.client.Collections.Update(ctx, operations.V3CollectionsCrudUpdateRequest{
			OrganizationID: req.OrgID.ToPointer(),
			CollectionUID:  req.CollectionID,
			CrudUpdateInputBody: &components.CrudUpdateInputBody{
				Name:        req.Name,
				Query:       req.Query,
				Description: req.Description.ToPointer(),
			},
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.Collection]{}
		return zero, err
	}
	collection := res.GetResponseEnvelopeCollection().GetResult()
	return Result[components.Collection]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     collection,
	}, nil
}

func (c *collectionsSDK) DeleteCollection(
	ctx context.Context,
	orgID mo.Option[string],
	collectionID string,
) (Metadata, ClientError) {
	start := time.Now()
	var res *operations.V3CollectionsCrudDeleteResponse
	err, attempts := c.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = c.censysSDK.client.Collections.Delete(ctx, operations.V3CollectionsCrudDeleteRequest{
			OrganizationID: orgID.ToPointer(),
			CollectionUID:  collectionID,
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		return Metadata{}, err
	}
	// The delete endpoint returns no body, only response metadata.
	return buildResponseMetadata(res, latency, attempts), nil
}
