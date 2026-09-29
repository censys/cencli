package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/dns"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	cmdutil "github.com/censys/cencli/internal/pkg/input"
	"github.com/censys/cencli/internal/pkg/refang"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/tape"
)

const (
	cmdName = "dns"

	defaultPageSize = 100
	minPageSize     = 1
	maxPageSize     = 100
	defaultMaxPages = 10
)

// Command implements the `dns` CLI command.
type Command struct {
	*command.BaseCommand
	// flags
	flags dnsCommandFlags
	// state populated during PreRun
	input    string // normalized name or IP, for titles and logs
	ip       mo.Option[assets.HostID]
	name     mo.Option[assets.DomainName]
	timeline bool
	params   dns.Params
	// result stored for rendering
	result fetched
	// services
	dnsSvc dns.Service
}

type dnsCommandFlags struct {
	start       flags.TimestampFlag
	end         flags.TimestampFlag
	duration    flags.HumanDurationFlag
	timeline    flags.BoolFlag
	recordTypes flags.StringSliceFlag
	pageSize    flags.IntegerFlag
	maxPages    flags.IntegerFlag
	orgID       flags.OrgIDFlag
}

var _ command.Command = (*Command)(nil)

// NewDNSCommand constructs a dns command bound to the provided context.
func NewDNSCommand(ctx *command.Context) *Command {
	return &Command{BaseCommand: command.NewBaseCommand(ctx)}
}

func (c *Command) Use() string { return fmt.Sprintf("%s <name|ip>", cmdName) }

func (c *Command) Short() string {
	return "Look up Active DNS records for domain names and IP addresses"
}

func (c *Command) Long() string {
	return "Look up Active DNS observations for a domain name or an IP address.\n\n" +
		"For a domain name, shows the records the name resolved to (A, AAAA, MX, NS, SOA, TXT). " +
		"For an IP address, shows the domain names that resolved to it.\n\n" +
		"Results cover a time window (default: the last 7 days). Use --timeline to show each observed " +
		"time range instead of one row per record.\n\n" +
		"For the DNS names in a host's current record, use \"censys view <ip>\".\n\n" +
		"This command is only available to organizations on the Censys Search and Censys Core plans."
}

func (c *Command) Examples() []string {
	return []string{
		"censys.com",
		"censys.com --record-type A,MX",
		"censys.com --timeline --duration 90d",
		"104.18.10.84",
		"104.18.10.84 --start 2026-06-01T00:00:00Z --duration 30d",
		"censys.com --output-format json",
	}
}

func (c *Command) Init() error {
	c.flags.start = flags.NewTimestampFlag(c.Flags(), false, "start", "s", mo.None[time.Time](), "start time")
	c.flags.end = flags.NewTimestampFlag(c.Flags(), false, "end", "e", mo.None[time.Time](), "end time")
	c.flags.duration = flags.NewHumanDurationFlag(c.Flags(), false, "duration", "d", mo.Some(7*24*time.Hour), "time window (e.g., 1d, 1w, 1y, 2h). Defaults to 7d")
	c.flags.timeline = flags.NewBoolFlag(c.Flags(), "timeline", "t", false, "show one row per observed time range instead of one row per record")
	c.flags.recordTypes = flags.NewStringSliceFlag(c.Flags(), false, "record-type", "r", nil, "record types to include (A, AAAA, MX, NS, SOA, TXT; IP addresses support A and AAAA)")
	c.flags.pageSize = flags.NewIntegerFlag(
		c.Flags(),
		false,
		"page-size",
		"n",
		mo.Some[int64](defaultPageSize),
		"number of records to return per page",
		mo.Some[int64](minPageSize),
		mo.Some[int64](maxPageSize),
	)
	c.flags.maxPages = flags.NewIntegerFlag(
		c.Flags(),
		false,
		"max-pages",
		"p",
		mo.Some[int64](defaultMaxPages),
		"maximum number of pages to fetch (-1 for all pages)",
		mo.None[int64](), // allow custom validation in PreRun (to support -1)
		mo.None[int64](), // no maximum
	)
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	return nil
}

