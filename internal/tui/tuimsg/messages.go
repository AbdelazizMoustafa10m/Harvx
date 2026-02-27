// Package tuimsg defines message types shared between the root TUI model and
// its sub-model packages (filetree, stats, etc.) to avoid circular imports.
package tuimsg

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
