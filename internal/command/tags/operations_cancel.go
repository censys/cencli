package tags

import (
	"context"
	"fmt"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
)

const operationsCancelCmdName = "cancel"

// OperationsCancelCommand implements `tags operations cancel <tag> <op-id>`,
// stopping a running bulk job. It does not confirm: cancelling only stops
// further processing, and the destructive step was the job it is stopping.
type OperationsCancelCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags operationsCancelCommandFlags
	// state - populated by PreRun
	orgID       mo.Option[identifiers.OrganizationID]
	tagID       identifiers.TagID
	operationID string
	// result stores the cancelled operation for rendering
	result tags.CancelOperationResult
}

type operationsCancelCommandFlags struct {
	orgID flags.OrgIDFlag
}

var _ command.Command = (*OperationsCancelCommand)(nil)

func NewOperationsCancelCommand(cmdContext *command.Context) *OperationsCancelCommand {
	return &OperationsCancelCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *OperationsCancelCommand) Use() string {
	return fmt.Sprintf("%s <tag> <operation-id>", operationsCancelCmdName)
}

func (c *OperationsCancelCommand) Short() string {
	return "Cancel a running bulk tag operation"
}

func (c *OperationsCancelCommand) Long() string {
	return `Cancel a running bulk tag operation, identified by the tag it belongs to, given by name or UUID, and the operation's UUID.

Cancelling stops the job from processing any more assets; it does not undo the assignments it has already made or removed. An operation that has already finished cannot be cancelled.`
}

func (c *OperationsCancelCommand) Examples() []string {
	return []string{
		"my-tag <operation-id> # Stop a running bulk job",
	}
}

func (c *OperationsCancelCommand) Args() command.PositionalArgs {
	return command.ExactArgs(2)
}

func (c *OperationsCancelCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *OperationsCancelCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *OperationsCancelCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	return nil
}

func (c *OperationsCancelCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
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

	return c.resolveTagsService()
}

func (c *OperationsCancelCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_is_uuid", c.tagID.UID().IsPresent(),
	)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Cancelling operation...",
		func(pctx context.Context) cenclierrors.CencliError {
			var cancelErr cenclierrors.CencliError
			c.result, cancelErr = c.tagsSvc.CancelOperation(pctx, tags.CancelOperationParams{
				OrgID:       c.orgID,
				TagID:       c.tagID,
				OperationID: c.operationID,
			})
			return cancelErr
		},
	)
	if err != nil {
		logger.Debug("cancel operation failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	// A cancelled status is the point of this command, so it never drives the exit
	// code the way it does after a --wait: succeeding here means exiting 0.
	return c.PrintData(c, c.result.Operation)
}

func (c *OperationsCancelCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}

// RenderShort renders the cancelled operation as a labeled detail view, the same
// way `operations get` shows it.
func (c *OperationsCancelCommand) RenderShort() cenclierrors.CencliError {
	return renderOperationDetail(c.result.Operation)
}
