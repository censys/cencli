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
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/term"
	"github.com/censys/cencli/internal/pkg/ui/form"
)

const deleteCmdName = "delete"

// DeleteCommand implements `tags delete <tag>`, removing a tag by name or UUID.
// It is the first destructive command and prompts for confirmation unless --yes
// is set; in a non-interactive terminal without --yes it refuses rather than
// deleting silently.
type DeleteCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags deleteCommandFlags
	// state - populated by PreRun
	orgID mo.Option[identifiers.OrganizationID]
	tagID identifiers.TagID
	yes   bool
	// result stores the deletion outcome for rendering
	result tags.DeleteResult
	// seams - overridable in tests; defaulted in NewDeleteCommand
	confirm    func(ctx context.Context, message string) (bool, error)
	stdinIsTTY func() bool
}

type deleteCommandFlags struct {
	orgID flags.OrgIDFlag
	yes   flags.BoolFlag
}

// deletedTag is the data-mode payload for a successful deletion; there is no tag
// body returned by the endpoint, so only the identifier is echoed.
type deletedTag struct {
	Tag     string `json:"tag" yaml:"tag"`
	Deleted bool   `json:"deleted" yaml:"deleted"`
}

var _ command.Command = (*DeleteCommand)(nil)

func NewDeleteCommand(cmdContext *command.Context) *DeleteCommand {
	return &DeleteCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
		confirm:     form.Confirm,
		stdinIsTTY:  func() bool { return term.IsTTY(os.Stdin) },
	}
}

func (c *DeleteCommand) Use() string {
	return fmt.Sprintf("%s <tag>", deleteCmdName)
}

func (c *DeleteCommand) Short() string {
	return "Delete a tag"
}

func (c *DeleteCommand) Long() string {
	return `Delete a tag by its name or UUID. This cannot be undone.

You are prompted to confirm before the tag is deleted. Use --yes to skip the prompt; in a non-interactive terminal --yes is required.`
}

func (c *DeleteCommand) Examples() []string {
	return []string{
		"my-tag # Delete a tag by name (prompts for confirmation)",
		"my-tag --yes # Delete without confirming",
	}
}

func (c *DeleteCommand) Args() command.PositionalArgs {
	return command.ExactArgs(1)
}

func (c *DeleteCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *DeleteCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *DeleteCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.yes = flags.NewBoolFlag(c.Flags(), "yes", "y", false, "skip the confirmation prompt")
	return nil
}

func (c *DeleteCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
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

	// Gate the confirmation before resolving the service so a non-interactive
	// invocation without --yes fails with a clear confirmation error rather than
	// deleting silently (and before any auth is required).
	if !c.yes && !c.stdinIsTTY() {
		return NewConfirmationRequiredError()
	}

	return c.resolveTagsService()
}

func (c *DeleteCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_is_uuid", c.tagID.UID().IsPresent(),
		"yes", c.yes,
	)

	if !c.yes {
		message := fmt.Sprintf("Delete tag %q? This cannot be undone.", c.tagID.String())
		confirmed, err := confirmAction(cmd.Context(), c.confirm, message)
		if err != nil {
			return err
		}
		if !confirmed {
			formatter.Println(formatter.Stderr, "Deletion aborted.")
			return nil
		}
	}

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Deleting tag...",
		func(pctx context.Context) cenclierrors.CencliError {
			var deleteErr cenclierrors.CencliError
			c.result, deleteErr = c.tagsSvc.DeleteTag(pctx, tags.DeleteParams{
				OrgID: c.orgID,
				TagID: c.tagID,
			})
			return deleteErr
		},
	)
	if err != nil {
		logger.Debug("delete tag failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	return c.PrintData(c, deletedTag{Tag: c.result.TagID, Deleted: true})
}

func (c *DeleteCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}

// RenderShort renders a confirmation line for the deleted tag (TTY-aware).
func (c *DeleteCommand) RenderShort() cenclierrors.CencliError {
	line := styles.GlobalStyles.Signature.Render(fmt.Sprintf("Tag %q deleted.", c.result.TagID))
	formatter.Println(formatter.Stdout, line)
	return nil
}
