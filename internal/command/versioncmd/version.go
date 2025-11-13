package versioncmd

import (
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	appversion "github.com/censys/cencli/internal/version"
)

type Command struct {
	*command.BaseCommand
}

var _ command.Command = (*Command)(nil)

func NewVersionCommand(cmdContext *command.Context) *Command {
	return &Command{BaseCommand: command.NewBaseCommand(cmdContext)}
}

func (c *Command) Use() string {
	return "version"
}

func (c *Command) Short() string {
	return "Print version information"
}

func (c *Command) Args() command.PositionalArgs {
	return command.RangeArgs(0, 0)
}

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeData
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeData}
}

func (c *Command) Init() error {
	return nil
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	info := appversion.BuildInfo()
	return c.PrintData(c, info)
}
