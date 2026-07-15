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

const getCmdName = "get"

// GetCommand implements `tags get <tag>`, retrieving a single tag by name or UUID.
type GetCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags getCommandFlags
	// state - populated by PreRun
	orgID mo.Option[identifiers.OrganizationID]
	tagID identifiers.TagID
	// result stores the fetched tag for rendering
	result tags.GetResult
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
	return fmt.Sprintf("%s <tag>", getCmdName)
}

func (c *GetCommand) Short() string {
	return "Retrieve a single tag by name or ID"
}

func (c *GetCommand) Long() string {
	return `Retrieve a single tag by its name or UUID.

Tag names are unique within an organization, so a name and its ID can be used interchangeably.`
}

func (c *GetCommand) Examples() []string {
	return []string{
		"my-tag # Get a tag by name",
		"018f9c2e-1b3a-7c4d-9e5f-6a7b8c9d0e1f # Get a tag by UUID",
		"my-tag --output-format json # Output as JSON",
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
	c.tagID = identifiers.NewTagID(args[0])
	return c.resolveTagsService()
}

func (c *GetCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_is_uuid", c.tagID.UID().IsPresent(),
	)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Fetching tag...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.tagsSvc.GetTag(pctx, tags.GetParams{
				OrgID: c.orgID,
				TagID: c.tagID,
			})
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("get tag failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	return c.PrintData(c, c.result.Tag)
}

func (c *GetCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}

// RenderShort renders the fetched tag as a labeled detail view (TTY-aware).
func (c *GetCommand) RenderShort() cenclierrors.CencliError {
	return renderTagDetail("━━━ Tag ━━━", c.result.Tag)
}
