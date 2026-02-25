// Package profile implements the profile selector component for the Harvx TUI.
// It provides Tab/Shift+Tab cycling through available profiles with wraparound.
package profile

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/harvx/harvx/internal/tui/tuimsg"
)

// Compile-time interface compliance check.
var _ tea.Model = Model{}

// Model is the profile selector Bubble Tea model. It maintains a list of
// available profile names and tracks the currently active profile index.
type Model struct {
	// profiles is the ordered list of available profile names.
	profiles []string

	// index is the current position within profiles.
	index int
}

// New creates a new profile selector Model with the given active profile name
// and the full list of available profile names. If profiles is empty or nil,
// the active profile is used as the sole entry.
func New(active string, profiles []string) Model {
	if len(profiles) == 0 {
		profiles = []string{active}
	}

	idx := 0
	for i, p := range profiles {
		if p == active {
			idx = i
			break
		}
	}

	return Model{
		profiles: profiles,
		index:    idx,
	}
}

// Init implements tea.Model. The profile selector has no initialization command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It handles ProfileChangedMsg to sync the active
// profile when changed externally.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tuimsg.ProfileChangedMsg); ok {
		for i, p := range m.profiles {
			if p == msg.ProfileName {
				m.index = i
				break
			}
		}
	}
	return m, nil
}

// View implements tea.Model. Profile rendering is handled by the view package;
// this returns an empty string.
func (m Model) View() string {
	return ""
}

// Current returns the name of the currently active profile.
func (m Model) Current() string {
	if len(m.profiles) == 0 {
		return ""
	}
	return m.profiles[m.index]
}

// Profiles returns the list of all available profile names.
func (m Model) Profiles() []string {
	return m.profiles
}

// Index returns the current profile index.
func (m Model) Index() int {
	return m.index
}

// Count returns the total number of available profiles.
func (m Model) Count() int {
	return len(m.profiles)
}

// Next cycles forward to the next profile, wrapping around at the end.
// It returns the updated model and a ProfileChangedMsg command.
func (m Model) Next() (Model, tea.Cmd) {
	if len(m.profiles) <= 1 {
		return m, nil
	}
	m.index = (m.index + 1) % len(m.profiles)
	name := m.profiles[m.index]
	return m, func() tea.Msg {
		return tuimsg.ProfileChangedMsg{ProfileName: name}
	}
}

// Prev cycles backward to the previous profile, wrapping around at the start.
// It returns the updated model and a ProfileChangedMsg command.
func (m Model) Prev() (Model, tea.Cmd) {
	if len(m.profiles) <= 1 {
		return m, nil
	}
	m.index = (m.index - 1 + len(m.profiles)) % len(m.profiles)
	name := m.profiles[m.index]
	return m, func() tea.Msg {
		return tuimsg.ProfileChangedMsg{ProfileName: name}
	}
}
