package config

import (
	"time"

	"github.com/censys/cencli/internal/pkg/formatter"
	"github.com/censys/cencli/internal/pkg/ui/table"
)

// helpers for building TUI tables in config commands

func truncateEnd(s string, max int) string { return formatter.TruncateEnd(s, max) }
func formatShort(t time.Time) string       { return formatter.FormatShortTime(t) }

func mostRecentIndexByLastUsed[T any](values []T, getLastUsed func(T) time.Time) int {
	if len(values) == 0 {
		return -1
	}
	idx := 0
	latest := getLastUsed(values[0])
	for i := 1; i < len(values); i++ {
		ts := getLastUsed(values[i])
		if ts.After(latest) {
			latest = ts
			idx = i
		}
	}
	return idx
}

// runValuesTable builds and runs a generic values table for config lists.
// titles and column widths are standardized; callers provide extraction and actions.
func runValuesTable[T any](
	title string,
	values []T,
	getID func(T) int64,
	getDesc func(T) string,
	getValue func(T) string,
	getLastUsed func(T) time.Time,
	onDelete func(selected T),
	onSelect func(selected T),
) error {
	selectedIdx := mostRecentIndexByLastUsed(values, getLastUsed)

	rowRenderer := func(v T) []string {
		valueDisplay := truncateEnd(getValue(v), 25)
		lastUsed := formatShort(getLastUsed(v))
		status := "Inactive"
		if selectedIdx >= 0 {
			// Compare by ID to avoid issues if values are not pointer-equal
			if getID(v) == getID(values[selectedIdx]) {
				status = "Active"
			}
		}
		return []string{
			status,
			formatter.Int64String(getID(v)),
			getDesc(v),
			valueDisplay,
			lastUsed,
		}
	}

	titles := []string{"Status", "ID", "Name", "Value", "Last Used"}
	columnWidths := []int{8, 4, 12, 20, 18}

	if len(values) > 0 {
		var keyActions []table.KeyAction[T]
		if onDelete != nil {
			keyActions = []table.KeyAction[T]{
				{
					Key:         "d",
					Description: "delete selected value",
					ShowConfirm: true,
					Action:      onDelete,
				},
			}
		}
		t := table.NewTable[T](
			titles,
			rowRenderer,
			table.WithHeight[T](min(len(values)+2, 10)),
			table.WithColumnWidths[T](columnWidths),
			table.WithTitle[T](title),
			table.WithKeyActions[T](keyActions),
			table.WithSelectDescription[T]("set as active"),
			table.WithSelectFunc(onSelect),
		)
		return t.Run(values)
	}

	// Empty state
	t := table.NewTable[T](
		titles,
		rowRenderer,
		table.WithHeight[T](5),
		table.WithColumnWidths[T](columnWidths),
		table.WithTitle[T](title),
	)
	return t.Run(values)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
