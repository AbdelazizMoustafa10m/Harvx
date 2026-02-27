package tui

import "errors"

// Clipboarder is an interface for writing text to the system clipboard.
// This abstraction allows testing without a real clipboard and defers the
// actual clipboard dependency (e.g., atotto/clipboard) to later integration.
type Clipboarder interface {
	// WriteAll writes the given text to the system clipboard.
	WriteAll(text string) error
}

// ErrClipboardUnavailable is returned when no clipboard implementation is available.
var ErrClipboardUnavailable = errors.New("clipboard not available on this system")

// noopClipboard is the default clipboard implementation that reports unavailability.
type noopClipboard struct{}

func (noopClipboard) WriteAll(_ string) error {
	return ErrClipboardUnavailable
}
