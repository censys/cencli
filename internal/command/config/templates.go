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
		newMigrateTemplatesCommand(c.Context),
	)
}

func (c *templatesCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *templatesCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *templatesCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return cenclierrors.NewCencliError(cmd.Help())
}

type migrateTemplatesCommand struct {
	*command.BaseCommand
	yes        bool
	accessible bool
	flags      migrateTemplatesCommandFlags
}

type migrateTemplatesCommandFlags struct {
	yes        flags.BoolFlag
	accessible flags.BoolFlag
}

var _ command.Command = (*migrateTemplatesCommand)(nil)

func newMigrateTemplatesCommand(cmdContext *command.Context) *migrateTemplatesCommand {
	cmd := &migrateTemplatesCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
	return cmd
}

func (c *migrateTemplatesCommand) Use() string { return "migrate" }
func (c *migrateTemplatesCommand) Short() string {
	return "Migrate templates to the latest defaults"
}

func (c *migrateTemplatesCommand) Long() string {
	return "Replace your existing templates with the latest default templates. This will overwrite any customizations you have made."
}

func (c *migrateTemplatesCommand) Init() error {
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

func (c *migrateTemplatesCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *migrateTemplatesCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
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

func (c *migrateTemplatesCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// Get the data directory
	dataDir, err := getDataDir()
	if err != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to get data directory: %w", err))
	}
	templatesDir := templates.GetTemplatesDir(dataDir)

	// Check if templates directory exists
	if _, err := os.Stat(templatesDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			formatter.Printf(formatter.Stderr, "No templates directory found at %s\n", templatesDir)
			return cenclierrors.NewCencliError(fmt.Errorf("templates directory does not exist"))
		}
		return cenclierrors.NewCencliError(fmt.Errorf("failed to check templates directory: %w", err))
	}

	// Get list of default templates
	defaultTemplateNames, err := templates.ListDefaultTemplates()
	if err != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to list default templates: %w", err))
	}

	if len(defaultTemplateNames) == 0 {
		formatter.Printf(formatter.Stdout, "No default templates found to migrate.\n")
		return nil
	}

	// Check which templates exist in the user's directory
	var existingTemplates []string
	for _, templateName := range defaultTemplateNames {
		templatePath := filepath.Join(templatesDir, templateName)
		if _, err := os.Stat(templatePath); err == nil {
			existingTemplates = append(existingTemplates, templateName)
		}
	}

	if len(existingTemplates) == 0 {
		formatter.Printf(formatter.Stdout, "No existing templates found to migrate.\n")
		return nil
	}

	// Show confirmation unless --yes flag is set
	if !c.yes {
		confirmed, err := c.confirmMigration(cmd, existingTemplates)
		if err != nil {
			return err
		}
		if !confirmed {
			formatter.Printf(formatter.Stdout, "Migration cancelled.\n")
			return nil
		}
	}

	// Perform the migration
	var migrated []string
	var failed []string

	for _, templateName := range defaultTemplateNames {
		if err := templates.CopyDefaultTemplate(templateName, templatesDir); err != nil {
			failed = append(failed, templateName)
			if !c.Config().Quiet {
				formatter.Printf(formatter.Stderr, "❌ Failed to migrate %s: %v\n", templateName, err)
			}
		} else {
			migrated = append(migrated, templateName)
		}
	}

	// Report results
	if len(migrated) > 0 && !c.Config().Quiet {
		formatter.Printf(formatter.Stdout, "✅ Successfully migrated %d template(s):\n", len(migrated))
		for _, name := range migrated {
			formatter.Printf(formatter.Stdout, "   - %s\n", name)
		}
	}

	if len(failed) > 0 {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to migrate %d template(s)", len(failed)))
	}

	return nil
}

func (c *migrateTemplatesCommand) confirmMigration(cmd *cobra.Command, templates []string) (bool, cenclierrors.CencliError) {
	description := "The following templates will be replaced with the latest defaults:\n"
	for _, name := range templates {
		description += fmt.Sprintf("  • %s\n", name)
	}
	description += "\n⚠️  This will overwrite any customizations you have made."

	var confirmed bool

	f := form.NewForm(
		huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Migrate Templates").
					Description(description).
					Affirmative("Yes, replace templates").
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
