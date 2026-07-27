package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/credential"
	authdom "github.com/censys/cencli/internal/pkg/domain/auth"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/oauth"
)

type statusCommand struct {
	*command.BaseCommand
	result statusResult
}

// statusResult is the machine-readable view of the active credential, rendered
// directly for data output (json/yaml/tree) and summarized by RenderShort.
type statusResult struct {
	Authenticated    bool   `json:"authenticated"`
	Method           string `json:"method,omitempty"` // "oauth" | "personal_access_token"
	Account          string `json:"account,omitempty"`
	Token            string `json:"token,omitempty"`           // personal access token description
	Scope            string `json:"scope,omitempty"`           // "organization" | "free_account" (oauth only)
	OrganizationID   string `json:"organization_id,omitempty"` // oauth org-bound session
	OrganizationName string `json:"organization_name,omitempty"`
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
	return []command.OutputType{command.OutputTypeData, command.OutputTypeShort}
}

func (c *statusCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *statusCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	result, err := c.gatherStatus(cmd.Context())
	if err != nil {
		return err
	}
	c.result = result
	return c.PrintData(c, c.result)
}

// gatherStatus resolves the active credential into a statusResult.
func (c *statusCommand) gatherStatus(ctx context.Context) (statusResult, cenclierrors.CencliError) {
	cred, kind, err := credential.Active(ctx, c.Store())
	if err != nil {
		if errors.Is(err, authdom.ErrAuthNotFound) {
			return statusResult{Authenticated: false}, nil
		}
		return statusResult{}, cenclierrors.NewCencliError(err)
	}

	switch kind {
	case credential.KindPersonalAccessToken:
		return statusResult{
			Authenticated: true,
			Method:        "personal_access_token",
			Token:         cred.Description,
		}, nil

	case credential.KindOAuth:
		sess, parseErr := oauth.ParseSession(cred.Value)
		if parseErr != nil {
			return statusResult{}, cenclierrors.NewCencliError(fmt.Errorf("%w (run `censys auth login` to log in again)", parseErr))
		}

		result := statusResult{
			Authenticated: true,
			Method:        "oauth",
			Account:       sess.Account(),
		}
		if sess.OrgID != "" {
			result.Scope = "organization"
			result.OrganizationID = sess.OrgID
			result.OrganizationName = sess.OrgName
		} else {
			result.Scope = "free_account"
		}
		return result, nil

	default:
		return statusResult{}, cenclierrors.NewCencliError(fmt.Errorf("unknown credential kind: %s", kind))
	}
}

func (c *statusCommand) RenderShort() cenclierrors.CencliError {
	r := c.result

	if !r.Authenticated {
		formatter.Printf(formatter.Stdout, "No credentials configured.\nRun `censys auth login` to log in with your browser, or `censys config auth add` to add a personal access token.\n")
		return nil
	}

	if r.Method != "oauth" {
		formatter.Printf(formatter.Stdout, "Authenticated with a personal access token [%s]\n", r.Token)
		formatter.Printf(formatter.Stdout, "Manage tokens with `censys config auth`.\n")
		return nil
	}

	if r.Account != "" {
		formatter.Printf(formatter.Stdout, "Logged in as [%s]\n", r.Account)
	} else {
		formatter.Printf(formatter.Stdout, "Logged in via `censys auth login`\n")
	}
	if r.Scope == "organization" {
		label := r.OrganizationName
		if label == "" {
			label = r.OrganizationID
		}
		formatter.Printf(formatter.Stdout, "Scoped to organization [%s].\n", label)
	} else {
		formatter.Printf(formatter.Stdout, "Scoped to your free account.\n")
	}
	return nil
}
