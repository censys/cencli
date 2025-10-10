package command

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestTemplate(t *testing.T) {
	tests := []struct {
		name     string
		cmd      func() *cobra.Command
		examples []string
		expected string
	}{
		{
			name: "basic command",
			cmd: func() *cobra.Command {
				root := &cobra.Command{
					Use:   "root",
					Short: "root short",
					Long:  "root long",
				}
				root.Flags().StringP("flag", "f", "default", "flag description")
				root.AddCommand(&cobra.Command{Use: "test", Short: "test short", Long: "test long"})
				root.InitDefaultCompletionCmd()
				root.InitDefaultHelpCmd()
				return root
			},
			examples: []string{"test --flag value", "test --flag value2"},
			expected: `
root long

Usage:
  root [command]

Examples:
  root test --flag value
  root test --flag value2

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command

Flags:
  -f, --flag string   flag description (default "default")

Additional help topics:
  root test       test short

Use "root [command] --help" for more information.
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := helpTemplate(tc.cmd(), tc.examples)
			actual = strings.TrimLeft(actual, " \t\r\n")
			expected := strings.TrimLeft(tc.expected, " \t\r\n")
			assert.Equal(t, expected, actual)
		})
	}
}

func TestStyledFlagUsages_MatchesPflagOutput(t *testing.T) {
	tests := []struct {
		name     string
		setupFn  func() *pflag.FlagSet
		testName string
	}{
		{
			name: "simple flags",
			setupFn: func() *pflag.FlagSet {
				fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
				fs.StringP("name", "n", "default", "your name")
				fs.IntP("count", "c", 0, "count of items")
				fs.BoolP("verbose", "v", false, "verbose output")
				return fs
			},
		},
		{
			name: "flags with various types",
			setupFn: func() *pflag.FlagSet {
				fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
				fs.String("output", "json", "output format")
				fs.Int("page-size", 100, "page size")
				fs.Duration("timeout", 0, "timeout duration")
				fs.StringSlice("fields", []string{}, "fields to include")
				return fs
			},
		},
		{
			name: "flags without shorthand",
			setupFn: func() *pflag.FlagSet {
				fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
				fs.String("long-flag-name", "", "a flag with no shorthand")
				fs.Bool("enable-feature", false, "enable a feature")
				return fs
			},
		},
		{
			name: "mixed flags",
			setupFn: func() *pflag.FlagSet {
				fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
				fs.StringP("collection-id", "c", "", "collection ID (optional)")
				fs.StringP("org-id", "o", "", "org ID")
				fs.IntP("max-pages", "m", 1, "maximum pages")
				fs.String("no-color", "", "disable colors")
				return fs
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := tc.setupFn()

			// Get output from pflag's FlagUsages
			pflagOutput := fs.FlagUsages()

			// Get output from our vendored styledFlagUsages
			vendoredOutput := styledFlagUsages(fs)

			// They should be identical
			assert.Equal(t, pflagOutput, vendoredOutput,
				"styledFlagUsages output should match pflag.FlagUsages output exactly")
		})
	}
}
