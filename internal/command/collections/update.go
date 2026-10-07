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

const updateCmdName = "update"

// UpdateCommand implements `collections update <collection-id>`.
type UpdateCommand struct {
	*command.BaseCommand
	// services the command uses
	collectionsSvc collections.Service
	// flags the command uses
	flags updateCommandFlags
	// state - populated by PreRun
	orgID        identifiers.OrganizationID
	collectionID identifiers.CollectionID
	name         mo.Option[string]
	query        mo.Option[string]
	description  mo.Option[string]
	// result stores the updated collection for rendering
	result collections.UpdateResult
}

type updateCommandFlags struct {
	orgID            flags.OrgIDFlag
	name             flags.StringFlag
	query            flags.StringFlag
	description      flags.StringFlag
	clearDescription flags.BoolFlag
}

var _ command.Command = (*UpdateCommand)(nil)

func NewUpdateCommand(cmdContext *command.Context) *UpdateCommand {
	return &UpdateCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *UpdateCommand) Use() string {
	return fmt.Sprintf("%s <collection-id>", updateCmdName)
}

func (c *UpdateCommand) Short() string {
	return "Update an existing collection"
}

func (c *UpdateCommand) Long() string {
	return `Update an existing collection by its UUID.

At least one mutation flag is required. Use --clear-description to remove a collection's description; it cannot be combined with --description.

The API replaces the whole collection on every update, so this command first reads the collection and keeps each value you do not change. A change that someone else makes between the read and the update is overwritten.`
}

func (c *UpdateCommand) Examples() []string {
	return []string{
		"<collection-id> --name renamed # Rename a collection",
		`<collection-id> --query "host.services.protocol=RDP" # Change the query`,
		`<collection-id> --description "Hosts to review" # Set a description`,
		"<collection-id> --clear-description # Remove the description",
	}
}

func (c *UpdateCommand) Args() command.PositionalArgs {
	return command.ExactArgs(1)
}

func (c *UpdateCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *UpdateCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *UpdateCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.name = flags.NewStringFlag(c.Flags(), false, "name", "", "", "a new name for the collection")
	c.flags.query = flags.NewStringFlag(c.Flags(), false, "query", "", "", "a new CenQL query for the collection")
	c.flags.description = flags.NewStringFlag(c.Flags(), false, "description", "", "", "a new description for the collection")
	c.flags.clearDescription = flags.NewBoolFlag(c.Flags(), "clear-description", "", false, "remove the collection's description")
	return nil
}

func (c *UpdateCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.collectionID, err = requireCollectionID(args[0])
	if err != nil {
		return err
	}

	name, err := c.flags.name.Value()
	if err != nil {
		return err
	}
	c.name = optionalNonEmpty(name)
	if cmd.Flags().Changed("name") && c.name.IsAbsent() {
		return NewBlankFlagError("name")
	}

	query, err := c.flags.query.Value()
	if err != nil {
		return err
	}
	c.query = optionalNonEmpty(query)
	if cmd.Flags().Changed("query") && c.query.IsAbsent() {
		return NewBlankFlagError("query")
	}

	description, err := c.flags.description.Value()
	if err != nil {
		return err
	}
	clearDescription, err := c.flags.clearDescription.Value()
	if err != nil {
		return err
	}
	if cmd.Flags().Changed("description") && clearDescription {
		return NewDescriptionConflictError()
	}
	if clearDescription {
		c.description = mo.Some("")
	} else {
		c.description = optionalNonEmpty(description)
	}

	if !c.name.IsPresent() && !c.query.IsPresent() && !c.description.IsPresent() {
		return NewNothingToUpdateError()
	}

	flagOrgID, err := c.flags.orgID.Value()
	if err != nil {
		return err
	}
	// Every collections endpoint requires an organization. Resolve it after the
	// input checks, as scan does, so an input error is reported first; this
	// still fails before any request when no organization is configured.
	c.orgID, err = c.ResolveRequiredOrgID(cmd, flagOrgID)
	if err != nil {
		return err
	}

	return c.resolveCollectionsService()
}

func (c *UpdateCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"name_set", c.name.IsPresent(),
		"query_set", c.query.IsPresent(),
		"description_set", c.description.IsPresent(),
	)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Updating collection...",
		func(pctx context.Context) cenclierrors.CencliError {
			var updateErr cenclierrors.CencliError
			c.result, updateErr = c.collectionsSvc.UpdateCollection(pctx, collections.UpdateParams{
				OrgID:        mo.Some(c.orgID),
				CollectionID: c.collectionID,
				Name:         c.name,
				Query:        c.query,
				Description:  c.description,
			})
			return updateErr
		},
	)
	if err != nil {
		logger.Debug("update collection failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	return c.PrintData(c, c.result.Collection)
}

func (c *UpdateCommand) resolveCollectionsService() cenclierrors.CencliError {
	svc, err := c.CollectionsService()
	if err != nil {
		return err
	}
	c.collectionsSvc = svc
	return nil
}

// RenderShort renders the updated collection as a labeled detail view (TTY-aware).
func (c *UpdateCommand) RenderShort() cenclierrors.CencliError {
	return renderCollectionDetail("━━━ Collection Updated ━━━", c.result.Collection)
}
