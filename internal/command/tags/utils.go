package tags

import (
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/input"
	"github.com/censys/cencli/internal/pkg/styles"
)

// requireTagID builds a TagID from a positional argument and rejects an empty
// (or whitespace-only) identifier, which no tag command can act on.
func requireTagID(raw string) (identifiers.TagID, cenclierrors.CencliError) {
	id := identifiers.NewTagID(raw)
	if id.String() == "" {
		return id, tags.NewEmptyTagIDError()
	}
	return id, nil
}

// requireOperationID validates a positional operation identifier. The endpoint
// declares operation_id as a UUID, so anything else is rejected here rather than
// spent on a request that could only come back as a 422.
func requireOperationID(raw string) (string, cenclierrors.CencliError) {
	trimmed := strings.TrimSpace(raw)
	if _, err := uuid.Parse(trimmed); err != nil {
		return "", tags.NewInvalidOperationIDError(trimmed)
	}
	return trimmed, nil
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
// until it is spent. Mirrors the same warning on search.
func warnFetchingAllPages(quiet bool, logger *slog.Logger, maxPages mo.Option[uint64]) {
	if quiet || maxPages.IsPresent() {
		return
	}
	msg := styles.GlobalStyles.Warning.Render(
		"Warning: fetching all pages (--max-pages=-1). This may take a while and increase API usage.")
	formatter.Println(formatter.Stderr, msg)
	logger.Debug("fetching all pages", "message", msg)
}

// uuidFilterString renders an optional UUID filter as the string the service
// layer threads through, leaving an absent filter absent.
func uuidFilterString(v mo.Option[uuid.UUID]) mo.Option[string] {
	if !v.IsPresent() {
		return mo.None[string]()
	}
	return mo.Some(v.MustGet().String())
}

// optionalNonEmpty treats a blank flag value as "filter not provided", so it is
// omitted from the request rather than sent as an empty filter.
func optionalNonEmpty(v string) mo.Option[string] {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return mo.None[string]()
	}
	return mo.Some(trimmed)
}

// gatherAssetIDs collects the assets an assign/unassign command should act on:
// from --input-file when set, otherwise the positional args after the tag (each
// comma-split).
func gatherAssetIDs(cmd *cobra.Command, inputFile flags.FileFlag, args []string) ([]string, cenclierrors.CencliError) {
	var raw []string
	if inputFile.IsSet() {
		lines, err := inputFile.Lines(cmd)
		if err != nil {
			return nil, err
		}
		raw = lines
	} else {
		assetArgs := args[1:]
		if len(assetArgs) == 0 {
			return nil, assets.NewNoAssetsError()
		}
		for _, a := range assetArgs {
			raw = append(raw, input.SplitString(a)...)
		}
	}

	return classifyAssetIDs(raw)
}

// classifyAssetIDs validates raw asset inputs and returns their normalized IDs.
// Mixed asset types are allowed — every caller acts on one asset per request, so
// AssetType() is never consulted and only unparseable inputs are rejected.
func classifyAssetIDs(raw []string) ([]string, cenclierrors.CencliError) {
	classifier := assets.NewAssetClassifier(raw...)
	if unknown := classifier.UnknownAssets(); len(unknown) > 0 {
		return nil, assets.NewInvalidAssetIDError(unknown[0], "unable to infer asset type")
	}
	ids := classifier.KnownAssetIDs()
	if len(ids) == 0 {
		return nil, assets.NewNoAssetsError()
	}
	return ids, nil
}
