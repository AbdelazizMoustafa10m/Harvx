package filetree

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/tui/tuimsg"
)

// buildTestModel creates a Model with a pre-populated tree for testing.
// The tree looks like:
//
//	root (expanded)
//	  src/ (expanded)
//	    main.go
//	    util.go
//	  lib/ (collapsed)
//	    helper.go
//	  README.md
func buildTestModel(t *testing.T) Model {
	t.Helper()

	m := New(".", nil)

	// Build tree manually.
	src := NewNode("src", "src", true)
	src.Expanded = true
	src.SetLoaded(true)
	mainGo := NewNode("src/main.go", "main.go", false)
	utilGo := NewNode("src/util.go", "util.go", false)
	src.AddChild(mainGo)
	src.AddChild(utilGo)

	lib := NewNode("lib", "lib", true)
	lib.SetLoaded(true)
	helper := NewNode("lib/helper.go", "helper.go", false)
	lib.AddChild(helper)

	readme := NewNode("README.md", "README.md", false)

	m.root.Children = nil
	m.root.AddChild(src)
	m.root.AddChild(lib)
	m.root.AddChild(readme)
	m.root.SortChildren()

	m.ready = true
	m.height = 20
	m.width = 80
	m.refreshVisible()

	return m
}

func TestNew(t *testing.T) {
	t.Parallel()

	m := New("/tmp/test", nil)
	assert.Equal(t, "/tmp/test", m.rootDir)
	assert.NotNil(t, m.root)
	assert.True(t, m.root.IsDir)
	assert.True(t, m.root.Expanded)
	assert.NotNil(t, m.loading)
	assert.False(t, m.ready)
}

func TestModel_Init(t *testing.T) {
	t.Parallel()

	m := New(".", nil)
	cmd := m.Init()
	assert.NotNil(t, cmd, "Init should return a command to load the root directory")
}

func TestModel_SetSize(t *testing.T) {
	t.Parallel()

	m := New(".", nil)
	m.SetSize(120, 40)
	assert.Equal(t, 120, m.Width())
	assert.Equal(t, 40, m.Height())
}

func TestModel_SelectedFiles_Empty(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	selected := m.SelectedFiles()
	assert.Empty(t, selected)
}

func TestModel_CursorNode(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	node := m.cursorNode()
	require.NotNil(t, node)
	// First visible node after sorting: lib (dir), README.md (file), src (dir), then src children.
	// Actually sorting: dirs first (lib, src), then files (README.md).
	// lib is collapsed so its children are hidden.
	// src is expanded so main.go, util.go are visible.
	// Visible order: lib, src, src/main.go, src/util.go, README.md
	assert.Equal(t, "lib", node.Name)
}

// --- Navigation tests ---

func TestModel_NavigateDown(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	// Visible: lib, src, main.go, util.go, README.md
	assert.Equal(t, 0, m.Cursor())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.Cursor())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 2, m.Cursor())
}

func TestModel_NavigateUp(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	m.cursor = 3
	m.adjustScroll()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 2, m.Cursor())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 1, m.Cursor())
}

func TestModel_NavigateUp_ClampsAtZero(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	assert.Equal(t, 0, m.Cursor())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.Cursor())
}

func TestModel_NavigateDown_ClampsAtEnd(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	lastIdx := len(m.Visible()) - 1
	m.cursor = lastIdx

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, lastIdx, m.Cursor())
}

func TestModel_Home(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	m.cursor = 3

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	assert.Equal(t, 0, m.Cursor())
}

func TestModel_End(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	assert.Equal(t, len(m.Visible())-1, m.Cursor())
}

func TestModel_PageDown(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	m.height = 2 // small viewport

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Equal(t, 2, m.Cursor())
}

func TestModel_PageUp(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	m.height = 2
	m.cursor = 4

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	assert.Equal(t, 2, m.Cursor())
}

// --- Expand/Collapse tests ---

func TestModel_ExpandCollapse_Enter(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	// Cursor is on "lib" (collapsed dir).
	assert.Equal(t, "lib", m.cursorNode().Name)
	assert.False(t, m.cursorNode().Expanded)

	// Expand with Enter.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, m.cursorNode().Expanded)

	// Collapse with Enter.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m.cursorNode().Expanded)
}

func TestModel_ExpandRight_CollapseLeft(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	// Cursor on lib (collapsed).
	assert.Equal(t, "lib", m.cursorNode().Name)

	// Right expands.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.True(t, m.cursorNode().Expanded)

	// Left collapses.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.False(t, m.cursorNode().Expanded)
}

func TestModel_Right_NoOpOnFile(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	// Move to a file node.
	m.cursor = 4 // README.md
	node := m.cursorNode()
	require.NotNil(t, node)
	assert.False(t, node.IsDir)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	// No change; still on the same file.
	assert.Equal(t, 4, m.Cursor())
}

func TestModel_Left_JumpsToParent(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	// Move to src/main.go (inside expanded src).
	// Visible: lib(0), src(1), main.go(2), util.go(3), README.md(4)
	m.cursor = 2
	node := m.cursorNode()
	require.NotNil(t, node)
	assert.Equal(t, "main.go", node.Name)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	// Should jump to parent "src".
	assert.Equal(t, 1, m.Cursor())
	assert.Equal(t, "src", m.cursorNode().Name)
}

