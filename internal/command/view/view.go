package view

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/view"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/config"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/assets"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/domain/responsemeta"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/input"
	"github.com/censys/cencli/internal/pkg/tape"
)

const (
	cmdName = "view"
)

type Command struct {
	*command.BaseCommand
	// services the command uses
	viewSvc view.Service
	// flags the command uses
	flags viewCommandFlags
	// state - populated by PreRun (through flags, etc.)
	assets    *assets.AssetClassifier
	assetType assets.AssetType
	orgID     mo.Option[identifiers.OrganizationID]
	atTime    mo.Option[time.Time]
	short     bool
}

type viewCommandFlags struct {
	orgID     flags.OrgIDFlag
	inputFile flags.FileFlag
	atTime    flags.TimestampFlag
	short     flags.BoolFlag
}

var _ command.Command = (*Command)(nil)

func NewViewCommand(cmdContext *command.Context) *Command {
	cmd := &Command{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
	return cmd
}

func (c *Command) Use() string {
	return fmt.Sprintf("%s <asset>", cmdName)
}

func (c *Command) Short() string {
	return "Retrieve information about hosts, certificates, and web properties"
}

func (c *Command) Long() string {
	return "Retrieve information about hosts, certificates, and web properties.\nSupports defanged IPs / URLs."
}

func (c *Command) Examples() []string {
	return []string{
		"8.8.8.8",
		"3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf",
		"platform.censys.io:80",
		"platform.censys.io # defaults to port 443",
		"platform.censys.io:80,google.com:80",
		"--input-file hosts.txt",
		"--input-file -  # read assets from STDIN",
		"platform.censys.io:80 --at-time 2025-09-15T14:30:00Z",
		"8.8.8.8 --short",
	}
}

func (c *Command) Init() error {
	// initialize command-specific flags
	c.flags.inputFile = flags.NewFileFlag(c.Flags(), false, "input-file", "i", "file to read the assets from. Overrides the positional argument.")
	c.flags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.flags.atTime = flags.NewTimestampFlag(c.Flags(), false, "at-time", "", mo.None[time.Time](), "view data as of this time (certificates not supported)")
	// add aliases: --at and -a
	c.flags.atTime.AddAlias("at", "a", "Alias for --at-time")
	c.flags.short = flags.NewShortFlag(c.Flags(), "")
	return nil
}

func (c *Command) Args() command.PositionalArgs {
	return command.RangeArgs(0, 1)
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	// parse flags first (avoid resolving service before validation)
	if err := c.parseAtTimeFlag(); err != nil {
		return err
	}
	if err := c.parseOrgIDFlag(); err != nil {
		return err
	}
	if err := c.parseShortFlag(); err != nil {
		return err
	}
	// gather assets and classify
	rawAssets, err := c.gatherRawAssets(cmd, args)
	if err != nil {
		return err
	}
	c.assets = assets.NewAssetClassifier(rawAssets...)
	c.assetType, err = c.assets.AssetType()
	if err != nil {
		return err
	}
	// check invariants - certificate asset does not support at-time
	if c.assetType == assets.AssetTypeCertificate && c.atTime.IsPresent() {
		return NewAtTimeNotSupportedError(c.assetType)
	}
	// resolve dependencies only after validation
	return c.resolveViewService()
}

// resolveViewService initializes the view service from the command context.
func (c *Command) resolveViewService() cenclierrors.CencliError {
	svc, err := c.ViewService()
	if err != nil {
		return err
	}
	c.viewSvc = svc
	return nil
}

// parseAtTimeFlag parses the optional at-time flag into c.atTime.
func (c *Command) parseAtTimeFlag() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.atTime, err = c.flags.atTime.Value(c.Config().DefaultTZ)
	if err != nil {
		return err
	}
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

// parseShortFlag parses the short flag into c.short.
func (c *Command) parseShortFlag() cenclierrors.CencliError {
	var err cenclierrors.CencliError
	c.short, err = c.flags.short.Value()
	if err != nil {
		return err
	}
	return nil
}

// gatherRawAssets returns raw asset strings from file, stdin, or positional args.
func (c *Command) gatherRawAssets(cmd *cobra.Command, args []string) ([]string, cenclierrors.CencliError) {
	if c.flags.inputFile.IsSet() {
		lines, err := c.flags.inputFile.Lines(cmd)
		if err != nil {
			return nil, err
		}
		return lines, nil
	}
	if len(args) == 0 {
		return nil, assets.NewNoAssetsError()
	}
	parts := input.SplitString(args[0])
	return parts, nil
}

func (c *Command) Run(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	count := c.assetInputCount()
	logger := c.Logger(cmdName).With(
		"assetType", string(c.assetType),
		"orgID_set", c.orgID.IsPresent(),
		"count", count,
	)

	var result assetResult
	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Fetching assets...",
		func(ctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			result, fetchErr = c.fetchAssetResult(ctx)
			return fetchErr
		},
	)
	if err != nil {
		logger.Debug("fetch failed", "error", err)
		return err
	}

	// Render the assets (even if partial)
	if renderErr := c.renderAssets(result); renderErr != nil {
		return renderErr
	}

	// If there was a partial error, print it to stderr after rendering the data
	if result.PartialError != nil {
		formatter.PrintError(result.PartialError, cmd)
	}

	return nil
}

