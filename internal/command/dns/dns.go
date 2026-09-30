package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/app/dns"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
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

	// maxInputs caps the inputs of one command. The API has no bulk lookup,
	// so each input is its own lookup, run one after another.
	maxInputs = 100
)

// Command implements the `dns` CLI command.
type Command struct {
	*command.BaseCommand
	// flags
	flags dnsCommandFlags
	// state populated during PreRun
	inputs   []lookupInput
	timeline bool
	params   dns.Params
	// results stored for rendering: one per input whose lookup succeeded, in input order
	results []fetched
	// services
	dnsSvc dns.Service
}

// lookupInput is one parsed input. Exactly one of ip and name is set.
type lookupInput struct {
	value string // normalized name or IP, for titles, logs, and de-duplication
	ip    mo.Option[assets.HostID]
	name  mo.Option[assets.DomainName]
}

type dnsCommandFlags struct {
	inputFile   flags.FileFlag
	start       flags.TimestampFlag
	end         flags.TimestampFlag
	duration    flags.HumanDurationFlag
	timeline    flags.BoolFlag
	recordTypes flags.StringSliceFlag
	pageSize    flags.IntegerFlag
	maxPages    flags.IntegerFlag
	orgID       flags.OrgIDFlag
	domain      flags.StringFlag
}

var _ command.Command = (*Command)(nil)

// NewDNSCommand constructs a dns command bound to the provided context.
func NewDNSCommand(ctx *command.Context) *Command {
	return &Command{BaseCommand: command.NewBaseCommand(ctx)}
}

func (c *Command) Use() string { return fmt.Sprintf("%s <name|ip>...", cmdName) }

func (c *Command) Short() string {
	return "Look up Active DNS records for domain names and IP addresses"
}

func (c *Command) Long() string {
	return "Look up Active DNS observations for a domain name or an IP address.\n\n" +
		"For a domain name, shows the records the name resolved to (A, AAAA, MX, NS, SOA, TXT). " +
		"For an IP address, shows the domain names that resolved to it.\n\n" +
		"Several names or IPs can be given, up to 100: as positional arguments, comma-separated lists, " +
		"or read from a file (or STDIN) with --input-file. Each input is looked up independently: " +
		"if one fails, the rest still proceed.\n\n" +
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
		"141.193.213.10 --timeline --domain censys.com",
		"censys.com,104.18.10.84",
		"--input-file iocs.txt",
		"--input-file -  # read inputs from STDIN",
		"censys.com --output-format json",
	}
}

func (c *Command) Init() error {
	c.flags.inputFile = flags.NewFileFlag(c.Flags(), false, "input-file", "i", "file to read the names or IPs from (or - for STDIN). Overrides positional arguments.")
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
	c.flags.domain = flags.NewStringFlag(c.Flags(), false, "domain", "", "", "limit an IP timeline to one domain name (IP input with --timeline only)")
	return nil
}

// Args accepts any number of positional arguments: gatherInputs enforces at
// least one input (from the arguments or --input-file) and at most maxInputs.
func (c *Command) Args() command.PositionalArgs { return command.MinimumNArgs(0) }

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData, command.OutputTypeTemplate}
}

func (c *Command) SupportsStreaming() bool {
	return true
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	inputs, err := c.gatherInputs(cmd, args)
	if err != nil {
		return err
	}
	c.inputs = inputs

	recordTypes, err := c.flags.recordTypes.Value()
	if err != nil {
		return err
	}
	// Validate --record-type before any service (and so the API client) is
	// needed, so an invalid value is reported even with no client configured.
	// Each lookup direction present in the inputs must support every value.
	// The service validates again for callers that invoke it directly.
	if c.hasInput(false) {
		if err := dns.ValidateRecordTypes(recordTypes, false); err != nil {
			return err
		}
	}
	if c.hasInput(true) {
		if err := dns.ValidateRecordTypes(recordTypes, true); err != nil {
			// A name among the inputs would otherwise support this record
			// type; say so, since the generic message alone reads as if the
			// type were rejected outright.
			if c.hasInput(false) {
				return withRecordTypeReason(err, "an IP address is among the inputs")
			}
			return err
		}
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
	domain, err := c.parseDomainFlag()
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
		Domain:      domain,
	}

	// resolve required services
	c.dnsSvc, err = c.DNSService()
	if err != nil {
		return err
	}
	return nil
}

