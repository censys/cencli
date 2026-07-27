package tags

import (
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

const cmdName = "tags"

// Command is the parent tags command that groups tag-management subcommands.
type Command struct {
	*command.BaseCommand
}

var _ command.Command = (*Command)(nil)

// NewTagsCommand creates a new tags command with all subcommands.
func NewTagsCommand(cmdContext *command.Context) *Command {
	return &Command{BaseCommand: command.NewBaseCommand(cmdContext)}
}

func (c *Command) Use() string {
	return cmdName
}

func (c *Command) Short() string {
	return "Manage tags and tag assignments for your organization"
}

func (c *Command) Long() string {
	return `Manage tags and tag assignments for your organization.

Use --org-id on any subcommand to target a different organization; otherwise your stored organization ID is used.`
}

func (c *Command) Args() command.PositionalArgs {
	return command.ExactArgs(0)
}

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort}
}

func (c *Command) Init() error {
	return c.AddSubCommands(
		NewListCommand(c.Context),
		NewGetCommand(c.Context),
		NewCreateCommand(c.Context),
		NewUpdateCommand(c.Context),
		NewDeleteCommand(c.Context),
		NewAssignCommand(c.Context),
		NewUnassignCommand(c.Context),
		NewAssignmentsCommand(c.Context),
		NewOperationsCommand(c.Context),
	)
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// Parent command shows help when run without subcommands.
	if err := cmd.Help(); err != nil {
		return cenclierrors.NewCencliError(err)
	}
	return nil
}
