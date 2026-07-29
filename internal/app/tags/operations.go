package tags

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/mo"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
)

const (
	// allTagsSelector lists operations across every tag in the organization.
	// Only the list endpoint accepts it; get and cancel require a tag UUID.
	allTagsSelector = "-"

	// Poll pacing for WaitForOperation: start responsive, then back off so a
	// long-running job does not burn requests.
	initialPollInterval = 2 * time.Second
	maxPollInterval     = 15 * time.Second
)

// ListOperations lists the asynchronous bulk jobs for one tag, or for every tag
// in the organization when no tag is given.
func (s *tagsService) ListOperations(
	ctx context.Context,
	params OperationsParams,
) (OperationsResult, cenclierrors.CencliError) {
	// validate filter enums against the API contract before making any request
	if err := validateOperationType(params.Type); err != nil {
		return OperationsResult{}, err
	}
	if err := validateOperationStatus(params.Status); err != nil {
		return OperationsResult{}, err
	}
	if err := validateOperationsOrderBy(params.OrderBy); err != nil {
		return OperationsResult{}, err
	}

	// handle pagination invariants
	if err := validatePaginationParams(params.PageSize, params.MaxPages); err != nil {
		return OperationsResult{}, err
	}

	orgIDStr := utilconvert.OptionalString(params.OrgID)

	// An absent tag lists org-wide; a name is resolved since the filter is by UUID.
	tagID := allTagsSelector
	if params.TagID.IsPresent() {
		resolved, resolveErr := s.resolveTagID(ctx, orgIDStr, params.TagID.MustGet())
		if resolveErr != nil {
			return OperationsResult{}, resolveErr
		}
		tagID = resolved
	}

	pageSize := optionalInt64(params.PageSize)

	listFn := func(pageToken mo.Option[string]) (client.Result[components.TagOperationsList], client.ClientError) {
		return s.client.ListTagOperations(ctx, client.ListTagOperationsRequest{
			OrgID:     orgIDStr,
			TagID:     tagID,
			Type:      params.Type,
			Status:    params.Status,
			OrderBy:   params.OrderBy,
			PageSize:  pageSize,
			PageToken: pageToken,
		})
	}

	page, err := paginate(ctx, params.MaxPages, "operations", listFn, extractOperationsPage)
	if err != nil {
		return OperationsResult{}, err
	}

	return OperationsResult{
		Meta:         page.Meta,
		Operations:   page.Items,
		TotalSize:    page.TotalSize,
		PartialError: page.PartialError,
	}, nil
}

// GetOperation retrieves a single bulk tag operation. Both path parameters are
// UUID-only, so a tag name is resolved first.
func (s *tagsService) GetOperation(
	ctx context.Context,
	params GetOperationParams,
) (GetOperationResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	tagID, operationID, err := s.resolveOperationTarget(ctx, orgIDStr, params.TagID, params.OperationID)
	if err != nil {
		return GetOperationResult{}, err
	}

	result, err := s.client.GetTagOperation(ctx, orgIDStr, tagID, operationID)
	if err != nil {
		return GetOperationResult{}, err
	}

	return newGetOperationResult(result), nil
}

// CancelOperation asks the API to stop a running bulk job. Work already
// committed before the cancellation is kept, so this narrows a job rather than
// undoing it. Both path parameters are UUID-only, so a tag name is resolved first.
func (s *tagsService) CancelOperation(
	ctx context.Context,
	params CancelOperationParams,
) (CancelOperationResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	tagID, operationID, err := s.resolveOperationTarget(ctx, orgIDStr, params.TagID, params.OperationID)
	if err != nil {
		return CancelOperationResult{}, err
	}

	result, err := s.client.CancelTagOperation(ctx, orgIDStr, tagID, operationID)
	if err != nil {
		return CancelOperationResult{}, err
	}

	meta, operation := mapOperationResult(result)
	return CancelOperationResult{Meta: meta, Operation: operation}, nil
}

// WaitForOperation polls an operation until it reaches a terminal status, the
// optional timeout expires, or the context is cancelled. A terminal status is
// not an error: every outcome comes back as a result, and the caller decides
// what it means for the exit code.
func (s *tagsService) WaitForOperation(
	ctx context.Context,
	params WaitParams,
) (GetOperationResult, cenclierrors.CencliError) {
	orgIDStr := utilconvert.OptionalString(params.OrgID)

	// Resolve once up front rather than per poll, so a name costs one extra
	// request for the whole wait instead of one per tick.
	tagID, operationID, err := s.resolveOperationTarget(ctx, orgIDStr, params.TagID, params.OperationID)
	if err != nil {
		return GetOperationResult{}, err
	}

	pollCtx := ctx
	if params.Timeout.IsPresent() {
		var cancel context.CancelFunc
		pollCtx, cancel = context.WithTimeout(ctx, params.Timeout.MustGet())
		defer cancel()
	}

	interval := initialPollInterval
	lastStatus := ""

	for {
		result, getErr := s.client.GetTagOperation(pollCtx, orgIDStr, tagID, operationID)
		if getErr != nil {
			// The parent going away outranks the timeout: report what actually
			// stopped us rather than whichever context observed it first.
			if waitErr := s.waitContextError(ctx, pollCtx, operationID, lastStatus, params.Timeout); waitErr != nil {
				return GetOperationResult{}, waitErr
			}
			return GetOperationResult{}, getErr
		}

		out := newGetOperationResult(result)
		lastStatus = out.Operation.Status

		if isTerminalStatus(out.Operation.Status) {
			return out, nil
		}

		reportOperationProgress(pollCtx, out.Operation)

		if sleepErr := s.sleep(pollCtx, interval); sleepErr != nil {
			if waitErr := s.waitContextError(ctx, pollCtx, operationID, lastStatus, params.Timeout); waitErr != nil {
				return GetOperationResult{}, waitErr
			}
			return GetOperationResult{}, cenclierrors.ParseContextError(sleepErr)
		}

		interval = min(interval*2, maxPollInterval)
	}
}

