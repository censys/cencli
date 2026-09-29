package scan

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	appscan "github.com/censys/cencli/internal/app/scan"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
)

const getCmdName = "get"

// GetCommand implements `scan get <scan-id>`, reading one tracked scan and
// optionally polling it until it completes.
type GetCommand struct {
	*command.BaseCommand
	// services the command uses
	scanSvc appscan.Service
	// flags the command uses
	flags getCommandFlags
	// state - populated by PreRun
	orgID   identifiers.OrganizationID
	scanID  string
	wait    bool
	timeout mo.Option[time.Duration]
	// result stores the scan for rendering
	result appscan.Result
}

type getCommandFlags struct {
	orgID   flags.OrgIDFlag
	wait    flags.BoolFlag
	timeout flags.HumanDurationFlag
}

var _ command.Command = (*GetCommand)(nil)

func NewGetCommand(cmdContext *command.Context) *GetCommand {
	return &GetCommand{BaseCommand: command.NewBaseCommand(cmdContext)}
}

func (c *GetCommand) Use() string {
	return fmt.Sprintf("%s <scan-id>", getCmdName)
}

func (c *GetCommand) Short() string {
	return "Show the status of a tracked scan"
}

func (c *GetCommand) Long() string {
	return `Show the status of a tracked scan by its ID. Works for any tracked scan, including ones started outside the CLI. Reading a scan is free.

Use --wait to poll until the scan completes. Waiting exits non-zero if no task produced results; without --wait the command reports the current state and exits 0, even for a scan that was rejected.`
}

func (c *GetCommand) Examples() []string {
	return []string{
		"<scan-id> # Show a scan's current status",
		"<scan-id> --wait # Poll until the scan completes",
		"<scan-id> --output-format json # Machine-readable output",
	}
}

func (c *GetCommand) Args() command.PositionalArgs {
	return command.ExactArgs(1)
}

func (c *GetCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *GetCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *GetCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.wait = flags.NewBoolFlag(c.Flags(), "wait", "w", false, "poll until the scan completes")
	c.flags.timeout = flags.NewHumanDurationFlag(c.Flags(), false, "timeout", "",
		mo.Some(defaultWaitTimeout), "how long to wait before giving up (requires --wait) - use 0 for no limit")
	return nil
}

func (c *GetCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.scanID, err = appscan.RequireScanID(strings.TrimSpace(args[0]))
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

func (c *GetCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"subcommand", getCmdName,
		"wait", c.wait,
		"timeout_set", c.timeout.IsPresent(),
	)

	if !c.wait {
		err := c.WithProgress(cmd.Context(), logger, "Fetching scan...",
			func(pctx context.Context) cenclierrors.CencliError {
				var getErr cenclierrors.CencliError
				c.result, getErr = c.scanSvc.Get(pctx, appscan.GetParams{OrgID: c.orgID, ScanID: c.scanID})
				return getErr
			})
		if err != nil {
			logger.Debug("get scan failed", "error", err)
			return err
		}
		c.PrintAppResponseMeta(c.result.Meta)
		return c.PrintData(c, c.result.Scan)
	}

	var waitErr cenclierrors.CencliError
	c.result, waitErr = waitForScan(cmd.Context(), c.BaseCommand, logger, c.scanSvc, appscan.WaitParams{
		OrgID:   c.orgID,
		ScanID:  c.scanID,
		Timeout: c.timeout,
	})
	// A wait that ends early still shows the last state it saw.
	if c.result.Scan != nil {
		c.PrintAppResponseMeta(c.result.Meta)
		if renderErr := c.PrintData(c, c.result.Scan); renderErr != nil {
			return renderErr
		}
	}
	return reportWaitOutcome(c.Config().Quiet, c.scanID, c.result.Scan, waitErr)
}

func (c *GetCommand) RenderShort() cenclierrors.CencliError {
	return renderTrackedScan(c.result.Scan)
}
