package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// statusBarHints is the default set of key hints shown in the status bar.
// Each pair is [key, description]. The separator "│" is inserted between them.
var statusBarHints = [][2]string{
	{"q", "quit"},
	{"?", "help"},
	{"enter", "generate"},
	{"space", "toggle"},
	{"tab", "profile"},
}

// RenderStatusBar renders the bottom status bar with key hints and profile name.
// The profile name is shown on the left with a slight accent; key hints appear
// on the right. The bar spans the full terminal width using inverse colors from
// the provided Styles.
func RenderStatusBar(profileName string, width int, s Styles) string {
	if width <= 0 {
		return ""
	}

	colors := s.Colors

	// Build the left side: profile name with accent color.
	leftStyle := lipgloss.NewStyle().
		Foreground(colors.Accent).
		Background(colors.StatusBarBg).
		Bold(true)

	left := " " + leftStyle.Render(profileName)
	leftWidth := lipgloss.Width(left)

	// Build the right side: key hints joined by "│".
	hints := buildHints(statusBarHints)
	hintsWidth := lipgloss.Width(hints)

	// Determine available space for the gap between left and right.
	// We need at least 1 char of padding on the right edge.
	available := width - leftWidth - 1
	if available < 0 {
		available = 0
	}

	// Truncate hints if the terminal is too narrow.
	if hintsWidth > available {
		hints = truncateHints(statusBarHints, available)
		hintsWidth = lipgloss.Width(hints)
	}

	// Calculate the gap between left and right sides.
	gap := width - leftWidth - hintsWidth
	if gap < 0 {
		gap = 0
	}

	content := left + strings.Repeat(" ", gap) + hints

	barStyle := lipgloss.NewStyle().
		Background(colors.StatusBarBg).
		Foreground(colors.StatusBarFg).
		Width(width)

	return barStyle.Render(content)
}

// buildHints joins all hint pairs into a single string separated by " │ ".
func buildHints(hints [][2]string) string {
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, h[0]+" "+h[1])
	}
	return strings.Join(parts, " │ ") + " "
}

// truncateHints progressively removes hints from the right until the result
// fits within maxWidth. If even a single hint does not fit, an empty string
// is returned.
func truncateHints(hints [][2]string, maxWidth int) string {
	for n := len(hints); n > 0; n-- {
		result := buildHints(hints[:n])
		if lipgloss.Width(result) <= maxWidth {
			return result
		}
	}
	return ""
}
