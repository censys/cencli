package censeye

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/censys/cencli/internal/app/censeye"
	"github.com/censys/cencli/internal/pkg/browser"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/term"
	"github.com/censys/cencli/internal/pkg/ui/rawtable"
	"github.com/censys/cencli/internal/pkg/ui/table"
)

// showInteractiveTable displays an interactive table where users can navigate with arrow keys
// and open queries in their browser by pressing Enter.
func (c *Command) showInteractiveTable(result censeye.InvestigateHostResult) cenclierrors.CencliError {
	if len(result.Entries) == 0 {
		fmt.Fprintf(formatter.Stdout, "\nNo results found.\n")
		return nil
	}

	// Sort entries in ascending order by count for interactive display
	entries := make([]censeye.ReportEntry, len(result.Entries))
	copy(entries, result.Entries)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].Query < entries[j].Query
		}
		return entries[i].Count < entries[j].Count
	})

	tbl := table.NewTable[censeye.ReportEntry](
		[]string{"Count", "!", "Query"},
		func(entry censeye.ReportEntry) []string {
			count := strconv.FormatInt(entry.Count, 10)
			indicator := " "
			if entry.Interesting {
				indicator = "*"
			}
			return []string{count, indicator, entry.Query}
		},
		table.WithColumnWidths[censeye.ReportEntry]([]int{15, 3, 80}),
		table.WithTitle[censeye.ReportEntry](fmt.Sprintf("CensEye Results for %s", c.hostID)),
		table.WithSelectFunc[censeye.ReportEntry](func(entry censeye.ReportEntry) {
			if entry.SearchURL != "" {
				_ = browser.Open(entry.SearchURL)
			}
		}),
		table.WithSelectDescription[censeye.ReportEntry]("open query in browser"),
		table.WithKeepOpenOnSelect[censeye.ReportEntry](true),
	)

	if err := tbl.Run(entries); err != nil {
		return cenclierrors.NewCencliError(
			fmt.Errorf("failed to display interactive table: %w", err),
		)
	}

	return nil
}

// showRawTable renders a non-interactive table with all results, followed by a pivots section
// and a summary line showing how many queries fell within the rarity bounds.
func (c *Command) showRawTable(result censeye.InvestigateHostResult) cenclierrors.CencliError {
	output := renderTableOutput(c.hostID, result.Entries)
	fmt.Fprint(formatter.Stdout, output)
	// render pivots output
	pivotsOutput := renderPivots(result.Entries)
	fmt.Fprint(formatter.Stdout, pivotsOutput)
	// summary line
	var interesting int
	for _, e := range result.Entries {
		if e.Interesting {
			interesting++
		}
	}
	fmt.Fprintf(formatter.Stdout, "Found %d interesting of %d within [%d,%d].\n",
		interesting, len(result.Entries), c.rarityMin, c.rarityMax)
	return nil
}

// renderTableOutput renders the results as a styled table with clickable links.
func renderTableOutput(hostID string, entries []censeye.ReportEntry) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n=== CensEye Results for %s ===\n\n", hostID))

	if len(entries) == 0 {
		sb.WriteString("No rules found.\n")
		return sb.String()
	}

	// Create table columns
	columns := []rawtable.Column[censeye.ReportEntry]{
		{
			Title: "Count",
			String: func(e censeye.ReportEntry) string {
				return fmt.Sprintf("%d", e.Count)
			},
			Style: func(s string, e censeye.ReportEntry) string {
				return styles.NewStyle(styles.ColorOffWhite).Render(s)
			},
			AlignRight: true,
		},
		{
			Title: "Query",
			String: func(e censeye.ReportEntry) string {
				val := e.Query
				if e.Interesting {
					val += " *"
				}
				return val
			},
			Style: func(s string, e censeye.ReportEntry) string {
				val := renderLink(e.Query, e.SearchURL)
				if e.Interesting {
					val = "* " + val
					return styles.NewStyle(styles.ColorOrange).Render(val)
				}
				return styles.NewStyle(styles.ColorTeal).Render(val)
			},
		},
	}

	table := rawtable.New(
		columns,
		rawtable.WithHeaderStyle[censeye.ReportEntry](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[censeye.ReportEntry](!formatter.StdoutIsTTY()),
	)

	sb.WriteString(table.Render(entries))
	sb.WriteString("\n")

	return sb.String()
}

// renderPivots renders the list of interesting queries as pivots.
func renderPivots(entries []censeye.ReportEntry) string {
	var sb strings.Builder

	var interesting []censeye.ReportEntry
	for _, entry := range entries {
		if entry.Interesting {
			interesting = append(interesting, entry)
		}
	}

	if len(interesting) == 0 {
		return ""
	}

	sb.WriteString("Pivots (interesting queries):\n")
	for _, entry := range interesting {
		link := renderLink(entry.Query, entry.SearchURL)
		sb.WriteString(fmt.Sprintf("  - %s\n", link))
	}
	sb.WriteString("\n")

	return sb.String()
}

// renderLink renders a link with the given text and url.
// if stdout is not a TTY, it returns the text without the link.
func renderLink(text, url string) string {
	if formatter.StdoutIsTTY() {
		return term.RenderLink(url, text)
	}
	return text
}
