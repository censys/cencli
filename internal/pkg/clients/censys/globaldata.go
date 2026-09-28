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
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#getwebpropertytimeline
	WebPropertyTimeline(
		ctx context.Context,
		orgID mo.Option[string],
		webPropertyID string,
		fromTime time.Time,
		toTime time.Time,
	) (Result[components.WebpropertyTimeline], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#createtrackedscan
	CreateWebPropertyRescan(
		ctx context.Context,
		orgID string,
		hostname string,
		port int,
	) (Result[components.TrackedScan], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#gettrackedscan
	GetTrackedScan(
		ctx context.Context,
		orgID string,
		scanID string,
	) (Result[components.TrackedScan], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#gethostenrichment
	EnrichHost(
		ctx context.Context,
		orgID mo.Option[string],
		hostIP string,
	) (Result[components.HostEnrichment], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#listdnsnameresolutionbounds
	ListDNSNameResolutionBounds(
		ctx context.Context,
		orgID mo.Option[string],
		name string,
		fromTime time.Time,
		toTime time.Time,
		recordTypes []string,
		pageSize mo.Option[int64],
		pageToken mo.Option[string],
	) (Result[components.DNSNameResolutionBoundResponse], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#listdnsnameresolutionranges
	ListDNSNameResolutionRanges(
		ctx context.Context,
		orgID mo.Option[string],
		name string,
		fromTime time.Time,
		toTime time.Time,
		recordTypes []string,
		pageSize mo.Option[int64],
		pageToken mo.Option[string],
	) (Result[components.DNSNameResolutionRangeResponse], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#listdnsipresolutionbounds
	ListDNSIPResolutionBounds(
		ctx context.Context,
		orgID mo.Option[string],
		ip string,
		fromTime time.Time,
		toTime time.Time,
		recordTypes []string,
		pageSize mo.Option[int64],
		pageToken mo.Option[string],
	) (Result[components.DNSIPResolutionBoundResponse], ClientError)
	// https://github.com/censys/censys-sdk-go/tree/main/docs/sdks/globaldata#listdnsipresolutionranges
	ListDNSIPResolutionRanges(
		ctx context.Context,
		orgID mo.Option[string],
		ip string,
		fromTime time.Time,
		toTime time.Time,
		recordTypes []string,
		pageSize mo.Option[int64],
		pageToken mo.Option[string],
	) (Result[components.DNSIPResolutionRangeResponse], ClientError)
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

func (g *globalDataSDK) WebPropertyTimeline(
	ctx context.Context,
	orgID mo.Option[string],
	webPropertyID string,
	fromTime time.Time,
	toTime time.Time,
) (Result[components.WebpropertyTimeline], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataAssetWebpropertyTimelineResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		var err error
		req := operations.V3GlobaldataAssetWebpropertyTimelineRequest{
			OrganizationID: orgID.ToPointer(),
			// url.PathEscape leaves ':' unescaped, which the API accepts although its
			// spec shows %3A. Pre-escaping would double-encode it to %253A.
			WebpropertyID: webPropertyID,
			// inverted in the API, as with HostTimeline
			StartTime: toTime,
			EndTime:   fromTime,
		}
		res, err = g.censysSDK.client.GlobalData.GetWebPropertyTimeline(ctx, req)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.WebpropertyTimeline]{}
		return zero, err
	}
	timeline := res.GetResponseEnvelopeWebpropertyTimeline().GetResult()
	return Result[components.WebpropertyTimeline]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     timeline,
	}, nil
}

func (g *globalDataSDK) CreateWebPropertyRescan(
	ctx context.Context,
	orgID string,
	hostname string,
	port int,
) (Result[components.TrackedScan], ClientError) {
	// Not retried: a 5xx or transport error can follow an accepted, charged
	// rescan, so a retry could charge twice.
	start := time.Now()
	res, err := g.censysSDK.client.GlobalData.CreateTrackedScan(ctx, operations.V3GlobaldataScansRescanRequest{
		OrganizationID: &orgID,
		ScansRescanInputBody: components.ScansRescanInputBody{
			Target: components.CreateScansRescanInputBodyTargetTwo(components.Two{
				WebOrigin: components.TargetWebOrigin{
					Hostname: hostname,
					Port:     port,
				},
			}),
		},
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.TrackedScan]{}
		return zero, NewClientError(err)
	}
	scan := res.GetResponseEnvelopeTrackedScan().GetResult()
	return Result[components.TrackedScan]{
		Metadata: buildResponseMetadata(res, latency, 1),
		Data:     scan,
	}, nil
}

func (g *globalDataSDK) GetTrackedScan(
	ctx context.Context,
	orgID string,
	scanID string,
) (Result[components.TrackedScan], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataScansGetResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = g.censysSDK.client.GlobalData.GetTrackedScan(ctx, operations.V3GlobaldataScansGetRequest{
			OrganizationID: &orgID,
			ScanID:         scanID,
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.TrackedScan]{}
		return zero, err
	}
	scan := res.GetResponseEnvelopeTrackedScan().GetResult()
	return Result[components.TrackedScan]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     scan,
	}, nil
}

func (g *globalDataSDK) EnrichHost(
	ctx context.Context,
	orgID mo.Option[string],
	hostIP string,
) (Result[components.HostEnrichment], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataAssetHostEnrichmentResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		var err error
		res, err = g.censysSDK.client.GlobalData.GetHostEnrichment(ctx, operations.V3GlobaldataAssetHostEnrichmentRequest{
			OrganizationID: orgID.ToPointer(),
			HostIP:         hostIP,
		})
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.HostEnrichment]{}
		return zero, err
	}
	// GetResult/GetResource are nil-safe on the generated types, returning a zero
	// HostEnrichment if the envelope or asset is absent.
	enrichment := res.GetResponseEnvelopeHostEnrichmentAsset().GetResult().GetResource()
	return Result[components.HostEnrichment]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     &enrichment,
	}, nil
}

