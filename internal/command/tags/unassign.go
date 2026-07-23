package tags

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/input"
	"github.com/censys/cencli/internal/pkg/term"
	"github.com/censys/cencli/internal/pkg/ui/form"
)

const unassignCmdName = "unassign"

// UnassignCommand implements `tags unassign <tag> [asset...]`, removing a tag
// from explicit assets. Removing more than one asset (or reading from
// --input-file) prompts for confirmation unless --yes is set.
type UnassignCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags unassignCommandFlags
	// state - populated by PreRun
	orgID         mo.Option[identifiers.OrganizationID]
	tagID         identifiers.TagID
	assetIDs      []string
	yes           bool
	confirmNeeded bool
	// result stores the unassignment outcome for rendering
	result tags.UnassignResult
	// seams - overridable in tests; defaulted in NewUnassignCommand
	confirm    func(ctx context.Context, message string) (bool, error)
	stdinIsTTY func() bool
}

type unassignCommandFlags struct {
	orgID     flags.OrgIDFlag
	inputFile flags.FileFlag
	yes       flags.BoolFlag
}

// unassignedAsset is the data-mode payload for a single unassignment outcome.
type unassignedAsset struct {
	Asset       string `json:"asset" yaml:"asset"`
	AssetType   string `json:"asset_type,omitempty" yaml:"asset_type,omitempty"`
	PlatformRef string `json:"platform_ref,omitempty" yaml:"platform_ref,omitempty"`
	Unassigned  bool   `json:"unassigned" yaml:"unassigned"`
	Error       string `json:"error,omitempty" yaml:"error,omitempty"`
}

var _ command.Command = (*UnassignCommand)(nil)

func NewUnassignCommand(cmdContext *command.Context) *UnassignCommand {
	return &UnassignCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
		confirm:     form.Confirm,
		stdinIsTTY:  func() bool { return term.IsTTY(os.Stdin) },
	}
}

func (c *UnassignCommand) Use() string {
	return fmt.Sprintf("%s <tag> [asset...]", unassignCmdName)
}

func (c *UnassignCommand) Short() string {
	return "Unassign a tag from one or more assets"
}

func (c *UnassignCommand) Long() string {
	return `Unassign a tag, by its name or UUID, from one or more assets (host IPs, certificate SHA-256 fingerprints, or web property hostname:port).

Assets can be passed as positional arguments or read from a file (or STDIN) with --input-file. Assets of different types can be mixed in a single call. Each asset is unassigned independently: if one fails the rest still proceed, and the per-asset outcomes are reported.

Unassigning more than one asset (or reading assets from --input-file) prompts for confirmation. Use --yes to skip the prompt; in a non-interactive terminal --yes is required for those cases.`
}

func (c *UnassignCommand) Examples() []string {
	return []string{
		"<tag> <ip-address>",
		"<tag> <ip-address> <ip-address>",
		"<tag> <hostname:port>",
		"<tag> <sha256-fingerprint>",
		"<tag> <ip-address> <hostname:port>  # asset types can be mixed",
		"<tag> --input-file <file>",
		"<tag> --input-file -  # read assets from STDIN",
	}
}

func (c *UnassignCommand) Args() command.PositionalArgs {
	// At least the tag; assets may instead come from --input-file.
	return command.MinimumNArgs(1)
}

func (c *UnassignCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *UnassignCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *UnassignCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.inputFile = flags.NewFileFlag(c.Flags(), false, "input-file", "i", "file to read the assets from (or - for STDIN). Overrides positional asset arguments.")
	c.flags.yes = flags.NewBoolFlag(c.Flags(), "yes", "y", false, "skip the confirmation prompt")
	return nil
}

