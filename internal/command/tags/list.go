package tags

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/tags"
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
	// maxPageSize is the largest page the tags endpoints accept; rejecting more
	// here saves a request that could only come back as a 422.
	maxPageSize     = 1000
	defaultMaxPages = 1
)

// ListCommand implements `tags list`, listing an organization's tags.
type ListCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags listCommandFlags
	// state - populated by PreRun
	orgID     mo.Option[identifiers.OrganizationID]
	privacy   mo.Option[string]
	name      mo.Option[string]
	createdBy mo.Option[string]
	orderBy   mo.Option[string]
	pageSize  mo.Option[uint64]
	maxPages  mo.Option[uint64]
	// result stores the list result for rendering
	result tags.ListResult
}

type listCommandFlags struct {
	orgID     flags.OrgIDFlag
	privacy   flags.StringFlag
	name      flags.StringFlag
	createdBy flags.UUIDFlag
	orderBy   flags.StringFlag
	pageSize  flags.IntegerFlag
	maxPages  flags.IntegerFlag
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
	return "List all tags"
}

func (c *ListCommand) Long() string {
	return `List all tags in your organization.

Results can be filtered by privacy, name, and creator, and sorted by various fields.`
}

func (c *ListCommand) Examples() []string {
	return []string{
		"# List all tags",
		"--privacy shared # List only shared tags",
		"--name my-tag # Filter by exact name",
		"--order-by name_desc # Sort by name descending",
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
	c.flags.privacy = flags.NewStringFlag(c.Flags(), false, "privacy", "", "", "filter by privacy (private, shared)")
	c.flags.name = flags.NewStringFlag(c.Flags(), false, "name", "", "", "filter by exact tag name")
	c.flags.createdBy = flags.NewUUIDFlag(c.Flags(), false, "created-by", "", mo.None[uuid.UUID](),
		"filter by the UUID of the tag's creator")
	c.flags.orderBy = flags.NewStringFlag(c.Flags(), false, "order-by", "", "", "sort order (name_asc, name_desc, created_at_asc, created_at_desc, updated_at_asc, updated_at_desc)")
	c.flags.pageSize = flags.NewIntegerFlag(
		c.Flags(),
		false,
		"page-size",
		"n",
		mo.Some[int64](defaultPageSize),
		"number of tags to return per page",
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

func (c *ListCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	if err := c.parseOrgIDFlag(); err != nil {
		return err
	}
	if err := c.parseFilterFlags(); err != nil {
		return err
	}
	if err := c.parsePaginationFlags(); err != nil {
		return err
	}
	return c.resolveTagsService()
}

func (c *ListCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"privacy_set", c.privacy.IsPresent(),
		"name_set", c.name.IsPresent(),
		"pageSize_set", c.pageSize.IsPresent(),
		"maxPages_set", c.maxPages.IsPresent(),
	)

	warnFetchingAllPages(c.Config().Quiet, logger, c.maxPages)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Fetching tags...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.tagsSvc.ListTags(pctx, tags.ListParams{
				OrgID:     c.orgID,
				Privacy:   c.privacy,
				Name:      c.name,
				CreatedBy: c.createdBy,
				OrderBy:   c.orderBy,
				PageSize:  c.pageSize,
				MaxPages:  c.maxPages,
			})
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("list tags failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	if renderErr := c.PrintData(c, c.result.Tags); renderErr != nil {
		return renderErr
	}

	if c.result.PartialError != nil {
		formatter.PrintError(c.result.PartialError, cmd)
	}

	return nil
}

func (c *ListCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}

func (c *ListCommand) parseOrgIDFlag() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	return err
}

// parseFilterFlags reads the optional string filters; a blank value omits the filter.
func (c *ListCommand) parseFilterFlags() cenclierrors.CencliError {
	privacy, err := c.flags.privacy.Value()
	if err != nil {
		return err
	}
	c.privacy = optionalNonEmpty(privacy)

	name, err := c.flags.name.Value()
	if err != nil {
		return err
	}
	c.name = optionalNonEmpty(name)

	createdBy, err := c.flags.createdBy.Value()
	if err != nil {
		return err
	}
	c.createdBy = uuidFilterString(createdBy)

	orderBy, err := c.flags.orderBy.Value()
	if err != nil {
		return err
	}
	c.orderBy = optionalNonEmpty(orderBy)

	return nil
}

func (c *ListCommand) parsePaginationFlags() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.pageSize, c.maxPages, err = parsePaginationFlags(c.flags.pageSize, c.flags.maxPages)
	return err
}
