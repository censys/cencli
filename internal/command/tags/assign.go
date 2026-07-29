package tags

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
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
	assignCmdName = "assign"

	// assignIndexLagNote warns that a fresh assignment is not immediately
	// searchable. Printed by both input modes, since both mutate assignments.
	assignIndexLagNote = "Note: newly assigned tags may take a few minutes to appear in `tags:` search results."
)

// AssignCommand implements `tags assign <tag> [asset...]`, linking a tag either
// to explicit assets (given positionally or via --input-file) or, with --query,
// to every asset matching a CenQL query. The query form submits an asynchronous
// bulk job and reports the operation tracking it.
type AssignCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags assignCommandFlags
	// state - populated by PreRun
	orgID    mo.Option[identifiers.OrganizationID]
	tagID    identifiers.TagID
	assetIDs []string
	// bulk state - only meaningful when bulk is true
	bulk      bool
	query     string
	maxAssets mo.Option[int64]
	wait      bool
	timeout   mo.Option[time.Duration]
	yes       bool
	// result stores the explicit-mode assignment outcome for rendering
	result tags.AssignResult
	// operation stores the bulk job for rendering: the submitted operation, then
	// the finished one once --wait has polled it
	operation tags.TagOperation
	// seams - overridable in tests; defaulted in NewAssignCommand
	confirm    func(ctx context.Context, message string) (bool, error)
	stdinIsTTY func() bool
}

type assignCommandFlags struct {
	orgID     flags.OrgIDFlag
	inputFile flags.FileFlag
	query     flags.StringFlag
	maxAssets flags.IntegerFlag
	wait      flags.BoolFlag
	timeout   flags.HumanDurationFlag
	yes       flags.BoolFlag
}

// assignedAsset is the data-mode payload for a single assignment outcome.
type assignedAsset struct {
	Asset        string `json:"asset" yaml:"asset"`
	AssignmentID string `json:"assignment_id,omitempty" yaml:"assignment_id,omitempty"`
	AssetType    string `json:"asset_type,omitempty" yaml:"asset_type,omitempty"`
	PlatformRef  string `json:"platform_ref,omitempty" yaml:"platform_ref,omitempty"`
	Assigned     bool   `json:"assigned" yaml:"assigned"`
	Error        string `json:"error,omitempty" yaml:"error,omitempty"`
}

var _ command.Command = (*AssignCommand)(nil)

func NewAssignCommand(cmdContext *command.Context) *AssignCommand {
	return &AssignCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
		confirm:     form.Confirm,
		stdinIsTTY:  func() bool { return term.IsTTY(os.Stdin) },
	}
}

func (c *AssignCommand) Use() string {
	return fmt.Sprintf("%s <tag> [asset...]", assignCmdName)
}

func (c *AssignCommand) Short() string {
	return "Assign a tag to one or more assets"
}

func (c *AssignCommand) Long() string {
	return `Assign a tag, by its name or UUID, to one or more assets (host IPs, certificate SHA-256 fingerprints, or web property hostname:port).

Assets can be passed as positional arguments or read from a file (or STDIN) with --input-file. Assets of different types can be mixed in a single call. Each asset is assigned independently: if one fails the rest still proceed, and the per-asset outcomes are reported.

Use --query instead to assign the tag to every asset matching a CenQL query. That starts an asynchronous bulk job and reports the operation tracking it; the two input modes cannot be combined. Bulk assignment always asks for confirmation unless --yes is set.`
}

func (c *AssignCommand) Examples() []string {
	return []string{
		"<tag> <ip-address>",
		"<tag> <ip-address> <ip-address>",
		"<tag> <hostname:port>",
		"<tag> <sha256-fingerprint>",
		"<tag> <ip-address> <hostname:port>  # asset types can be mixed",
		"<tag> --input-file <file>",
		"<tag> --input-file -  # read assets from STDIN",
		"<tag> --query 'host.services.port: 22'  # assign every matching asset",
		"<tag> --query 'host.services.port: 22' --max-assets 1000",
		"<tag> --query 'host.services.port: 22' --wait  # poll until the job finishes",
	}
}

