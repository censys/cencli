package config

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/ui/form"
	"github.com/censys/cencli/internal/store"
)

type addOrganizationIDCommand struct {
	*command.BaseCommand
	accessible bool
	flags      addOrganizationIDCommandFlags
}

type addOrganizationIDCommandFlags struct {
	accessible flags.BoolFlag
	name       flags.StringFlag
	value      flags.StringFlag
	valueFile  flags.FileFlag
	activate   flags.BoolFlag
}

var _ command.Command = (*addOrganizationIDCommand)(nil)

func newAddOrganizationIDCommand(cmdContext *command.Context) *addOrganizationIDCommand {
	cmd := &addOrganizationIDCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
	return cmd
}

func (c *addOrganizationIDCommand) Use() string   { return "add" }
func (c *addOrganizationIDCommand) Short() string { return "Add a new organization ID value" }
func (c *addOrganizationIDCommand) Long() string {
	return "Add a new organization ID value that will be used across API requests."
}
func (c *addOrganizationIDCommand) DisableTimeout() bool { return true }

func (c *addOrganizationIDCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *addOrganizationIDCommand) Init() error {
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
		"friendly name/description associated with this organization ID (default: ci)",
	)
	c.flags.value = flags.NewStringFlag(
		c.Flags(),
		false,
		"value",
		"",
		"",
		"organization ID value (UUID; non-interactive)",
	)
	c.flags.valueFile = flags.NewFileFlag(
		c.Flags(),
		false,
		"value-file",
		"",
		"read the organization ID value from a file or '-' for stdin (non-interactive)",
	)
	c.flags.activate = flags.NewBoolFlag(
		c.Flags(),
		"activate",
		"",
		true,
		"mark the added organization ID as active (default: true)",
	)
	return nil
}

func (c *addOrganizationIDCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.accessible, err = c.flags.accessible.Value()
	if err != nil {
		return err
	}
	return nil
}

func (c *addOrganizationIDCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// Non-interactive path when value or value-file provided
	filePath, _ := c.flags.valueFile.Value()
	valueStr, _ := c.flags.value.Value()
	if filePath != "" || valueStr != "" {
		var org string
		var err cenclierrors.CencliError
		if filePath != "" {
			var lines []string
			lines, err = c.flags.valueFile.Lines(cmd)
			if err != nil {
				return err
			}
			for _, ln := range lines {
				if ln != "" {
					org = ln
					break
				}
			}
			if org == "" {
				return cenclierrors.NewCencliError(fmt.Errorf("value-file is empty"))
			}
		} else {
			org = valueStr
		}

		// Validate UUID format
		if _, uerr := uuid.Parse(org); uerr != nil {
			return cenclierrors.NewCencliError(fmt.Errorf("organization ID must be a valid UUID: %w", uerr))
		}

		name, nerr := c.flags.name.Value()
		if nerr != nil {
			return nerr
		}
		if name == "" {
			name = "ci"
		}

		rec, aerr := c.Store().AddValueForGlobal(cmd.Context(), config.OrgIDGlobalName, name, org)
		if aerr != nil {
			return cenclierrors.NewCencliError(fmt.Errorf("failed to add global value: %w", aerr))
		}

		activate, actErr := c.flags.activate.Value()
		if actErr != nil {
			return actErr
		}
		if activate {
			if uerr := c.Store().UpdateGlobalLastUsedAtToNow(cmd.Context(), rec.ID); uerr != nil {
				return cenclierrors.NewCencliError(fmt.Errorf("failed to activate organization ID: %w", uerr))
			}
		}

		formatter.Printf(formatter.Stdout, "✅ Added new organization ID [%s]\n", name)
		return nil
	}

	var value string
	var name string

	f := form.NewForm(
		huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Enter a new organization ID").
					Description("Enter the organization ID value that will be used in API requests").
					Value(&value).
					Validate(form.NonEmptyUUID("organization ID must be a valid UUID")),
				huh.NewInput().
					Title("Enter a name for this organization ID").
					Description("A friendly name to help identify this organization ID").
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

	_, err = c.Store().AddValueForGlobal(cmd.Context(), config.OrgIDGlobalName, name, value)
	if err != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to add global value: %w", err))
	}

	formatter.Printf(formatter.Stdout, "✅ Added new organization ID [%s]\n", name)
	return nil
}

type deleteOrganizationIDCommand struct {
	*command.BaseCommand
}

var _ command.Command = (*deleteOrganizationIDCommand)(nil)

func newDeleteOrganizationIDCommand(ctx *command.Context) *deleteOrganizationIDCommand {
	return &deleteOrganizationIDCommand{BaseCommand: command.NewBaseCommand(ctx)}
}

func (c *deleteOrganizationIDCommand) Use() string          { return "delete <id>" }
func (c *deleteOrganizationIDCommand) Short() string        { return "Delete a stored organization ID" }
func (c *deleteOrganizationIDCommand) DisableTimeout() bool { return true }

func (c *deleteOrganizationIDCommand) Args() command.PositionalArgs { return command.ExactArgs(1) }

func (c *deleteOrganizationIDCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *deleteOrganizationIDCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return newInvalidOrganizationIDError(args[0], "not an integer")
	}

	_, err = c.Store().DeleteValueForGlobal(cmd.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrGlobalNotFound) {
			return newInvalidOrganizationIDError(args[0], "record does not exist")
		}
		return cenclierrors.NewCencliError(fmt.Errorf("failed to delete organization ID value: %w", err))
	}

	formatter.Printf(formatter.Stdout, "✅ Deleted organization ID [%s]\n", args[0])
	return nil
}

type InvalidOrganizationIDError interface {
	cenclierrors.CencliError
}

type invalidOrganizationIDError struct {
	id     string
	reason string
}

type activateOrganizationIDCommand struct {
	*command.BaseCommand
}

var _ command.Command = (*activateOrganizationIDCommand)(nil)

func newActivateOrganizationIDCommand(ctx *command.Context) *activateOrganizationIDCommand {
	return &activateOrganizationIDCommand{BaseCommand: command.NewBaseCommand(ctx)}
}

func (c *activateOrganizationIDCommand) Use() string          { return "activate <id>" }
func (c *activateOrganizationIDCommand) Short() string        { return "Set the active organization ID" }
func (c *activateOrganizationIDCommand) DisableTimeout() bool { return true }

func (c *activateOrganizationIDCommand) Args() command.PositionalArgs { return command.ExactArgs(1) }

func (c *activateOrganizationIDCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *activateOrganizationIDCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return newInvalidOrganizationIDError(args[0], "not an integer")
	}

	err = c.Store().UpdateGlobalLastUsedAtToNow(cmd.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrGlobalNotFound) {
			return newInvalidOrganizationIDError(args[0], "record does not exist")
		}
		return cenclierrors.NewCencliError(fmt.Errorf("failed to activate organization ID: %w", err))
	}

	formatter.Printf(formatter.Stdout, "✅ Activated organization ID [%s]\n", args[0])
	return nil
}

var _ InvalidOrganizationIDError = &invalidOrganizationIDError{}

func newInvalidOrganizationIDError(id string, reason string) InvalidOrganizationIDError {
	return &invalidOrganizationIDError{id: id, reason: reason}
}

func (e *invalidOrganizationIDError) Error() string {
	return fmt.Sprintf("invalid record ID: %s (%s)", e.id, e.reason)
}

func (e *invalidOrganizationIDError) Title() string {
	return "Invalid Record ID"
}

func (e *invalidOrganizationIDError) ShouldPrintUsage() bool {
	return true
}
