package command

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/censys/cencli/internal/pkg/styles"
)

const (
	// descriptionMaxLineLength is the maximum character width for wrapping
	// long descriptions in help text.
	descriptionMaxLineLength = 90
)

// usageTemplate renders a usage block with examples and flags.
func usageTemplate(cmd *cobra.Command, examples []string) string {
	var b strings.Builder
	b.WriteString(styles.GlobalStyles.Info.Render("Usage:") + "\n")
	if cmd.Runnable() {
		b.WriteString("  " + useLine(cmd) + "\n")
	}
	if cmd.HasAvailableSubCommands() {
		cmdPath := cmd.CommandPath()
		parts := strings.SplitN(cmdPath, " ", 2)
		formattedPath := styles.GlobalStyles.Signature.Render(parts[0])
		if len(parts) > 1 {
			formattedPath += " " + styles.GlobalStyles.Primary.Render(parts[1])
		}
		b.WriteString("  " + formattedPath + " " + styles.GlobalStyles.Secondary.Render("[command]") + "\n")
	}
	if len(examples) > 0 {
		b.WriteString("\n" + styles.GlobalStyles.Info.Render("Examples:") + "\n")
		for _, example := range examples {
			cmdPath := cmd.CommandPath()
			b.WriteString("  " + renderExampleLine(cmdPath, example) + "\n")
		}
	}
	if cmd.HasAvailableSubCommands() {
		cmds := cmd.Commands()
		if len(cmd.Groups()) == 0 {
			b.WriteString("\n" + styles.GlobalStyles.Info.Render("Available Commands:") + "\n")
			for _, c := range cmds {
				if c.IsAvailableCommand() || c.Name() == "help" {
					name := fmt.Sprintf("%-*s", cmd.NamePadding(), c.Name())
					fmt.Fprintf(&b, "  %s %s\n",
						styles.GlobalStyles.Signature.Render(name),
						styles.GlobalStyles.Secondary.Render(c.Short))
				}
			}
		} else {
			for _, g := range cmd.Groups() {
				b.WriteString("\n" + styles.GlobalStyles.Info.Render(g.Title) + "\n")
				for _, c := range cmds {
					if c.GroupID == g.ID && (c.IsAvailableCommand() || c.Name() == "help") {
						name := fmt.Sprintf("%-*s", cmd.NamePadding(), c.Name())
						fmt.Fprintf(&b, "  %s %s\n",
							styles.GlobalStyles.Signature.Render(name),
							styles.GlobalStyles.Tertiary.Render(c.Short))
					}
				}
			}
			if !cmd.AllChildCommandsHaveGroup() {
				b.WriteString("\n" + styles.GlobalStyles.Info.Render("Additional Commands:") + "\n")
				for _, c := range cmds {
					if c.GroupID == "" && (c.IsAvailableCommand() || c.Name() == "help") {
						name := fmt.Sprintf("%-*s", cmd.NamePadding(), c.Name())
						fmt.Fprintf(&b, "  %s %s\n",
							styles.GlobalStyles.Signature.Render(name),
							styles.GlobalStyles.Tertiary.Render(c.Short))
					}
				}
			}
		}
	}
	if cmd.HasAvailableLocalFlags() {
		b.WriteString("\n" + styles.GlobalStyles.Info.Render("Flags:") + "\n")
		b.WriteString(styledFlagUsages(cmd.LocalFlags()))
	}
	if cmd.HasAvailableInheritedFlags() {
		b.WriteString("\n" + styles.GlobalStyles.Info.Render("Global Flags:") + "\n")
		b.WriteString(styledFlagUsages(cmd.InheritedFlags()))
	}
	if cmd.HasHelpSubCommands() {
		b.WriteString("\n" + styles.GlobalStyles.Info.Render("Additional help topics:") + "\n")
		for _, c := range cmd.Commands() {
			if c.IsAdditionalHelpTopicCommand() {
				fmt.Fprintf(&b, "  %-15s %s\n", styles.GlobalStyles.Primary.Render(c.CommandPath()), styles.GlobalStyles.Secondary.Render(c.Short))
			}
		}
	}
	return b.String()
}

// HelpTemplate renders the full help text for a command.
func helpTemplate(cmd *cobra.Command, examples []string) string {
	var b strings.Builder
	text := cmd.Long
	if text == "" {
		text = cmd.Short
	}
	if text != "" {
		wrappedText := wrapText(strings.TrimRight(text, " \t\r\n"), descriptionMaxLineLength)
		b.WriteString(styles.GlobalStyles.Primary.Render(wrappedText))
		b.WriteString("\n\n")
	}
	if cmd.Runnable() || cmd.HasSubCommands() {
		b.WriteString(usageTemplate(cmd, examples))
	}
	return b.String()
}

