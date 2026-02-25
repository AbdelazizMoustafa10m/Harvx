package filetree

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/harvx/harvx/internal/tui/tuimsg"
)

// keyMap defines the file tree key bindings. These are local to the filetree
// package; global keys (quit, help) are handled by the parent model.
var keyMap = struct {
	Up       key.Binding
	Down     key.Binding
	Right    key.Binding
	Left     key.Binding
	Toggle   key.Binding
	Enter    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
}{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "move down"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("right/l", "expand"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("left/h", "collapse"),
	),
	Toggle: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "expand/collapse"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("pgdown", "page down"),
	),
	Home: key.NewBinding(
		key.WithKeys("home", "g"),
		key.WithHelp("home/g", "first"),
	),
	End: key.NewBinding(
		key.WithKeys("end", "G"),
		key.WithHelp("end/G", "last"),
	),
}

// Update handles key events, directory loading messages, and window resize
// events for the file tree model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case DirLoadedMsg:
		return m.handleDirLoaded(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}

// handleKey processes keyboard input for tree navigation and toggling.
func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keyMap.Up):
		m.cursor--
		m.clampCursor()
		m.adjustScroll()
		return m, nil

	case key.Matches(msg, keyMap.Down):
		m.cursor++
		m.clampCursor()
		m.adjustScroll()
		return m, nil

	case key.Matches(msg, keyMap.Right):
		return m.handleExpand()

	case key.Matches(msg, keyMap.Left):
		return m.handleCollapse()

	case key.Matches(msg, keyMap.Toggle):
		return m.handleToggle()

	case key.Matches(msg, keyMap.Enter):
		return m.handleEnter()

	case key.Matches(msg, keyMap.PageUp):
		step := m.height
		if step <= 0 {
			step = 10
		}
		m.cursor -= step
		m.clampCursor()
		m.adjustScroll()
		return m, nil

	case key.Matches(msg, keyMap.PageDown):
		step := m.height
		if step <= 0 {
			step = 10
		}
		m.cursor += step
		m.clampCursor()
		m.adjustScroll()
		return m, nil

	case key.Matches(msg, keyMap.Home):
		m.cursor = 0
		m.adjustScroll()
		return m, nil

	case key.Matches(msg, keyMap.End):
		if len(m.visible) > 0 {
			m.cursor = len(m.visible) - 1
		}
		m.adjustScroll()
		return m, nil
	}

	return m, nil
}

// handleExpand expands a collapsed directory. For files, this is a no-op.
// If the directory has not been loaded yet, a lazy load command is returned.
func (m Model) handleExpand() (Model, tea.Cmd) {
	node := m.cursorNode()
	if node == nil || !node.IsDir {
		return m, nil
	}

	if node.Expanded {
		return m, nil
	}

	node.Expanded = true

	if !node.loaded && !m.loading[node.Path] {
		m.loading[node.Path] = true
		m.refreshVisible()
		m.clampCursor()
		m.adjustScroll()
		return m, loadDirCmd(m.rootDir, node.Path, m.ignorer)
	}

	m.refreshVisible()
	m.clampCursor()
	m.adjustScroll()
	return m, nil
}

// handleCollapse collapses an expanded directory. On files, it jumps to the
// parent directory.
func (m Model) handleCollapse() (Model, tea.Cmd) {
	node := m.cursorNode()
	if node == nil {
		return m, nil
	}

	if node.IsDir && node.Expanded {
		node.Expanded = false
		m.refreshVisible()
		m.clampCursor()
		m.adjustScroll()
		return m, nil
	}

	// On a file or collapsed dir, jump to parent.
	if node.Parent != nil && node.Parent != m.root {
		for i, v := range m.visible {
			if v == node.Parent {
				m.cursor = i
				break
			}
		}
		m.adjustScroll()
	}

	return m, nil
}

// handleToggle toggles the inclusion state of the current node and sends a
// FileToggledMsg for each affected file.
func (m Model) handleToggle() (Model, tea.Cmd) {
	node := m.cursorNode()
	if node == nil {
		return m, nil
	}

	node.Toggle()

	// Collect all affected file paths and their new states.
	var cmds []tea.Cmd
	if node.IsDir {
		// Send messages for all file descendants.
		node.visitFiles(func(n *Node) {
			cmds = append(cmds, func() tea.Msg {
				return tuimsg.FileToggledMsg{
					Path:     n.Path,
					Included: n.Included == Included,
				}
			})
		})
	} else {
		included := node.Included == Included
		cmds = append(cmds, func() tea.Msg {
			return tuimsg.FileToggledMsg{
				Path:     node.Path,
				Included: included,
			}
		})
	}

	return m, tea.Batch(cmds...)
}

// handleEnter toggles expand/collapse on directories. For files, this is a no-op.
func (m Model) handleEnter() (Model, tea.Cmd) {
	node := m.cursorNode()
	if node == nil || !node.IsDir {
		return m, nil
	}

	if node.Expanded {
		node.Expanded = false
		m.refreshVisible()
		m.clampCursor()
		m.adjustScroll()
		return m, nil
	}

	return m.handleExpand()
}

// handleDirLoaded processes the result of a lazy directory load, attaching
// children to the target directory node.
func (m Model) handleDirLoaded(msg DirLoadedMsg) (Model, tea.Cmd) {
	delete(m.loading, msg.Path)

	if msg.Err != nil {
		m.logger.Error("directory load failed",
			"path", msg.Path,
			"error", msg.Err,
		)
		return m, nil
	}

	// Find the target directory node.
	var target *Node
	if msg.Path == "" {
		target = m.root
	} else {
		target = m.root.FindByPath(msg.Path)
	}

	if target == nil {
		m.logger.Warn("directory node not found for loaded children",
			"path", msg.Path,
		)
		return m, nil
	}

	// Attach children.
	target.Children = nil
	for _, child := range msg.Children {
		target.AddChild(child)
	}
	target.SortChildren()
	target.SetLoaded(true)

	if !m.ready {
		m.ready = true
	}

	m.refreshVisible()
	m.clampCursor()
	m.adjustScroll()
	return m, nil
}

// visitFiles calls fn for each file node (non-directory) in the subtree.
func (n *Node) visitFiles(fn func(*Node)) {
	for _, child := range n.Children {
		if child.IsDir {
			child.visitFiles(fn)
		} else {
			fn(child)
		}
	}
}
