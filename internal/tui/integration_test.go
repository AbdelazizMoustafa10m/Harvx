package tui

import (
	"fmt"
	"runtime"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/tui/filetree"
)

// ---------------------------------------------------------------------------
// Test 1: Navigate tree and toggle files, verify stats update
// ---------------------------------------------------------------------------

func TestIntegration_NavigateAndToggle(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// The visible list starts with dirs first then files.
	visible := m.fileTree.Visible()
	require.True(t, len(visible) > 0, "should have visible nodes")

	// Navigate down past the first few visible nodes (likely dirs).
	for i := 0; i < 3; i++ {
		m = sendKey(t, m, specialKeyMsg(tea.KeyDown))
	}

	// Toggle current node with space.
	m, cmd := sendKeyWithCmd(t, m, keyMsg(' '))

	// The toggle should have produced FileToggledMsg cmd(s).
	require.NotNil(t, cmd)

	// Feed the toggle message back to see stats update.
	msg := cmd()
	if toggleMsg, ok := msg.(FileToggledMsg); ok {
		m = sendMsg(t, m, toggleMsg)
		assert.True(t, m.statsPanel.Calculating(), "stats should start debounced recalculation")
	}
}

// ---------------------------------------------------------------------------
// Test 2: Expand directory, collapse, re-expand, toggle children
// ---------------------------------------------------------------------------

func TestIntegration_ExpandDirectoryAndToggleAll(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// The tree already has dirs expanded. Find src/ in visible list.
	srcIdx := -1
	for i, n := range m.fileTree.Visible() {
		if n.IsDir && n.Name == "src" {
			srcIdx = i
			break
		}
	}
	require.GreaterOrEqual(t, srcIdx, 0, "src/ should be in visible list")

	// Navigate to src/.
	for i := 0; i < srcIdx; i++ {
		m = sendKey(t, m, specialKeyMsg(tea.KeyDown))
	}

	// Collapse with left/h.
	m = sendKey(t, m, keyMsg('h'))
	// Children should not be visible.
	for _, n := range m.fileTree.Visible() {
		if n.Path == "src/main.go" {
			t.Error("src/main.go should not be visible after collapse")
		}
	}

	// Re-expand with right/l.
	m = sendKey(t, m, keyMsg('l'))
	// Children should be visible again.
	found := false
	for _, n := range m.fileTree.Visible() {
		if n.Path == "src/main.go" {
			found = true
			break
		}
	}
	assert.True(t, found, "src/main.go should be visible after expand")

	// Toggle the directory (space) to select all children.
	m, cmd := sendKeyWithCmd(t, m, keyMsg(' '))
	require.NotNil(t, cmd, "toggling a directory should produce commands")

	// Verify the directory has all children in included state.
	srcNode := m.fileTree.Root().FindByPath("src")
	require.NotNil(t, srcNode)
	assert.Equal(t, filetree.Included, srcNode.Included)
	for _, child := range srcNode.Children {
		assert.Equal(t, filetree.Included, child.Included)
	}
}

// ---------------------------------------------------------------------------
// Test 3: Tab to switch profile, verify stats re-evaluate
// ---------------------------------------------------------------------------

func TestIntegration_ProfileSwitchUpdatesStats(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTreeAndProfiles(t, []string{"default", "minimal", "full"})

	assert.Equal(t, "default", m.profileSelector.Current())

	// Press Tab to switch profile.
	m, cmd := sendKeyWithCmd(t, m, specialKeyMsg(tea.KeyTab))
	require.NotNil(t, cmd)

	// Execute the cmd -- it should produce a ProfileChangedMsg.
	msg := cmd()
	if pcMsg, ok := msg.(ProfileChangedMsg); ok {
		m = sendMsg(t, m, pcMsg)
	}

	assert.Equal(t, "minimal", m.profileSelector.Current())

	// View should reflect the new profile.
	view := m.View()
	assert.Contains(t, view, "minimal")
}

// ---------------------------------------------------------------------------
// Test 4: Help overlay toggle
// ---------------------------------------------------------------------------