// gatherInputs reads the raw inputs, then parses each one. It fails on the
// first invalid input, before any lookup runs, and de-duplicates by the
// normalized input, keeping the first occurrence's position.
func (c *Command) gatherInputs(cmd *cobra.Command, args []string) ([]lookupInput, cenclierrors.CencliError) {
	raws, err := c.gatherRawInputs(cmd, args)
	if err != nil {
		return nil, err
	}

	var inputs []lookupInput
	seen := make(map[string]bool)
	for _, raw := range raws {
		input, err := parseInput(raw)
		if err != nil {
			return nil, err
		}
		if seen[input.value] {
			continue
		}
		seen[input.value] = true
		inputs = append(inputs, input)
	}

	switch {
	case len(inputs) == 0:
		return nil, assets.NewNoAssetsError()
	case len(inputs) > maxInputs:
		return nil, assets.NewTooManyAssetsError(len(inputs), maxInputs)
	}
	return inputs, nil
}

// gatherRawInputs returns the raw input strings: --input-file's lines
// (overriding the positional arguments, as in view), taken as-is, one input
// per line; or the positional arguments, with each argument's comma list
// split (as view splits its positional argument). A file line is never
// comma-split: unlike an argument, it cannot be quoted apart from a shell's
// own list-separating commas, so splitting it would misread a name or a URL
// that legitimately contains a comma.
func (c *Command) gatherRawInputs(cmd *cobra.Command, args []string) ([]string, cenclierrors.CencliError) {
	if c.flags.inputFile.IsSet() {
		return c.flags.inputFile.Lines(cmd)
	}
	var raws []string
	for _, arg := range args {
		raws = append(raws, splitInput(arg)...)
	}
	return raws, nil
}

// splitInput splits one argument or file line into its comma-separated inputs.
func splitInput(raw string) []string {
	// A pasted URL (defanged or not) may contain a comma in its path or query
	// (e.g. "https://censys.com/a,b"), which is not a list of assets. Detect
	// that case on the raw argument: a scheme, defanged or not, always leaves
	// a literal "//" behind ("https://", "hxxp://", "hxxps://", "https[:]//"),
	// while a comma-separated list of names or IPs never does. Checking the
	// raw argument (instead of refang.RefangURL's output) matters because
	// RefangURL prepends "http://" to some bare inputs, which would make a
	// list like "8.8.8.8,example.com" look like a URL and skip the split.
	if strings.Contains(raw, "//") {
		return []string{raw}
	}
	return cmdutil.SplitString(raw)
}

// parseInput detects the lookup direction: an IP address looks up the names
// that resolved to it; anything else must be a domain name. It does not use
// assets.AssetClassifier, which has no domain-name type and reads a bare name
// as a web property on port 443.
func parseInput(value string) (lookupInput, cenclierrors.CencliError) {
	if isCIDR(value) {
		return lookupInput{}, assets.NewInvalidAssetIDError(value, "a CIDR range is not supported; give one IP address")
	}

	if ip, err := assets.NewHostID(value); err == nil {
		return lookupInput{value: ip.String(), ip: mo.Some(ip)}, nil
	}
	name, err := assets.NewDomainName(value)
	if err != nil {
		// Keep the reason: it tells the user how to fix the input ("remove the port").
		return lookupInput{}, assets.NewInvalidAssetIDError(value, err.Error())
	}
	// A pasted URL whose host is an IP (e.g. https://8.8.8.8/) parses as a domain
	// name here, because NewDomainName strips the scheme and path and does not
	// itself reject IP-shaped input. Recover the IP direction so it is not
	// misrouted to the name lookup.
	if ip, err := assets.NewHostID(name.String()); err == nil {
		return lookupInput{value: ip.String(), ip: mo.Some(ip)}, nil
	}
	return lookupInput{value: name.String(), name: mo.Some(name)}, nil
}

// hasInput reports whether any input is an IP (isIP) or a name (!isIP).
func (c *Command) hasInput(isIP bool) bool {
	return slices.ContainsFunc(c.inputs, func(in lookupInput) bool { return in.ip.IsPresent() == isIP })
}

