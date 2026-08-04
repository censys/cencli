package tags

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/input"
)

const assignmentsCmdName = "assignments"

// AssignmentsCommand implements `tags assignments <tag>`, listing the assets a
// tag is assigned to. Supports NDJSON streaming for tags with many assets.
type AssignmentsCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags assignmentsCommandFlags
	// state - populated by PreRun
	orgID         mo.Option[identifiers.OrganizationID]
	tagID         identifiers.TagID
	asset         mo.Option[string]
	assetType     mo.Option[string]
	createdBy     mo.Option[string]
	createdBefore mo.Option[time.Time]
	createdAfter  mo.Option[time.Time]
	orderBy       mo.Option[string]
	pageSize      mo.Option[uint64]
	maxPages      mo.Option[uint64]
	// result stores the assignments for rendering
	result tags.AssignmentsResult
}

type assignmentsCommandFlags struct {
	orgID         flags.OrgIDFlag
	asset         flags.StringSliceFlag
	assetType     flags.StringFlag
	createdBy     flags.UUIDFlag
	createdBefore flags.TimestampFlag
	createdAfter  flags.TimestampFlag
	orderBy       flags.StringFlag
	pageSize      flags.IntegerFlag
	maxPages      flags.IntegerFlag
}

var _ command.Command = (*AssignmentsCommand)(nil)

func NewAssignmentsCommand(cmdContext *command.Context) *AssignmentsCommand {
	return &AssignmentsCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *AssignmentsCommand) Use() string {
	return fmt.Sprintf("%s <tag>", assignmentsCmdName)
}

func (c *AssignmentsCommand) Short() string {
	return "List the assets a tag is assigned to"
}

func (c *AssignmentsCommand) Long() string {
	return `List the assets a tag, given by its name or UUID, is assigned to.

Results can be filtered by asset, asset type, creator, and creation time. Use --streaming to emit each assignment as NDJSON as it is fetched.`
}

func (c *AssignmentsCommand) Examples() []string {
	return []string{
		"my-tag # List a tag's assignments",
		"my-tag --asset-type host # Only host assignments",
		"my-tag --asset <ip-address> # Check whether one asset is assigned",
		"my-tag --created-after 2025-01-01T00:00:00Z # Only recent assignments",
		"my-tag --max-pages -1 # Fetch every page",
		"my-tag --streaming # Emit NDJSON as assignments are fetched",
	}
}

func (c *AssignmentsCommand) Args() command.PositionalArgs {
	return command.ExactArgs(1)
}

func (c *AssignmentsCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *AssignmentsCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *AssignmentsCommand) SupportsStreaming() bool {
	return true
}

func (c *AssignmentsCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	// A slice, though only one asset is accepted: repeating the flag then lands
	// in the same slice as a comma-separated list, so both spellings of "more
	// than one asset" hit the one rejection below instead of silently winning.
	c.flags.asset = flags.NewStringSliceFlag(c.Flags(), false, "asset", "", nil,
		"filter by one asset (host IP, certificate SHA-256 fingerprint, or web property hostname:port) - giving more than one is an error")
	c.flags.assetType = flags.NewStringFlag(c.Flags(), false, "asset-type", "", "", "filter by asset type (host, web_property, certificate)")
	c.flags.createdBy = flags.NewUUIDFlag(c.Flags(), false, "created-by", "", mo.None[uuid.UUID](),
		"filter by the UUID of the assignment's creator")
	c.flags.createdBefore = flags.NewTimestampFlag(c.Flags(), false, "created-before", "", mo.None[time.Time](), "only assignments created before this time")
	c.flags.createdAfter = flags.NewTimestampFlag(c.Flags(), false, "created-after", "", mo.None[time.Time](), "only assignments created after this time")
	c.flags.orderBy = flags.NewStringFlag(c.Flags(), false, "order-by", "", "", "sort order (create_time_asc, create_time_desc)")
	c.flags.pageSize = flags.NewIntegerFlag(
		c.Flags(),
		false,
		"page-size",
		"n",
		mo.Some[int64](defaultPageSize),
		"number of assignments to return per page",
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

func (c *AssignmentsCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}

	c.tagID, err = requireTagID(args[0])
	if err != nil {
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

func (c *AssignmentsCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_is_uuid", c.tagID.UID().IsPresent(),
		"asset_set", c.asset.IsPresent(),
		"assetType_set", c.assetType.IsPresent(),
		"pageSize_set", c.pageSize.IsPresent(),
		"maxPages_set", c.maxPages.IsPresent(),
	)

	warnFetchingAllPages(c.Config().Quiet, logger, c.maxPages)

	// Set up streaming output (no-op for non-streaming formats)
	ctx, stopStreaming := c.WithStreamingOutput(cmd.Context(), logger)
	defer stopStreaming(nil)

	err := c.WithProgress(
		ctx,
		logger,
		"Fetching assignments...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.tagsSvc.ListAssignments(pctx, tags.AssignmentsParams{
				OrgID:         c.orgID,
				TagID:         c.tagID,
				AssetID:       c.asset,
				AssetType:     c.assetType,
				CreatedBy:     c.createdBy,
				CreatedBefore: c.createdBefore,
				CreatedAfter:  c.createdAfter,
				OrderBy:       c.orderBy,
				PageSize:      c.pageSize,
				MaxPages:      c.maxPages,
			})
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("list assignments failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	if renderErr := c.PrintData(c, c.result.Assignments); renderErr != nil {
		return renderErr
	}

	if c.result.PartialError != nil {
		formatter.PrintError(c.result.PartialError, cmd)
	}

	return nil
}

func (c *AssignmentsCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}

// parseFilterFlags reads the optional filters; a blank value omits the filter.
func (c *AssignmentsCommand) parseFilterFlags() cenclierrors.CencliError {
	asset, err := c.flags.asset.Value()
	if err != nil {
		return err
	}
	c.asset, err = c.parseAssetFilter(asset)
	if err != nil {
		return err
	}

	assetType, err := c.flags.assetType.Value()
	if err != nil {
		return err
	}
	c.assetType = optionalNonEmpty(assetType)

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

	c.createdBefore, err = c.flags.createdBefore.Value(c.Config().DefaultTZ)
	if err != nil {
		return err
	}
	c.createdAfter, err = c.flags.createdAfter.Value(c.Config().DefaultTZ)
	return err
}

// parseAssetFilter validates the --asset filter so a mistyped asset fails fast
// instead of silently matching nothing. The endpoint filters on one asset, so
// more than one is rejected rather than silently truncated - whether they were
// comma-separated, given as a repeated flag, or both.
func (c *AssignmentsCommand) parseAssetFilter(raw []string) (mo.Option[string], cenclierrors.CencliError) {
	var split []string
	for _, value := range raw {
		if strings.TrimSpace(value) == "" {
			continue
		}
		split = append(split, input.SplitString(value)...)
	}
	if len(split) == 0 {
		return mo.None[string](), nil
	}

	ids, err := classifyAssetIDs(split)
	if err != nil {
		return mo.None[string](), err
	}
	if len(ids) > 1 {
		return mo.None[string](), assets.NewTooManyAssetsError(len(ids), 1)
	}
	return mo.Some(ids[0]), nil
}

func (c *AssignmentsCommand) parsePaginationFlags() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.pageSize, c.maxPages, err = parsePaginationFlags(c.flags.pageSize, c.flags.maxPages)
	return err
}
