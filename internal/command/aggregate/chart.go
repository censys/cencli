package aggregate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/aggregate"
	"github.com/censys/cencli/internal/app/chartgen"
	"github.com/censys/cencli/internal/app/chartgen/prompts"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/store"
)

const (
	chartCmdName          = "chart"
	defaultChartNumImages = 1
	defaultChartDir       = "output"
	defaultChartName      = "chart"
)

type chartCommand struct {
	*command.BaseCommand
	// services
	aggregateSvc aggregate.Service
	chartgenSvc  chartgen.Service
	// flags
	flags chartCommandFlags
	// state
	collectionID mo.Option[identifiers.CollectionID]
	orgID        mo.Option[identifiers.OrganizationID]
	query        string
	field        string
	numBuckets   int64
	dir          string
	name         string
	chartType    string
	numImages    int
}

type chartCommandFlags struct {
	orgID        flags.OrgIDFlag
	collectionID flags.UUIDFlag
	numBuckets   flags.IntegerFlag
	dir          flags.StringFlag
	name         flags.StringFlag
	chartType    flags.StringFlag
	numImages    flags.IntegerFlag
}

var _ command.Command = (*chartCommand)(nil)

func newChartCommand(cmdContext *command.Context) *chartCommand {
	return &chartCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *chartCommand) Use() string {
	return fmt.Sprintf("%s <query> <field>", chartCmdName)
}

func (c *chartCommand) Short() string {
	return "Generate a chart image from aggregation data using AI"
}

func (c *chartCommand) Long() string {
	return `Generate a chart image from aggregation data using Gemini AI.

This command fetches aggregation data from Censys and uses Google's Gemini AI
to generate a visualization chart. The generated chart is saved as a PNG file.

Before using this command, configure your Gemini API key:
  censys config gemini set

Available chart types: ` + strings.Join(prompts.ChartTypes(), ", ")
}

func (c *chartCommand) Args() command.PositionalArgs {
	return command.ExactArgs(2)
}

func (c *chartCommand) Examples() []string {
	return []string{
		`"host.location.continent='Asia'" host.location.city --type geomap`,
		`"host.services.port=443" host.services.software.product --type bar --dir ./charts`,
		`"host.services.port=22" host.location.country --type choropleth --name ssh-by-country`,
	}
}

func (c *chartCommand) Init() error {
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
		mo.Some[int64](500),
		"number of buckets to use for aggregation",
		mo.Some[int64](minNumBuckets),
		mo.Some[int64](maxNumBuckets),
	)
	c.flags.dir = flags.NewStringFlag(
		c.Flags(),
		false,
		"dir",
		"d",
		defaultChartDir,
		"output directory for generated files",
	)
	c.flags.name = flags.NewStringFlag(
		c.Flags(),
		false,
		"name",
		"",
		defaultChartName,
		"base name for output files (without extension)",
	)
	c.flags.chartType = flags.NewStringFlag(
		c.Flags(),
		false,
		"type",
		"t",
		"",
		"chart type ("+strings.Join(prompts.ChartTypes(), ", ")+")",
	)
	c.flags.numImages = flags.NewIntegerFlag(
		c.Flags(),
		false,
		"tries",
		"",
		mo.Some[int64](defaultChartNumImages),
		"number of chart images to generate",
		mo.Some[int64](1),
		mo.Some[int64](10),
	)
	return nil
}

func (c *chartCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError

	// Get aggregate service
	c.aggregateSvc, err = c.AggregateService()
	if err != nil {
		return err
	}

	// Get Gemini API key and create chartgen service
	apiKey, storeErr := c.Store().GetLastUsedGlobalByName(cmd.Context(), config.GeminiAPIKeyGlobalName)
	if storeErr != nil {
		if errors.Is(storeErr, store.ErrGlobalNotFound) {
			return newGeminiAPIKeyNotConfiguredError()
		}
		return cenclierrors.NewCencliError(fmt.Errorf("failed to get Gemini API key: %w", storeErr))
	}
	c.chartgenSvc = chartgen.New(apiKey.Value)

	// Parse args
	c.query = args[0]
	c.field = args[1]

	// Validate flags
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}

	collectionID, err := c.flags.collectionID.Value()
	if err != nil {
		return err
	}
	if collectionID.IsPresent() {
		c.collectionID = mo.Some(identifiers.NewCollectionID(collectionID.MustGet()))
	}

	numBuckets, err := c.flags.numBuckets.Value()
	if err != nil {
		return err
	}
	if numBuckets.IsPresent() {
		c.numBuckets = numBuckets.MustGet()
	}

	c.dir, err = c.flags.dir.Value()
	if err != nil {
		return err
	}

	c.name, err = c.flags.name.Value()
	if err != nil {
		return err
	}

	c.chartType, err = c.flags.chartType.Value()
	if err != nil {
		return err
	}

	numImages, err := c.flags.numImages.Value()
	if err != nil {
		return err
	}
	if numImages.IsPresent() {
		c.numImages = int(numImages.MustGet())
	} else {
		c.numImages = defaultChartNumImages
	}

	return nil
}

