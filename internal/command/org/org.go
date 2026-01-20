package org

import (
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/command/org/credits"
	"github.com/censys/cencli/internal/command/org/details"
	"github.com/censys/cencli/internal/command/org/members"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// Command is the parent org command that groups organization-related subcommands.
type Command struct {
	*command.BaseCommand
}

var _ command.Command = (*Command)(nil)

// NewOrgCommand creates a new org command with all subcommands.
func NewOrgCommand(cmdContext *command.Context) *Command {
	return &Command{BaseCommand: command.NewBaseCommand(cmdContext)}
}

func (c *Command) Use() string {
	return "org"
}

func (c *Command) Short() string {
	return "Manage and view organization details"
}

func (c *Command) Long() string {
	return `Manage and view organization details including credits, members, and organization information.

By default, these commands use your stored organization ID. If no organization ID is stored,
or you want to query a different organization, use the --org-id flag on each subcommand.

To set your default organization ID, run: censys config org-id set <org-id>`
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
		credits.NewCreditsCommand(c.Context),
		members.NewMembersCommand(c.Context),
		details.NewDetailsCommand(c.Context),
	)
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// Parent command shows help when run without subcommands
	if err := cmd.Help(); err != nil {
		return cenclierrors.NewCencliError(err)
	}
	return nil
}
