package members

import (
	"context"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/organizations"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
)

const cmdName = "members"

// Command displays organization members.
type Command struct {
	*command.BaseCommand
	// services
	orgSvc organizations.Service
	// flags
	flags membersFlags
	// state
	orgID       identifiers.OrganizationID
	interactive bool
	// result
	result organizations.OrganizationMembersResult
}

type membersFlags struct {
	orgID       flags.OrgIDFlag
	interactive flags.BoolFlag
}

var _ command.Command = (*Command)(nil)

// NewMembersCommand creates a new org members command.
func NewMembersCommand(cmdContext *command.Context) *Command {
	return &Command{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *Command) Use() string {
	return cmdName
}

func (c *Command) Short() string {
	return "List organization members"
}

func (c *Command) Long() string {
	return `List members in your organization.

This command displays all members including their email, name, roles, and last login time.

By default, the stored organization ID is used. Use --org-id to query a specific organization.
Use --interactive for a navigable table view.`
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
		"# List members for your stored organization",
		"--interactive  # List members in an interactive table",
		"--org-id <uuid>  # List members for a specific organization",
		"--output-format json  # Output as JSON",
	}
}

func (c *Command) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(
		c.Flags(),
		"",
	)
	c.flags.interactive = flags.NewBoolFlag(
		c.Flags(),
		"interactive",
		"i",
		false,
		"display results in an interactive table (TUI)",
	)
	return nil
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgSvc, err = c.OrganizationsService()
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

	// Get interactive flag
	c.interactive, err = c.flags.interactive.Value()
	if err != nil {
		return err
	}

	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	err := c.WithProgress(
		cmd.Context(),
		c.Logger(cmdName),
		"Fetching organization members...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.orgSvc.ListOrganizationMembers(
				pctx,
				c.orgID,
				mo.None[uint](), // pageSize - get all
				mo.None[uint](), // maxPages - no limit
			)
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
	if c.interactive {
		return c.showInteractiveTable(c.result)
	}
	return c.showRawTable(c.result)
}
