package dns

import (
	"context"
	"errors"

	"github.com/samber/mo"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/app/pagination"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/assets"
)

//go:generate mockgen -destination=../../../gen/app/dns/mocks/dnsservice_mock.go -package=mocks -mock_names Service=MockDNSService . Service

// Service provides Active DNS resolution lookups.
type Service interface {
	// NameResolutions returns one aggregated row per record of a name.
	NameResolutions(ctx context.Context, name assets.DomainName, params Params) (NameResolutionsResult, cenclierrors.CencliError)
	// NameResolutionRanges returns one row per observed time range of a name's records.
	NameResolutionRanges(ctx context.Context, name assets.DomainName, params Params) (NameResolutionRangesResult, cenclierrors.CencliError)
	// IPResolutions returns one aggregated row per domain that resolved to an IP.
	IPResolutions(ctx context.Context, ip assets.HostID, params Params) (IPResolutionsResult, cenclierrors.CencliError)
	// IPResolutionRanges returns one row per observed time range of the domains that resolved to an IP.
	IPResolutionRanges(ctx context.Context, ip assets.HostID, params Params) (IPResolutionRangesResult, cenclierrors.CencliError)
}

type dnsService struct {
	client client.Client
}

func New(client client.Client) Service {
	return &dnsService{client: client}
}

// progressLabel names the items in pagination progress messages.
const progressLabel = "DNS records"

func (s *dnsService) NameResolutions(ctx context.Context, name assets.DomainName, params Params) (NameResolutionsResult, cenclierrors.CencliError) {
	orgID := utilconvert.OptionalString(params.OrgID)
	page, err := lookup(ctx, params, nameRecordTypes,
		func(recordTypes []string, pageToken mo.Option[string]) (client.Result[components.DNSNameResolutionBoundResponse], client.ClientError) {
			return s.client.ListDNSNameResolutionBounds(ctx, orgID, name.String(), params.FromTime, params.ToTime, recordTypes, optionalInt64(params.PageSize), pageToken)
		},
		func(r *components.DNSNameResolutionBoundResponse) pagination.PageData[*components.DNSResolutionRecord] {
			return pagination.PageData[*components.DNSResolutionRecord]{Items: pointers(r.Records), TotalSize: r.TotalRecords, NextPageToken: r.NextPageToken}
		},
	)
	if err != nil {
		return NameResolutionsResult{}, err
	}
	return NameResolutionsResult{Meta: page.Meta, Records: nonNil(page.Items), TotalRecords: page.TotalSize, PartialError: page.PartialError}, nil
}

func (s *dnsService) NameResolutionRanges(ctx context.Context, name assets.DomainName, params Params) (NameResolutionRangesResult, cenclierrors.CencliError) {
	orgID := utilconvert.OptionalString(params.OrgID)
	page, err := lookup(ctx, params, nameRecordTypes,
		func(recordTypes []string, pageToken mo.Option[string]) (client.Result[components.DNSNameResolutionRangeResponse], client.ClientError) {
			return s.client.ListDNSNameResolutionRanges(ctx, orgID, name.String(), params.FromTime, params.ToTime, recordTypes, optionalInt64(params.PageSize), pageToken)
		},
		func(r *components.DNSNameResolutionRangeResponse) pagination.PageData[*components.DNSResolutionRangeRecord] {
			return pagination.PageData[*components.DNSResolutionRangeRecord]{Items: pointers(r.Records), TotalSize: r.TotalRecords, NextPageToken: r.NextPageToken}
		},
	)
	if err != nil {
		return NameResolutionRangesResult{}, err
	}
	return NameResolutionRangesResult{Meta: page.Meta, Records: nonNil(page.Items), TotalRecords: page.TotalSize, PartialError: page.PartialError}, nil
}

