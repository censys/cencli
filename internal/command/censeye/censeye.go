package censeye

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/censeye"
	"github.com/censys/cencli/internal/app/progress"
	"github.com/censys/cencli/internal/app/view"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/input"
	"github.com/censys/cencli/internal/pkg/tape"
)

const (
	cmdName = "censeye"

	defaultRarityMin = 2
	defaultRarityMax = 100
)

// Command implements the `censeye` CLI command.
// It analyzes a single host, compiles field-value rules, retrieves counts
// from the threat hunting service, and prints queries along with a rarity
// indicator based on configurable bounds.
type Command struct {
	*command.BaseCommand
	// services the command uses
	censeyeSvc censeye.Service
	viewSvc    view.Service
	// flags the command uses
	flags censeyeCommandFlags
	// state parsed from flags/args
	orgID       mo.Option[identifiers.OrganizationID]
	rarityMin   uint64
	rarityMax   uint64
	interactive bool
	includeURL  bool
	hostID      string
	// result stored for rendering
	result censeye.InvestigateHostResult
}

type censeyeCommandFlags struct {
	orgID       flags.OrgIDFlag
	inputFile   flags.FileFlag
	rarityMin   flags.IntegerFlag
	rarityMax   flags.IntegerFlag
	interactive flags.BoolFlag
	includeURL  flags.BoolFlag
}

var _ command.Command = (*Command)(nil)

// NewCenseyeCommand constructs a new Command with the provided context.
func NewCenseyeCommand(ctx *command.Context) *Command {
	return &Command{BaseCommand: command.NewBaseCommand(ctx)}
}

func (c *Command) Use() string { return cmdName + " <asset>" }
func (c *Command) Short() string {
	return "Analyze a host and generate pivotable queries with rarity bounds"
}
func (c *Command) Args() command.PositionalArgs { return command.RangeArgs(0, 1) }

// Long returns a detailed description of the command and its flags.
func (c *Command) Long() string {
	return "CensEye helps you identify assets on the internet that share a specific key-value pair with the asset you are currently viewing. It extracts data values then shows how many other assets present the same value. This allows you to pivot into related infrastructure and begin building queries based on shared characteristics."
}

// Examples demonstrates typical usage patterns.
func (c *Command) Examples() []string {
	return []string{
		"8.8.8.8",
		"--rarity-min 2 --rarity-max 25 1.1.1.1",
		"--interactive 192.168.1.1",
		"--output-format json --include-url 192.168.1.1",
	}
}

func (c *Command) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.inputFile = flags.NewFileFlag(
		c.Flags(),
		false,
		"input-file",
		"i",
		"file to read the assets from. Overrides the positional argument.",
	)
	c.flags.rarityMin = flags.NewIntegerFlag(
		c.Flags(),
		false, // not required
		"rarity-min",
		"m",
		mo.Some(int64(defaultRarityMin)),
		"minimum host count for interesting results (must be non-zero)",
		mo.Some(int64(1)), // min value
		mo.None[int64](),  // no max value
	)
	c.flags.rarityMax = flags.NewIntegerFlag(
		c.Flags(),
		false, // not required
		"rarity-max",
		"M",
		mo.Some(int64(defaultRarityMax)),
		"maximum host count for interesting results (must be non-zero)",
		mo.Some(int64(1)), // min value
		mo.None[int64](),  // no max value
	)
	c.flags.interactive = flags.NewBoolFlag(
		c.Flags(),
		"interactive",
		"I",
		false,
		"display results in an interactive table (TUI)",
	)
	c.flags.includeURL = flags.NewBoolFlag(
		c.Flags(),
		"include-url",
		"",
		false,
		"include a Platform search URL in the output",
	)
	return nil
}

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeData, command.OutputTypeShort}
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}
	// validate the hostID
	var providedAssets []string
	if c.flags.inputFile.IsSet() {
		lines, err := c.flags.inputFile.Lines(cmd)
		if err != nil {
			return err
		}
		providedAssets = lines
	} else {
		providedAssets = args
	}
	if len(providedAssets) == 0 {
		return assets.NewNoAssetsError()
	}
	if len(providedAssets) > 1 {
		return assets.NewTooManyAssetsError(len(providedAssets), 1)
	}
	c.hostID = providedAssets[0]
	// validate rarity flags
	minVal, err := c.flags.rarityMin.Value()
	if err != nil {
		return err
	}
	if !minVal.IsPresent() {
		return newInvalidRarityFlagError("rarity-min", "value is required")
	}
	c.rarityMin = uint64(minVal.MustGet()) // already asserted non-zero

	maxVal, err := c.flags.rarityMax.Value()
	if err != nil {
		return err
	}
	if !maxVal.IsPresent() {
		return newInvalidRarityFlagError("rarity-max", "value is required")
	}
	c.rarityMax = uint64(maxVal.MustGet()) // already asserted non-zero

	if c.rarityMin > c.rarityMax {
		return newInvalidRarityFlagError("rarity-min", "must be less than or equal to rarity-max")
	}
	// validate interactive (if present)
	c.interactive, err = c.flags.interactive.Value()
	if err != nil {
		return err
	}
	// validate includeURL (if present)
	c.includeURL, err = c.flags.includeURL.Value()
	if err != nil {
		return err
	}
	// resolve services
	err = c.resolveServices()
	if err != nil {
		return err
	}
	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With("hostID", c.hostID)

	if err := c.WithProgress(
		cmd.Context(),
		logger,
		c.fetchMessage(),
		func(pctx context.Context) cenclierrors.CencliError {
			asset, fetchErr := c.fetchAsset(pctx, c.hostID)
			if fetchErr != nil {
				return fetchErr
			}
			host, ok := asset.(*assets.Host)
			if !ok {
				return cenclierrors.NewCencliError(fmt.Errorf("expected host asset, got %T", asset))
			}

			progress.ReportMessage(pctx, progress.StageProcess, "Investigating host...")
			res, investigateErr := c.censeyeSvc.InvestigateHost(pctx, c.orgID, host, c.rarityMin, c.rarityMax)
			if investigateErr != nil {
				return investigateErr
			}
			c.result = res
			return nil
		},
	); err != nil {
		return err
	}

	// Print response metadata
	c.PrintAppResponseMeta(c.result.Meta)

	return c.PrintData(c, c.result.Entries)
}

