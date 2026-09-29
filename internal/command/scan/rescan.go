package scan

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	appscan "github.com/censys/cencli/internal/app/scan"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	cmdutil "github.com/censys/cencli/internal/pkg/input"
)

const rescanCmdName = "rescan"

// RescanCommand implements `scan rescan <hostname:port>`, requesting one live
// rescan of a web property and optionally polling it until it completes.
type RescanCommand struct {
	*command.BaseCommand
	// services the command uses
	scanSvc appscan.Service
	// flags the command uses
	flags rescanCommandFlags
	// state - populated by PreRun
	orgID       identifiers.OrganizationID
	webProperty assets.WebPropertyID
	wait        bool
	timeout     mo.Option[time.Duration]
	// result stores the scan for rendering
	result appscan.Result
}

type rescanCommandFlags struct {
	orgID   flags.OrgIDFlag
	wait    flags.BoolFlag
	timeout flags.HumanDurationFlag
}

var _ command.Command = (*RescanCommand)(nil)

func NewRescanCommand(cmdContext *command.Context) *RescanCommand {
	return &RescanCommand{BaseCommand: command.NewBaseCommand(cmdContext)}
}

func (c *RescanCommand) Use() string {
	return fmt.Sprintf("%s <hostname:port>", rescanCmdName)
}

func (c *RescanCommand) Short() string {
	return "Request a live rescan of a web property (10 credits)"
}

func (c *RescanCommand) Long() string {
	return `Request a live rescan of a web property. Costs 10 credits per accepted request and requires an organization.

The request is sent once and never retried automatically. If it is interrupted or the API fails mid-request, the rescan may still have been accepted and charged, so check ` + "`censys org credits`" + ` before trying again.

Without --wait the command prints the new scan's ID and exits. Use --wait to poll until the scan completes; waiting exits non-zero if no task produced results. Once a scan completes, ` + "`censys view <hostname:port>`" + ` shows the fresh data.

Only web properties can be rescanned; hosts and certificates are not supported.`
}

func (c *RescanCommand) Examples() []string {
	return []string{
		"example.com:443 # Request a rescan and print the scan ID",
		"example.com:443 --wait # Request a rescan and poll until it completes",
		"example.com:443 --wait --timeout 5m # Give up waiting after 5 minutes",
	}
}

func (c *RescanCommand) Args() command.PositionalArgs {
	return command.ExactArgs(1)
}

func (c *RescanCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *RescanCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *RescanCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.wait = flags.NewBoolFlag(c.Flags(), "wait", "w", false, "poll until the scan completes")
	c.flags.timeout = flags.NewHumanDurationFlag(c.Flags(), false, "timeout", "",
		mo.Some(defaultWaitTimeout), "how long to wait before giving up (requires --wait) - use 0 for no limit")
	return nil
}

func (c *RescanCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.webProperty, err = parseRescanTarget(args[0])
	if err != nil {
		return err
	}

	c.wait, c.timeout, err = command.ParseWaitFlags(cmd, c.flags.wait, c.flags.timeout)
	if err != nil {
		return err
	}

	c.scanSvc, err = c.ScanService()
	if err != nil {
		return err
	}

	flagOrgID, err := c.flags.orgID.Value()
	if err != nil {
		return err
	}
	c.orgID, err = c.ResolveRequiredOrgID(cmd, flagOrgID)
	return err
}

// parseRescanTarget accepts exactly one web property. Hosts and certificates
// are recognised so the error can say what was given instead.
func parseRescanTarget(arg string) (assets.WebPropertyID, cenclierrors.CencliError) {
	classifier := assets.NewAssetClassifier(cmdutil.SplitString(arg)...)
	assetType, err := classifier.AssetType()
	if err != nil {
		return assets.WebPropertyID{}, err
	}
	if classifier.KnownAssetCount() != 1 {
		return assets.WebPropertyID{}, assets.NewTooManyAssetsError(classifier.KnownAssetCount(), 1)
	}
	if assetType != assets.AssetTypeWebProperty {
		return assets.WebPropertyID{}, NewUnsupportedScanAssetError(arg, assetType)
	}
	return classifier.WebPropertyIDs()[0], nil
}

func (c *RescanCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"subcommand", rescanCmdName,
		"wait", c.wait,
		"timeout_set", c.timeout.IsPresent(),
	)
	quiet := c.Config().Quiet

	err := c.WithProgress(cmd.Context(), logger, "Requesting rescan...",
		func(pctx context.Context) cenclierrors.CencliError {
			var rescanErr cenclierrors.CencliError
			c.result, rescanErr = c.scanSvc.Rescan(pctx, appscan.RescanParams{
				OrgID:       c.orgID,
				WebProperty: c.webProperty,
			})
			return rescanErr
		})
	if err != nil {
		if cenclierrors.IsInterrupted(err) {
			printRescanMayBeChargedNote(quiet)
		}
		logger.Debug("rescan failed", "error", err)
		return err
	}
	scanID := c.result.Scan.ID

	if !c.wait {
		c.PrintAppResponseMeta(c.result.Meta)
		if renderErr := c.PrintData(c, c.result.Scan); renderErr != nil {
			return renderErr
		}
		printScanTrackHint(quiet, scanID)
		return nil
	}

	// Show the latest state seen, or the accepted request if no poll succeeded,
	// so the scan ID is never lost.
	waited, waitErr := waitForScan(cmd.Context(), c.BaseCommand, logger, c.scanSvc, appscan.WaitParams{
		OrgID:   c.orgID,
		ScanID:  scanID,
		Timeout: c.timeout,
	})
	if waited.Scan != nil {
		c.result = waited
	}
	c.PrintAppResponseMeta(c.result.Meta)
	if renderErr := c.PrintData(c, c.result.Scan); renderErr != nil {
		return renderErr
	}
	return reportWaitOutcome(quiet, scanID, c.result.Scan, waitErr)
}

func (c *RescanCommand) RenderShort() cenclierrors.CencliError {
	return renderTrackedScan(c.result.Scan)
}