// assetResult is a tagged union that carries meta and the concrete asset list.
// It keeps render logic simple without spreading type switches across the call sites.
type assetResult struct {
	Type          assets.AssetType
	Meta          *responsemeta.ResponseMeta
	Hosts         []*assets.Host
	Certificates  []*assets.Certificate
	WebProperties []*assets.WebProperty
	// PartialError contains any error encountered after the first successful request.
	// When present, the result contains partial data and the error should be reported to the user.
	PartialError cenclierrors.CencliError
}

func (r assetResult) Data() any {
	switch r.Type {
	case assets.AssetTypeHost:
		return r.Hosts
	case assets.AssetTypeCertificate:
		return r.Certificates
	case assets.AssetTypeWebProperty:
		return r.WebProperties
	default:
		return nil
	}
}

// renderAssets prints response metadata and either a short summary or structured data.
func (c *Command) renderAssets(res assetResult) cenclierrors.CencliError {
	c.PrintAppResponseMeta(res.Meta)
	if c.short {
		return c.printShort(res)
	}
	return c.PrintData(res.Data())
}

// assetInputCount returns the number of input assets based on the inferred asset type.
func (c *Command) assetInputCount() int {
	switch c.assetType {
	case assets.AssetTypeHost:
		return len(c.assets.HostIDs())
	case assets.AssetTypeCertificate:
		return len(c.assets.CertificateIDs())
	case assets.AssetTypeWebProperty:
		return len(c.assets.WebPropertyIDs())
	default:
		return 0
	}
}

// fetchAssetResult delegates to the appropriate view service method based on asset type.
func (c *Command) fetchAssetResult(ctx context.Context) (assetResult, cenclierrors.CencliError) {
	switch c.assetType {
	case assets.AssetTypeHost:
		result, err := c.viewSvc.GetHosts(ctx, c.orgID, c.assets.HostIDs(), c.atTime)
		if err != nil {
			return assetResult{}, err
		}
		return assetResult{
			Type:         assets.AssetTypeHost,
			Meta:         result.Meta,
			Hosts:        result.Hosts,
			PartialError: result.PartialError,
		}, nil
	case assets.AssetTypeCertificate:
		result, err := c.viewSvc.GetCertificates(ctx, c.orgID, c.assets.CertificateIDs())
		if err != nil {
			return assetResult{}, err
		}
		return assetResult{
			Type:         assets.AssetTypeCertificate,
			Meta:         result.Meta,
			Certificates: result.Certificates,
			PartialError: result.PartialError,
		}, nil
	case assets.AssetTypeWebProperty:
		result, err := c.viewSvc.GetWebProperties(ctx, c.orgID, c.assets.WebPropertyIDs(), c.atTime)
		if err != nil {
			return assetResult{}, err
		}
		return assetResult{
			Type:          assets.AssetTypeWebProperty,
			Meta:          result.Meta,
			WebProperties: result.WebProperties,
			PartialError:  result.PartialError,
		}, nil
	default:
		return assetResult{}, NewUnsupportedAssetTypeError(c.assetType, "no way to fetch this asset's data")
	}
}

func (c *Command) printShort(res assetResult) cenclierrors.CencliError {
	templateEntity, err := templateEntityFromAssetType(res.Type)
	if err != nil {
		return err
	}
	return c.PrintDataWithTemplate(templateEntity, res.Data())
}

func templateEntityFromAssetType(assetType assets.AssetType) (config.TemplateEntity, cenclierrors.CencliError) {
	switch assetType {
	case assets.AssetTypeHost:
		return config.TemplateEntityHost, nil
	case assets.AssetTypeCertificate:
		return config.TemplateEntityCertificate, nil
	case assets.AssetTypeWebProperty:
		return config.TemplateEntityWebProperty, nil
	default:
		return "", NewUnsupportedAssetTypeError(assetType, "templating not supported for this asset type")
	}
}

func (*Command) Tapes(recorder *tape.Recorder) []tape.Tape {
	tallerConfig := tape.DefaultTapeConfig()
	tallerConfig.Height = 800
	return []tape.Tape{
		tape.NewTape("view",
			tape.DefaultTapeConfig(),
			recorder.Type(
				"view 8.8.8.8",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
			recorder.Type(
				"view platform.censys.io:80",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
			recorder.Type(
				"view 3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
		),
		tape.NewTape("view-short",
			tallerConfig,
			recorder.Type(
				"view --short 8.8.8.8 ",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
			recorder.Type(
				"view --short platform.censys.io:80",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
			recorder.Type(
				"view --short 3daf2843a77b6f4e6af43cd9b6f6746053b8c928e056e8a724808db8905a94cf",
				tape.WithSleepAfter(3),
				tape.WithClearAfter(),
			),
		),
	}
}
