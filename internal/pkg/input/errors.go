package input

import "github.com/censys/cencli/internal/pkg/cenclierrors"

type InvalidInputFileError interface {
	cenclierrors.CencliError
}

type invalidInputFileError struct {
	fileName string
	err      error
}

var _ InvalidInputFileError = &invalidInputFileError{}

func newInvalidInputFileError(providedInputFile string, err error) InvalidInputFileError {
	return &invalidInputFileError{
		fileName: providedInputFile,
		err:      err,
	}
}

func (e *invalidInputFileError) Error() string {
	return "invalid input file: " + e.fileName + ": " + e.err.Error()
}

func (e *invalidInputFileError) Title() string {
	return "Invalid Input File"
}

func (e *invalidInputFileError) ShouldPrintUsage() bool {
	return true
}
