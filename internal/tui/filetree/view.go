package filetree

import (
	"fmt"
	"strings"
)

// View renders the file tree as a string. This is a placeholder implementation;
// full lipgloss styling will be added in T-081.
func (m Model) View() string {
	if !m.ready {
		return "  Loading file tree..."
	}

	if len(m.visible) == 0 {
		return "  (empty directory)"
	}

	var b strings.Builder

	start := m.offset
	if start < 0 {
		start = 0
	}

	var end int
	if m.height > 0 {
		end = start + m.height
	} else {
		end = len(m.visible)
	}
	if end > len(m.visible) {
		end = len(m.visible)
	}

	for i := start; i < end; i++ {
		node := m.visible[i]

		// Cursor indicator.
		if i == m.cursor {
			b.WriteString("> ")
		} else {
			b.WriteString("  ")
		}

		// Indentation based on depth.
		b.WriteString(strings.Repeat("  ", node.Depth()))

		// Inclusion state indicator.
		b.WriteString(inclusionIndicator(node.Included))
		b.WriteString(" ")

		// Directory or file indicator.
		if node.IsDir {
			if m.loading[node.Path] {
				b.WriteString("  Loading...")
			} else if node.Expanded {
				fmt.Fprintf(&b, "%s %s/", dirExpandedIcon, node.Name)
			} else {
				fmt.Fprintf(&b, "%s %s/", dirCollapsedIcon, node.Name)
			}
		} else {
			b.WriteString(node.Name)
		}

		b.WriteString("\n")
	}

	return b.String()
}

const (
	dirCollapsedIcon = "▸"
	dirExpandedIcon  = "▾"
)

// inclusionIndicator returns the display string for a given inclusion state.
func inclusionIndicator(state InclusionState) string {
	switch state {
	case Included:
		return "[+]"
	case Excluded:
		return "[ ]"
	case Partial:
		return "[~]"
	default:
		return "[?]"
	}
}
