package tags

import (
	"fmt"
	"strings"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/ui/rawtable"
)

// RenderShort renders the tag list as a styled table (TTY-aware).
func (c *ListCommand) RenderShort() cenclierrors.CencliError {
	if len(c.result.Tags) == 0 {
		fmt.Fprintf(formatter.Stdout, "\nNo tags found.\n")
		return nil
	}

	columns := []rawtable.Column[tags.Tag]{
		{
			Title:  "Name",
			String: func(t tags.Tag) string { return t.Name },
			Style: func(s string, _ tags.Tag) string {
				return styles.NewStyle(styles.ColorTeal).Render(s)
			},
		},
		{
			Title:  "Privacy",
			String: func(t tags.Tag) string { return t.Privacy },
			Style: func(s string, _ tags.Tag) string {
				return styles.NewStyle(styles.ColorSage).Render(s)
			},
		},
		{
			Title: "Description",
			String: func(t tags.Tag) string {
				if t.Description != nil && *t.Description != "" {
					return *t.Description
				}
				return "-"
			},
			Style: func(s string, _ tags.Tag) string {
				return styles.NewStyle(styles.ColorOffWhite).Render(s)
			},
		},
		{
			Title:  "Created By",
			String: func(t tags.Tag) string { return t.CreatedBy },
			Style: func(s string, _ tags.Tag) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
		{
			Title:  "Created At",
			String: func(t tags.Tag) string { return t.CreatedAt.Format("2006-01-02 15:04") },
			Style: func(s string, _ tags.Tag) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
	}

	tbl := rawtable.New(
		columns,
		rawtable.WithHeaderStyle[tags.Tag](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[tags.Tag](!formatter.StdoutIsTTY()),
	)

	// Surface the API's total when it exceeds what was fetched (e.g. paginated
	// with --max-pages), so users know the listing is truncated.
	countText := fmt.Sprintf("Tags (%d)", len(c.result.Tags))
	if c.result.TotalSize > int64(len(c.result.Tags)) {
		countText = fmt.Sprintf("Tags (%d of %d)", len(c.result.Tags), c.result.TotalSize)
	}
	title := styles.GlobalStyles.Signature.Bold(true).Render(countText)
	fmt.Fprintf(formatter.Stdout, "\n%s\n\n", title)
	fmt.Fprint(formatter.Stdout, tbl.Render(c.result.Tags))
	fmt.Fprintf(formatter.Stdout, "\n")

	return nil
}

// RenderShort renders the per-asset assignment outcomes as a styled table
// (TTY-aware).
func (c *AssignCommand) RenderShort() cenclierrors.CencliError {
	views := c.assignmentViews()
	if len(views) == 0 {
		fmt.Fprintf(formatter.Stdout, "\nNo assets assigned.\n")
		return nil
	}

	columns := []rawtable.Column[assignedAsset]{
		{
			Title:  "Asset",
			String: func(a assignedAsset) string { return a.Asset },
			Style: func(s string, _ assignedAsset) string {
				return styles.NewStyle(styles.ColorTeal).Render(s)
			},
		},
		{
			Title: "Type",
			String: func(a assignedAsset) string {
				if a.AssetType == "" {
					return "-"
				}
				return a.AssetType
			},
			Style: func(s string, _ assignedAsset) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
		{
			Title: "Status",
			String: func(a assignedAsset) string {
				if a.Assigned {
					return "assigned"
				}
				if a.Error != "" {
					return "failed: " + a.Error
				}
				return "failed"
			},
			Style: func(s string, a assignedAsset) string {
				if a.Assigned {
					return styles.NewStyle(styles.ColorSage).Render(s)
				}
				return styles.NewStyle(styles.ColorRed).Render(s)
			},
		},
	}

	tbl := rawtable.New(
		columns,
		rawtable.WithHeaderStyle[assignedAsset](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[assignedAsset](!formatter.StdoutIsTTY()),
	)

	header := fmt.Sprintf("Assigned tag %q to %d of %d asset(s)",
		c.result.TagID, len(c.result.Assignments), len(views))
	title := styles.GlobalStyles.Signature.Bold(true).Render(header)
	fmt.Fprintf(formatter.Stdout, "\n%s\n\n", title)
	fmt.Fprint(formatter.Stdout, tbl.Render(views))
	fmt.Fprintf(formatter.Stdout, "\n")

	return nil
}

// renderTagDetail renders a single tag as a labeled detail view (TTY-aware),
// under the given section header. Shared by the get and create commands.
func renderTagDetail(header string, t tags.Tag) cenclierrors.CencliError {
	var out strings.Builder
	out.WriteRune('\n')
	out.WriteString(styles.GlobalStyles.Signature.Render(header))
	out.WriteRune('\n')
	out.WriteRune('\n')

	writeField(&out, "Name", t.Name)
	writeField(&out, "ID", t.ID)
	writeField(&out, "Privacy", t.Privacy)

	description := "-"
	if t.Description != nil && *t.Description != "" {
		description = *t.Description
	}
	writeField(&out, "Description", description)

	writeField(&out, "Created By", t.CreatedBy)
	writeField(&out, "Created At", t.CreatedAt.Format("2006-01-02 15:04:05 MST"))
	writeField(&out, "Updated At", t.UpdatedAt.Format("2006-01-02 15:04:05 MST"))

	formatter.Println(formatter.Stdout, out.String())
	return nil
}

// writeField appends a padded label / value line to a detail view.
func writeField(out *strings.Builder, label, value string) {
	labelStyled := styles.GlobalStyles.Primary.Render(fmt.Sprintf("%-13s", label+":"))
	valueStyled := styles.GlobalStyles.Comment.Render(value)
	fmt.Fprintf(out, "  %s %s\n", labelStyled, valueStyled)
}
