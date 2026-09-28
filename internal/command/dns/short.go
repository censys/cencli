package dns

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/ui/rawtable"
)

// recordTypeOrder is the display order for name records, sourced from the SDK's
// generated enum.
var recordTypeOrder = []string{
	string(components.DNSResolutionRecordRecordTypeA),
	string(components.DNSResolutionRecordRecordTypeAaaa),
	string(components.DNSResolutionRecordRecordTypeMx),
	string(components.DNSResolutionRecordRecordTypeNs),
	string(components.DNSResolutionRecordRecordTypeSoa),
	string(components.DNSResolutionRecordRecordTypeTxt),
}

// maxTXTWidth limits TXT values in the table; data output keeps the full value.
const maxTXTWidth = 60

// recordRow is one table row in the short output, for any of the four lookups.
type recordRow struct {
	Domain string // IP lookups only
	Type   string
	Value  string // name lookups only
	First  time.Time
	Last   time.Time
	// priority is the MX priority, kept for sorting name records numerically;
	// recordValue already renders it into Value, so this is not displayed.
	priority *int64
}

// RenderShort renders the DNS lookup as a styled table (TTY-aware).
func (c *Command) RenderShort() cenclierrors.CencliError {
	window := styles.NewStyle(styles.ColorGray).Render(fmt.Sprintf("Active %s → %s UTC",
		formatter.FormatShortTime(c.params.FromTime.UTC()), formatter.FormatShortTime(c.params.ToTime.UTC())))

	rows := c.rows()
	if len(rows) == 0 {
		fmt.Fprintf(formatter.Stdout, "\n%s\n\nNo DNS records found in this window. Widen it with --duration (e.g. -d 90d).\n", window)
		return nil
	}
	sortRows(rows, c.ip.IsPresent(), c.timeline)

	tbl := rawtable.New(
		c.columns(),
		rawtable.WithHeaderStyle[recordRow](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[recordRow](!formatter.StdoutIsTTY()),
	)

	// Surface the API's total when it exceeds what was fetched (e.g. paginated
	// with --max-pages), so users know the listing is truncated.
	count := fmt.Sprintf("%d", len(rows))
	if c.result.total > int64(len(rows)) {
		count = fmt.Sprintf("%d of %d", len(rows), c.result.total)
		if !c.Config().Quiet {
			formatter.Println(formatter.Stderr, fmt.Sprintf("Showing %s records. Use --max-pages -1 to fetch all.", count))
		}
	}
	title := styles.GlobalStyles.Signature.Bold(true).Render(fmt.Sprintf("%s (%s)", c.title(), count))
	fmt.Fprintf(formatter.Stdout, "\n%s\n%s\n\n", title, window)
	fmt.Fprint(formatter.Stdout, tbl.Render(rows))
	fmt.Fprintf(formatter.Stdout, "\n")
	return nil
}

// rows converts the stored lookup result to table rows.
func (c *Command) rows() []recordRow {
	switch records := c.result.records.(type) {
	case []*components.DNSResolutionRecord:
		return rowsFromNameRecords(records)
	case []*components.DNSResolutionRangeRecord:
		return rowsFromNameRanges(records)
	case []*components.DNSIPResolutionRecord:
		return rowsFromIPRecords(records)
	case []*components.DNSIPResolutionRangeRecord:
		return rowsFromIPRanges(records)
	default:
		return nil
	}
}

func (c *Command) title() string {
	switch {
	case c.timeline:
		return fmt.Sprintf("DNS timeline for %s", c.input)
	case c.ip.IsPresent():
		return fmt.Sprintf("Domains resolving to %s", c.input)
	default:
		return fmt.Sprintf("DNS records for %s", c.input)
	}
}

func (c *Command) columns() []rawtable.Column[recordRow] {
	firstTitle, lastTitle := "First Seen", "Last Seen"
	if c.timeline {
		firstTitle, lastTitle = "First Observed", "Last Observed"
	}
	gray := func(s string, _ recordRow) string { return styles.NewStyle(styles.ColorGray).Render(s) }
	typeCol := rawtable.Column[recordRow]{
		Title:  "Type",
		String: func(r recordRow) string { return r.Type },
		Style:  func(s string, _ recordRow) string { return styles.NewStyle(styles.ColorTeal).Render(s) },
	}
	timeCols := []rawtable.Column[recordRow]{
		{Title: firstTitle, String: func(r recordRow) string { return formatter.FormatShortTime(r.First.UTC()) }, Style: gray},
		{Title: lastTitle, String: func(r recordRow) string { return formatter.FormatShortTime(r.Last.UTC()) }, Style: gray},
	}

	if c.ip.IsPresent() {
		domainCol := rawtable.Column[recordRow]{
			Title:  "Domain",
			String: func(r recordRow) string { return r.Domain },
			Style:  func(s string, _ recordRow) string { return styles.NewStyle(styles.ColorOffWhite).Render(s) },
		}
		return append([]rawtable.Column[recordRow]{domainCol, typeCol}, timeCols...)
	}
	valueCol := rawtable.Column[recordRow]{
		Title:  "Value",
		String: func(r recordRow) string { return r.Value },
		Style:  func(s string, _ recordRow) string { return styles.NewStyle(styles.ColorOffWhite).Render(s) },
	}
	return append([]rawtable.Column[recordRow]{typeCol, valueCol}, timeCols...)
}

// sortRows orders rows for display: a timeline oldest first; an IP lookup by
// last seen, newest first; a name lookup by record type, then (for MX) the
// numeric priority, then value.
func sortRows(rows []recordRow, isIP, timeline bool) {
	switch {
	case timeline:
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].First.Before(rows[j].First) })
	case isIP:
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Last.After(rows[j].Last) })
	default:
		sort.SliceStable(rows, func(i, j int) bool {
			ti, tj := recordTypeRank(rows[i].Type), recordTypeRank(rows[j].Type)
			if ti != tj {
				return ti < tj
			}
			if rows[i].Type == "MX" {
				switch pi, pj := rows[i].priority, rows[j].priority; {
				case pi != nil && pj != nil && *pi != *pj:
					return *pi < *pj
				case pi != nil && pj == nil:
					return true
				case pi == nil && pj != nil:
					return false
				}
			}
			return rows[i].Value < rows[j].Value
		})
	}
}

