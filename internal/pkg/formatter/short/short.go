package short

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/censys/cencli/internal/pkg/styles"
)

// Val returns the value of a pointer, or the zero value if the pointer is nil
func Val[T any](p *T, zero T) T {
	if p == nil {
		return zero
	}
	return *p
}

// Separator is a standard separator line for short output
func Separator() string {
	return "------------------------------------------------------------\n"
}

// SeparatorWithLabel returns a separator with a centered label (e.g., "--- Host #1 ---")
func SeparatorWithLabel(label string) string {
	const sepWidth = 60
	const minDashes = 3

	if label == "" {
		return Separator()
	}

	labelLen := len(label)
	totalDashes := sepWidth - labelLen - 2 // -2 for spaces around label

	if totalDashes < minDashes*2 {
		return Separator()
	}

	leftDashes := totalDashes / 2
	rightDashes := totalDashes - leftDashes

	return fmt.Sprintf("%s %s %s\n",
		strings.Repeat("-", leftDashes),
		label,
		strings.Repeat("-", rightDashes))
}

type Block struct {
	s          strings.Builder
	keyStyle   lipgloss.Style
	valueStyle lipgloss.Style
	indent     int
	// item mode for list items with hyphen prefix
	itemMode   bool // we're inside an item
	itemIndent int  // indent for fields inside an item
}

type BlockOption func(*Block)

func WithKeyStyle(style lipgloss.Style) BlockOption {
	return func(b *Block) {
		b.keyStyle = style
	}
}

func WithValueStyle(style lipgloss.Style) BlockOption {
	return func(b *Block) {
		b.valueStyle = style
	}
}

func WithIndent(n int) BlockOption {
	return func(b *Block) {
		b.indent = n
	}
}

func NewBlock(opts ...BlockOption) *Block {
	b := &Block{
		keyStyle:   styles.GlobalStyles.Primary,
		valueStyle: styles.GlobalStyles.Tertiary,
		indent:     2,
		itemIndent: 6, // default: indent + 4
	}
	for _, opt := range opts {
		opt(b)
	}
	// Update itemIndent based on final indent value
	b.itemIndent = b.indent + 4
	return b
}

func (b *Block) String() string {
	return b.s.String()
}

func (b *Block) Write(content string) {
	b.s.WriteString(content)
}

// WriteLine writes content followed by a newline to the block.
func (b *Block) WriteLine(content string) {
	b.s.WriteString(content)
	b.s.WriteString("\n")
}

// Title writes a section title to the block.
func (b *Block) Title(t string) {
	b.s.WriteString(b.keyStyle.Render(t))
	b.s.WriteString("\n")
}

// Field writes a key-value pair to the block with proper indentation and styling.
// Automatically skips empty values.
func (b *Block) Field(k, v string) {
	if strings.TrimSpace(v) == "" {
		return
	}
	b.s.WriteString(strings.Repeat(" ", b.indent))
	b.s.WriteString(b.keyStyle.Render(k))
	b.s.WriteString(": ")
	b.s.WriteString(b.valueStyle.Render(v))
	b.s.WriteString("\n")
}

// Fieldf writes a formatted key-value pair to the block with proper indentation and styling.
// Automatically skips empty values.
func (b *Block) Fieldf(k, format string, args ...any) {
	b.Field(k, fmt.Sprintf(format, args...))
}

// Fields writes multiple key-value pairs to the block.
func (b *Block) Fields(m map[string]string) {
	for k, v := range m {
		b.Field(k, v)
	}
}

// Separator writes a standard separator line to the block.
func (b *Block) Separator() {
	b.s.WriteString(Separator())
}

// SeparatorWithLabel writes a separator with a label to the block.
func (b *Block) SeparatorWithLabel(label string) {
	b.s.WriteString(SeparatorWithLabel(label))
}

// Newline writes a blank line to the block.
func (b *Block) Newline() {
	b.s.WriteString("\n")
}

// Item starts a new list item with a hyphen prefix.
// The title can be raw text or already-styled text.
func (b *Block) Item(title string) {
	b.itemMode = true

	b.s.WriteString(strings.Repeat(" ", b.indent))
	b.s.WriteString("- ")
	b.s.WriteString(title) // raw styled string from caller
	b.s.WriteString("\n")
}

// ItemField writes a key-value pair indented under the current item.
// Automatically skips empty values.
func (b *Block) ItemField(k, v string) {
	if !b.itemMode || strings.TrimSpace(v) == "" {
		return
	}

	b.s.WriteString(strings.Repeat(" ", b.itemIndent))
	b.s.WriteString(b.keyStyle.Render(k))
	b.s.WriteString(": ")
	b.s.WriteString(b.valueStyle.Render(v))
	b.s.WriteString("\n")
}

// EndItem closes the current item and adds spacing.
func (b *Block) EndItem() {
	if !b.itemMode {
		return
	}
	b.s.WriteString("\n")
	b.itemMode = false
}

type Line struct {
	s          strings.Builder
	labelStyle lipgloss.Style
	valueStyle lipgloss.Style
}

type LineOption func(*Line)

func WithLabelStyle(style lipgloss.Style) LineOption {
	return func(l *Line) {
		l.labelStyle = style
	}
}

func WithLineValueStyle(style lipgloss.Style) LineOption {
	return func(l *Line) {
		l.valueStyle = style
	}
}

func NewLine(opts ...LineOption) *Line {
	l := &Line{
		labelStyle: styles.GlobalStyles.Primary,
		valueStyle: styles.GlobalStyles.Tertiary,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *Line) String() string {
	return l.s.String()
}

func (l *Line) Write(label, value string) {
	l.s.WriteString(l.labelStyle.Render(label))
	if value != "" {
		l.s.WriteString(": ")
		l.s.WriteString(l.valueStyle.Render(value))
	}
	l.s.WriteString("\n")
}

// WriteInline writes a label and value without a trailing newline.
func (l *Line) WriteInline(label, value string) {
	l.s.WriteString(l.labelStyle.Render(label))
	if value != "" {
		l.s.WriteString(": ")
		l.s.WriteString(l.valueStyle.Render(value))
	}
}

func (l *Line) Newline() {
	l.s.WriteString("\n")
}

// Section is a convenience function that creates a block with a title and applies a function to it.
func Section(title string, f func(*Block)) string {
	b := NewBlock()
	b.Title(title)
	f(b)
	return b.String()
}
