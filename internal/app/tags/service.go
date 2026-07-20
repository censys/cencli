package tags

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/samber/mo"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

//go:generate mockgen -destination=../../../gen/app/tags/mocks/tagsservice_mock.go -package=mocks -mock_names Service=MockTagsService . Service

// Service provides tag management capabilities.
type Service interface {
	ListTags(ctx context.Context, params ListParams) (ListResult, cenclierrors.CencliError)
	GetTag(ctx context.Context, params GetParams) (GetResult, cenclierrors.CencliError)
	CreateTag(ctx context.Context, params CreateParams) (CreateResult, cenclierrors.CencliError)
	UpdateTag(ctx context.Context, params UpdateParams) (UpdateResult, cenclierrors.CencliError)
	DeleteTag(ctx context.Context, params DeleteParams) (DeleteResult, cenclierrors.CencliError)
}

type tagsService struct {
	client client.Client
}

func New(client client.Client) Service {
	return &tagsService{client: client}
}

func (s *tagsService) ListTags(
	ctx context.Context,
	params ListParams,
) (ListResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	// validate filter enums against the API contract before making any request
	if err := validateOrderBy(params.OrderBy); err != nil {
		return ListResult{}, err
	}
	if err := validatePrivacy(params.Privacy); err != nil {
		return ListResult{}, err
	}

	// handle pagination invariants
	if params.PageSize.IsPresent() && params.PageSize.MustGet() == 0 {
		return ListResult{}, NewInvalidPaginationParamsError("page size must be greater than 0")
	}
	if params.MaxPages.IsPresent() && params.MaxPages.MustGet() == 0 {
		return ListResult{}, NewInvalidPaginationParamsError("max pages must be greater than 0")
	}

	pageSize := mo.None[int64]()
	if params.PageSize.IsPresent() {
		pageSize = mo.Some(int64(params.PageSize.MustGet()))
	}

	listFn := func(pageToken mo.Option[string]) (client.Result[components.TagsList], client.ClientError) {
		return s.client.ListTags(ctx, client.ListTagsRequest{
			OrgID:     orgIDStr,
			PageSize:  pageSize,
			PageToken: pageToken,
			OrderBy:   params.OrderBy,
			Name:      params.Name,
			CreatedBy: params.CreatedBy,
			Privacy:   params.Privacy,
		})
	}

	return s.listWithPagination(ctx, listFn, params.MaxPages)
}

// GetTag retrieves a single tag by name or UUID. The endpoint accepts either
// interchangeably, so the raw identifier is passed straight through (no resolve
// roundtrip).
func (s *tagsService) GetTag(
	ctx context.Context,
	params GetParams,
) (GetResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	result, err := s.client.GetTag(ctx, orgIDStr, params.TagID.String())
	if err != nil {
		return GetResult{}, err
	}

	var meta *responsemeta.ResponseMeta
	if result.Metadata.Request != nil || result.Metadata.Response != nil {
		meta = responsemeta.NewResponseMeta(
			result.Metadata.Request,
			result.Metadata.Response,
			result.Metadata.Latency,
			result.Metadata.Attempts,
		)
	}

	var tag Tag
	if result.Data != nil {
		tag = mapTag(*result.Data)
	}

	return GetResult{Meta: meta, Tag: tag}, nil
}

// CreateTag creates a new tag. The name (non-empty) and privacy are validated
// against the API contract before the request is made.
func (s *tagsService) CreateTag(
	ctx context.Context,
	params CreateParams,
) (CreateResult, cenclierrors.CencliError) {
	if strings.TrimSpace(params.Name) == "" {
		return CreateResult{}, NewInvalidTagNameError()
	}
	if err := validatePrivacy(mo.Some(params.Privacy)); err != nil {
		return CreateResult{}, err
	}

	orgIDStr := utilconvert.OptionalString(params.OrgID)

	result, err := s.client.CreateTag(ctx, client.CreateTagRequest{
		OrgID:       orgIDStr,
		Name:        params.Name,
		Description: params.Description,
		Privacy:     params.Privacy,
	})
	if err != nil {
		return CreateResult{}, err
	}

	var meta *responsemeta.ResponseMeta
	if result.Metadata.Request != nil || result.Metadata.Response != nil {
		meta = responsemeta.NewResponseMeta(
			result.Metadata.Request,
			result.Metadata.Response,
			result.Metadata.Latency,
			result.Metadata.Attempts,
		)
	}

	var tag Tag
	if result.Data != nil {
		tag = mapTag(*result.Data)
	}

	return CreateResult{Meta: meta, Tag: tag}, nil
}