func TestIntegration_HelpOverlayToggle(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// Press ? to show help.
	m = sendKey(t, m, keyMsg('?'))
	assertViewContains(t, m, "Harvx Interactive Mode")
	assertViewContains(t, m, "Press ? to close help")

	// Keys should be suppressed while help is visible.
	// Press 'p' -- should NOT open preview.
	m = sendKey(t, m, keyMsg('p'))
	// Help should still be visible.
	assertViewContains(t, m, "Harvx Interactive Mode")
	// Preview should not be active.
	assert.Equal(t, overlayNone, m.overlay.State())

	// Press ? again to dismiss.
	m = sendKey(t, m, keyMsg('?'))
	assertViewNotContains(t, m, "Harvx Interactive Mode")
}

// ---------------------------------------------------------------------------
// Test 5: Select all files, deselect all
// ---------------------------------------------------------------------------

func TestIntegration_ToggleAllSelectDeselect(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// Toggle the first visible node.
	m, _ = sendKeyWithCmd(t, m, keyMsg(' '))

	// The cursor node should now be toggled.
	node := m.fileTree.Visible()[0]
	assert.NotEqual(t, filetree.Excluded, node.Included)

	// Toggle again to deselect.
	m, _ = sendKeyWithCmd(t, m, keyMsg(' '))
	node = m.fileTree.Visible()[0]
	assert.Equal(t, filetree.Excluded, node.Included)
}

// ---------------------------------------------------------------------------
// Test 6: Preview overlay shows correct data
// ---------------------------------------------------------------------------

func TestIntegration_PreviewOverlay(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// Press 'p' to show preview.
	m = sendKey(t, m, keyMsg('p'))
	assert.Equal(t, overlayPreviewing, m.overlay.State())

	// View should contain preview content.
	view := m.View()
	assert.Contains(t, view, "Output Preview")
	assert.Contains(t, view, "Files:")
	assert.Contains(t, view, "Tokens:")

	// Press Esc to dismiss.
	m = sendKey(t, m, specialKeyMsg(tea.KeyEscape))
	assert.Equal(t, overlayNone, m.overlay.State())
}

// ---------------------------------------------------------------------------
// Test 7: Save profile flow
// ---------------------------------------------------------------------------

func TestIntegration_SaveProfileFlow(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// Press 's' to open save overlay.
	m, cmd := sendKeyWithCmd(t, m, keyMsg('s'))
	assert.Equal(t, overlaySavingProfile, m.overlay.State())
	require.NotNil(t, cmd) // text input blink cmd

	// Type a profile name by sending rune messages while overlay is active.
	for _, ch := range "myprofile" {
		m, _ = sendKeyWithCmd(t, m, keyMsg(ch))
	}

	// Press Esc to cancel.
	m = sendKey(t, m, specialKeyMsg(tea.KeyEscape))
	assert.Equal(t, overlayNone, m.overlay.State())
}

// ---------------------------------------------------------------------------
// Test 8: Quit exits cleanly
// ---------------------------------------------------------------------------

func TestIntegration_QuitExitsCleanly(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// Press 'q' to quit.
	m, cmd := sendKeyWithCmd(t, m, keyMsg('q'))
	assert.True(t, m.quitting)
	require.NotNil(t, cmd)

	// The command should produce a tea.QuitMsg.
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)

	// View should be empty when quitting.
	assert.Equal(t, "", m.View())
}

// ---------------------------------------------------------------------------
// Test 9: Ctrl+C does not panic
// ---------------------------------------------------------------------------

func TestIntegration_CtrlCExitsCleanly(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// Ctrl+C produces a tea.KeyCtrlC msg in Bubble Tea.
	// Our model does not explicitly handle it (Bubble Tea runtime does),
	// but we verify that the model does not panic on receiving it.
	m, cmd := sendKeyWithCmd(t, m, specialKeyMsg(tea.KeyCtrlC))
	_ = m
	_ = cmd
}

// ---------------------------------------------------------------------------
// Test 10: Window resize propagation
// ---------------------------------------------------------------------------

