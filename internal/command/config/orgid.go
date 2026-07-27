package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
)

// organizationIDCommand handles the organization-id global parameter
type organizationIDCommand struct {
	*command.BaseCommand
	accessible bool
	flags      organizationIDCommandFlags
}

type organizationIDCommandFlags struct {
	accessible flags.BoolFlag
}

var _ command.Command = (*organizationIDCommand)(nil)

func newOrganizationIDCommand(cmdContext *command.Context) *organizationIDCommand {
	cmd := &organizationIDCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}

	return cmd
}

func (c *organizationIDCommand) Use() string { return "org-id" }
func (c *organizationIDCommand) Short() string {
	return "Manage organization ID global values (personal access tokens only)"
}

func (c *organizationIDCommand) Long() string {
	return `View and manage organization ID values used across API requests.

These values apply only when authenticating with a personal access token, which
is not organization-scoped. If you logged in with "censys auth login", the
organization is fixed by that login and these values are ignored.`
}

func (c *organizationIDCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *organizationIDCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *organizationIDCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort}
}

func (c *organizationIDCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.accessible, err = c.flags.accessible.Value()
	if err != nil {
		return err
	}
	return nil
}

func (c *organizationIDCommand) Init() error {
	c.flags.accessible = flags.NewBoolFlag(
		c.Flags(),
		"accessible",
		"a",
		false,
		"enable accessible mode (non-redrawing)",
	)
	err := c.AddSubCommands(
		newAddOrganizationIDCommand(c.Context),
		newDeleteOrganizationIDCommand(c.Context),
		newActivateOrganizationIDCommand(c.Context),
	)
	return err
}

func (c *organizationIDCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	values, err := c.Store().GetValuesForGlobal(cmd.Context(), config.OrgIDGlobalName)
	if err != nil && !errors.Is(err, store.ErrGlobalNotFound) {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to get global values: %w", err))
	}

	if errors.Is(err, store.ErrGlobalNotFound) || len(values) == 0 {
		formatter.Printf(formatter.Stdout, "No organization IDs found. Use `%s` to add one.\n", cmd.CommandPath()+" add")
		return nil
	}

	if c.accessible {
		return c.printTable(cmd, values)
	}
	return c.runTable(cmd, values)
}

func (c *organizationIDCommand) runTable(cmd *cobra.Command, values []*store.ValueForGlobal) cenclierrors.CencliError {
	err := runValuesTable[*store.ValueForGlobal](
		"🌐 Stored Organization IDs",
		values,
		func(v *store.ValueForGlobal) int64 { return v.ID },
		func(v *store.ValueForGlobal) string { return v.Description },
		func(v *store.ValueForGlobal) string { return v.Value },
		func(v *store.ValueForGlobal) time.Time { return v.LastUsedAt },
		func(selected *store.ValueForGlobal) {
			_, err := c.Store().DeleteValueForGlobal(cmd.Context(), selected.ID)
			if err != nil {
				if !c.Config().Quiet {
					formatter.Printf(formatter.Stderr, "❌ Failed to delete global value: %v\n", err)
				}
				return
			}
			if !c.Config().Quiet {
				formatter.Printf(formatter.Stdout, "✅ Deleted organization ID \"%s\"\n", selected.Description)
			}
		},
		func(selected *store.ValueForGlobal) {
			err := c.Store().UpdateGlobalLastUsedAtToNow(cmd.Context(), selected.ID)
			if err != nil {
				if !c.Config().Quiet {
					formatter.Printf(formatter.Stderr, "❌ Failed to update global selection: %v\n", err)
				}
				return
			}
			if !c.Config().Quiet {
				formatter.Printf(formatter.Stdout, "✅ Selected new organization ID [%s]\n", selected.Description)
			}
		},
	)
	return cenclierrors.NewCencliError(err)
}

func (c *organizationIDCommand) printTable(cmd *cobra.Command, values []*store.ValueForGlobal) cenclierrors.CencliError {
	for _, v := range values {
		formatter.Printf(formatter.Stdout, "%s\n", v.String(false))
	}
	return nil
}
