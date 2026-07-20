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

const updateCmdName = "update"

// UpdateCommand implements `tags update <tag>`, mutating an existing tag by name or UUID.
type UpdateCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags updateCommandFlags
	// state - populated by PreRun
	orgID       mo.Option[identifiers.OrganizationID]
	tagID       identifiers.TagID
	name        mo.Option[string]
	privacy     mo.Option[string]
	description mo.Option[string]
	// result stores the updated tag for rendering
	result tags.UpdateResult
}

type updateCommandFlags struct {
	orgID            flags.OrgIDFlag
	name             flags.StringFlag
	privacy          flags.StringFlag
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
	return fmt.Sprintf("%s <tag>", updateCmdName)
}

func (c *UpdateCommand) Short() string {
	return "Update an existing tag"
}

func (c *UpdateCommand) Long() string {
	return `Update an existing tag by its name or UUID.

At least one mutation flag is required. Use --clear-description to remove a tag's description; it cannot be combined with --description.`
}

func (c *UpdateCommand) Examples() []string {
	return []string{
		`my-tag --description "Assets flagged for review" # Set a description`,
		"my-tag --privacy shared # Make a tag visible to the organization",
		"my-tag --name renamed-tag # Rename a tag",
		"my-tag --clear-description # Remove the description",
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
	c.flags.name = flags.NewStringFlag(c.Flags(), false, "name", "", "", "a new name for the tag")
	c.flags.privacy = flags.NewStringFlag(c.Flags(), false, "privacy", "", "", "tag visibility (private, shared)")
	c.flags.description = flags.NewStringFlag(c.Flags(), false, "description", "", "", "a new description for the tag")
	c.flags.clearDescription = flags.NewBoolFlag(c.Flags(), "clear-description", "", false, "remove the tag's description")
	return nil
}

func (c *UpdateCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}
	c.tagID = identifiers.NewTagID(args[0])

	name, err := c.flags.name.Value()
	if err != nil {
		return err
	}
	c.name = optionalNonEmpty(name)

	privacy, err := c.flags.privacy.Value()
	if err != nil {
		return err
	}
	c.privacy = optionalNonEmpty(privacy)

	description, err := c.flags.description.Value()
	if err != nil {
		return err
	}
	clearDescription, err := c.flags.clearDescription.Value()
	if err != nil {
		return err
	}
	if description != "" && clearDescription {
		return NewDescriptionConflictError()
	}
	if clearDescription {
		c.description = mo.Some("")
	} else {
		c.description = optionalNonEmpty(description)
	}

	if !c.name.IsPresent() && !c.privacy.IsPresent() && !c.description.IsPresent() {
		return NewNothingToUpdateError()
	}

	return c.resolveTagsService()
}

func (c *UpdateCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_is_uuid", c.tagID.UID().IsPresent(),
		"name_set", c.name.IsPresent(),
		"privacy_set", c.privacy.IsPresent(),
		"description_set", c.description.IsPresent(),
	)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Updating tag...",
		func(pctx context.Context) cenclierrors.CencliError {
			var updateErr cenclierrors.CencliError
			c.result, updateErr = c.tagsSvc.UpdateTag(pctx, tags.UpdateParams{
				OrgID:       c.orgID,
				TagID:       c.tagID,
				Name:        c.name,
				Description: c.description,
				Privacy:     c.privacy,
			})
			return updateErr
		},
	)
	if err != nil {
		logger.Debug("update tag failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	return c.PrintData(c, c.result.Tag)
}

func (c *UpdateCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}

// RenderShort renders the updated tag as a labeled detail view (TTY-aware).
func (c *UpdateCommand) RenderShort() cenclierrors.CencliError {
	return renderTagDetail("━━━ Tag Updated ━━━", c.result.Tag)
}
