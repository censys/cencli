package rescan

import (
	"context"
	"fmt"
	"strings"

	"github.com/censys/censys-sdk-go/models/components"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/censys/cencli/internal/app/rescan"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/domain/identifiers"
	"github.com/censys/cencli/internal/pkg/flags"
	"github.com/censys/cencli/internal/pkg/tape"
)

const cmdName = "rescan"

// Command implements the `rescan` subcommand.
type Command struct {
	*command.BaseCommand
	rescanSvc rescan.Service
	cmdFlags  rescanCommandFlags
	// parsed state
	orgID  mo.Option[identifiers.OrganizationID]
	params rescan.Params
	result rescan.Result
}

type rescanCommandFlags struct {
	orgID             flags.OrgIDFlag
	ip                flags.StringFlag
	hostname          flags.StringFlag
	port              flags.IntegerFlag
	protocol          flags.StringFlag
	transportProtocol flags.StringFlag
}

var _ command.Command = (*Command)(nil)

func NewRescanCommand(cmdContext *command.Context) *Command {
	return &Command{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *Command) Use() string  { return cmdName }
func (c *Command) Short() string { return "Initiate a live rescan and track its progress" }
func (c *Command) Long() string {
	return "Initiate a live rescan of a known host service or web property and track the scan's\nprogress until it completes.\n\nThis costs 10 credits per rescan and is available to Enterprise customers."
}

func (c *Command) Args() command.PositionalArgs { return command.ExactArgs(0) }

func (c *Command) DefaultOutputType() command.OutputType { return command.OutputTypeData }
func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeData}
}
func (c *Command) SupportsStreaming() bool { return false }

func (c *Command) Examples() []string {
	return []string{
		`--ip 1.2.3.4 --port 443 --protocol HTTP --transport-protocol tcp`,
		`--hostname example.com --port 443`,
	}
}

func (c *Command) Init() error {
	c.cmdFlags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.cmdFlags.ip = flags.NewStringFlag(c.Flags(), false, "ip", "", "", "IP address of the service to rescan (mutually exclusive with --hostname)")
	c.cmdFlags.hostname = flags.NewStringFlag(c.Flags(), false, "hostname", "", "", "hostname of the web origin to rescan (mutually exclusive with --ip)")
	c.cmdFlags.port = flags.NewIntegerFlag(c.Flags(), true, "port", "p", mo.None[int64](), "port number", mo.Some[int64](1), mo.Some[int64](65535))
	c.cmdFlags.protocol = flags.NewStringFlag(c.Flags(), false, "protocol", "", "", "service protocol name (e.g. HTTP, SSH) — required when using --ip")
	c.cmdFlags.transportProtocol = flags.NewStringFlag(c.Flags(), false, "transport-protocol", "", "tcp", fmt.Sprintf("transport protocol (%s)", strings.Join(validTransportProtocols(), ", ")))
	return nil
}

func (c *Command) PreRun(cmd *cobra.Command, _ []string) cenclierrors.CencliError {
	if err := c.parseOrgIDFlag(cmd.Context()); err != nil {
		return err
	}
	if err := c.parseTargetFlags(cmd); err != nil {
		return err
	}
	return c.resolveRescanService()
}

func (c *Command) Run(cmd *cobra.Command, _ []string) cenclierrors.CencliError {
	logger := c.Logger(cmdName).With(
		"orgID_set", c.orgID.IsPresent(),
		"targetType", c.params.TargetType,
	)

	err := c.WithProgress(
		cmd.Context(),
		logger,
		"Initiating rescan...",
		func(pctx context.Context) cenclierrors.CencliError {
			var fetchErr cenclierrors.CencliError
			c.result, fetchErr = c.rescanSvc.Rescan(pctx, c.params)
			return fetchErr
		},
	)
	if err != nil {
		return err
	}

	c.PrintAppResponseMeta(c.result.Meta)
	return c.PrintData(c, c.result.Scan)
}

func (c *Command) RenderShort() cenclierrors.CencliError    { return nil }
func (c *Command) RenderTemplate() cenclierrors.CencliError { return nil }

func (c *Command) parseOrgIDFlag(ctx context.Context) cenclierrors.CencliError {
	flagOrgID, err := c.cmdFlags.orgID.Value()
	if err != nil {
		return err
	}
	c.orgID, err = c.ResolveOrgID(ctx, flagOrgID)
	return err
}

func (c *Command) parseTargetFlags(cmd *cobra.Command) cenclierrors.CencliError {
	ipSet := cmd.Flags().Changed("ip")
	hostnameSet := cmd.Flags().Changed("hostname")

	if ipSet && hostnameSet {
		return flags.NewConflictingFlagsError("ip", "hostname")
	}
	if !ipSet && !hostnameSet {
		return cenclierrors.NewCencliError(fmt.Errorf("one of --ip or --hostname is required"))
	}

	portVal, err := c.cmdFlags.port.Value()
	if err != nil {
		return err
	}
	port := int(portVal.MustGet())

	c.params.OrgID = c.orgID

	if hostnameSet {
		hostname, err := c.cmdFlags.hostname.Value()
		if err != nil {
			return err
		}
		c.params.TargetType = rescan.TargetTypeWebOrigin
		c.params.Hostname = hostname
		c.params.Port = port
		return nil
	}

	// service target
	ip, err := c.cmdFlags.ip.Value()
	if err != nil {
		return err
	}
	protocol, err := c.cmdFlags.protocol.Value()
	if err != nil {
		return err
	}
	if protocol == "" {
		return cenclierrors.NewCencliError(fmt.Errorf("--protocol is required when using --ip"))
	}
	transportStr, err := c.cmdFlags.transportProtocol.Value()
	if err != nil {
		return err
	}
	transport, terr := parseTransportProtocol(transportStr)
	if terr != nil {
		return terr
	}

	c.params.TargetType = rescan.TargetTypeService
	c.params.IP = ip
	c.params.Port = port
	c.params.Protocol = protocol
	c.params.TransportProtocol = transport
	return nil
}

func (c *Command) resolveRescanService() cenclierrors.CencliError {
	svc, err := c.RescanService()
	if err != nil {
		return err
	}
	c.rescanSvc = svc
	return nil
}

func parseTransportProtocol(s string) (components.TargetTransportProtocol, cenclierrors.CencliError) {
	switch strings.ToLower(s) {
	case "tcp":
		return components.TargetTransportProtocolTCP, nil
	case "udp":
		return components.TargetTransportProtocolUDP, nil
	case "icmp":
		return components.TargetTransportProtocolIcmp, nil
	case "quic":
		return components.TargetTransportProtocolQuic, nil
	default:
		return "", cenclierrors.NewCencliError(fmt.Errorf(
			"invalid --transport-protocol %q: must be one of %s",
			s, strings.Join(validTransportProtocols(), ", "),
		))
	}
}

func validTransportProtocols() []string {
	return []string{"tcp", "udp", "icmp", "quic"}
}

func (*Command) Tapes(_ *tape.Recorder) []tape.Tape { return nil }
