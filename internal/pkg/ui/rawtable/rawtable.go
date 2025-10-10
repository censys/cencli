package rawtable

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Table represents a simple table renderer for struct data.
type Table[T any] struct {
	columns        []Column[T]
	headerStyle    lipgloss.Style
	stylesDisabled bool
}

// Column defines a single column.
type Column[T any] struct {
	// Title is the column header text
	Title string

	// String extracts the cell value from a row as a plain string
	String func(row T) string

	// Style applies styling to a cell value (optional, can be nil)
	Style func(rendered string, row T) string

	// AlignRight aligns the content to the right (default is left)
	AlignRight bool
}

// Option is a functional option for configuring a Table.
type Option[T any] func(*Table[T])

// WithHeaderStyle sets the style for column headers.
func WithHeaderStyle[T any](style lipgloss.Style) Option[T] {
	return func(t *Table[T]) {
		t.headerStyle = style
	}
}

// WithStylesDisabled sets whether to disable styles.
func WithStylesDisabled[T any](disabled bool) Option[T] {
	return func(t *Table[T]) {
		t.stylesDisabled = disabled
	}
}

// New creates a new Table with the given columns.
func New[T any](columns []Column[T], opts ...Option[T]) *Table[T] {
	t := &Table[T]{
		columns:        columns,
		headerStyle:    lipgloss.NewStyle(), // No style by default
		stylesDisabled: false,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Render renders the table with the given data rows and returns a string.
func (t *Table[T]) Render(rows []T) string {
	if len(rows) == 0 {
		return ""
	}

	// Extract plain string values from all cells
	headers := make([]string, len(t.columns))
	plainRows := make([][]string, len(rows))

	for i, col := range t.columns {
		headers[i] = col.Title
	}

	for r := range rows {
		plainRows[r] = make([]string, len(t.columns))
		for c, col := range t.columns {
			plainRows[r][c] = col.String(rows[r])
		}
	}

	// Calculate column widths based on plain text
	widths := t.calculateWidths(headers, plainRows)

	// Build the table
	var sb strings.Builder

	// Header row
	for i, header := range headers {
		paddedHeader := pad(header, widths[i], false) // Headers always left-aligned
		if !t.stylesDisabled {
			paddedHeader = t.headerStyle.Render(paddedHeader)
		}
		sb.WriteString(paddedHeader)
		if i < len(headers)-1 {
			sb.WriteString("   ")
		}
	}
	sb.WriteString("\n\n")

	// Data rows
	for r, row := range plainRows {
		for c, cell := range row {
			// Pad the plain text first
			paddedCell := pad(cell, widths[c], t.columns[c].AlignRight)

			// Apply column style if provided
			finalCell := paddedCell
			if t.columns[c].Style != nil && !t.stylesDisabled {
				finalCell = t.columns[c].Style(paddedCell, rows[r])
			}

			sb.WriteString(finalCell)
			if c < len(row)-1 {
				sb.WriteString(" | ")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// calculateWidths determines the display width for each column.
func (t *Table[T]) calculateWidths(headers []string, rows [][]string) []int {
	widths := make([]int, len(t.columns))

	for i := range t.columns {
		// Start with header width
		width := runewidth.StringWidth(headers[i])

		// Check all rows
		for _, row := range rows {
			cellWidth := runewidth.StringWidth(row[i])
			if cellWidth > width {
				width = cellWidth
			}
		}

		widths[i] = width
	}

	return widths
}

// pad pads a string with spaces to reach the target width.
// If alignRight is true, pads on the left; otherwise pads on the right.
func pad(s string, width int, alignRight bool) string {
	currentWidth := runewidth.StringWidth(s)
	if currentWidth >= width {
		return s
	}
	padding := strings.Repeat(" ", width-currentWidth)
	if alignRight {
		return padding + s
	}
	return s + padding
}
