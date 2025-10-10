package censys

import (
	"context"
	"time"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/censys/censys-sdk-go/models/operations"
	"github.com/samber/mo"
)

//go:generate mockgen -destination=../../../../gen/client/mocks/globaldata_mock.go -package=mocks github.com/censys/cencli/internal/pkg/clients/censys GlobalDataClient
type GlobalDataClient interface {
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#gethosts
	GetHosts(
		ctx context.Context,
		orgID mo.Option[string],
		hostIDs []string,
		atTime mo.Option[time.Time],
	) (Result[[]components.Host], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#getcertificates
	GetCertificates(
		ctx context.Context,
		orgID mo.Option[string],
		certificateIDs []string,
	) (Result[[]components.Certificate], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#getwebproperties
	GetWebProperties(
		ctx context.Context,
		orgID mo.Option[string],
		webPropertyIDs []string,
		atTime mo.Option[time.Time],
	) (Result[[]components.Webproperty], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#search
	Search(
		ctx context.Context,
		orgID mo.Option[string],
		query string,
		fields []string,
		pageSize mo.Option[int64],
		pageToken mo.Option[string],
	) (Result[components.SearchQueryResponse], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#aggregate
	Aggregate(
		ctx context.Context,
		orgID mo.Option[string],
		query string,
		field string,
		numBuckets int64,
		countByLevel mo.Option[string],
		filterByQuery mo.Option[bool],
	) (Result[components.SearchAggregateResponse], ClientError)
	// https://github.com/censys/censys-sdk-go/blob/v0.22.2/models/operations/v3globaldataassethosttimeline.go
	// Note: the SDK client has the parameters backwards; fromTime is the end time and toTime is the start time. This is a mistake, but we need to keep it to not break existing API usage, so we abstract this miscommunication through this function's parameters.
	HostTimeline(
		ctx context.Context,
		orgID mo.Option[string],
		hostID string,
		fromTime time.Time,
		toTime time.Time,
	) (Result[components.HostTimeline], ClientError)
}

type globalDataSDK struct {
	*censysSDK
}

var _ GlobalDataClient = &globalDataSDK{}

func newGlobalDataSDK(censysSDK *censysSDK) *globalDataSDK {
	return &globalDataSDK{
		censysSDK: censysSDK,
	}
}

func (g *globalDataSDK) GetHosts(
	ctx context.Context,
	orgID mo.Option[string],
	hostIDs []string,
	atTime mo.Option[time.Time],
) (Result[[]components.Host], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataAssetHostListPostResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = g.censysSDK.client.GlobalData.GetHosts(ctx, operations.V3GlobaldataAssetHostListPostRequest{
			OrganizationID: orgID.ToPointer(),
			AssetHostListInputBody: components.AssetHostListInputBody{
				HostIds: hostIDs,
				AtTime:  atTime.ToPointer(),
			},
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[[]components.Host]{}
		return zero, err
	}
	hostAssets := res.GetResponseEnvelopeListHostAsset().GetResult()
	var hosts []components.Host
	for _, hostAsset := range hostAssets {
		hosts = append(hosts, hostAsset.GetResource())
	}
	return Result[[]components.Host]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     &hosts,
	}, nil
}

func (g *globalDataSDK) GetCertificates(ctx context.Context, orgID mo.Option[string], certificateIDs []string) (Result[[]components.Certificate], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataAssetCertificateListPostResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = g.censysSDK.client.GlobalData.GetCertificates(ctx, operations.V3GlobaldataAssetCertificateListPostRequest{
			OrganizationID: orgID.ToPointer(),
			AssetCertificateListInputBody: components.AssetCertificateListInputBody{
				CertificateIds: certificateIDs,
			},
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[[]components.Certificate]{}
		return zero, err
	}
	certificateAssets := res.GetResponseEnvelopeListCertificateAsset().GetResult()
	var certificates []components.Certificate
	for _, certificateAsset := range certificateAssets {
		certificates = append(certificates, certificateAsset.GetResource())
	}
	return Result[[]components.Certificate]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     &certificates,
	}, nil
}

func (g *globalDataSDK) GetWebProperties(
	ctx context.Context,
	orgID mo.Option[string],
	webPropertyIDs []string,
	atTime mo.Option[time.Time],
) (Result[[]components.Webproperty], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataAssetWebpropertyListPostResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = g.censysSDK.client.GlobalData.GetWebProperties(ctx, operations.V3GlobaldataAssetWebpropertyListPostRequest{
			OrganizationID: orgID.ToPointer(),
			AssetWebpropertyListInputBody: components.AssetWebpropertyListInputBody{
				WebpropertyIds: webPropertyIDs,
				AtTime:         atTime.ToPointer(),
			},
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[[]components.Webproperty]{}
		return zero, err
	}
	webPropertyAssets := res.GetResponseEnvelopeListWebpropertyAsset().GetResult()
	var webProperties []components.Webproperty
	for _, webPropertyAsset := range webPropertyAssets {
		webProperties = append(webProperties, webPropertyAsset.GetResource())
	}
	return Result[[]components.Webproperty]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     &webProperties,
	}, nil
}

func (g *globalDataSDK) Search(
	ctx context.Context,
	orgID mo.Option[string],
	query string,
	fields []string,
	pageSize mo.Option[int64],
	pageToken mo.Option[string],
) (Result[components.SearchQueryResponse], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataSearchQueryResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = g.censysSDK.client.GlobalData.Search(ctx, operations.V3GlobaldataSearchQueryRequest{
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

func (g *globalDataSDK) Aggregate(
	ctx context.Context,
	orgID mo.Option[string],
	query string,
	field string,
	numBuckets int64,
	countByLevel mo.Option[string],
	filterByQuery mo.Option[bool],
) (Result[components.SearchAggregateResponse], ClientError) {
	start := time.Now()
	res, err := g.censysSDK.client.GlobalData.Aggregate(ctx, operations.V3GlobaldataSearchAggregateRequest{
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

func (g *globalDataSDK) HostTimeline(
	ctx context.Context,
	orgID mo.Option[string],
	hostID string,
	fromTime time.Time,
	toTime time.Time,
) (Result[components.HostTimeline], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataAssetHostTimelineResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		var err error
		req := operations.V3GlobaldataAssetHostTimelineRequest{
			OrganizationID: orgID.ToPointer(),
			HostID:         hostID,
			// this is very backwards, but the API was accidentally written this way;
			// we need to keep it to not break existing API usage, so we abstract
			// this miscommunication through this function's parameters
			StartTime: toTime,
			EndTime:   fromTime,
		}
		res, err = g.censysSDK.client.GlobalData.GetHostTimeline(ctx, req)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.HostTimeline]{}
		return zero, err
	}
	timeline := res.GetResponseEnvelopeHostTimeline().GetResult()
	return Result[components.HostTimeline]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     timeline,
	}, nil
}
