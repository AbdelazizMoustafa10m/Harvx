package help

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Category groups related keybindings for display.
type Category struct {
	Title    string
	Bindings []Binding
}

// Binding represents a single keybinding entry.
type Binding struct {
	Key  string
	Desc string
}

// DefaultCategories returns the standard help categories for the TUI.
func DefaultCategories() []Category {
	return []Category{
		{
			Title: "Navigation",
			Bindings: []Binding{
				{"up/k", "Move up"},
				{"down/j", "Move down"},
				{"left/h", "Collapse / parent"},
				{"right/l", "Expand"},
				{"PgUp/PgDn", "Page up / down"},
				{"Home/g", "First item"},
				{"End/G", "Last item"},
			},
		},
		{
			Title: "Selection",
			Bindings: []Binding{
				{"Space", "Toggle include/exclude"},
				{"a", "Select all visible"},
				{"n", "Deselect all visible"},
			},
		},
		{
			Title: "Filtering",
			Bindings: []Binding{
				{"/", "Search files"},
				{"t", "Cycle tier view"},
				{"Ctrl+L", "Clear filter"},
			},
		},
		{
			Title: "Profiles",
			Bindings: []Binding{
				{"Tab", "Next profile"},
				{"Shift+Tab", "Previous profile"},
			},
		},
		{
			Title: "Actions",
			Bindings: []Binding{
				{"Enter", "Generate output"},
				{"p", "Preview stats"},
				{"s", "Save as profile"},
				{"e", "Export to clipboard"},
				{"q/Esc", "Quit"},
			},
		},
	}
}

// View renders the help overlay as a centered bordered box. The isDark flag
// selects the color palette. Width and height are the terminal dimensions.
func View(width, height int, isDark bool) string {
	categories := DefaultCategories()

	headerStyle := lipgloss.NewStyle().Bold(true)
	if isDark {
		headerStyle = headerStyle.Foreground(lipgloss.Color("117"))
	} else {
		headerStyle = headerStyle.Foreground(lipgloss.Color("33"))
	}

	categoryStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	if isDark {
		categoryStyle = categoryStyle.Foreground(lipgloss.Color("215"))
	} else {
		categoryStyle = categoryStyle.Foreground(lipgloss.Color("172"))
	}

	keyStyle := lipgloss.NewStyle().Bold(true)
	if isDark {
		keyStyle = keyStyle.Foreground(lipgloss.Color("252"))
	} else {
		keyStyle = keyStyle.Foreground(lipgloss.Color("237"))
	}

	descStyle := lipgloss.NewStyle()
	if isDark {
		descStyle = descStyle.Foreground(lipgloss.Color("248"))
	} else {
		descStyle = descStyle.Foreground(lipgloss.Color("240"))
	}

	dimStyle := lipgloss.NewStyle()
	if isDark {
		dimStyle = dimStyle.Foreground(lipgloss.Color("240"))
	} else {
		dimStyle = dimStyle.Foreground(lipgloss.Color("246"))
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("Harvx Interactive Mode"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("\u2500", 36)))
	b.WriteString("\n\n")

	for i, cat := range categories {
		b.WriteString(categoryStyle.Render(cat.Title))
		b.WriteString("\n")
		for _, bind := range cat.Bindings {
			fmt.Fprintf(&b, "  %s  %s\n",
				keyStyle.Render(fmt.Sprintf("%-12s", bind.Key)),
				descStyle.Render(bind.Desc),
			)
		}
		if i < len(categories)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Press ? or Esc to close"))

	boxWidth := 50
	if boxWidth > width-4 {
		boxWidth = width - 4
	}
	if boxWidth < 30 {
		boxWidth = 30
	}

	borderColor := lipgloss.Color("63")
	if !isDark {
		borderColor = lipgloss.Color("33")
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(b.String())

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}
