package scan

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/censys/censys-sdk-go/models/components"

	appscan "github.com/censys/cencli/internal/app/scan"
	"github.com/censys/cencli/internal/command"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/ui/rawtable"
)

// renderTrackedScan renders one scan as a detail view followed by its tasks
// (TTY-aware). Shared by rescan and get.
func renderTrackedScan(s *appscan.TrackedScan) cenclierrors.CencliError {
	if s == nil {
		return nil
	}

	var out strings.Builder
	out.WriteRune('\n')
	out.WriteString(styles.GlobalStyles.Signature.Render("━━━ Tracked Scan ━━━"))
	out.WriteRune('\n')
	out.WriteRune('\n')

	command.WriteDetailField(&out, "ID", s.ID)
	command.WriteDetailField(&out, "Target", targetLabel(s.Target))
	command.WriteDetailField(&out, "Created", valueOrDash(s.CreateTime))
	completed := "no"
	if s.Completed {
		completed = "yes"
	}
	command.WriteDetailField(&out, "Completed", completed)
	formatter.Println(formatter.Stdout, out.String())

	if len(s.Tasks) == 0 {
		formatter.Println(formatter.Stdout, "No tasks yet.")
		return nil
	}

	columns := []rawtable.Column[appscan.Task]{
		{
			Title:  "Task",
			String: func(t appscan.Task) string { return valueOrDash(&t.Description) },
			Style: func(s string, _ appscan.Task) string {
				return styles.NewStyle(styles.ColorOffWhite).Render(s)
			},
		},
		{
			Title:  "Status",
			String: func(t appscan.Task) string { return taskStatusLabel(t.Status) },
			Style:  func(s string, t appscan.Task) string { return styleTaskStatus(s, t.Status) },
		},
		{
			Title:  "Updated",
			String: func(t appscan.Task) string { return valueOrDash(t.UpdateTime) },
			Style: func(s string, _ appscan.Task) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
	}

	tbl := rawtable.New(
		columns,
		rawtable.WithHeaderStyle[appscan.Task](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[appscan.Task](!formatter.StdoutIsTTY()),
	)
	fmt.Fprint(formatter.Stdout, tbl.Render(s.Tasks))
	fmt.Fprintf(formatter.Stdout, "\n")
	return nil
}

// targetLabel labels any target kind; `scan get` can show scans the CLI did
// not start.
func targetLabel(t *components.TrackedScanScanTarget) string {
	switch {
	case t == nil:
		return "-"
	case t.WebOrigin != nil:
		return hostPort(t.WebOrigin.Hostname, t.WebOrigin.Port) + " (web_origin)"
	case t.ServiceID != nil:
		label := hostPort(t.ServiceID.IP, t.ServiceID.Port)
		if t.ServiceID.Protocol != nil {
			label += "/" + *t.ServiceID.Protocol
		}
		return label + " (service_id)"
	case t.HostPort != nil:
		return hostPort(t.HostPort.IP, t.HostPort.Port) + " (host_port)"
	case t.HostnamePort != nil:
		return hostPort(t.HostnamePort.Hostname, t.HostnamePort.Port) + " (hostname_port)"
	default:
		return "-"
	}
}

func hostPort(host *string, port *int) string {
	h := valueOrDash(host)
	if port == nil {
		return h
	}
	return net.JoinHostPort(h, strconv.Itoa(*port))
}

func valueOrDash(v *string) string {
	if v == nil || *v == "" {
		return "-"
	}
	return *v
}

// taskStatusLabel names the API's empty status rather than printing nothing.
func taskStatusLabel(status string) string {
	if status == "" {
		return "pending"
	}
	return status
}

// styleTaskStatus colors a task status by outcome: produced results, failed to,
// or still going.
func styleTaskStatus(s, status string) string {
	switch status {
	case appscan.TaskStatusScanned, appscan.TaskStatusCompleted:
		return styles.NewStyle(styles.ColorSage).Render(s)
	case appscan.TaskStatusRejected, appscan.TaskStatusTimedOut:
		return styles.NewStyle(styles.ColorRed).Render(s)
	case appscan.TaskStatusIgnored:
		return styles.GlobalStyles.Warning.Render(s)
	default:
		return styles.NewStyle(styles.ColorTeal).Render(s)
	}
}
