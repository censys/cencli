package tags

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/term"
	"github.com/censys/cencli/internal/pkg/ui/form"
)

const (
	unassignCmdName = "unassign"

	// unassignIndexLagNote warns that a removed assignment is not immediately
	// gone from search. Printed by both input modes, since both mutate assignments.
	unassignIndexLagNote = "Note: unassigned tags may take a few minutes to disappear from `tags:` search results."
)

// UnassignCommand implements `tags unassign <tag> [asset...]`, removing a tag
// either from explicit assets (given positionally or via --input-file) or, with
// --all or a time filter, from the assignments matching that filter. The filtered
// form submits an asynchronous bulk job and reports the operation tracking it.
type UnassignCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags unassignCommandFlags
	// state - populated by PreRun
	orgID    mo.Option[identifiers.OrganizationID]
	tagID    identifiers.TagID
	assetIDs []string
	yes      bool
	// bulk state - only meaningful when bulk is true
	bulk          bool
	all           bool
	createdBefore mo.Option[time.Time]
	createdAfter  mo.Option[time.Time]
	wait          bool
	timeout       mo.Option[time.Duration]
	// result stores the explicit-mode unassignment outcome for rendering
	result tags.UnassignResult
	// operation stores the bulk job for rendering: the submitted operation, then
	// the finished one once --wait has polled it
	operation tags.TagOperation
	// seams - overridable in tests; defaulted in NewUnassignCommand
	confirm    func(ctx context.Context, message string) (bool, error)
	stdinIsTTY func() bool
}

type unassignCommandFlags struct {
	orgID         flags.OrgIDFlag
	inputFile     flags.FileFlag
	all           flags.BoolFlag
	createdBefore flags.TimestampFlag
	createdAfter  flags.TimestampFlag
	wait          flags.BoolFlag
	timeout       flags.HumanDurationFlag
	yes           flags.BoolFlag
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

Use --all instead to remove every one of the tag's assignments, or --created-before/--created-after to remove only those created in a time window. Either form starts an asynchronous bulk job and reports the operation tracking it; it cannot be combined with explicit assets, and --all cannot be narrowed by a time filter. Bulk unassignment always asks for confirmation unless --yes is set.`
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
		"<tag> --all  # remove every one of the tag's assignments",
		"<tag> --created-before 2026-01-01T00:00:00Z  # only assignments made before then",
		"<tag> --all --wait  # poll until the job finishes",
	}
}

func (c *UnassignCommand) Args() command.PositionalArgs {
	// At least the tag; assets may instead come from --input-file, or the bulk
	// filters may select them.
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
	c.flags.all = flags.NewBoolFlag(c.Flags(), "all", "", false,
		"remove every one of the tag's assignments. Starts a bulk job instead of unassigning explicit assets.")
	c.flags.createdBefore = flags.NewTimestampFlag(c.Flags(), false, "created-before", "", mo.None[time.Time](),
		"only unassign assignments created before this time. Starts a bulk job.")
	c.flags.createdAfter = flags.NewTimestampFlag(c.Flags(), false, "created-after", "", mo.None[time.Time](),
		"only unassign assignments created after this time. Starts a bulk job.")
	c.flags.wait = flags.NewBoolFlag(c.Flags(), "wait", "w", false,
		"poll the bulk job until it reaches a final status (requires --all or a time filter)")
	c.flags.timeout = flags.NewHumanDurationFlag(c.Flags(), false, "timeout", "",
		mo.Some(defaultWaitTimeout), "how long to wait before giving up (requires --wait) - use 0 for no limit")
	c.flags.yes = flags.NewBoolFlag(c.Flags(), "yes", "y", false,
		"skip the confirmation prompt (requires --all or a time filter)")
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

	if err := c.parseModeFlags(cmd, args); err != nil {
		return err
	}

	if !c.bulk {
		c.assetIDs, err = gatherAssetIDs(cmd, c.flags.inputFile, args)
		if err != nil {
			return err
		}
	} else if !c.yes && !c.stdinIsTTY() {
		// Only a bulk removal confirms, and the gate sits before the service is
		// resolved so a non-interactive run without --yes fails cleanly rather
		// than submitting the job (and before any auth is required).
		return NewConfirmationRequiredError()
	}

	return c.resolveTagsService()
}

// parseModeFlags decides between explicit and bulk unassignment and rejects the
// combinations that cannot mean anything. Bulk is only ever chosen by --all or a
// time filter; it is never inferred from missing asset arguments.
func (c *UnassignCommand) parseModeFlags(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.all, err = c.flags.all.Value()
	if err != nil {
		return err
	}
	c.createdBefore, err = c.flags.createdBefore.Value(c.Config().DefaultTZ)
	if err != nil {
		return err
	}
	c.createdAfter, err = c.flags.createdAfter.Value(c.Config().DefaultTZ)
	if err != nil {
		return err
	}

	timeFiltered := c.createdBefore.IsPresent() || c.createdAfter.IsPresent()
	c.bulk = c.all || timeFiltered

	if c.bulk {
		// --all already means every assignment, so narrowing it is contradictory.
		if c.all && timeFiltered {
			return NewAllWithTimeFilterError()
		}
		if len(args) > 1 || c.flags.inputFile.IsSet() {
			return NewUnassignModeConflictError()
		}
		// Reject an inverted window here as well as in the service, so it fails
		// before credentials are needed - the same reason a blank --query is
		// caught in assign's PreRun.
		if err := tags.ValidateTimeWindow(c.createdBefore, c.createdAfter); err != nil {
			return err
		}
	}

	// Flags that only steer a bulk job would silently do nothing in explicit mode.
	// --yes is one of them: explicit unassignment never prompts.
	for _, name := range []string{"wait", "timeout", "yes"} {
		if !c.bulk && cmd.Flags().Changed(name) {
			return NewFlagRequiresAllError(name)
		}
	}

	c.wait, c.timeout, err = parseWaitFlags(cmd, c.flags.wait, c.flags.timeout)
	return err
}

