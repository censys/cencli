package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/datetime"
	"github.com/censys/cencli/internal/pkg/formatter"
)

type Config struct {
	OutputFormat  formatter.OutputFormat            `yaml:"output-format" mapstructure:"output-format" doc:"Default output format (json|yaml|ndjson|tree)"`
	NoColor       bool                              `yaml:"no-color" mapstructure:"no-color" doc:"Disable ANSI colors and styles"`
	NoSpinner     bool                              `yaml:"no-spinner" mapstructure:"no-spinner" doc:"Disable spinner during operations"`
	Quiet         bool                              `yaml:"quiet" mapstructure:"quiet" doc:"Suppress non-essential output"`
	Debug         bool                              `yaml:"debug" mapstructure:"debug"`
	Timeout       time.Duration                     `yaml:"timeout" mapstructure:"timeout" doc:"Overall command timeout (e.g. 30s, 2m)"`
	RetryStrategy RetryStrategy                     `yaml:"retry-strategy" mapstructure:"retry-strategy"`
	Templates     map[TemplateEntity]TemplateConfig `yaml:"templates" mapstructure:"templates"`
	Search        SearchConfig                      `yaml:"search" mapstructure:"search"`
	DefaultTZ     datetime.TimeZone                 `yaml:"default-tz" mapstructure:"default-tz" doc:"Default timezone for timestamps"`
}

// SearchConfig contains defaults for search pagination.
type SearchConfig struct {
	// PageSize sets the default number of results per page for search.
	// Must be >= 1.
	PageSize int64 `yaml:"page-size" mapstructure:"page-size" doc:"Default number of results per page (must be >= 1)"`
	// MaxPages limits the number of pages fetched. Set to -1 for unlimited.
	// 0 is invalid and will be rejected.
	MaxPages int64 `yaml:"max-pages" mapstructure:"max-pages" doc:"Number of pages to fetch (max is 100)"`
}

var defaultConfig = &Config{
	OutputFormat: formatter.OutputFormatJSON,
	NoColor:      false,
	NoSpinner:    false,
	Quiet:        false,
	Debug:        false,
	Timeout:      30 * time.Second,
	RetryStrategy: RetryStrategy{
		MaxAttempts: 2,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Backoff:     BackoffFixed,
	},
	DefaultTZ: datetime.TimeZoneUTC,
	Templates: map[TemplateEntity]TemplateConfig{
		// will be potentially updated at runtime
		TemplateEntityHost:         {},
		TemplateEntityCertificate:  {},
		TemplateEntityWebProperty:  {},
		TemplateEntitySearchResult: {},
	},
	Search: SearchConfig{
		PageSize: 100,
		MaxPages: 1,
	},
}

const (
	noColorKey   = "no-color"
	noSpinnerKey = "no-spinner"
	quietKey     = "quiet"
	debugKey     = "debug"
	timeoutKey   = "timeout"
)

func New(dataDir string) (*Config, cenclierrors.CencliError) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dataDir)
	viper.SetEnvPrefix("CENCLI")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	configPath := filepath.Join(dataDir, "config.yaml")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, newInvalidConfigError(fmt.Errorf("failed to read config file: %w", err).Error())
		}

		if err := setViperDefaults(defaultConfig); err != nil {
			return nil, err
		}

		if err := viper.WriteConfigAs(configPath); err != nil {
			return nil, newInvalidConfigError(fmt.Errorf("failed to write config file: %w", err).Error())
		}

	} else {
		// Config file was read successfully, but we still need to set defaults for any missing keys
		if err := setViperDefaults(defaultConfig); err != nil {
			return nil, err
		}
	}

	cfg := &Config{}
	err := cfg.Unmarshal()
	if err != nil {
		return nil, err
	}

	// Initialize templates after config is loaded
	if err := initTemplates(dataDir, cfg); err != nil {
		var cencliErr cenclierrors.CencliError
		if errors.As(err, &cencliErr) {
			return nil, cencliErr
		}
		return nil, newInvalidConfigError(fmt.Errorf("failed to initialize templates: %w", err).Error())
	}

	// Write the updated config back to the file to persist template paths
	if err := viper.WriteConfig(); err != nil {
		return nil, newInvalidConfigError(fmt.Errorf("failed to write updated config file: %w", err).Error())
	}

	// Add doc comments to the config file
	if err := addDocCommentsToYAML(configPath, cfg); err != nil {
		return nil, newInvalidConfigError(fmt.Errorf("failed to add doc comments to config file: %w", err).Error())
	}

	return cfg, nil
}

