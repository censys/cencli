package config

import (
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/spf13/cobra"
)

type printCommand struct {
	*command.BaseCommand
}

var _ command.Command = (*printCommand)(nil)

func newPrintCommand(ctx *command.Context) *printCommand {
	return &printCommand{BaseCommand: command.NewBaseCommand(ctx)}
}

func (c *printCommand) Use() string   { return "print" }
func (c *printCommand) Short() string { return "Print the current configuration" }

func (c *printCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *printCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *printCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return c.PrintYAML(c.Config())
}
