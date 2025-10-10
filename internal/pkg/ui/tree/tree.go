package tree

import (
	tea "github.com/charmbracelet/bubbletea"
)

const (
	defaultHeight = 20
	defaultWidth  = 80
)

type options func(*treeModel)

// Run creates a tree view for the given data and runs the interactive program
func Run(data any, opts ...options) error {
	nodes := parseNodes(data)
	m := &treeModel{
		nodes:  nodes,
		cursor: 0,
		height: defaultHeight,
		width:  defaultWidth,
		styles: defaultStyles(),
	}

	for _, opt := range opts {
		opt(m)
	}

	m.updateFlatNodes()

	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
