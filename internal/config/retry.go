package config

import (
	"encoding"
	"errors"
	"fmt"
	"time"
)

type RetryStrategy struct {
	MaxAttempts uint64        `yaml:"max-attempts" mapstructure:"max-attempts"`
	BaseDelay   time.Duration `yaml:"base-delay" mapstructure:"base-delay"`
	MaxDelay    time.Duration `yaml:"max-delay" mapstructure:"max-delay"`
	Backoff     BackoffType   `yaml:"backoff" mapstructure:"backoff" doc:"Backoff strategy (fixed|linear|exponential)"`
}

var defaultRetryStrategy = RetryStrategy{
	MaxAttempts: 2,
	BaseDelay:   500 * time.Millisecond,
	MaxDelay:    30 * time.Second,
	Backoff:     BackoffFixed,
}

type BackoffType string

const (
	BackoffFixed       BackoffType = "fixed"
	BackoffExponential BackoffType = "exponential"
	BackoffLinear      BackoffType = "linear"
)

var ErrInvalidBackoffType = errors.New("invalid backoff type")

func (b BackoffType) String() string {
	return string(b)
}

var _ encoding.TextUnmarshaler = (*BackoffType)(nil)

func (b *BackoffType) UnmarshalText(text []byte) error {
	s := string(text)
	switch s {
	case "fixed":
		*b = BackoffFixed
	case "exponential":
		*b = BackoffExponential
	case "linear":
		*b = BackoffLinear
	default:
		return fmt.Errorf("%w: %s", ErrInvalidBackoffType, s)
	}
	return nil
}
