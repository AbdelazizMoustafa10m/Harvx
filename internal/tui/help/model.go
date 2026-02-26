// Package help implements the help overlay for the Harvx TUI. It displays
// all available keybindings organized by category in a centered bordered box.
package help

import tea "github.com/charmbracelet/bubbletea"

// Model is the help overlay model. It tracks whether the overlay is visible
// and handles dismiss key events.
type Model struct {
	Visible bool
}

// New creates a new help model that starts hidden.
func New() Model {
	return Model{}
}

// Toggle flips the visibility state.
func (m Model) Toggle() Model {
	m.Visible = !m.Visible
	return m
}

// Update handles key events when the help overlay is visible. Pressing "?"
// or Esc dismisses the overlay.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.Visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?", "esc":
			m.Visible = false
			return m, nil
		}
	}
	return m, nil
}
