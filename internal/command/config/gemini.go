package config

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/ui/form"
	"github.com/censys/cencli/internal/store"
)

type geminiCommand struct {
	*command.BaseCommand
}

var _ command.Command = (*geminiCommand)(nil)

func newGeminiCommand(cmdContext *command.Context) *geminiCommand {
	cmd := &geminiCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
	return cmd
}

func (c *geminiCommand) Use() string   { return "gemini" }
func (c *geminiCommand) Short() string { return "Manage Gemini API key for chart generation" }
func (c *geminiCommand) Long() string {
	return "View and manage your Gemini API key used for AI-powered chart generation."
}

func (c *geminiCommand) Init() error {
	return c.AddSubCommands(
		newSetGeminiCommand(c.Context),
		newDeleteGeminiCommand(c.Context),
	)
}

func (c *geminiCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }
func (c *geminiCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *geminiCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	value, err := c.Store().GetLastUsedGlobalByName(cmd.Context(), config.GeminiAPIKeyGlobalName)
	if err != nil {
		if errors.Is(err, store.ErrGlobalNotFound) {
			formatter.Printf(formatter.Stdout, "No Gemini API key configured. Use `%s` to add one.\n", cmd.CommandPath()+" add")
			return nil
		}
		return cenclierrors.NewCencliError(fmt.Errorf("failed to get Gemini API key: %w", err))
	}

	// Show masked key
	maskedKey := maskAPIKey(value.Value)
	formatter.Printf(formatter.Stdout, "Gemini API key: %s\n", maskedKey)
	formatter.Printf(formatter.Stdout, "Description: %s\n", value.Description)
	formatter.Printf(formatter.Stdout, "Added: %s\n", value.CreatedAt.Format("2006-01-02 15:04:05"))
	return nil
}

// maskAPIKey masks all but the last 4 characters of the API key
func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

// setGeminiCommand sets the Gemini API key
type setGeminiCommand struct {
	*command.BaseCommand
	accessible bool
	flags      setGeminiCommandFlags
}

type setGeminiCommandFlags struct {
	accessible flags.BoolFlag
	value      flags.StringFlag
	name       flags.StringFlag
}

var _ command.Command = (*setGeminiCommand)(nil)

func newSetGeminiCommand(cmdContext *command.Context) *setGeminiCommand {
	cmd := &setGeminiCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
	return cmd
}

func (c *setGeminiCommand) Use() string   { return "add" }
func (c *setGeminiCommand) Short() string { return "Set the Gemini API key" }
func (c *setGeminiCommand) Long() string {
	return "Set the Gemini API key used for AI-powered chart generation. Get your API key from https://aistudio.google.com/apikey"
}

func (c *setGeminiCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *setGeminiCommand) Init() error {
	c.flags.accessible = flags.NewBoolFlag(
		c.Flags(),
		"accessible",
		"a",
		false,
		"enable accessible mode (non-redrawing)",
	)
	c.flags.value = flags.NewStringFlag(
		c.Flags(),
		false,
		"value",
		"",
		"",
		"Gemini API key value (non-interactive)",
	)
	c.flags.name = flags.NewStringFlag(
		c.Flags(),
		false,
		"name",
		"n",
		"default",
		"friendly name/description for this key",
	)
	return nil
}

func (c *setGeminiCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.accessible, err = c.flags.accessible.Value()
	if err != nil {
		return err
	}
	return nil
}

func (c *setGeminiCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// Non-interactive path when value provided
	valueStr, _ := c.flags.value.Value()
	if valueStr != "" {
		name, nerr := c.flags.name.Value()
		if nerr != nil {
			return nerr
		}
		if name == "" {
			name = "default"
		}

		// Delete existing key if present
		existing, _ := c.Store().GetValuesForGlobal(cmd.Context(), config.GeminiAPIKeyGlobalName)
		for _, v := range existing {
			_, _ = c.Store().DeleteValueForGlobal(cmd.Context(), v.ID)
		}

		_, aerr := c.Store().AddValueForGlobal(cmd.Context(), config.GeminiAPIKeyGlobalName, name, valueStr)
		if aerr != nil {
			return cenclierrors.NewCencliError(fmt.Errorf("failed to set Gemini API key: %w", aerr))
		}

		formatter.Printf(formatter.Stdout, "✅ Gemini API key configured [%s]\n", name)
		return nil
	}

	// Interactive mode
	var value string
	var name string

	f := form.NewForm(
		huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					EchoMode(huh.EchoModePassword).
					Title("Enter your Gemini API key").
					Description("Get your API key from https://aistudio.google.com/apikey").
					Value(&value).
					Validate(form.NonEmpty("API key cannot be empty")),
				huh.NewInput().
					Title("Enter a name for this key").
					Description("A friendly name to help identify this key").
					Placeholder("default").
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

	if name == "" {
		name = "default"
	}

	// Delete existing key if present
	existing, _ := c.Store().GetValuesForGlobal(cmd.Context(), config.GeminiAPIKeyGlobalName)
	for _, v := range existing {
		_, _ = c.Store().DeleteValueForGlobal(cmd.Context(), v.ID)
	}

	_, serr := c.Store().AddValueForGlobal(cmd.Context(), config.GeminiAPIKeyGlobalName, name, value)
	if serr != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to set Gemini API key: %w", serr))
	}

	formatter.Printf(formatter.Stdout, "✅ Gemini API key configured [%s]\n", name)
	return nil
}

// deleteGeminiCommand deletes the Gemini API key
type deleteGeminiCommand struct {
	*command.BaseCommand
}

var _ command.Command = (*deleteGeminiCommand)(nil)

func newDeleteGeminiCommand(ctx *command.Context) *deleteGeminiCommand {
	return &deleteGeminiCommand{BaseCommand: command.NewBaseCommand(ctx)}
}

func (c *deleteGeminiCommand) Use() string   { return "delete" }
func (c *deleteGeminiCommand) Short() string { return "Delete the Gemini API key" }

func (c *deleteGeminiCommand) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *deleteGeminiCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return nil
}

func (c *deleteGeminiCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	values, err := c.Store().GetValuesForGlobal(cmd.Context(), config.GeminiAPIKeyGlobalName)
	if err != nil {
		if errors.Is(err, store.ErrGlobalNotFound) {
			formatter.Printf(formatter.Stdout, "No Gemini API key configured.\n")
			return nil
		}
		return cenclierrors.NewCencliError(fmt.Errorf("failed to get Gemini API key: %w", err))
	}

	if len(values) == 0 {
		formatter.Printf(formatter.Stdout, "No Gemini API key configured.\n")
		return nil
	}

	for _, v := range values {
		_, derr := c.Store().DeleteValueForGlobal(cmd.Context(), v.ID)
		if derr != nil {
			return cenclierrors.NewCencliError(fmt.Errorf("failed to delete Gemini API key: %w", derr))
		}
	}

	formatter.Printf(formatter.Stdout, "✅ Gemini API key deleted\n")
	return nil
}
