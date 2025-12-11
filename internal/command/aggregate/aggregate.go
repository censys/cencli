package aggregate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/aggregate"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/tape"
	"github.com/censys/cencli/internal/pkg/ui/rawtable"
	"github.com/censys/cencli/internal/pkg/ui/table"
)

const (
	cmdName = "aggregate"

	defaultNumBuckets = 25
	minNumBuckets     = 1
	maxNumBuckets     = 10000
)

type Command struct {
	*command.BaseCommand
	// services the command uses
	aggregateSvc aggregate.Service
	// flags the command uses
	flags aggregateCommandFlags
	// state - populated by PreRun (through flags, args, etc.)
	collectionID  mo.Option[identifiers.CollectionID]
	orgID         mo.Option[identifiers.OrganizationID]
	query         string
	field         string
	numBuckets    int64
	countByLevel  mo.Option[aggregate.CountByLevel]
	filterByQuery bool
	interactive   bool
	raw           bool
}

type aggregateCommandFlags struct {
	orgID         flags.OrgIDFlag
	collectionID  flags.UUIDFlag
	numBuckets    flags.IntegerFlag
	countByLevel  flags.StringFlag
	filterByQuery flags.BoolFlag
	interactive   flags.BoolFlag
	raw           flags.BoolFlag
}

var _ command.Command = (*Command)(nil)

func NewAggregateCommand(cmdContext *command.Context) *Command {
	return &Command{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *Command) Use() string {
	return fmt.Sprintf("%s <query> <field>", cmdName)
}

func (c *Command) Short() string {
	return "Aggregate results for a Platform search query"
}

func (c *Command) Long() string {
	return `Aggregate results for a Platform search query. This functionality is equivalent to the Report Builder in the Platform web UI.`
}

func (c *Command) Args() command.PositionalArgs {
	return command.ExactArgs(2)
}

func (c *Command) Examples() []string {
	return []string{
		`"host.services.protocol=SSH" "host.services.port"`,
		`-c <your-collection-id> "services.service_name:HTTP" "services.port"`,
	}
}

func (c *Command) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.collectionID = flags.NewUUIDFlag(
		c.Flags(),
		false,
		"collection-id",
		"c",
		mo.None[uuid.UUID](),
		"collection to aggregate within (optional)",
	)
	c.flags.numBuckets = flags.NewIntegerFlag(
		c.Flags(),
		false,
		"num-buckets",
		"n",
		mo.Some[int64](defaultNumBuckets),
		"number of buckets to split results into",
		mo.Some[int64](minNumBuckets),
		mo.Some[int64](maxNumBuckets),
	)
	c.flags.countByLevel = flags.NewStringFlag(
		c.Flags(),
		false,
		"count-by-level",
		"l",
		"",
		"which document level's count is returned per term bucket",
	)
	c.flags.filterByQuery = flags.NewBoolFlag(
		c.Flags(),
		"filter-by-query",
		"f",
		false,
		"whether aggregation results are limited to values that match the query",
	)
	c.flags.interactive = flags.NewBoolFlag(
		c.Flags(),
		"interactive",
		"i",
		false,
		"display results in an interactive table (TUI)",
	)
	c.flags.raw = flags.NewBoolFlag(
		c.Flags(),
		"raw",
		"r",
		false,
		"output raw data",
	)
	// Add chart subcommand
	return c.AddSubCommands(
		newChartCommand(c.Context),
	)
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.aggregateSvc, err = c.AggregateService()
	if err != nil {
		return err
	}
	// args have already been validated
	c.query = args[0]
	c.field = args[1]
	// validate orgID (if present)
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}
	// validate collectionID (if present)
	collectionID, err := c.flags.collectionID.Value()
	if err != nil {
		return err
	}
	if collectionID.IsPresent() {
		c.collectionID = mo.Some(identifiers.NewCollectionID(collectionID.MustGet()))
	}
	// validate numBuckets (if present)
	numBuckets, err := c.flags.numBuckets.Value()
	if err != nil {
		return err
	}
	if numBuckets.IsPresent() {
		c.numBuckets = numBuckets.MustGet()
	}
	// validate countByLevel (if present)
	countByLevel, err := c.flags.countByLevel.Value()
	if err != nil {
		return err
	}
	if countByLevel != "" {
		c.countByLevel = mo.Some(aggregate.CountByLevel(countByLevel))
	}
	// validate filterByQuery (if present)
	c.filterByQuery, err = c.flags.filterByQuery.Value()
	if err != nil {
		return err
	}
	// validate interactive (if present)
	c.interactive, err = c.flags.interactive.Value()
	if err != nil {
		return err
	}
	// validate raw (if present)
	c.raw, err = c.flags.raw.Value()
	if err != nil {
		return err
	}
	// validate that raw and interactive are not both set
	if c.raw && c.interactive {
		return flags.NewConflictingFlagsError("raw", "interactive")
	}
	return nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"collectionID_set", c.collectionID.IsPresent(),
		"query", c.query,
		"field", c.field,
		"numBuckets", c.numBuckets,
		"countByLevel_set", c.countByLevel.IsPresent(),
		"filterByQuery", c.filterByQuery,
	)
	var result aggregate.Result
	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Fetching aggregation results...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			result, fetchErr = c.fetchAggregateResult(pctx)
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("fetch failed", "error", err)
		return err
	}
	return c.renderAggregateResult(result)
}

