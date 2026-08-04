package tags

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/mo"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
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
	Assign(ctx context.Context, params AssignParams) (AssignResult, cenclierrors.CencliError)
	BulkAssign(ctx context.Context, params BulkAssignParams) (BulkAssignResult, cenclierrors.CencliError)
	Unassign(ctx context.Context, params UnassignParams) (UnassignResult, cenclierrors.CencliError)
	BulkUnassign(ctx context.Context, params BulkUnassignParams) (BulkUnassignResult, cenclierrors.CencliError)
	ListAssignments(ctx context.Context, params AssignmentsParams) (AssignmentsResult, cenclierrors.CencliError)
	ListOperations(ctx context.Context, params OperationsParams) (OperationsResult, cenclierrors.CencliError)
	GetOperation(ctx context.Context, params GetOperationParams) (GetOperationResult, cenclierrors.CencliError)
	CancelOperation(ctx context.Context, params CancelOperationParams) (CancelOperationResult, cenclierrors.CencliError)
	WaitForOperation(ctx context.Context, params WaitParams) (GetOperationResult, cenclierrors.CencliError)
}

type tagsService struct {
	client client.Client
	// sleep paces the WaitForOperation poll loop. It is a field so tests can
	// substitute a fake clock; the repo has no shared clock abstraction.
	sleep func(ctx context.Context, d time.Duration) error
}

func New(client client.Client) Service {
	return &tagsService{client: client, sleep: sleepWithContext}
}

// sleepWithContext waits for d, returning early if the context is cancelled.
// Mirrors the timer/select pattern the client's retry loop uses.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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
	if err := validatePaginationParams(params.PageSize, params.MaxPages); err != nil {
		return ListResult{}, err
	}

	pageSize := optionalInt64(params.PageSize)

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

	page, err := paginate(ctx, params.MaxPages, "tags", listFn, extractTagsPage)
	if err != nil {
		return ListResult{}, err
	}

	return ListResult{
		Meta:         page.Meta,
		Tags:         page.Items,
		TotalSize:    page.TotalSize,
		PartialError: page.PartialError,
	}, nil
}

// ListAssignments lists the assets a tag is assigned to. The endpoint is keyed by
// UUID, so a name is resolved first.
func (s *tagsService) ListAssignments(
	ctx context.Context,
	params AssignmentsParams,
) (AssignmentsResult, cenclierrors.CencliError) {
	// validate filter enums against the API contract before making any request
	if err := validateAssignmentsOrderBy(params.OrderBy); err != nil {
		return AssignmentsResult{}, err
	}
	if err := validateAssetType(params.AssetType); err != nil {
		return AssignmentsResult{}, err
	}
	if err := ValidateTimeWindow(params.CreatedBefore, params.CreatedAfter); err != nil {
		return AssignmentsResult{}, err
	}

	// handle pagination invariants
	if err := validatePaginationParams(params.PageSize, params.MaxPages); err != nil {
		return AssignmentsResult{}, err
	}

	orgIDStr := utilconvert.OptionalString(params.OrgID)

	pageSize := optionalInt64(params.PageSize)

	// paginate only returns a hard error when the *first* page failed, so a retry
	// here cannot re-emit anything already streamed.
	page, err := callWithTag(ctx, s, orgIDStr, params.TagID,
		func(tagID string) (paginated[Assignment], cenclierrors.CencliError) {
			listFn := func(pageToken mo.Option[string]) (client.Result[components.TagAssignmentsList], client.ClientError) {
				return s.client.ListTagAssignments(ctx, client.ListTagAssignmentsRequest{
					OrgID:         orgIDStr,
					TagID:         tagID,
					AssetID:       params.AssetID,
					AssetType:     params.AssetType,
					CreatedBy:     params.CreatedBy,
					CreatedBefore: params.CreatedBefore,
					CreatedAfter:  params.CreatedAfter,
					OrderBy:       params.OrderBy,
					PageSize:      pageSize,
					PageToken:     pageToken,
				})
			}
			return paginate(ctx, params.MaxPages, "assignments", listFn, extractAssignmentsPage)
		})
	if err != nil {
		return AssignmentsResult{}, err
	}

	return AssignmentsResult{
		Meta:         page.Meta,
		Assignments:  page.Items,
		TotalSize:    page.TotalSize,
		PartialError: page.PartialError,
	}, nil
}