// isCIDR reports whether raw is a bare CIDR range ("<ip>/<prefix-length>"),
// which Active DNS lookups do not support: a range has no single IP to look
// up. A URL path that happens to start with digits (e.g. "https://8.8.8.8/32")
// is not a CIDR range, so a scheme rules it out. The IP part is refanged
// (as NewHostID does) before the check, so a defanged CIDR such as
// "8[.]8[.]8[.]8/32" is caught instead of falling through to a lookup of
// the wrong, path-truncated name. A defanged slash ("8.8.8.8[/]32") is
// refanged first, as refang.RefangURL does.
func isCIDR(raw string) bool {
	if strings.Contains(raw, "://") {
		return false
	}
	raw = strings.ReplaceAll(raw, "[/]", "/")
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
	return net.ParseIP(strings.TrimSpace(refang.RefangIP(ipPart))) != nil
}

// parseDomainFlag reads --domain. It only applies to IP lookups with
// --timeline (the API supports the filter only on IP ranges), so any other
// use, including a name among the inputs, is a usage error before a service
// (and so the API client) is needed.
// An unset --domain is read as absent, so an IP timeline stays unfiltered.
// But an explicitly empty (or all-whitespace) value is rejected instead: it
// would otherwise widen the lookup to every domain silently, for example when
// the value comes from an unset shell variable. This check runs before the
// misuse check above, so a blank --domain is reported as such even without
// --timeline or an IP input.
// The value is parsed with assets.NewDomainName, so defanged input normalizes
// and an invalid name is rejected with the usual Invalid Asset ID error.
func (c *Command) parseDomainFlag() (mo.Option[assets.DomainName], cenclierrors.CencliError) {
	raw, err := c.flags.domain.Value()
	if err != nil {
		return mo.None[assets.DomainName](), err
	}
	if c.Flags().Changed("domain") && strings.TrimSpace(raw) == "" {
		return mo.None[assets.DomainName](), NewDomainFlagEmptyError()
	}
	if raw == "" {
		return mo.None[assets.DomainName](), nil
	}
	if c.hasInput(false) || !c.timeline {
		return mo.None[assets.DomainName](), NewDomainFlagMisuseError()
	}
	domain, derr := assets.NewDomainName(raw)
	if derr != nil {
		return mo.None[assets.DomainName](), assets.NewInvalidAssetIDError(raw, derr.Error())
	}
	return mo.Some(domain), nil
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

// fetched is what Run and RenderShort need from one input's lookup, for any
// of the four lookups.
type fetched struct {
	input lookupInput
	meta  *responsemeta.ResponseMeta
	// records holds one of the four wrapped record slices (see RenderShort).
	records      any
	total        int64
	partialError cenclierrors.CencliError
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"inputs", len(c.inputs),
		"timeline", c.timeline,
		"start", c.params.FromTime.Format(time.RFC3339),
		"end", c.params.ToTime.Format(time.RFC3339),
	)

	// Warn before fetching all pages; copied from search, which owns the same warning.
	if !c.Config().Quiet && !c.params.MaxPages.IsPresent() {
		scope := ""
		if len(c.inputs) > 1 {
			scope = fmt.Sprintf(" for each of %d inputs", len(c.inputs))
		}
		msg := styles.GlobalStyles.Warning.Render(fmt.Sprintf(
			"Warning: fetching all pages (--max-pages=-1)%s. This may take a while and increase API usage.", scope))
		formatter.Println(formatter.Stderr, msg)
		logger.Debug("fetching all pages", "message", msg)
	}

	// Set up streaming output (no-op for non-streaming formats)
	ctx, stopStreaming := c.WithStreamingOutput(cmd.Context(), logger)
	defer stopStreaming(nil)

	// Look up each input in turn. A failed input does not stop the others; its
	// error is printed after the output.
	var failures []cenclierrors.CencliError
	var accessDeniedErr cenclierrors.CencliError
	for i, input := range c.inputs {
		// A prior input can succeed with partial data (a later page failed on
		// cancellation) without returning a fetch error itself, so the context
		// must be checked here too: otherwise this input's lookup would still
		// start, only to fail before any request goes out.
		if ctx.Err() != nil {
			c.reportInterrupted(len(c.inputs) - i)
			break
		}
		result, err := c.fetchWithProgress(ctx, logger, input, i)
		if err != nil {
			logger.Debug("dns fetch failed", "input", input.value, "error", err)
			if dns.IsAccessDeniedError(err) {
				// A plan restriction fails every lookup the same way, so stop
				// here. With nothing collected yet, there is no output to
				// keep: print the earlier failures in input order, then fail
				// exactly as an unrecoverable error would.
				if len(c.results) == 0 {
					for _, failure := range failures {
						formatter.PrintError(failure, cmd)
					}
					return err
				}
				// Otherwise keep what was already collected: print it (below,
				// the same way a normal run would), then report the error.
				accessDeniedErr = err
				break
			}
			failures = append(failures, c.withInput(input, err))
			// After an interrupt, every remaining lookup would fail the same way.
			if ctx.Err() != nil {
				c.reportInterrupted(len(c.inputs) - i - 1)
				break
			}
			continue
		}
		if len(c.inputs) == 1 {
			c.PrintAppResponseMeta(result.meta)
		}
		c.results = append(c.results, result)
	}

	if len(c.results) == 0 {
		// Print every failure but the last in input order, then return the
		// last: the caller prints a returned error right after Run's own
		// stderr output, so this is the only ordering that puts every
		// failure on stderr in input order. With one failure, nothing is
		// printed here and that failure is returned, unchanged from before.
		for _, failure := range failures[:len(failures)-1] {
			formatter.PrintError(failure, cmd)
		}
		return failures[len(failures)-1]
	}

	// With several inputs, one metadata block per input would be noise (100
	// inputs, 100 blocks); print one combined block instead.
	if len(c.inputs) > 1 {
		c.PrintAppResponseMeta(combinedResponseMeta(c.results))
	}

	// PrintData handles streaming vs buffered automatically
	if printErr := c.PrintData(c, c.data()); printErr != nil {
		return printErr
	}
	// Flush any streamed records, so the errors below follow all of the output.
	stopStreaming(nil)

	// Print partial errors (a later page failed) and failed inputs to stderr after the data
	for _, result := range c.results {
		if result.partialError != nil {
			formatter.PrintError(c.withInput(result.input, result.partialError), cmd)
		}
	}
	for _, failure := range failures {
		formatter.PrintError(failure, cmd)
	}

	if accessDeniedErr != nil {
		return accessDeniedErr
	}
	return nil
}

