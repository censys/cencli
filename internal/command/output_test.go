package command

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	storemocks "github.com/censys/cencli/gen/store/mocks"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
)

// TestOutputFormatBinding tests that output format flag bindings work correctly
// when commands have different default output types.
func TestOutputFormatBinding(t *testing.T) {
	tests := []struct {
		name                  string
		setupCommands         func(ctx *Context) (root Command, children []Command)
		commandToExecute      string // which command to run (empty = root)
		args                  []string
		expectedConfigFormat  formatter.OutputFormat // what Config().OutputFormat should be
		expectedFlagValue     string                 // what the flag value should be
		configFileFormat      formatter.OutputFormat // what's in the config file
		assertBeforeExecution func(t *testing.T, rootCmd *cobra.Command)
		assertAfterUnmarshal  func(t *testing.T, cfg *config.Config)
	}{
		{
			name: "root command with json default",
			setupCommands: func(ctx *Context) (Command, []Command) {
				root := newTestCommand(ctx)
				root.useFn = func() string { return "root" }
				return root, nil
			},
			commandToExecute:     "",
			args:                 []string{},
			configFileFormat:     formatter.OutputFormatJSON,
			expectedConfigFormat: formatter.OutputFormatJSON,
			expectedFlagValue:    "json",
		},
		{
			name: "child command with short default - no user override",
			setupCommands: func(ctx *Context) (Command, []Command) {
				root := newTestCommand(ctx)
				root.useFn = func() string { return "root" }

				child := newTestCommand(ctx)
				child.useFn = func() string { return "child" }
				child.defaultOutputTypeFn = func() OutputType {
					return OutputTypeShort
				}
				child.supportedOutputTypesFn = func() []OutputType {
					return []OutputType{OutputTypeData, OutputTypeShort}
				}

				return root, []Command{child}
			},
			commandToExecute:     "child",
			args:                 []string{},
			configFileFormat:     formatter.OutputFormatJSON,
			expectedConfigFormat: formatter.OutputFormatShort, // Should be short due to command default
			expectedFlagValue:    "short",
		},
		{
			name: "child command with short default - user overrides with json",
			setupCommands: func(ctx *Context) (Command, []Command) {
				root := newTestCommand(ctx)
				root.useFn = func() string { return "root" }

				child := newTestCommand(ctx)
				child.useFn = func() string { return "child" }
				child.defaultOutputTypeFn = func() OutputType {
					return OutputTypeShort
				}
				child.supportedOutputTypesFn = func() []OutputType {
					return []OutputType{OutputTypeData, OutputTypeShort}
				}

				return root, []Command{child}
			},
			commandToExecute:     "child",
			args:                 []string{"--output-format", "json"},
			configFileFormat:     formatter.OutputFormatJSON,
			expectedConfigFormat: formatter.OutputFormatJSON, // User override wins
			expectedFlagValue:    "json",
		},
		{
			name: "child command with short default - yaml in config file",
			setupCommands: func(ctx *Context) (Command, []Command) {
				root := newTestCommand(ctx)
				root.useFn = func() string { return "root" }

				child := newTestCommand(ctx)
				child.useFn = func() string { return "child" }
				child.defaultOutputTypeFn = func() OutputType {
					return OutputTypeShort
				}
				child.supportedOutputTypesFn = func() []OutputType {
					return []OutputType{OutputTypeData, OutputTypeShort}
				}

				return root, []Command{child}
			},
			commandToExecute:     "child",
			args:                 []string{},
			configFileFormat:     formatter.OutputFormatYAML,
			expectedConfigFormat: formatter.OutputFormatShort, // Command default wins over config file
			expectedFlagValue:    "short",
		},
		{
			name: "sibling commands with different defaults don't conflict",
			setupCommands: func(ctx *Context) (Command, []Command) {
				root := newTestCommand(ctx)
				root.useFn = func() string { return "root" }

				child1 := newTestCommand(ctx)
				child1.useFn = func() string { return "child1" }
				child1.defaultOutputTypeFn = func() OutputType {
					return OutputTypeShort
				}
				child1.supportedOutputTypesFn = func() []OutputType {
					return []OutputType{OutputTypeData, OutputTypeShort}
				}

				child2 := newTestCommand(ctx)
				child2.useFn = func() string { return "child2" }
				child2.defaultOutputTypeFn = func() OutputType {
					return OutputTypeData
				}

				return root, []Command{child1, child2}
			},
			commandToExecute:     "child2",
			args:                 []string{},
			configFileFormat:     formatter.OutputFormatJSON,
			expectedConfigFormat: formatter.OutputFormatJSON, // child2 should use raw default (json)
			expectedFlagValue:    "json",
		},
		{
			name: "template default command",
			setupCommands: func(ctx *Context) (Command, []Command) {
				root := newTestCommand(ctx)
				root.useFn = func() string { return "root" }

				child := newTestCommand(ctx)
				child.useFn = func() string { return "templated" }
				child.defaultOutputTypeFn = func() OutputType {
					return OutputTypeTemplate
				}
				child.supportedOutputTypesFn = func() []OutputType {
					return []OutputType{OutputTypeData, OutputTypeTemplate}
				}

				return root, []Command{child}
			},
			commandToExecute:     "templated",
			args:                 []string{},
			configFileFormat:     formatter.OutputFormatJSON,
			expectedConfigFormat: formatter.OutputFormatTemplate,
			expectedFlagValue:    "template",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset Viper for each test
			viper.Reset()

			// Create temp directory and config
			tempDir := t.TempDir()
			cfg, err := config.New(tempDir)
			require.NoError(t, err)

			// Set config file format
			cfg.OutputFormat = tc.configFileFormat

			// Create mock store
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := storemocks.NewMockStore(ctrl)

			// Create command context
			cmdContext := NewCommandContext(cfg, mockStore)

			// Setup commands
			rootCmd, children := tc.setupCommands(cmdContext)

			// Convert root to cobra
			cobraRoot, err := RootCommandToCobra(rootCmd)
			require.NoError(t, err)

			// Bind global flags
			require.NoError(t, config.BindGlobalFlags(cobraRoot.PersistentFlags(), cfg))

			// Add children
			for _, child := range children {
				require.NoError(t, rootCmd.AddSubCommands(child))
			}

			// Optional assertions before execution
			if tc.assertBeforeExecution != nil {
				tc.assertBeforeExecution(t, cobraRoot)
			}

			// Build command args
			args := []string{}
			if tc.commandToExecute != "" {
				args = append(args, tc.commandToExecute)
			}
			args = append(args, tc.args...)
			cobraRoot.SetArgs(args)

			// Capture output
			var stdout, stderr bytes.Buffer
			formatter.Stdout = &stdout
			formatter.Stderr = &stderr
			cobraRoot.SetOut(&stdout)
			cobraRoot.SetErr(&stderr)

			// Execute command
			cmdErr := cobraRoot.Execute()
			require.NoError(t, cmdErr)

			// After execution, config should have been unmarshaled
			// Check that Config().OutputFormat has the expected value
			assert.Equal(t, tc.expectedConfigFormat, cfg.OutputFormat,
				"Config().OutputFormat should be %s but got %s",
				tc.expectedConfigFormat, cfg.OutputFormat)

			// Also verify the flag value matches
			var targetCmd *cobra.Command
			if tc.commandToExecute == "" {
				targetCmd = cobraRoot
			} else {
				var findErr error
				targetCmd, _, findErr = cobraRoot.Find([]string{tc.commandToExecute})
				require.NoError(t, findErr)
			}

			flag := targetCmd.Flag("output-format")
			require.NotNil(t, flag, "output-format flag should exist")
			assert.Equal(t, tc.expectedFlagValue, flag.Value.String(),
				"Flag value should be %s but got %s",
				tc.expectedFlagValue, flag.Value.String())

			// Optional assertions after unmarshal
			if tc.assertAfterUnmarshal != nil {
				tc.assertAfterUnmarshal(t, cfg)
			}
		})
	}
}

