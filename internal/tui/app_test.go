package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tui/profile"
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

// mockClipboard is a test double for the Clipboarder interface.
type mockClipboard struct {
	written string
	err     error
}

func (m *mockClipboard) WriteAll(text string) error {
	m.written = text
	return m.err
}

// --- Constructor tests ---

func TestNew_Success(t *testing.T) {
	t.Parallel()

	m, err := New(validCfg(), validPipeline())
	require.NoError(t, err)
	assert.Equal(t, "default", m.profileSelector.Current())
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

func TestNew_WithProfileNames(t *testing.T) {
	t.Parallel()

	m, err := New(validCfg(), validPipeline(), Options{
		ProfileNames: []string{"default", "minimal", "full"},
	})
	require.NoError(t, err)
	assert.Equal(t, "default", m.profileSelector.Current())
	assert.Equal(t, 3, m.profileSelector.Count())
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

func TestUpdate_EnterStartsGenerate(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.ready = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)

	// Generate action activates the generating overlay.
	assert.Equal(t, overlayGenerating, model.overlay.State())
	require.NotNil(t, cmd)
}

func TestUpdate_EnterSuppressedDuringHelp(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.helpOverlay.visible = true

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// When help is visible, Enter should not trigger generation.
	assert.Nil(t, cmd)
}

func TestUpdate_PreviewAction(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.ready = true

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model := updated.(Model)

	assert.Equal(t, overlayPreviewing, model.overlay.State())
}

func TestUpdate_SaveAction(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.ready = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model := updated.(Model)

	assert.Equal(t, overlaySavingProfile, model.overlay.State())
	require.NotNil(t, cmd, "save should return a blink cmd for text input")
}

func TestUpdate_ExportAction(t *testing.T) {
	t.Parallel()

	cb := &mockClipboard{}
	m, err := New(validCfg(), validPipeline(), Options{
		Clipboard: cb,
	})
	require.NoError(t, err)
	m.ready = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	_ = updated.(Model)

	require.NotNil(t, cmd)
	// Execute the command to trigger clipboard write.
	msg := cmd()
	cbMsg, ok := msg.(clipboardCompleteMsg)
	assert.True(t, ok)
	assert.Equal(t, 0, cbMsg.count, "no files selected initially")
}

func TestUpdate_TabCyclesProfile(t *testing.T) {
	t.Parallel()

	m := mustNewModelWithProfiles(t, []string{"default", "minimal", "full"})
	assert.Equal(t, "default", m.profileSelector.Current())

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := updated.(Model)

	assert.Equal(t, "minimal", model.profileSelector.Current())
	require.NotNil(t, cmd)
}

func TestUpdate_ShiftTabCyclesProfileBackward(t *testing.T) {
	t.Parallel()

	m := mustNewModelWithProfiles(t, []string{"default", "minimal", "full"})
	assert.Equal(t, "default", m.profileSelector.Current())

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model := updated.(Model)

	assert.Equal(t, "full", model.profileSelector.Current())
	require.NotNil(t, cmd)
}

// --- Overlay key handling tests ---

func TestOverlay_PreviewDismissOnEsc(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.ready = true

	// Open preview.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model := updated.(Model)
	assert.Equal(t, overlayPreviewing, model.overlay.State())

	// Press Esc to dismiss.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updated.(Model)
	assert.Equal(t, overlayNone, model.overlay.State())
}

func TestOverlay_SaveProfileCancel(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.ready = true

	// Open save overlay.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model := updated.(Model)
	assert.Equal(t, overlaySavingProfile, model.overlay.State())

	// Press Esc to cancel.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = updated.(Model)
	assert.Equal(t, overlayNone, model.overlay.State())
}

func TestOverlay_GeneratingQuit(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.overlay.startGenerating()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	model := updated.(Model)

	assert.True(t, model.quitting)
	require.NotNil(t, cmd)
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

	m := mustNewModelWithProfiles(t, []string{"default", "minimal"})
	msg := ProfileChangedMsg{ProfileName: "minimal"}

	updated, cmd := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, "minimal", model.profileSelector.Current())
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

func TestUpdate_ToastDismiss(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	_ = m.toast.show("test message")
	assert.True(t, m.toast.visible)

	updated, _ := m.Update(toastDismissMsg{id: m.toast.id})
	model := updated.(Model)
	assert.False(t, model.toast.visible)
}

func TestUpdate_GenerateCompleteSuccess(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.overlay.startGenerating()

	result := &pipeline.RunResult{}
	updated, cmd := m.Update(generateCompleteMsg{result: result})
	model := updated.(Model)

	assert.True(t, model.quitting)
	assert.Equal(t, overlayNone, model.overlay.State())
	require.NotNil(t, cmd)
}

func TestUpdate_GenerateCompleteError(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.overlay.startGenerating()

	updated, cmd := m.Update(generateCompleteMsg{err: errors.New("pipeline failed")})
	model := updated.(Model)

	assert.False(t, model.quitting)
	assert.Equal(t, overlayNone, model.overlay.State())
	assert.NotNil(t, model.err)
	require.NotNil(t, cmd, "should return toast cmd")
}

func TestUpdate_ClipboardComplete(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)

	updated, cmd := m.Update(clipboardCompleteMsg{count: 5})
	model := updated.(Model)

	assert.True(t, model.toast.visible)
	assert.Contains(t, model.toast.message, "5 file path(s)")
	require.NotNil(t, cmd)
}

func TestUpdate_ClipboardError(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)

	updated, cmd := m.Update(clipboardCompleteMsg{err: ErrClipboardUnavailable})
	model := updated.(Model)

	assert.True(t, model.toast.visible)
	assert.Contains(t, model.toast.message, "clipboard not available")
	require.NotNil(t, cmd)
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

func TestView_OverlayPreview(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.ready = true
	m.width = 80
	m.height = 24
	m.overlay.startPreview(10, 5000, 128000, 3.9, nil, nil)

	view := m.View()
	assert.Contains(t, view, "Output Preview")
	assert.Contains(t, view, "10")
}

func TestView_ToastDisplayed(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.ready = true
	m.width = 100
	m.height = 30
	_ = m.toast.show("Profile saved!")

	view := m.View()
	assert.Contains(t, view, "Profile saved!")
}

// --- Profile selector sub-model tests ---

func TestProfileSelector_Next_SingleProfile(t *testing.T) {
	t.Parallel()

	ps := profile.New("default", nil)
	next, cmd := ps.Next()
	assert.Equal(t, "default", next.Current(), "single profile should not change on next()")
	assert.Nil(t, cmd)
}

func TestProfileSelector_Next_MultipleProfiles(t *testing.T) {
	t.Parallel()

	ps := profile.New("default", []string{"default", "minimal", "full"})

	ps, _ = ps.Next()
	assert.Equal(t, "minimal", ps.Current())

	ps, _ = ps.Next()
	assert.Equal(t, "full", ps.Current())

	ps, _ = ps.Next()
	assert.Equal(t, "default", ps.Current(), "should wrap around")
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

func mustNewModelWithProfiles(t *testing.T, profiles []string) Model {
	t.Helper()
	m, err := New(validCfg(), validPipeline(), Options{
		ProfileNames: profiles,
	})
	require.NoError(t, err)
	return m
}
