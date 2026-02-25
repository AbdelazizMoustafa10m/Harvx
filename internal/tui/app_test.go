package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tui/stats"
)

func validCfg() *config.ResolvedConfig {
	return &config.ResolvedConfig{
		Profile:     config.DefaultProfile(),
		ProfileName: "default",
	}
}

func validPipeline() *pipeline.Pipeline {
	return pipeline.NewPipeline()
}

// --- Constructor tests ---

func TestNew_Success(t *testing.T) {
	t.Parallel()

	m, err := New(validCfg(), validPipeline())
	require.NoError(t, err)
	assert.Equal(t, "default", m.profileSelector.current)
	assert.False(t, m.ready)
	assert.False(t, m.quitting)
	assert.Nil(t, m.err)
}

func TestNew_NilPipeline(t *testing.T) {
	t.Parallel()

	_, err := New(validCfg(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline must not be nil")
}

func TestNew_NilConfig(t *testing.T) {
	t.Parallel()

	_, err := New(nil, validPipeline())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config must not be nil")
}

// --- Init test ---

func TestModel_Init(t *testing.T) {
	t.Parallel()

	m, err := New(validCfg(), validPipeline())
	require.NoError(t, err)

	cmd := m.Init()
	// Init now returns a command to scan the root directory.
	assert.NotNil(t, cmd, "Init should return a cmd to load root directory")
}

// --- Key handling tests ---

func TestUpdate_QuitOnQ(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	model := updated.(Model)

	assert.True(t, model.quitting)
	require.NotNil(t, cmd, "quit should produce a tea.Quit cmd")

	// Execute the command and verify it produces the quit message.
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)
}

func TestUpdate_QuitOnEsc(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model := updated.(Model)

	assert.True(t, model.quitting)
	require.NotNil(t, cmd)
}

func TestUpdate_HelpToggle(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	assert.False(t, m.helpOverlay.visible)

	// First press: show help.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	model := updated.(Model)
	assert.True(t, model.helpOverlay.visible)
	assert.Nil(t, cmd)

	// Second press: hide help.
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	model = updated.(Model)
	assert.False(t, model.helpOverlay.visible)
	assert.Nil(t, cmd)
}

func TestUpdate_CtrlGProducesGenerateMsg(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	// Make the model ready so it is not in initializing state.
	m.ready = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	_ = updated.(Model)

	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, GenerateRequestedMsg{}, msg)
}

func TestUpdate_CtrlGSuppressedDuringHelp(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.helpOverlay.visible = true

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	// When help is visible, Ctrl+G should not produce a generate command.
	// The cmds slice is empty, so tea.Batch returns nil.
	assert.Nil(t, cmd)
}

// --- WindowSizeMsg propagation ---

func TestUpdate_WindowSizeMsg(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	assert.False(t, m.ready)

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(Model)

	assert.True(t, model.ready)
	assert.Equal(t, 120, model.width)
	assert.Equal(t, 40, model.height)
	assert.Nil(t, cmd)

	// Verify sub-models received the size update.
	assert.Equal(t, 38, model.fileTree.Height()) // 40 - 2
	assert.Equal(t, 38, model.statsPanel.Height()) // 40 - 2
}

// --- Message routing tests ---

func TestUpdate_FileToggledMsg(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	msg := FileToggledMsg{Path: "main.go", Included: true}

	updated, cmd := m.Update(msg)
	model := updated.(Model)

	// FileToggledMsg is forwarded to stats for debounced token recalculation.
	// The stats panel sets calculating=true and returns a debounce tick cmd.
	assert.True(t, model.statsPanel.Calculating())
	assert.NotNil(t, cmd, "should return debounce tick cmd")
}

func TestUpdate_TokenCountUpdatedMsg(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	msg := TokenCountUpdatedMsg{
		TotalTokens: 5000,
		FileCount:   42,
		BudgetUsed:  75.5,
	}

	updated, cmd := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, 5000, model.statsPanel.TotalTokens())
	assert.Equal(t, 42, model.statsPanel.SelectedFiles())
	assert.InDelta(t, 75.5, model.statsPanel.BudgetUsed(), 0.001)
	assert.Nil(t, cmd)
}

func TestUpdate_ProfileChangedMsg(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	msg := ProfileChangedMsg{ProfileName: "minimal"}

	updated, cmd := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, "minimal", model.profileSelector.current)
	assert.Nil(t, cmd)
}

func TestUpdate_ErrorMsg(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	testErr := errors.New("something went wrong")
	msg := ErrorMsg{Err: testErr}

	updated, cmd := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, testErr, model.err)
	assert.Nil(t, cmd)
}

// --- View tests ---

func TestView_Initializing(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	view := m.View()
	assert.Equal(t, "Initializing...", view)
}

func TestView_Quitting(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.quitting = true
	view := m.View()
	assert.Equal(t, "", view)
}

func TestView_ErrorDisplay(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.ready = true
	m.width = 80
	m.height = 24
	m.err = errors.New("disk full")

	view := m.View()
	assert.Contains(t, view, "Error: disk full")
	assert.Contains(t, view, "Press q to quit")
}

func TestView_HelpOverlay(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.ready = true
	m.width = 80
	m.height = 24
	m.helpOverlay.visible = true

	view := m.View()
	assert.Contains(t, view, "Harvx Interactive Mode")
	assert.Contains(t, view, "Press ? to close help")
}

func TestView_NormalLayout(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.ready = true
	m.width = 100
	m.height = 30

	view := m.View()
	// Should contain the status bar with profile info.
	assert.Contains(t, view, "Profile: default")
	// Should contain file tree content (loading or empty).
	assert.Contains(t, view, "Loading file tree")
	// Should contain stats panel.
	assert.Contains(t, view, "Stats")
}

// --- Sub-model unit tests ---

func TestProfileSelector_Next_SingleProfile(t *testing.T) {
	t.Parallel()

	ps := newProfileSelectorModel("default")
	next := ps.next()
	assert.Equal(t, "default", next.current, "single profile should not change on next()")
}

func TestProfileSelector_Next_MultipleProfiles(t *testing.T) {
	t.Parallel()

	ps := profileSelectorModel{
		current:  "default",
		profiles: []string{"default", "minimal", "full"},
		index:    0,
	}

	ps = ps.next()
	assert.Equal(t, "minimal", ps.current)

	ps = ps.next()
	assert.Equal(t, "full", ps.current)

	ps = ps.next()
	assert.Equal(t, "default", ps.current, "should wrap around")
}

func TestStatsPanelModel_HandleTokenUpdate(t *testing.T) {
	t.Parallel()

	sp := stats.New(stats.Options{MaxTokens: 100000})
	updated, _ := sp.Update(TokenCountUpdatedMsg{
		TotalTokens: 1234,
		FileCount:   10,
		BudgetUsed:  50.0,
	})
	sp = updated.(stats.Model)

	assert.Equal(t, 1234, sp.TotalTokens())
	assert.Equal(t, 10, sp.SelectedFiles())
	assert.InDelta(t, 50.0, sp.BudgetUsed(), 0.001)
}

// --- Helpers ---

func mustNewModel(t *testing.T) Model {
	t.Helper()
	m, err := New(validCfg(), validPipeline())
	require.NoError(t, err)
	return m
}
