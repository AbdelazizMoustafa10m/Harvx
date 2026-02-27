package profile

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Color constants for profile display.
var (
	profileNameBg    = lipgloss.Color("63")  // Distinct purple background
	profileNameFg    = lipgloss.Color("255") // White text
	profileLabelFg   = lipgloss.Color("252") // Light gray label
	profileIndexFg   = lipgloss.Color("240") // Dim gray for index indicator
)

// RenderHeader renders the profile display header line showing the active
// profile name styled with a distinct background color and the profile index
// indicator. Example: "Profile: [finvault] (2/3)"
func RenderHeader(m Model, targetName string) string {
	labelStyle := lipgloss.NewStyle().Foreground(profileLabelFg)
	nameStyle := lipgloss.NewStyle().
		Background(profileNameBg).
		Foreground(profileNameFg).
		Padding(0, 1).
		Bold(true)

	var result string
	result = labelStyle.Render("Profile: ") + nameStyle.Render(m.Current())

	if targetName != "" {
		result += labelStyle.Render(" | Target: ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render(targetName)
	}

	if m.Count() > 1 {
		indexStyle := lipgloss.NewStyle().Foreground(profileIndexFg)
		result += indexStyle.Render(fmt.Sprintf(" (%d/%d)", m.Index()+1, m.Count()))
	}

	return result
}
