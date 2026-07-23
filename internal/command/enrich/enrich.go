package enrich

import (
	"context"
	"strings"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/enrich"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/formatter/short"
	"github.com/censys/cencli/internal/pkg/input"
	"github.com/censys/cencli/internal/pkg/tape"
)

const (
	cmdName = "enrich"
)

type Command struct {
	*command.BaseCommand
	// services the command uses
	enrichSvc enrich.Service
	// flags the command uses
	flags enrichCommandFlags
	// state - populated by PreRun
	hostIDs []assets.HostID
	orgID   mo.Option[identifiers.OrganizationID]
	// result stores the enrichment result for rendering
	result enrich.Result
}

type enrichCommandFlags struct {
	orgID     flags.OrgIDFlag
	inputFile flags.FileFlag
}

var _ command.Command = (*Command)(nil)

func NewEnrichCommand(cmdContext *command.Context) *Command {
	return &Command{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *Command) Use() string {
	return cmdName + " <ip>"
}

func (c *Command) Short() string {
	return "Enrich host IPs with curated Censys data for high-volume SOC lookups"
}

func (c *Command) Long() string {
	return "Enrich one or more host IPs using the Censys Host Enrichment API: a lightweight, " +
		"curated, fixed subset of host IPv4/IPv6 data — location, autonomous system, whois, DNS, " +
		"labels, greynoise, reputation, network and privacy classifications, services, and " +
		"third-party verdicts.\n" +
		"Purpose-built for high-volume, automated lookups in SOC environments such as SIEM and " +
		"SOAR integrations, enrichment lookups do not consume credits. Available on the Censys " +
		"Core plan. Supports defanged IPs and requires an organization ID."
}

func (c *Command) Examples() []string {
	return []string{
		"8.8.8.8",
		"8.8.8.8,9.9.9.9",
		"--input-file ips.txt",
		"--input-file -  # read IPs from STDIN",
		"8.8.8.8 --output-format short",
	}
}

func (c *Command) Init() error {
	c.flags.inputFile = flags.NewFileFlag(c.Flags(), false, "input-file", "i", "file to read the host IPs from. Overrides the positional argument.")
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	return nil
}

func (c *Command) Args() command.PositionalArgs {
	return command.RangeArgs(0, 1)
}

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeData
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeData, command.OutputTypeShort}
}

func (c *Command) SupportsStreaming() bool {
	return true
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	if err := c.parseOrgIDFlag(); err != nil {
		return err
	}

	rawHosts, err := c.gatherRawHosts(cmd, args)
	if err != nil {
		return err
	}
	hostIDs, err := parseHostIDs(rawHosts)
	if err != nil {
		return err
	}
	c.hostIDs = hostIDs

	// Enrichment requires an organization ID. Fail early with a helpful message
	// rather than letting the API reject the request.
	if !c.orgID.IsPresent() && !c.HasOrgID() {
		return cenclierrors.NewNoOrgIDError()
	}

	return c.resolveEnrichService()
}

func (c *Command) resolveEnrichService() cenclierrors.CencliError {
	svc, err := c.EnrichService()
	if err != nil {
		return err
	}
	c.enrichSvc = svc
	return nil
}

func (c *Command) parseOrgIDFlag() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	return err
}

// gatherRawHosts returns raw host strings from file, stdin, or positional args.
func (c *Command) gatherRawHosts(cmd *cobra.Command, args []string) ([]string, cenclierrors.CencliError) {
	if c.flags.inputFile.IsSet() {
		return c.flags.inputFile.Lines(cmd)
	}
	if len(args) == 0 {
		return nil, NewNoHostsError()
	}
	return input.SplitString(args[0]), nil
}

// parseHostIDs validates each raw input as an IP, rejecting non-IPs with a clear error.
func parseHostIDs(raw []string) ([]assets.HostID, cenclierrors.CencliError) {
	hostIDs := make([]assets.HostID, 0, len(raw))
	for _, r := range raw {
		if strings.TrimSpace(r) == "" {
			continue
		}
		hostID, err := assets.NewHostID(r)
		if err != nil {
			return nil, NewInvalidHostError(r)
		}
		hostIDs = append(hostIDs, hostID)
	}
	if len(hostIDs) == 0 {
		return nil, NewNoHostsError()
	}
	return hostIDs, nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"count", len(c.hostIDs),
	)

	// Set up streaming output (no-op for non-streaming formats)
	ctx, stopStreaming := c.WithStreamingOutput(cmd.Context(), logger)
	defer stopStreaming(nil)

	err := c.WithProgress(
		ctx,
		logger,
		"Enriching hosts...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.enrichSvc.EnrichHosts(pctx, c.orgID, c.hostIDs)
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("enrichment failed", "error", err)
		return err
	}

	// Print response metadata
	c.PrintAppResponseMeta(c.result.Meta)

	// PrintData handles streaming vs buffered automatically
	if renderErr := c.PrintData(c, c.result.Hosts); renderErr != nil {
		return renderErr
	}

	// If there was a partial error, print it to stderr after rendering the data
	if c.result.PartialError != nil {
		formatter.PrintError(c.result.PartialError, cmd)
	}

	return nil
}

func (c *Command) RenderShort() cenclierrors.CencliError {
	formatter.Println(formatter.Stdout, short.EnrichedHosts(c.result.Hosts))
	return nil
}

func (*Command) Tapes(recorder *tape.Recorder) []tape.Tape {
	tallerConfig := tape.DefaultTapeConfig()
	tallerConfig.Height = 1200
	return []tape.Tape{
		tape.NewTape("enrich-short",
			tallerConfig,
			recorder.Type(
				"enrich 104.168.107.43 -O short",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
		),
	}
}
