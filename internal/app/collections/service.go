package collections

import (
	"context"
	"errors"
	"strings"

	"github.com/samber/mo"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/app/pagination"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

//go:generate mockgen -destination=../../../gen/app/collections/mocks/collectionsservice_mock.go -package=mocks -mock_names Service=MockCollectionsService . Service

// Service provides collection management capabilities.
type Service interface {
	ListCollections(ctx context.Context, params ListParams) (ListResult, cenclierrors.CencliError)
	GetCollection(ctx context.Context, params GetParams) (GetResult, cenclierrors.CencliError)
	CreateCollection(ctx context.Context, params CreateParams) (CreateResult, cenclierrors.CencliError)
	UpdateCollection(ctx context.Context, params UpdateParams) (UpdateResult, cenclierrors.CencliError)
	DeleteCollection(ctx context.Context, params DeleteParams) (DeleteResult, cenclierrors.CencliError)
}

type collectionsService struct {
	client client.Client
}

func New(client client.Client) Service {
	return &collectionsService{client: client}
}

func (s *collectionsService) ListCollections(
	ctx context.Context,
	params ListParams,
) (ListResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	// validate filter enums against the API contract before making any request
	if err := validateStatuses(params.Statuses); err != nil {
		return ListResult{}, err
	}

	// handle pagination invariants
	if err := validatePaginationParams(params.PageSize, params.MaxPages); err != nil {
		return ListResult{}, err
	}

	pageSize := optionalInt64(params.PageSize)

	listFn := func(pageToken mo.Option[string]) (client.Result[components.ListCollectionsResponseV1], client.ClientError) {
		return s.client.ListCollections(ctx, client.ListCollectionsRequest{
			OrgID:     orgIDStr,
			PageSize:  pageSize,
			PageToken: pageToken,
			Statuses:  params.Statuses,
		})
	}

	page, err := pagination.Paginate(ctx, params.MaxPages, "collections", listFn, extractCollectionsPage)
	if err != nil {
		return ListResult{}, err
	}

	return ListResult{
		Meta:         page.Meta,
		Collections:  page.Items,
		TotalSize:    page.TotalSize,
		PartialError: page.PartialError,
		HasMore:      page.HasMore,
	}, nil
}

// extractCollectionsPage adapts a collections list envelope for the paginator.
func extractCollectionsPage(list *components.ListCollectionsResponseV1) pagination.Page[Collection] {
	items := make([]Collection, 0, len(list.Collections))
	for _, c := range list.Collections {
		items = append(items, mapCollection(c))
	}
	return pagination.Page[Collection]{Items: items, NextPageToken: list.NextPageToken}
}

func (s *collectionsService) GetCollection(
	ctx context.Context,
	params GetParams,
) (GetResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	result, err := s.client.GetCollection(ctx, orgIDStr, params.CollectionID.String())
	if err != nil {
		return GetResult{}, err
	}

	var collection Collection
	if result.Data != nil {
		collection = mapCollection(*result.Data)
	}

	return GetResult{Meta: newResponseMeta(result.Metadata), Collection: collection}, nil
}

func (s *collectionsService) CreateCollection(
	ctx context.Context,
	params CreateParams,
) (CreateResult, cenclierrors.CencliError) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return CreateResult{}, NewInvalidCollectionNameError()
	}
	query := strings.TrimSpace(params.Query)
	if query == "" {
		return CreateResult{}, NewEmptyQueryError()
	}

	result, err := s.client.CreateCollection(ctx, client.CreateCollectionRequest{
		OrgID:       utilconvert.OptionalString(params.OrgID),
		Name:        name,
		Query:       query,
		Description: params.Description,
	})
	if err != nil {
		if isCollectionLimitError(err) {
			return CreateResult{}, NewCollectionLimitError(apiDetail(err))
		}
		return CreateResult{}, err
	}

	var collection Collection
	if result.Data != nil {
		collection = mapCollection(*result.Data)
	}

	return CreateResult{Meta: newResponseMeta(result.Metadata), Collection: collection}, nil
}