// TestMultipleCommandsSequentially tests that running different commands
// sequentially doesn't cause binding conflicts
func TestMultipleCommandsSequentially(t *testing.T) {
	// Reset Viper
	viper.Reset()

	// Create temp directory and config
	tempDir := t.TempDir()
	cfg, err := config.New(tempDir)
	require.NoError(t, err)
	cfg.OutputFormat = formatter.OutputFormatYAML

	// Create mock store
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := storemocks.NewMockStore(ctrl)

	// Create command context
	cmdContext := NewCommandContext(cfg, mockStore)

	// Create root command
	root := newTestCommand(cmdContext)
	root.useFn = func() string { return "root" }

	// Create child1 with short default
	child1 := newTestCommand(cmdContext)
	child1.useFn = func() string { return "child1" }
	child1.defaultOutputTypeFn = func() OutputType {
		return OutputTypeShort
	}
	child1.supportedOutputTypesFn = func() []OutputType {
		return []OutputType{OutputTypeData, OutputTypeShort}
	}
	child1.runFn = func(cmd *cobra.Command, args []string) cenclierrors.CencliError {
		// Should see "short" in config
		assert.Equal(t, formatter.OutputFormatShort, child1.Config().OutputFormat)
		return nil
	}

	// Create child2 with raw default
	child2 := newTestCommand(cmdContext)
	child2.useFn = func() string { return "child2" }
	child2.defaultOutputTypeFn = func() OutputType {
		return OutputTypeData
	}
	child2.runFn = func(cmd *cobra.Command, args []string) cenclierrors.CencliError {
		// Should see "yaml" from config file, not "short" from child1
		assert.Equal(t, formatter.OutputFormatYAML, child2.Config().OutputFormat)
		return nil
	}

	// Build command tree
	cobraRoot, err := RootCommandToCobra(root)
	require.NoError(t, err)
	require.NoError(t, config.BindGlobalFlags(cobraRoot.PersistentFlags(), cfg))
	require.NoError(t, root.AddSubCommands(child1, child2))

	// Test 1: Run child1 (should have short)
	t.Run("run child1 with short default", func(t *testing.T) {
		viper.Reset()
		cfg.OutputFormat = formatter.OutputFormatYAML // Reset config

		var stdout, stderr bytes.Buffer
		formatter.Stdout = &stdout
		formatter.Stderr = &stderr

		cobraRoot.SetArgs([]string{"child1"})
		err := cobraRoot.Execute()
		require.NoError(t, err)
	})

	// Test 2: Run child2 (should have yaml, not affected by child1)
	t.Run("run child2 with raw default", func(t *testing.T) {
		viper.Reset()
		cfg.OutputFormat = formatter.OutputFormatYAML // Reset config

		var stdout, stderr bytes.Buffer
		formatter.Stdout = &stdout
		formatter.Stderr = &stderr

		cobraRoot.SetArgs([]string{"child2"})
		err := cobraRoot.Execute()
		require.NoError(t, err)
	})
}
