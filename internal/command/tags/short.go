package tags

import (
	"fmt"

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