// RenderShort renders the censeye results as a human-readable table.
// If the interactive flag is set, displays an interactive TUI table.
// Otherwise, displays a static styled table with pivots.
func (c *Command) RenderShort() cenclierrors.CencliError {
	if c.interactive {
		return c.showInteractiveTable(c.result)
	}
	// Default: show raw table
	return c.showRawTable(c.result)
}

func (c *Command) resolveServices() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.viewSvc, err = c.ViewService()
	if err != nil {
		return err
	}
	c.censeyeSvc, err = c.CenseyeService()
	if err != nil {
		return err
	}
	return nil
}

// fetchAsset fetches an asset from the view service.
func (c *Command) fetchAsset(ctx context.Context, arg string) (assets.Asset, cenclierrors.CencliError) {
	parsed := input.SplitString(arg)
	classifier := assets.NewAssetClassifier(parsed...)
	assetType, err := classifier.AssetType()
	if err != nil {
		return nil, err
	}
	var result assets.Asset
	switch assetType {
	case assets.AssetTypeHost:
		hostIDs := classifier.HostIDs()
		if len(hostIDs) != 1 {
			return nil, assets.NewTooManyAssetsError(len(hostIDs), 1)
		}
		hostID := hostIDs[0]
		hosts, err := c.viewSvc.GetHosts(ctx, c.orgID, []assets.HostID{hostID}, mo.None[time.Time]())
		if err != nil {
			return nil, err
		}
		if len(hosts.Hosts) == 0 {
			return nil, newHostNotFoundError(hostID.String())
		}
		result = hosts.Hosts[0]
	default:
		return nil, newErrorAssetTypeNotSupportedError(assetType)
	}
	return result, nil
}

// fetchMessage returns a contextual message for the fetch stage indicating input source.
func (c *Command) fetchMessage() string {
	baseMsg := fmt.Sprintf("Fetching host %s", c.hostID)
	if c.flags.inputFile.IsSet() {
		value, _ := c.flags.inputFile.Value()
		if value == input.StdInSentinel {
			return baseMsg + " (from stdin)..."
		}
		return baseMsg + " (from file)..."
	}
	return baseMsg + "..."
}

func (*Command) Tapes(recorder *tape.Recorder) []tape.Tape {
	bigConfig := &tape.Config{
		Width:    2000,
		Height:   1600,
		FontSize: 20,
	}
	return []tape.Tape{
		tape.NewTape("censeye-interactive",
			tape.DefaultTapeConfig(),
			recorder.Type(
				"censeye 145.131.8.169 -i",
				tape.WithSleepAfter(10),
			),
			recorder.SpamPress("j", 50),
			recorder.Sleep(5),
		),
		tape.NewTape("censeye-json",
			tape.DefaultTapeConfig(),
			recorder.Type(
				"censeye 145.131.8.169 --output-format json --include-url",
				tape.WithSleepAfter(15),
			),
		),
		tape.NewTape("censeye",
			bigConfig,
			recorder.Type(
				"censeye 145.131.8.169",
				tape.WithSleepAfter(11),
			),
		),
	}
}