func (g *globalDataSDK) ListDNSNameResolutionBounds(
	ctx context.Context,
	orgID mo.Option[string],
	name string,
	fromTime time.Time,
	toTime time.Time,
	recordTypes []string,
	pageSize mo.Option[int64],
	pageToken mo.Option[string],
) (Result[components.DNSNameResolutionBoundResponse], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataDNSNameResolutionBoundResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		startStr := fromTime.UTC().Format(time.RFC3339)
		endStr := toTime.UTC().Format(time.RFC3339)
		req := operations.V3GlobaldataDNSNameResolutionBoundRequest{
			OrganizationID: orgID.ToPointer(),
			Name:           name,
			StartTime:      &startStr,
			EndTime:        &endStr,
			PageToken:      pageToken.ToPointer(),
			RecordTypes:    dnsRecordTypes[operations.V3GlobaldataDNSNameResolutionBoundQueryParamRecordTypes](recordTypes),
		}
		if pageSize.IsPresent() {
			ps := int(pageSize.MustGet())
			req.PageSize = &ps
		}
		var err error
		res, err = g.censysSDK.client.GlobalData.ListDNSNameResolutionBounds(ctx, req)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.DNSNameResolutionBoundResponse]{}
		return zero, err
	}
	bounds := res.GetResponseEnvelopeDNSNameResolutionBoundResponse().GetResult()
	return Result[components.DNSNameResolutionBoundResponse]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     bounds,
	}, nil
}

func (g *globalDataSDK) ListDNSNameResolutionRanges(
	ctx context.Context,
	orgID mo.Option[string],
	name string,
	fromTime time.Time,
	toTime time.Time,
	recordTypes []string,
	pageSize mo.Option[int64],
	pageToken mo.Option[string],
) (Result[components.DNSNameResolutionRangeResponse], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataDNSNameResolutionRangesResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		startStr := fromTime.UTC().Format(time.RFC3339)
		endStr := toTime.UTC().Format(time.RFC3339)
		req := operations.V3GlobaldataDNSNameResolutionRangesRequest{
			OrganizationID: orgID.ToPointer(),
			Name:           name,
			StartTime:      &startStr,
			EndTime:        &endStr,
			PageToken:      pageToken.ToPointer(),
			RecordTypes:    dnsRecordTypes[operations.V3GlobaldataDNSNameResolutionRangesQueryParamRecordTypes](recordTypes),
		}
		if pageSize.IsPresent() {
			ps := int(pageSize.MustGet())
			req.PageSize = &ps
		}
		var err error
		res, err = g.censysSDK.client.GlobalData.ListDNSNameResolutionRanges(ctx, req)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.DNSNameResolutionRangeResponse]{}
		return zero, err
	}
	ranges := res.GetResponseEnvelopeDNSNameResolutionRangeResponse().GetResult()
	return Result[components.DNSNameResolutionRangeResponse]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     ranges,
	}, nil
}

