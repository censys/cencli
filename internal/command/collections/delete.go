package collections

import (
	"context"
	"fmt"
	"os"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/collections"
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

// DeleteCommand implements `collections delete <collection-id>`.
// It prompts for confirmation unless --yes is set; in a non-interactive
// terminal without --yes it refuses rather than deleting silently.
type DeleteCommand struct {
	*command.BaseCommand
	// services the command uses
	collectionsSvc collections.Service
	// flags the command uses
	flags deleteCommandFlags
	// state - populated by PreRun
	orgID        mo.Option[identifiers.OrganizationID]
	collectionID identifiers.CollectionID
	yes          bool
	// result stores the deletion outcome for rendering
	result collections.DeleteResult
	// seams - overridable in tests; defaulted in NewDeleteCommand
	confirm    func(ctx context.Context, message string) (bool, error)
	stdinIsTTY func() bool
}

type deleteCommandFlags struct {
	orgID flags.OrgIDFlag
	yes   flags.BoolFlag
}

// deletedCollection is the data-mode payload for a successful deletion; the
// endpoint returns no body, so only the identifier is echoed.
type deletedCollection struct {
	Collection string `json:"collection" yaml:"collection"`
	Deleted    bool   `json:"deleted" yaml:"deleted"`
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
	return fmt.Sprintf("%s <collection-id>", deleteCmdName)
}

func (c *DeleteCommand) Short() string {
	return "Delete a collection"
}

func (c *DeleteCommand) Long() string {
	return `Delete a collection by its UUID. This cannot be undone.

You are prompted to confirm before the collection is deleted. Use --yes to skip the prompt; in a non-interactive terminal --yes is required.`
}

func (c *DeleteCommand) Examples() []string {
	return []string{
		"<collection-id> # Delete a collection (prompts for confirmation)",
		"<collection-id> --yes # Delete without confirming",
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
	c.collectionID, err = requireCollectionID(args[0])
	if err != nil {
		return err
	}

	// Gate the confirmation before resolving the service so a non-interactive
	// invocation without --yes fails with a clear confirmation error rather than
	// deleting silently (and before any auth is required).
	if !c.yes && !c.stdinIsTTY() {
		return NewConfirmationRequiredError()
	}

	return c.resolveCollectionsService()
}

func (c *DeleteCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"yes", c.yes,
	)

	if !c.yes {
		message := fmt.Sprintf("Delete collection %q? This cannot be undone.", c.collectionID.String())
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
		"Deleting collection...",
		func(pctx context.Context) cenclierrors.CencliError {
			var deleteErr cenclierrors.CencliError
			c.result, deleteErr = c.collectionsSvc.DeleteCollection(pctx, collections.DeleteParams{
				OrgID:        c.orgID,
				CollectionID: c.collectionID,
			})
			return deleteErr
		},
	)
	if err != nil {
		logger.Debug("delete collection failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	return c.PrintData(c, deletedCollection{Collection: c.result.CollectionID, Deleted: true})
}

func (c *DeleteCommand) resolveCollectionsService() cenclierrors.CencliError {
	svc, err := c.CollectionsService()
	if err != nil {
		return err
	}
	c.collectionsSvc = svc
	return nil
}

// RenderShort renders a confirmation line for the deleted collection (TTY-aware).
func (c *DeleteCommand) RenderShort() cenclierrors.CencliError {
	line := styles.GlobalStyles.Signature.Render(fmt.Sprintf("Collection %q deleted.", c.result.CollectionID))
	formatter.Println(formatter.Stdout, line)
	return nil
}