func useLine(cmd *cobra.Command) string {
	useLine := cmd.UseLine()
	parts := strings.Fields(useLine)
	if len(parts) == 0 {
		return ""
	}
	rootCmdStr := styles.GlobalStyles.Signature.Render(parts[0])
	if len(parts) == 1 {
		return rootCmdStr
	}
	subCmdStr := styles.GlobalStyles.Primary.Render(parts[1])
	if len(parts) > 2 {
		rest := styles.GlobalStyles.Secondary.Render(strings.Join(parts[2:], " "))
		return fmt.Sprintf("%s %s %s", rootCmdStr, subCmdStr, rest)
	}
	return fmt.Sprintf("%s %s", rootCmdStr, subCmdStr)
}

func renderExampleLine(cmdPath, example string) string {
	parts := strings.SplitN(cmdPath, " ", 2)
	base := styles.GlobalStyles.Signature.Render(parts[0])
	subcommand := ""
	if len(parts) > 1 {
		subcommand = styles.GlobalStyles.Primary.Render(parts[1])
	}

	commentIdx := strings.Index(example, "#")
	var beforeComment, comment string
	if commentIdx >= 0 {
		beforeComment = example[:commentIdx]
		comment = styles.GlobalStyles.Comment.Render(example[commentIdx:])
	} else {
		beforeComment = example
	}

	beforeComment = styles.GlobalStyles.Secondary.Render(strings.TrimSpace(beforeComment))

	if subcommand != "" {
		if comment != "" {
			return fmt.Sprintf("%s %s %s", base, subcommand, beforeComment+" "+comment)
		}
		return fmt.Sprintf("%s %s %s", base, subcommand, beforeComment)
	}
	if comment != "" {
		return fmt.Sprintf("%s %s", base, beforeComment+" "+comment)
	}
	return fmt.Sprintf("%s %s", base, beforeComment)
}

// =============================================================================
// The below functions are almost entirely vendored from pflag/flag.go
// since there's no way to add styling to the output.
// =============================================================================

// styledFlagUsages returns flag usage information with styling applied.
// Designed to be identical to pflag.FlagUsagesWrapped(0).
func styledFlagUsages(f *pflag.FlagSet) string {
	buf := new(bytes.Buffer)

	lines := make([]string, 0)

	maxlen := 0
	f.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}

		line := ""
		if flag.Shorthand != "" && flag.ShorthandDeprecated == "" {
			line = fmt.Sprintf("  -%s, --%s", flag.Shorthand, flag.Name)
		} else {
			line = fmt.Sprintf("      --%s", flag.Name)
		}

		varname, usage := pflag.UnquoteUsage(flag)
		if varname != "" {
			line += " " + varname
		}
		if flag.NoOptDefVal != "" {
			switch flag.Value.Type() {
			case "string":
				line += fmt.Sprintf("[=\"%s\"]", flag.NoOptDefVal)
			case "bool", "boolfunc":
				if flag.NoOptDefVal != "true" {
					line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
				}
			case "count":
				if flag.NoOptDefVal != "+1" {
					line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
				}
			default:
				line += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
			}
		}

		// This special character will be replaced with spacing once the
		// correct alignment is calculated
		line += "\x00"
		if len(line) > maxlen {
			maxlen = len(line)
		}

		line += usage
		if !flagDefaultIsZeroValue(flag) {
			if flag.Value.Type() == "string" {
				line += fmt.Sprintf(" (default %q)", flag.DefValue)
			} else {
				line += fmt.Sprintf(" (default %s)", flag.DefValue)
			}
		}
		if len(flag.Deprecated) != 0 {
			line += fmt.Sprintf(" (DEPRECATED: %s)", flag.Deprecated)
		}

		lines = append(lines, line)
	})

	for _, line := range lines {
		sidx := strings.Index(line, "\x00")
		spacing := strings.Repeat(" ", maxlen-sidx)
		// maxlen + 2 comes from + 1 for the \x00 and + 1 for the (deliberate) off-by-one in maxlen-sidx

		// Apply styling to the output
		flagPart := line[:sidx]
		usagePart := line[sidx+1:]

		// Style the flag part (flag names)
		styledFlagPart := applyFlagNameStyling(flagPart)

		// Style the usage part (description and defaults)
		styledUsagePart := applyUsageStyling(usagePart)

		fmt.Fprintln(buf, styledFlagPart, spacing, flagWrap(maxlen+2, 0, styledUsagePart))
	}

	return buf.String()
}

// applyFlagNameStyling applies styling to flag names.
func applyFlagNameStyling(flagPart string) string {
	leadingSpaces := ""
	content := flagPart
	for i, r := range flagPart {
		if r != ' ' {
			leadingSpaces = flagPart[:i]
			content = flagPart[i:]
			break
		}
	}
	return leadingSpaces + styles.GlobalStyles.Primary.Render(content)
}