func (c *AssignCommand) Args() command.PositionalArgs {
	// At least the tag; assets may instead come from --input-file or --query.
	return command.MinimumNArgs(1)
}

func (c *AssignCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *AssignCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *AssignCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.inputFile = flags.NewFileFlag(c.Flags(), false, "input-file", "i", "file to read the assets from (or - for STDIN). Overrides positional asset arguments.")
	c.flags.query = flags.NewStringFlag(c.Flags(), false, "query", "", "",
		"CenQL query selecting the assets to tag. Starts a bulk job instead of assigning explicit assets.")
	c.flags.maxAssets = flags.NewIntegerFlag(
		c.Flags(),
		false,
		"max-assets",
		"",
		mo.None[int64](),
		"cap the number of assets a bulk job tags (requires --query). The effective cap is the smaller of this and your plan's tag asset limit.",
		mo.Some[int64](0),
		mo.None[int64](),
	)
	c.flags.wait = flags.NewBoolFlag(c.Flags(), "wait", "w", false,
		"poll the bulk job until it reaches a final status (requires --query)")
	c.flags.timeout = flags.NewHumanDurationFlag(c.Flags(), false, "timeout", "",
		mo.Some(defaultWaitTimeout), "how long to wait before giving up (requires --wait) - use 0 for no limit")
	c.flags.yes = flags.NewBoolFlag(c.Flags(), "yes", "y", false, "skip the confirmation prompt")
	return nil
}

func (c *AssignCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}

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
		// Gate the confirmation before resolving the service so a non-interactive
		// invocation without --yes fails with a clear confirmation error rather
		// than submitting a large job silently (and before any auth is required).
		return NewConfirmationRequiredError()
	}

	return c.resolveTagsService()
}

// parseModeFlags decides between explicit and bulk assignment and rejects the
// combinations that cannot mean anything. Bulk is only ever chosen by --query;
// it is never inferred from missing asset arguments.
func (c *AssignCommand) parseModeFlags(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	query, err := c.flags.query.Value()
	if err != nil {
		return err
	}
	c.bulk = cmd.Flags().Changed("query")
	c.query = strings.TrimSpace(query)

	if c.bulk {
		// A blank query would match nothing, so reject it before it can reach a
		// confirmation prompt or spend an operation.
		if c.query == "" {
			return tags.NewEmptyQueryError()
		}
		if len(args) > 1 || c.flags.inputFile.IsSet() {
			return NewAssignModeConflictError()
		}
	}

	// Flags that only steer a bulk job would silently do nothing in explicit mode.
	for _, name := range []string{"max-assets", "wait", "timeout"} {
		if !c.bulk && cmd.Flags().Changed(name) {
			return NewFlagRequiresQueryError(name)
		}
	}

	c.maxAssets, err = c.flags.maxAssets.Value()
	if err != nil {
		return err
	}
	c.yes, err = c.flags.yes.Value()
	if err != nil {
		return err
	}

	c.wait, c.timeout, err = parseWaitFlags(cmd, c.flags.wait, c.flags.timeout)
	return err
}

func (c *AssignCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_is_uuid", c.tagID.UID().IsPresent(),
		"bulk", c.bulk,
	)

	if c.bulk {
		return c.runBulk(cmd, logger.With("wait", c.wait, "max_assets_set", c.maxAssets.IsPresent()))
	}
	return c.runExplicit(cmd, logger.With("count", len(c.assetIDs)))
}

