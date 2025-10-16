package command

import (
	"bytes"
	"testing"

	"github.com/samber/mo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
	"go.uber.org/mock/gomock"
)

type testCommand struct {
	*BaseCommand
	useFn     func() string
	shortFn   func() string
	longFn    func() string
	argsFn    func() PositionalArgs
	preRunFn  func(cmd *cobra.Command, args []string) cenclierrors.CencliError
	runFn     func(cmd *cobra.Command, args []string) cenclierrors.CencliError
	postRunFn func(cmd *cobra.Command, args []string) cenclierrors.CencliError
	initFn    func(c Command) error
}

var _ Command = &testCommand{}

func (c *testCommand) Use() string          { return c.useFn() }
func (c *testCommand) Short() string        { return c.shortFn() }
func (c *testCommand) Long() string         { return c.longFn() }
func (c *testCommand) Args() PositionalArgs { return c.argsFn() }
func (c *testCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return c.preRunFn(cmd, args)
}

func (c *testCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return c.runFn(cmd, args)
}

func (c *testCommand) PostRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	return c.postRunFn(cmd, args)
}
func (c *testCommand) Init() error { return c.initFn(c) }

func newTestCommand(cmdContext *Context) *testCommand {
	return &testCommand{
		BaseCommand: NewBaseCommand(cmdContext),
		useFn:       func() string { return "test" },
		shortFn:     func() string { return "test" },
		longFn:      func() string { return "test" },
		argsFn:      func() PositionalArgs { return cobra.NoArgs },
		preRunFn:    func(cmd *cobra.Command, args []string) cenclierrors.CencliError { return nil },
		runFn:       func(cmd *cobra.Command, args []string) cenclierrors.CencliError { return nil },
		postRunFn:   func(cmd *cobra.Command, args []string) cenclierrors.CencliError { return nil },
		initFn: func(c Command) error {
			return nil
		},
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		name                string
		cmd                 func(commandContext *Context) Command
		registerErrContains mo.Option[string]
		store               func(ctrl *gomock.Controller) store.Store
		args                []string
		flags               map[string]string
		assert              func(t *testing.T, stdout, stderr string, err error)
	}{
		{
			name: "basic command",
			cmd: func(commandContext *Context) Command {
				command := newTestCommand(commandContext)
				command.runFn = func(cmd *cobra.Command, args []string) cenclierrors.CencliError {
					formatter.Println(formatter.Stdout, "test")
					return nil
				}
				return command
			},
			store: func(ctrl *gomock.Controller) store.Store { return storemocks.NewMockStore(ctrl) },
			args:  []string{},
			assert: func(t *testing.T, stdout, stderr string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "test\n", stdout)
			},
		},
		{
			name: "default output format",
			cmd: func(commandContext *Context) Command {
				command := newTestCommand(commandContext)
				command.runFn = func(cmd *cobra.Command, args []string) cenclierrors.CencliError {
					formatter.Println(formatter.Stdout, command.Config().OutputFormat)
					return nil
				}
				return command
			},
			args:  []string{},
			store: func(ctrl *gomock.Controller) store.Store { return storemocks.NewMockStore(ctrl) },
			assert: func(t *testing.T, stdout, stderr string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "json\n", stdout)
			},
		},
		{
			name: "overriden output format",
			cmd: func(commandContext *Context) Command {
				command := newTestCommand(commandContext)
				command.runFn = func(cmd *cobra.Command, args []string) cenclierrors.CencliError {
					formatter.Println(formatter.Stdout, command.Config().OutputFormat)
					return nil
				}
				return command
			},
			args:  []string{"--output-format", "yaml"},
			flags: map[string]string{},
			store: func(ctrl *gomock.Controller) store.Store { return storemocks.NewMockStore(ctrl) },
			assert: func(t *testing.T, stdout, stderr string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "yaml\n", stdout)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			viper.Reset()
			cfg, err := config.New(tempDir)
			require.NoError(t, err)

			rootCmd := &cobra.Command{}
			// Bind global flags at root to make output-format available to subcommands
			// (binding will occur after root command is built)
			rootCmd.SetArgs(tc.args)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			cmdContext := NewCommandContext(cfg, tc.store(ctrl))

			rootCmd, err = RootCommandToCobra(tc.cmd(cmdContext))
			if tc.registerErrContains.IsPresent() {
				assert.ErrorContains(t, err, tc.registerErrContains.MustGet())
				return
			}
			require.NoError(t, err)

			require.NoError(t, config.BindGlobalFlags(rootCmd.PersistentFlags()))
			rootCmd.SetArgs(tc.args)

			var stdout, stderr bytes.Buffer
			formatter.Stdout = &stdout
			formatter.Stderr = &stderr

			for flag, value := range tc.flags {
				if setErr := rootCmd.Flags().Set(flag, value); setErr != nil {
					t.Fatalf("failed to set flag %s: %v", flag, setErr)
				}
			}

			cmdErr := rootCmd.Execute()
			tc.assert(t, stdout.String(), stderr.String(), cmdErr)
		})
	}
}
