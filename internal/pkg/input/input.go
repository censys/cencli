package input

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

const (
	StdInSentinel = "-"
)

type inputReader struct {
	scanner     *bufio.Scanner
	trimSpace   bool
	leaveBlanks bool
}

func newInputReader(scanner *bufio.Scanner, opts ...ReaderOption) *inputReader {
	reader := &inputReader{
		scanner:   scanner,
		trimSpace: true,
	}
	for _, opt := range opts {
		opt(reader)
	}
	return reader
}

type ReaderOption func(*inputReader)

// WithDontTrimSpace sets the input reader to not trim whitespace.
func WithDontTrimSpace() ReaderOption {
	return func(reader *inputReader) { reader.trimSpace = false }
}

// WithLeaveBlanks sets the input reader to leave blanks.
func WithLeaveBlanks() ReaderOption {
	return func(reader *inputReader) { reader.leaveBlanks = true }
}

// readLinesFromScanner scans lines from the provided scanner, trimming whitespace and skipping blanks.
func (r *inputReader) readLinesFromScanner() ([]string, error) {
	// Increase the maximum token size to handle large lines/files.
	// Default is 64K; bump to 10MB which is plenty for CLI inputs.
	r.scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var lines []string
	for r.scanner.Scan() {
		line := r.scanner.Text()
		if r.trimSpace {
			line = strings.TrimSpace(line)
		}
		if line != "" || r.leaveBlanks {
			lines = append(lines, line)
		}
	}
	if err := r.scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// ReadLinesFromStdin reads lines from stdin using the command's input reader.
func ReadLinesFromStdin(r io.Reader, opts ...ReaderOption) ([]string, cenclierrors.CencliError) {
	lines, err := newInputReader(bufio.NewScanner(r), opts...).readLinesFromScanner()
	if err != nil {
		return nil, cenclierrors.NewCencliError(err)
	}
	return lines, nil
}

// ReadLinesFromFile reads lines from a file using the command's input reader.
func ReadLinesFromFile(filePath string, opts ...ReaderOption) ([]string, cenclierrors.CencliError) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, newInvalidInputFileError(filePath, err)
	}
	defer file.Close()
	lines, scanErr := newInputReader(bufio.NewScanner(file), opts...).readLinesFromScanner()
	if scanErr != nil {
		return nil, newInvalidInputFileError(filePath, scanErr)
	}
	return lines, nil
}

type SplitStringOption func(*splitStringOptions)

type splitStringOptions struct {
	delimiter string
}

// WithDelimiter sets the split string option to use the given delimiter.
func WithDelimiter(delimiter string) SplitStringOption {
	return func(options *splitStringOptions) { options.delimiter = delimiter }
}

// SplitString splits a string into a list of strings using the given delimiter.
// If the delimiter is not provided, it defaults to ",".
// Will always trim whitespace from the parts.
func SplitString(line string, opts ...SplitStringOption) []string {
	options := &splitStringOptions{
		delimiter: ",",
	}
	for _, opt := range opts {
		opt(options)
	}
	parts := strings.Split(line, options.delimiter)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
