package command

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
)

// Command is an interface that all CLI commands must implement.
// This allows new Commands to not have to worry about cobra specifics.
// Everything that implements Command MUST embed BaseCommand.
type Command interface {
	// AddSubCommand adds one or more subcommands to the command.
	// Should not be implemented.
	AddSubCommands(cmds ...Command) error
	// Use returns the name of the command as it will be used in the CLI.
	// Must be implemented.
	Use() string
	// Short returns the short description of the command.
	// Must be implemented.
	Short() string
	// Long returns the long description of the command.
	// Not required to implement.
	Long() string
	// Examples returns the examples for the command.
	// Not required to implement.
	Examples() []string
	// Args returns the positional argument function for the command.
	// Must be implemented.
	Args() PositionalArgs
	// PreRun executes before the main command logic.
	// Must be implemented, since many commands can benefit from it.
	// If you really don't need it, just return nil.
	PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError
	// Run executes the main command logic.
	// Must be implemented.
	Run(cmd *cobra.Command, args []string) cenclierrors.CencliError
	// PostRun executes after the main command logic.
	// Not required to implement.
	PostRun(cmd *cobra.Command, args []string) cenclierrors.CencliError
	// HelpFunc allows the command to customize the help function.
	// Be careful with this, as it will override the default help function,
	// which will contain all necessary information by default.
	// Not required to implement.
	HelpFunc(cmd *cobra.Command, examples []string)
	// UsageFunc defines the usage function for the command.
	// Not required to implement. Allows customization of the usage output for the command.
	UsageFunc(cmd *cobra.Command, examples []string)
	// Init will run before the underlying cobra command is initialized.
	// This can be useful for binding persistent flags, etc.
	// You should NOT modify the underlying cobra command in this function.
	// If you need to, the Command interface should be expanded to accomplish
	// what you are trying to do.
	// Not required to implement.
	Init() error
	// Flags returns the underlying flag set for the command.
	// Used for modifying a command's flags.
	// Should not be implemented.
	Flags() *pflag.FlagSet
	// PersistentFlags returns the underlying persistent flag set for the command.
	// Used for modifying persistent flags on the command.
	// Should not be implemented.
	PersistentFlags() *pflag.FlagSet
	// init is used to internally initialize the command.
	// For example, it will set the persistent pre-run function to unmarshal the config
	// so it is available to the command.
	// Also serves as a guard to prevent a Command from being implemented without embedding BaseCommand.
	init(Command)
	// command returns the underlying cobra command.
	// This is not exposed in an effort to prevent manual
	// modification of the cobra command.
	command() *cobra.Command
}

// toCobra converts a domain Command to a Cobra command.
// This should only need to be called outside of this package
// when building the root command.
func toCobra(cmd Command) (*cobra.Command, error) {
	cobraCmd := cmd.command()
	cobraCmd.SilenceUsage = true
	cobraCmd.SilenceErrors = true

	cobraCmd.SetHelpFunc(func(runtimeCmd *cobra.Command, args []string) {
		cmd.HelpFunc(runtimeCmd, cmd.Examples())
	})

	cobraCmd.SetUsageFunc(func(runtimeCmd *cobra.Command) error {
		cmd.UsageFunc(runtimeCmd, cmd.Examples())
		return nil
	})

	if err := cmd.Init(); err != nil {
		return nil, fmt.Errorf("failed during Init(): %w", err)
	}
	cmd.init(cmd)

	cobraCmd.Use = cmd.Use()
	if cobraCmd.Use == "" {
		return nil, fmt.Errorf("Use() is empty")
	}
	cobraCmd.Short = cmd.Short()
	if cobraCmd.Short == "" {
		return nil, fmt.Errorf("Short() is empty")
	}
	cobraCmd.Long = cmd.Long()

	if args := cmd.Args(); args == nil {
		return nil, fmt.Errorf("Args() is nil")
	}
	cobraCmd.Args = func(runtimeCmd *cobra.Command, args []string) error {
		return cmd.Args()(runtimeCmd, args)
	}

	cobraCmd.PreRunE = func(c *cobra.Command, args []string) error {
		return cmd.PreRun(c, args)
	}
	cobraCmd.RunE = func(c *cobra.Command, args []string) error {
		return cmd.Run(c, args)
	}
	cobraCmd.PostRunE = func(c *cobra.Command, args []string) error {
		return cmd.PostRun(c, args)
	}

	cobraCmd.SetOut(formatter.Stdout)
	cobraCmd.SetErr(formatter.Stderr)
	return cobraCmd, nil
}

// RootCommandToCobra is essentially toCobra(), with different
// naming to prevent non-root commands from using it.
// Useful for building the top-level command and for testing.
func RootCommandToCobra(root Command) (*cobra.Command, cenclierrors.CencliError) {
	cobraCmd, err := toCobra(root)
	if err != nil {
		return nil, cenclierrors.NewCencliError(err)
	}
	return cobraCmd, nil
}