// UpdateTag mutates an existing tag by name or UUID. Privacy, when provided, is
// validated; the raw identifier is passed straight through (the endpoint
// accepts name or UUID, like GetTag).
func (s *tagsService) UpdateTag(
	ctx context.Context,
	params UpdateParams,
) (UpdateResult, cenclierrors.CencliError) {
	if err := validatePrivacy(params.Privacy); err != nil {
		return UpdateResult{}, err
	}

	orgIDStr := utilconvert.OptionalString(params.OrgID)

	result, err := s.client.UpdateTag(ctx, client.UpdateTagRequest{
		OrgID:       orgIDStr,
		TagID:       params.TagID.String(),
		Name:        params.Name,
		Description: params.Description,
		Privacy:     params.Privacy,
	})
	if err != nil {
		return UpdateResult{}, err
	}

	var meta *responsemeta.ResponseMeta
	if result.Metadata.Request != nil || result.Metadata.Response != nil {
		meta = responsemeta.NewResponseMeta(
			result.Metadata.Request,
			result.Metadata.Response,
			result.Metadata.Latency,
			result.Metadata.Attempts,
		)
	}

	var tag Tag
	if result.Data != nil {
		tag = mapTag(*result.Data)
	}

	return UpdateResult{Meta: meta, Tag: tag}, nil
}

// DeleteTag removes a tag by name or UUID. The raw identifier is passed straight
// through (the endpoint accepts name or UUID, like GetTag/UpdateTag).
func (s *tagsService) DeleteTag(
	ctx context.Context,
	params DeleteParams,
) (DeleteResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	metadata, err := s.client.DeleteTag(ctx, orgIDStr, params.TagID.String())
	if err != nil {
		return DeleteResult{}, err
	}

	var meta *responsemeta.ResponseMeta
	if metadata.Request != nil || metadata.Response != nil {
		meta = responsemeta.NewResponseMeta(
			metadata.Request,
			metadata.Response,
			metadata.Latency,
			metadata.Attempts,
		)
	}

	return DeleteResult{Meta: meta, TagID: params.TagID.String()}, nil
}

func (s *tagsService) listWithPagination(
	ctx context.Context,
	listFn func(mo.Option[string]) (client.Result[components.TagsList], client.ClientError),
	maxPages mo.Option[uint64],
) (ListResult, cenclierrors.CencliError) {
	var allTags []Tag
	var totalSize int64
	var lastMeta *responsemeta.ResponseMeta
	var pagesProcessed uint64
	var firstError cenclierrors.CencliError
	pageToken := mo.None[string]()

	start := time.Now()

	for {
		if maxPages.IsPresent() && pagesProcessed >= maxPages.MustGet() {
			break
		}

		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			contextErr := cenclierrors.ParseContextError(err)
			if pagesProcessed > 0 {
				if lastMeta != nil {
					lastMeta.Latency = time.Since(start)
					lastMeta.PageCount = pagesProcessed
				}
				return ListResult{
					Meta:         lastMeta,
					Tags:         allTags,
					TotalSize:    totalSize,
					PartialError: cenclierrors.ToPartialError(contextErr),
				}, nil
			}
			return ListResult{}, contextErr
		}

		s.reportListProgress(ctx, pagesProcessed, len(allTags), maxPages)

		result, err := listFn(pageToken)
		if err != nil {
			// First page: return the error immediately.
			if pagesProcessed == 0 {
				return ListResult{}, err
			}
			// Otherwise record it, report it, and return partial results.
			firstError = err
			progress.ReportError(ctx, progress.StageFetch, err)
			break
		}

		if result.Metadata.Request != nil || result.Metadata.Response != nil {
			lastMeta = responsemeta.NewResponseMeta(result.Metadata.Request, result.Metadata.Response, 0, result.Metadata.Attempts)
		}

		if result.Data == nil {
			pagesProcessed++
			break
		}

		for _, t := range result.Data.Tags {
			allTags = append(allTags, mapTag(t))
		}
		totalSize = result.Data.TotalSize
		pagesProcessed++

		nextPageToken := ""
		if npt := result.Data.GetNextPageToken(); npt != nil {
			nextPageToken = *npt
		}
		if nextPageToken == "" || len(result.Data.Tags) == 0 {
			break
		}

		if maxPages.IsPresent() && pagesProcessed >= maxPages.MustGet() {
			break
		}

		pageToken = mo.Some(nextPageToken)
	}

	if lastMeta != nil {
		lastMeta.Latency = time.Since(start)
		lastMeta.PageCount = pagesProcessed
	}

	return ListResult{
		Meta:         lastMeta,
		Tags:         allTags,
		TotalSize:    totalSize,
		PartialError: cenclierrors.ToPartialError(firstError),
	}, nil
}

func (s *tagsService) reportListProgress(ctx context.Context, page uint64, collected int, maxPages mo.Option[uint64]) {
	if page == 0 {
		return
	}

	var msg string
	if maxPages.IsPresent() {
		msg = fmt.Sprintf("Fetching tags (page %d/%d, %d collected)...", page+1, maxPages.MustGet(), collected)
	} else {
		msg = fmt.Sprintf("Fetching tags (page %d, %d collected)...", page+1, collected)
	}

	progress.ReportMessage(ctx, progress.StageFetch, msg)
}

// mapTag converts an SDK tag into the domain DTO.
func mapTag(t components.Tag) Tag {
	return Tag{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Privacy:     string(t.Privacy),
		CreatedBy:   t.CreatedBy,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
