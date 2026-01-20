package credits

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/credits"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

const cmdName = "credits"

type Command struct {
	*command.BaseCommand
	// services the command uses
	creditsSvc credits.Service
	// result stored for rendering
	result credits.UserCreditDetailsResult
}

var _ command.Command = (*Command)(nil)

func NewCreditsCommand(cmdContext *command.Context) *Command {
	return &Command{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *Command) Use() string {
	return cmdName
}

func (c *Command) Short() string {
	return "Display credit details for your Censys account"
}

func (c *Command) Long() string {
	return `Display credit details for your Free user Censys account.

Note: This command only shows free user credits. If you want to see organization credits,
run "censys org credits" instead.`
}

func (c *Command) Args() command.PositionalArgs {
	return command.ExactArgs(0)
}

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeData, command.OutputTypeShort}
}

func (c *Command) Examples() []string {
	return []string{
		"# Show free user credits",
	}
}

func (c *Command) Init() error {
	return nil
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.creditsSvc, err = c.CreditsService()
	if err != nil {
		return err
	}
	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	err := c.WithProgress(
		cmd.Context(),
		c.Logger(cmdName),
		"Fetching user credits...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.creditsSvc.GetUserCreditDetails(pctx)
			return fetchErr
		},
	)
	if err != nil {
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)
	return c.PrintData(c, c.result.Data)
}

func (c *Command) RenderShort() cenclierrors.CencliError {
	return c.showUserCredits(c.result)
}
