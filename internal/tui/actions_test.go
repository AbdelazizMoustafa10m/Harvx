package tui

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/pipeline"
)

func TestHandleGenerate_ActivatesOverlay(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	updated, cmd := m.handleGenerate()

	assert.Equal(t, overlayGenerating, updated.overlay.State())
	require.NotNil(t, cmd)
}

func TestHandleGenerateComplete_Success(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.overlay.startGenerating()

	result := &pipeline.RunResult{}
	updated, cmd := m.handleGenerateComplete(generateCompleteMsg{result: result})

	assert.True(t, updated.quitting)
	assert.Equal(t, overlayNone, updated.overlay.State())
	require.NotNil(t, cmd)
}

func TestHandleGenerateComplete_Error(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.overlay.startGenerating()

	updated, cmd := m.handleGenerateComplete(generateCompleteMsg{
		err: errors.New("pipeline broke"),
	})

	assert.False(t, updated.quitting)
	assert.Equal(t, overlayNone, updated.overlay.State())
	assert.NotNil(t, updated.err)
	require.NotNil(t, cmd, "should return toast cmd")
}

func TestHandlePreview_ActivatesOverlay(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	updated, cmd := m.handlePreview()

	assert.Equal(t, overlayPreviewing, updated.overlay.State())
	assert.Nil(t, cmd)
}

func TestHandleSaveProfile_ActivatesOverlay(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	updated, cmd := m.handleSaveProfile()

	assert.Equal(t, overlaySavingProfile, updated.overlay.State())
	require.NotNil(t, cmd, "should return text input blink cmd")
}

func TestHandleExportClipboard_NoFiles(t *testing.T) {
	t.Parallel()

	cb := &mockClipboard{}
	m, err := New(validCfg(), validPipeline(), Options{Clipboard: cb})
	require.NoError(t, err)

	_, cmd := m.handleExportClipboard()
	require.NotNil(t, cmd)

	msg := cmd()
	cbMsg, ok := msg.(clipboardCompleteMsg)
	assert.True(t, ok)
	assert.Equal(t, 0, cbMsg.count)
}

func TestHandleExportClipboard_ClipboardError(t *testing.T) {
	t.Parallel()

	cb := &mockClipboard{err: ErrClipboardUnavailable}
	m, err := New(validCfg(), validPipeline(), Options{Clipboard: cb})
	require.NoError(t, err)

	updated, cmd := m.handleClipboardComplete(clipboardCompleteMsg{
		err: ErrClipboardUnavailable,
	})

	assert.True(t, updated.toast.visible)
	assert.Contains(t, updated.toast.message, "clipboard not available")
	require.NotNil(t, cmd)
}

func TestHandleSaveProfileConfirm_EmptyName(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)
	m.overlay.startSaveProfile()

	// Simulate empty input.
	updated, cmd := m.handleSaveProfileConfirm()

	assert.Equal(t, overlayNone, updated.overlay.State())
	assert.True(t, updated.toast.visible)
	assert.Contains(t, updated.toast.message, "cannot be empty")
	require.NotNil(t, cmd)
}

func TestHandleSaveProfileComplete_Success(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)

	updated, cmd := m.handleSaveProfileComplete(saveProfileCompleteMsg{
		profileName: "myprofile",
	})

	assert.True(t, updated.toast.visible)
	assert.Contains(t, updated.toast.message, "myprofile")
	assert.Contains(t, updated.toast.message, "saved")
	require.NotNil(t, cmd)
}

func TestHandleSaveProfileComplete_Error(t *testing.T) {
	t.Parallel()

	m := mustNewModel(t)

	updated, cmd := m.handleSaveProfileComplete(saveProfileCompleteMsg{
		err: errors.New("permission denied"),
	})

	assert.True(t, updated.toast.visible)
	assert.Contains(t, updated.toast.message, "Save failed")
	require.NotNil(t, cmd)
}
