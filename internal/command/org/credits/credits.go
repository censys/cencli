package credits

import (
	"context"

	"github.com/spf13/cobra"

	appcredits "github.com/censys/cencli/internal/app/credits"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
)

const cmdName = "credits"

// Command displays credit details for an organization.
type Command struct {
	*command.BaseCommand
	// services
	creditsSvc appcredits.Service
	// flags
	flags creditsFlags
	// state
	orgID identifiers.OrganizationID
	// result
	result appcredits.OrganizationCreditDetailsResult
}

type creditsFlags struct {
	orgID flags.OrgIDFlag
}

var _ command.Command = (*Command)(nil)

// NewCreditsCommand creates a new org credits command.
func NewCreditsCommand(cmdContext *command.Context) *Command {
	return &Command{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *Command) Use() string {
	return cmdName
}

func (c *Command) Short() string {
	return "Display credit details for your organization"
}

func (c *Command) Long() string {
	return `Display credit details for your organization.

This command shows your organization's credit balance, auto-replenish configuration,
and any credit expirations.

By default, the stored organization ID is used. Use --org-id to query a specific organization.`
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
		"# Show credits for your stored organization",
		"--org-id <uuid>  # Show credits for a specific organization",
	}
}

func (c *Command) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(
		c.Flags(),
		"",
	)
	return nil
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.creditsSvc, err = c.CreditsService()
	if err != nil {
		return err
	}

	orgIDFromFlag, err := c.flags.orgID.Value()
	if err != nil {
		return err
	}
	if orgIDFromFlag.IsPresent() {
		c.orgID = orgIDFromFlag.MustGet()
	} else {
		storedOrgID, err := c.GetStoredOrgID(cmd.Context())
		if err != nil {
			return err
		}
		if storedOrgID.IsPresent() {
			c.orgID = storedOrgID.MustGet()
		}
	}
	// if no org ID is found, return an error
	if c.orgID.IsZero() {
		return cenclierrors.NewNoOrgIDError()
	}
	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	err := c.WithProgress(
		cmd.Context(),
		c.Logger(cmdName),
		"Fetching organization credits...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.creditsSvc.GetOrganizationCreditDetails(pctx, c.orgID)
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
	return c.showOrgCredits(c.result)
}
