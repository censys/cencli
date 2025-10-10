package rawtable

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

type TestData struct {
	ID     int
	Name   string
	Active bool
	Score  float64
}

func TestNew(t *testing.T) {
	columns := []Column[TestData]{
		{
			Title:  "ID",
			String: func(td TestData) string { return fmt.Sprint(td.ID) },
		},
	}

	table := New(columns)
	if table == nil {
		t.Fatal("New returned nil")
	}
	if len(table.columns) != 1 {
		t.Errorf("expected 1 column, got %d", len(table.columns))
	}
}

func TestRender_EmptyData(t *testing.T) {
	columns := []Column[TestData]{
		{
			Title:  "ID",
			String: func(td TestData) string { return fmt.Sprint(td.ID) },
		},
	}

	table := New(columns)
	result := table.Render([]TestData{})

	if result != "" {
		t.Errorf("expected empty string for empty data, got: %q", result)
	}
}

func TestRender_BasicTable(t *testing.T) {
	data := []TestData{
		{ID: 1, Name: "Alice", Active: true, Score: 95.5},
		{ID: 2, Name: "Bob", Active: false, Score: 87.3},
		{ID: 3, Name: "Charlie", Active: true, Score: 92.1},
	}

	columns := []Column[TestData]{
		{
			Title:  "ID",
			String: func(td TestData) string { return fmt.Sprint(td.ID) },
		},
		{
			Title:  "Name",
			String: func(td TestData) string { return td.Name },
		},
		{
			Title: "Status",
			String: func(td TestData) string {
				if td.Active {
					return "Active"
				}
				return "Inactive"
			},
		},
	}

	table := New(columns, WithStylesDisabled[TestData](true))
	result := table.Render(data)

	// Check that result contains expected content
	if !strings.Contains(result, "Alice") {
		t.Error("expected result to contain 'Alice'")
	}
	if !strings.Contains(result, "Bob") {
		t.Error("expected result to contain 'Bob'")
	}
	if !strings.Contains(result, "Charlie") {
		t.Error("expected result to contain 'Charlie'")
	}
	if !strings.Contains(result, "ID") {
		t.Error("expected result to contain 'ID' header")
	}
	if !strings.Contains(result, "Name") {
		t.Error("expected result to contain 'Name' header")
	}
}

func TestRender_WithAlignment(t *testing.T) {
	data := []TestData{
		{ID: 1, Name: "Alice", Score: 95.5},
		{ID: 999, Name: "Bob", Score: 87.3},
	}

	columns := []Column[TestData]{
		{
			Title:      "ID",
			String:     func(td TestData) string { return fmt.Sprint(td.ID) },
			AlignRight: true,
		},
		{
			Title:  "Name",
			String: func(td TestData) string { return td.Name },
		},
		{
			Title:      "Score",
			String:     func(td TestData) string { return fmt.Sprintf("%.1f", td.Score) },
			AlignRight: true,
		},
	}

	table := New(columns, WithStylesDisabled[TestData](true))
	result := table.Render(data)

	// Check basic content
	if !strings.Contains(result, "Alice") {
		t.Error("expected result to contain 'Alice'")
	}
	if !strings.Contains(result, "95.5") {
		t.Error("expected result to contain '95.5'")
	}
	if !strings.Contains(result, "999") {
		t.Error("expected result to contain '999'")
	}
}

func TestRender_WithStyle(t *testing.T) {
	data := []TestData{
		{ID: 1, Name: "Alice", Active: true},
		{ID: 2, Name: "Bob", Active: false},
	}

	columns := []Column[TestData]{
		{
			Title:  "ID",
			String: func(td TestData) string { return fmt.Sprint(td.ID) },
		},
		{
			Title:  "Name",
			String: func(td TestData) string { return td.Name },
		},
		{
			Title: "Status",
			String: func(td TestData) string {
				if td.Active {
					return "Active"
				}
				return "Inactive"
			},
			Style: func(s string, td TestData) string {
				if td.Active {
					return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(s)
				}
				return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(s)
			},
		},
	}

	table := New(columns)
	result := table.Render(data)

	// With styles enabled, result should contain ANSI codes
	if !strings.Contains(result, "Alice") {
		t.Error("expected result to contain 'Alice'")
	}
	if !strings.Contains(result, "Bob") {
		t.Error("expected result to contain 'Bob'")
	}
}

func TestRender_WithHeaderStyle(t *testing.T) {
	data := []TestData{
		{ID: 1, Name: "Alice"},
	}

	columns := []Column[TestData]{
		{
			Title:  "ID",
			String: func(td TestData) string { return fmt.Sprint(td.ID) },
		},
		{
			Title:  "Name",
			String: func(td TestData) string { return td.Name },
		},
	}

	headerStyle := lipgloss.NewStyle().Bold(true)
	table := New(columns, WithHeaderStyle[TestData](headerStyle))
	result := table.Render(data)

	// Check that content is present
	if !strings.Contains(result, "Alice") {
		t.Error("expected result to contain 'Alice'")
	}
	if !strings.Contains(result, "ID") {
		t.Error("expected result to contain 'ID' header")
	}
}

func TestRender_StylesDisabled(t *testing.T) {
	data := []TestData{
		{ID: 1, Name: "Alice", Active: true},
	}

	columns := []Column[TestData]{
		{
			Title:  "Name",
			String: func(td TestData) string { return td.Name },
			Style: func(s string, td TestData) string {
				// This should not be applied when styles are disabled
				return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(s)
			},
		},
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	table := New(columns, WithHeaderStyle[TestData](headerStyle), WithStylesDisabled[TestData](true))
	result := table.Render(data)

	// With styles disabled, result should not contain ANSI escape codes
	if !strings.Contains(result, "Alice") {
		t.Error("expected result to contain 'Alice'")
	}

	// Check that no ANSI codes are present (they start with ESC character)
	if strings.Contains(result, "\x1b[") {
		t.Error("expected no ANSI codes when styles are disabled")
	}
}

func TestRender_MultipleColumns(t *testing.T) {
	data := []TestData{
		{ID: 1, Name: "Alice", Active: true, Score: 95.5},
		{ID: 2, Name: "Bob", Active: false, Score: 87.3},
	}

	columns := []Column[TestData]{
		{
			Title:      "ID",
			String:     func(td TestData) string { return fmt.Sprint(td.ID) },
			AlignRight: true,
		},
		{
			Title:  "Name",
			String: func(td TestData) string { return td.Name },
		},
		{
			Title: "Active",
			String: func(td TestData) string {
				if td.Active {
					return "✓"
				}
				return "✗"
			},
		},
		{
			Title:      "Score",
			String:     func(td TestData) string { return fmt.Sprintf("%.1f", td.Score) },
			AlignRight: true,
		},
	}

	table := New(columns, WithStylesDisabled[TestData](true))
	result := table.Render(data)

	// Verify all column headers are present
	if !strings.Contains(result, "ID") {
		t.Error("expected result to contain 'ID' header")
	}
	if !strings.Contains(result, "Name") {
		t.Error("expected result to contain 'Name' header")
	}
	if !strings.Contains(result, "Active") {
		t.Error("expected result to contain 'Active' header")
	}
	if !strings.Contains(result, "Score") {
		t.Error("expected result to contain 'Score' header")
	}

	// Verify data is present
	if !strings.Contains(result, "Alice") {
		t.Error("expected result to contain 'Alice'")
	}
	if !strings.Contains(result, "Bob") {
		t.Error("expected result to contain 'Bob'")
	}
}