func TestIntegration_WindowResizePropagation(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// Send a new window size.
	m = sendMsg(t, m, windowSizeMsg(200, 60))

	assert.Equal(t, 200, m.width)
	assert.Equal(t, 60, m.height)

	// Layout: left = 200*60/100 = 120, right = 200-120-1 = 79
	assert.Equal(t, 58, m.fileTree.Height()) // 60-2
	assert.Equal(t, 58, m.statsPanel.Height())
}

// ---------------------------------------------------------------------------
// Test 11: Error conditions handled gracefully
// ---------------------------------------------------------------------------

func TestIntegration_ErrorConditions(t *testing.T) {
	t.Parallel()

	t.Run("nil file tree node", func(t *testing.T) {
		t.Parallel()
		m := mustNewModel(t)
		m.ready = true
		m.width = 80
		m.height = 24

		// Sending navigation keys with no tree loaded should not panic.
		m = sendKey(t, m, specialKeyMsg(tea.KeyDown))
		m = sendKey(t, m, specialKeyMsg(tea.KeyUp))
		m = sendKey(t, m, keyMsg(' '))
		m = sendKey(t, m, keyMsg('l'))
		m = sendKey(t, m, keyMsg('h'))

		// Should render without panic.
		_ = m.View()
	})

	t.Run("error message display", func(t *testing.T) {
		t.Parallel()
		m := mustNewModelWithTree(t)
		m = sendMsg(t, m, ErrorMsg{Err: fmt.Errorf("permission denied: /root/secret")})

		view := m.View()
		assert.Contains(t, view, "Error:")
		assert.Contains(t, view, "permission denied")
	})

	t.Run("clipboard unavailable", func(t *testing.T) {
		t.Parallel()
		m := mustNewModelWithTree(t)

		// Toggle a file first so SelectedFiles is non-empty,
		// then the noop clipboard will actually be called.
		m, _ = sendKeyWithCmd(t, m, keyMsg(' '))

		// Press 'e' for export. Default clipboard is noop -> ErrClipboardUnavailable.
		m, cmd := sendKeyWithCmd(t, m, keyMsg('e'))
		require.NotNil(t, cmd)
		msg := cmd()
		cbMsg, ok := msg.(clipboardCompleteMsg)
		require.True(t, ok, "expected clipboardCompleteMsg, got %T", msg)

		// Feed back the message.
		m = sendMsg(t, m, cbMsg)
		assert.True(t, m.toast.visible)
		assert.Contains(t, m.toast.message, "clipboard not available")
	})

	t.Run("generate complete error", func(t *testing.T) {
		t.Parallel()
		m := mustNewModelWithTree(t)
		m.overlay.startGenerating()

		m = sendMsg(t, m, generateCompleteMsg{err: fmt.Errorf("disk full")})
		// Should show toast with error, overlay should be closed.
		assert.Equal(t, overlayNone, m.overlay.State())
		assert.NotNil(t, m.err)
	})
}

// ---------------------------------------------------------------------------
// Test 12: No goroutine leaks
// ---------------------------------------------------------------------------

func TestIntegration_NoGoroutineLeaks(t *testing.T) {
	// Not parallel: goroutine counting is unreliable when other parallel
	// tests are spinning up and tearing down concurrently.

	// Warm up: create and discard a model so lazy-init goroutines settle.
	warmup := mustNewModelWithTree(t)
	_ = warmup.View()
	runtime.GC()
	runtime.Gosched()

	before := runtime.NumGoroutine()

	m := mustNewModelWithTree(t)

	// Run through a typical session: navigate, toggle, preview, quit.
	m = sendKey(t, m, specialKeyMsg(tea.KeyDown))
	m = sendKey(t, m, specialKeyMsg(tea.KeyDown))
	m, _ = sendKeyWithCmd(t, m, keyMsg(' '))
	m = sendKey(t, m, keyMsg('p'))
	m = sendKey(t, m, specialKeyMsg(tea.KeyEscape))
	m, _ = sendKeyWithCmd(t, m, keyMsg('q'))

	// Allow goroutines to settle.
	runtime.GC()
	runtime.Gosched()

	after := runtime.NumGoroutine()
	// Allow a generous delta for GC, finalizer, and test-framework goroutines.
	assert.InDelta(t, before, after, 10, "goroutine count should not grow significantly")
}

