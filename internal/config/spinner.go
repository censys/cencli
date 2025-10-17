package config

type SpinnerConfig struct {
	Disabled                   bool   `yaml:"disabled" mapstructure:"disabled" doc:"Disable spinner during operations"`
	StartStopwatchAfterSeconds uint64 `yaml:"start-stopwatch-after" mapstructure:"start-stopwatch-after" doc:"Show stopwatch in the spinner after this many seconds"`
}

var defaultSpinnerConfig = SpinnerConfig{
	Disabled:                   false,
	StartStopwatchAfterSeconds: 5,
}