func (c *Command) Args() command.PositionalArgs { return command.ExactArgs(1) }

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *Command) SupportsStreaming() bool {
	return true
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	if err := c.parseInput(args[0]); err != nil {
		return err
	}

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
	fromTime, toTime, err := flags.ResolveTimeWindow(startOpt, endOpt, durationOpt)
	if err != nil {
		return err
	}
	logger := c.Logger(cmdName)
	logger.Debug("Time window", "start", fromTime.Format(time.RFC3339), "end", toTime.Format(time.RFC3339))

	c.timeline, err = c.flags.timeline.Value()
	if err != nil {
		return err
	}
	recordTypes, err := c.flags.recordTypes.Value()
	if err != nil {
		return err
	}
	pageSize, maxPages, err := c.parsePaginationFlags()
	if err != nil {
		return err
	}

	// parse org id
	flagOrgID, err := c.flags.orgID.Value()
	if err != nil {
		return err
	}
	// Route through the credential-aware resolver: --org-id applies only to
	// personal access tokens, so this rejects it when the credential defines the
	// organization itself, and otherwise supplies the credential's organization.
	orgID, err := c.ResolveOrgID(cmd.Context(), flagOrgID)
	if err != nil {
		return err
	}

	c.params = dns.Params{
		OrgID:       orgID,
		FromTime:    fromTime,
		ToTime:      toTime,
		RecordTypes: recordTypes,
		PageSize:    pageSize,
		MaxPages:    maxPages,
	}

	// resolve required services
	c.dnsSvc, err = c.DNSService()
	if err != nil {
		return err
	}
	return nil
}

// parseInput detects the lookup direction: an IP address looks up the names
// that resolved to it; anything else must be a domain name. It does not use
// assets.AssetClassifier, which has no domain-name type and reads a bare name
// as a web property on port 443.
func (c *Command) parseInput(raw string) cenclierrors.CencliError {
	value := raw
	// A pasted URL (defanged or not) may contain a comma in its path or query
	// (e.g. "https://censys.com/a,b"), which is not a list of assets. Detect
	// that case by refanging for a scheme before splitting on commas, so a URL
	// is treated as one input instead of being split apart.
	if !strings.Contains(refang.RefangURL(raw), "://") {
		values := cmdutil.SplitString(raw)
		switch len(values) {
		case 0:
			return assets.NewNoAssetsError()
		case 1:
		default:
			return assets.NewTooManyAssetsError(len(values), 1)
		}
		value = values[0]
	}

	if isCIDR(value) {
		return assets.NewInvalidAssetIDError(value, "a CIDR range is not supported; give one IP address")
	}

	if ip, err := assets.NewHostID(value); err == nil {
		c.ip = mo.Some(ip)
		c.input = ip.String()
		return nil
	}
	name, err := assets.NewDomainName(value)
	if err != nil {
		// Keep the reason: it tells the user how to fix the input ("remove the port").
		return assets.NewInvalidAssetIDError(value, err.Error())
	}
	// A pasted URL whose host is an IP (e.g. https://8.8.8.8/) parses as a domain
	// name here, because NewDomainName strips the scheme and path and does not
	// itself reject IP-shaped input. Recover the IP direction so it is not
	// misrouted to the name lookup.
	if ip, err := assets.NewHostID(name.String()); err == nil {
		c.ip = mo.Some(ip)
		c.input = ip.String()
		return nil
	}
	c.name = mo.Some(name)
	c.input = name.String()
	return nil
}

// isCIDR reports whether raw is a bare CIDR range ("<ip>/<prefix-length>"),
// which Active DNS lookups do not support: a range has no single IP to look
// up. A URL path that happens to start with digits (e.g. "https://8.8.8.8/32")
// is not a CIDR range, so a scheme rules it out.
func isCIDR(raw string) bool {
	if strings.Contains(raw, "://") {
		return false
	}
	i := strings.LastIndex(raw, "/")
	if i < 0 {
		return false
	}
	ipPart, prefix := raw[:i], raw[i+1:]
	if prefix == "" {
		return false
	}
	for _, r := range prefix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return net.ParseIP(strings.TrimSpace(ipPart)) != nil
}

