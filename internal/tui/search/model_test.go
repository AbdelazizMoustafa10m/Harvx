package search

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	m := New()
	assert.False(t, m.Active())
	assert.False(t, m.Filtered())
	assert.Empty(t, m.Query())
}

func TestActivate(t *testing.T) {
	t.Parallel()

	m := New()
	m, cmd := m.Activate()

	assert.True(t, m.Active())
	assert.NotNil(t, cmd, "activate should return a focus command")
}

func TestDeactivate_KeepFilter(t *testing.T) {
	t.Parallel()

	m := New()
	m, _ = m.Activate()

	// Type a query.
	m.input.SetValue("main")
	m, cmd := m.Deactivate(true)

	assert.False(t, m.Active())
	assert.True(t, m.Filtered())
	assert.Equal(t, "main", m.Query())
	require.NotNil(t, cmd)

	// Execute the command to get the message.
	msg := cmd()
	applied, ok := msg.(FilterAppliedMsg)
	require.True(t, ok)
	assert.Equal(t, "main", applied.Query)
	assert.True(t, applied.Active)
}

func TestDeactivate_ClearFilter(t *testing.T) {
	t.Parallel()

	m := New()
	m, _ = m.Activate()
	m.input.SetValue("main")

	m, cmd := m.Deactivate(false)

	assert.False(t, m.Active())
	assert.False(t, m.Filtered())
	assert.Empty(t, m.Query())
	require.NotNil(t, cmd)

	msg := cmd()
	applied, ok := msg.(FilterAppliedMsg)
	require.True(t, ok)
	assert.Empty(t, applied.Query)
	assert.False(t, applied.Active)
}

func TestClearFilter(t *testing.T) {
	t.Parallel()

	m := New()
	m, _ = m.Activate()
	m.input.SetValue("test")
	m, _ = m.Deactivate(true)
	assert.True(t, m.Filtered())

	m, cmd := m.ClearFilter()
	assert.False(t, m.Active())
	assert.False(t, m.Filtered())
	assert.Empty(t, m.Query())
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(FilterClearedMsg)
	assert.True(t, ok)
}

func TestUpdate_EnterKeepsFilter(t *testing.T) {
	t.Parallel()

	m := New()
	m, _ = m.Activate()
	m.input.SetValue("util")

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.False(t, m.Active())
	assert.True(t, m.Filtered())
	assert.Equal(t, "util", m.Query())
	require.NotNil(t, cmd)
}

func TestUpdate_EscClearsFilter(t *testing.T) {
	t.Parallel()

	m := New()
	m, _ = m.Activate()
	m.input.SetValue("util")

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})

	assert.False(t, m.Active())
	assert.False(t, m.Filtered())
	assert.Empty(t, m.Query())
	require.NotNil(t, cmd)
}

func TestUpdate_InactiveNoOp(t *testing.T) {
	t.Parallel()

	m := New()
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	assert.False(t, m.Active())
	assert.Nil(t, cmd)
}

func TestUpdate_LiveFilter(t *testing.T) {
	t.Parallel()

	m := New()
	m, _ = m.Activate()

	// Typing updates the query in real-time.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	assert.Equal(t, "m", m.Query())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Equal(t, "ma", m.Query())
}

func TestView_InactiveEmpty(t *testing.T) {
	t.Parallel()

	m := New()
	assert.Empty(t, m.View())
}

func TestView_ActiveShowsInput(t *testing.T) {
	t.Parallel()

	m := New()
	m, _ = m.Activate()

	view := m.View()
	assert.NotEmpty(t, view)
}