func (g *globalDataSDK) ListDNSIPResolutionBounds(
	ctx context.Context,
	orgID mo.Option[string],
	ip string,
	fromTime time.Time,
	toTime time.Time,
	recordTypes []string,
	pageSize mo.Option[int64],
	pageToken mo.Option[string],
) (Result[components.DNSIPResolutionBoundResponse], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataDNSIPResolutionBoundResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		startStr := fromTime.UTC().Format(time.RFC3339)
		endStr := toTime.UTC().Format(time.RFC3339)
		req := operations.V3GlobaldataDNSIPResolutionBoundRequest{
			OrganizationID: orgID.ToPointer(),
			IP:             ip,
			StartTime:      &startStr,
			EndTime:        &endStr,
			PageToken:      pageToken.ToPointer(),
			RecordTypes:    dnsRecordTypes[operations.RecordTypes](recordTypes),
		}
		if pageSize.IsPresent() {
			ps := int(pageSize.MustGet())
			req.PageSize = &ps
		}
		var err error
		res, err = g.censysSDK.client.GlobalData.ListDNSIPResolutionBounds(ctx, req)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.DNSIPResolutionBoundResponse]{}
		return zero, err
	}
	bounds := res.GetResponseEnvelopeDNSIPResolutionBoundResponse().GetResult()
	return Result[components.DNSIPResolutionBoundResponse]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     bounds,
	}, nil
}

func (g *globalDataSDK) ListDNSIPResolutionRanges(
	ctx context.Context,
	orgID mo.Option[string],
	ip string,
	fromTime time.Time,
	toTime time.Time,
	recordTypes []string,
	pageSize mo.Option[int64],
	pageToken mo.Option[string],
) (Result[components.DNSIPResolutionRangeResponse], ClientError) {
	start := time.Now()
	var res *operations.V3GlobaldataDNSIPResolutionRangesResponse
	err, attempts := g.executeWithRetry(ctx, func() ClientError {
		startStr := fromTime.UTC().Format(time.RFC3339)
		endStr := toTime.UTC().Format(time.RFC3339)
		req := operations.V3GlobaldataDNSIPResolutionRangesRequest{
			OrganizationID: orgID.ToPointer(),
			IP:             ip,
			StartTime:      &startStr,
			EndTime:        &endStr,
			PageToken:      pageToken.ToPointer(),
			RecordTypes:    dnsRecordTypes[operations.QueryParamRecordTypes](recordTypes),
		}
		if pageSize.IsPresent() {
			ps := int(pageSize.MustGet())
			req.PageSize = &ps
		}
		var err error
		res, err = g.censysSDK.client.GlobalData.ListDNSIPResolutionRanges(ctx, req)
		if err != nil {
			return NewClientError(err)
		}
		return nil
	})
	latency := time.Since(start)
	if err != nil {
		zero := Result[components.DNSIPResolutionRangeResponse]{}
		return zero, err
	}
	ranges := res.GetResponseEnvelopeDNSIPResolutionRangeResponse().GetResult()
	return Result[components.DNSIPResolutionRangeResponse]{
		Metadata: buildResponseMetadata(res, latency, attempts),
		Data:     ranges,
	}, nil
}

// dnsRecordTypes converts record-type strings to an SDK record-type enum. The
// SDK generates a separate enum type for each DNS operation, so callers pass
// plain strings and the conversion stays here.
func dnsRecordTypes[T ~string](values []string) []T {
	if len(values) == 0 {
		return nil
	}
	out := make([]T, 0, len(values))
	for _, v := range values {
		out = append(out, T(v))
	}
	return out
}
