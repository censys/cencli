package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/search"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/formatter/short"
	"github.com/censys/cencli/internal/pkg/jq"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/tape"
)

const (
	cmdName = "search"

	defaultPageSize = 100
	minPageSize     = 1

	defaultMaxPages = 1
)

// Command implements the `search` subcommand, providing asset search capabilities.
// It parses flags and delegates to the `search.Service` to perform queries.
type Command struct {
	*command.BaseCommand
	// services the command uses
	searchSvc search.Service
	// flags the command uses
	flags searchCommandFlags
	// state - populated by PreRun (through flags, args, etc.)
	query        string
	fields       []string
	collectionID mo.Option[identifiers.CollectionID]
	orgID        mo.Option[identifiers.OrganizationID]
	pageSize     mo.Option[uint64]
	maxPages     mo.Option[uint64]
	count        bool
	allPages     bool   // --all: suppresses the "fetching all pages" warning
	jqExpr       string // --jq: path expression applied to each result
	// result stores the search result for rendering
	result search.Result
}

// searchCommandFlags contains all flag handles used by the search command.
type searchCommandFlags struct {
	orgID        flags.OrgIDFlag
	collectionID flags.UUIDFlag
	fields       flags.StringSliceFlag
	pageSize     flags.IntegerFlag
	maxPages     flags.IntegerFlag
	count        flags.BoolFlag
	all          flags.BoolFlag
	jq           flags.StringFlag
}

var _ command.Command = (*Command)(nil)

func NewSearchCommand(cmdContext *command.Context) *Command {
	return &Command{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

// Long returns a detailed description of the command.
func (c *Command) Long() string {
	return "Run a search query across Censys data. Queries must be written in the Censys Query Language."
}

func (c *Command) Use() string {
	return fmt.Sprintf("%s <query>", cmdName)
}

func (c *Command) Short() string {
	return "Execute a search query across Censys data"
}

func (c *Command) Args() command.PositionalArgs {
	return command.ExactArgs(1)
}

func (c *Command) DefaultOutputType() command.OutputType {
	return command.OutputTypeData
}

func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeData, command.OutputTypeTemplate, command.OutputTypeShort}
}

func (c *Command) SupportsStreaming() bool {
	return true
}

func (c *Command) Examples() []string {
	return []string{
		`"host.ip: '1.1.1.1/16'"`,
		`--fields host.ip,host.location.country "host.services: (protocol=SSH and not port: 22)"`,
		`--collection-id <your-collection-id> "host.services.protocol=SSH"`,
		`--page-size 50 --max-pages 5 "cert.names=censys.com"`,
		`--all "host.services.port: 443 and host.location.country: Germany"`,
		`--count "host.services.protocol=SSH"`,
		`--jq .host.ip "host.services.protocol=SSH"`,
		`--all --jq ".host.services[].port" "host.location.country: Germany"`,
	}
}

// Init sets up command flags and config-backed defaults.
func (c *Command) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.collectionID = flags.NewUUIDFlag(
		c.Flags(),
		false,
		"collection-id",
		"c",
		mo.None[uuid.UUID](),
		"collection to search within (optional)",
	)
	c.flags.fields = flags.NewStringSliceFlag(
		c.Flags(),
		false,
		"fields",
		"f",
		[]string{},
		"fields to return in response (optional)",
	)
	// Use config-backed defaults for pagination
	defaultPS := int64(defaultPageSize)
	if v := c.Config().Search.PageSize; v > 0 {
		defaultPS = v
	}
	defaultMP := int64(defaultMaxPages)
	if v := c.Config().Search.MaxPages; v != 0 { // 0 is invalid; keep 1 if 0
		defaultMP = v
	}
	c.flags.pageSize = flags.NewIntegerFlag(
		c.Flags(),
		false,
		"page-size",
		"n",
		mo.Some[int64](defaultPS),
		"number of results to return per page",
		mo.Some[int64](minPageSize),
		mo.None[int64](), // no maximum
	)
	c.flags.maxPages = flags.NewIntegerFlag(
		c.Flags(),
		false,
		"max-pages",
		"p",
		mo.Some[int64](defaultMP),
		"maximum number of pages to fetch (-1 for all pages)",
		mo.None[int64](), // allow custom validation in PreRun (to support -1)
		mo.None[int64](), // no maximum
	)
	c.flags.count = flags.NewBoolFlag(
		c.Flags(),
		"count",
		"",
		false,
		"print only the total number of matching results (from the first page) instead of the results themselves",
	)
	c.flags.all = flags.NewBoolFlag(
		c.Flags(),
		"all",
		"A",
		false,
		"fetch all pages of results (shorthand for --max-pages=-1)",
	)
	c.flags.jq = flags.NewStringFlag(
		c.Flags(),
		false,
		"jq",
		"",
		"",
		`filter output using a jq-style path (e.g. .host.ip, .host.services[].port)`,
	)
	return nil
}

