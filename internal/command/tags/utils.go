package tags

import (
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/input"
)

// requireTagID builds a TagID from a positional argument and rejects an empty
// (or whitespace-only) identifier, which no tag command can act on. Shared by
// get, update, and delete so the check is applied uniformly at the boundary.
func requireTagID(raw string) (identifiers.TagID, cenclierrors.CencliError) {
	id := identifiers.NewTagID(raw)
	if id.String() == "" {
		return id, tags.NewEmptyTagIDError()
	}
	return id, nil
}

// gatherAssetIDs collects and classifies the assets an assign/unassign command
// should act on: from --input-file when set, otherwise the positional args after
// the tag (each comma-split). Mixed asset types are allowed — each assignment is
// an independent request, so AssetType() is not called and only genuinely
// unparseable inputs are rejected. An empty result is an error. Shared by assign
// and unassign.
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