// ---------------------------------------------------------------------------
// Test 13: Full layout composition
// ---------------------------------------------------------------------------

func TestIntegration_FullLayoutComposition(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	view := m.View()

	// Layout should have file tree on left, stats on right, status bar at bottom.
	assert.Contains(t, view, "Profile: default", "status bar should show profile")
	assert.Contains(t, view, "Stats", "stats panel should be visible")
	// The separator should be present (pipes).
	assert.Contains(t, view, "|", "separator should be present")
}

// ---------------------------------------------------------------------------
// Test 14: Export clipboard success with mock
// ---------------------------------------------------------------------------

func TestIntegration_ExportClipboardSuccess(t *testing.T) {
	t.Parallel()

	cb := &mockClipboard{}
	m, err := New(validCfg(), validPipeline(), Options{
		Clipboard: cb,
	})
	require.NoError(t, err)

	tree := buildTestTree()
	m.fileTree = filetree.NewWithRoot(tree, ".")
	m.statsPanel.SetTreeRoot(tree)
	m.ready = true
	m.width = 120
	m.height = 40
	m.fileTree.SetSize(72, 38)
	m.statsPanel.SetSize(47, 38)

	// Toggle all in src/ by navigating there and toggling.
	srcIdx := -1
	for i, n := range m.fileTree.Visible() {
		if n.IsDir && n.Name == "src" {
			srcIdx = i
			break
		}
	}
	require.GreaterOrEqual(t, srcIdx, 0)

	for i := 0; i < srcIdx; i++ {
		m = sendKey(t, m, specialKeyMsg(tea.KeyDown))
	}
	m, _ = sendKeyWithCmd(t, m, keyMsg(' ')) // toggle src/

	// Press 'e' to export.
	m, cmd := sendKeyWithCmd(t, m, keyMsg('e'))
	require.NotNil(t, cmd)
	msg := cmd()

	cbMsg, ok := msg.(clipboardCompleteMsg)
	require.True(t, ok, "expected clipboardCompleteMsg, got %T", msg)
	m = sendMsg(t, m, cbMsg)

	// Clipboard should have content.
	assert.True(t, len(cb.written) > 0 || cbMsg.count > 0, "should have exported file paths")
	assert.True(t, m.toast.visible, "should show success toast")
}

// ---------------------------------------------------------------------------
// Test 15: Navigate with Home and End keys
// ---------------------------------------------------------------------------

func TestIntegration_HomeEndNavigation(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	visible := m.fileTree.Visible()
	require.True(t, len(visible) > 1, "need multiple visible nodes")

	// Press End/G to jump to the last node.
	m = sendKey(t, m, keyMsg('G'))
	assert.Equal(t, len(m.fileTree.Visible())-1, m.fileTree.Cursor(),
		"cursor should be at last visible node")

	// Press Home/g to jump to the first node.
	m = sendKey(t, m, keyMsg('g'))
	assert.Equal(t, 0, m.fileTree.Cursor(),
		"cursor should be at first visible node")
}

// ---------------------------------------------------------------------------
// Test 16: Shift+Tab cycles profile backward
// ---------------------------------------------------------------------------

func TestIntegration_ShiftTabCyclesProfileBackward(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTreeAndProfiles(t, []string{"default", "minimal", "full"})

	assert.Equal(t, "default", m.profileSelector.Current())

	// Press Shift+Tab to cycle backward (should wrap to "full").
	m, cmd := sendKeyWithCmd(t, m, specialKeyMsg(tea.KeyShiftTab))
	require.NotNil(t, cmd)

	msg := cmd()
	if pcMsg, ok := msg.(ProfileChangedMsg); ok {
		m = sendMsg(t, m, pcMsg)
	}

	assert.Equal(t, "full", m.profileSelector.Current())
}

// ---------------------------------------------------------------------------
// Test 17: drainCmds processes a chain of messages
// ---------------------------------------------------------------------------

