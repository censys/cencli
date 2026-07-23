package auth

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/oauth"
)

type logoutCommand struct {
	*command.BaseCommand
}

var _ command.Command = (*logoutCommand)(nil)

func newLogoutCommand(cmdContext *command.Context) *logoutCommand {
	return &logoutCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *logoutCommand) Use() string   { return "logout" }
func (c *logoutCommand) Short() string { return "Log out of the Censys Platform" }
func (c *logoutCommand) Long() string {
	return `Revoke the OAuth2 session obtained via "censys auth login" and remove it
from local storage. Stored personal access tokens are not affected.`
}

func (c *logoutCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *logoutCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *logoutCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort}
}

func (c *logoutCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *logoutCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	sessions, err := c.Store().GetValuesForAuth(cmd.Context(), config.OAuthSessionName)
	if err != nil && !errors.Is(err, authdom.ErrAuthNotFound) {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to read oauth session: %w", err))
	}
	if len(sessions) == 0 {
		formatter.Printf(formatter.Stdout, "You are not logged in.\n")
		return nil
	}

	oauthClient := newOAuthClient()

	for _, rec := range sessions {
		// Best-effort revocation: revoking the refresh token invalidates the
		// whole grant server-side. Local removal below is what logs us out.
		if sess, parseErr := oauth.ParseSession(rec.Value); parseErr == nil {
			token := sess.RefreshToken
			if token == "" {
				token = sess.AccessToken
			}
			if revokeErr := oauthClient.Revoke(cmd.Context(), token); revokeErr != nil {
				formatter.Printf(formatter.Stderr, "⚠️  Failed to revoke session with the authorization server: %v\n", revokeErr)
			}
		}
		if _, delErr := c.Store().DeleteValueForAuth(cmd.Context(), rec.ID); delErr != nil {
			return cenclierrors.NewCencliError(fmt.Errorf("failed to remove oauth session: %w", delErr))
		}
	}

	formatter.Printf(formatter.Stdout, "✅ Logged out.\n")
	return nil
}
