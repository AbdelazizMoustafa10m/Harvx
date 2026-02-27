package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/harvx/harvx/internal/pipeline"
)

// generateCompleteMsg is sent when the pipeline generation finishes.
type generateCompleteMsg struct {
	result *pipeline.RunResult
	err    error
}

// saveProfileCompleteMsg is sent after a profile is saved to disk.
type saveProfileCompleteMsg struct {
	profileName string
	err         error
}

// clipboardCompleteMsg is sent after writing to the clipboard.
type clipboardCompleteMsg struct {
	count int
	err   error
}

// handleGenerate starts the pipeline generation in a background goroutine.
// It activates the generating overlay and returns a spinner tick command
// along with the generation command.
func (m Model) handleGenerate() (Model, tea.Cmd) {
	spinnerCmd := m.overlay.startGenerating()
	p := m.pipeline
	selectedFiles := m.fileTree.SelectedFiles()

	genCmd := func() tea.Msg {
		result, err := p.Run(context.Background(), pipeline.RunOptions{
			Dir: m.rootDir(),
		})
		_ = selectedFiles // Selected files used for filtering if needed.
		return generateCompleteMsg{result: result, err: err}
	}

	return m, tea.Batch(spinnerCmd, genCmd)
}

// handleGenerateComplete processes the result of a pipeline generation run.
func (m Model) handleGenerateComplete(msg generateCompleteMsg) (Model, tea.Cmd) {
	m.overlay.close()

	if msg.err != nil {
		m.err = msg.err
		toastCmd := m.toast.show("Generation failed: " + msg.err.Error())
		return m, toastCmd
	}

	// Pipeline finished successfully. Show a toast and then quit.
	m.quitting = true
	return m, tea.Quit
}

// handlePreview shows the preview overlay with current stats.
func (m Model) handlePreview() (Model, tea.Cmd) {
	m.overlay.startPreview(
		m.statsPanel.SelectedFiles(),
		m.statsPanel.TotalTokens(),
		m.statsPanel.MaxTokens(),
		m.statsPanel.BudgetUsed(),
		m.statsPanel.TierBreakdown(),
		m.statsPanel.TierTokens(),
	)
	return m, nil
}

// handleSaveProfile activates the save-as-profile text input overlay.
func (m Model) handleSaveProfile() (Model, tea.Cmd) {
	cmd := m.overlay.startSaveProfile()
	return m, cmd
}

// handleSaveProfileConfirm serializes the current selection and saves it to
// harvx.toml as a new profile.
func (m Model) handleSaveProfileConfirm() (Model, tea.Cmd) {
	name := strings.TrimSpace(m.overlay.input.Value())
	if name == "" {
		toastCmd := m.toast.show("Profile name cannot be empty")
		m.overlay.close()
		return m, toastCmd
	}

	m.overlay.close()
	selectedFiles := m.fileTree.SelectedFiles()
	configPath := "harvx.toml"

	cmd := func() tea.Msg {
		err := appendProfileToFile(configPath, name, selectedFiles)
		return saveProfileCompleteMsg{profileName: name, err: err}
	}

	return m, cmd
}

// handleSaveProfileComplete handles the result of a profile save operation.
func (m Model) handleSaveProfileComplete(msg saveProfileCompleteMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		toastCmd := m.toast.show("Save failed: " + msg.err.Error())
		return m, toastCmd
	}
	toastCmd := m.toast.show("Profile \"" + msg.profileName + "\" saved to harvx.toml")
	return m, toastCmd
}

// handleExportClipboard copies the selected file paths to the system clipboard.
func (m Model) handleExportClipboard() (Model, tea.Cmd) {
	selectedFiles := m.fileTree.SelectedFiles()
	cb := m.clipboard

	cmd := func() tea.Msg {
		if len(selectedFiles) == 0 {
			return clipboardCompleteMsg{count: 0, err: nil}
		}
		text := strings.Join(selectedFiles, "\n")
		err := cb.WriteAll(text)
		return clipboardCompleteMsg{count: len(selectedFiles), err: err}
	}

	return m, cmd
}

// handleClipboardComplete handles the result of a clipboard export operation.
func (m Model) handleClipboardComplete(msg clipboardCompleteMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		toastCmd := m.toast.show("Clipboard: " + msg.err.Error())
		return m, toastCmd
	}
	if msg.count == 0 {
		toastCmd := m.toast.show("No files selected to export")
		return m, toastCmd
	}
	toastCmd := m.toast.show(fmt.Sprintf("Copied %d file path(s) to clipboard", msg.count))
	return m, toastCmd
}

// rootDir returns the root directory for the pipeline run.
func (m Model) rootDir() string {
	return "."
}
