package collections

import (
	"context"
	"fmt"
	"strings"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/collections"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
)

const createCmdName = "create"

// CreateCommand implements `collections create <name> --query <cenql>`.
type CreateCommand struct {
	*command.BaseCommand
	// services the command uses
	collectionsSvc collections.Service
	// flags the command uses
	flags createCommandFlags
	// state - populated by PreRun
	orgID       identifiers.OrganizationID
	name        string
	query       string
	description mo.Option[string]
	// result stores the created collection for rendering
	result collections.CreateResult
}

type createCommandFlags struct {
	orgID       flags.OrgIDFlag
	query       flags.StringFlag
	description flags.StringFlag
}

var _ command.Command = (*CreateCommand)(nil)

func NewCreateCommand(cmdContext *command.Context) *CreateCommand {
	return &CreateCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *CreateCommand) Use() string {
	return fmt.Sprintf("%s <name>", createCmdName)
}

func (c *CreateCommand) Short() string {
	return "Create a new collection"
}

func (c *CreateCommand) Long() string {
	return `Create a new collection from a CenQL query.

Censys populates the collection in the background. A new collection can report the "populating" status until the first build finishes.`
}

func (c *CreateCommand) Examples() []string {
	return []string{
		`ssh-hosts --query "host.services.protocol=SSH" # Create a collection`,
		`ssh-hosts --query "host.services.protocol=SSH" --description "All SSH hosts" # Create a collection with a description`,
	}
}

func (c *CreateCommand) Args() command.PositionalArgs {
	return command.ExactArgs(1)
}

func (c *CreateCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *CreateCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *CreateCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.query = flags.NewStringFlag(c.Flags(), true, "query", "", "", "CenQL query that selects the collection's assets")
	c.flags.description = flags.NewStringFlag(c.Flags(), false, "description", "", "", "a human-readable description of the collection")
	return nil
}

func (c *CreateCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	c.name = strings.TrimSpace(args[0])

	query, err := c.flags.query.Value()
	if err != nil {
		return err
	}
	c.query = strings.TrimSpace(query)

	// Reject blank input here as well as in the service, so it is reported
	// before the organization and auth checks below.
	if c.name == "" {
		return collections.NewInvalidCollectionNameError()
	}
	if c.query == "" {
		return collections.NewEmptyQueryError()
	}

	description, err := c.flags.description.Value()
	if err != nil {
		return err
	}
	c.description = optionalNonEmpty(description)

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

func (c *CreateCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"description_set", c.description.IsPresent(),
	)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Creating collection...",
		func(pctx context.Context) cenclierrors.CencliError {
			var createErr cenclierrors.CencliError
			c.result, createErr = c.collectionsSvc.CreateCollection(pctx, collections.CreateParams{
				OrgID:       mo.Some(c.orgID),
				Name:        c.name,
				Query:       c.query,
				Description: c.description,
			})
			return createErr
		},
	)
	if err != nil {
		logger.Debug("create collection failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	if renderErr := c.PrintData(c, c.result.Collection); renderErr != nil {
		return renderErr
	}

	if c.result.Collection.ID != "" {
		printNote(c.Config().Quiet, fmt.Sprintf(
			"Search this collection: censys search --collection-id %s %s", c.result.Collection.ID, shellQuote(c.query)))
	}

	return nil
}

func (c *CreateCommand) resolveCollectionsService() cenclierrors.CencliError {
	svc, err := c.CollectionsService()
	if err != nil {
		return err
	}
	c.collectionsSvc = svc
	return nil
}

// RenderShort renders the created collection as a labeled detail view (TTY-aware).
func (c *CreateCommand) RenderShort() cenclierrors.CencliError {
	return renderCollectionDetail("━━━ Collection Created ━━━", c.result.Collection)
}
