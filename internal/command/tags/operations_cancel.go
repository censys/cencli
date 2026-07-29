package tags

import (
	"context"
	"fmt"
	"os"

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

const operationsCancelCmdName = "cancel"

// OperationsCancelCommand implements `tags operations cancel <tag> <op-id>`,
// stopping a running bulk job. It prompts for confirmation unless --yes is set,
// since cancelling does not undo the work already committed.
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
	yes         bool
	// result stores the cancelled operation for rendering
	result tags.CancelOperationResult
	// seams - overridable in tests; defaulted in NewOperationsCancelCommand
	confirm    func(ctx context.Context, message string) (bool, error)
	stdinIsTTY func() bool
}

type operationsCancelCommandFlags struct {
	orgID flags.OrgIDFlag
	yes   flags.BoolFlag
}

var _ command.Command = (*OperationsCancelCommand)(nil)

func NewOperationsCancelCommand(cmdContext *command.Context) *OperationsCancelCommand {
	return &OperationsCancelCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
		confirm:     form.Confirm,
		stdinIsTTY:  func() bool { return term.IsTTY(os.Stdin) },
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

Cancelling stops the job from processing any more assets; it does not undo the assignments it has already made or removed. An operation that has already finished cannot be cancelled.

You are prompted to confirm before the cancellation is requested. Use --yes to skip the prompt; in a non-interactive terminal --yes is required.`
}

func (c *OperationsCancelCommand) Examples() []string {
	return []string{
		"my-tag <operation-id> # Cancel an operation (prompts for confirmation)",
		"my-tag <operation-id> --yes # Cancel without confirming",
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
	c.flags.yes = flags.NewBoolFlag(c.Flags(), "yes", "y", false, "skip the confirmation prompt")
	return nil
}

func (c *OperationsCancelCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}
	c.yes, err = c.flags.yes.Value()
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

	// Gate the confirmation before resolving the service so a non-interactive
	// invocation without --yes fails with a clear confirmation error rather than
	// cancelling silently (and before any auth is required).
	if !c.yes && !c.stdinIsTTY() {
		return NewConfirmationRequiredError()
	}

	return c.resolveTagsService()
}

func (c *OperationsCancelCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_is_uuid", c.tagID.UID().IsPresent(),
		"yes", c.yes,
	)

	if !c.yes {
		message := fmt.Sprintf(
			"Cancel operation %s? Assets it has already processed keep their change.", c.operationID)
		confirmed, err := confirmAction(cmd.Context(), c.confirm, message)
		if err != nil {
			return err
		}
		if !confirmed {
			formatter.Println(formatter.Stderr, "Cancellation aborted.")
			return nil
		}
	}

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
