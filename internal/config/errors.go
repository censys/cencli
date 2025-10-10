package config

import (
	"fmt"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

type InvalidConfigError interface {
	cenclierrors.CencliError
}

type invalidConfigError struct {
	key    string
	reason string
}

var _ InvalidConfigError = &invalidConfigError{}

func newInvalidConfigErrorWithKey(key string, reason string) InvalidConfigError {
	return &invalidConfigError{key: key, reason: reason}
}

func newInvalidConfigError(reason string) InvalidConfigError {
	return &invalidConfigError{reason: reason}
}

func (e *invalidConfigError) Error() string {
	if e.key != "" {
		return fmt.Sprintf("failed to load config for %s: %s", e.key, e.reason)
	}
	return fmt.Sprintf("failed to load config: %s", e.reason)
}

func (e *invalidConfigError) Title() string {
	return "Failed to load config"
}

func (e *invalidConfigError) ShouldPrintUsage() bool {
	return true
}
