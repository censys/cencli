package collections

import (
	"context"
	"strings"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/collections"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
)

const (
	listCmdName = "list"

	defaultPageSize = 100
	minPageSize     = 1
	defaultMaxPages = 1
)

// ListCommand implements `collections list`, listing an organization's collections.
type ListCommand struct {
	*command.BaseCommand
	// services the command uses
	collectionsSvc collections.Service
	// flags the command uses
	flags listCommandFlags
	// state - populated by PreRun
	orgID    identifiers.OrganizationID
	statuses []string
	pageSize mo.Option[uint64]
	maxPages mo.Option[uint64]
	// result stores the list result for rendering
	result collections.ListResult
}

type listCommandFlags struct {
	orgID    flags.OrgIDFlag
	status   flags.StringSliceFlag
	pageSize flags.IntegerFlag
	maxPages flags.IntegerFlag
}

var _ command.Command = (*ListCommand)(nil)

func NewListCommand(cmdContext *command.Context) *ListCommand {
	return &ListCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *ListCommand) Use() string {
	return listCmdName
}

func (c *ListCommand) Short() string {
	return "List collections"
}

func (c *ListCommand) Long() string {
	return `List collections in your organization. By default only the first page (100 collections) is fetched; use --max-pages -1 to fetch every page.

Results can be filtered by status. Repeat --status, or separate values with commas, to match more than one status.`
}

func (c *ListCommand) Examples() []string {
	return []string{
		"# List all collections",
		"--status active # List only active collections",
		"--status active,paused # List active and paused collections",
		"--max-pages -1 # Fetch every page",
		"--output-format json # Output as JSON",
	}
}

func (c *ListCommand) Args() command.PositionalArgs {
	return command.ExactArgs(0)
}

func (c *ListCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *ListCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *ListCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.status = flags.NewStringSliceFlag(c.Flags(), false, "status", "", nil,
		"filter by status ("+strings.Join(collections.SupportedStatuses, ", ")+"); repeatable")
	c.flags.pageSize = flags.NewIntegerFlag(
		c.Flags(),
		false,
		"page-size",
		"n",
		mo.Some[int64](defaultPageSize),
		"number of collections to return per page",
		mo.Some[int64](minPageSize),
		mo.None[int64](), // the collections API documents no maximum page size
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
	return nil
}

func (c *ListCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	statuses, err := c.flags.status.Value()
	if err != nil {
		return err
	}
	c.statuses = nonEmpty(statuses)
	c.pageSize, c.maxPages, err = parsePaginationFlags(c.flags.pageSize, c.flags.maxPages)
	if err != nil {
		return err
	}
	flagOrgID, err := c.flags.orgID.Value()
	if err != nil {
		return err
	}
	// Every collections endpoint requires an organization. Resolve it after the
	// input checks, as scan does, so an input error is reported first; this
	// still fails before any request when no organization is configured.
	c.orgID, err = c.ResolveRequiredOrgID(cmd, flagOrgID)
	if err != nil {
		return err
	}

	return c.resolveCollectionsService()
}

func (c *ListCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"statuses", c.statuses,
		"pageSize_set", c.pageSize.IsPresent(),
		"maxPages_set", c.maxPages.IsPresent(),
	)

	warnFetchingAllPages(c.Config().Quiet, logger, c.maxPages)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Fetching collections...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.collectionsSvc.ListCollections(pctx, collections.ListParams{
				OrgID:    mo.Some(c.orgID),
				Statuses: c.statuses,
				PageSize: c.pageSize,
				MaxPages: c.maxPages,
			})
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("list collections failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	if renderErr := c.PrintData(c, c.result.Collections); renderErr != nil {
		return renderErr
	}

	if c.result.HasMore {
		printNote(c.Config().Quiet, "Note: more collections are available; use --max-pages -1 to fetch all pages, or raise --max-pages.")
	}

	if c.result.PartialError != nil {
		formatter.PrintError(c.result.PartialError, cmd)
	}

	return nil
}

func (c *ListCommand) resolveCollectionsService() cenclierrors.CencliError {
	svc, err := c.CollectionsService()
	if err != nil {
		return err
	}
	c.collectionsSvc = svc
	return nil
}

// nonEmpty drops blank values, so --status "" leaves the filter off.
func nonEmpty(values []string) []string {
	var out []string
	for _, v := range values {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
