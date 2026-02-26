package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LayoutParams holds everything needed to render the full TUI layout.
// The caller populates it from the root Model's state and the pre-computed
// Styles value.
type LayoutParams struct {
	// Styles is the pre-computed style set for the current terminal size
	// and theme. It contains panel dimensions, border colors, and lipgloss
	// styles.
	Styles Styles

	// FileTreeView is the rendered string from the file tree sub-model.
	FileTreeView string

	// StatsView is the rendered string from the stats panel sub-model.
	StatsView string

	// TitleBar is the pre-rendered title bar string.
	TitleBar string

	// StatusBar is the pre-rendered status bar string.
	StatusBar string

	// Mode is the layout mode derived from terminal dimensions.
	Mode LayoutMode

	// Width is the terminal width in columns.
	Width int

	// Height is the terminal height in rows.
	Height int
}

// RenderLayout composes the full TUI view from sub-model views. It arranges
// panels according to the layout mode, joining them horizontally for two-panel
// modes or rendering only the file tree in single-panel mode. A "terminal too
// small" message is shown when the terminal is below minimum size.
func RenderLayout(p LayoutParams) string {
	if p.Mode == LayoutTooSmall {
		return RenderTooSmall(p.Width, p.Height)
	}

	var mainView string

	switch p.Mode {
	case LayoutSinglePanel:
		mainView = renderSinglePanel(p)
	case LayoutCompressed, LayoutFull:
		mainView = renderTwoPanels(p)
	default:
		mainView = p.FileTreeView
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		p.TitleBar,
		mainView,
		p.StatusBar,
	)
}

// renderSinglePanel renders only the file tree panel for narrow terminals.
func renderSinglePanel(p LayoutParams) string {
	isDark := isDarkTheme(p.Styles.Colors)
	return RenderPanelWithBorder(
		p.FileTreeView,
		"Files",
		p.Styles.LeftPanelWidth,
		p.Styles.ContentHeight,
		p.Styles.Colors.BorderActive,
		isDark,
	)
}

// renderTwoPanels renders the file tree and stats side by side with a
// vertical separator between them.
func renderTwoPanels(p LayoutParams) string {
	isDark := isDarkTheme(p.Styles.Colors)

	leftPanel := RenderPanelWithBorder(
		p.FileTreeView,
		"Files",
		p.Styles.LeftPanelWidth,
		p.Styles.ContentHeight,
		p.Styles.Colors.BorderActive,
		isDark,
	)

	rightPanel := RenderPanelWithBorder(
		p.StatsView,
		"Stats",
		p.Styles.RightPanelWidth,
		p.Styles.ContentHeight,
		p.Styles.Colors.BorderInactive,
		isDark,
	)

	separator := renderVerticalSeparator(p.Styles.ContentHeight, p.Styles.Colors.BorderInactive)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, separator, rightPanel)
}

// isDarkTheme infers whether the theme is dark by checking if the background
// color is a low ANSI 256 value. The dark palette uses "235" for background
// while the light palette uses "253".
func isDarkTheme(colors ThemeColors) bool {
	return colors.Background == lipgloss.Color("235")
}

// renderVerticalSeparator creates a single-column vertical border line of
// the specified height using the box-drawing character "│".
func renderVerticalSeparator(height int, color lipgloss.Color) string {
	if height < 1 {
		height = 1
	}

	style := lipgloss.NewStyle().
		Foreground(color).
		Width(1)

	lines := make([]string, height)
	for i := range lines {
		lines[i] = "│"
	}
	return style.Render(strings.Join(lines, "\n"))
}

