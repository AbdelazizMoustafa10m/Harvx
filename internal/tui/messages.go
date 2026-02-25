// Package tui implements the Bubble Tea interactive terminal UI for Harvx.
package tui

import tea "github.com/charmbracelet/bubbletea"

// FileToggledMsg is sent when a file's inclusion state changes in the file tree.
type FileToggledMsg struct {
	Path     string
	Included bool
}

// TokenCountUpdatedMsg is sent when the token count changes after file selection.
type TokenCountUpdatedMsg struct {
	TotalTokens int
	FileCount   int
	BudgetUsed  float64 // percentage 0-100
}

// ProfileChangedMsg is sent when the user switches to a different profile.
type ProfileChangedMsg struct {
	ProfileName string
}

// GenerateRequestedMsg is sent when the user presses Enter to trigger generation.
type GenerateRequestedMsg struct{}

// ErrorMsg wraps an error for display in the TUI.
type ErrorMsg struct {
	Err error
}

// Error implements the error interface.
func (e ErrorMsg) Error() string { return e.Err.Error() }

// WindowSizeMsg wraps tea.WindowSizeMsg for internal routing.
// We re-export this so sub-models don't need to import bubbletea directly.
type WindowSizeMsg = tea.WindowSizeMsg
