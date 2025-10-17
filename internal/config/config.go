package config

import (
	"errors"
	"fmt"
	"path/filepath"
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
	Spinner       SpinnerConfig                     `yaml:"spinner" mapstructure:"spinner"`
	Quiet         bool                              `yaml:"quiet" mapstructure:"quiet" doc:"Suppress non-essential output"`
	Debug         bool                              `yaml:"debug" mapstructure:"debug"`
	Timeouts      TimeoutConfig                     `yaml:"timeouts" mapstructure:"timeouts"`
	RetryStrategy RetryStrategy                     `yaml:"retry-strategy" mapstructure:"retry-strategy"`
	Templates     map[TemplateEntity]TemplateConfig `yaml:"templates" mapstructure:"templates"`
	Search        SearchConfig                      `yaml:"search" mapstructure:"search"`
	DefaultTZ     datetime.TimeZone                 `yaml:"default-tz" mapstructure:"default-tz" doc:"Default timezone for timestamps"`
}

var defaultConfig = &Config{
	OutputFormat:  formatter.OutputFormatJSON,
	NoColor:       false,
	Spinner:       defaultSpinnerConfig,
	Quiet:         false,
	Debug:         false,
	Timeouts:      defaultTimeoutConfig,
	RetryStrategy: defaultRetryStrategy,
	DefaultTZ:     datetime.TimeZoneUTC,
	Templates:     defaultTemplateConfig,
	Search:        defaultSearchConfig,
}

const (
	noColorKey     = "no-color"
	noSpinnerKey   = "no-spinner"
	quietKey       = "quiet"
	debugKey       = "debug"
	timeoutHTTPKey = "timeout-http"
)

func New(dataDir string) (*Config, cenclierrors.CencliError) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dataDir)
	viper.SetEnvPrefix("CENCLI")
	viper.SetEnvKeyReplacer(strings.NewReplacer(
		".", "_",
		"-", "_",
	))
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
		rejectNegativeDurationHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToUint64HookFunc(),
		validateUint64HookFunc(),
		mapstructure.TextUnmarshallerHookFunc(),
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
	// Bind no-spinner flag to spinner.disabled config path
	if err := addPersistentBoolAndBindToPath(persistentFlags, noSpinnerKey, "spinner.disabled", defaultConfig.Spinner.Disabled, "disable spinner during operations", ""); err != nil {
		return fmt.Errorf("failed to bind no-spinner flag: %w", err)
	}
	if err := addPersistentBoolAndBind(persistentFlags, quietKey, false, "suppress non-essential output", "q"); err != nil {
		return fmt.Errorf("failed to bind quiet flag: %w", err)
	}
	if err := addPersistentBoolAndBind(persistentFlags, debugKey, false, "enable debug logging", ""); err != nil {
		return fmt.Errorf("failed to bind debug flag: %w", err)
	}
	// Bind timeout-http flag to timeouts.http config path
	if err := addPersistentDurationAndBindToPath(persistentFlags, timeoutHTTPKey, "timeouts.http", defaultConfig.Timeouts.HTTP, "per-request timeout for HTTP requests (e.g. 10s, 1m) - use 0 to disable"); err != nil {
		return fmt.Errorf("failed to bind timeout-http flag: %w", err)
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

// addPersistentBoolAndBindToPath defines a persistent boolean flag and binds it to viper using a different config path.
// This is useful when the flag name doesn't match the nested config structure.
func addPersistentBoolAndBindToPath(persistentFlags *pflag.FlagSet, flagName string, viperPath string, defaultValue bool, usage string, short string) error {
	if short != "" {
		persistentFlags.BoolP(flagName, short, defaultValue, usage)
	} else {
		persistentFlags.Bool(flagName, defaultValue, usage)
	}
	return viper.BindPFlag(viperPath, persistentFlags.Lookup(flagName))
}

// addPersistentDurationAndBindToPath defines a persistent duration flag and binds it to viper using a different config path.
// This is useful when the flag name doesn't match the nested config structure.
func addPersistentDurationAndBindToPath(persistentFlags *pflag.FlagSet, flagName string, viperPath string, defaultValue time.Duration, usage string) error {
	persistentFlags.Duration(flagName, defaultValue, usage)
	return viper.BindPFlag(viperPath, persistentFlags.Lookup(flagName))
}
