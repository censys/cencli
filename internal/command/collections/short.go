package collections

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/censys/cencli/internal/app/collections"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/ui/rawtable"
)

// detailTimeLayout is the timestamp format the detail views share. It keeps the
// zone, so a value is never shown in a different zone than the one it has.
const detailTimeLayout = "2006-01-02 15:04:05 MST"

// RenderShort renders the collection list as a styled table (TTY-aware).
func (c *ListCommand) RenderShort() cenclierrors.CencliError {
	if len(c.result.Collections) == 0 {
		fmt.Fprintf(formatter.Stdout, "\nNo collections found.\n")
		return nil
	}

	columns := []rawtable.Column[collections.Collection]{
		{
			Title:  "ID",
			String: func(col collections.Collection) string { return col.ID },
			Style: func(s string, _ collections.Collection) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
		{
			Title:  "Name",
			String: func(col collections.Collection) string { return col.Name },
			Style: func(s string, _ collections.Collection) string {
				return styles.NewStyle(styles.ColorTeal).Render(s)
			},
		},
		{
			Title:  "Status",
			String: func(col collections.Collection) string { return col.Status },
			Style: func(s string, _ collections.Collection) string {
				return styles.NewStyle(styles.ColorSage).Render(s)
			},
		},
		{
			Title:  "Assets",
			String: func(col collections.Collection) string { return strconv.FormatInt(col.TotalAssets, 10) },
			Style: func(s string, _ collections.Collection) string {
				return styles.NewStyle(styles.ColorOffWhite).Render(s)
			},
		},
		{
			Title:  "+24h",
			String: func(col collections.Collection) string { return strconv.FormatInt(col.AddedAssets24Hours, 10) },
			Style: func(s string, _ collections.Collection) string {
				return styles.NewStyle(styles.ColorOffWhite).Render(s)
			},
		},
		{
			Title:  "-24h",
			String: func(col collections.Collection) string { return strconv.FormatInt(col.RemovedAssets24Hours, 10) },
			Style: func(s string, _ collections.Collection) string {
				return styles.NewStyle(styles.ColorOffWhite).Render(s)
			},
		},
		{
			Title:  "Created At",
			String: func(col collections.Collection) string { return col.CreateTime.Format("2006-01-02 15:04") },
			Style: func(s string, _ collections.Collection) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
	}

	tbl := rawtable.New(
		columns,
		rawtable.WithHeaderStyle[collections.Collection](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[collections.Collection](!formatter.StdoutIsTTY()),
	)

	// The list endpoint reports no total, so the count is what was fetched.
	title := styles.GlobalStyles.Signature.Bold(true).Render(fmt.Sprintf("Collections (%d)", len(c.result.Collections)))
	fmt.Fprintf(formatter.Stdout, "\n%s\n\n", title)
	fmt.Fprint(formatter.Stdout, tbl.Render(c.result.Collections))
	fmt.Fprintf(formatter.Stdout, "\n")

	return nil
}

// renderCollectionDetail renders a single collection as a labeled detail view (TTY-aware) under the given section header. The get, create, and update commands share it.
func renderCollectionDetail(header string, col collections.Collection) cenclierrors.CencliError {
	var out strings.Builder
	out.WriteRune('\n')
	out.WriteString(styles.GlobalStyles.Signature.Render(header))
	out.WriteRune('\n')
	out.WriteRune('\n')

	writeField(&out, "Name", col.Name)
	writeField(&out, "ID", col.ID)

	description := "-"
	if col.Description != "" {
		description = col.Description
	}
	writeField(&out, "Description", description)
	writeField(&out, "Query", col.Query)
	writeField(&out, "Status", col.Status)
	if col.StatusReason != nil && *col.StatusReason != "" {
		writeField(&out, "Reason", *col.StatusReason)
	}
	writeField(&out, "Assets", strconv.FormatInt(col.TotalAssets, 10))
	writeField(&out, "Added 24h", strconv.FormatInt(col.AddedAssets24Hours, 10))
	writeField(&out, "Removed 24h", strconv.FormatInt(col.RemovedAssets24Hours, 10))

	createdBy := "-"
	if col.CreatedBy != nil && *col.CreatedBy != "" {
		createdBy = *col.CreatedBy
	}
	writeField(&out, "Created By", createdBy)
	writeField(&out, "Created At", col.CreateTime.Format(detailTimeLayout))

	formatter.Println(formatter.Stdout, out.String())
	return nil
}

// writeField appends a padded label / value line to a detail view.
func writeField(out *strings.Builder, label, value string) {
	labelStyled := styles.GlobalStyles.Primary.Render(fmt.Sprintf("%-13s", label+":"))
	valueStyled := styles.GlobalStyles.Comment.Render(value)
	fmt.Fprintf(out, "  %s %s\n", labelStyled, valueStyled)
}