// recordTypeRank returns recordTypeOrder's index for a known record type. An
// unrecognized type (not yet added to recordTypeOrder) sorts after every known
// type, rather than before it as slices.Index's -1 would.
func recordTypeRank(recordType string) int {
	if i := slices.Index(recordTypeOrder, recordType); i >= 0 {
		return i
	}
	return len(recordTypeOrder)
}

// recordValue renders a name record's type-specific fields the way dig shows them.
func recordValue(recordType string, ip, mailServer, nameServer, mname, rname, value *string, priority *int64) string {
	deref := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	switch recordType {
	case "A", "AAAA":
		return deref(ip)
	case "MX":
		if priority != nil {
			return fmt.Sprintf("%d %s", *priority, deref(mailServer))
		}
		return deref(mailServer)
	case "NS":
		return deref(nameServer)
	case "SOA":
		if mname == nil || rname == nil {
			return deref(mname) + deref(rname)
		}
		return *mname + " " + *rname
	case "TXT":
		return formatter.TruncateEnd(deref(value), maxTXTWidth)
	default:
		return ""
	}
}

func rowsFromNameRecords(records []*components.DNSResolutionRecord) []recordRow {
	rows := make([]recordRow, 0, len(records))
	for _, r := range records {
		t := string(r.RecordType)
		rows = append(rows, recordRow{
			Type: t, Value: recordValue(t, r.IP, r.MailServer, r.NameServer, r.Mname, r.Rname, r.Value, r.Priority),
			First: r.FirstSeen, Last: r.LastSeen, priority: r.Priority,
		})
	}
	return rows
}

func rowsFromNameRanges(records []*components.DNSResolutionRangeRecord) []recordRow {
	rows := make([]recordRow, 0, len(records))
	for _, r := range records {
		t := string(r.RecordType)
		rows = append(rows, recordRow{
			Type: t, Value: recordValue(t, r.IP, r.MailServer, r.NameServer, r.Mname, r.Rname, r.Value, r.Priority),
			First: r.FirstObserved, Last: r.LastObserved, priority: r.Priority,
		})
	}
	return rows
}

func rowsFromIPRecords(records []*components.DNSIPResolutionRecord) []recordRow {
	rows := make([]recordRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, recordRow{Domain: r.Domain, Type: string(r.RecordType), First: r.FirstSeen, Last: r.LastSeen})
	}
	return rows
}

func rowsFromIPRanges(records []*components.DNSIPResolutionRangeRecord) []recordRow {
	rows := make([]recordRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, recordRow{Domain: r.Domain, Type: string(r.RecordType), First: r.FirstSeen, Last: r.LastSeen})
	}
	return rows
}
