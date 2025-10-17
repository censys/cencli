package search

import (
	"context"
	"fmt"

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
	short        bool
}

// searchCommandFlags contains all flag handles used by the search command.
type searchCommandFlags struct {
	orgID        flags.OrgIDFlag
	collectionID flags.UUIDFlag
	fields       flags.StringSliceFlag
	pageSize     flags.IntegerFlag
	maxPages     flags.IntegerFlag
	short        flags.BoolFlag
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

func (c *Command) Examples() []string {
	return []string{
		`"host.ip: 1.1.1.1/16"`,
		`--fields host.ip,host.location "host.services: (protocol=SSH and not port: 22)"`,
		`--collection-id <your-collection-id> "host.services.protocol=SSH"`,
		`--page-size 50 --max-pages 5 "cert.names=censys.com"`,
		`--max-pages -1 "host.services.port: 443 and host.location.country: Germany"`,
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
	c.flags.short = flags.NewShortFlag(c.Flags(), "")
	return nil
}

// PreRun validates flags and prepares the command for execution.
func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// args have already been validated
	c.query = args[0]

	if err := c.parseOrgIDFlag(); err != nil {
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
	if err := c.parseShortFlag(); err != nil {
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
		"query", c.query,
	)
	if !c.Config().Quiet && !c.maxPages.IsPresent() {
		msg := styles.GlobalStyles.Warning.Render("Warning: fetching all pages (--max-pages=-1). This may take a while and increase API usage.")
		formatter.Println(formatter.Stderr, msg)
		logger.Debug("fetching all pages", "message", msg)
	}

	var result search.Result

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Fetching search results...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			result, fetchErr = c.fetchSearchResult(pctx)
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("fetch failed", "error", err)
		return err
	}

	// Render the search results (even if partial)
	if renderErr := c.renderSearchResult(result); renderErr != nil {
		return renderErr
	}

	// If there was a partial error, print it to stderr after rendering the data
	if result.PartialError != nil {
		formatter.PrintError(result.PartialError, cmd)
	}

	return nil
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

// renderSearchResult prints response metadata and the results, using a short template when requested.
func (c *Command) renderSearchResult(result search.Result) cenclierrors.CencliError {
	c.PrintAppResponseMeta(result.Meta)
	// wrap each hit with the type it is, to help differentiate in the output
	data := make([]any, len(result.Hits))
	for i, hit := range result.Hits {
		data[i] = map[string]any{
			hit.AssetType().String(): hit,
		}
	}
	if c.short && !c.Config().Quiet {
		return c.PrintDataWithTemplate(config.TemplateEntitySearchResult, data)
	}
	return c.PrintData(data)
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

// parseOrgIDFlag parses the optional org-id flag into c.orgID.
func (c *Command) parseOrgIDFlag() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
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

// parseFieldsFlag parses the optional fields flag into c.fields.
func (c *Command) parseFieldsFlag() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.fields, err = c.flags.fields.Value()
	if err != nil {
		return err
	}
	return nil
}

// parseShortFlag parses the short flag into c.short.
func (c *Command) parseShortFlag() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.short, err = c.flags.short.Value()
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
