package tags

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/censys/cencli/internal/app/tags"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/censys/cencli/internal/pkg/ui/rawtable"
)

// detailTimeLayout is the timestamp format the detail views and confirmation
// prompts share. It keeps the zone, so a value the user gave in local time is
// never shown back to them as another.
const detailTimeLayout = "2006-01-02 15:04:05 MST"

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

// RenderShort renders a tag's assignments as a styled table (TTY-aware).
func (c *AssignmentsCommand) RenderShort() cenclierrors.CencliError {
	if len(c.result.Assignments) == 0 {
		fmt.Fprintf(formatter.Stdout, "\nNo assignments found.\n")
		return nil
	}

	columns := []rawtable.Column[tags.Assignment]{
		{
			Title:  "Asset",
			String: func(a tags.Assignment) string { return a.AssetID },
			Style: func(s string, _ tags.Assignment) string {
				return styles.NewStyle(styles.ColorTeal).Render(s)
			},
		},
		{
			Title:  "Type",
			String: func(a tags.Assignment) string { return a.AssetType },
			Style: func(s string, _ tags.Assignment) string {
				return styles.NewStyle(styles.ColorSage).Render(s)
			},
		},
		{
			Title:  "Created By",
			String: func(a tags.Assignment) string { return a.CreatedBy },
			Style: func(s string, _ tags.Assignment) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
		{
			Title:  "Created At",
			String: func(a tags.Assignment) string { return a.CreatedAt.Format("2006-01-02 15:04") },
			Style: func(s string, _ tags.Assignment) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
	}

	tbl := rawtable.New(
		columns,
		rawtable.WithHeaderStyle[tags.Assignment](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[tags.Assignment](!formatter.StdoutIsTTY()),
	)

	// Show the API's total when it exceeds what was fetched, so a truncated
	// listing says so.
	countText := fmt.Sprintf("Assignments (%d)", len(c.result.Assignments))
	if c.result.TotalSize > int64(len(c.result.Assignments)) {
		countText = fmt.Sprintf("Assignments (%d of %d)", len(c.result.Assignments), c.result.TotalSize)
	}
	title := styles.GlobalStyles.Signature.Bold(true).Render(countText)
	fmt.Fprintf(formatter.Stdout, "\n%s\n\n", title)
	fmt.Fprint(formatter.Stdout, tbl.Render(c.result.Assignments))
	fmt.Fprintf(formatter.Stdout, "\n")

	return nil
}

// failureReason renders a per-asset error for a table cell, or "-" for a row
// that succeeded. The status is worth the characters: it distinguishes an asset
// that already carries the tag (409) from one the caller cannot touch (403).
func failureReason(detail string, status *int64) string {
	if detail == "" {
		return "-"
	}
	if status == nil {
		return detail
	}
	return fmt.Sprintf("%s (%d)", detail, *status)
}

// RenderShort renders the per-asset assignment outcomes as a styled table
// (TTY-aware), or the tracking operation when the assignment was a bulk job.
func (c *AssignCommand) RenderShort() cenclierrors.CencliError {
	if c.bulk {
		return renderOperationDetail(c.operation)
	}

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
				return "failed"
			},
			Style: func(s string, a assignedAsset) string {
				if a.Assigned {
					return styles.NewStyle(styles.ColorSage).Render(s)
				}
				return styles.NewStyle(styles.ColorRed).Render(s)
			},
		},
		{
			// Its own column so a long message cannot stretch Status.
			Title:  "Error",
			String: func(a assignedAsset) string { return failureReason(a.Error, a.ErrorStatus) },
			Style: func(s string, _ assignedAsset) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
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

// RenderShort renders the per-asset unassignment outcomes as a styled table
// (TTY-aware), or the tracking operation when the removal was a bulk job.
func (c *UnassignCommand) RenderShort() cenclierrors.CencliError {
	if c.bulk {
		return renderOperationDetail(c.operation)
	}

	views := c.unassignmentViews()
	if len(views) == 0 {
		fmt.Fprintf(formatter.Stdout, "\nNo assets unassigned.\n")
		return nil
	}

	columns := []rawtable.Column[unassignedAsset]{
		{
			Title:  "Asset",
			String: func(a unassignedAsset) string { return a.Asset },
			Style: func(s string, _ unassignedAsset) string {
				return styles.NewStyle(styles.ColorTeal).Render(s)
			},
		},
		{
			Title: "Type",
			String: func(a unassignedAsset) string {
				if a.AssetType == "" {
					return "-"
				}
				return a.AssetType
			},
			Style: func(s string, _ unassignedAsset) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
		{
			Title: "Status",
			String: func(a unassignedAsset) string {
				if a.Unassigned {
					return "unassigned"
				}
				return "failed"
			},
			Style: func(s string, a unassignedAsset) string {
				if a.Unassigned {
					return styles.NewStyle(styles.ColorSage).Render(s)
				}
				return styles.NewStyle(styles.ColorRed).Render(s)
			},
		},
		{
			// Its own column so a long message cannot stretch Status.
			Title:  "Error",
			String: func(a unassignedAsset) string { return failureReason(a.Error, a.ErrorStatus) },
			Style: func(s string, _ unassignedAsset) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
	}

	tbl := rawtable.New(
		columns,
		rawtable.WithHeaderStyle[unassignedAsset](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[unassignedAsset](!formatter.StdoutIsTTY()),
	)

	header := fmt.Sprintf("Unassigned tag %q from %d of %d asset(s)",
		c.result.TagID, len(c.result.Unassigned), len(views))
	title := styles.GlobalStyles.Signature.Bold(true).Render(header)
	fmt.Fprintf(formatter.Stdout, "\n%s\n\n", title)
	fmt.Fprint(formatter.Stdout, tbl.Render(views))
	fmt.Fprintf(formatter.Stdout, "\n")

	return nil
}

// RenderShort renders the bulk tag operations as a styled table (TTY-aware).
func (c *OperationsListCommand) RenderShort() cenclierrors.CencliError {
	if len(c.result.Operations) == 0 {
		fmt.Fprintf(formatter.Stdout, "\nNo operations found.\n")
		return nil
	}

	columns := []rawtable.Column[tags.TagOperation]{
		{
			Title:  "ID",
			String: func(o tags.TagOperation) string { return o.ID },
			Style: func(s string, _ tags.TagOperation) string {
				return styles.NewStyle(styles.ColorTeal).Render(s)
			},
		},
		{
			Title:  "Tag",
			String: func(o tags.TagOperation) string { return o.TagName },
			Style: func(s string, _ tags.TagOperation) string {
				return styles.NewStyle(styles.ColorOffWhite).Render(s)
			},
		},
		{
			Title:  "Type",
			String: func(o tags.TagOperation) string { return o.Type },
			Style: func(s string, _ tags.TagOperation) string {
				return styles.NewStyle(styles.ColorSage).Render(s)
			},
		},
		{
			Title:  "Status",
			String: func(o tags.TagOperation) string { return o.Status },
			Style:  func(s string, o tags.TagOperation) string { return styleOperationStatus(s, o.Status) },
		},
		{
			Title:  "Progress",
			String: operationProgress,
			Style: func(s string, _ tags.TagOperation) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
		{
			Title:  "Created At",
			String: func(o tags.TagOperation) string { return o.CreatedAt.Format("2006-01-02 15:04") },
			Style: func(s string, _ tags.TagOperation) string {
				return styles.NewStyle(styles.ColorGray).Render(s)
			},
		},
	}

	tbl := rawtable.New(
		columns,
		rawtable.WithHeaderStyle[tags.TagOperation](styles.NewStyle(styles.ColorOffWhite).Bold(true)),
		rawtable.WithStylesDisabled[tags.TagOperation](!formatter.StdoutIsTTY()),
	)

	// Show the API's total when it exceeds what was fetched, so a truncated
	// listing says so.
	countText := fmt.Sprintf("Operations (%d)", len(c.result.Operations))
	if c.result.TotalSize > int64(len(c.result.Operations)) {
		countText = fmt.Sprintf("Operations (%d of %d)", len(c.result.Operations), c.result.TotalSize)
	}
	title := styles.GlobalStyles.Signature.Bold(true).Render(countText)
	fmt.Fprintf(formatter.Stdout, "\n%s\n\n", title)
	fmt.Fprint(formatter.Stdout, tbl.Render(c.result.Operations))
	fmt.Fprintf(formatter.Stdout, "\n")

	return nil
}

// RenderShort renders a single operation as a labeled detail view (TTY-aware).
func (c *OperationsGetCommand) RenderShort() cenclierrors.CencliError {
	return renderOperationDetail(c.result.Operation)
}

// renderOperationDetail renders one bulk operation as a labeled detail view
// (TTY-aware). Shared by `operations get` and the bulk assign submit, so a job
// reads the same however you arrived at it.
func renderOperationDetail(op tags.TagOperation) cenclierrors.CencliError {
	var out strings.Builder
	out.WriteRune('\n')
	out.WriteString(styles.GlobalStyles.Signature.Render("━━━ Tag Operation ━━━"))
	out.WriteRune('\n')
	out.WriteRune('\n')

	writeField(&out, "ID", op.ID)
	writeField(&out, "Tag", op.TagName)
	writeField(&out, "Tag ID", op.TagID)
	writeField(&out, "Type", op.Type)
	writeField(&out, "Status", styleOperationStatus(op.Status, op.Status))
	writeField(&out, "Progress", operationProgress(op))
	writeField(&out, "Succeeded", strconv.FormatInt(op.SuccessfulCount, 10))

	// Only bulk_create operations carry the query that produced them.
	if op.Query != nil && *op.Query != "" {
		writeField(&out, "Query", *op.Query)
	}

	writeField(&out, "Created At", op.CreatedAt.Format(detailTimeLayout))
	if op.EndedAt != nil {
		writeField(&out, "Ended At", op.EndedAt.Format(detailTimeLayout))
	}
	if op.StatusMessage != nil && *op.StatusMessage != "" {
		writeField(&out, "Message", *op.StatusMessage)
	}
	if op.ErrorMessage != nil && *op.ErrorMessage != "" {
		writeField(&out, "Error", *op.ErrorMessage)
	}

	formatter.Println(formatter.Stdout, out.String())
	return nil
}

// operationProgress renders how far an operation got. The total is unknown until
// completion for bulk_delete, so it is only shown once the API reports one.
func operationProgress(o tags.TagOperation) string {
	if o.TotalCount > 0 {
		return fmt.Sprintf("%d/%d", o.ProcessedCount, o.TotalCount)
	}
	return strconv.FormatInt(o.ProcessedCount, 10)
}

// styleOperationStatus colors a status by outcome: done, capped, or broken.
func styleOperationStatus(s, status string) string {
	switch status {
	case statusSucceeded:
		return styles.NewStyle(styles.ColorSage).Render(s)
	case statusFailed, statusCancelled:
		return styles.NewStyle(styles.ColorRed).Render(s)
	case statusLimitReached:
		return styles.GlobalStyles.Warning.Render(s)
	default:
		return styles.NewStyle(styles.ColorTeal).Render(s)
	}
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
	writeField(&out, "Created At", t.CreatedAt.Format(detailTimeLayout))
	writeField(&out, "Updated At", t.UpdatedAt.Format(detailTimeLayout))

	// Only `get --asset-count` populates the count.
	if t.AssetCount != nil {
		writeField(&out, "Assets", strconv.FormatInt(*t.AssetCount, 10))
	}

	formatter.Println(formatter.Stdout, out.String())
	return nil
}

// writeField appends a padded label / value line to a detail view.
func writeField(out *strings.Builder, label, value string) {
	labelStyled := styles.GlobalStyles.Primary.Render(fmt.Sprintf("%-13s", label+":"))
	valueStyled := styles.GlobalStyles.Comment.Render(value)
	fmt.Fprintf(out, "  %s %s\n", labelStyled, valueStyled)
}