// PreRun validates flags and prepares the command for execution.
func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// args have already been validated
	c.query = args[0]

	if err := c.parseOrgIDFlag(cmd.Context()); err != nil {
		return err
	}
	if err := c.parseCollectionIDFlag(); err != nil {
		return err
	}
	if err := c.parsePaginationFlags(); err != nil {
		return err
	}
	if err := c.parseFieldsFlag(); err != nil {
		return err
	}
	if err := c.parseCountFlag(); err != nil {
		return err
	}
	if err := c.parseAllFlag(); err != nil {
		return err
	}
	if err := c.parseJQFlag(); err != nil {
		return err
	}
	return c.resolveSearchService()
}

// Run executes the command by calling the search service and rendering results.
func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"collectionID_set", c.collectionID.IsPresent(),
		"fields_set", len(c.fields) > 0,
		"pageSize_set", c.pageSize.IsPresent(),
		"maxPages_set", c.maxPages.IsPresent(),
		"count", c.count,
		"query", c.query,
	)

	if c.count {
		return c.runCount(cmd, logger)
	}

	if !c.Config().Quiet && !c.maxPages.IsPresent() && !c.allPages {
		msg := styles.GlobalStyles.Warning.Render("Warning: fetching all pages (--max-pages=-1). This may take a while and increase API usage.")
		formatter.Println(formatter.Stderr, msg)
		logger.Debug("fetching all pages", "message", msg)
	}

	// Set up streaming output (no-op for non-streaming formats or when --jq is set)
	ctx, stopStreaming := c.WithStreamingOutput(cmd.Context(), logger)
	defer stopStreaming(nil)

	err := c.WithProgress(
		ctx,
		logger,
		"Fetching search results...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.fetchSearchResult(pctx)
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("fetch failed", "error", err)
		return err
	}

	// Print response metadata
	c.PrintAppResponseMeta(c.result.Meta)

	if c.jqExpr != "" {
		return c.runJQ()
	}

	// PrintData handles streaming vs buffered automatically
	data := c.prepareSearchData()
	if renderErr := c.PrintData(c, data); renderErr != nil {
		return renderErr
	}

	// If there was a partial error, print it to stderr after rendering the data
	if c.result.PartialError != nil {
		formatter.PrintError(c.result.PartialError, cmd)
	}

	return nil
}

// runCount fetches a single minimal page and prints only the total number of
// matching results. It intentionally skips streaming and per-asset rendering.
func (c *Command) runCount(cmd *cobra.Command, logger *slog.Logger) cenclierrors.CencliError {
	c.warnIgnoredCountFlags(cmd, logger)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Counting search results...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.fetchSearchResult(pctx)
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("count fetch failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	// The field name mirrors "total_hits" in the search API response so that
	// structured output is consistent with the full result payload.
	data := map[string]any{"total_hits": c.result.TotalHits}
	plain := fmt.Sprintf("%d", c.result.TotalHits)
	return c.PrintValueByFormat(data, plain)
}

// warnIgnoredCountFlags emits a warning when flags that have no effect in
// --count mode are explicitly set. Pagination is forced to a single minimal
// page and no per-asset data is rendered, so these flags are silently ignored.
func (c *Command) warnIgnoredCountFlags(cmd *cobra.Command, logger *slog.Logger) {
	if c.Config().Quiet {
		return
	}

	var ignored []string
	for _, name := range []string{"page-size", "max-pages", "all", "fields", "jq", config.StreamingFlagName} {
		if f := cmd.Flag(name); f != nil && f.Changed {
			ignored = append(ignored, "--"+name)
		}
	}
	if len(ignored) == 0 {
		return
	}

	msg := styles.GlobalStyles.Warning.Render(
		fmt.Sprintf("Warning: --count ignores %s.", strings.Join(ignored, ", ")),
	)
	formatter.Println(formatter.Stderr, msg)
	logger.Debug("count mode ignoring flags", "flags", ignored)
}

func (c *Command) fetchSearchResult(ctx context.Context) (search.Result, cenclierrors.CencliError) {
	params := search.Params{
		OrgID:        c.orgID,
		CollectionID: c.collectionID,
		Query:        c.query,
		Fields:       c.fields,
		PageSize:     c.pageSize,
		MaxPages:     c.maxPages,
	}

	return c.searchSvc.Search(ctx, params)
}

// prepareSearchData wraps each hit with its type to help differentiate in the output.
func (c *Command) prepareSearchData() []any {
	data := make([]any, len(c.result.Hits))
	for i, hit := range c.result.Hits {
		data[i] = map[string]any{
			hit.AssetType().String(): hit,
		}
	}
	return data
}

// RenderTemplate renders search results using a handlebars template.
func (c *Command) RenderTemplate() cenclierrors.CencliError {
	data := c.prepareSearchData()
	return c.PrintDataWithTemplate(config.TemplateEntitySearchResult, data)
}

// RenderShort renders search results in short format.
func (c *Command) RenderShort() cenclierrors.CencliError {
	output := short.SearchHits(c.result.Hits)
	formatter.Println(formatter.Stdout, output)
	return nil
}

// parseAllFlag parses --all and, when set, overrides max-pages to unlimited.
func (c *Command) parseAllFlag() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.allPages, err = c.flags.all.Value()
	if err != nil {
		return err
	}
	if c.allPages {
		c.maxPages = mo.None[uint64]()
	}
	return nil
}

