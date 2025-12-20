package history

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/history"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	cmdutil "github.com/censys/cencli/internal/pkg/input"
	"github.com/censys/cencli/internal/pkg/tape"
)

const (
	cmdName = "history"
)

// Command implements the `history` CLI command.
type Command struct {
	*command.BaseCommand
	// flags
	flags historyCommandFlags
	// state populated during PreRun
	assets    *assets.AssetClassifier
	assetType assets.AssetType
	assetID   string // single asset ID string
	start     time.Time
	end       time.Time
	orgID     mo.Option[identifiers.OrganizationID]
	// services
	historySvc history.Service
}

type historyCommandFlags struct {
	start    flags.TimestampFlag
	end      flags.TimestampFlag
	duration flags.HumanDurationFlag
	orgID    flags.OrgIDFlag
}

var _ command.Command = (*Command)(nil)

// NewHistoryCommand constructs a history command bound to the provided context.
func NewHistoryCommand(ctx *command.Context) *Command {
	return &Command{BaseCommand: command.NewBaseCommand(ctx)}
}

func (c *Command) Use() string { return fmt.Sprintf("%s <asset>", cmdName) }

func (c *Command) Short() string {
	return "Retrieve historical data for hosts, web properties, and certificates"
}

func (c *Command) Long() string {
	return "Explore how hosts, web properties, and certificates have changed over time.\n\n" +
		"Returns raw data showing events, observations, and snapshots for the specified time window.\n\n" +
		"To retrieve certificate history, you must have access to the Threat Hunting module."
}

func (c *Command) Examples() []string {
	return []string{
		"8.8.8.8 --start 2025-01-01T00:00:00Z --duration 30d",
		"example.com:443 --end 2025-05-31T00:00:00Z --duration 72d",
		"56a06a23... --start 2025-01-01T00:00:00Z --end 2025-01-31T00:00:00Z",
		"example.com:443 --duration 7d",
		"8.8.8.8 --duration 14d",
	}
}

func (c *Command) Init() error {
	// Flags
	c.flags.start = flags.NewTimestampFlag(c.Flags(), false, "start", "s", mo.None[time.Time](), "start time")
	c.flags.end = flags.NewTimestampFlag(c.Flags(), false, "end", "e", mo.None[time.Time](), "end time")
	c.flags.duration = flags.NewHumanDurationFlag(c.Flags(), false, "duration", "d", mo.Some(7*24*time.Hour), "time window (e.g., 1d, 1w, 1y, 2h). Defaults to 7d")
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	return nil
}

