package root

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/command"
	aggregatecmd "github.com/censys/cencli/internal/command/aggregate"
	censeyecmd "github.com/censys/cencli/internal/command/censeye"
	completioncmd "github.com/censys/cencli/internal/command/completion"
	configcmd "github.com/censys/cencli/internal/command/config"
	creditscmd "github.com/censys/cencli/internal/command/credits"
	historycmd "github.com/censys/cencli/internal/command/history"
	searchcmd "github.com/censys/cencli/internal/command/search"
	versioncmd "github.com/censys/cencli/internal/command/versioncmd"
	"github.com/censys/cencli/internal/command/view"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/censyscopy"
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

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort}
}

func (c *Command) Init() error {
	if err := config.BindGlobalFlags(c.PersistentFlags(), c.Config()); err != nil {
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
		creditscmd.NewCreditsCommand(c.Context),
	)
}

func (c *Command) HelpFunc(cmd *cobra.Command, _ []string) {
	if !formatter.StdoutIsTTY() {
		restore := styles.TemporarilyDisableStyles()
		defer restore()
	}
	formatter.Println(formatter.Stdout, rootHelpFunc(formatter.Stdout, cmd))
}

func (c *Command) UsageFunc(cmd *cobra.Command, _ []string) {
	if !formatter.StderrIsTTY() {
		restore := styles.TemporarilyDisableStyles()
		defer restore()
	}
	formatter.Println(formatter.Stderr, rootHelpFunc(formatter.Stderr, cmd))
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
// Takes a writer to determine if the output is a TTY for link rendering.
func rootHelpFunc(w io.Writer, cmd *cobra.Command) string {
	var b strings.Builder

	// Welcome message
	b.WriteString(styles.GlobalStyles.Signature.Render("censys"))
	b.WriteString(styles.GlobalStyles.Primary.Render(" - "+censyscopy.Title()) + "\n\n")
	b.WriteString(styles.GlobalStyles.Primary.Render(censyscopy.Description()) + "\n\n")
	b.WriteString(styles.GlobalStyles.Primary.Render("Get started by exploring the available commands below.") + "\n\n")

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
					styles.GlobalStyles.Secondary.Render(c.Short))
			}
		}
		b.WriteRune('\n')
	}

	if cmd.HasAvailableSubCommands() {
		b.WriteString(styles.GlobalStyles.Comment.Render(
			fmt.Sprintf("Run \"%s [command] --help\" for help with a specific command.", cmd.CommandPath()),
		))
		b.WriteRune('\n')
	}
	b.WriteRune('\n')
	b.WriteString(styles.GlobalStyles.Primary.Render(censyscopy.DocumentationCLI(w)))
	b.WriteRune('\n')
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
				"view platform.censys.io:80 --output-format short",
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
