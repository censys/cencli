package tags

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
)

const operationsGetCmdName = "get"

// OperationsGetCommand implements `tags operations get <tag> <op-id>`,
// retrieving one bulk job and optionally polling it until it finishes.
type OperationsGetCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags operationsGetCommandFlags
	// state - populated by PreRun
	orgID       mo.Option[identifiers.OrganizationID]
	tagID       identifiers.TagID
	operationID string
	wait        bool
	timeout     mo.Option[time.Duration]
	// result stores the operation for rendering
	result tags.GetOperationResult
}

type operationsGetCommandFlags struct {
	orgID   flags.OrgIDFlag
	wait    flags.BoolFlag
	timeout flags.HumanDurationFlag
}

var _ command.Command = (*OperationsGetCommand)(nil)

func NewOperationsGetCommand(cmdContext *command.Context) *OperationsGetCommand {
	return &OperationsGetCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *OperationsGetCommand) Use() string {
	return fmt.Sprintf("%s <tag> <operation-id>", operationsGetCmdName)
}

func (c *OperationsGetCommand) Short() string {
	return "Retrieve a single bulk tag operation"
}

func (c *OperationsGetCommand) Long() string {
	return `Retrieve a single bulk tag operation by the tag it belongs to, given by name or UUID, and the operation's UUID.

Use --wait to poll until the operation finishes. Waiting exits non-zero if the operation ends up failed or cancelled; without --wait the command simply reports the current status and exits 0.`
}

func (c *OperationsGetCommand) Examples() []string {
	return []string{
		"my-tag <operation-id> # Show an operation's current status",
		"my-tag <operation-id> --wait # Poll until the operation finishes",
		"my-tag <operation-id> --wait --timeout 5m # Give up waiting after 5 minutes",
	}
}

func (c *OperationsGetCommand) Args() command.PositionalArgs {
	return command.ExactArgs(2)
}

func (c *OperationsGetCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *OperationsGetCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *OperationsGetCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.wait = flags.NewBoolFlag(c.Flags(), "wait", "w", false, "poll until the operation reaches a final status")
	c.flags.timeout = flags.NewHumanDurationFlag(c.Flags(), false, "timeout", "",
		mo.Some(defaultWaitTimeout), "how long to wait before giving up (requires --wait) - use 0 for no limit")
	return nil
}

func (c *OperationsGetCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}

	c.tagID, err = requireTagID(args[0])
	if err != nil {
		return err
	}
	c.operationID, err = requireOperationID(args[1])
	if err != nil {
		return err
	}

	c.wait, c.timeout, err = parseWaitFlags(cmd, c.flags.wait, c.flags.timeout)
	if err != nil {
		return err
	}

	return c.resolveTagsService()
}

func (c *OperationsGetCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_is_uuid", c.tagID.UID().IsPresent(),
		"wait", c.wait,
		"timeout_set", c.timeout.IsPresent(),
	)

	if err := c.fetch(cmd.Context(), logger); err != nil {
		if c.wait && cenclierrors.IsInterrupted(err) {
			printOperationStillRunningNote(c.Config().Quiet, c.tagID.String(), c.operationID)
		}
		logger.Debug("get operation failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	if renderErr := c.PrintData(c, c.result.Operation); renderErr != nil {
		return renderErr
	}

	// Only a wait reports the outcome through the exit code: a plain read of a
	// failed operation is itself a successful read.
	if !c.wait {
		return nil
	}
	return reportOperationTerminalStatus(c.result.Operation)
}

// fetch retrieves the operation once, or polls it when --wait is set.
func (c *OperationsGetCommand) fetch(ctx context.Context, logger *slog.Logger) cenclierrors.CencliError {
	if !c.wait {
		return c.WithProgress(ctx, logger, "Fetching operation...",
			func(pctx context.Context) cenclierrors.CencliError {
				var getErr cenclierrors.CencliError
				c.result, getErr = c.tagsSvc.GetOperation(pctx, tags.GetOperationParams{
					OrgID:       c.orgID,
					TagID:       c.tagID,
					OperationID: c.operationID,
				})
				return getErr
			})
	}

	result, err := waitForOperation(ctx, c.BaseCommand, logger, c.tagsSvc, tags.WaitParams{
		OrgID:       c.orgID,
		TagID:       c.tagID,
		OperationID: c.operationID,
		Timeout:     c.timeout,
	})
	c.result = result
	return err
}

func (c *OperationsGetCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}
