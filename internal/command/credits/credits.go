package credits

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/credits"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/flags"
)

const cmdName = "credits"

// creditsMode indicates whether to fetch org or user credits
type creditsMode int

const (
	modeOrg creditsMode = iota
	modeUser
)

type Command struct {
	*command.BaseCommand
	// services the command uses
	creditsSvc credits.Service
	// flags the command uses
	flags creditsCommandFlags
	// state - populated by PreRun (through flags, args, etc.)
	mode  creditsMode
	orgID uuid.UUID
}

type creditsCommandFlags struct {
	orgID    flags.UUIDFlag
	freeUser flags.BoolFlag
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
	return `Display credit details for your Censys account.

By default, if you have an organization ID configured, this command shows
organization credits. Otherwise, it shows your free user credits.

Use --free-user to explicitly show free user credits even when an org ID
is configured. Use --org-id to query a specific organization.`
}

func (c *Command) Args() command.PositionalArgs {
	return command.ExactArgs(0)
}

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeData
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeData}
}

func (c *Command) Examples() []string {
	return []string{
		"# Show credits (org if configured, otherwise free user)",
		"--free-user  # Show free user credits",
		"--org-id <uuid>  # Show credits for a specific organization",
	}
}

func (c *Command) Init() error {
	c.flags.orgID = flags.NewUUIDFlag(
		c.Flags(),
		false,
		"org-id",
		"o",
		mo.None[uuid.UUID](),
		"organization ID to query credits for (overrides stored ID)",
	)
	c.flags.freeUser = flags.NewBoolFlag(
		c.Flags(),
		"free-user",
		"u",
		false,
		"show free user credits instead of organization credits",
	)
	return nil
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.creditsSvc, err = c.CreditsService()
	if err != nil {
		return err
	}

	// Get flag values
	orgID, err := c.flags.orgID.Value()
	if err != nil {
		return err
	}
	orgIDProvided := orgID.IsPresent()

	freeUser, err := c.flags.freeUser.Value()
	if err != nil {
		return err
	}

	// Check for conflicting flags
	if orgIDProvided && freeUser {
		return flags.NewConflictingFlagsError("org-id", "free-user")
	}

	// Determine mode based on flags and stored org ID
	switch {
	case freeUser:
		c.mode = modeUser
	case orgIDProvided:
		c.mode = modeOrg
		c.orgID = orgID.MustGet()
	case c.HasOrgID():
		storedOrgID, storedErr := c.GetStoredOrgID(cmd.Context())
		if storedErr != nil {
			return storedErr
		}
		if storedOrgID.IsPresent() {
			c.mode = modeOrg
			c.orgID = storedOrgID.MustGet()
		} else {
			c.mode = modeUser
		}
	default:
		c.mode = modeUser
	}

	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	if c.mode == modeOrg {
		return c.runOrgCredits(cmd.Context())
	}
	return c.runUserCredits(cmd.Context())
}

func (c *Command) runOrgCredits(ctx context.Context) cenclierrors.CencliError {
	var result credits.OrganizationCreditDetailsResult
	err := c.WithProgress(
		ctx,
		c.Logger(cmdName),
		"Fetching organization credits...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			result, fetchErr = c.creditsSvc.GetOrganizationCreditDetails(pctx, c.orgID)
			return fetchErr
		},
	)
	if err != nil {
		return err
	}

	c.PrintAppResponseMeta(result.Meta)
	return c.PrintData(c, result.Data)
}

func (c *Command) runUserCredits(ctx context.Context) cenclierrors.CencliError {
	var result credits.UserCreditDetailsResult
	err := c.WithProgress(
		ctx,
		c.Logger(cmdName),
		"Fetching user credits...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			result, fetchErr = c.creditsSvc.GetUserCreditDetails(pctx)
			return fetchErr
		},
	)
	if err != nil {
		return err
	}

	c.PrintAppResponseMeta(result.Meta)
	return c.PrintData(c, result.Data)
}
