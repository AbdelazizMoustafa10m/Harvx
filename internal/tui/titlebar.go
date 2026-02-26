package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// appName is the application name displayed in the title bar.
const appName = "Harvx"

// RenderTitleBar renders the top title bar with app name, version, and
// directory. The left side shows "Harvx v<version>" and the right side shows
// the target directory path. The bar spans the full terminal width using
// inverse colors from the provided Styles. If the directory path is too long
// for the available space, it is truncated with a "..." prefix.
func RenderTitleBar(version, dir string, width int, s Styles) string {
	if width <= 0 {
		return ""
	}

	colors := s.Colors

	// Build the left side: "Harvx v1.2.3".
	left := " " + appName
	if version != "" {
		left += " " + version
	}
	leftWidth := lipgloss.Width(left)

	// Build the right side: directory path, possibly truncated.
	right := truncateDir(dir, width-leftWidth-2) // 2 for minimum gap + trailing space
	rightWidth := lipgloss.Width(right)

	// Calculate the gap between left and right sides.
	gap := width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	content := left + strings.Repeat(" ", gap-1) + right + " "

	barStyle := lipgloss.NewStyle().
		Background(colors.TitleBarBg).
		Foreground(colors.TitleBarFg).
		Bold(true).
		Width(width)

	return barStyle.Render(content)
}

// truncateDir truncates a directory path to fit within maxWidth characters.
// If the path is longer than maxWidth, it is truncated with a "..." prefix.
// Returns an empty string if maxWidth is too small.
func truncateDir(dir string, maxWidth int) string {
	if maxWidth <= 0 || dir == "" {
		return ""
	}

	if len(dir) <= maxWidth {
		return dir
	}

	// Need at least 4 chars for "...X".
	if maxWidth < 4 {
		return dir[:maxWidth]
	}

	// Truncate from the left, preserving the rightmost portion of the path.
	return "..." + dir[len(dir)-(maxWidth-3):]
}