func TestModel_ExpandMakesChildrenVisible(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	// lib is at cursor 0, collapsed.
	before := len(m.Visible())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	after := len(m.Visible())

	// lib has one child (helper.go), so visible count increases by 1.
	assert.Equal(t, before+1, after)
}

func TestModel_CollapseHidesChildren(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	// src is at cursor 1, expanded with 2 children.
	before := len(m.Visible())
	m.cursor = 1
	assert.Equal(t, "src", m.cursorNode().Name)
	assert.True(t, m.cursorNode().Expanded)

	// Collapse src.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	after := len(m.Visible())

	// Two children should be hidden.
	assert.Equal(t, before-2, after)
}

// --- Toggle tests ---

func TestModel_Toggle_File(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	// Move to README.md (file).
	m.cursor = 4
	assert.Equal(t, "README.md", m.cursorNode().Name)

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, Included, m.cursorNode().Included)
	require.NotNil(t, cmd, "toggle should produce a FileToggledMsg command")

	// Execute the batch and check messages.
	msgs := executeBatch(cmd)
	require.Len(t, msgs, 1)
	ftMsg, ok := msgs[0].(tuimsg.FileToggledMsg)
	require.True(t, ok)
	assert.Equal(t, "README.md", ftMsg.Path)
	assert.True(t, ftMsg.Included)
}

func TestModel_Toggle_Directory(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	// Cursor on src (expanded dir with 2 files).
	m.cursor = 1
	assert.Equal(t, "src", m.cursorNode().Name)

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, Included, m.cursorNode().Included)

	// All children should be included.
	for _, child := range m.cursorNode().Children {
		assert.Equal(t, Included, child.Included)
	}

	require.NotNil(t, cmd)
	msgs := executeBatch(cmd)
	// Should produce a FileToggledMsg for each file descendant.
	assert.Len(t, msgs, 2) // main.go and util.go
}

// --- DirLoadedMsg tests ---

func TestModel_DirLoadedMsg(t *testing.T) {
	t.Parallel()

	m := New(".", nil)
	m.height = 20
	m.width = 80

	children := []*Node{
		NewNode("src", "src", true),
		NewNode("README.md", "README.md", false),
	}

	m, _ = m.Update(DirLoadedMsg{
		Path:     "",
		Children: children,
	})

	assert.True(t, m.Ready())
	assert.Len(t, m.Root().Children, 2)
	assert.True(t, m.Root().Loaded())
}

func TestModel_DirLoadedMsg_Error(t *testing.T) {
	t.Parallel()

	m := New(".", nil)
	m.height = 20
	m.width = 80

	m, _ = m.Update(DirLoadedMsg{
		Path: "nonexistent",
		Err:  assert.AnError,
	})

	// Should not crash; model remains functional.
	assert.False(t, m.Ready())
}

// --- Scroll offset tests ---

func TestModel_ScrollKeepsCursorVisible(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	m.height = 3 // Can show 3 nodes at a time.

	// Move down to force scrolling.
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Cursor should be at index 4, offset adjusted.
	assert.Equal(t, 4, m.Cursor())
	assert.True(t, m.Offset() <= m.Cursor())
	assert.True(t, m.Cursor() < m.Offset()+m.Height())
}

func TestModel_ScrollOnPageDown(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	m.height = 2

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Equal(t, 2, m.Cursor())
	// Offset should be adjusted so cursor is visible.
	assert.True(t, m.Offset() <= m.Cursor())
}

// --- View tests ---

func TestModel_View_Loading(t *testing.T) {
	t.Parallel()

	m := New(".", nil)
	view := m.View()
	assert.Contains(t, view, "Loading file tree")
}

func TestModel_View_Empty(t *testing.T) {
	t.Parallel()

	m := New(".", nil)
	m.ready = true
	m.refreshVisible()

	view := m.View()
	assert.Contains(t, view, "empty directory")
}

func TestModel_View_WithContent(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	view := m.View()

	// Should show tree nodes.
	assert.Contains(t, view, "lib")
	assert.Contains(t, view, "src")
	assert.Contains(t, view, "main.go")
	assert.Contains(t, view, "README.md")
}

func TestModel_View_CursorIndicator(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	view := m.View()
	assert.Contains(t, view, "> ")
}

// --- WindowSizeMsg ---

func TestModel_WindowSizeMsg(t *testing.T) {
	t.Parallel()

	m := buildTestModel(t)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	assert.Equal(t, 100, m.Width())
	assert.Equal(t, 50, m.Height())
}

// --- Helper ---

// executeBatch executes a tea.Cmd that may be a batch and collects all messages.
func executeBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}

	msg := cmd()
	if msg == nil {
		return nil
	}

	// Check if it is a BatchMsg (a slice of commands).
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			if c != nil {
				msgs = append(msgs, executeBatch(c)...)
			}
		}
		return msgs
	}

	return []tea.Msg{msg}
}
