package config

import (
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

type Command struct {
	*command.BaseCommand
}

var _ command.Command = (*Command)(nil)

func NewConfigCommand(cmdContext *command.Context) *Command {
	cmd := &Command{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}

	return cmd
}

func (c *Command) Use() string   { return "config" }
func (c *Command) Short() string { return "Manage configuration" }
func (c *Command) Long() string  { return "Manage configuration" }

func (c *Command) Init() error {
	return c.AddSubCommands(
		newAuthCommand(c.Context),
		newOrganizationIDCommand(c.Context),
		newPrintCommand(c.Context),
		newTemplatesCommand(c.Context),
	)
}

func (c *Command) Args() command.PositionalArgs { return command.ExactArgs(0) }
func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return cenclierrors.NewCencliError(cmd.Help())
}
