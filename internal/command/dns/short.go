package dns

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/censys/censys-sdk-go/models/components"

	"github.com/censys/cencli/internal/app/dns"
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

// sanitizeCell removes characters from a table cell that would break the
// table's one-row-per-record layout or reach the terminal unescaped: newline,
// carriage return, and tab become a single space, and any other control
// character (e.g. ESC 0x1b, BEL 0x07, or a C1 control such as U+009B) is
// dropped.
func sanitizeCell(v string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t':
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, v)
}

// truncateRunesEnd is formatter.TruncateEnd's byte-oriented truncation done by
// rune instead, so a multi-byte character in a TXT value is never split.
func truncateRunesEnd(s string, max int) string {
	runes := []rune(s)
	if max <= 0 || len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

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

// emptyMessage is shown in place of the table for an input with no records.
const emptyMessage = "No DNS records found in this window. Widen it with --duration (e.g. -d 90d)."

// RenderShort renders the DNS lookups as styled tables (TTY-aware). One input
// prints its title, the window, and its table. Several inputs print the
// window once, then one section (title and table) per input, in input order.
func (c *Command) RenderShort() cenclierrors.CencliError {
	window := styles.NewStyle(styles.ColorGray).Render(fmt.Sprintf("Active %s → %s UTC",
		formatter.FormatShortTime(c.params.FromTime.UTC()), formatter.FormatShortTime(c.params.ToTime.UTC())))

	if len(c.inputs) == 1 {
		c.renderSection(c.results[0], window)
		return nil
	}
	fmt.Fprintf(formatter.Stdout, "\n%s\n", window)
	for _, result := range c.results {
		c.renderSection(result, "")
	}
	fmt.Fprintf(formatter.Stdout, "\n")
	return nil
}

// renderSection prints one input's title and table. A non-empty window is
// printed under the title (the one-input layout), and that layout also keeps
// the empty result to the window and the empty message.
func (c *Command) renderSection(result fetched, window string) {
	isIP := result.input.ip.IsPresent()
	rows := rowsOf(result.records)
	if len(rows) == 0 {
		if window != "" {
			fmt.Fprintf(formatter.Stdout, "\n%s\n\n%s\n", window, emptyMessage)
			return
		}
		title := styles.GlobalStyles.Signature.Bold(true).Render(fmt.Sprintf("%s (0)", c.title(result.input)))
		fmt.Fprintf(formatter.Stdout, "\n%s\n\n%s\n", title, emptyMessage)
		return
	}
	sortRows(rows, isIP, c.timeline)

	tbl := rawtable.New(
		c.columns(isIP),
		rawtable.WithHeaderStyle[recordRow](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[recordRow](!formatter.StdoutIsTTY()),
	)

	// Surface the API's total when it exceeds what was fetched (e.g. paginated
	// with --max-pages), so users know the listing is truncated.
	count := fmt.Sprintf("%d", len(rows))
	if result.total > int64(len(rows)) {
		count = fmt.Sprintf("%d of %d", len(rows), result.total)
		// Only advise --max-pages -1 when it is the reason the fetch stopped: a
		// finite --max-pages was set (an absent value means -1, already fetching
		// everything), and no later page failed (a partial error already explains
		// the truncation, printed after the table).
		if !c.Config().Quiet && c.params.MaxPages.IsPresent() && result.partialError == nil {
			records := "records"
			if len(c.inputs) > 1 {
				records = "records for " + result.input.value
			}
			formatter.Println(formatter.Stderr, fmt.Sprintf("Showing %s %s. Use --max-pages -1 to fetch all.", count, records))
		}
	}
	title := styles.GlobalStyles.Signature.Bold(true).Render(fmt.Sprintf("%s (%s)", c.title(result.input), count))
	if window != "" {
		fmt.Fprintf(formatter.Stdout, "\n%s\n%s\n\n", title, window)
		fmt.Fprint(formatter.Stdout, tbl.Render(rows))
		fmt.Fprintf(formatter.Stdout, "\n")
		return
	}
	fmt.Fprintf(formatter.Stdout, "\n%s\n\n", title)
	fmt.Fprint(formatter.Stdout, tbl.Render(rows))
}

// rowsOf converts one input's stored records to table rows.
func rowsOf(records any) []recordRow {
	switch records := records.(type) {
	case []*dns.NameRecord:
		return rowsFromNameRecords(records)
	case []*dns.NameRangeRecord:
		return rowsFromNameRanges(records)
	case []*dns.IPRecord:
		return rowsFromIPRecords(records)
	case []*dns.IPRangeRecord:
		return rowsFromIPRanges(records)
	default:
		return nil
	}
}

func (c *Command) title(input lookupInput) string {
	switch {
	case c.timeline:
		return fmt.Sprintf("DNS timeline for %s", input.value)
	case input.ip.IsPresent():
		return fmt.Sprintf("Domains resolving to %s", input.value)
	default:
		return fmt.Sprintf("DNS records for %s", input.value)
	}
}

func (c *Command) columns(isIP bool) []rawtable.Column[recordRow] {
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

	if isIP {
		domainCol := rawtable.Column[recordRow]{
			Title:  "Domain",
			String: func(r recordRow) string { return sanitizeCell(r.Domain) },
			Style:  func(s string, _ recordRow) string { return styles.NewStyle(styles.ColorOffWhite).Render(s) },
		}
		return append([]rawtable.Column[recordRow]{domainCol, typeCol}, timeCols...)
	}
	valueCol := rawtable.Column[recordRow]{
		Title:  "Value",
		String: func(r recordRow) string { return sanitizeCell(r.Value) },
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
		return truncateRunesEnd(sanitizeCell(deref(value)), maxTXTWidth)
	default:
		return ""
	}
}

func rowsFromNameRecords(records []*dns.NameRecord) []recordRow {
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

func rowsFromNameRanges(records []*dns.NameRangeRecord) []recordRow {
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

func rowsFromIPRecords(records []*dns.IPRecord) []recordRow {
	rows := make([]recordRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, recordRow{Domain: r.Domain, Type: string(r.RecordType), First: r.FirstSeen, Last: r.LastSeen})
	}
	return rows
}

func rowsFromIPRanges(records []*dns.IPRangeRecord) []recordRow {
	rows := make([]recordRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, recordRow{Domain: r.Domain, Type: string(r.RecordType), First: r.FirstSeen, Last: r.LastSeen})
	}
	return rows
}