// waitContextError explains why polling stopped, preferring the caller's own
// cancellation over an expired wait timeout. It returns nil when neither
// context is done, leaving the underlying error to speak for itself.
func (s *tagsService) waitContextError(
	ctx, pollCtx context.Context,
	operationID, lastStatus string,
	timeout mo.Option[time.Duration],
) cenclierrors.CencliError {
	if parentErr := ctx.Err(); parentErr != nil {
		return cenclierrors.ParseContextError(parentErr)
	}
	if pollCtx.Err() != nil && timeout.IsPresent() {
		status := lastStatus
		if status == "" {
			status = "unfinished"
		}
		return NewOperationWaitTimeoutError(operationID, status, timeout.MustGet())
	}
	return nil
}

// resolveOperationTarget turns a caller-supplied tag identifier and operation ID
// into the UUID pair the operation endpoints require.
func (s *tagsService) resolveOperationTarget(
	ctx context.Context,
	orgID mo.Option[string],
	tagID identifiers.TagID,
	operationID string,
) (string, string, cenclierrors.CencliError) {
	if _, err := uuid.Parse(operationID); err != nil {
		return "", "", NewInvalidOperationIDError(operationID)
	}

	resolved, err := s.resolveTagID(ctx, orgID, tagID)
	if err != nil {
		return "", "", err
	}
	return resolved, operationID, nil
}

// isTerminalStatus reports whether an operation has finished, for any outcome.
func isTerminalStatus(status string) bool {
	switch components.TagOperationStatus(status) {
	case components.TagOperationStatusSucceeded,
		components.TagOperationStatusLimitReached,
		components.TagOperationStatusFailed,
		components.TagOperationStatusCancelled:
		return true
	default:
		return false
	}
}

// reportOperationProgress surfaces how far a running operation has got. The
// total is approximate for bulk_create, so it is only shown when known.
func reportOperationProgress(ctx context.Context, op TagOperation) {
	msg := fmt.Sprintf("Operation %s: %d processed", op.Status, op.ProcessedCount)
	if op.TotalCount > 0 {
		msg = fmt.Sprintf("Operation %s: %d/%d processed (approx)",
			op.Status, op.ProcessedCount, op.TotalCount)
	}
	progress.ReportMessage(ctx, progress.StageProcess, msg)
}

// newGetOperationResult builds the single-operation result from a client call.
func newGetOperationResult(result client.Result[components.TagOperation]) GetOperationResult {
	meta, operation := mapOperationResult(result)
	return GetOperationResult{Meta: meta, Operation: operation}
}

// mapOperationResult unpacks a client response carrying one operation. Shared by
// every endpoint that answers with a TagOperation: get, wait, and bulk submit.
func mapOperationResult(
	result client.Result[components.TagOperation],
) (*responsemeta.ResponseMeta, TagOperation) {
	var meta *responsemeta.ResponseMeta
	if result.Metadata.Request != nil || result.Metadata.Response != nil {
		meta = responsemeta.NewResponseMeta(
			result.Metadata.Request,
			result.Metadata.Response,
			result.Metadata.Latency,
			result.Metadata.Attempts,
		)
	}

	var operation TagOperation
	if result.Data != nil {
		operation = mapTagOperation(*result.Data)
	}
	return meta, operation
}

// extractOperationsPage adapts an operations list envelope for the paginator.
func extractOperationsPage(list *components.TagOperationsList) pageData[TagOperation] {
	items := make([]TagOperation, 0, len(list.Operations))
	for _, op := range list.Operations {
		items = append(items, mapTagOperation(op))
	}

	nextPageToken := ""
	if npt := list.GetNextPageToken(); npt != nil {
		nextPageToken = *npt
	}

	return pageData[TagOperation]{Items: items, TotalSize: list.TotalSize, NextPageToken: nextPageToken}
}

// mapTagOperation converts an SDK tag operation into the domain DTO.
func mapTagOperation(op components.TagOperation) TagOperation {
	return TagOperation{
		ID:              op.ID,
		TagID:           op.TagID,
		TagName:         op.TagName,
		Type:            string(op.Type),
		Status:          string(op.Status),
		Query:           op.Query,
		TotalCount:      op.TotalCount,
		ProcessedCount:  op.ProcessedCount,
		SuccessfulCount: op.SuccessfulCount,
		StatusMessage:   op.StatusMessage,
		ErrorMessage:    op.ErrorMessage,
		CreatedAt:       op.CreatedAt,
		EndedAt:         op.EndedAt,
	}
}
