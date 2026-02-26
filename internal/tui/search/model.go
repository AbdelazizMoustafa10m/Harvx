// Package search implements the search/filter component for the Harvx TUI.
// It provides a text input for fuzzy file path filtering and manages the
// active/inactive state of search mode.
package search

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// FilterAppliedMsg signals the root model that the search filter changed.
type FilterAppliedMsg struct {
	Query  string
	Active bool
}

// FilterClearedMsg signals the filter was cleared.
type FilterClearedMsg struct{}

// Model is the search/filter Bubble Tea sub-model. It wraps a text input
// component and tracks whether a filter is currently active.
type Model struct {
	input    textinput.Model
	active   bool   // search mode is active (text input shown)
	query    string // current filter query (persists after Enter)
	filtered bool   // whether a filter is currently applied
}

// New creates a new search model with a pre-configured text input.
func New() Model {
	ti := textinput.New()
	ti.Placeholder = "Search files..."
	ti.Prompt = "/ "
	ti.CharLimit = 100
	return Model{input: ti}
}

// Active returns whether search mode is currently active (text input visible).
func (m Model) Active() bool { return m.active }

// Query returns the current search/filter query string.
func (m Model) Query() string { return m.query }

// Filtered returns whether a filter is currently applied.
func (m Model) Filtered() bool { return m.filtered }

// Activate enters search mode, focusing the text input.
func (m Model) Activate() (Model, tea.Cmd) {
	m.active = true
	m.input.SetValue(m.query) // pre-fill with existing filter
	cmd := m.input.Focus()
	return m, cmd
}

// Deactivate exits search mode. If keepFilter is true, the current input
// becomes the persistent filter. If false, the filter is cleared.
func (m Model) Deactivate(keepFilter bool) (Model, tea.Cmd) {
	m.active = false
	m.input.Blur()
	if keepFilter {
		m.query = m.input.Value()
		m.filtered = m.query != ""
	} else {
		m.query = ""
		m.filtered = false
		m.input.SetValue("")
	}
	q := m.query
	f := m.filtered
	return m, func() tea.Msg {
		return FilterAppliedMsg{Query: q, Active: f}
	}
}

// ClearFilter removes the current filter and resets the input.
func (m Model) ClearFilter() (Model, tea.Cmd) {
	m.query = ""
	m.filtered = false
	m.input.SetValue("")
	if m.active {
		m.active = false
		m.input.Blur()
	}
	return m, func() tea.Msg {
		return FilterClearedMsg{}
	}
}

// Update handles messages while search mode is active.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m.Deactivate(true) // keep filter
		case tea.KeyEscape:
			return m.Deactivate(false) // clear filter
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			// Live filter as user types.
			m.query = m.input.Value()
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View returns the search input view when active.
func (m Model) View() string {
	if !m.active {
		return ""
	}
	return m.input.View()
}
