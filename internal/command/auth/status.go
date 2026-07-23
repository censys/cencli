package auth

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/oauth"
)

type statusCommand struct {
	*command.BaseCommand
}

var _ command.Command = (*statusCommand)(nil)

func newStatusCommand(cmdContext *command.Context) *statusCommand {
	return &statusCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *statusCommand) Use() string   { return "status" }
func (c *statusCommand) Short() string { return "Show the credential used for API requests" }
func (c *statusCommand) Long() string {
	return `Show which credential (OAuth2 login or personal access token) is currently
used to authenticate API requests.`
}

func (c *statusCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *statusCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *statusCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort}
}

func (c *statusCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *statusCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	cred, isOAuth, err := client.ActiveCredential(cmd.Context(), c.Store())
	if err != nil {
		if errors.Is(err, authdom.ErrAuthNotFound) {
			formatter.Printf(formatter.Stdout, "No credentials configured.\nRun `censys auth login` to log in with your browser, or `censys config auth add` to add a personal access token.\n")
			return nil
		}
		return cenclierrors.NewCencliError(err)
	}

	if !isOAuth {
		formatter.Printf(formatter.Stdout, "Authenticated with a personal access token [%s]\n", cred.Description)
		formatter.Printf(formatter.Stdout, "Manage tokens with `censys config auth`.\n")
		return nil
	}

	sess, parseErr := oauth.ParseSession(cred.Value)
	if parseErr != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("%w (run `censys auth login` to log in again)", parseErr))
	}

	if account := sess.Account(); account != "" {
		formatter.Printf(formatter.Stdout, "Logged in as [%s]\n", account)
	} else {
		formatter.Printf(formatter.Stdout, "Logged in via `censys auth login`\n")
	}
	return nil
}