func (c *UnassignCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}

	yes, err := c.flags.yes.Value()
	if err != nil {
		return err
	}
	c.yes = yes

	c.tagID, err = requireTagID(args[0])
	if err != nil {
		return err
	}

	rawAssets, err := c.gatherRawAssets(cmd, args)
	if err != nil {
		return err
	}

	// Don't call AssetType(): mixed types are fine here (each unassignment is
	// independent), so reject only genuinely unparseable inputs.
	classifier := assets.NewAssetClassifier(rawAssets...)
	if unknown := classifier.UnknownAssets(); len(unknown) > 0 {
		return assets.NewInvalidAssetIDError(unknown[0], "unable to infer asset type")
	}
	c.assetIDs = classifier.KnownAssetIDs()
	if len(c.assetIDs) == 0 {
		return assets.NewNoAssetsError()
	}

	// A single positional asset does not prompt.
	c.confirmNeeded = c.flags.inputFile.IsSet() || len(c.assetIDs) > 1

	// Gate before resolving the service so a non-interactive run that needs a
	// prompt fails cleanly rather than proceeding (and before auth is required).
	if c.confirmNeeded && !c.yes && !c.stdinIsTTY() {
		return NewConfirmationRequiredError()
	}

	return c.resolveTagsService()
}

// gatherRawAssets returns raw asset strings from --input-file when set,
// otherwise from the positional args after the tag (each comma-split).
func (c *UnassignCommand) gatherRawAssets(cmd *cobra.Command, args []string) ([]string, cenclierrors.CencliError) {
	if c.flags.inputFile.IsSet() {
		return c.flags.inputFile.Lines(cmd)
	}
	assetArgs := args[1:]
	if len(assetArgs) == 0 {
		return nil, assets.NewNoAssetsError()
	}
	var raw []string
	for _, a := range assetArgs {
		raw = append(raw, input.SplitString(a)...)
	}
	return raw, nil
}

func (c *UnassignCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_is_uuid", c.tagID.UID().IsPresent(),
		"count", len(c.assetIDs),
		"yes", c.yes,
	)

	if c.confirmNeeded && !c.yes {
		message := fmt.Sprintf("Unassign tag %q from %d asset(s)?", c.tagID.String(), len(c.assetIDs))
		confirmed, err := c.confirm(cmd.Context(), message)
		if err != nil {
			if errors.Is(err, form.ErrUserAborted) {
				return cenclierrors.NewInterruptedError()
			}
			return cenclierrors.NewCencliError(err)
		}
		if !confirmed {
			formatter.Println(formatter.Stderr, "Unassign aborted.")
			return nil
		}
	}

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Unassigning tag...",
		func(pctx context.Context) cenclierrors.CencliError {
			var unassignErr cenclierrors.CencliError
			c.result, unassignErr = c.tagsSvc.Unassign(pctx, tags.UnassignParams{
				OrgID:    c.orgID,
				TagID:    c.tagID,
				AssetIDs: c.assetIDs,
			})
			return unassignErr
		},
	)
	if err != nil {
		logger.Debug("unassign tag failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	if renderErr := c.PrintData(c, c.unassignmentViews()); renderErr != nil {
		return renderErr
	}

	if len(c.result.Unassigned) > 0 {
		formatter.Println(formatter.Stderr,
			"Note: unassigned tags may take a few minutes to disappear from `tags:` search results.")
	}

	if c.result.PartialError != nil {
		formatter.PrintError(c.result.PartialError, cmd)
	}

	return nil
}

func (c *UnassignCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}

// unassignmentViews builds the render payload: successes first, then failures.
func (c *UnassignCommand) unassignmentViews() []unassignedAsset {
	views := make([]unassignedAsset, 0, len(c.result.Unassigned)+len(c.result.Failures))
	for _, a := range c.result.Unassigned {
		views = append(views, unassignedAsset{
			Asset:       a.AssetID,
			AssetType:   a.AssetType,
			PlatformRef: a.PlatformRef,
			Unassigned:  true,
		})
	}
	for _, f := range c.result.Failures {
		views = append(views, unassignedAsset{
			Asset:      f.AssetID,
			Unassigned: false,
			Error:      f.Err.Error(),
		})
	}
	return views
}
