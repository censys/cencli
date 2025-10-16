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

type authCommand struct {
	*command.BaseCommand
	accessible bool
	flags      authCommandFlags
}

type authCommandFlags struct {
	accessible flags.BoolFlag
}

var _ command.Command = (*authCommand)(nil)

func newAuthCommand(cmdContext *command.Context) *authCommand {
	cmd := &authCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
	return cmd
}

func (c *authCommand) Use() string          { return "auth" }
func (c *authCommand) Short() string        { return "Manage personal access token authentication" }
func (c *authCommand) Long() string         { return "View and manage your personal access token values." }
func (c *authCommand) DisableTimeout() bool { return true }

func (c *authCommand) Init() error {
	c.flags.accessible = flags.NewBoolFlag(
		c.Flags(),
		"accessible",
		"a",
		false,
		"enable accessible mode (non-redrawing)",
	)
	err := c.AddSubCommands(
		newAddAuthCommand(c.Context),
		newDeleteAuthCommand(c.Context),
		newActivateAuthCommand(c.Context),
	)
	return err
}

func (c *authCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }
func (c *authCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.accessible, err = c.flags.accessible.Value()
	if err != nil {
		return err
	}
	return nil
}

func (c *authCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	values, err := c.Store().GetValuesForAuth(cmd.Context(), config.AuthName)
	if err != nil && !errors.Is(err, store.ErrAuthNotFound) {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to get auth values: %w", err))
	}

	if errors.Is(err, store.ErrAuthNotFound) || len(values) == 0 {
		formatter.Printf(formatter.Stdout, "No personal access tokens found. Use `%s` to add one.\n", cmd.CommandPath()+" add")
		return nil
	}

	if c.accessible {
		return c.printTable(cmd, values)
	}
	return c.runTable(cmd, values)
}

func (c *authCommand) runTable(cmd *cobra.Command, values []*store.ValueForAuth) cenclierrors.CencliError {
	err := runValuesTable[*store.ValueForAuth](
		"🔑 Stored Personal Access Tokens",
		values,
		func(v *store.ValueForAuth) int64 { return v.ID },
		func(v *store.ValueForAuth) string { return v.Description },
		func(v *store.ValueForAuth) string { return v.Value },
		func(v *store.ValueForAuth) time.Time { return v.LastUsedAt },
		func(selected *store.ValueForAuth) {
			_, err := c.Store().DeleteValueForAuth(cmd.Context(), selected.ID)
			if err != nil {
				if !c.Config().Quiet {
					formatter.Printf(formatter.Stderr, "❌ Failed to delete auth value: %v\n", err)
				}
				return
			}
			if !c.Config().Quiet {
				formatter.Printf(formatter.Stdout, "✅ Deleted personal access token \"%s\"\n", selected.Description)
			}
		},
		func(selected *store.ValueForAuth) {
			err := c.Store().UpdateAuthLastUsedAtToNow(cmd.Context(), selected.ID)
			if err != nil {
				if !c.Config().Quiet {
					formatter.Printf(formatter.Stderr, "❌ Failed to update auth selection: %v\n", err)
				}
				return
			}
			if !c.Config().Quiet {
				formatter.Printf(formatter.Stdout, "✅ Selected new personal access token [%s]\n", selected.Description)
			}
		},
	)
	return cenclierrors.NewCencliError(err)
}

func (c *authCommand) printTable(cmd *cobra.Command, values []*store.ValueForAuth) cenclierrors.CencliError {
	for _, v := range values {
		formatter.Printf(formatter.Stdout, "%s\n", v.String(false))
	}
	return nil
}
