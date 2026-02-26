package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/harvx/harvx/internal/tui/filetree"
)

// sendKey is a helper that sends a key message to a model and returns the
// updated model. It asserts that the returned tea.Model is the expected type.
func sendKey(t *testing.T, m Model, k tea.KeyMsg) Model {
	t.Helper()
	updated, _ := m.Update(k)
	return updated.(Model)
}

// sendKeyWithCmd is like sendKey but also returns the command.
func sendKeyWithCmd(t *testing.T, m Model, k tea.KeyMsg) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(k)
	return updated.(Model), cmd
}

// sendMsg sends an arbitrary message to a model and returns the updated model.
func sendMsg(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	updated, _ := m.Update(msg)
	return updated.(Model)
}

// sendMsgWithCmd sends an arbitrary message and returns both model and cmd.
func sendMsgWithCmd(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

// drainCmds executes all commands in a batch and feeds their resulting messages
// back into the model, up to maxIterations to prevent infinite loops.
// Returns the final model state after processing all messages.
func drainCmds(t *testing.T, m Model, cmd tea.Cmd, maxIterations int) Model {
	t.Helper()
	for i := 0; i < maxIterations && cmd != nil; i++ {
		msg := cmd()
		if msg == nil {
			break
		}
		// Skip tea.QuitMsg -- don't feed it back.
		if _, ok := msg.(tea.QuitMsg); ok {
			break
		}
		var nextCmd tea.Cmd
		m, nextCmd = sendMsgWithCmd(t, m, msg)
		cmd = nextCmd
	}
	return m
}

// keyMsg creates a tea.KeyMsg for a rune key (e.g., 'q', '?', 'p').
func keyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// specialKeyMsg creates a tea.KeyMsg for a special key (e.g., tea.KeyEnter).
func specialKeyMsg(k tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: k}
}

// windowSizeMsg returns a standard window size message for testing.
func windowSizeMsg(w, h int) tea.WindowSizeMsg {
	return tea.WindowSizeMsg{Width: w, Height: h}
}

// buildTestTree creates a tree structure programmatically for integration
// testing. It builds a root node with a realistic directory layout without
// hitting the filesystem.
func buildTestTree() *filetree.Node {
	root := filetree.NewNode("", "root", true)
	root.SetLoaded(true)

	// src/ directory
	src := filetree.NewNode("src", "src", true)
	root.AddChild(src)
	src.SetLoaded(true)
	src.Expanded = true

	mainGo := filetree.NewNode("src/main.go", "main.go", false)
	mainGo.TokenCount = 500
	mainGo.Tier = 1
	src.AddChild(mainGo)

	utilsGo := filetree.NewNode("src/utils.go", "utils.go", false)
	utilsGo.TokenCount = 300
	utilsGo.Tier = 2
	src.AddChild(utilsGo)

	appTs := filetree.NewNode("src/app.ts", "app.ts", false)
	appTs.TokenCount = 800
	appTs.Tier = 1
	src.AddChild(appTs)

	// docs/ directory
	docs := filetree.NewNode("docs", "docs", true)
	root.AddChild(docs)
	docs.SetLoaded(true)
	docs.Expanded = true

	readme := filetree.NewNode("docs/README.md", "README.md", false)
	readme.TokenCount = 200
	readme.Tier = 4
	docs.AddChild(readme)

	// Top-level files
	makefile := filetree.NewNode("Makefile", "Makefile", false)
	makefile.TokenCount = 150
	makefile.Tier = 0
	makefile.IsPriority = true
	root.AddChild(makefile)

	goMod := filetree.NewNode("go.mod", "go.mod", false)
	goMod.TokenCount = 50
	goMod.Tier = 0
	root.AddChild(goMod)

	// tests/ directory
	tests := filetree.NewNode("tests", "tests", true)
	root.AddChild(tests)
	tests.SetLoaded(true)
	tests.Expanded = true

	testMain := filetree.NewNode("tests/main_test.go", "main_test.go", false)
	testMain.TokenCount = 400
	testMain.Tier = 3
	tests.AddChild(testMain)

	root.SortChildren()
	for _, child := range root.Children {
		if child.IsDir {
			child.SortChildren()
		}
	}

	return root
}

// mustNewModelWithTree creates a model with a pre-built file tree for testing
// that does not require filesystem access. This injects the tree directly.
func mustNewModelWithTree(t *testing.T) Model {
	t.Helper()
	m := mustNewModel(t)

	// Build and inject the test tree.
	tree := buildTestTree()
	m.fileTree = filetree.NewWithRoot(tree, ".")
	m.statsPanel.SetTreeRoot(tree)

	// Mark as ready with a reasonable window size and compute styles.
	m.ready = true
	m.width = 120
	m.height = 40
	m.styles = NewStyles(true, 120, 40)
	// Inner panel sizes: left = 78-4 = 74, right = 41-4 = 37, height = 38-2 = 36
	m.fileTree.SetSize(m.styles.LeftPanelWidth-4, m.styles.ContentHeight-2)
	m.statsPanel.SetSize(m.styles.RightPanelWidth-4, m.styles.ContentHeight-2)

	return m
}

// mustNewModelWithTreeAndProfiles creates a model with a pre-built tree and
// multiple profiles.
func mustNewModelWithTreeAndProfiles(t *testing.T, profiles []string) Model {
	t.Helper()
	m := mustNewModelWithProfiles(t, profiles)

	tree := buildTestTree()
	m.fileTree = filetree.NewWithRoot(tree, ".")
	m.statsPanel.SetTreeRoot(tree)

	m.ready = true
	m.width = 120
	m.height = 40
	m.styles = NewStyles(true, 120, 40)
	m.fileTree.SetSize(m.styles.LeftPanelWidth-4, m.styles.ContentHeight-2)
	m.statsPanel.SetSize(m.styles.RightPanelWidth-4, m.styles.ContentHeight-2)

	return m
}

// assertViewContains asserts that the model's View() output contains the
// expected string.
func assertViewContains(t *testing.T, m Model, expected string) {
	t.Helper()
	view := m.View()
	if !strings.Contains(view, expected) {
		t.Errorf("View() does not contain %q\nView output:\n%s", expected, view)
	}
}

// assertViewNotContains asserts that the model's View() output does NOT
// contain the given string.
func assertViewNotContains(t *testing.T, m Model, unexpected string) {
	t.Helper()
	view := m.View()
	if strings.Contains(view, unexpected) {
		t.Errorf("View() should not contain %q\nView output:\n%s", unexpected, view)
	}
}