func (c *UnassignCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_is_uuid", c.tagID.UID().IsPresent(),
		"yes", c.yes,
		"bulk", c.bulk,
	)

	if c.bulk {
		return c.runBulk(cmd, logger.With("wait", c.wait, "all", c.all))
	}
	return c.runExplicit(cmd, logger.With("count", len(c.assetIDs)))
}

// runExplicit removes the tag from each given asset, one lookup+delete per asset.
// It does not confirm: the assets were named on the command line, so there is
// nothing the caller could learn from a prompt that they did not already type.
func (c *UnassignCommand) runExplicit(cmd *cobra.Command, logger *slog.Logger) cenclierrors.CencliError {
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
		printNote(c.Config().Quiet, unassignIndexLagNote)
	}

	if c.result.PartialError != nil {
		formatter.PrintError(c.result.PartialError, cmd)
	}

	return nil
}

// runBulk submits a filter-driven bulk removal and reports the operation tracking
// it, optionally polling that operation until it finishes.
func (c *UnassignCommand) runBulk(cmd *cobra.Command, logger *slog.Logger) cenclierrors.CencliError {
	if !c.yes {
		confirmed, err := confirmAction(cmd.Context(), c.confirm, c.confirmMessage())
		if err != nil {
			return err
		}
		if !confirmed {
			formatter.Println(formatter.Stderr, "Unassign aborted.")
			return nil
		}
	}

	var submitted tags.BulkUnassignResult
	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Submitting bulk unassignment...",
		func(pctx context.Context) cenclierrors.CencliError {
			var submitErr cenclierrors.CencliError
			submitted, submitErr = c.tagsSvc.BulkUnassign(pctx, tags.BulkUnassignParams{
				OrgID:         c.orgID,
				TagID:         c.tagID,
				CreatedBefore: c.createdBefore,
				CreatedAfter:  c.createdAfter,
			})
			return submitErr
		},
	)
	if err != nil {
		logger.Debug("submit bulk unassignment failed", "error", err)
		return err
	}

	c.operation = submitted.Operation
	c.PrintAppResponseMeta(submitted.Meta)

	// The job now exists server-side whatever happens next, so any exit that
	// leaves it unfinished says how to pick it back up.
	if c.wait {
		if waitErr := c.waitForSubmitted(cmd.Context(), logger); waitErr != nil {
			return waitErr
		}
	}

	if renderErr := c.PrintData(c, c.operation); renderErr != nil {
		return renderErr
	}

	quiet := c.Config().Quiet
	if !c.wait {
		printOperationTrackHint(quiet, c.tagID.String(), c.operation.ID)
		printNote(quiet, unassignIndexLagNote)
		return nil
	}

	if statusErr := reportOperationTerminalStatus(c.operation); statusErr != nil {
		return statusErr
	}
	printNote(quiet, unassignIndexLagNote)
	return nil
}

// waitForSubmitted polls the job just submitted, replacing the operation being
// rendered with the finished one.
func (c *UnassignCommand) waitForSubmitted(ctx context.Context, logger *slog.Logger) cenclierrors.CencliError {
	operation, err := followSubmittedOperation(ctx, c.BaseCommand, logger, c.tagsSvc, tags.WaitParams{
		OrgID:       c.orgID,
		TagID:       c.tagID,
		OperationID: c.operation.ID,
		Timeout:     c.timeout,
	})
	if err != nil {
		logger.Debug("wait for bulk unassignment failed", "error", err)
		return err
	}

	c.operation = operation
	return nil
}

// confirmMessage spells out the scope of a bulk removal, since the difference
// between wiping a tag and trimming a time window is not otherwise visible.
func (c *UnassignCommand) confirmMessage() string {
	scope := "ALL assigned assets"
	switch {
	case c.createdBefore.IsPresent() && c.createdAfter.IsPresent():
		scope = fmt.Sprintf("assignments created between %s and %s",
			c.createdAfter.MustGet().Format(detailTimeLayout),
			c.createdBefore.MustGet().Format(detailTimeLayout))
	case c.createdBefore.IsPresent():
		scope = fmt.Sprintf("assignments created before %s",
			c.createdBefore.MustGet().Format(detailTimeLayout))
	case c.createdAfter.IsPresent():
		scope = fmt.Sprintf("assignments created after %s",
			c.createdAfter.MustGet().Format(detailTimeLayout))
	}
	return fmt.Sprintf("Unassign tag %q from %s?", c.tagID.String(), scope)
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
