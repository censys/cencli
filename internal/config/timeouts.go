package config

import (
	"time"
)

type TimeoutConfig struct {
	HTTP time.Duration `yaml:"http" mapstructure:"http" doc:"Per-request timeout for HTTP requests (e.g. 10s, 1m). Set to 0 to disable"`
}

var defaultTimeoutConfig = TimeoutConfig{
	HTTP: 0,
}