func (c *Command) Args() command.PositionalArgs { return command.ExactArgs(1) }

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeData
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeData}
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// gather assets
	rawAssets := cmdutil.SplitString(args[0])
	c.assets = assets.NewAssetClassifier(rawAssets...)
	var err cenclierrors.CencliError
	c.assetType, err = c.assets.AssetType()
	if err != nil {
		return err
	}
	if c.assets.KnownAssetCount() != 1 {
		return assets.NewTooManyAssetsError(c.assets.KnownAssetCount(), 1)
	}
	c.assetID = c.assets.KnownAssetIDs()[0]

	// resolve time window
	startOpt, err := c.flags.start.Value(c.Config().DefaultTZ)
	if err != nil {
		return err
	}
	endOpt, err := c.flags.end.Value(c.Config().DefaultTZ)
	if err != nil {
		return err
	}
	durationOpt, err := c.flags.duration.Value()
	if err != nil {
		return err
	}
	c.start, c.end, err = resolveTimeWindow(startOpt, endOpt, durationOpt)
	if err != nil {
		return err
	}
	logger := c.Logger(cmdName)
	logger.Debug("Time window", "start", c.start.Format(time.RFC3339), "end", c.end.Format(time.RFC3339))
	// parse org id
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}
	// resolve required services
	c.historySvc, err = c.HistoryService()
	if err != nil {
		return err
	}

	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"assetID", c.assetID,
		"assetType", c.assetType.String(),
		"start", c.start.Format(time.RFC3339),
		"end", c.end.Format(time.RFC3339),
	)

	var result interface{}
	err := c.WithProgress(
		cmd.Context(),
		logger,
		fmt.Sprintf("Fetching history for %s...", c.assetID),
		func(pctx context.Context) cenclierrors.CencliError {
			// Service will report detailed progress during fetch (pagination, day-by-day, etc.)
			var fetchErr cenclierrors.CencliError
			switch c.assetType {
			case assets.AssetTypeHost:
				result, fetchErr = c.historySvc.GetHostHistory(pctx, c.orgID, c.assets.HostIDs()[0], c.start, c.end)
			case assets.AssetTypeCertificate:
				result, fetchErr = c.historySvc.GetCertificateHistory(pctx, c.orgID, c.assets.CertificateIDs()[0], c.start, c.end)
			case assets.AssetTypeWebProperty:
				result, fetchErr = c.historySvc.GetWebPropertyHistory(pctx, c.orgID, c.assets.WebPropertyIDs()[0], c.start, c.end)
			default:
				return cenclierrors.NewCencliError(fmt.Errorf("unsupported asset type: %s", c.assetType))
			}
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("history fetch failed", "error", err)
		return err
	}

	// Print response metadata and output raw JSON (even if partial)
	var partialError cenclierrors.CencliError
	switch c.assetType {
	case assets.AssetTypeHost:
		hostResult := result.(history.HostHistoryResult)
		c.PrintAppResponseMeta(hostResult.Meta)
		if printErr := c.PrintData(c, hostResult.Events); printErr != nil {
			return printErr
		}
		partialError = hostResult.PartialError
	case assets.AssetTypeCertificate:
		certResult := result.(history.CertificateHistoryResult)
		c.PrintAppResponseMeta(certResult.Meta)
		if printErr := c.PrintData(c, certResult.Ranges); printErr != nil {
			return printErr
		}
		partialError = certResult.PartialError
	case assets.AssetTypeWebProperty:
		webPropResult := result.(history.WebPropertyHistoryResult)
		c.PrintAppResponseMeta(webPropResult.Meta)
		if printErr := c.PrintData(c, webPropResult.Snapshots); printErr != nil {
			return printErr
		}
		partialError = webPropResult.PartialError
	default:
		return cenclierrors.NewCencliError(fmt.Errorf("unsupported asset type: %s", c.assetType))
	}

	// If there was a partial error, print it to stderr after rendering the data
	if partialError != nil {
		formatter.PrintError(partialError, cmd)
	}

	return nil
}

// resolveTimeWindow determines the start and end times based on the provided flags.
func resolveTimeWindow(
	startOpt mo.Option[time.Time],
	endOpt mo.Option[time.Time],
	durationOpt mo.Option[time.Duration],
) (time.Time, time.Time, cenclierrors.CencliError) {
	var start, end time.Time
	var duration time.Duration

	if durationOpt.IsPresent() {
		duration = durationOpt.MustGet()
	} else {
		duration = 7 * 24 * time.Hour // default to 7 days
	}

	hasStart := startOpt.IsPresent()
	hasEnd := endOpt.IsPresent()

	if hasStart {
		start = startOpt.MustGet()
	}
	if hasEnd {
		end = endOpt.MustGet()
	}

	switch {
	case hasStart && hasEnd:
		// both start and end are set, use them as is
		if end.Before(start) {
			return time.Time{}, time.Time{}, newInvalidTimeWindowError("end time must be after start time")
		}
		return start, end, nil
	case hasStart:
		// only start is set, calculate end
		return start, start.Add(duration), nil
	case hasEnd:
		// only end is set, calculate start
		return end.Add(-duration), end, nil
	default:
		// neither is set, use now as end and calculate start
		end = time.Now().UTC()
		return end.Add(-duration), end, nil
	}
}

func (*Command) Tapes(recorder *tape.Recorder) []tape.Tape {
	return []tape.Tape{
		tape.NewTape("history",
			tape.DefaultTapeConfig(),
			recorder.Type(
				"history 8.8.8.8 --duration 2h",
				tape.WithSleepAfter(12),
			),
		),
	}
}
