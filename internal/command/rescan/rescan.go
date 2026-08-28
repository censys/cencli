package rescan

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
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
	transportProtocol flags.StringFlag
}

var _ command.Command = (*Command)(nil)

func NewRescanCommand(cmdContext *command.Context) *Command {
	return &Command{
		BaseCommand: command.NewBaseCommand(cmdContext),
	}
}

func (c *Command) Use() string   { return fmt.Sprintf("%s <url>", cmdName) }
func (c *Command) Short() string { return "Initiate a live rescan and track its progress" }
func (c *Command) Long() string {
	return `Initiate a live rescan of a known host service or web property and track the
scan's progress until it completes.

The target is specified as a URL. The scheme is used as the application protocol
for host services. For HTTPS services, Censys models the protocol as HTTP (with a
TLS object), so use http:// when targeting those.

If the host is an IP address, the target is treated as a host service. If it is a
hostname, it is treated as a web origin.

This costs 10 credits per rescan and is available to Enterprise customers.`
}

func (c *Command) Args() command.PositionalArgs { return command.ExactArgs(1) }

func (c *Command) DefaultOutputType() command.OutputType { return command.OutputTypeData }
func (c *Command) SupportedOutputTypes() []command.OutputType {
	return []command.OutputType{command.OutputTypeData}
}
func (c *Command) SupportsStreaming() bool { return false }

func (c *Command) Examples() []string {
	return []string{
		`http://1.1.1.1:80`,
		`rtsp://203.0.113.5:554`,
		`http://example.com:8088/`,
		`ssh://192.0.2.10:22 --transport-protocol tcp`,
	}
}

func (c *Command) Init() error {
	c.cmdFlags.orgID = flags.NewOrgIDFlag(c.Flags(), "")
	c.cmdFlags.transportProtocol = flags.NewStringFlag(
		c.Flags(), false,
		"transport-protocol", "",
		"tcp",
		fmt.Sprintf("transport protocol for host service targets (%s)", strings.Join(validTransportProtocols(), ", ")),
	)
	return nil
}

func (c *Command) PreRun(cmd *cobra.Command, args []string) cenclierrors.CencliError {
	if err := c.parseOrgIDFlag(cmd.Context()); err != nil {
		return err
	}
	if err := c.parseURLArg(args[0]); err != nil {
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

// schemeDefaults maps URL schemes to their Censys protocol name, default port,
// and default transport protocol. Schemes that are TLS variants of a base
// protocol (e.g. https, smtps) map to the base protocol name since Censys
// models TLS as a property of the service rather than a separate protocol.
type schemeDefaults struct {
	protocol  string
	port      int
	transport components.TargetTransportProtocol
}

var knownSchemes = map[string]schemeDefaults{
	"http":       {"HTTP", 80, components.TargetTransportProtocolTCP},
	"https":      {"HTTP", 443, components.TargetTransportProtocolTCP},
	"ssh":        {"SSH", 22, components.TargetTransportProtocolTCP},
	"ftp":        {"FTP", 21, components.TargetTransportProtocolTCP},
	"ftps":       {"FTP", 990, components.TargetTransportProtocolTCP},
	"smtp":       {"SMTP", 25, components.TargetTransportProtocolTCP},
	"smtps":      {"SMTP", 465, components.TargetTransportProtocolTCP},
	"imap":       {"IMAP", 143, components.TargetTransportProtocolTCP},
	"imaps":      {"IMAP", 993, components.TargetTransportProtocolTCP},
	"pop3":       {"POP3", 110, components.TargetTransportProtocolTCP},
	"pop3s":      {"POP3", 995, components.TargetTransportProtocolTCP},
	"telnet":     {"TELNET", 23, components.TargetTransportProtocolTCP},
	"rdp":        {"RDP", 3389, components.TargetTransportProtocolTCP},
	"vnc":        {"VNC", 5900, components.TargetTransportProtocolTCP},
	"rtsp":       {"RTSP", 554, components.TargetTransportProtocolTCP},
	"ldap":       {"LDAP", 389, components.TargetTransportProtocolTCP},
	"ldaps":      {"LDAP", 636, components.TargetTransportProtocolTCP},
	"dns":        {"DNS", 53, components.TargetTransportProtocolUDP},
	"mysql":      {"MYSQL", 3306, components.TargetTransportProtocolTCP},
	"redis":      {"REDIS", 6379, components.TargetTransportProtocolTCP},
	"mqtt":       {"MQTT", 1883, components.TargetTransportProtocolTCP},
}

func (c *Command) parseURLArg(raw string) cenclierrors.CencliError {
	// Ensure we have a scheme so url.Parse works correctly.
	if !strings.Contains(raw, "://") {
		return cenclierrors.NewCencliError(fmt.Errorf("invalid URL %q: must include a scheme (e.g. http://)", raw))
	}

	u, err := url.Parse(raw)
	if err != nil {
		return cenclierrors.NewCencliError(fmt.Errorf("invalid URL %q: %w", raw, err))
	}

	if u.Scheme == "" {
		return cenclierrors.NewCencliError(fmt.Errorf("invalid URL %q: missing scheme", raw))
	}

	host := u.Hostname()
	if host == "" {
		return cenclierrors.NewCencliError(fmt.Errorf("invalid URL %q: missing host", raw))
	}

	defaults := knownSchemes[strings.ToLower(u.Scheme)]

	// Resolve port: explicit in URL wins, then scheme default, then error.
	port := defaults.port
	if portStr := u.Port(); portStr != "" {
		p, perr := strconv.Atoi(portStr)
		if perr != nil || p < 1 || p > 65535 {
			return cenclierrors.NewCencliError(fmt.Errorf("invalid URL %q: port %q is not a valid port number", raw, portStr))
		}
		port = p
	}
	if port == 0 {
		return cenclierrors.NewCencliError(fmt.Errorf("invalid URL %q: port is required for scheme %q (e.g. %s://host:PORT/)", raw, u.Scheme, u.Scheme))
	}

	c.params.OrgID = c.orgID
	c.params.Port = port

	if net.ParseIP(host) != nil {
		// IP address → host service target
		protocol := defaults.protocol
		if protocol == "" {
			protocol = strings.ToUpper(u.Scheme)
		}

		// Start from the scheme-derived default; honour --transport-protocol when
		// explicitly set so the user can override (e.g. force UDP on a custom port).
		transport := defaults.transport
		if transport == "" {
			transport = components.TargetTransportProtocolTCP
		}
		if c.Flags().Changed("transport-protocol") {
			transportStr, ferr := c.cmdFlags.transportProtocol.Value()
			if ferr != nil {
				return ferr
			}
			overrideTransport, terr := parseTransportProtocol(transportStr)
			if terr != nil {
				return terr
			}
			transport = overrideTransport
		}

		c.params.TargetType = rescan.TargetTypeService
		c.params.IP = host
		c.params.Protocol = protocol
		c.params.TransportProtocol = transport
	} else {
		// Hostname → web origin target (scheme and transport not used by API)
		c.params.TargetType = rescan.TargetTypeWebOrigin
		c.params.Hostname = host
	}

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
