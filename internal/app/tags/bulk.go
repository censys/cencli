package tags

import (
	"context"
	"strings"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	utilconvert "github.com/censys/cencli/internal/pkg/convertutil"
)

// BulkAssign assigns a tag (by name or UUID) to every asset matching a CenQL
// query. The endpoint answers 202 with the operation tracking the job, so this
// returns as soon as the job is accepted; the caller decides whether to wait.
func (s *tagsService) BulkAssign(
	ctx context.Context,
	params BulkAssignParams,
) (BulkAssignResult, cenclierrors.CencliError) {
	query := strings.TrimSpace(params.Query)
	if query == "" {
		return BulkAssignResult{}, NewEmptyQueryError()
	}

	orgIDStr := utilconvert.OptionalString(params.OrgID)

	// The endpoint keys off the tag UUID, so a name costs one lookup first.
	result, err := callWithTag(ctx, s, orgIDStr, params.TagID,
		func(tagID string) (client.Result[components.TagOperation], cenclierrors.CencliError) {
			return s.client.BulkCreateTagAssignments(ctx, client.BulkCreateTagAssignmentsRequest{
				OrgID:     orgIDStr,
				TagID:     tagID,
				Query:     query,
				MaxAssets: params.MaxAssets,
			})
		})
	if err != nil {
		return BulkAssignResult{}, err
	}

	meta, operation := mapOperationResult(result)
	return BulkAssignResult{Meta: meta, Operation: operation}, nil
}

// BulkUnassign removes a tag (by name or UUID) from the assignments matching the
// given time filters, or from every assignment when no filter is given. Like
// BulkAssign it returns as soon as the job is accepted.
func (s *tagsService) BulkUnassign(
	ctx context.Context,
	params BulkUnassignParams,
) (BulkUnassignResult, cenclierrors.CencliError) {
	// An impossible window would remove nothing while still spending an
	// operation, which reads like a successful wipe.
	if err := ValidateTimeWindow(params.CreatedBefore, params.CreatedAfter); err != nil {
		return BulkUnassignResult{}, err
	}

	orgIDStr := utilconvert.OptionalString(params.OrgID)

	// The endpoint keys off the tag UUID, so a name costs one lookup first.
	result, err := callWithTag(ctx, s, orgIDStr, params.TagID,
		func(tagID string) (client.Result[components.TagOperation], cenclierrors.CencliError) {
			return s.client.BulkDeleteTagAssignments(ctx, client.BulkDeleteTagAssignmentsRequest{
				OrgID:         orgIDStr,
				TagID:         tagID,
				CreatedBefore: params.CreatedBefore,
				CreatedAfter:  params.CreatedAfter,
			})
		})
	if err != nil {
		return BulkUnassignResult{}, err
	}

	meta, operation := mapOperationResult(result)
	return BulkUnassignResult{Meta: meta, Operation: operation}, nil
}
