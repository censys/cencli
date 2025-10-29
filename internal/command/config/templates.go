package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config/templates"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/ui/form"
)

// getDataDir returns the data directory path using the same logic as main.go
func getDataDir() (string, error) {
	if override := os.Getenv("CENCLI_DATA_DIR"); override != "" {
		return override, nil
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".config", "cencli"), nil
}

type templatesCommand struct {
	*command.BaseCommand
}

var _ command.Command = (*templatesCommand)(nil)

func newTemplatesCommand(cmdContext *command.Context) *templatesCommand {
	cmd := &templatesCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
	return cmd
}

func (c *templatesCommand) Use() string   { return "templates" }
func (c *templatesCommand) Short() string { return "Manage output templates" }
func (c *templatesCommand) Long() string  { return "View and manage your output templates." }

func (c *templatesCommand) Init() error {
	return c.AddSubCommands(
		newResetTemplatesCommand(c.Context),
	)
}

func (c *templatesCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *templatesCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *templatesCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return cenclierrors.NewCencliError(cmd.Help())
}

type resetTemplatesCommand struct {
	*command.BaseCommand
	yes        bool
	accessible bool
	flags      resetTemplatesCommandFlags
}

type resetTemplatesCommandFlags struct {
	yes        flags.BoolFlag
	accessible flags.BoolFlag
}

var _ command.Command = (*resetTemplatesCommand)(nil)

func newResetTemplatesCommand(cmdContext *command.Context) *resetTemplatesCommand {
	cmd := &resetTemplatesCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
	return cmd
}

func (c *resetTemplatesCommand) Use() string { return "reset" }
func (c *resetTemplatesCommand) Short() string {
	return "Reset templates to the latest defaults"
}

func (c *resetTemplatesCommand) Long() string {
	return "Replace your existing templates with the latest default templates. This will overwrite any customizations you have made and fix any broken template files."
}

func (c *resetTemplatesCommand) Init() error {
	c.flags.yes = flags.NewBoolFlag(
		c.Flags(),
		"yes",
		"y",
		false,
		"skip confirmation prompt",
	)
	c.flags.accessible = flags.NewBoolFlag(
		c.Flags(),
		"accessible",
		"a",
		false,
		"enable accessible mode (non-redrawing)",
	)
	return nil
}

func (c *resetTemplatesCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *resetTemplatesCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.yes, err = c.flags.yes.Value()
	if err != nil {
		return err
	}
	c.accessible, err = c.flags.accessible.Value()
	if err != nil {
		return err
	}
	return nil
}

func (c *resetTemplatesCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// Get the data directory
	dataDir, err := getDataDir()
	if err != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to get data directory: %w", err))
	}

	// Get list of default templates for confirmation
	defaultTemplateNames, err := templates.ListDefaultTemplates()
	if err != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to list default templates: %w", err))
	}

	if len(defaultTemplateNames) == 0 {
		formatter.Printf(formatter.Stdout, "No default templates found to reset.\n")
		return nil
	}

	// Show confirmation unless --yes flag is set
	if !c.yes {
		confirmed, err := c.confirmReset(cmd, defaultTemplateNames)
		if err != nil {
			return err
		}
		if !confirmed {
			formatter.Printf(formatter.Stdout, "Reset cancelled.\n")
			return nil
		}
	}

	// Perform the reset - this will overwrite all templates and ignore errors
	reset, err := templates.ResetTemplates(dataDir)
	if err != nil {
		return cenclierrors.NewCencliError(err)
	}

	// Report results
	if len(reset) > 0 && !c.Config().Quiet {
		formatter.Printf(formatter.Stdout, "✅ Successfully reset %d template(s):\n", len(reset))
		for _, name := range reset {
			formatter.Printf(formatter.Stdout, "   - %s\n", name)
		}
	}

	if len(reset) == 0 {
		formatter.Printf(formatter.Stderr, "⚠️  No templates were reset. Check that default templates are available.\n")
	}

	return nil
}

func (c *resetTemplatesCommand) confirmReset(cmd *cobra.Command, templates []string) (bool, cenclierrors.CencliError) {
	description := "The following templates will be reset to the latest defaults:\n"
	for _, name := range templates {
		description += fmt.Sprintf("  • %s\n", name)
	}
	description += "\n⚠️  This will overwrite any customizations you have made and fix any broken template files."

	var confirmed bool

	f := form.NewForm(
		huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Reset Templates").
					Description(description).
					Affirmative("Yes, reset templates").
					Negative("Cancel").
					Value(&confirmed),
			),
		),
		form.WithAccessible(c.accessible),
	)

	err := f.RunWithContext(cmd.Context())
	if err != nil {
		if errors.Is(err, form.ErrUserAborted) {
			return false, nil
		}
		return false, cenclierrors.NewCencliError(err)
	}

	return confirmed, nil
}