// runExplicit assigns the tag to each given asset, one request per asset.
func (c *AssignCommand) runExplicit(cmd *cobra.Command, logger *slog.Logger) cenclierrors.CencliError {
	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Assigning tag...",
		func(pctx context.Context) cenclierrors.CencliError {
			var assignErr cenclierrors.CencliError
			c.result, assignErr = c.tagsSvc.Assign(pctx, tags.AssignParams{
				OrgID:    c.orgID,
				TagID:    c.tagID,
				AssetIDs: c.assetIDs,
			})
			return assignErr
		},
	)
	if err != nil {
		logger.Debug("assign tag failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	if renderErr := c.PrintData(c, c.assignmentViews()); renderErr != nil {
		return renderErr
	}

	if len(c.result.Assignments) > 0 {
		printNote(c.Config().Quiet, assignIndexLagNote)
	}

	if c.result.PartialError != nil {
		formatter.PrintError(c.result.PartialError, cmd)
	}

	return nil
}

// runBulk submits a query-driven bulk job and reports the operation tracking it,
// optionally polling that operation until it finishes.
func (c *AssignCommand) runBulk(cmd *cobra.Command, logger *slog.Logger) cenclierrors.CencliError {
	if !c.yes {
		confirmed, err := confirmAction(cmd.Context(), c.confirm, c.confirmMessage())
		if err != nil {
			return err
		}
		if !confirmed {
			formatter.Println(formatter.Stderr, "Assignment aborted.")
			return nil
		}
	}

	var submitted tags.BulkAssignResult
	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Submitting bulk assignment...",
		func(pctx context.Context) cenclierrors.CencliError {
			var submitErr cenclierrors.CencliError
			submitted, submitErr = c.tagsSvc.BulkAssign(pctx, tags.BulkAssignParams{
				OrgID:     c.orgID,
				TagID:     c.tagID,
				Query:     c.query,
				MaxAssets: c.maxAssets,
			})
			return submitErr
		},
	)
	if err != nil {
		logger.Debug("submit bulk assignment failed", "error", err)
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
		printNote(quiet, assignIndexLagNote)
		return nil
	}

	if statusErr := reportOperationTerminalStatus(c.operation); statusErr != nil {
		return statusErr
	}
	printNote(quiet, assignIndexLagNote)
	return nil
}

// waitForSubmitted polls the job just submitted, replacing the operation being
// rendered with the finished one.
func (c *AssignCommand) waitForSubmitted(ctx context.Context, logger *slog.Logger) cenclierrors.CencliError {
	operation, err := followSubmittedOperation(ctx, c.BaseCommand, logger, c.tagsSvc, tags.WaitParams{
		OrgID:       c.orgID,
		TagID:       c.tagID,
		OperationID: c.operation.ID,
		Timeout:     c.timeout,
	})
	if err != nil {
		logger.Debug("wait for bulk assignment failed", "error", err)
		return err
	}

	c.operation = operation
	return nil
}

// confirmMessage spells out what a bulk assignment is about to do, including the
// cap that will actually apply.
func (c *AssignCommand) confirmMessage() string {
	limit := "your plan's tag asset limit"
	if c.maxAssets.IsPresent() && c.maxAssets.MustGet() > 0 {
		limit = fmt.Sprintf("at most %d asset(s)", c.maxAssets.MustGet())
	}
	return fmt.Sprintf("Assign tag %q to every asset matching %q (%s)?",
		c.tagID.String(), c.query, limit)
}

func (c *AssignCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}

// assignmentViews builds the render payload: successes first, then failures.
func (c *AssignCommand) assignmentViews() []assignedAsset {
	views := make([]assignedAsset, 0, len(c.result.Assignments)+len(c.result.Failures))
	for _, a := range c.result.Assignments {
		views = append(views, assignedAsset{
			Asset:        a.AssetID,
			AssignmentID: a.ID,
			AssetType:    a.AssetType,
			PlatformRef:  a.PlatformRef,
			Assigned:     true,
		})
	}
	for _, f := range c.result.Failures {
		views = append(views, assignedAsset{
			Asset:    f.AssetID,
			Assigned: false,
			Error:    f.Err.Error(),
		})
	}
	return views
}
