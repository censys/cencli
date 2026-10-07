package collections

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/mo"

	"github.com/censys/cencli/internal/app/collections"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/formatter"
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

// shellQuote wraps s in single quotes for a POSIX shell, so a copied command
// line passes it through literally ($, backticks, and spaces included).
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
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