// isCollectionLimitError reports whether err is the create endpoint's 412
// "Collection limit exceeded" response. The endpoint uses 412 for no other reason.
func isCollectionLimitError(err cenclierrors.CencliError) bool {
	var coded interface{ StatusCode() mo.Option[int64] }
	if !errors.As(err, &coded) {
		return false
	}
	status := coded.StatusCode()
	return status.IsPresent() && status.MustGet() == 412
}

// apiDetail returns the API's one-line summary of err, when err carries one.
func apiDetail(err cenclierrors.CencliError) mo.Option[string] {
	var detailed interface{ Detail() mo.Option[string] }
	if !errors.As(err, &detailed) {
		return mo.None[string]()
	}
	return detailed.Detail()
}

// UpdateCollection changes a collection's name, query, or description. The API
// replaces the whole collection and requires the name and query every time, so
// this reads the current collection and fills in each field the caller left
// absent. A change made by someone else between the read and the
// write is overwritten.
func (s *collectionsService) UpdateCollection(
	ctx context.Context,
	params UpdateParams,
) (UpdateResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)
	collectionID := params.CollectionID.String()

	current, err := s.client.GetCollection(ctx, orgIDStr, collectionID)
	if err != nil {
		return UpdateResult{}, err
	}
	// Without the current values the replacement would carry an empty name and
	// query, which would overwrite the collection.
	if current.Data == nil {
		return UpdateResult{}, NewMissingCollectionError(collectionID)
	}

	name := params.Name.OrElse(current.Data.Name)
	query := params.Query.OrElse(current.Data.Query)
	if strings.TrimSpace(name) == "" || strings.TrimSpace(query) == "" {
		return UpdateResult{}, NewMissingCollectionError(collectionID)
	}

	result, err := s.client.UpdateCollection(ctx, client.UpdateCollectionRequest{
		OrgID:        orgIDStr,
		CollectionID: collectionID,
		Name:         name,
		Query:        query,
		Description:  mo.Some(params.Description.OrElse(current.Data.Description)),
	})
	if err != nil {
		return UpdateResult{}, err
	}

	var collection Collection
	if result.Data != nil {
		collection = mapCollection(*result.Data)
	}

	return UpdateResult{Meta: newResponseMeta(result.Metadata), Collection: collection}, nil
}

func (s *collectionsService) DeleteCollection(
	ctx context.Context,
	params DeleteParams,
) (DeleteResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	metadata, err := s.client.DeleteCollection(ctx, orgIDStr, params.CollectionID.String())
	if err != nil {
		return DeleteResult{}, err
	}

	return DeleteResult{Meta: newResponseMeta(metadata), CollectionID: params.CollectionID.String()}, nil
}

// newResponseMeta converts client metadata into the response metadata the
// command layer renders, or nil when the call carried none.
func newResponseMeta(md client.Metadata) *responsemeta.ResponseMeta {
	if md.Request == nil && md.Response == nil {
		return nil
	}
	return responsemeta.NewResponseMeta(md.Request, md.Response, md.Latency, md.Attempts)
}

// optionalInt64 narrows an unsigned page size to the signed type the client sends.
func optionalInt64(v mo.Option[uint64]) mo.Option[int64] {
	if !v.IsPresent() {
		return mo.None[int64]()
	}
	return mo.Some(int64(v.MustGet()))
}

// mapCollection converts an SDK collection into the domain DTO.
func mapCollection(c components.Collection) Collection {
	var reason *string
	if c.StatusReason != nil {
		r := string(*c.StatusReason)
		reason = &r
	}
	return Collection{
		ID:                   c.ID,
		Name:                 c.Name,
		Description:          c.Description,
		Query:                c.Query,
		Status:               string(c.Status),
		StatusReason:         reason,
		TotalAssets:          c.TotalAssets,
		AddedAssets24Hours:   c.AddedAssets24Hours,
		RemovedAssets24Hours: c.RemovedAssets24Hours,
		CreatedBy:            c.CreatedBy,
		CreateTime:           c.CreateTime,
	}
}