func (c *Config) Unmarshal() cenclierrors.CencliError {
	hooks := mapstructure.ComposeDecodeHookFunc(
		rejectNumericDurationHookFunc(),
		mapstructure.StringToUint64HookFunc(),
		validateUint64HookFunc(),
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	if err := viper.Unmarshal(c, viper.DecodeHook(hooks)); err != nil {
		return newInvalidConfigError(fmt.Errorf("failed to unmarshal config: %w", err).Error())
	}

	return nil
}

// BindGlobalFlags binds all global configuration flags to viper.
// This should be called on the root command.
func BindGlobalFlags(persistentFlags *pflag.FlagSet) error {
	if err := addPersistentBoolAndBind(persistentFlags, noColorKey, false, "disable ANSI colors and styles", ""); err != nil {
		return fmt.Errorf("failed to bind no-color flag: %w", err)
	}
	if err := addPersistentBoolAndBind(persistentFlags, noSpinnerKey, false, "disable spinner during operations", ""); err != nil {
		return fmt.Errorf("failed to bind no-spinner flag: %w", err)
	}
	if err := addPersistentBoolAndBind(persistentFlags, quietKey, false, "suppress non-essential output", "q"); err != nil {
		return fmt.Errorf("failed to bind quiet flag: %w", err)
	}
	if err := addPersistentBoolAndBind(persistentFlags, debugKey, false, "enable debug logging", ""); err != nil {
		return fmt.Errorf("failed to bind debug flag: %w", err)
	}
	if err := addPersistentDurationAndBind(persistentFlags, timeoutKey, defaultConfig.Timeout, "overall command timeout (e.g. 30s, 2m)"); err != nil {
		return fmt.Errorf("failed to bind timeout flag: %w", err)
	}
	if err := formatter.BindOutputFormat(persistentFlags); err != nil {
		return fmt.Errorf("failed to bind output-format flag: %w", err)
	}
	return nil
}

// addPersistentBoolAndBind defines a persistent boolean flag and binds it to viper using the same key.
// It returns any error produced during viper binding.
func addPersistentBoolAndBind(persistentFlags *pflag.FlagSet, name string, defaultValue bool, usage string, short string) error {
	if short != "" {
		persistentFlags.BoolP(name, short, defaultValue, usage)
	} else {
		persistentFlags.Bool(name, defaultValue, usage)
	}
	return viper.BindPFlag(name, persistentFlags.Lookup(name))
}

// addPersistentDurationAndBind defines a persistent duration flag and binds it to viper using the same key.
// Duration values accept forms like "30s", "2m", etc.
func addPersistentDurationAndBind(persistentFlags *pflag.FlagSet, name string, defaultValue time.Duration, usage string) error {
	persistentFlags.Duration(name, defaultValue, usage)
	return viper.BindPFlag(name, persistentFlags.Lookup(name))
}

// rejectNumericDurationHookFunc disallows numeric values for time.Duration fields,
// forcing users to include an explicit unit (e.g., "30s", "2m").
func rejectNumericDurationHookFunc() mapstructure.DecodeHookFuncType {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if to == reflect.TypeOf(time.Duration(0)) {
			switch from.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64:
				return nil, fmt.Errorf("missing unit in duration")
			}
		}
		return data, nil
	}
}

// validateUint64HookFunc validates that values being decoded to uint64 are not negative.
// This prevents negative values from wrapping around to large positive numbers.
func validateUint64HookFunc() mapstructure.DecodeHookFuncType {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if to.Kind() == reflect.Uint64 {
			switch from.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				// Check if the signed integer is negative
				val := reflect.ValueOf(data)
				if val.Int() < 0 {
					return nil, fmt.Errorf("value cannot be negative")
				}
			case reflect.Float32, reflect.Float64:
				// Check if the float is negative
				val := reflect.ValueOf(data)
				if val.Float() < 0 {
					return nil, fmt.Errorf("value cannot be negative")
				}
			}
		}
		return data, nil
	}
}
