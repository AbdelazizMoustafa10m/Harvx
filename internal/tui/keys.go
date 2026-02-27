package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the global key bindings for the TUI application.
type KeyMap struct {
	Quit           key.Binding
	Help           key.Binding
	Generate       key.Binding
	Preview        key.Binding
	Save           key.Binding
	Export         key.Binding
	ProfileTab     key.Binding
	ProfileBackTab key.Binding
	Up             key.Binding
	Down           key.Binding
	Toggle         key.Binding
	Search         key.Binding
	TierView       key.Binding
	SelectAll      key.Binding
	SelectNone     key.Binding
	ClearFilter    key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q/esc", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Generate: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "generate"),
		),
		Preview: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "preview"),
		),
		Save: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save as profile"),
		),
		Export: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "export to clipboard"),
		),
		ProfileTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next profile"),
		),
		ProfileBackTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev profile"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle file"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search files"),
		),
		TierView: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "cycle tier view"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "select all visible"),
		),
		SelectNone: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "deselect all visible"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear filter"),
		),
	}
}

// ShortHelp returns key bindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit, k.Generate}
}

// FullHelp returns key bindings for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Toggle},
		{k.Search, k.TierView, k.SelectAll, k.SelectNone, k.ClearFilter},
		{k.Generate, k.Preview, k.Save, k.Export},
		{k.ProfileTab, k.ProfileBackTab},
		{k.Help, k.Quit},
	}
}
