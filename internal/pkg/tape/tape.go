package tape

import (
	"strings"
)

// Tape represents a VHS tape recording with a name, configuration, and commands.
type Tape struct {
	Name     string
	config   *Config
	commands string
}

// Config defines the visual settings for a tape recording.
type Config struct {
	Width    int
	Height   int
	FontSize int
}

type typeOptions struct {
	sleepAfter int
	clearAfter bool
}

// NewTape creates a new Tape with the given name, config, and VHS commands.
func NewTape(name string, config *Config, commands ...string) Tape {
	return Tape{
		Name:     name,
		config:   config,
		commands: strings.Join(commands, "\n"),
	}
}

// DefaultTapeConfig returns a TapeConfig with default dimensions and font size.
func DefaultTapeConfig() *Config {
	return &Config{
		Width:    1400,
		Height:   600,
		FontSize: 25,
	}
}
