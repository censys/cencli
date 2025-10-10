package flags

import "github.com/spf13/pflag"

const (
	shortFlagName  = "short"
	shortFlagShort = "s"
	shortFlagDesc  = "render data through a configurable template"
)

type shortFlag struct {
	*boolFlag
}

// NewShortFlag instantiates a new short flag on a given flag set.
// Returns a ShortFlag that can be used to get the value of the flag.
func NewShortFlag(flags *pflag.FlagSet, shortOverride string) BoolFlag {
	short := shortFlagShort
	if shortOverride != "" {
		short = shortOverride
	}
	return &shortFlag{
		boolFlag: NewBoolFlag(flags, shortFlagName, short, false, shortFlagDesc),
	}
}
