package config

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/censyscopy"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/ui/form"
	"github.com/censys/cencli/internal/store"
)

type addAuthCommand struct {
	*command.BaseCommand
	accessible bool
	flags      addAuthCommandFlags
}

type addAuthCommandFlags struct {
	accessible flags.BoolFlag
	name       flags.StringFlag
	value      flags.StringFlag
	valueFile  flags.FileFlag
	activate   flags.BoolFlag
}

var _ command.Command = (*addAuthCommand)(nil)

func newAddAuthCommand(cmdContext *command.Context) *addAuthCommand {
	cmd := &addAuthCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
	return cmd
}

func (c *addAuthCommand) Use() string   { return "add" }
func (c *addAuthCommand) Short() string { return "Add a new personal access token" }
func (c *addAuthCommand) Long() string  { return "Add a new personal access token for authentication." }

func (c *addAuthCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *addAuthCommand) Init() error {
	c.flags.accessible = flags.NewBoolFlag(
		c.Flags(),
		"accessible",
		"a",
		false,
		"enable accessible mode (non-redrawing)",
	)
	// Non-interactive seeding flags
	c.flags.name = flags.NewStringFlag(
		c.Flags(),
		false,
		"name",
		"n",
		"ci",
		"friendly name/description associated with this token (default: ci)",
	)
	c.flags.value = flags.NewStringFlag(
		c.Flags(),
		false,
		"value",
		"",
		"",
		"personal access token value (non-interactive)",
	)
	c.flags.valueFile = flags.NewFileFlag(
		c.Flags(),
		false,
		"value-file",
		"",
		"read the token value from a file or '-' for stdin (non-interactive)",
	)
	c.flags.activate = flags.NewBoolFlag(
		c.Flags(),
		"activate",
		"",
		true,
		"mark the added token as active (default: true)",
	)
	return nil
}

func (c *addAuthCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.accessible, err = c.flags.accessible.Value()
	if err != nil {
		return err
	}
	return nil
}

func (c *addAuthCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// Non-interactive path when value or value-file provided
	// Prefer value-file if both are present
	filePath, _ := c.flags.valueFile.Value()
	valueStr, _ := c.flags.value.Value()
	if filePath != "" || valueStr != "" {
		var token string
		var err cenclierrors.CencliError
		if filePath != "" {
			// Use first non-empty line from the file/stdin
			var lines []string
			lines, err = c.flags.valueFile.Lines(cmd)
			if err != nil {
				return err
			}
			for _, ln := range lines {
				if ln != "" {
					token = ln
					break
				}
			}
			if token == "" {
				return cenclierrors.NewCencliError(fmt.Errorf("value-file is empty"))
			}
		} else {
			token = valueStr
		}

		name, nerr := c.flags.name.Value()
		if nerr != nil {
			return nerr
		}
		if name == "" {
			name = "ci"
		}

		rec, aerr := c.Store().AddValueForAuth(cmd.Context(), config.AuthName, name, token)
		if aerr != nil {
			return cenclierrors.NewCencliError(fmt.Errorf("failed to add auth value: %w", aerr))
		}

		activate, actErr := c.flags.activate.Value()
		if actErr != nil {
			return actErr
		}
		if activate {
			if uerr := c.Store().UpdateAuthLastUsedAtToNow(cmd.Context(), rec.ID); uerr != nil {
				return cenclierrors.NewCencliError(fmt.Errorf("failed to activate auth: %w", uerr))
			}
		}

		formatter.Printf(formatter.Stdout, "✅ Added new personal access token [%s]\n", name)
		return nil
	}

	var value string
	var name string

	f := form.NewForm(
		huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					EchoMode(huh.EchoModePassword).
					Title("Enter a new personal access token").
					Description(censyscopy.DocumentationPAT(formatter.Stdout)).
					Value(&value).
					Validate(form.NonEmpty("token value cannot be empty")),
				huh.NewInput().
					Title("Enter a name for this token").
					Description("A friendly name to help identify this token").
					Validate(form.NonEmpty("name cannot be empty")).
					Value(&name),
			),
		),
		form.WithAccessible(c.accessible),
	)

	err := f.RunWithContext(cmd.Context())
	if err != nil {
		if errors.Is(err, form.ErrUserAborted) {
			return nil
		}
		return cenclierrors.NewCencliError(err)
	}

	_, err = c.Store().AddValueForAuth(cmd.Context(), config.AuthName, name, value)
	if err != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to add auth value: %w", err))
	}

	formatter.Printf(formatter.Stdout, "✅ Added new personal access token [%s]\n", name)
	return nil
}

type deleteAuthCommand struct {
	*command.BaseCommand
}

var _ command.Command = (*deleteAuthCommand)(nil)

func newDeleteAuthCommand(ctx *command.Context) *deleteAuthCommand {
	return &deleteAuthCommand{BaseCommand: command.NewBaseCommand(ctx)}
}

func (c *deleteAuthCommand) Use() string   { return "delete <id>" }
func (c *deleteAuthCommand) Short() string { return "Delete a personal access token" }

func (c *deleteAuthCommand) Args() command.PositionalArgs { return command.ExactArgs(1) }

func (c *deleteAuthCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *deleteAuthCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return newInvalidAuthIDError(args[0], "not an integer")
	}

	_, err = c.Store().DeleteValueForAuth(cmd.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrAuthNotFound) {
			return newInvalidAuthIDError(args[0], "record does not exist")
		}
		return cenclierrors.NewCencliError(fmt.Errorf("failed to delete auth value: %w", err))
	}

	formatter.Printf(formatter.Stdout, "✅ Deleted personal access token [%s]\n", args[0])
	return nil
}

type activateAuthCommand struct {
	*command.BaseCommand
}

var _ command.Command = (*activateAuthCommand)(nil)

func newActivateAuthCommand(ctx *command.Context) *activateAuthCommand {
	return &activateAuthCommand{BaseCommand: command.NewBaseCommand(ctx)}
}

func (c *activateAuthCommand) Use() string   { return "activate <id>" }
func (c *activateAuthCommand) Short() string { return "Set the active personal access token" }

func (c *activateAuthCommand) Args() command.PositionalArgs { return command.ExactArgs(1) }

func (c *activateAuthCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *activateAuthCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return newInvalidAuthIDError(args[0], "not an integer")
	}

	err = c.Store().UpdateAuthLastUsedAtToNow(cmd.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrAuthNotFound) {
			return newInvalidAuthIDError(args[0], "record does not exist")
		}
		return cenclierrors.NewCencliError(fmt.Errorf("failed to activate auth: %w", err))
	}

	formatter.Printf(formatter.Stdout, "✅ Activated personal access token [%s]\n", args[0])
	return nil
}

type InvalidAuthIDError interface {
	cenclierrors.CencliError
}

type invalidAuthIDError struct {
	id     string
	reason string
}

var _ InvalidAuthIDError = &invalidAuthIDError{}

func newInvalidAuthIDError(id string, reason string) InvalidAuthIDError {
	return &invalidAuthIDError{id: id, reason: reason}
}

func (e *invalidAuthIDError) Error() string {
	return fmt.Sprintf("invalid auth ID: %s (%s)", e.id, e.reason)
}

func (e *invalidAuthIDError) Title() string {
	return "Invalid Auth ID"
}

func (e *invalidAuthIDError) ShouldPrintUsage() bool {
	return true
}
