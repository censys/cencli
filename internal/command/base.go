package command

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	applog "github.com/censys/cencli/internal/pkg/log"
	"github.com/censys/cencli/internal/pkg/styles"
)

// BaseCommand is what each Command implementation must embed.
// This allows new Commands to not have to worry about dependency injection.
// BaseCommand intentionally does not implement the Command interface,
// to "force" subcommands to implement required methods.
type BaseCommand struct {
	*Context
	rootCmd *cobra.Command
}

func NewBaseCommand(cmdContext *Context) *BaseCommand {
	return &BaseCommand{Context: cmdContext, rootCmd: &cobra.Command{}}
}

func (b *BaseCommand) command() *cobra.Command { return b.rootCmd }

func (b *BaseCommand) Flags() *pflag.FlagSet { return b.rootCmd.Flags() }

func (b *BaseCommand) PersistentFlags() *pflag.FlagSet { return b.rootCmd.PersistentFlags() }

func (b *BaseCommand) InheritedFlags() *pflag.FlagSet { return b.rootCmd.InheritedFlags() }

func (b *BaseCommand) AddSubCommands(cmds ...Command) error {
	for _, cmd := range cmds {
		c, err := toCobra(cmd)
		if err != nil {
			return fmt.Errorf("failed to build command %s: %w", cmd.Use(), err)
		}
		b.rootCmd.AddCommand(c)

		if err := applyOutputFormatDefaultsRecursive(c, cmd); err != nil {
			return fmt.Errorf("failed to apply output format defaults for %s: %w", cmd.Use(), err)
		}
	}
	return nil
}

func (b *BaseCommand) PostRun(cmd *cobra.Command, args []string) cenclierrors.CencliError { return nil }

func (b *BaseCommand) HelpFunc(cmd *cobra.Command, examples []string) {
	if !formatter.StdoutIsTTY() {
		restore := styles.TemporarilyDisableStyles()
		defer restore()
	}
	formatter.Println(formatter.Stdout, helpTemplate(cmd, examples))
}

func (b *BaseCommand) UsageFunc(cmd *cobra.Command, examples []string) {
	if !formatter.StderrIsTTY() {
		restore := styles.TemporarilyDisableStyles()
		defer restore()
	}
	formatter.Println(formatter.Stderr, usageTemplate(cmd, examples))
}

func (b *BaseCommand) Init() error { return nil }

func (b *BaseCommand) Examples() []string { return []string{} }

func (b *BaseCommand) Long() string { return "" }

func (b *BaseCommand) DefaultOutputType() OutputType {
	return OutputTypeData
}

func (b *BaseCommand) SupportedOutputTypes() []OutputType {
	return []OutputType{OutputTypeData}
}

func (b *BaseCommand) SupportsStreaming() bool {
	return false
}

func (b *BaseCommand) RenderShort() cenclierrors.CencliError {
	// this should theoretically never happen, since the command should not be executed if the output format is not supported
	return cenclierrors.NewCencliError(fmt.Errorf("short output not supported for this command"))
}

func (b *BaseCommand) RenderTemplate() cenclierrors.CencliError {
	// this should theoretically never happen, since the command should not be executed if the output format is not supported
	return cenclierrors.NewCencliError(fmt.Errorf("template output not supported for this command"))
}

func (b *BaseCommand) init(cmd Command) {
	b.rootCmd.PersistentPreRunE = func(cobraCmd *cobra.Command, args []string) error {
		// unmarshal the config so it is available to the command
		if err := b.Config().Unmarshal(); err != nil {
			return err
		}

		// Update color settings after config is re-unmarshaled to respect command-line flags
		b.Context.updateColorSettings()

		// Validate streaming mode for conflicts and support
		if err := validateStreamingMode(cobraCmd, cmd, b.config.Streaming); err != nil {
			return err
		}

		// special case for output format
		// we need to manually inspect the command's flags to see if the user explicitly set the output format
		// since there are some shenanigans with the flag binding and the default value being set after unmarshal
		b.config.OutputFormat = getOutputFormatValue(cobraCmd, cmd, b.config.OutputFormat)

		// Validate output format before command execution
		if err := validateOutputFormat(b.config.OutputFormat, cmd); err != nil {
			return err
		}

		// set the logger
		b.SetLogger(applog.New(b.Config().Debug, nil))
		return nil
	}
}