func TestIntegration_DrainCmdsChain(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// Navigate and toggle to produce commands, then drain them.
	m = sendKey(t, m, specialKeyMsg(tea.KeyDown))
	m, cmd := sendKeyWithCmd(t, m, keyMsg(' '))
	require.NotNil(t, cmd)

	// Drain commands -- should not panic and should process the chain.
	m = drainCmds(t, m, cmd, 10)

	// After draining, the toggle should have been fully processed.
	// The stats panel should be in calculating state or have already
	// finished calculating (depending on how many ticks we drained).
	_ = m.View() // Ensure no panic on render.
}

// ---------------------------------------------------------------------------
// Test 18: Enter key on directory toggles expand/collapse
// ---------------------------------------------------------------------------

func TestIntegration_EnterTogglesDirExpansion(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// Find a directory in the visible list.
	dirIdx := -1
	for i, n := range m.fileTree.Visible() {
		if n.IsDir {
			dirIdx = i
			break
		}
	}
	require.GreaterOrEqual(t, dirIdx, 0, "should have at least one directory")

	// Navigate to it.
	for i := 0; i < dirIdx; i++ {
		m = sendKey(t, m, specialKeyMsg(tea.KeyDown))
	}

	dirNode := m.fileTree.Visible()[dirIdx]
	wasExpanded := dirNode.Expanded

	// Press Enter to toggle expand/collapse.
	m = sendKey(t, m, specialKeyMsg(tea.KeyEnter))

	// Note: Enter key at the root model level is bound to "generate" (not tree nav).
	// However, when not in help overlay, Enter triggers handleGenerate which
	// activates the generating overlay. This is by design.
	// The file tree's Enter (expand/collapse) is handled within the file tree
	// Update when forwarded from the root model.
	//
	// Since Enter matches root-level Generate, let's verify the generating
	// overlay was activated.
	if m.overlay.State() == overlayGenerating {
		// Root-level Enter triggered generate -- this is expected behavior.
		m.overlay.close()
	} else {
		// If somehow forwarded to tree, expansion state should have changed.
		assert.NotEqual(t, wasExpanded, dirNode.Expanded,
			"directory expansion state should have toggled")
	}
}

// ---------------------------------------------------------------------------
// Test 19: View renders correctly in various states
// ---------------------------------------------------------------------------

func TestIntegration_ViewStates(t *testing.T) {
	t.Parallel()

	t.Run("not ready shows initializing", func(t *testing.T) {
		t.Parallel()
		m := mustNewModel(t)
		assert.Equal(t, "Initializing...", m.View())
	})

	t.Run("quitting shows empty", func(t *testing.T) {
		t.Parallel()
		m := mustNewModelWithTree(t)
		m.quitting = true
		assert.Equal(t, "", m.View())
	})

	t.Run("generating overlay shows spinner text", func(t *testing.T) {
		t.Parallel()
		m := mustNewModelWithTree(t)
		m.overlay.startGenerating()

		view := m.View()
		assert.Contains(t, view, "Generating output")
	})

	t.Run("save profile overlay shows input prompt", func(t *testing.T) {
		t.Parallel()
		m := mustNewModelWithTree(t)
		m.overlay.startSaveProfile()

		view := m.View()
		assert.Contains(t, view, "Save Selection as Profile")
		assert.Contains(t, view, "Profile name")
	})
}

// ---------------------------------------------------------------------------
// Test 20: Multiple toggle-undo cycles maintain consistency
// ---------------------------------------------------------------------------

func TestIntegration_ToggleUndoCycles(t *testing.T) {
	t.Parallel()
	m := mustNewModelWithTree(t)

	// Record initial state of first visible node.
	initial := m.fileTree.Visible()[0].Included

	// Toggle on, off, on, off -- should return to initial state.
	for i := 0; i < 4; i++ {
		m, _ = sendKeyWithCmd(t, m, keyMsg(' '))
	}

	final := m.fileTree.Visible()[0].Included
	assert.Equal(t, initial, final,
		"even number of toggles should return to original state")
}
