package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// overlayState represents which overlay is currently active.
type overlayState int

const (
	overlayNone overlayState = iota
	overlayGenerating
	overlayPreviewing
	overlaySavingProfile
)

// overlayModel manages the state for all overlay types: spinner (generating),
// preview summary, and text input (save-as-profile).
type overlayModel struct {
	state   overlayState
	spinner spinner.Model
	input   textinput.Model

	// Preview data.
	previewFileCount int
	previewTokens    int
	previewMaxTokens int
	previewBudget    float64
	previewTiers     map[int]int // tier -> file count
	previewTierToks  map[int]int // tier -> token count
}

// newOverlayModel creates a new overlay model with a configured spinner and
// text input component.
func newOverlayModel() overlayModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "profile-name"
	ti.CharLimit = 64
	ti.Width = 30

	return overlayModel{
		state:   overlayNone,
		spinner: s,
		input:   ti,
	}
}

// Active returns true if any overlay is currently visible.
func (m overlayModel) Active() bool {
	return m.state != overlayNone
}

// State returns the current overlay state.
func (m overlayModel) State() overlayState {
	return m.state
}

// startGenerating activates the spinner overlay for pipeline generation.
func (m *overlayModel) startGenerating() tea.Cmd {
	m.state = overlayGenerating
	return m.spinner.Tick
}

// startPreview activates the preview overlay with the given stats.
func (m *overlayModel) startPreview(fileCount, tokens, maxTokens int, budget float64, tiers, tierToks map[int]int) {
	m.state = overlayPreviewing
	m.previewFileCount = fileCount
	m.previewTokens = tokens
	m.previewMaxTokens = maxTokens
	m.previewBudget = budget
	m.previewTiers = tiers
	m.previewTierToks = tierToks
}

// startSaveProfile activates the text input overlay for naming a new profile.
func (m *overlayModel) startSaveProfile() tea.Cmd {
	m.state = overlaySavingProfile
	m.input.Reset()
	focusCmd := m.input.Focus()
	return focusCmd
}

// close dismisses the current overlay.
func (m *overlayModel) close() {
	m.state = overlayNone
	m.input.Blur()
}

// update handles messages for the currently active overlay.
func (m *overlayModel) update(msg tea.Msg) tea.Cmd {
	switch m.state {
	case overlayGenerating:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return cmd

	case overlaySavingProfile:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	}
	return nil
}

// view renders the overlay content centered in the given dimensions.
func (m overlayModel) view(width, height int) string {
	switch m.state {
	case overlayGenerating:
		return m.renderGenerating(width, height)
	case overlayPreviewing:
		return m.renderPreview(width, height)
	case overlaySavingProfile:
		return m.renderSaveProfile(width, height)
	default:
		return ""
	}
}

// renderGenerating renders the spinner overlay.
func (m overlayModel) renderGenerating(width, height int) string {
	content := fmt.Sprintf("%s Generating output...", m.spinner.View())

	return centerOverlay(content, width, height)
}

// renderPreview renders the preview summary overlay.
func (m overlayModel) renderPreview(width, height int) string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("117")).
		Bold(true)

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	b.WriteString(headerStyle.Render("Output Preview"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 30)))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Files:  "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", m.previewFileCount)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Tokens: "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d / %d", m.previewTokens, m.previewMaxTokens)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Budget: "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%.1f%%", m.previewBudget)))
	b.WriteString("\n\n")

	// Tier breakdown.
	tierLabels := map[int]string{
		0: "critical",
		1: "primary",
		2: "secondary",
		3: "tests",
		4: "docs",
		5: "low",
	}

	b.WriteString(headerStyle.Render("Tier Breakdown"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 30)))
	b.WriteString("\n")

	for tier := 0; tier <= 5; tier++ {
		count := m.previewTiers[tier]
		tokens := m.previewTierToks[tier]
		label := tierLabels[tier]
		tierName := fmt.Sprintf("T%d (%s)", tier, label)

		if count == 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %-16s    --      --", tierName)))
		} else {
			b.WriteString(labelStyle.Render(fmt.Sprintf("  %-16s", tierName)))
			b.WriteString(valueStyle.Render(fmt.Sprintf(" %4d  %6d tok", count, tokens)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Press Esc to close"))

	return centerOverlay(b.String(), width, height)
}

// renderSaveProfile renders the text input overlay for naming a new profile.
func (m overlayModel) renderSaveProfile(width, height int) string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("117")).
		Bold(true)

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	b.WriteString(headerStyle.Render("Save Selection as Profile"))
	b.WriteString("\n\n")
	b.WriteString("Profile name: ")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Enter to confirm, Esc to cancel"))

	return centerOverlay(b.String(), width, height)
}

// centerOverlay renders content within a centered bordered box.
func centerOverlay(content string, width, height int) string {
	boxWidth := 50
	if boxWidth > width-4 {
		boxWidth = width - 4
	}
	if boxWidth < 20 {
		boxWidth = 20
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(content)

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}
