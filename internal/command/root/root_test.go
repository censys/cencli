package root

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/formatter"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRegexp.ReplaceAllString(s, "") }

func TestRoot_HelpListsCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	formatter.Stdout = &stdout
	formatter.Stderr = &stderr

	// Ensure clean Viper state per test to avoid leaking template paths
	viper.Reset()
	cfg, err := config.New(t.TempDir())
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := command.NewCommandContext(cfg, storemocks.NewMockStore(ctrl))

	root, cerr := command.RootCommandToCobra(NewRootCommand(ctx))
	require.NoError(t, cerr)

	root.SetArgs([]string{"--help"})
	require.NoError(t, root.Execute())

	out := stripANSI(stdout.String())
	require.Contains(t, out, "Available Commands:")
	// Assert essential commands present
	require.Contains(t, out, "view")
	require.Contains(t, out, "search")
}

func TestRootHelpFunc_WithSubcommands(t *testing.T) {
	// Build a real root command through our adapter to ensure available subcommands are wired
	viper.Reset()
	cfg, err := config.New(t.TempDir())
	require.NoError(t, err)
	ctx := command.NewCommandContext(cfg, nil)
	root := NewRootCommand(ctx)
	cobraCmd, cErr := command.RootCommandToCobra(root)
	if cErr != nil {
		t.Fatalf("RootCommandToCobra: %v", cErr)
	}

	out := stripANSI(rootHelpFunc(formatter.Stdout, cobraCmd))

	if !strings.Contains(out, "censys - The Censys CLI") {
		t.Fatalf("expected welcome header in help output, got: %s", out)
	}
	// Ensure key subcommands appear by name
	for _, name := range []string{"view", "config", "search", "aggregate", "version"} {
		if !strings.Contains(out, name) {
			t.Fatalf("expected subcommand %q listed, got: %s", name, out)
		}
	}
}

func TestRootHelpFunc_NoSubcommands(t *testing.T) {
	// Simulate no subcommands by constructing a minimal cobra command
	cobraCmd, _ := command.RootCommandToCobra(&Command{BaseCommand: command.NewBaseCommand(command.NewCommandContext(&config.Config{}, nil))})
	// Remove children
	cobraCmd.ResetCommands()
	out := stripANSI(rootHelpFunc(formatter.Stdout, cobraCmd))
	if strings.Contains(out, "Available Commands:") {
		t.Fatalf("did not expect Available Commands section when none exist, got: %s", out)
	}
}
