package input

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInvalidInputFileError(t *testing.T) {
	baseErr := errors.New("file not found")
	err := newInvalidInputFileError("test.txt", baseErr)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test.txt")
	assert.Contains(t, err.Error(), "file not found")
	assert.Contains(t, err.Error(), "invalid input file")
	assert.Equal(t, "Invalid Input File", err.Title())
	assert.True(t, err.ShouldPrintUsage()) // This error does print usage
}

func TestNewInvalidInputFileError_EmptyPath(t *testing.T) {
	baseErr := errors.New("cannot open file")
	err := newInvalidInputFileError("", baseErr)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid input file")
	assert.Contains(t, err.Error(), "cannot open file")
	assert.Equal(t, "Invalid Input File", err.Title())
	assert.True(t, err.ShouldPrintUsage())
}

func TestNewInvalidInputFileError_AbsolutePath(t *testing.T) {
	baseErr := errors.New("permission denied")
	err := newInvalidInputFileError("/absolute/path/to/file.txt", baseErr)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "/absolute/path/to/file.txt")
	assert.Contains(t, err.Error(), "permission denied")
	assert.Contains(t, err.Error(), "invalid input file")
	assert.Equal(t, "Invalid Input File", err.Title())
	assert.True(t, err.ShouldPrintUsage())
}

func TestNewInvalidInputFileError_RelativePath(t *testing.T) {
	baseErr := errors.New("is a directory")
	err := newInvalidInputFileError("../relative/path.txt", baseErr)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "../relative/path.txt")
	assert.Contains(t, err.Error(), "is a directory")
	assert.Contains(t, err.Error(), "invalid input file")
	assert.Equal(t, "Invalid Input File", err.Title())
	assert.True(t, err.ShouldPrintUsage())
}
