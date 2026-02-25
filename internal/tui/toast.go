package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// defaultToastDuration is the auto-dismiss duration for toast messages.
const defaultToastDuration = 2 * time.Second

// toastDismissMsg is sent when a toast's auto-dismiss timer expires.
type toastDismissMsg struct {
	id int
}

// toastModel holds the state for a brief auto-dismissing status message.
type toastModel struct {
	message  string
	visible  bool
	id       int
	duration time.Duration
}

// newToastModel returns a zero-value toastModel that is not visible.
func newToastModel() toastModel {
	return toastModel{
		duration: defaultToastDuration,
	}
}

// show displays a toast message and returns a tea.Cmd that will auto-dismiss
// it after the configured duration.
func (m *toastModel) show(message string) tea.Cmd {
	m.id++
	m.message = message
	m.visible = true
	id := m.id
	dur := m.duration
	return tea.Tick(dur, func(_ time.Time) tea.Msg {
		return toastDismissMsg{id: id}
	})
}

// dismiss handles a toastDismissMsg. It hides the toast only if the dismiss
// message's ID matches the current toast (stale dismissals are ignored).
func (m *toastModel) dismiss(msg toastDismissMsg) {
	if msg.id == m.id {
		m.visible = false
		m.message = ""
	}
}

// view renders the toast message. Returns an empty string if not visible.
func (m toastModel) view(width int) string {
	if !m.visible || m.message == "" {
		return ""
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Padding(0, 2).
		Width(width).
		Align(lipgloss.Center)

	return style.Render(m.message)
}
