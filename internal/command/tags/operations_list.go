package tags

import (
	"context"
	"fmt"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
)

const operationsListCmdName = "list"

// OperationsListCommand implements `tags operations list [<tag>]`, listing the
// asynchronous bulk jobs for one tag or for the whole organization.
type OperationsListCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags operationsListCommandFlags
	// state - populated by PreRun
	orgID    mo.Option[identifiers.OrganizationID]
	tagID    mo.Option[identifiers.TagID]
	opType   mo.Option[string]
	status   mo.Option[string]
	orderBy  mo.Option[string]
	pageSize mo.Option[uint64]
	maxPages mo.Option[uint64]
	// result stores the operations for rendering
	result tags.OperationsResult
}

type operationsListCommandFlags struct {
	orgID    flags.OrgIDFlag
	opType   flags.StringFlag
	status   flags.StringFlag
	orderBy  flags.StringFlag
	pageSize flags.IntegerFlag
	maxPages flags.IntegerFlag
}

var _ command.Command = (*OperationsListCommand)(nil)

func NewOperationsListCommand(cmdContext *command.Context) *OperationsListCommand {
	return &OperationsListCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *OperationsListCommand) Use() string {
	return fmt.Sprintf("%s [<tag>]", operationsListCmdName)
}

func (c *OperationsListCommand) Short() string {
	return "List bulk tag operations"
}

func (c *OperationsListCommand) Long() string {
	return `List the asynchronous jobs created by bulk tag operations.

Given a tag, by its name or UUID, only that tag's operations are listed; omit it to list operations across every tag in the organization.`
}

func (c *OperationsListCommand) Examples() []string {
	return []string{
		"# List operations across every tag",
		"my-tag # List one tag's operations",
		"my-tag --status running # Only operations still in flight",
		"--type bulk_delete # Only bulk unassign jobs",
		"--max-pages -1 # Fetch every page",
	}
}

func (c *OperationsListCommand) Args() command.PositionalArgs {
	return command.RangeArgs(0, 1)
}

func (c *OperationsListCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *OperationsListCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *OperationsListCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.opType = flags.NewStringFlag(c.Flags(), false, "type", "", "", "filter by operation type (bulk_create, bulk_delete)")
	c.flags.status = flags.NewStringFlag(c.Flags(), false, "status", "", "",
		"filter by status (pending, running, succeeded, limit_reached, failed, cancelled)")
	c.flags.orderBy = flags.NewStringFlag(c.Flags(), false, "order-by", "", "", "sort order (create_time_asc, create_time_desc)")
	c.flags.pageSize = flags.NewIntegerFlag(
		c.Flags(),
		false,
		"page-size",
		"n",
		mo.Some[int64](defaultPageSize),
		"number of operations to return per page",
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
	return nil
}

func (c *OperationsListCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}

	// The tag is optional here; without it the service lists org-wide.
	if len(args) == 1 {
		tagID, tagErr := requireTagID(args[0])
		if tagErr != nil {
			return tagErr
		}
		c.tagID = mo.Some(tagID)
	}

	if err := c.parseFilterFlags(); err != nil {
		return err
	}
	if err := c.parsePaginationFlags(); err != nil {
		return err
	}

	return c.resolveTagsService()
}

func (c *OperationsListCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_set", c.tagID.IsPresent(),
		"type_set", c.opType.IsPresent(),
		"status_set", c.status.IsPresent(),
		"pageSize_set", c.pageSize.IsPresent(),
		"maxPages_set", c.maxPages.IsPresent(),
	)

	warnFetchingAllPages(c.Config().Quiet, logger, c.maxPages)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Fetching operations...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.tagsSvc.ListOperations(pctx, tags.OperationsParams{
				OrgID:    c.orgID,
				TagID:    c.tagID,
				Type:     c.opType,
				Status:   c.status,
				OrderBy:  c.orderBy,
				PageSize: c.pageSize,
				MaxPages: c.maxPages,
			})
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("list operations failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	if renderErr := c.PrintData(c, c.result.Operations); renderErr != nil {
		return renderErr
	}

	if c.result.PartialError != nil {
		formatter.PrintError(c.result.PartialError, cmd)
	}

	return nil
}

func (c *OperationsListCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}

// parseFilterFlags reads the optional filters; a blank value omits the filter.
func (c *OperationsListCommand) parseFilterFlags() cenclierrors.CencliError {
	opType, err := c.flags.opType.Value()
	if err != nil {
		return err
	}
	c.opType = optionalNonEmpty(opType)

	status, err := c.flags.status.Value()
	if err != nil {
		return err
	}
	c.status = optionalNonEmpty(status)

	orderBy, err := c.flags.orderBy.Value()
	if err != nil {
		return err
	}
	c.orderBy = optionalNonEmpty(orderBy)

	return nil
}

func (c *OperationsListCommand) parsePaginationFlags() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.pageSize, c.maxPages, err = parsePaginationFlags(c.flags.pageSize, c.flags.maxPages)
	return err
}
