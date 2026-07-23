// Package auth implements the `censys auth` command group: browser-based
// OAuth2 login (like `gcloud auth login`), logout, and credential status.
package auth

import (
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/oauth"
)

// newOAuthClient builds the OAuth2 client used by the auth commands. It is a
// package-level indirection so tests can substitute a client backed by a stub
// transport instead of reaching the real authorization server.
var newOAuthClient = func() *oauth.Client {
	return oauth.NewClient(oauth.Config{}, &http.Client{Timeout: 30 * time.Second})
}

type Command struct {
	*command.BaseCommand
}

var _ command.Command = (*Command)(nil)

func NewAuthCommand(cmdContext *command.Context) *Command {
	return &Command{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *Command) Use() string   { return "auth" }
func (c *Command) Short() string { return "Log in and manage your Censys credentials" }
func (c *Command) Long() string {
	return "Authenticate with the Censys Platform using your browser, view credential status, and log out."
}

func (c *Command) Init() error {
	return c.AddSubCommands(
		newLoginCommand(c.Context),
		newLogoutCommand(c.Context),
		newStatusCommand(c.Context),
	)
}

func (c *Command) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort}
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return cenclierrors.NewCencliError(cmd.Help())
}