// RenderPanelWithBorder wraps content in a bordered panel with an inline
// title in the top border. The frame uses rounded Unicode box-drawing
// characters:
//
//	╭─ Title ──────────╮
//	│ content           │
//	╰───────────────────╯
//
// Content inside the panel has 1 character of horizontal padding.
func RenderPanelWithBorder(content, title string, width, height int, borderColor lipgloss.Color, isDark bool) string {
	if width < 4 {
		width = 4
	}
	if height < 3 {
		height = 3
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Foreground(borderColor).Bold(true)

	// Inner width excludes the left and right border characters (2) and
	// 1 char padding on each side (2) = 4 total.
	innerWidth := width - 4
	if innerWidth < 1 {
		innerWidth = 1
	}

	// Build the top border with inline title: "╭─ Title ───...─╮"
	topBorder := buildTopBorder(title, width, borderStyle, titleStyle)

	// Build the bottom border: "╰───...───╯"
	bottomFill := width - 2 // exclude corners
	if bottomFill < 0 {
		bottomFill = 0
	}
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", bottomFill) + "╯")

	// Content lines: split, pad/truncate, and wrap with side borders.
	// The number of content lines is the total height minus top and bottom
	// border rows.
	contentLines := height - 2
	if contentLines < 1 {
		contentLines = 1
	}

	contentFg := lipgloss.Color("252")
	if !isDark {
		contentFg = lipgloss.Color("234")
	}
	contentStyle := lipgloss.NewStyle().
		Foreground(contentFg).
		Width(innerWidth)

	// Render the content with fixed width so lipgloss handles wrapping and
	// truncation, then split into individual lines.
	rendered := contentStyle.Render(content)
	rawLines := strings.Split(rendered, "\n")

	// Pad or truncate to exactly contentLines rows.
	paddedLines := make([]string, contentLines)
	for i := 0; i < contentLines; i++ {
		if i < len(rawLines) {
			paddedLines[i] = rawLines[i]
		} else {
			paddedLines[i] = ""
		}
	}

	// Wrap each line with side borders and 1-char padding.
	bodyRows := make([]string, 0, contentLines)
	leftBorder := borderStyle.Render("│")
	rightBorder := borderStyle.Render("│")

	for _, line := range paddedLines {
		// Measure the visual width of the line to calculate right padding.
		lineWidth := lipgloss.Width(line)
		rightPad := innerWidth - lineWidth
		if rightPad < 0 {
			rightPad = 0
		}
		row := leftBorder + " " + line + strings.Repeat(" ", rightPad) + " " + rightBorder
		bodyRows = append(bodyRows, row)
	}

	body := strings.Join(bodyRows, "\n")

	return topBorder + "\n" + body + "\n" + bottomBorder
}

// buildTopBorder constructs the top border with an inline title. The format
// is: "╭─ Title ─────...──╮" where dashes fill the remaining width.
func buildTopBorder(title string, width int, borderStyle, titleStyle lipgloss.Style) string {
	if title == "" {
		fill := width - 2
		if fill < 0 {
			fill = 0
		}
		return borderStyle.Render("╭" + strings.Repeat("─", fill) + "╮")
	}

	// "╭─ " prefix (3 rendered chars) + title + " " suffix (1 char) + remaining dashes + "╮" (1 char)
	prefix := borderStyle.Render("╭─ ")
	titleRendered := titleStyle.Render(title)
	suffix := borderStyle.Render(" ")

	// Calculate remaining width for dash fill.
	// Visual width: 3 (prefix) + title length + 1 (space after title) + fill + 1 (╮)
	titleVisualWidth := lipgloss.Width(title)
	usedWidth := 3 + titleVisualWidth + 1 + 1 // prefix + title + space + closing corner
	remaining := width - usedWidth
	if remaining < 0 {
		remaining = 0
	}

	fill := borderStyle.Render(strings.Repeat("─", remaining) + "╮")

	return prefix + titleRendered + suffix + fill
}

// RenderTooSmall renders a centered warning message when the terminal is
// below the minimum usable size (40 columns by 12 rows).
func RenderTooSmall(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	msg := "Terminal too small (min: 40\u00d712)\nResize your terminal to continue."

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("208")).
		Bold(true)

	styledMsg := style.Render(msg)

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		styledMsg,
	)
}