// parsePaginationFlags reads --page-size and --max-pages. A --max-pages of -1
// means all pages, which is returned as an absent value.
func (c *Command) parsePaginationFlags() (mo.Option[uint64], mo.Option[uint64], cenclierrors.CencliError) {
	pageSize := mo.None[uint64]()
	maxPages := mo.None[uint64]()

	rawPageSize, err := c.flags.pageSize.Value()
	if err != nil {
		return pageSize, maxPages, err
	}
	if rawPageSize.IsPresent() {
		pageSize = mo.Some(uint64(rawPageSize.MustGet()))
	}

	rawMaxPages, err := c.flags.maxPages.Value()
	if err != nil {
		return pageSize, maxPages, err
	}
	if rawMaxPages.IsPresent() {
		switch v := rawMaxPages.MustGet(); {
		case v == -1:
			maxPages = mo.None[uint64]()
		case v <= 0:
			return pageSize, maxPages, flags.NewIntegerFlagInvalidValueError("max-pages", v, "must be -1 or >= 1")
		default:
			maxPages = mo.Some(uint64(v))
		}
	}
	return pageSize, maxPages, nil
}

// fetched is what Run and RenderShort need from any of the four lookups.
type fetched struct {
	meta *responsemeta.ResponseMeta
	// records holds one of the four SDK record slices (see RenderShort).
	records      any
	total        int64
	partialError cenclierrors.CencliError
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"input", c.input,
		"timeline", c.timeline,
		"start", c.params.FromTime.Format(time.RFC3339),
		"end", c.params.ToTime.Format(time.RFC3339),
	)

	// Warn before fetching all pages; copied from search, which owns the same warning.
	if !c.Config().Quiet && !c.params.MaxPages.IsPresent() {
		msg := styles.GlobalStyles.Warning.Render(
			"Warning: fetching all pages (--max-pages=-1). This may take a while and increase API usage.")
		formatter.Println(formatter.Stderr, msg)
		logger.Debug("fetching all pages", "message", msg)
	}

	// Set up streaming output (no-op for non-streaming formats)
	ctx, stopStreaming := c.WithStreamingOutput(cmd.Context(), logger)
	defer stopStreaming(nil)

	err := c.WithProgress(
		ctx,
		logger,
		fmt.Sprintf("Fetching DNS records for %s...", c.input),
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.fetch(pctx)
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("dns fetch failed", "error", err)
		return err
	}

	// Print response metadata and output (PrintData handles streaming vs buffered automatically)
	c.PrintAppResponseMeta(c.result.meta)
	if printErr := c.PrintData(c, c.result.records); printErr != nil {
		return printErr
	}

	// If there was a partial error, print it to stderr after rendering the data
	if c.result.partialError != nil {
		formatter.PrintError(c.result.partialError, cmd)
	}
	return nil
}

// fetch calls the lookup that matches the input direction and --timeline.
func (c *Command) fetch(ctx context.Context) (fetched, cenclierrors.CencliError) {
	switch {
	case c.ip.IsPresent() && c.timeline:
		r, err := c.dnsSvc.IPResolutionRanges(ctx, c.ip.MustGet(), c.params)
		if err != nil {
			return fetched{}, err
		}
		return fetched{meta: r.Meta, records: r.Records, total: r.TotalRecords, partialError: r.PartialError}, nil
	case c.ip.IsPresent():
		r, err := c.dnsSvc.IPResolutions(ctx, c.ip.MustGet(), c.params)
		if err != nil {
			return fetched{}, err
		}
		return fetched{meta: r.Meta, records: r.Records, total: r.TotalRecords, partialError: r.PartialError}, nil
	case c.timeline:
		r, err := c.dnsSvc.NameResolutionRanges(ctx, c.name.MustGet(), c.params)
		if err != nil {
			return fetched{}, err
		}
		return fetched{meta: r.Meta, records: r.Records, total: r.TotalRecords, partialError: r.PartialError}, nil
	default:
		r, err := c.dnsSvc.NameResolutions(ctx, c.name.MustGet(), c.params)
		if err != nil {
			return fetched{}, err
		}
		return fetched{meta: r.Meta, records: r.Records, total: r.TotalRecords, partialError: r.PartialError}, nil
	}
}

func (*Command) Tapes(recorder *tape.Recorder) []tape.Tape {
	return []tape.Tape{
		tape.NewTape("dns",
			tape.DefaultTapeConfig(),
			recorder.Type(
				"dns censys.com",
				tape.WithSleepAfter(8),
			),
		),
		tape.NewTape("dns-timeline",
			tape.DefaultTapeConfig(),
			recorder.Type(
				"dns censys.com --timeline --record-type A --duration 90d",
				tape.WithSleepAfter(8),
			),
		),
	}
}
