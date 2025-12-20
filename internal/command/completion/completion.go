package completion

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

type Command struct {
	*command.BaseCommand
}

var _ command.Command = (*Command)(nil)

func NewCompletionCommand(cmdContext *command.Context) *Command {
	return &Command{BaseCommand: command.NewBaseCommand(cmdContext)}
}

func (c *Command) Use() string   { return "completion <target shell>" }
func (c *Command) Short() string { return "Generate shell completion scripts" }
func (c *Command) Long() string {
	return "Generate shell completion scripts for the Censys CLI. You can save the output to a file and source it in your shell configuration file.\n\nFor example, if you saved output to ~/.censys_zsh.sh, you can add 'source ~/.censys_zsh.sh' to your .zshrc."
}

func (c *Command) Args() command.PositionalArgs {
	return command.RangeArgs(1, 1)
}

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort}
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *Command) Examples() []string {
	return []string{
		"zsh > censys_zsh.sh",
		"bash > censys_bash.sh",
		"fish > censys_fish.sh",
		"powershell > censys_powershell.ps1",
	}
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	switch args[0] {
	case "bash":
		if err := cmd.Root().GenBashCompletion(cmd.OutOrStdout()); err != nil {
			return cenclierrors.NewCencliError(err)
		}
	case "zsh":
		if err := cmd.Root().GenZshCompletion(cmd.OutOrStdout()); err != nil {
			return cenclierrors.NewCencliError(err)
		}
	case "fish":
		if err := cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true); err != nil {
			return cenclierrors.NewCencliError(err)
		}
	case "powershell":
		if err := cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout()); err != nil {
			return cenclierrors.NewCencliError(err)
		}
	default:
		return cenclierrors.NewCencliError(fmt.Errorf("invalid shell %q", args[0]))
	}
	return nil
}
