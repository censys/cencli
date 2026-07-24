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

const assignCmdName = "assign"

// AssignCommand implements `tags assign <tag> [asset...]`, linking a tag to
// explicit assets given positionally or via --input-file (a file or - for
// stdin). Mixed asset types are allowed since each assignment is independent.
type AssignCommand struct {
	*command.BaseCommand
	// services the command uses
	tagsSvc tags.Service
	// flags the command uses
	flags assignCommandFlags
	// state - populated by PreRun
	orgID    mo.Option[identifiers.OrganizationID]
	tagID    identifiers.TagID
	assetIDs []string
	// result stores the assignment outcome for rendering
	result tags.AssignResult
}

type assignCommandFlags struct {
	orgID     flags.OrgIDFlag
	inputFile flags.FileFlag
}

// assignedAsset is the data-mode payload for a single assignment outcome.
type assignedAsset struct {
	Asset        string `json:"asset" yaml:"asset"`
	AssignmentID string `json:"assignment_id,omitempty" yaml:"assignment_id,omitempty"`
	AssetType    string `json:"asset_type,omitempty" yaml:"asset_type,omitempty"`
	PlatformRef  string `json:"platform_ref,omitempty" yaml:"platform_ref,omitempty"`
	Assigned     bool   `json:"assigned" yaml:"assigned"`
	Error        string `json:"error,omitempty" yaml:"error,omitempty"`
}

var _ command.Command = (*AssignCommand)(nil)

func NewAssignCommand(cmdContext *command.Context) *AssignCommand {
	return &AssignCommand{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *AssignCommand) Use() string {
	return fmt.Sprintf("%s <tag> [asset...]", assignCmdName)
}

func (c *AssignCommand) Short() string {
	return "Assign a tag to one or more assets"
}

func (c *AssignCommand) Long() string {
	return `Assign a tag, by its name or UUID, to one or more assets (host IPs, certificate SHA-256 fingerprints, or web property hostname:port).

Assets can be passed as positional arguments or read from a file (or STDIN) with --input-file. Assets of different types can be mixed in a single call. Each asset is assigned independently: if one fails the rest still proceed, and the per-asset outcomes are reported.`
}

func (c *AssignCommand) Examples() []string {
	return []string{
		"<tag> <ip-address>",
		"<tag> <ip-address> <ip-address>",
		"<tag> <hostname:port>",
		"<tag> <sha256-fingerprint>",
		"<tag> <ip-address> <hostname:port>  # asset types can be mixed",
		"<tag> --input-file <file>",
		"<tag> --input-file -  # read assets from STDIN",
	}
}

func (c *AssignCommand) Args() command.PositionalArgs {
	// At least the tag; assets may instead come from --input-file.
	return command.MinimumNArgs(1)
}

func (c *AssignCommand) DefaultOutputType() command.OutputType {
	return command.OutputTypeShort
}

func (c *AssignCommand) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeShort, command.OutputTypeData}
}

func (c *AssignCommand) Init() error {
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.inputFile = flags.NewFileFlag(c.Flags(), false, "input-file", "i", "file to read the assets from (or - for STDIN). Overrides positional asset arguments.")
	return nil
}

func (c *AssignCommand) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.orgID, err = c.flags.orgID.Value()
	if err != nil {
		return err
	}

	c.tagID, err = requireTagID(args[0])
	if err != nil {
		return err
	}

	c.assetIDs, err = gatherAssetIDs(cmd, c.flags.inputFile, args)
	if err != nil {
		return err
	}

	return c.resolveTagsService()
}

func (c *AssignCommand) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"tagID_is_uuid", c.tagID.UID().IsPresent(),
		"count", len(c.assetIDs),
	)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Assigning tag...",
		func(pctx context.Context) cenclierrors.CencliError {
			var assignErr cenclierrors.CencliError
			c.result, assignErr = c.tagsSvc.Assign(pctx, tags.AssignParams{
				OrgID:    c.orgID,
				TagID:    c.tagID,
				AssetIDs: c.assetIDs,
			})
			return assignErr
		},
	)
	if err != nil {
		logger.Debug("assign tag failed", "error", err)
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)

	if renderErr := c.PrintData(c, c.assignmentViews()); renderErr != nil {
		return renderErr
	}

	if len(c.result.Assignments) > 0 {
		formatter.Println(formatter.Stderr,
			"Note: newly assigned tags may take a few minutes to appear in `tags:` search results.")
	}

	if c.result.PartialError != nil {
		formatter.PrintError(c.result.PartialError, cmd)
	}

	return nil
}

func (c *AssignCommand) resolveTagsService() cenclierrors.CencliError {
	svc, err := c.TagsService()
	if err != nil {
		return err
	}
	c.tagsSvc = svc
	return nil
}

// assignmentViews builds the render payload: successes first, then failures.
func (c *AssignCommand) assignmentViews() []assignedAsset {
	views := make([]assignedAsset, 0, len(c.result.Assignments)+len(c.result.Failures))
	for _, a := range c.result.Assignments {
		views = append(views, assignedAsset{
			Asset:        a.AssetID,
			AssignmentID: a.ID,
			AssetType:    a.AssetType,
			PlatformRef:  a.PlatformRef,
			Assigned:     true,
		})
	}
	for _, f := range c.result.Failures {
		views = append(views, assignedAsset{
			Asset:    f.AssetID,
			Assigned: false,
			Error:    f.Err.Error(),
		})
	}
	return views
}
