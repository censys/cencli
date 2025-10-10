package input

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadLinesFromStdin(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		opts           []ReaderOption
		expectedOutput []string
	}{
		{
			name:           "basic input with default options",
			input:          "line1\nline2\nline3",
			opts:           nil,
			expectedOutput: []string{"line1", "line2", "line3"},
		},
		{
			name:           "input with whitespace - default trim",
			input:          "  line1  \n  line2  \n  line3  ",
			opts:           nil,
			expectedOutput: []string{"line1", "line2", "line3"},
		},
		{
			name:           "input with whitespace - don't trim",
			input:          "  line1  \n  line2  \n  line3  ",
			opts:           []ReaderOption{WithDontTrimSpace()},
			expectedOutput: []string{"  line1  ", "  line2  ", "  line3  "},
		},
		{
			name:           "input with blank lines - default skip",
			input:          "line1\n\nline2\n\nline3",
			opts:           nil,
			expectedOutput: []string{"line1", "line2", "line3"},
		},
		{
			name:           "input with blank lines - leave blanks",
			input:          "line1\n\nline2\n\nline3",
			opts:           []ReaderOption{WithLeaveBlanks()},
			expectedOutput: []string{"line1", "", "line2", "", "line3"},
		},
		{
			name:           "input with whitespace-only lines - default skip",
			input:          "line1\n   \nline2\n\t\nline3",
			opts:           nil,
			expectedOutput: []string{"line1", "line2", "line3"},
		},
		{
			name:           "input with whitespace-only lines - leave blanks and don't trim",
			input:          "line1\n   \nline2\n\t\nline3",
			opts:           []ReaderOption{WithLeaveBlanks(), WithDontTrimSpace()},
			expectedOutput: []string{"line1", "   ", "line2", "\t", "line3"},
		},
		{
			name:           "input with whitespace-only lines - leave blanks but trim",
			input:          "line1\n   \nline2\n\t\nline3",
			opts:           []ReaderOption{WithLeaveBlanks()},
			expectedOutput: []string{"line1", "", "line2", "", "line3"},
		},
		{
			name:           "empty input",
			input:          "",
			opts:           nil,
			expectedOutput: nil,
		},
		{
			name:           "only blank lines",
			input:          "\n\n\n",
			opts:           nil,
			expectedOutput: nil,
		},
		{
			name:           "only blank lines - leave blanks",
			input:          "\n\n\n",
			opts:           []ReaderOption{WithLeaveBlanks()},
			expectedOutput: []string{"", "", ""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader(tc.input))

			result, err := ReadLinesFromStdin(cmd.InOrStdin(), tc.opts...)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedOutput, result)
		})
	}
}

func TestReadLinesFromFile(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()

	// Create test file with various content
	testFile := tempDir + "/test.txt"
	testContent := "line1\n  line2  \n\nline3\n   \nline4"
	err := os.WriteFile(testFile, []byte(testContent), 0o644)
	require.NoError(t, err)

	// Create empty test file
	emptyFile := tempDir + "/empty.txt"
	err = os.WriteFile(emptyFile, []byte(""), 0o644)
	require.NoError(t, err)

	tests := []struct {
		name           string
		filePath       string
		opts           []ReaderOption
		expectedOutput []string
		expectError    bool
		expectedError  error
	}{
		{
			name:           "valid file with default options",
			filePath:       testFile,
			opts:           nil,
			expectedOutput: []string{"line1", "line2", "line3", "line4"},
			expectError:    false,
		},
		{
			name:           "valid file - don't trim whitespace",
			filePath:       testFile,
			opts:           []ReaderOption{WithDontTrimSpace()},
			expectedOutput: []string{"line1", "  line2  ", "line3", "   ", "line4"},
			expectError:    false,
		},
		{
			name:           "valid file - leave blanks",
			filePath:       testFile,
			opts:           []ReaderOption{WithLeaveBlanks()},
			expectedOutput: []string{"line1", "line2", "", "line3", "", "line4"},
			expectError:    false,
		},
		{
			name:           "valid file - leave blanks and don't trim",
			filePath:       testFile,
			opts:           []ReaderOption{WithLeaveBlanks(), WithDontTrimSpace()},
			expectedOutput: []string{"line1", "  line2  ", "", "line3", "   ", "line4"},
			expectError:    false,
		},
		{
			name:           "empty file",
			filePath:       emptyFile,
			opts:           nil,
			expectedOutput: nil,
			expectError:    false,
		},
		{
			name:           "non-existent file",
			filePath:       tempDir + "/nonexistent.txt",
			opts:           nil,
			expectedOutput: nil,
			expectError:    true,
			expectedError:  newInvalidInputFileError(tempDir+"/nonexistent.txt", os.ErrNotExist),
		},
		{
			name:           "invalid file path",
			filePath:       "/invalid/path/file.txt",
			opts:           nil,
			expectedOutput: nil,
			expectError:    true,
			expectedError:  newInvalidInputFileError("/invalid/path/file.txt", os.ErrNotExist),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ReadLinesFromFile(tc.filePath, tc.opts...)

			if tc.expectError {
				require.Error(t, err)
				if tc.expectedError != nil {
					// Compare error types and messages
					assert.IsType(t, tc.expectedError, err)
					// Note: We can't compare the exact error because the underlying OS error may vary
					// but we can check that it's the right type of error
				}
				assert.Equal(t, tc.expectedOutput, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedOutput, result)
			}
		})
	}
}