func (s *dnsService) IPResolutions(ctx context.Context, ip assets.HostID, params Params) (IPResolutionsResult, cenclierrors.CencliError) {
	orgID := utilconvert.OptionalString(params.OrgID)
	page, err := lookup(ctx, params, ipRecordTypes,
		func(recordTypes []string, pageToken mo.Option[string]) (client.Result[components.DNSIPResolutionBoundResponse], client.ClientError) {
			return s.client.ListDNSIPResolutionBounds(ctx, orgID, ip.String(), params.FromTime, params.ToTime, recordTypes, optionalInt64(params.PageSize), pageToken)
		},
		func(r *components.DNSIPResolutionBoundResponse) pagination.PageData[*components.DNSIPResolutionRecord] {
			return pagination.PageData[*components.DNSIPResolutionRecord]{Items: pointers(r.Records), TotalSize: r.TotalRecords, NextPageToken: r.NextPageToken}
		},
	)
	if err != nil {
		return IPResolutionsResult{}, err
	}
	return IPResolutionsResult{Meta: page.Meta, Records: nonNil(page.Items), TotalRecords: page.TotalSize, PartialError: page.PartialError}, nil
}

func (s *dnsService) IPResolutionRanges(ctx context.Context, ip assets.HostID, params Params) (IPResolutionRangesResult, cenclierrors.CencliError) {
	orgID := utilconvert.OptionalString(params.OrgID)
	page, err := lookup(ctx, params, ipRecordTypes,
		func(recordTypes []string, pageToken mo.Option[string]) (client.Result[components.DNSIPResolutionRangeResponse], client.ClientError) {
			return s.client.ListDNSIPResolutionRanges(ctx, orgID, ip.String(), params.FromTime, params.ToTime, recordTypes, optionalInt64(params.PageSize), pageToken)
		},
		func(r *components.DNSIPResolutionRangeResponse) pagination.PageData[*components.DNSIPResolutionRangeRecord] {
			return pagination.PageData[*components.DNSIPResolutionRangeRecord]{Items: pointers(r.Records), TotalSize: r.TotalRecords, NextPageToken: r.NextPageToken}
		},
	)
	if err != nil {
		return IPResolutionRangesResult{}, err
	}
	return IPResolutionRangesResult{Meta: page.Meta, Records: nonNil(page.Items), TotalRecords: page.TotalSize, PartialError: page.PartialError}, nil
}

// lookup validates params, then pages a DNS resolution endpoint and maps a 403
// to the plan-requirement error. It factors out the prepare -> paginate ->
// mapAccessError sequence shared by all four service methods: fetch takes the
// normalized record types and a page token, and extract pulls the paginator's
// PageData out of one page's response.
func lookup[Page, Item any](
	ctx context.Context,
	params Params,
	supported []string,
	fetch func(recordTypes []string, pageToken mo.Option[string]) (client.Result[Page], client.ClientError),
	extract func(*Page) pagination.PageData[Item],
) (pagination.Paginated[Item], cenclierrors.CencliError) {
	recordTypes, err := prepare(params, supported)
	if err != nil {
		return pagination.Paginated[Item]{}, err
	}
	page, err := pagination.Paginate(ctx, params.MaxPages, progressLabel,
		func(pageToken mo.Option[string]) (client.Result[Page], client.ClientError) {
			return fetch(recordTypes, pageToken)
		},
		extract,
	)
	if err != nil {
		return pagination.Paginated[Item]{}, mapAccessError(err)
	}
	return page, nil
}

// prepare validates the pagination values and returns the normalized record types.
func prepare(params Params, supported []string) ([]string, cenclierrors.CencliError) {
	if err := validatePaginationParams(params.PageSize, params.MaxPages); err != nil {
		return nil, err
	}
	return normalizeRecordTypes(params.RecordTypes, supported)
}

// mapAccessError replaces a 403 with an error that names the plan requirement.
// The status check follows isMissingTagError in the tags app package.
func mapAccessError(err cenclierrors.CencliError) cenclierrors.CencliError {
	var coded interface{ StatusCode() mo.Option[int64] }
	if errors.As(err, &coded) {
		if status := coded.StatusCode(); status.IsPresent() && status.MustGet() == 403 {
			return NewAccessDeniedError()
		}
	}
	return err
}

// optionalInt64 converts an optional page size for the client.
func optionalInt64(v mo.Option[uint64]) mo.Option[int64] {
	if v.IsPresent() {
		return mo.Some(int64(v.MustGet()))
	}
	return mo.None[int64]()
}

// pointers returns a pointer to each element, so records can be streamed and
// printed without copying.
func pointers[T any](items []T) []*T {
	out := make([]*T, len(items))
	for i := range items {
		out[i] = &items[i]
	}
	return out
}

// nonNil keeps an empty result an empty list, so JSON output prints [] not null.
func nonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
