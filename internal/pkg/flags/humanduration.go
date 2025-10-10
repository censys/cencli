package flags

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

// HumanDurationFlag represents a duration flag that accepts extended units like d, w, y.
// Examples: "2h", "30m", "7d", "1w", "1y", or combinations like "1d12h".
// Returns a time.Duration after normalizing to hours.
type HumanDurationFlag interface {
	// Value returns the current value of the flag.
	// If the flag is marked as required but not provided,
	// it returns an error of type RequiredFlagNotSetError.
	// If the flag has an invalid duration, it returns an error of type InvalidDurationFlagError.
	// An optional value is returned to keep callers from having to compare to 0.
	Value() (mo.Option[time.Duration], cenclierrors.CencliError)
}

type humanDurationFlag struct {
	*stringFlag
	defaultValue mo.Option[time.Duration]
}

// NewHumanDurationFlag instantiates a new extended duration flag on a given flag set.
// Supports Go durations (e.g., 2h45m) and human units d=24h, w=7d, y=365d.
func NewHumanDurationFlag(flags *pflag.FlagSet, required bool, name, short string, defaultValue mo.Option[time.Duration], desc string) HumanDurationFlag {
	if required && defaultValue.IsPresent() {
		panic("flags: required duration flag cannot also have a default value: --" + name)
	}
	var defaultStr string
	if defaultValue.IsPresent() {
		defaultStr = defaultValue.MustGet().String()
	}
	return &humanDurationFlag{
		stringFlag:   NewStringFlag(flags, required, name, short, defaultStr, desc),
		defaultValue: defaultValue,
	}
}

func (f *humanDurationFlag) Value() (mo.Option[time.Duration], cenclierrors.CencliError) {
	f.trimSpace()
	strValue, err := f.stringFlag.Value()
	if err != nil {
		return mo.None[time.Duration](), err
	}
	if !f.wasProvided() {
		return f.defaultValue, nil
	}
	// Try native parser first
	if d, parseErr := time.ParseDuration(strValue); parseErr == nil {
		return mo.Some(d), nil
	}
	// Fallback to human parser
	d, parseErr := parseHumanDuration(strValue)
	if parseErr != nil {
		return mo.None[time.Duration](), NewInvalidDurationFlagError(f.name, strValue)
	}
	return mo.Some(d), nil
}

// parseHumanDuration parses durations containing tokens like 7d, 1w, 1y, 12h, 30m, 45s.
// Supports concatenated tokens (e.g., 1d12h30m). Does not support fractions.
func parseHumanDuration(input string) (time.Duration, error) {
	trimmed := strings.TrimSpace(strings.ToLower(input))
	if trimmed == "" {
		return 0, fmt.Errorf("empty duration")
	}
	// Tokenize number+unit pairs
	re := regexp.MustCompile(`(?i)(\d+)([smhdwy])`)
	matches := re.FindAllStringSubmatch(trimmed, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid duration format: %s", input)
	}
	var total time.Duration
	for _, m := range matches {
		if len(m) != 3 {
			return 0, fmt.Errorf("invalid token: %v", m)
		}
		num, _ := strconv.ParseInt(m[1], 10, 64)
		unit := m[2]
		switch unit {
		case "s":
			total += time.Duration(num) * time.Second
		case "m":
			total += time.Duration(num) * time.Minute
		case "h":
			total += time.Duration(num) * time.Hour
		case "d":
			total += time.Duration(num) * 24 * time.Hour
		case "w":
			total += time.Duration(num) * 7 * 24 * time.Hour
		case "y":
			// Approximate year as 365 days for CLI convenience
			total += time.Duration(num) * 365 * 24 * time.Hour
		default:
			return 0, fmt.Errorf("unsupported unit: %s", unit)
		}
	}
	return total, nil
}
