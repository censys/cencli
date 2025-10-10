package cenclierrors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCencliError(t *testing.T) {
	baseErr := errors.New("base error")
	cencliErr := NewCencliError(baseErr)

	assert.NotNil(t, cencliErr)
	assert.Contains(t, cencliErr.Error(), "base error")
	assert.Equal(t, "Unknown Error", cencliErr.Title())
	assert.False(t, cencliErr.ShouldPrintUsage())
}

func TestCencliError_Implementation(t *testing.T) {
	err := &cencliError{
		err: errors.New("test error"),
	}

	assert.Equal(t, "test error", err.Error())
	assert.Equal(t, "Unknown Error", err.Title())
	assert.False(t, err.ShouldPrintUsage())
}

func TestCencliError_WrappedError(t *testing.T) {
	innerErr := errors.New("inner error")
	wrappedErr := fmt.Errorf("wrapped: %w", innerErr)
	cencliErr := NewCencliError(wrappedErr)

	assert.Contains(t, cencliErr.Error(), "wrapped")
	assert.Contains(t, cencliErr.Error(), "inner error")
	assert.Equal(t, "Unknown Error", cencliErr.Title())
}

func TestCencliError_NilHandling(t *testing.T) {
	// Test that nil returns nil
	cencliErr := NewCencliError(nil)

	assert.Nil(t, cencliErr)
}

func TestCencliError_AvoidDoubleWrapping(t *testing.T) {
	// Create a CencliError
	baseErr := errors.New("base error")
	firstWrap := NewCencliError(baseErr)

	// Wrap it again - should return the same error, not double-wrap
	secondWrap := NewCencliError(firstWrap)

	assert.Equal(t, firstWrap, secondWrap)
}