func TestSplitString(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		opts           []SplitStringOption
		expectedOutput []string
	}{
		{
			name:           "basic comma-separated values",
			input:          "a,b,c",
			opts:           nil,
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			name:           "comma-separated with whitespace",
			input:          "a, b , c",
			opts:           nil,
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			name:           "comma-separated with extra whitespace",
			input:          "  a  ,  b  ,  c  ",
			opts:           nil,
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			name:           "comma-separated with empty parts",
			input:          "a,,b,c",
			opts:           nil,
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			name:           "comma-separated with whitespace-only parts",
			input:          "a,   ,b,c",
			opts:           nil,
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			name:           "single value",
			input:          "single",
			opts:           nil,
			expectedOutput: []string{"single"},
		},
		{
			name:           "single value with whitespace",
			input:          "  single  ",
			opts:           nil,
			expectedOutput: []string{"single"},
		},
		{
			name:           "empty string",
			input:          "",
			opts:           nil,
			expectedOutput: []string{},
		},
		{
			name:           "only commas",
			input:          ",,,",
			opts:           nil,
			expectedOutput: []string{},
		},
		{
			name:           "only whitespace and commas",
			input:          " , , , ",
			opts:           nil,
			expectedOutput: []string{},
		},
		{
			name:           "custom delimiter - semicolon",
			input:          "a;b;c",
			opts:           []SplitStringOption{WithDelimiter(";")},
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			name:           "custom delimiter - pipe",
			input:          "a|b|c",
			opts:           []SplitStringOption{WithDelimiter("|")},
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			name:           "custom delimiter - space",
			input:          "a b c",
			opts:           []SplitStringOption{WithDelimiter(" ")},
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			name:           "custom delimiter - newline",
			input:          "a\nb\nc",
			opts:           []SplitStringOption{WithDelimiter("\n")},
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			name:           "custom delimiter with whitespace",
			input:          "a ; b ; c",
			opts:           []SplitStringOption{WithDelimiter(";")},
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			name:           "custom delimiter - multi-character",
			input:          "a::b::c",
			opts:           []SplitStringOption{WithDelimiter("::")},
			expectedOutput: []string{"a", "b", "c"},
		},
		{
			name:           "no delimiter found",
			input:          "no-delimiter-here",
			opts:           []SplitStringOption{WithDelimiter(",")},
			expectedOutput: []string{"no-delimiter-here"},
		},
		{
			name:           "delimiter at start and end",
			input:          ",a,b,c,",
			opts:           nil,
			expectedOutput: []string{"a", "b", "c"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SplitString(tc.input, tc.opts...)
			assert.Equal(t, tc.expectedOutput, result)
		})
	}
}
