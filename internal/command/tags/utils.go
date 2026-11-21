package tags

import (
	"context"
	"errors"
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
	"github.com/censys/cencli/internal/pkg/ui/form"
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

// assetTypesByID maps each already-validated asset ID to its type, so a failed
// asset can still report one - the API only echoes a type back for assets it
// accepted. The values use the API's vocabulary, since successful rows in the
// same table are labeled by it: the domain says "webproperty", the API
// "web_property".
func assetTypesByID(ids []string) map[string]string {
	classifier := assets.NewAssetClassifier(ids...)
	types := make(map[string]string, len(ids))

	for _, h := range classifier.HostIDs() {
		types[h.String()] = "host"
	}
	for _, c := range classifier.CertificateIDs() {
		types[c.String()] = "certificate"
	}
	for _, w := range classifier.WebPropertyIDs() {
		types[w.String()] = "web_property"
	}
	return types
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