// applyUsageStyling applies styling to the usage description and default values.
func applyUsageStyling(usagePart string) string {
	return styles.GlobalStyles.Secondary.Render(usagePart)
}

// flagDefaultIsZeroValue returns true if the default value for this flag represents
// a zero value. Vendored from pflag since it's unexported.
func flagDefaultIsZeroValue(f *pflag.Flag) bool {
	switch f.Value.Type() {
	case "bool", "boolfunc":
		return f.DefValue == "false" || f.DefValue == ""
	case "duration":
		// Beginning in Go 1.7, duration zero values are "0s"
		return f.DefValue == "0" || f.DefValue == "0s"
	case "int", "int8", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "count", "float32", "float64":
		return f.DefValue == "0"
	case "string":
		return f.DefValue == ""
	case "ip", "ipMask", "ipNet":
		return f.DefValue == "<nil>"
	case "intSlice", "stringSlice", "stringArray":
		return f.DefValue == "[]"
	default:
		switch f.DefValue {
		case "false", "<nil>", "", "0":
			return true
		}
		return false
	}
}

// flagWrapN splits the string `s` on whitespace into an initial substring up to
// `i` runes in length and the remainder. Will go `slop` over `i` if
// that encompasses the entire string (which allows the caller to
// avoid short orphan words on the final line).
// Vendored from pflag since it's unexported.
func flagWrapN(i, slop int, s string) (string, string) {
	if i+slop > len(s) {
		return s, ""
	}

	w := strings.LastIndexAny(s[:i], " \t\n")
	if w <= 0 {
		return s, ""
	}
	nlPos := strings.LastIndex(s[:i], "\n")
	if nlPos > 0 && nlPos < w {
		return s[:nlPos], s[nlPos+1:]
	}
	return s[:w], s[w+1:]
}

// flagWrap wraps the string `s` to a maximum width `w` with leading indent
// `i`. The first line is not indented (this is assumed to be done by
// caller). Pass `w` == 0 to do no wrapping.
// Vendored from pflag since it's unexported.
func flagWrap(i, w int, s string) string {
	if w == 0 {
		return strings.ReplaceAll(s, "\n", "\n"+strings.Repeat(" ", i))
	}

	// space between indent i and end of line width w into which
	// we should wrap the text.
	wrap := w - i

	var r, l string

	// Not enough space for sensible wrapping. Wrap as a block on
	// the next line instead.
	if wrap < 24 {
		i = 16
		wrap = w - i
		r += "\n" + strings.Repeat(" ", i)
	}
	// If still not enough space then don't even try to wrap.
	if wrap < 24 {
		return strings.ReplaceAll(s, "\n", r)
	}

	// Try to avoid short orphan words on the final line, by
	// allowing wrapN to go a bit over if that would fit in the
	// remainder of the line.
	slop := 5
	wrap -= slop

	// Handle first line, which is indented by the caller (or the
	// special case above)
	l, s = flagWrapN(wrap, slop, s)
	r += strings.ReplaceAll(l, "\n", "\n"+strings.Repeat(" ", i))

	// Now wrap the rest
	for s != "" {
		var t string

		t, s = flagWrapN(wrap, slop, s)
		r += "\n" + strings.Repeat(" ", i) + strings.ReplaceAll(t, "\n", "\n"+strings.Repeat(" ", i))
	}

	return r
}

// wrapText wraps text to fit within maxWidth characters per line.
// It preserves existing newlines and wraps long lines at word boundaries.
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}

	// Split by existing newlines to preserve paragraph structure
	paragraphs := strings.Split(text, "\n")
	var result strings.Builder

	for i, paragraph := range paragraphs {
		if i > 0 {
			result.WriteString("\n")
		}

		// Skip empty lines
		if strings.TrimSpace(paragraph) == "" {
			continue
		}

		// Wrap the paragraph
		wrapped := wrapLine(paragraph, maxWidth)
		result.WriteString(wrapped)
	}

	return result.String()
}

// wrapLine wraps a single line of text to maxWidth.
func wrapLine(line string, maxWidth int) string {
	words := strings.Fields(line)
	if len(words) == 0 {
		return ""
	}

	var result strings.Builder
	currentLineLen := 0

	for _, word := range words {
		wordLen := len(word)

		// First word on the line
		if currentLineLen == 0 {
			result.WriteString(word)
			currentLineLen = wordLen
			continue
		}

		// Check if adding this word would exceed the limit
		if currentLineLen+1+wordLen > maxWidth {
			// Start a new line
			result.WriteString("\n")
			result.WriteString(word)
			currentLineLen = wordLen
		} else {
			// Add to current line
			result.WriteString(" ")
			result.WriteString(word)
			currentLineLen += 1 + wordLen
		}
	}

	return result.String()
}
