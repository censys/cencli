package aggregate

import (
	"context"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/censys-sdk-go/models/components"
)

//go:generate mockgen -destination=../../../gen/app/aggregate/mocks/aggregateservice_mock.go -package=mocks -mock_names Service=MockAggregateService . Service

// Service is the search service for the aggregate command.
type Service interface {
	// Aggregate performs an aggregation given the provided parameters.
	Aggregate(ctx context.Context, params Params) (Result, cenclierrors.CencliError)
}

type aggregateService struct {
	client client.Client
}

func New(client client.Client) Service {
	return &aggregateService{client: client}
}

func (s *aggregateService) Aggregate(
	ctx context.Context,
	params Params,
) (Result, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	var res client.Result[components.SearchAggregateResponse]
	var err client.ClientError

	// Convert CountByLevel to the string option expected by the client
	countByStr := countByLevelToString(params.CountByLevel)

	if params.CollectionID.IsPresent() {
		// Use collection aggregate
		collectionIDStr := utilconvert.OptionalString(params.CollectionID)
		res, err = s.client.AggregateCollection(
			ctx,
			collectionIDStr.MustGet(),
			orgIDStr,
			params.Query,
			params.Field,
			params.NumBuckets,
			countByStr,
			params.FilterByQuery,
		)
	} else {
		// Use global aggregate
		res, err = s.client.Aggregate(
			ctx,
			orgIDStr,
			params.Query,
			params.Field,
			params.NumBuckets,
			countByStr,
			params.FilterByQuery,
		)
	}

	if err != nil {
		return Result{}, err
	}
	return Result{
		Meta:    responsemeta.NewResponseMeta(res.Metadata.Request, res.Metadata.Response, res.Metadata.Latency, res.Metadata.Attempts),
		Buckets: parseBuckets(res.Data.Buckets),
	}, nil
}
