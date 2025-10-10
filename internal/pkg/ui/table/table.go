package table

import (
	"github.com/censys/cencli/internal/pkg/styles"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tableComponent[T any] struct {
	table            table.Model
	cols             []table.Column
	rowsData         []T
	rowRenderer      RowRenderer[T]
	onSelect         func(T)
	title            string
	keyActions       []KeyAction[T]
	selectDesc       string
	keepOpenOnSelect bool
}

type RowRenderer[T any] func(T) []string

type KeyAction[T any] struct {
	Key         string
	Description string
	Action      func(T)
	ShowConfirm bool
}

type tableComponentOptions[T any] struct {
	height           int
	styles           table.Styles
	onSelect         func(T)
	columnWidths     []int
	title            string
	keyActions       []KeyAction[T]
	selectDesc       string
	keepOpenOnSelect bool
}

type tableComponentOption[T any] func(*tableComponentOptions[T])

func WithHeight[T any](h int) tableComponentOption[T] {
	return func(t *tableComponentOptions[T]) {
		t.height = h
	}
}

func WithStyles[T any](s table.Styles) tableComponentOption[T] {
	return func(t *tableComponentOptions[T]) {
		t.styles = s
	}
}

func WithSelectFunc[T any](fn func(T)) tableComponentOption[T] {
	return func(t *tableComponentOptions[T]) {
		t.onSelect = fn
	}
}

func WithColumnWidths[T any](widths []int) tableComponentOption[T] {
	return func(t *tableComponentOptions[T]) {
		t.columnWidths = widths
	}
}

func WithTitle[T any](title string) tableComponentOption[T] {
	return func(t *tableComponentOptions[T]) {
		t.title = title
	}
}

func WithKeyActions[T any](actions []KeyAction[T]) tableComponentOption[T] {
	return func(t *tableComponentOptions[T]) {
		t.keyActions = actions
	}
}

func WithSelectDescription[T any](desc string) tableComponentOption[T] {
	return func(t *tableComponentOptions[T]) {
		t.selectDesc = desc
	}
}

func WithKeepOpenOnSelect[T any](keepOpen bool) tableComponentOption[T] {
	return func(t *tableComponentOptions[T]) {
		t.keepOpenOnSelect = keepOpen
	}
}

func NewTable[T any](titles []string, rowRenderer RowRenderer[T], opts ...tableComponentOption[T]) *tableComponent[T] {
	// Set up defaults
	options := &tableComponentOptions[T]{
		height: 7,
		styles: defaultStyles(),
	}
	for _, opt := range opts {
		opt(options)
	}

	cols := make([]table.Column, len(titles))
	for i, t := range titles {
		var width int
		if i < len(options.columnWidths) {
			width = options.columnWidths[i]
		} else {
			width = len([]rune(t)) + 5
		}
		cols[i] = table.Column{
			Title: t,
			Width: width,
		}
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithHeight(options.height),
		table.WithStyles(options.styles),
		table.WithFocused(true),
	)

	return &tableComponent[T]{
		table:            t,
		cols:             cols,
		rowRenderer:      rowRenderer,
		onSelect:         options.onSelect,
		title:            options.title,
		keyActions:       options.keyActions,
		selectDesc:       options.selectDesc,
		keepOpenOnSelect: options.keepOpenOnSelect,
	}
}

func defaultStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(styles.ColorTeal).
		Bold(false)
	return s
}

func (t *tableComponent[T]) setRows(data []T) *tableComponent[T] {
	t.rowsData = data
	rows := make([]table.Row, 0, len(data))
	for _, d := range data {
		row := t.rowRenderer(d)
		r := make(table.Row, len(row))
		copy(r, row)
		rows = append(rows, r)
	}
	t.table.SetRows(rows)
	return t
}

func (t *tableComponent[T]) Run(rows []T) error {
	t.setRows(rows)
	m := model[T]{t: t}
	_, err := tea.NewProgram(m).Run()
	return err
}