func (c *chartCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(chartCmdName).With(
		"query", c.query,
		"field", c.field,
		"chartType", c.chartType,
		"numBuckets", c.numBuckets,
	)

	// Fetch aggregation data
	var aggResult aggregate.Result
	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Fetching aggregation data...",
		func(pctx context.Context) cenclierrors.CencliError {
			params := aggregate.Params{
				OrgID:        c.orgID,
				CollectionID: c.collectionID,
				Query:        c.query,
				Field:        c.field,
				NumBuckets:   c.numBuckets,
			}
			var fetchErr cenclierrors.CencliError
			aggResult, fetchErr = c.aggregateSvc.Aggregate(pctx, params)
			return fetchErr
		},
	)
	if err != nil {
		return err
	}

	if len(aggResult.Buckets) == 0 {
		formatter.Printf(formatter.Stdout, "No aggregation results found for query.\n")
		return nil
	}

	// Convert buckets to chartgen format
	chartBuckets := make([]prompts.Bucket, len(aggResult.Buckets))
	var totalCount uint64
	for i, b := range aggResult.Buckets {
		chartBuckets[i] = prompts.Bucket{
			Key:   b.Key,
			Count: b.Count,
		}
		totalCount += b.Count
	}

	// Generate chart
	var chartResult chartgen.Result
	err = c.WithProgress(
		cmd.Context(),
		logger,
		"Generating chart with AI...",
		func(pctx context.Context) cenclierrors.CencliError {
			params := chartgen.Params{
				Query:      c.query,
				Field:      c.field,
				ChartType:  c.chartType,
				Buckets:    chartBuckets,
				TotalCount: totalCount,
				OtherCount: 0,
				NumImages:  c.numImages,
			}
			var genErr cenclierrors.CencliError
			chartResult, genErr = c.chartgenSvc.GenerateChart(pctx, params)
			return genErr
		},
	)
	if err != nil {
		return err
	}

	// Prepare output directory
	dir := c.dir
	if !filepath.IsAbs(dir) {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return cenclierrors.NewCencliError(fmt.Errorf("failed to get working directory: %w", wdErr))
		}
		dir = filepath.Join(wd, dir)
	}

	if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to create output directory: %w", mkErr))
	}

	// Write prompt file
	promptPath := filepath.Join(dir, c.name+".txt")
	if writeErr := os.WriteFile(promptPath, []byte(chartResult.Prompt), 0644); writeErr != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("failed to write prompt file: %w", writeErr))
	}
	formatter.Printf(formatter.Stdout, "✅ Wrote prompt: %s\n", promptPath)

	// Write image files
	for i, imageData := range chartResult.Images {
		imagePath := filepath.Join(dir, fmt.Sprintf("%s_%d.png", c.name, i))
		if writeErr := os.WriteFile(imagePath, imageData, 0644); writeErr != nil {
			return cenclierrors.NewCencliError(fmt.Errorf("failed to write image file: %w", writeErr))
		}
		formatter.Printf(formatter.Stdout, "✅ Wrote image: %s\n", imagePath)
	}

	return nil
}

// GeminiAPIKeyNotConfiguredError indicates the Gemini API key needs to be set up.
type GeminiAPIKeyNotConfiguredError interface {
	cenclierrors.CencliError
}

type geminiAPIKeyNotConfiguredError struct{}

var _ GeminiAPIKeyNotConfiguredError = &geminiAPIKeyNotConfiguredError{}

func newGeminiAPIKeyNotConfiguredError() GeminiAPIKeyNotConfiguredError {
	return &geminiAPIKeyNotConfiguredError{}
}

func (e *geminiAPIKeyNotConfiguredError) Error() string {
	return "Gemini API key not configured. Run 'censys config gemini set' to configure it."
}

func (e *geminiAPIKeyNotConfiguredError) Title() string {
	return "Gemini API Key Not Configured"
}

func (e *geminiAPIKeyNotConfiguredError) ShouldPrintUsage() bool {
	return false
}
