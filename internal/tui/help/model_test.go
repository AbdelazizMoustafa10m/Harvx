package help

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	m := New()
	assert.False(t, m.Visible)
}

func TestToggle(t *testing.T) {
	t.Parallel()

	m := New()
	m = m.Toggle()
	assert.True(t, m.Visible)

	m = m.Toggle()
	assert.False(t, m.Visible)
}

func TestUpdate_DismissWithQuestion(t *testing.T) {
	t.Parallel()

	m := New()
	m.Visible = true

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.False(t, m.Visible)
	assert.Nil(t, cmd)
}

func TestUpdate_DismissWithEsc(t *testing.T) {
	t.Parallel()

	m := New()
	m.Visible = true

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, m.Visible)
	assert.Nil(t, cmd)
}

func TestUpdate_InvisibleNoOp(t *testing.T) {
	t.Parallel()

	m := New()
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.False(t, m.Visible)
	assert.Nil(t, cmd)
}

func TestUpdate_IgnoresOtherKeys(t *testing.T) {
	t.Parallel()

	m := New()
	m.Visible = true

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.True(t, m.Visible, "other keys should not dismiss help")
	assert.Nil(t, cmd)
}

func TestDefaultCategories(t *testing.T) {
	t.Parallel()

	categories := DefaultCategories()
	assert.NotEmpty(t, categories)

	// Check expected category names.
	names := make([]string, len(categories))
	for i, c := range categories {
		names[i] = c.Title
	}

	assert.Contains(t, names, "Navigation")
	assert.Contains(t, names, "Selection")
	assert.Contains(t, names, "Filtering")
	assert.Contains(t, names, "Profiles")
	assert.Contains(t, names, "Actions")
}

func TestView_ContainsCategories(t *testing.T) {
	t.Parallel()

	view := View(80, 40, true)

	assert.Contains(t, view, "Harvx Interactive Mode")
	assert.Contains(t, view, "Navigation")
	assert.Contains(t, view, "Selection")
	assert.Contains(t, view, "Filtering")
	assert.Contains(t, view, "Profiles")
	assert.Contains(t, view, "Actions")
	assert.Contains(t, view, "Press ? or Esc to close")
}

func TestView_ContainsKeyBindings(t *testing.T) {
	t.Parallel()

	view := View(80, 40, true)

	// Check that specific key bindings are present.
	assert.Contains(t, view, "Move up")
	assert.Contains(t, view, "Toggle include")
	assert.Contains(t, view, "Search files")
	assert.Contains(t, view, "Cycle tier view")
	assert.Contains(t, view, "Generate output")
}

func TestView_LightMode(t *testing.T) {
	t.Parallel()

	view := View(80, 40, false)
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Harvx Interactive Mode")
}

func TestView_SmallTerminal(t *testing.T) {
	t.Parallel()

	// Should not panic on small terminals.
	view := View(30, 20, true)
	assert.NotEmpty(t, view)
}
