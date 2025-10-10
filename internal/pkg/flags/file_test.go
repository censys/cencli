package flags

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fileFlagName  = "test-file-flag"
	fileFlagShort = "f"
)

func TestFileFlag(t *testing.T) {
	tests := []struct {
		name     string
		required bool
		setup    func(t *testing.T, tempDir string)
		args     []string
		assert   func(t *testing.T, tempDir, value string, err error)
	}{
		{
			name:     "stdin sentinel '-' optional",
			required: false,
			args:     []string{"--" + fileFlagName, "-"},
			assert: func(t *testing.T, tempDir, value string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "-", value)
			},
		},
		{
			name:     "stdin sentinel '-' required",
			required: true,
			args:     []string{"--" + fileFlagName, "-"},
			assert: func(t *testing.T, tempDir, value string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "-", value)
			},
		},
		{
			name:     "valid file with spaces - required",
			required: true,
			setup: func(t *testing.T, tempDir string) {
				path := filepath.Join(tempDir, "test.txt")
				err := os.WriteFile(path, []byte("test"), 0o644)
				require.NoError(t, err)
			},
			args: []string{"--" + fileFlagName, "test.txt   "},
			assert: func(t *testing.T, tempDir, value string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, filepath.Join(tempDir, "test.txt"), value)
			},
		},
		{
			name:     "directory path returns directory error",
			required: true,
			setup: func(t *testing.T, tempDir string) {
				// Ensure a directory exists and pass it as the flag value
				_ = os.MkdirAll(filepath.Join(tempDir, "adir"), 0o755)
			},
			args: []string{"--" + fileFlagName, "adir"},
			assert: func(t *testing.T, tempDir, value string, err error) {
				assert.Error(t, err)
				expected := NewInvalidFileFlagError(fileFlagName, filepath.Join(tempDir, "adir"), fmt.Errorf("is a directory"))
				assert.Equal(t, expected, err)
			},
		},
		{
			name:     "invalid file perms - required",
			required: true,
			setup: func(t *testing.T, tempDir string) {
				path := filepath.Join(tempDir, "test.txt")
				err := os.WriteFile(path, []byte("test"), 0o000)
				require.NoError(t, err)
			},
			args: []string{"--" + fileFlagName, "test.txt"},
			assert: func(t *testing.T, tempDir, value string, err error) {
				assert.Error(t, err)
				expectedParseErr := fmt.Errorf("open %s: %w", filepath.Join(tempDir, "test.txt"), os.ErrPermission)
				assert.Equal(t, NewInvalidFileFlagError(fileFlagName, filepath.Join(tempDir, "test.txt"), expectedParseErr), err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, tempDir)
			}

			// modify the args so that the argument has the absolute path, except for '-' sentinel
			if len(tc.args) == 2 && tc.args[1] != "-" {
				tc.args[1] = filepath.Join(tempDir, tc.args[1])
			}

			cmd := &cobra.Command{}
			flag := NewFileFlag(cmd.Flags(), tc.required, fileFlagName, fileFlagShort, "A File Flag")
			cmd.SetArgs(tc.args)
			cmd.Run = func(cmd *cobra.Command, args []string) {
				value, err := flag.Value()
				tc.assert(t, tempDir, value, err)
			}
			require.NoError(t, cmd.Execute())
		})
	}
}

func TestFileFlag_IsSet(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "flag not provided",
			args:     []string{},
			expected: false,
		},
		{
			name:     "flag provided with value",
			args:     []string{"--" + fileFlagName, "-"},
			expected: true,
		},
		{
			name:     "flag provided with short form",
			args:     []string{"-" + fileFlagShort, "-"},
			expected: true,
		},
		{
			name:     "flag provided with empty value",
			args:     []string{"--" + fileFlagName, ""},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			flag := NewFileFlag(cmd.Flags(), false, fileFlagName, fileFlagShort, "A File Flag")
			cmd.SetArgs(tc.args)
			cmd.Run = func(cmd *cobra.Command, args []string) {
				assert.Equal(t, tc.expected, flag.IsSet())
			}
			require.NoError(t, cmd.Execute())
		})
	}
}

func TestFileFlag_Lines(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, tempDir string) string
		args     []string
		expected []string
		wantErr  bool
	}{
		{
			name: "read lines from valid file",
			setup: func(t *testing.T, tempDir string) string {
				path := filepath.Join(tempDir, "test.txt")
				content := "line1\nline2\n  line3  \n\nline4\n"
				err := os.WriteFile(path, []byte(content), 0o644)
				require.NoError(t, err)
				return path
			},
			expected: []string{"line1", "line2", "line3", "line4"},
			wantErr:  false,
		},
		{
			name: "read lines from file with only whitespace",
			setup: func(t *testing.T, tempDir string) string {
				path := filepath.Join(tempDir, "whitespace.txt")
				content := "   \n\t\n  \n"
				err := os.WriteFile(path, []byte(content), 0o644)
				require.NoError(t, err)
				return path
			},
			expected: nil,
			wantErr:  false,
		},
		{
			name: "read lines from empty file",
			setup: func(t *testing.T, tempDir string) string {
				path := filepath.Join(tempDir, "empty.txt")
				err := os.WriteFile(path, []byte(""), 0o644)
				require.NoError(t, err)
				return path
			},
			expected: nil,
			wantErr:  false,
		},
		{
			name: "read lines from single line file",
			setup: func(t *testing.T, tempDir string) string {
				path := filepath.Join(tempDir, "single.txt")
				content := "single line"
				err := os.WriteFile(path, []byte(content), 0o644)
				require.NoError(t, err)
				return path
			},
			expected: []string{"single line"},
			wantErr:  false,
		},
		{
			name: "error reading from non-existent file",
			setup: func(t *testing.T, tempDir string) string {
				return filepath.Join(tempDir, "nonexistent.txt")
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "error reading from file with no permissions",
			setup: func(t *testing.T, tempDir string) string {
				path := filepath.Join(tempDir, "noperm.txt")
				content := "test content"
				err := os.WriteFile(path, []byte(content), 0o000)
				require.NoError(t, err)
				return path
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filePath := tc.setup(t, tempDir)

			cmd := &cobra.Command{}
			flag := NewFileFlag(cmd.Flags(), false, fileFlagName, fileFlagShort, "A File Flag")
			cmd.SetArgs([]string{"--" + fileFlagName, filePath})
			cmd.Run = func(cmd *cobra.Command, args []string) {
				lines, err := flag.Lines(cmd)
				if tc.wantErr {
					assert.Error(t, err)
					assert.Nil(t, lines)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.expected, lines)
				}
			}
			require.NoError(t, cmd.Execute())
		})
	}
}

func TestFileFlag_Lines_Stdin(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "read lines from stdin",
			input:    "line1\nline2\n  line3  \n\nline4\n",
			expected: []string{"line1", "line2", "line3", "line4"},
		},
		{
			name:     "read empty stdin",
			input:    "",
			expected: nil,
		},
		{
			name:     "read stdin with only whitespace",
			input:    "   \n\t\n  \n",
			expected: nil,
		},
		{
			name:     "read single line from stdin",
			input:    "single line",
			expected: []string{"single line"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader(tc.input))
			flag := NewFileFlag(cmd.Flags(), false, fileFlagName, fileFlagShort, "A File Flag")
			cmd.SetArgs([]string{"--" + fileFlagName, "-"})
			cmd.Run = func(cmd *cobra.Command, args []string) {
				lines, err := flag.Lines(cmd)
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, lines)
			}
			require.NoError(t, cmd.Execute())
		})
	}
}