type model[T any] struct {
	t               *tableComponent[T]
	showingConfirm  bool
	confirmAction   *KeyAction[T]
	confirmSelected T
}

func (m model[T]) Init() tea.Cmd { return nil }

func (m model[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Handle confirmation dialog
		if m.showingConfirm {
			switch keyMsg.String() {
			case "y", "Y", "enter":
				// User confirmed, execute the action
				if m.confirmAction != nil {
					m.confirmAction.Action(m.confirmSelected)
				}
				return m, tea.Quit
			case "n", "N", "esc", "q":
				// User cancelled, go back to table
				m.showingConfirm = false
				m.confirmAction = nil
				return m, nil
			}
			return m, nil
		}

		// Normal table handling
		switch keyMsg.String() {
		case "esc":
			if m.t.table.Focused() {
				m.t.table.Blur()
			} else {
				m.t.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.t.onSelect != nil && len(m.t.rowsData) > 0 {
				idx := m.t.table.Cursor()
				if 0 <= idx && idx < len(m.t.rowsData) {
					selected := m.t.rowsData[idx]
					m.t.onSelect(selected)
					if !m.t.keepOpenOnSelect {
						return m, tea.Quit
					}
				}
			}
		default:
			// Check for custom key actions
			for _, action := range m.t.keyActions {
				if keyMsg.String() == action.Key {
					var selected T
					if len(m.t.rowsData) > 0 {
						idx := m.t.table.Cursor()
						if 0 <= idx && idx < len(m.t.rowsData) {
							selected = m.t.rowsData[idx]
						}
					}

					// Check if confirmation is needed
					if action.ShowConfirm {
						m.showingConfirm = true
						m.confirmAction = &action
						m.confirmSelected = selected
						return m, nil
					}
					// Execute action immediately
					action.Action(selected)
					return m, tea.Quit
				}
			}
		}
	}

	// Only update table if not showing confirmation
	if !m.showingConfirm {
		m.t.table, cmd = m.t.table.Update(msg)
	}
	return m, cmd
}

func (m model[T]) View() string {
	// If showing confirmation dialog, render it instead of the table
	if m.showingConfirm {
		return m.renderConfirmationDialog()
	}

	content := m.t.table.View()

	if m.t.title != "" {

		highlightStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("57")).
			Background(lipgloss.Color("229"))

		explanationStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			MarginBottom(1)

		title := styles.GlobalStyles.Signature.
			Bold(true).
			MarginBottom(1).
			MarginTop(1).
			Render(m.t.title)

		// Build dynamic instructions based on configuration
		instructions := "Use ↑/↓ to navigate"

		// Add select instruction if configured
		if m.t.onSelect != nil {
			selectDesc := m.t.selectDesc
			if selectDesc == "" {
				selectDesc = "select"
			}
			instructions += ", " + highlightStyle.Render("Enter") + " to " + selectDesc
		}

		// Add custom key action instructions
		for _, action := range m.t.keyActions {
			instructions += ", " + highlightStyle.Render(action.Key) + " to " + action.Description
		}

		// Always add quit instruction
		instructions += ", " + highlightStyle.Render("q") + " to quit"

		explanation := explanationStyle.Render(instructions)

		return title + "\n" + content + "\n" + explanation + "\n"
	}

	return content + "\n"
}

func (m model[T]) renderConfirmationDialog() string {
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		MarginTop(2).
		MarginBottom(2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		MarginBottom(1)

	highlightStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("57")).
		Background(lipgloss.Color("229"))

	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginTop(1)

	title := titleStyle.Render("⚠️  Confirmation Required")

	var message string
	if m.confirmAction != nil {
		message = "Are you sure you want to " + m.confirmAction.Description + "?"
	} else {
		message = "Are you sure you want to proceed?"
	}

	instructions := instructionStyle.Render(
		highlightStyle.Render("Y/Enter") + " to confirm, " +
			highlightStyle.Render("N/Esc") + " to cancel")

	dialogContent := title + "\n" + message + "\n" + instructions

	return dialogStyle.Render(dialogContent)
}
