package collections

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/mo"

	"github.com/censys/cencli/internal/app/collections"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/ui/form"
)

// requireCollectionID validates a positional collection identifier. The
// endpoints declare collection_uid as a UUID, so anything else is rejected here
// rather than spent on a request that could only fail.
func requireCollectionID(raw string) (identifiers.CollectionID, cenclierrors.CencliError) {
	trimmed := strings.TrimSpace(raw)
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return identifiers.CollectionID{}, collections.NewInvalidCollectionIDError(trimmed)
	}
	return identifiers.NewCollectionID(parsed), nil
}

// parsePaginationFlags reads the shared --page-size/--max-pages pair. The
// max-pages flag is built without a lower bound so the -1 "all pages" sentinel
// gets through, which leaves rejecting 0 and negatives to this switch.
func parsePaginationFlags(
	pageSizeFlag, maxPagesFlag flags.IntegerFlag,
) (pageSize, maxPages mo.Option[uint64], err cenclierrors.CencliError) {
	rawPageSize, err := pageSizeFlag.Value()
	if err != nil {
		return pageSize, maxPages, err
	}
	if rawPageSize.IsPresent() {
		pageSize = mo.Some(uint64(rawPageSize.MustGet()))
	}

	rawMaxPages, err := maxPagesFlag.Value()
	if err != nil {
		return pageSize, maxPages, err
	}
	if rawMaxPages.IsPresent() {
		switch v := rawMaxPages.MustGet(); {
		case v == -1:
			maxPages = mo.None[uint64]()
		case v <= 0:
			return pageSize, maxPages, flags.NewIntegerFlagInvalidValueError("max-pages", v, "must be -1 or >= 1")
		default:
			maxPages = mo.Some(uint64(v))
		}
	}

	return pageSize, maxPages, nil
}

// warnFetchingAllPages tells the user that --max-pages=-1 will keep requesting
// pages until the server runs out, since the call count is otherwise invisible
// until it is spent. Mirrors the same warning on search and tags.
func warnFetchingAllPages(quiet bool, logger *slog.Logger, maxPages mo.Option[uint64]) {
	if quiet || maxPages.IsPresent() {
		return
	}
	msg := styles.GlobalStyles.Warning.Render(
		"Warning: fetching all pages (--max-pages=-1). This may take a while and increase API usage.")
	formatter.Println(formatter.Stderr, msg)
	logger.Debug("fetching all pages", "message", msg)
}

// optionalNonEmpty treats a blank flag value as "not provided", so it is
// omitted from the request rather than sent as an empty value.
func optionalNonEmpty(v string) mo.Option[string] {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return mo.None[string]()
	}
	return mo.Some(trimmed)
}

// printNote writes an advisory line to stderr unless --quiet asked for silence.
// Outcome messages (an abort, an error) are not notes and always print.
func printNote(quiet bool, message string) {
	if quiet {
		return
	}
	formatter.Println(formatter.Stderr, message)
}

// confirmAction asks the user to approve a destructive action, translating an
// aborted prompt into the repo's interrupted error. A false answer means the
// caller should stop without treating it as a failure.
func confirmAction(
	ctx context.Context,
	confirm func(ctx context.Context, message string) (bool, error),
	message string,
) (bool, cenclierrors.CencliError) {
	confirmed, err := confirm(ctx, message)
	if err != nil {
		if errors.Is(err, form.ErrUserAborted) {
			return false, cenclierrors.NewInterruptedError()
		}
		return false, cenclierrors.NewCencliError(err)
	}
	return confirmed, nil
}
