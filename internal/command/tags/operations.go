package tags

import (
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

const operationsCmdName = "operations"

// OperationsCommand is the parent for the tag-operations family, which tracks
// the asynchronous jobs created by bulk assign and unassign.
type OperationsCommand struct {
	*command.BaseCommand
}

var _ command.Command = (*OperationsCommand)(nil)

// NewOperationsCommand creates the operations parent command with its subcommands.
func NewOperationsCommand(cmdContext *command.Context) *OperationsCommand {
	return &OperationsCommand{BaseCommand: command.NewBaseCommand(cmdContext)}
}

func (c *OperationsCommand) Use() string {
	return operationsCmdName
}

func (c *OperationsCommand) Short() string {
	return "Track the asynchronous jobs created by bulk tag operations"
}

func (c *OperationsCommand) Long() string {
	return `Track the asynchronous jobs created by bulk tag operations.

Bulk assign and unassign submit a job rather than acting immediately; these commands list those jobs and inspect a single one.`
}

func (c *OperationsCommand) Args() command.PositionalArgs {
	return command.ExactArgs(0)
}

func (c *OperationsCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *OperationsCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort}
}

func (c *OperationsCommand) Init() error {
	return c.AddSubCommands(
		NewOperationsListCommand(c.Context),
		NewOperationsGetCommand(c.Context),
	)
}

func (c *OperationsCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *OperationsCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// Parent command shows help when run without subcommands.
	if err := cmd.Help(); err != nil {
		return cenclierrors.NewCencliError(err)
	}
	return nil
}
