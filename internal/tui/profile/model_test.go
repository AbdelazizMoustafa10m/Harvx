package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/tui/tuimsg"
)

func TestNew_SingleProfile(t *testing.T) {
	t.Parallel()

	m := New("default", nil)
	assert.Equal(t, "default", m.Current())
	assert.Equal(t, 1, m.Count())
	assert.Equal(t, 0, m.Index())
}

func TestNew_MultipleProfiles(t *testing.T) {
	t.Parallel()

	m := New("minimal", []string{"default", "minimal", "full"})
	assert.Equal(t, "minimal", m.Current())
	assert.Equal(t, 1, m.Index())
	assert.Equal(t, 3, m.Count())
}

func TestNew_ActiveNotInList(t *testing.T) {
	t.Parallel()

	m := New("unknown", []string{"default", "minimal"})
	// Falls back to index 0 since "unknown" is not found.
	assert.Equal(t, "default", m.Current())
	assert.Equal(t, 0, m.Index())
}

func TestNext_SingleProfile(t *testing.T) {
	t.Parallel()

	m := New("default", nil)
	next, cmd := m.Next()
	assert.Equal(t, "default", next.Current())
	assert.Nil(t, cmd, "single profile should not produce a command")
}

func TestNext_CyclesForward(t *testing.T) {
	t.Parallel()

	m := New("default", []string{"default", "minimal", "full"})
	assert.Equal(t, "default", m.Current())

	m, cmd := m.Next()
	require.NotNil(t, cmd)
	msg := cmd().(tuimsg.ProfileChangedMsg)
	assert.Equal(t, "minimal", msg.ProfileName)
	assert.Equal(t, "minimal", m.Current())

	m, cmd = m.Next()
	require.NotNil(t, cmd)
	msg = cmd().(tuimsg.ProfileChangedMsg)
	assert.Equal(t, "full", msg.ProfileName)
	assert.Equal(t, "full", m.Current())

	m, cmd = m.Next()
	require.NotNil(t, cmd)
	msg = cmd().(tuimsg.ProfileChangedMsg)
	assert.Equal(t, "default", msg.ProfileName)
	assert.Equal(t, "default", m.Current(), "should wrap around")
}

func TestPrev_SingleProfile(t *testing.T) {
	t.Parallel()

	m := New("default", nil)
	prev, cmd := m.Prev()
	assert.Equal(t, "default", prev.Current())
	assert.Nil(t, cmd, "single profile should not produce a command")
}

func TestPrev_CyclesBackward(t *testing.T) {
	t.Parallel()

	m := New("default", []string{"default", "minimal", "full"})
	assert.Equal(t, "default", m.Current())

	m, cmd := m.Prev()
	require.NotNil(t, cmd)
	msg := cmd().(tuimsg.ProfileChangedMsg)
	assert.Equal(t, "full", msg.ProfileName)
	assert.Equal(t, "full", m.Current(), "should wrap to last")

	m, cmd = m.Prev()
	require.NotNil(t, cmd)
	msg = cmd().(tuimsg.ProfileChangedMsg)
	assert.Equal(t, "minimal", msg.ProfileName)
	assert.Equal(t, "minimal", m.Current())
}

func TestUpdate_ProfileChangedMsg(t *testing.T) {
	t.Parallel()

	m := New("default", []string{"default", "minimal", "full"})
	assert.Equal(t, 0, m.Index())

	updated, _ := m.Update(tuimsg.ProfileChangedMsg{ProfileName: "full"})
	m = updated.(Model)
	assert.Equal(t, "full", m.Current())
	assert.Equal(t, 2, m.Index())
}

func TestRenderHeader_SingleProfile(t *testing.T) {
	t.Parallel()

	m := New("default", nil)
	header := RenderHeader(m, "claude")
	assert.Contains(t, header, "Profile:")
	assert.Contains(t, header, "default")
	assert.Contains(t, header, "Target:")
	assert.Contains(t, header, "claude")
}

func TestRenderHeader_MultipleProfiles(t *testing.T) {
	t.Parallel()

	m := New("minimal", []string{"default", "minimal", "full"})
	header := RenderHeader(m, "")
	assert.Contains(t, header, "minimal")
	assert.Contains(t, header, "2/3")
}

func TestRenderHeader_NoTarget(t *testing.T) {
	t.Parallel()

	m := New("default", nil)
	header := RenderHeader(m, "")
	assert.Contains(t, header, "default")
	assert.NotContains(t, header, "Target:")
}
