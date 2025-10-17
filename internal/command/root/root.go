package root

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	aggregatecmd "github.com/censys/cencli/internal/command/aggregate"
	censeyecmd "github.com/censys/cencli/internal/command/censeye"
	completioncmd "github.com/censys/cencli/internal/command/completion"
	configcmd "github.com/censys/cencli/internal/command/config"
	historycmd "github.com/censys/cencli/internal/command/history"
	searchcmd "github.com/censys/cencli/internal/command/search"
	versioncmd "github.com/censys/cencli/internal/command/versioncmd"
	"github.com/censys/cencli/internal/command/view"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/tape"
)

type Command struct {
	*command.BaseCommand
}

var _ command.Command = (*Command)(nil)

func NewRootCommand(cmdContext *command.Context) *Command {
	return &Command{BaseCommand: command.NewBaseCommand(cmdContext)}
}

func (c *Command) Use() string {
	return "censys"
}

func (c *Command) Short() string {
	return "Censys CLI"
}

func (c *Command) Args() command.PositionalArgs {
	return command.ExactArgs(0)
}

func (c *Command) Init() error {
	if err := config.BindGlobalFlags(c.PersistentFlags()); err != nil {
		return fmt.Errorf("failed to bind global flags: %w", err)
	}

	return c.AddSubCommands(
		view.NewViewCommand(c.Context),
		configcmd.NewConfigCommand(c.Context),
		versioncmd.NewVersionCommand(c.Context),
		completioncmd.NewCompletionCommand(c.Context),
		historycmd.NewHistoryCommand(c.Context),
		searchcmd.NewSearchCommand(c.Context),
		aggregatecmd.NewAggregateCommand(c.Context),
		censeyecmd.NewCenseyeCommand(c.Context),
	)
}

func (c *Command) HelpFunc(cmd *cobra.Command, _ []string) {
	formatter.Println(formatter.Stdout, rootHelpFunc(cmd))
}

func (c *Command) UsageFunc(cmd *cobra.Command, _ []string) {
	formatter.Println(formatter.Stderr, rootHelpFunc(cmd))
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError { return nil }

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// Root command shows help when run without subcommands
	if err := cmd.Help(); err != nil {
		return cenclierrors.NewCencliError(err)
	}
	return nil
}

// rootHelpFunc renders a special welcome message and command list for the root command.
func rootHelpFunc(cmd *cobra.Command) string {
	var b strings.Builder

	// Welcome message
	b.WriteString(styles.GlobalStyles.Signature.Render("censys - The Censys CLI") + "\n\n")
	b.WriteString(styles.GlobalStyles.Secondary.Render("The Censys CLI tool helps you interact with Censys services from the command line.") + "\n\n")
	b.WriteString(styles.GlobalStyles.Secondary.Render("Get started by exploring the available commands below.") + "\n\n")

	// Available Commands section
	if cmd.HasAvailableSubCommands() {
		b.WriteString(styles.GlobalStyles.Info.Render("Available Commands:") + "\n")
		cmds := cmd.Commands()
		for _, c := range cmds {
			if c.IsAvailableCommand() {
				// Format with padding first, then apply color
				name := fmt.Sprintf("%-*s", cmd.NamePadding(), c.Name())
				fmt.Fprintf(&b, "  %s %s\n",
					styles.GlobalStyles.Signature.Render(name),
					styles.GlobalStyles.Tertiary.Render(c.Short))
			}
		}
		b.WriteString("\n")
	}

	if cmd.HasAvailableSubCommands() {
		b.WriteString(styles.GlobalStyles.Secondary.Render("Use \""))
		b.WriteString(styles.GlobalStyles.Signature.Render(cmd.CommandPath() + " [command] --help"))
		b.WriteString(styles.GlobalStyles.Secondary.Render("\" for more information about a specific command.") + "\n")
	}

	return b.String()
}

func (*Command) Tapes(recorder *tape.Recorder) []tape.Tape {
	return []tape.Tape{
		tape.NewTape("cencli",
			tape.DefaultTapeConfig(),
			// simple view
			recorder.Type(
				"view 8.8.8.8",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
			// view with template output
			recorder.Type(
				"view platform.censys.io:80 --short",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
			// search with interactive tree
			recorder.Type(
				"search 'host.services.protocol=SSH' -O tree",
				tape.WithSleepAfter(3),
			),
			recorder.Press("j", 2), // down 2
			recorder.Press("l", 1), // right 1
			recorder.Press("j", 1), // down 1
			recorder.Press("l", 1), // right 1
			recorder.Press("j", 5), // down 5
			recorder.Press("l", 1), // right 1
			recorder.Sleep(1),
			recorder.Press("q", 1), // quit
			recorder.Clear(),
			recorder.Type(
				"aggregate 'host.services.port=22' host.services.protocol -i",
				tape.WithSleepAfter(2),
			),
			recorder.Press("j", 3),
		),
	}
}
