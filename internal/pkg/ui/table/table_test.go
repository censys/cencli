package table

import (
	"testing"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type row struct{ a, b string }

func (r row) asRow() []string { return []string{r.a, r.b} }

func TestNewTable_WithOptionsAndSetRows(t *testing.T) {
	cols := []string{"A", "B"}
	tt := NewTable[row](cols, func(r row) []string { return r.asRow() },
		WithTitle[row]("Title"),
		WithHeight[row](5),
		WithStyles[row](table.Styles{}),
		WithColumnWidths[row]([]int{3, 7}),
		WithSelectDescription[row]("choose"),
	)

	if tt == nil {
		t.Fatal("NewTable returned nil")
	}
	if len(tt.cols) != len(cols) {
		t.Fatalf("columns mismatch: got %d want %d", len(tt.cols), len(cols))
	}

	// Set rows and ensure internal data mirrors input
	in := []row{{"x", "y"}, {"1", "2"}}
	tt.setRows(in)
	if len(tt.rowsData) != len(in) {
		t.Fatalf("rowsData length mismatch: got %d want %d", len(tt.rowsData), len(in))
	}
}

func TestModelInitUpdateView(t *testing.T) {
	cols := []string{"A", "B"}
	tt := NewTable[row](cols, func(r row) []string { return r.asRow() }, WithTitle[row]("My Title"))
	tt.setRows([]row{{"x", "y"}})
	m := model[row]{t: tt}

	// Init returns nil
	if m.Init() != nil {
		t.Fatalf("Init should return nil")
	}

	// View returns a non-empty string and includes title
	if v := m.View(); v == "" {
		t.Fatalf("View should not be empty")
	}
}

func TestModelConfirmActionFlow(t *testing.T) {
	cols := []string{"A", "B"}
	executed := false
	action := KeyAction[row]{
		Key:         "x",
		Description: "delete",
		Action:      func(r row) { executed = true },
		ShowConfirm: true,
	}
	tt := NewTable[row](cols, func(r row) []string { return r.asRow() }, WithKeyActions[row]([]KeyAction[row]{action}))
	tt.setRows([]row{{"a", "b"}})
	m := model[row]{t: tt}

	// Trigger the action key
	if mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}); mm != nil {
		m = mm.(model[row])
	}
	// While in confirm mode, View should render dialog
	if view := m.View(); view == "" {
		t.Fatalf("confirm dialog view should not be empty")
	}
	// Confirm with 'y'
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !executed {
		t.Fatalf("expected action to execute on confirm")
	}
}

func TestModelCancelConfirmReturnsToTable(t *testing.T) {
	cols := []string{"A", "B"}
	executed := false
	action := KeyAction[row]{
		Key:         "x",
		Description: "delete",
		Action:      func(r row) { executed = true },
		ShowConfirm: true,
	}
	tt := NewTable[row](cols, func(r row) []string { return r.asRow() }, WithKeyActions[row]([]KeyAction[row]{action}))
	tt.setRows([]row{{"a", "b"}})
	m := model[row]{t: tt}

	// Trigger confirm dialog
	if mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}); mm != nil {
		m = mm.(model[row])
	}
	// Cancel with 'n'
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if executed {
		t.Fatalf("unexpected execution after cancel")
	}
}

func TestModelOnSelectEnter(t *testing.T) {
	cols := []string{"A", "B"}
	selected := false
	tt := NewTable[row](cols, func(r row) []string { return r.asRow() }, WithSelectFunc[row](func(r row) { selected = true }))
	tt.setRows([]row{{"a", "b"}})
	m := model[row]{t: tt}
	// Press enter
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !selected {
		t.Fatalf("expected select func to be called on enter")
	}
}
