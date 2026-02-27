package search

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HighlightMatches renders a path string with matched character positions
// highlighted using the provided style. Non-matched characters use the
// normal style.
func HighlightMatches(path string, indices []int, highlightStyle, normalStyle lipgloss.Style) string {
	if len(indices) == 0 {
		return normalStyle.Render(path)
	}

	matchSet := make(map[int]bool, len(indices))
	for _, idx := range indices {
		matchSet[idx] = true
	}

	var b strings.Builder
	runes := []rune(path)
	for i, r := range runes {
		if matchSet[i] {
			b.WriteString(highlightStyle.Render(string(r)))
		} else {
			b.WriteString(normalStyle.Render(string(r)))
		}
	}
	return b.String()
}

// RenderFilterIndicator returns a styled "Filter: <query>" string for the
// panel header. Returns an empty string if query is empty.
func RenderFilterIndicator(query string, style lipgloss.Style) string {
	if query == "" {
		return ""
	}
	return style.Render("Filter: " + query)
}
