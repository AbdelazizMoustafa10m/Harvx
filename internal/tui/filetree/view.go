package filetree

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ViewOptions controls the visual rendering of the file tree.
type ViewOptions struct {
	// IsDark selects dark or light theme colors.
	IsDark bool
	// Width is the available width in columns.
	Width int
}

// View renders the file tree as a styled string. This implements virtual
// scrolling: only nodes within the viewport (offset to offset+height) are
// rendered, which is critical for large repos.
func (m Model) View() string {
	if !m.ready {
		return "  Loading file tree..."
	}

	if len(m.visible) == 0 {
		return "  (empty directory)"
	}

	colors := m.viewColors()

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
		isCursor := i == m.cursor

		line := m.renderNode(node, i, colors)

		if isCursor {
			// Apply cursor highlight background to the entire line.
			cursorStyle := lipgloss.NewStyle().
				Background(colors.cursorBg)
			// Pad line to full width so background fills the row.
			lineWidth := lipgloss.Width(line)
			if m.width > 0 && lineWidth < m.width {
				line = line + strings.Repeat(" ", m.width-lineWidth)
			}
			line = cursorStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

// viewColors holds resolved lipgloss colors for the current theme.
type viewColors struct {
	includedGreen lipgloss.Color
	excludedGray  lipgloss.Color
	partialYellow lipgloss.Color
	cursorBg      lipgloss.Color
	secretRed     lipgloss.Color
	tokenCountDim lipgloss.Color
	tierGold      lipgloss.Color
	tierGreen     lipgloss.Color
	tierBlue      lipgloss.Color
	tierCyan      lipgloss.Color
	tierMagenta   lipgloss.Color
	tierDim       lipgloss.Color
	foreground    lipgloss.Color
}

// darkViewColors returns colors for dark terminal backgrounds.
func darkViewColors() viewColors {
	return viewColors{
		includedGreen: lipgloss.Color("34"),
		excludedGray:  lipgloss.Color("240"),
		partialYellow: lipgloss.Color("220"),
		cursorBg:      lipgloss.Color("238"),
		secretRed:     lipgloss.Color("196"),
		tokenCountDim: lipgloss.Color("242"),
		tierGold:      lipgloss.Color("220"),
		tierGreen:     lipgloss.Color("34"),
		tierBlue:      lipgloss.Color("33"),
		tierCyan:      lipgloss.Color("36"),
		tierMagenta:   lipgloss.Color("133"),
		tierDim:       lipgloss.Color("240"),
		foreground:    lipgloss.Color("252"),
	}
}

// lightViewColors returns colors for light terminal backgrounds.
func lightViewColors() viewColors {
	return viewColors{
		includedGreen: lipgloss.Color("28"),
		excludedGray:  lipgloss.Color("246"),
		partialYellow: lipgloss.Color("172"),
		cursorBg:      lipgloss.Color("254"),
		secretRed:     lipgloss.Color("160"),
		tokenCountDim: lipgloss.Color("244"),
		tierGold:      lipgloss.Color("172"),
		tierGreen:     lipgloss.Color("28"),
		tierBlue:      lipgloss.Color("27"),
		tierCyan:      lipgloss.Color("30"),
		tierMagenta:   lipgloss.Color("127"),
		tierDim:       lipgloss.Color("246"),
		foreground:    lipgloss.Color("237"),
	}
}

// viewColors returns the color set for the model's current theme.
func (m Model) viewColors() viewColors {
	if m.isDark {
		return darkViewColors()
	}
	return lightViewColors()
}

// renderNode renders a single node line with tree prefix, inclusion indicator,
// name with tier color, and optional suffixes (priority star, secret shield,
// token count).
func (m Model) renderNode(node *Node, idx int, colors viewColors) string {
	var parts []string

	// 1. Tree-drawing prefix.
	prefix := m.treePrefix(node)
	treePrefixStyle := lipgloss.NewStyle().Foreground(colors.excludedGray)
	parts = append(parts, treePrefixStyle.Render(prefix))

	// 2. Inclusion indicator.
	parts = append(parts, renderInclusionIndicator(node.Included, colors))
	parts = append(parts, " ")

	// 3. Directory/file icon and name.
	if node.IsDir {
		if m.loading[node.Path] {
			dimStyle := lipgloss.NewStyle().Foreground(colors.tokenCountDim)
			parts = append(parts, dimStyle.Render("⏳ Loading..."))
		} else {
			var icon string
			if node.Expanded {
				icon = DirExpanded
			} else {
				icon = DirCollapsed
			}
			dirStyle := lipgloss.NewStyle().
				Foreground(colors.foreground).
				Bold(true)
			parts = append(parts, dirStyle.Render(icon+" "+node.Name+"/"))
		}
	} else {
		// File name with tier color coding.
		nameStyle := lipgloss.NewStyle().
			Foreground(tierColor(node.Tier, colors))
		parts = append(parts, nameStyle.Render(node.Name))

		// Priority star.
		if node.IsPriority {
			starStyle := lipgloss.NewStyle().Foreground(colors.tierGold)
			parts = append(parts, " ", starStyle.Render(PriorityIcon))
		}

		// Secret shield.
		if node.HasSecrets {
			shieldStyle := lipgloss.NewStyle().Foreground(colors.secretRed)
			parts = append(parts, " ", shieldStyle.Render(SecretIcon))
		}

		// Token count.
		if node.TokenCount > 0 {
			tokenStr := fmt.Sprintf("(%s tok)", formatThousands(node.TokenCount))
			tokenStyle := lipgloss.NewStyle().Foreground(colors.tokenCountDim)
			parts = append(parts, " ", tokenStyle.Render(tokenStr))
		}
	}

	line := strings.Join(parts, "")

	// Truncate long lines with ellipsis if needed.
	if m.width > 0 {
		lineWidth := lipgloss.Width(line)
		if lineWidth > m.width {
			line = truncateWithEllipsis(line, m.width)
		}
	}

	return line
}

// treePrefix generates the Unicode tree-drawing prefix for a node based on
// its position in the tree. It walks up the parent chain to build the indent
// stack, then renders the correct characters.
func (m Model) treePrefix(node *Node) string {
	if node.Depth() == 0 {
		// Top-level nodes don't get tree lines.
		return ""
	}

	// Build prefix by walking up the tree.
	// We need to know for each depth level whether the ancestor is the last
	// child of its parent. The last level gets "└── " or "├── ", and each
	// prior level gets "│   " or "    ".
	depth := node.Depth()
	isLast := make([]bool, depth)

	current := node
	for d := depth - 1; d >= 0; d-- {
		if current.Parent != nil {
			children := current.Parent.Children
			isLast[d] = len(children) > 0 && children[len(children)-1] == current
		}
		current = current.Parent
	}

	var prefix strings.Builder
	for d := 0; d < depth-1; d++ {
		if isLast[d] {
			prefix.WriteString(TreeSpace)
		} else {
			prefix.WriteString(TreePipe)
		}
	}

	// Last segment.
	if isLast[depth-1] {
		prefix.WriteString(TreeLast)
	} else {
		prefix.WriteString(TreeBranch)
	}

	return prefix.String()
}

// renderInclusionIndicator returns a styled inclusion indicator string.
func renderInclusionIndicator(state InclusionState, colors viewColors) string {
	switch state {
	case Included:
		style := lipgloss.NewStyle().Foreground(colors.includedGreen)
		return style.Render(IncludedIcon)
	case Excluded:
		style := lipgloss.NewStyle().Foreground(colors.excludedGray)
		return style.Render(ExcludedIcon)
	case Partial:
		style := lipgloss.NewStyle().Foreground(colors.partialYellow)
		return style.Render(PartialIcon)
	default:
		return "[?]"
	}
}

// tierColor returns the lipgloss color for a given relevance tier.
func tierColor(tier int, colors viewColors) lipgloss.Color {
	switch tier {
	case 0:
		return colors.tierGold
	case 1:
		return colors.tierGreen
	case 2:
		return colors.tierBlue
	case 3:
		return colors.tierCyan
	case 4:
		return colors.tierMagenta
	case 5:
		return colors.tierDim
	default:
		return colors.foreground
	}
}

// formatThousands formats an integer with thousands separators.
func formatThousands(n int) string {
	if n < 0 {
		return "-" + formatThousands(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	remainder := len(s) % 3
	if remainder > 0 {
		result.WriteString(s[:remainder])
	}
	for i := remainder; i < len(s); i += 3 {
		if result.Len() > 0 {
			result.WriteByte(',')
		}
		result.WriteString(s[i : i+3])
	}
	return result.String()
}

// truncateWithEllipsis truncates a string to fit within maxWidth columns,
// appending "..." if truncated. It operates on runes to handle multi-byte
// correctly but uses lipgloss.Width for visual width measurement.
func truncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return "..."[:maxWidth]
	}
	targetWidth := maxWidth - 3 // room for "..."

	runes := []rune(s)
	var result strings.Builder
	for _, r := range runes {
		result.WriteRune(r)
		if lipgloss.Width(result.String()) >= targetWidth {
			return result.String() + "..."
		}
	}
	return s
}