// reportInterrupted tells the user how many inputs an interrupt (context
// cancellation) left unlooked-up. remaining is everything after the input
// whose lookup was cancelled; the cancelled input itself was attempted (and
// so already counted as a failure), so it is not included. This is an
// outcome, not an advisory note, so it prints regardless of --quiet.
func (c *Command) reportInterrupted(remaining int) {
	if remaining <= 0 || len(c.inputs) <= 1 {
		return
	}
	noun := "inputs"
	if remaining == 1 {
		noun = "input"
	}
	formatter.Println(formatter.Stderr, fmt.Sprintf("interrupted; %d %s not looked up", remaining, noun))
}

// combinedResponseMeta reports one metadata block for several inputs: the
// last successful lookup's method, URL, and status, plus the latency and
// page count summed across every lookup that returned one.
func combinedResponseMeta(results []fetched) *responsemeta.ResponseMeta {
	var last *responsemeta.ResponseMeta
	var latency time.Duration
	var pages uint64
	for _, result := range results {
		if result.meta == nil {
			continue
		}
		last = result.meta
		latency += result.meta.Latency
		pages += result.meta.PageCount
	}
	if last == nil {
		return nil
	}
	combined := *last
	combined.Latency = latency
	combined.PageCount = pages
	return &combined
}

// fetchWithProgress looks up one input, showing its progress; i is the
// input's index in c.inputs.
func (c *Command) fetchWithProgress(ctx context.Context, logger *slog.Logger, input lookupInput, i int) (fetched, cenclierrors.CencliError) {
	msg := fmt.Sprintf("Fetching DNS records for %s...", input.value)
	if len(c.inputs) > 1 {
		msg = fmt.Sprintf("Fetching DNS records for %s (%d/%d)...", input.value, i+1, len(c.inputs))
	}
	var result fetched
	err := c.WithProgress(ctx, logger, msg, func(pctx context.Context) cenclierrors.CencliError {
		var fetchErr cenclierrors.CencliError
		result, fetchErr = c.fetch(pctx, input)
		return fetchErr
	})
	return result, err
}

// withInput names the input in err when the command has several inputs, so
// the user can tell which one failed. With one input, err is unchanged.
func (c *Command) withInput(input lookupInput, err cenclierrors.CencliError) cenclierrors.CencliError {
	if len(c.inputs) == 1 {
		return err
	}
	return newInputError(input.value, err)
}

