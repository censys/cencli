package tags

import (
	"context"
	"strings"

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
	tagID, resolveErr := s.resolveTagID(ctx, orgIDStr, params.TagID)
	if resolveErr != nil {
		return BulkAssignResult{}, resolveErr
	}

	result, err := s.client.BulkCreateTagAssignments(ctx, client.BulkCreateTagAssignmentsRequest{
		OrgID:     orgIDStr,
		TagID:     tagID,
		Query:     query,
		MaxAssets: params.MaxAssets,
	})
	if err != nil {
		return BulkAssignResult{}, err
	}

	meta, operation := mapOperationResult(result)
	return BulkAssignResult{Meta: meta, Operation: operation}, nil
}