// extractTagsPage adapts a tags list envelope for the paginator.
func extractTagsPage(list *components.TagsList) pageData[Tag] {
	items := make([]Tag, 0, len(list.Tags))
	for _, t := range list.Tags {
		items = append(items, mapTag(t))
	}

	nextPageToken := ""
	if npt := list.GetNextPageToken(); npt != nil {
		nextPageToken = *npt
	}

	return pageData[Tag]{Items: items, TotalSize: list.TotalSize, NextPageToken: nextPageToken}
}

// extractAssignmentsPage adapts an assignments list envelope for the paginator.
func extractAssignmentsPage(list *components.TagAssignmentsList) pageData[Assignment] {
	items := make([]Assignment, 0, len(list.Assignments))
	for _, a := range list.Assignments {
		items = append(items, mapTagAssignment(a))
	}

	nextPageToken := ""
	if npt := list.GetNextPageToken(); npt != nil {
		nextPageToken = *npt
	}

	return pageData[Assignment]{Items: items, TotalSize: list.TotalSize, NextPageToken: nextPageToken}
}

// optionalInt64 narrows an unsigned page size to the signed type the client sends.
func optionalInt64(v mo.Option[uint64]) mo.Option[int64] {
	if !v.IsPresent() {
		return mo.None[int64]()
	}
	return mo.Some(int64(v.MustGet()))
}