// data returns every input's records as one list, in input order. It is never
// nil, so JSON output prints [] (not null) when there are no records.
func (c *Command) data() []any {
	data := []any{}
	for _, result := range c.results {
		switch records := result.records.(type) {
		case []*dns.NameRecord:
			data = appendAll(data, records)
		case []*dns.NameRangeRecord:
			data = appendAll(data, records)
		case []*dns.IPRecord:
			data = appendAll(data, records)
		case []*dns.IPRangeRecord:
			data = appendAll(data, records)
		}
	}
	return data
}

// RenderTemplate renders every input's records using a handlebars template.
// See templateData for what each record holds.
func (c *Command) RenderTemplate() cenclierrors.CencliError {
	data, err := c.templateData()
	if err != nil {
		return cenclierrors.NewCencliError(err)
	}
	return c.PrintDataWithTemplate(config.TemplateEntityDNS, data)
}

// templateKeys is every key a template may read on a record: the JSON field
// names of the four SDK record types, plus "input".
var templateKeys = append(jsonFieldNames(
	components.DNSResolutionRecord{},
	components.DNSResolutionRangeRecord{},
	components.DNSIPResolutionRecord{},
	components.DNSIPResolutionRangeRecord{},
), "input")

// jsonFieldNames returns the JSON names of the fields of each struct value.
func jsonFieldNames(values ...any) []string {
	var names []string
	for _, v := range values {
		t := reflect.TypeOf(v)
		for i := range t.NumField() {
			name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
			if name != "" && name != "-" && !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	return names
}

// templateData returns the records Data output prints, as maps keyed by the
// JSON field names, for the template engine. Handlebars looks up a key that a
// record lacks in the parent context (the list of records), which would print
// that field collected from every other record; so each map holds every key in
// templateKeys, nil when the record lacks it. Every string is stripped of
// control characters, so a template can print it unescaped. "timeline" tells
// a template whether the records are observed time ranges (--timeline).
func (c *Command) templateData() ([]map[string]any, error) {
	encoded, err := json.Marshal(c.data())
	if err != nil {
		return nil, fmt.Errorf("failed to encode records for the template: %w", err)
	}
	var records []map[string]any
	if err := json.Unmarshal(encoded, &records); err != nil {
		return nil, fmt.Errorf("failed to decode records for the template: %w", err)
	}
	for _, record := range records {
		for key, value := range record {
			if s, ok := value.(string); ok {
				record[key] = sanitizeCell(s)
			}
		}
		for _, key := range templateKeys {
			if _, ok := record[key]; !ok {
				record[key] = nil
			}
		}
		record["timeline"] = c.timeline
	}
	return records, nil
}

func appendAll[T any](dst []any, items []T) []any {
	for _, item := range items {
		dst = append(dst, item)
	}
	return dst
}

// fetch calls the lookup that matches the input's direction and --timeline.
func (c *Command) fetch(ctx context.Context, input lookupInput) (fetched, cenclierrors.CencliError) {
	switch {
	case input.ip.IsPresent() && c.timeline:
		r, err := c.dnsSvc.IPResolutionRanges(ctx, input.ip.MustGet(), c.params)
		if err != nil {
			return fetched{}, err
		}
		return fetched{input: input, meta: r.Meta, records: r.Records, total: r.TotalRecords, partialError: r.PartialError}, nil
	case input.ip.IsPresent():
		r, err := c.dnsSvc.IPResolutions(ctx, input.ip.MustGet(), c.params)
		if err != nil {
			return fetched{}, err
		}
		return fetched{input: input, meta: r.Meta, records: r.Records, total: r.TotalRecords, partialError: r.PartialError}, nil
	case c.timeline:
		r, err := c.dnsSvc.NameResolutionRanges(ctx, input.name.MustGet(), c.params)
		if err != nil {
			return fetched{}, err
		}
		return fetched{input: input, meta: r.Meta, records: r.Records, total: r.TotalRecords, partialError: r.PartialError}, nil
	default:
		r, err := c.dnsSvc.NameResolutions(ctx, input.name.MustGet(), c.params)
		if err != nil {
			return fetched{}, err
		}
		return fetched{input: input, meta: r.Meta, records: r.Records, total: r.TotalRecords, partialError: r.PartialError}, nil
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
