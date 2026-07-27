package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/organizations"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/browser"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	client "github.com/censys/cencli/internal/pkg/clients/censys"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
)

type loginCommand struct {
	*command.BaseCommand
	flags loginCommandFlags
}

type loginCommandFlags struct {
	noBrowser flags.BoolFlag
}

var _ command.Command = (*loginCommand)(nil)

func newLoginCommand(cmdContext *command.Context) *loginCommand {
	return &loginCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *loginCommand) Use() string   { return "login" }
func (c *loginCommand) Short() string { return "Log in to the Censys Platform via your browser" }
func (c *loginCommand) Long() string {
	return `Obtain credentials for the Censys Platform via your web browser.

This starts a local listener, opens your browser to the Censys authorization
server, and captures the resulting tokens after you log in. Tokens are stored
locally and refreshed automatically; run "censys auth logout" to remove them.

The obtained login takes precedence over any stored personal access token.
To switch back to a personal access token, log out or run
"censys config auth activate <id>".`
}

func (c *loginCommand) Examples() []string {
	return []string{
		"censys auth login",
		"censys auth login --no-browser",
	}
}

func (c *loginCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *loginCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *loginCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort}
}

func (c *loginCommand) Init() error {
	c.flags.noBrowser = flags.NewBoolFlag(
		c.Flags(),
		"no-browser",
		"",
		false,
		"print the login URL instead of opening a browser",
	)
	return nil
}

func (c *loginCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *loginCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	noBrowser, err := c.flags.noBrowser.Value()
	if err != nil {
		return err
	}

	oauthClient := newOAuthClient()

	sess, loginErr := oauthClient.Login(cmd.Context(), func(authorizeURL string) error {
		if noBrowser {
			formatter.Printf(formatter.Stdout, "Go to the following link in your browser:\n\n    %s\n\nWaiting for login to complete...\n", authorizeURL)
			return nil
		}
		formatter.Printf(formatter.Stdout, "Your browser has been opened to visit:\n\n    %s\n\nWaiting for login to complete...\n", authorizeURL)
		if openErr := browser.Open(authorizeURL); openErr != nil {
			formatter.Printf(formatter.Stderr, "⚠️  Could not open your browser (%v) - open the link above manually.\n", openErr)
		}
		return nil
	})
	if loginErr != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("login failed: %w", loginErr))
	}

	// Replace any previous session: insert the new one first (so a failure
	// never leaves the user logged out), then drop the old rows.
	previous, prevErr := c.Store().GetValuesForAuth(cmd.Context(), config.OAuthSessionName)
	if prevErr != nil && !errors.Is(prevErr, authdom.ErrAuthNotFound) {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to read existing oauth session: %w", prevErr))
	}

	description := sess.Account()
	if description == "" {
		description = "oauth login"
	}

	value, marshalErr := sess.Marshal()
	if marshalErr != nil {
		return cenclierrors.NewCencliError(marshalErr)
	}
	added, addErr := c.Store().AddValueForAuth(cmd.Context(), config.OAuthSessionName, description, value)
	if addErr != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to store oauth session: %w", addErr))
	}

	// Best-effort: the token has only the org ID, so resolve the name via the
	// API (authenticating with the just-stored session) and persist it so
	// `auth status` can show it offline. On failure we keep the ID.
	if sess.OrgID != "" {
		if name := c.resolveOrgName(cmd.Context(), sess.OrgID); name != "" {
			sess.OrgName = name
			if named, mErr := sess.Marshal(); mErr == nil {
				// Update the row inserted above in place, so the session is never
				// duplicated and never briefly absent.
				_, _ = c.Store().UpdateValueForAuth(cmd.Context(), added.ID, description, named)
			}
		}
	}
	for _, old := range previous {
		if _, delErr := c.Store().DeleteValueForAuth(cmd.Context(), old.ID); delErr != nil {
			return cenclierrors.NewCencliError(fmt.Errorf("failed to remove previous oauth session: %w", delErr))
		}
	}

	if account := sess.Account(); account != "" {
		formatter.Printf(formatter.Stdout, "\n✅ You are now logged in as [%s]\n", account)
	} else {
		formatter.Printf(formatter.Stdout, "\n✅ You are now logged in\n")
	}
	if sess.OrgID != "" {
		formatter.Printf(formatter.Stdout, "This session is scoped to organization [%s].\n", sess.OrgLabel())
	} else {
		formatter.Printf(formatter.Stdout, "This session is scoped to your free account.\n")
	}
	if sess.RefreshToken == "" {
		formatter.Printf(formatter.Stderr, "⚠️  No refresh token was granted; you will need to log in again when the session expires.\n")
	}
	return nil
}

// resolveOrgName returns the organization's name via the API, or "" on any
// failure (the caller falls back to the org ID).
func (c *loginCommand) resolveOrgName(ctx context.Context, orgID string) string {
	uid, err := uuid.Parse(orgID)
	if err != nil {
		return ""
	}
	cli, err := client.NewCensysSDK(ctx, c.Store(), c.Config())
	if err != nil {
		return ""
	}
	// Only the name is needed; skip member counts, which add seconds to the call.
	res, orgErr := organizations.New(cli).GetOrganizationDetails(ctx, identifiers.NewOrganizationID(uid), false)
	if orgErr != nil {
		return ""
	}
	return res.Data.Name
}
