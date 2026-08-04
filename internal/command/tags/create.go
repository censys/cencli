package tags

import (
	"context"
	"fmt"
	"strings"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
)

const createCmdName = "create"

// CreateCommand implements `tags create <name>`, creating a new tag.
type CreateCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags createCommandFlags
	// state - populated by PreRun
	orgID       mo.Option[identifiers.OrganizationID]
	name        string
	privacy     string
	description mo.Option[string]
	// result stores the created tag for rendering
	result tags.CreateResult
}

type createCommandFlags struct {
	orgID       flags.OrgIDFlag
	privacy     flags.StringFlag
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
	return "Create a new tag"
}

func (c *CreateCommand) Long() string {
	return `Create a new tag with the given name.

Tag names must be unique within an organization. New tags are private by default; use --privacy shared to make a tag visible to all organization members.`
}

func (c *CreateCommand) Examples() []string {
	return []string{
		"my-tag # Create a private tag",
		"my-tag --privacy shared # Create a shared tag",
		`my-tag --description "Assets flagged for review" # Create a tag with a description`,
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
	c.flags.privacy = flags.NewStringFlag(c.Flags(), false, "privacy", "", "private", "tag visibility (private, shared)")
	c.flags.description = flags.NewStringFlag(c.Flags(), false, "description", "", "", "a human-readable description of the tag")
	return nil
}

func (c *CreateCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}
	c.name = strings.TrimSpace(args[0])

	privacy, err := c.flags.privacy.Value()
	if err != nil {
		return err
	}
	c.privacy = privacy

	description, err := c.flags.description.Value()
	if err != nil {
		return err
	}
	c.description = optionalNonEmpty(description)

	return c.resolveTagsService()
}

func (c *CreateCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"privacy", c.privacy,
		"description_set", c.description.IsPresent(),
	)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Creating tag...",
		func(pctx context.Context) cenclierrors.CencliError {
			var createErr cenclierrors.CencliError
			c.result, createErr = c.tagsSvc.CreateTag(pctx, tags.CreateParams{
				OrgID:       c.orgID,
				Name:        c.name,
				Description: c.description,
				Privacy:     c.privacy,
			})
			return createErr
		},
	)
	if err != nil {
		logger.Debug("create tag failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	return c.PrintData(c, c.result.Tag)
}

func (c *CreateCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}

// RenderShort renders the created tag as a labeled detail view (TTY-aware).
func (c *CreateCommand) RenderShort() cenclierrors.CencliError {
	return renderTagDetail("━━━ Tag Created ━━━", c.result.Tag)
}
