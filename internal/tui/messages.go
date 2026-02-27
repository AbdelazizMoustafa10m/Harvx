// Package tui implements the Bubble Tea interactive terminal UI for Harvx.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/harvx/harvx/internal/tui/tuimsg"
)

// Re-export shared message types so existing consumers of the tui package
// continue to work without change.

// FileToggledMsg is sent when a file's inclusion state changes in the file tree.
type FileToggledMsg = tuimsg.FileToggledMsg

// TokenCountUpdatedMsg is sent when the token count changes after file selection.
type TokenCountUpdatedMsg = tuimsg.TokenCountUpdatedMsg

// ProfileChangedMsg is sent when the user switches to a different profile.
type ProfileChangedMsg = tuimsg.ProfileChangedMsg

// GenerateRequestedMsg is sent when the user presses Enter to trigger generation.
type GenerateRequestedMsg = tuimsg.GenerateRequestedMsg

// ErrorMsg wraps an error for display in the TUI.
type ErrorMsg = tuimsg.ErrorMsg

// WindowSizeMsg wraps tea.WindowSizeMsg for internal routing.
// We re-export this so sub-models don't need to import bubbletea directly.
type WindowSizeMsg = tea.WindowSizeMsg
