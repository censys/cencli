package collections

import (
	"context"
	"fmt"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/collections"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
)

const getCmdName = "get"

// GetCommand implements `collections get <collection-id>`, showing one collection.
type GetCommand struct {
	*command.BaseCommand
	// services the command uses
	collectionsSvc collections.Service
	// flags the command uses
	flags getCommandFlags
	// state - populated by PreRun
	orgID        mo.Option[identifiers.OrganizationID]
	collectionID identifiers.CollectionID
	// result stores the collection for rendering
	result collections.GetResult
}

type getCommandFlags struct {
	orgID flags.OrgIDFlag
}

var _ command.Command = (*GetCommand)(nil)

func NewGetCommand(cmdContext *command.Context) *GetCommand {
	return &GetCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *GetCommand) Use() string {
	return fmt.Sprintf("%s <collection-id>", getCmdName)
}

func (c *GetCommand) Short() string {
	return "Retrieve a single collection"
}

func (c *GetCommand) Long() string {
	return `Retrieve a single collection by its UUID.

Use "censys collections list" to find a collection's ID.`
}

func (c *GetCommand) Examples() []string {
	return []string{
		"<collection-id> # Get a collection",
		"<collection-id> --output-format json # Output as JSON",
	}
}

func (c *GetCommand) Args() command.PositionalArgs {
	return command.ExactArgs(1)
}

func (c *GetCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *GetCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *GetCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	return nil
}

func (c *GetCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}
	c.collectionID, err = requireCollectionID(args[0])
	if err != nil {
		return err
	}
	return c.resolveCollectionsService()
}

func (c *GetCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With("orgID_set", c.orgID.IsPresent())

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Fetching collection...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.collectionsSvc.GetCollection(pctx, collections.GetParams{
				OrgID:        c.orgID,
				CollectionID: c.collectionID,
			})
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("get collection failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	return c.PrintData(c, c.result.Collection)
}

func (c *GetCommand) resolveCollectionsService() cenclierrors.CencliError {
	svc, err := c.CollectionsService()
	if err != nil {
		return err
	}
	c.collectionsSvc = svc
	return nil
}

// RenderShort renders a collection as a labeled detail view (TTY-aware).
func (c *GetCommand) RenderShort() cenclierrors.CencliError {
	return renderCollectionDetail("━━━ Collection ━━━", c.result.Collection)
}