// GetTag retrieves a single tag by name or UUID. The endpoint accepts either
// interchangeably, so the raw identifier is passed straight through (no resolve
// roundtrip). A second request counts the tag's assignments, which the tag
// payload itself does not carry.
func (s *tagsService) GetTag(
	ctx context.Context,
	params GetParams,
) (GetResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	// The endpoint resolves names itself, so this does not go through
	// resolveTagID - but it reads a UUID-shaped value as an ID exactly like the
	// rest of the API does, so the same name fallback applies.
	result, err := s.client.GetTag(ctx, orgIDStr, params.TagID.String())
	if err != nil {
		retry := s.resolveNameCollision(ctx, orgIDStr, params.TagID, err)
		if retry.IsAbsent() {
			return GetResult{}, err
		}
		result, err = s.client.GetTag(ctx, orgIDStr, retry.MustGet())
		if err != nil {
			return GetResult{}, err
		}
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

	// An empty ID means the response carried no tag to count against, so there is
	// nothing to ask the assignments endpoint about.
	var countErr cenclierrors.CencliError
	if tag.ID != "" {
		tag.AssetCount, countErr = s.assetCount(ctx, orgIDStr, tag.ID)
	}

	return GetResult{
		Meta:         meta,
		Tag:          tag,
		PartialError: cenclierrors.ToPartialError(countErr),
	}, nil
}

// assetCount reports how many assets a tag is assigned to, reading the total off
// a single-item assignments page. The UUID comes from the tag just fetched, so no
// resolution is needed. A failure is surfaced alongside the tag, not instead of it.
func (s *tagsService) assetCount(
	ctx context.Context,
	orgID mo.Option[string],
	tagID string,
) (*int64, cenclierrors.CencliError) {
	progress.ReportMessage(ctx, progress.StageFetch, "Counting assigned assets...")

	result, err := s.client.ListTagAssignments(ctx, client.ListTagAssignmentsRequest{
		OrgID:    orgID,
		TagID:    tagID,
		PageSize: mo.Some(int64(1)),
	})
	if err != nil {
		return nil, err
	}
	if result.Data == nil {
		return nil, nil
	}

	count := result.Data.TotalSize
	return &count, nil
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
// validated; a name is resolved to a UUID (the update endpoint accepts a UUID
// only) before the write.
func (s *tagsService) UpdateTag(
	ctx context.Context,
	params UpdateParams,
) (UpdateResult, cenclierrors.CencliError) {
	if err := validatePrivacy(params.Privacy); err != nil {
		return UpdateResult{}, err
	}

	orgIDStr := utilconvert.OptionalString(params.OrgID)

	result, err := callWithTag(ctx, s, orgIDStr, params.TagID,
		func(tagID string) (client.Result[components.Tag], cenclierrors.CencliError) {
			return s.client.UpdateTag(ctx, client.UpdateTagRequest{
				OrgID:       orgIDStr,
				TagID:       tagID,
				Name:        params.Name,
				Description: params.Description,
				Privacy:     params.Privacy,
			})
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

// DeleteTag removes a tag by name or UUID. A name is resolved to a UUID (the
// delete endpoint accepts a UUID only) before the deletion.
func (s *tagsService) DeleteTag(
	ctx context.Context,
	params DeleteParams,
) (DeleteResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	metadata, err := callWithTag(ctx, s, orgIDStr, params.TagID,
		func(tagID string) (client.Metadata, cenclierrors.CencliError) {
			return s.client.DeleteTag(ctx, orgIDStr, tagID)
		})
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

// Assign links a tag (by name or UUID) to explicit assets, one request per
// asset. A failure on one asset does not abort the rest: if every asset fails
// the first error is returned, otherwise the successes come back with a
// PartialError summarizing the failures.
func (s *tagsService) Assign(
	ctx context.Context,
	params AssignParams,
) (AssignResult, cenclierrors.CencliError) {
	if len(params.AssetIDs) == 0 {
		return AssignResult{}, NewNoAssetsError()
	}

	orgIDStr := utilconvert.OptionalString(params.OrgID)

	tagID, resolveErr := s.resolveTagID(ctx, orgIDStr, params.TagID)
	if resolveErr != nil {
		return AssignResult{}, resolveErr
	}

	result, err := s.assignEach(ctx, orgIDStr, params, tagID)
	if retry := s.retryTagForRun(ctx, orgIDStr, params.TagID, len(result.Assignments), result.Failures, err); retry.IsPresent() {
		return s.assignEach(ctx, orgIDStr, params, retry.MustGet())
	}
	return result, err
}

// assignEach runs the per-asset loop against one resolved tag ID.
func (s *tagsService) assignEach(
	ctx context.Context,
	orgIDStr mo.Option[string],
	params AssignParams,
	tagID string,
) (AssignResult, cenclierrors.CencliError) {
	total := len(params.AssetIDs)
	assignments := make([]Assignment, 0, total)
	var failures []AssignmentFailure
	var firstErr cenclierrors.CencliError
	var meta *responsemeta.ResponseMeta

	for i, assetID := range params.AssetIDs {
		// Stop early on cancellation, keeping whatever succeeded so far.
		if err := ctx.Err(); err != nil {
			firstErr = cenclierrors.ParseContextError(err)
			break
		}

		progress.ReportMessage(ctx, progress.StageProcess,
			fmt.Sprintf("Assigning tag (%d/%d)...", i+1, total))

		result, err := s.client.CreateTagAssignment(ctx, client.CreateTagAssignmentRequest{
			OrgID:   orgIDStr,
			TagID:   tagID,
			AssetID: assetID,
		})
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failures = append(failures, newAssignmentFailure(assetID, err))
			continue
		}

		if meta == nil && (result.Metadata.Request != nil || result.Metadata.Response != nil) {
			meta = responsemeta.NewResponseMeta(
				result.Metadata.Request,
				result.Metadata.Response,
				result.Metadata.Latency,
				result.Metadata.Attempts,
			)
		}
		if result.Data != nil {
			assignments = append(assignments, mapTagAssignment(*result.Data))
		}
	}

	partial, fatal := perAssetOutcome(len(assignments), len(failures), firstErr,
		func() cenclierrors.CencliError { return newAssignPartialError(len(failures), total) })
	if fatal != nil {
		return AssignResult{}, fatal
	}

	return AssignResult{
		Meta:         meta,
		TagID:        params.TagID.String(),
		Assignments:  assignments,
		Failures:     failures,
		PartialError: partial,
	}, nil
}

// Unassign removes a tag (by name or UUID) from explicit assets, looking up each
// asset's assignment before deleting it. Like Assign it is continue-on-error; an
// asset with no assignment is a per-asset failure.
func (s *tagsService) Unassign(
	ctx context.Context,
	params UnassignParams,
) (UnassignResult, cenclierrors.CencliError) {
	if len(params.AssetIDs) == 0 {
		return UnassignResult{}, NewNoAssetsError()
	}

	orgIDStr := utilconvert.OptionalString(params.OrgID)

	tagID, resolveErr := s.resolveTagID(ctx, orgIDStr, params.TagID)
	if resolveErr != nil {
		return UnassignResult{}, resolveErr
	}

	result, err := s.unassignEach(ctx, orgIDStr, params, tagID)
	if retry := s.retryTagForRun(ctx, orgIDStr, params.TagID, len(result.Unassigned), result.Failures, err); retry.IsPresent() {
		return s.unassignEach(ctx, orgIDStr, params, retry.MustGet())
	}
	return result, err
}

// unassignEach runs the per-asset lookup+delete loop against one resolved tag ID.
func (s *tagsService) unassignEach(
	ctx context.Context,
	orgIDStr mo.Option[string],
	params UnassignParams,
	tagID string,
) (UnassignResult, cenclierrors.CencliError) {
	total := len(params.AssetIDs)
	unassigned := make([]Assignment, 0, total)
	var failures []AssignmentFailure
	var firstErr cenclierrors.CencliError
	var meta *responsemeta.ResponseMeta

	for i, assetID := range params.AssetIDs {
		// Stop early on cancellation, keeping whatever succeeded so far.
		if err := ctx.Err(); err != nil {
			firstErr = cenclierrors.ParseContextError(err)
			break
		}

		progress.ReportMessage(ctx, progress.StageProcess,
			fmt.Sprintf("Unassigning tag (%d/%d)...", i+1, total))

		// The delete endpoint is keyed by assignment ID, not asset ID, so look it up.
		listResult, err := s.client.ListTagAssignments(ctx, client.ListTagAssignmentsRequest{
			OrgID:    orgIDStr,
			TagID:    tagID,
			AssetID:  mo.Some(assetID),
			PageSize: mo.Some(int64(1)),
		})
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failures = append(failures, newAssignmentFailure(assetID, err))
			continue
		}
		if listResult.Data == nil || len(listResult.Data.Assignments) == 0 {
			notAssigned := NewAssetNotAssignedError(assetID)
			if firstErr == nil {
				firstErr = notAssigned
			}
			failures = append(failures, newAssignmentFailure(assetID, notAssigned))
			continue
		}

		assignment := listResult.Data.Assignments[0]
		metadata, err := s.client.DeleteTagAssignment(ctx, orgIDStr, tagID, assignment.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failures = append(failures, newAssignmentFailure(assetID, err))
			continue
		}

		if meta == nil && (metadata.Request != nil || metadata.Response != nil) {
			meta = responsemeta.NewResponseMeta(
				metadata.Request,
				metadata.Response,
				metadata.Latency,
				metadata.Attempts,
			)
		}
		unassigned = append(unassigned, mapTagAssignment(assignment))
	}

	partial, fatal := perAssetOutcome(len(unassigned), len(failures), firstErr,
		func() cenclierrors.CencliError { return newUnassignPartialError(len(failures), total) })
	if fatal != nil {
		return UnassignResult{}, fatal
	}

	return UnassignResult{
		Meta:         meta,
		TagID:        params.TagID.String(),
		Unassigned:   unassigned,
		Failures:     failures,
		PartialError: partial,
	}, nil
}

// resolveTagID returns a concrete tag UUID for the given identifier. A value
// that already parses as a UUID is returned unchanged with no API call; a name
// is resolved to its UUID via an exact-match ListTags lookup. Reused by the
// UUID-only endpoints (update, delete, and later assignments/operations).
func (s *tagsService) resolveTagID(
	ctx context.Context,
	orgID mo.Option[string],
	tagID identifiers.TagID,
) (string, cenclierrors.CencliError) {
	// Never look up an empty identifier: an empty name filter matches nothing
	// meaningful and would resolve to an arbitrary tag.
	if tagID.String() == "" {
		return "", NewEmptyTagIDError()
	}

	// A UUID needs no resolution — pass it straight through, no lookup. If no tag
	// actually has that ID, resolveNameCollision retries it as a name afterwards.
	if tagID.UID().IsPresent() {
		return tagID.String(), nil
	}

	id, err := s.lookupTagIDByName(ctx, orgID, tagID.String())
	if err != nil {
		return "", err
	}
	if id.IsAbsent() {
		return "", NewTagNotFoundError(tagID.String())
	}
	return id.MustGet(), nil
}

// lookupTagIDByName finds a tag's UUID by its exact name. An absent result means
// no tag carries that name, which is not on its own an error.
func (s *tagsService) lookupTagIDByName(
	ctx context.Context,
	orgID mo.Option[string],
	name string,
) (mo.Option[string], cenclierrors.CencliError) {
	result, err := s.client.ListTags(ctx, client.ListTagsRequest{
		OrgID:    orgID,
		Name:     mo.Some(name),
		PageSize: mo.Some(int64(1)),
	})
	if err != nil {
		return mo.None[string](), err
	}
	if result.Data == nil || len(result.Data.Tags) == 0 {
		return mo.None[string](), nil
	}
	return mo.Some(result.Data.Tags[0].ID), nil
}

// resolveNameCollision settles the case resolveTagID cannot: an identifier that
// parses as a UUID but is actually a tag's name. IDs win, so it runs only once
// the API has said nothing carries that ID, keeping the common path at zero
// extra requests. Absent means "keep the original error".
func (s *tagsService) resolveNameCollision(
	ctx context.Context,
	orgID mo.Option[string],
	tagID identifiers.TagID,
	cause cenclierrors.CencliError,
) mo.Option[string] {
	if !tagID.UID().IsPresent() || !isMissingTagError(cause) {
		return mo.None[string]()
	}

	// A failure here is not worth surfacing - the caller already has a real error.
	id, err := s.lookupTagIDByName(ctx, orgID, tagID.String())
	if err != nil || id.IsAbsent() || id.MustGet() == tagID.String() {
		return mo.None[string]()
	}
	return id
}

// callWithTag resolves the identifier, runs fn against it, and retries once as a
// tag name if the API says nothing carries that ID. Every single-request tag verb
// uses it, so the two readings are tried in the same order everywhere.
func callWithTag[T any](
	ctx context.Context,
	s *tagsService,
	orgID mo.Option[string],
	tagID identifiers.TagID,
	fn func(resolved string) (T, cenclierrors.CencliError),
) (T, cenclierrors.CencliError) {
	var zero T

	resolved, err := s.resolveTagID(ctx, orgID, tagID)
	if err != nil {
		return zero, err
	}

	result, err := fn(resolved)
	if err == nil {
		return result, nil
	}
	if retry := s.resolveNameCollision(ctx, orgID, tagID, err); retry.IsPresent() {
		return fn(retry.MustGet())
	}
	return zero, err
}

// retryTagForRun reports the tag to re-run a per-asset loop against: set only
// when the run placed nothing and every asset failed the way a missing tag does,
// so re-running cannot duplicate work. Shared by Assign and Unassign.
func (s *tagsService) retryTagForRun(
	ctx context.Context,
	orgID mo.Option[string],
	tagID identifiers.TagID,
	succeeded int,
	failures []AssignmentFailure,
	err cenclierrors.CencliError,
) mo.Option[string] {
	if err != nil || succeeded > 0 || len(failures) == 0 {
		return mo.None[string]()
	}
	return s.resolveNameCollision(ctx, orgID, tagID, failures[0].Err)
}

// isMissingTagError reports whether the API said no such tag exists. A 403 counts:
// the API masks existence, so forbidden and missing are indistinguishable.
func isMissingTagError(err cenclierrors.CencliError) bool {
	var coded interface{ StatusCode() mo.Option[int64] }
	if !errors.As(err, &coded) {
		return false
	}
	status := coded.StatusCode()
	return status.IsPresent() && (status.MustGet() == 404 || status.MustGet() == 403)
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

// mapTagAssignment converts an SDK tag assignment into the domain DTO.
func mapTagAssignment(a components.TagAssignment) Assignment {
	return Assignment{
		ID:          a.ID,
		TagID:       a.TagID,
		AssetID:     a.AssetID,
		AssetType:   string(a.AssetType),
		PlatformRef: a.PlatformRef,
		CreatedBy:   a.CreatedBy,
		CreatedAt:   a.CreatedAt,
	}
}
