package scan

import (
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

const cmdName = "scan"

// Command is the parent scan command that groups live-scan subcommands.
type Command struct {
	*command.BaseCommand
}

var _ command.Command = (*Command)(nil)

// NewScanCommand creates a new scan command with all subcommands.
func NewScanCommand(cmdContext *command.Context) *Command {
	return &Command{BaseCommand: command.NewBaseCommand(cmdContext)}
}

func (c *Command) Use() string {
	return cmdName
}

func (c *Command) Short() string {
	return "Request live rescans of web properties and track scan status"
}

func (c *Command) Long() string {
	return `Request a live rescan of a web property and follow scans until they complete.

Scanning requires an organization (Enterprise). A rescan costs 10 credits per accepted request; reading a scan's status is free.`
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
		NewRescanCommand(c.Context),
		NewGetCommand(c.Context),
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
