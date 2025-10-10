package tree

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/censys/cencli/internal/pkg/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

type treeModel struct {
	nodes         []*node // Root nodes
	cursor        int     // Current cursor position
	flatNodes     []*node // Flattened view of visible nodes
	height        int     // Terminal height
	width         int     // Terminal width
	styles        Styles  // Styling configuration
	statusMessage string  // Status message to display
}

// clearStatusMsg is a message to clear the status message
type clearStatusMsg struct{}

func (m *treeModel) Init() tea.Cmd {
	return nil
}

func (m *treeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		return m, nil

	case clearStatusMsg:
		m.statusMessage = ""
		return m, nil

	case tea.KeyMsg:
		// Clear status message on any key press
		m.statusMessage = ""

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.flatNodes)-1 {
				m.cursor++
			}

		case "enter":
			// Copy leaf node value to clipboard or toggle expansion for non-leaf nodes
			if m.cursor < len(m.flatNodes) {
				node := m.flatNodes[m.cursor]
				if node.IsLeaf {
					// Copy the value to clipboard, stripping quotes from strings
					value := node.Value
					// Remove surrounding quotes if present
					if unquoted, err := strconv.Unquote(value); err == nil {
						value = unquoted
					}
					if err := clipboard.Copy(value); err == nil {
						m.statusMessage = "copied value to clipboard"
					} else {
						m.statusMessage = "failed to copy to clipboard"
					}
					return m, clearStatusAfter(2 * time.Second)
				}
				// For non-leaf nodes, toggle expansion
				node.Expanded = !node.Expanded
				m.updateFlatNodes()
			}

		case " ":
			// Space always toggles expansion
			if m.cursor < len(m.flatNodes) {
				node := m.flatNodes[m.cursor]
				if !node.IsLeaf {
					node.Expanded = !node.Expanded
					m.updateFlatNodes()
				}
			}

		case "right", "l":
			// Expand current node
			if m.cursor < len(m.flatNodes) {
				node := m.flatNodes[m.cursor]
				if !node.IsLeaf && !node.Expanded {
					node.Expanded = true
					m.updateFlatNodes()
				}
			}

		case "left", "h":
			// Collapse current node
			if m.cursor < len(m.flatNodes) {
				node := m.flatNodes[m.cursor]
				if !node.IsLeaf && node.Expanded {
					node.Expanded = false
					m.updateFlatNodes()
				}
			}
		}
	}

	return m, nil
}

// clearStatusAfter returns a command that sends a clearStatusMsg after a delay
func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func (m *treeModel) View() string {
	if len(m.flatNodes) == 0 {
		return "No data to display"
	}

	var b strings.Builder

	visibleHeight := m.height - 5 // Account for header and footer
	start := 0
	end := len(m.flatNodes)

	if len(m.flatNodes) > visibleHeight {
		// Center the cursor in the view
		start = m.cursor - visibleHeight/2
		if start < 0 {
			start = 0
		}
		end = start + visibleHeight
		if end > len(m.flatNodes) {
			end = len(m.flatNodes)
			start = end - visibleHeight
			if start < 0 {
				start = 0
			}
		}
	}

	// Render visible nodes
	for i := start; i < end; i++ {
		node := m.flatNodes[i]
		line := m.renderNode(node, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Display status message if present, otherwise show help
	if m.statusMessage != "" {
		b.WriteString(m.styles.SelectedStyle.Render(m.statusMessage))
	} else {
		b.WriteString(m.styles.HelpStyle.Render("↑/↓: navigate, ←/→/space/enter: expand/collapse, enter (leaf): copy value, q: quit"))
	}
	b.WriteString(m.styles.FooterStyle.Render(fmt.Sprintf(" (%d/%d)", m.cursor+1, len(m.flatNodes))))
	b.WriteString("\n")

	return b.String()
}

func (m *treeModel) renderNode(node *node, selected bool) string {
	depth := m.getNodeDepth(node)
	indent := strings.Repeat("  ", depth)

	var icon string
	switch {
	case node.IsLeaf:
		icon = m.styles.LeafSymbol
	case node.Expanded:
		icon = m.styles.ExpandedSymbol
	default:
		icon = m.styles.CollapsedSymbol
	}

	key := node.Key
	if key == "" {
		key = "data"
	}

	// Determine what value to show
	var value string
	switch {
	case node.IsLeaf:
		// Leaf nodes always show their value
		value = node.Value
	case node.Expanded:
		// Expanded non-leaf nodes show no value (just the key and children)
		value = ""
	default:
		// Collapsed non-leaf nodes show their summary
		value = node.Value
	}

	// Calculate available width for the value
	prefixLength := len(indent) + len(icon) + len(key) + len(": ")
	availableWidth := m.width - prefixLength - 3 // Reserve 3 chars for "..."

	// Truncate the raw value if it's too long, before styling
	if value != "" && m.width > 0 && availableWidth > 0 && len(value) > availableWidth {
		if availableWidth > 3 {
			value = value[:availableWidth-3] + "..."
		} else {
			value = "..."
		}
	}

	styledKey := m.styles.KeyStyle.Render(key)

	var line string
	if value == "" {
		// No value to show (expanded non-leaf nodes)
		line = fmt.Sprintf("%s%s%s", indent, icon, styledKey)
	} else {
		// Show key: value
		styledValue := m.styleValue(value)
		line = fmt.Sprintf("%s%s%s: %s", indent, icon, styledKey, styledValue)
	}

	if selected {
		return m.styles.SelectedStyle.Render(line)
	}

	return line
}

func (m *treeModel) styleValue(value string) string {
	// Determine the type of value and apply appropriate styling
	if value == "null" {
		return m.styles.NullStyle.Render(value)
	}

	// Check if it's a boolean
	if value == "true" || value == "false" {
		return m.styles.BoolStyle.Render(value)
	}

	// Check if it's a string (starts and ends with quotes)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return m.styles.StringStyle.Render(value)
	}

	// Check if it's an array indicator
	if strings.HasPrefix(value, "array[") {
		return m.styles.ArrayStyle.Render(value)
	}

	// check if its a number
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return m.styles.NumberStyle.Render(value)
	}

	return m.styles.ObjectStyle.Render(value)
}

// getNodeDepth calculates the depth of a node in the tree
func (m *treeModel) getNodeDepth(node *node) int {
	depth := 0
	current := node.Parent
	for current != nil {
		depth++
		current = current.Parent
	}
	return depth
}

func (m *treeModel) updateFlatNodes() {
	m.flatNodes = []*node{}
	for _, node := range m.nodes {
		m.addNodeToFlat(node)
	}

	if m.cursor >= len(m.flatNodes) {
		m.cursor = len(m.flatNodes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *treeModel) addNodeToFlat(node *node) {
	m.flatNodes = append(m.flatNodes, node)

	if node.Expanded && !node.IsLeaf {
		for _, child := range node.Children {
			m.addNodeToFlat(child)
		}
	}
}