func (c *Command) fetchAggregateResult(ctx context.Context) (aggregate.Result, cenclierrors.CencliError) {
	params := c.buildAggregateParams()
	result, err := c.aggregateSvc.Aggregate(ctx, params)
	if err != nil {
		return aggregate.Result{}, err
	}
	return result, nil
}

// buildAggregateParams prepares the aggregation parameters from command state.
func (c *Command) buildAggregateParams() aggregate.Params {
	return aggregate.Params{
		OrgID:         c.orgID,
		CollectionID:  c.collectionID,
		Query:         c.query,
		Field:         c.field,
		NumBuckets:    c.numBuckets,
		CountByLevel:  c.countByLevel,
		FilterByQuery: mo.Some(c.filterByQuery),
	}
}

func (c *Command) renderAggregateResult(result aggregate.Result) cenclierrors.CencliError {
	c.PrintAppResponseMeta(result.Meta)
	if c.interactive {
		return c.showInteractiveTable(result)
	}
	if c.raw {
		c.PrintAppResponseMeta(result.Meta)
		return c.PrintData(result.Buckets)
	}
	// Default: show raw table
	return c.showRawTable(result)
}

// buildTableTitle constructs a title string that includes the query, count-by-level, and filter-by-query settings.
func (c *Command) buildTableTitle() string {
	title := fmt.Sprintf("query: %s", c.query)

	if c.countByLevel.IsPresent() {
		title += fmt.Sprintf(" | count by: %s", c.countByLevel.MustGet())
	} else {
		title += " | count by: \"\""
	}

	if c.filterByQuery {
		title += " | filtered: true"
	} else {
		title += " | filtered: false"
	}

	return title
}

func (c *Command) showInteractiveTable(result aggregate.Result) cenclierrors.CencliError {
	title := c.buildTableTitle()
	tbl := table.NewTable[aggregate.Bucket](
		[]string{"count", c.field},
		func(bucket aggregate.Bucket) []string {
			return []string{
				strconv.FormatUint(bucket.Count, 10),
				bucket.Key,
			}
		},
		table.WithColumnWidths[aggregate.Bucket]([]int{15, len(c.field) + 5}),
		table.WithTitle[aggregate.Bucket](title),
	)
	if err := tbl.Run(result.Buckets); err != nil {
		return cenclierrors.NewCencliError(
			fmt.Errorf("failed to display interactive table: %w", err),
		)
	}
	return nil
}

func (c *Command) showRawTable(result aggregate.Result) cenclierrors.CencliError {
	if len(result.Buckets) == 0 {
		fmt.Fprintf(formatter.Stdout, "\nNo results found.\n")
		return nil
	}

	columns := []rawtable.Column[aggregate.Bucket]{
		{
			Title: "Count",
			String: func(b aggregate.Bucket) string {
				return strconv.FormatUint(b.Count, 10)
			},
			Style: func(s string, b aggregate.Bucket) string {
				return styles.NewStyle(styles.ColorOffWhite).Render(s)
			},
			AlignRight: true,
		},
		{
			Title: c.field,
			String: func(b aggregate.Bucket) string {
				return b.Key
			},
			Style: func(s string, b aggregate.Bucket) string {
				return styles.NewStyle(styles.ColorTeal).Render(s)
			},
		},
	}

	tbl := rawtable.New(
		columns,
		rawtable.WithHeaderStyle[aggregate.Bucket](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[aggregate.Bucket](!formatter.StdoutIsTTY()),
	)

	fmt.Fprintf(formatter.Stdout, "\n=== Aggregation Results ===\n\n")
	fmt.Fprintf(formatter.Stdout, "%s\n\n", c.buildTableTitle())
	fmt.Fprint(formatter.Stdout, tbl.Render(result.Buckets))

	return nil
}

func (*Command) Tapes(recorder *tape.Recorder) []tape.Tape {
	return []tape.Tape{
		tape.NewTape("aggregate",
			tape.DefaultTapeConfig(),
			recorder.Type(
				"aggregate 'host.services.port=22' host.services.protocol -n 5",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
			recorder.Type(
				"aggregate 'host.services.port=22' host.services.protocol --raw",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
			recorder.Type(
				"aggregate 'host.services.port=22' host.services.protocol -i",
				tape.WithSleepAfter(3),
			),
			recorder.Press("j", 3),
		),
	}
}