// parseJQFlag parses --jq and validates the expression at PreRun time.
func (c *Command) parseJQFlag() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.jqExpr, err = c.flags.jq.Value()
	if err != nil {
		return err
	}
	if c.jqExpr != "" {
		if _, parseErr := jq.Parse(c.jqExpr); parseErr != nil {
			return cenclierrors.NewCencliError(parseErr)
		}
	}
	return nil
}

// runJQ applies the --jq expression to every result hit and prints one value per line.
func (c *Command) runJQ() cenclierrors.CencliError {
	for _, item := range c.prepareSearchData() {
		raw, err := json.Marshal(item)
		if err != nil {
			continue
		}
		values, err := jq.EvalJSON(c.jqExpr, raw)
		if err != nil {
			return cenclierrors.NewCencliError(err)
		}
		for _, v := range values {
			formatter.Println(formatter.Stdout, jq.FormatValue(v))
		}
	}
	if c.result.PartialError != nil {
		formatter.Println(formatter.Stderr, c.result.PartialError.Error())
	}
	return nil
}

// resolveSearchService initializes the search service from the command context.
func (c *Command) resolveSearchService() cenclierrors.CencliError {
	svc, err := c.SearchService()
	if err != nil {
		return err
	}
	c.searchSvc = svc
	return nil
}

// parseOrgIDFlag resolves the organization for the request into c.orgID.
func (c *Command) parseOrgIDFlag(ctx context.Context) cenclierrors.CencliError {
	flagOrgID, err := c.flags.orgID.Value()
	if err != nil {
		return err
	}
	// Route through the credential-aware resolver: --org-id applies only to
	// personal access tokens, so this rejects it when the credential defines the
	// organization itself, and otherwise supplies the credential's organization.
	c.orgID, err = c.ResolveOrgID(ctx, flagOrgID)
	if err != nil {
		return err
	}
	return nil
}

// parseCollectionIDFlag parses the optional collection-id flag into c.collectionID.
func (c *Command) parseCollectionIDFlag() cenclierrors.CencliError {
	collectionID, err := c.flags.collectionID.Value()
	if err != nil {
		return err
	}
	if collectionID.IsPresent() {
		c.collectionID = mo.Some(identifiers.NewCollectionID(collectionID.MustGet()))
	}
	return nil
}

// parsePaginationFlags validates and parses page-size and max-pages flags.
func (c *Command) parsePaginationFlags() cenclierrors.CencliError {
	pageSize, err := c.flags.pageSize.Value()
	if err != nil {
		return err
	}
	if pageSize.IsPresent() {
		// this wont wrap around since the flag enforces this is non-negative
		c.pageSize = mo.Some(uint64(pageSize.MustGet()))
	}

	maxPages, err := c.flags.maxPages.Value()
	if err != nil {
		return err
	}
	if maxPages.IsPresent() {
		// Support -1 for unlimited pages; 0 and negatives (except -1) invalid
		switch v := maxPages.MustGet(); {
		case v == -1:
			c.maxPages = mo.None[uint64]()
		case v <= 0:
			return flags.NewIntegerFlagInvalidValueError("max-pages", v, "must be -1 or >= 1")
		default:
			// this wont wrap around since we guard negatives and zero
			c.maxPages = mo.Some(uint64(v))
		}
	}
	return nil
}

// parseCountFlag parses the --count flag. When set, only the total number of
// matching results is needed, so pagination is forced to a single, minimal page.
// The API returns the total count on every page regardless of page size, so a
// page size of 1 minimizes the payload. Explicit --page-size/--max-pages values
// are overridden because they are meaningless in count mode.
func (c *Command) parseCountFlag() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.count, err = c.flags.count.Value()
	if err != nil {
		return err
	}
	if c.count {
		c.pageSize = mo.Some[uint64](1)
		c.maxPages = mo.Some[uint64](1)
	}
	return nil
}

// parseFieldsFlag parses the optional fields flag into c.fields.
func (c *Command) parseFieldsFlag() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.fields, err = c.flags.fields.Value()
	if err != nil {
		return err
	}
	return nil
}

func (*Command) Tapes(recorder *tape.Recorder) []tape.Tape {
	return []tape.Tape{
		tape.NewTape("search",
			tape.DefaultTapeConfig(),
			recorder.Type(
				"search censys.com --page-size 1",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
			recorder.Type(
				"search 'host.services: (protocol=SSH)' --fields host.ip",
				tape.WithSleepAfter(3),
			),
		),
	}
}
