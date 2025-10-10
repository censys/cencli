package completion

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/formatter"
	"go.uber.org/mock/gomock"
)

func TestCompletion_InvalidShell(t *testing.T) {
	var stdout, stderr bytes.Buffer
	formatter.Stdout = &stdout
	formatter.Stderr = &stderr

	viper.Reset()
	cfg, err := config.New(t.TempDir())
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := command.NewCommandContext(cfg, storemocks.NewMockStore(ctrl))

	root, cerr := command.RootCommandToCobra(NewCompletionCommand(ctx))
	require.NoError(t, cerr)

	root.SetArgs([]string{"invalid-shell"})
	execErr := root.Execute()
	require.Error(t, execErr)
}

func TestCompletion_BashAndZsh(t *testing.T) {
	var stdout, stderr bytes.Buffer
	formatter.Stdout = &stdout
	formatter.Stderr = &stderr

	viper.Reset()
	cfg, err := config.New(t.TempDir())
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := command.NewCommandContext(cfg, storemocks.NewMockStore(ctrl))

	root := &cobra.Command{Use: "root"}
	comp, err2 := command.RootCommandToCobra(NewCompletionCommand(ctx))
	require.NoError(t, err2)
	root.AddCommand(comp)

	root.SetArgs([]string{"completion", "bash"})
	require.NoError(t, root.Execute())
	// Cobra bash completion should output a script with complete invocations
	require.Contains(t, stdout.String(), "complete -o")
	stdout.Reset()

	root.SetArgs([]string{"completion", "zsh"})
	require.NoError(t, root.Execute())
	require.Contains(t, stdout.String(), "compdef")
}

func TestCompletion_FishAndPowershell(t *testing.T) {
	var stdout, stderr bytes.Buffer
	formatter.Stdout = &stdout
	formatter.Stderr = &stderr

	viper.Reset()
	cfg, err := config.New(t.TempDir())
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := command.NewCommandContext(cfg, storemocks.NewMockStore(ctrl))

	root := &cobra.Command{Use: "root"}
	comp, err2 := command.RootCommandToCobra(NewCompletionCommand(ctx))
	require.NoError(t, err2)
	root.AddCommand(comp)

	root.SetArgs([]string{"completion", "fish"})
	require.NoError(t, root.Execute())
	require.Contains(t, stdout.String(), "complete ")
	stdout.Reset()

	root.SetArgs([]string{"completion", "powershell"})
	require.NoError(t, root.Execute())
	require.Contains(t, stdout.String(), "Register-ArgumentCompleter")
}
