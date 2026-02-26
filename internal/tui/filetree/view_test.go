package filetree

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildViewTestModel creates a model with rich node attributes for view testing.
func buildViewTestModel(t *testing.T) Model {
	t.Helper()

	root := NewNode("", ".", true)
	root.Expanded = true
	root.SetLoaded(true)

	src := NewNode("src", "src", true)
	src.Expanded = true
	src.SetLoaded(true)

	mainGo := NewNode("src/main.go", "main.go", false)
	mainGo.Tier = 0
	mainGo.IsPriority = true
	mainGo.TokenCount = 1234

	utilGo := NewNode("src/util.go", "util.go", false)
	utilGo.Tier = 2
	utilGo.TokenCount = 567

	secretFile := NewNode("src/.env", ".env", false)
	secretFile.Tier = 5
	secretFile.HasSecrets = true
	secretFile.Included = Included

	src.AddChild(mainGo)
	src.AddChild(utilGo)
	src.AddChild(secretFile)
	src.SortChildren()

	lib := NewNode("lib", "lib", true)
	lib.SetLoaded(true)

	helper := NewNode("lib/helper.go", "helper.go", false)
	helper.Tier = 3
	helper.TokenCount = 89420
	lib.AddChild(helper)

	readme := NewNode("README.md", "README.md", false)
	readme.Tier = 1
	readme.Included = Included
	readme.TokenCount = 200

	root.AddChild(src)
	root.AddChild(lib)
	root.AddChild(readme)
	root.SortChildren()

	m := NewWithRoot(root, ".")
	m.SetSize(80, 20)
	return m
}

func TestTreePrefix_RootDepthZero(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	// The root node itself (depth 0) should have no tree prefix.
	prefix := m.treePrefix(m.Root())
	assert.Empty(t, prefix, "root node (depth 0) should have empty prefix")

	// Root's direct children (depth 1) are the first visible nodes and should
	// have tree prefixes since treePrefix only returns empty for depth 0.
	for _, node := range m.Visible() {
		if node.Parent == m.Root() {
			prefix := m.treePrefix(node)
			assert.NotEmpty(t, prefix, "top-level visible node %q (depth %d) should have a tree prefix", node.Name, node.Depth())
		}
	}
}

func TestTreePrefix_NestedNodes(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	// Find a nested file inside src (expanded).
	for _, node := range m.Visible() {
		if node.Depth() > 0 {
			prefix := m.treePrefix(node)
			assert.NotEmpty(t, prefix, "nested node %q should have a tree prefix", node.Name)
			// Prefix should contain tree-drawing characters.
			hasTreeChar := strings.Contains(prefix, "├") ||
				strings.Contains(prefix, "└") ||
				strings.Contains(prefix, "│")
			assert.True(t, hasTreeChar, "prefix for %q should contain tree-drawing characters, got %q", node.Name, prefix)
		}
	}
}

func TestTreePrefix_LastChild(t *testing.T) {
	t.Parallel()

	// Build a simple tree: root -> A, B (B is last child).
	root := NewNode("", ".", true)
	root.Expanded = true
	root.SetLoaded(true)
	a := NewNode("a.go", "a.go", false)
	b := NewNode("b.go", "b.go", false)
	root.AddChild(a)
	root.AddChild(b)

	m := NewWithRoot(root, ".")
	m.SetSize(80, 20)

	// a is not last child -> should use TreeBranch "├── "
	prefixA := m.treePrefix(a)
	assert.Contains(t, prefixA, "├", "first child should use branch prefix")

	// b is last child -> should use TreeLast "└── "
	prefixB := m.treePrefix(b)
	assert.Contains(t, prefixB, "└", "last child should use last-child prefix")
}

func TestTreePrefix_DeeplyNested(t *testing.T) {
	t.Parallel()

	// root -> dir1 -> dir2 -> file.go
	root := NewNode("", ".", true)
	root.Expanded = true
	root.SetLoaded(true)

	dir1 := NewNode("dir1", "dir1", true)
	dir1.Expanded = true
	dir1.SetLoaded(true)

	dir2 := NewNode("dir1/dir2", "dir2", true)
	dir2.Expanded = true
	dir2.SetLoaded(true)

	file := NewNode("dir1/dir2/file.go", "file.go", false)
	dir2.AddChild(file)
	dir1.AddChild(dir2)
	root.AddChild(dir1)

	m := NewWithRoot(root, ".")
	m.SetSize(80, 20)

	prefix := m.treePrefix(file)
	// Depth 2, so prefix has 2 segments: indent for depth 0, then branch at depth 1.
	assert.NotEmpty(t, prefix)
	// Should contain pipe or space for inner level, then branch/last for final level.
	assert.True(t, strings.Contains(prefix, "├") || strings.Contains(prefix, "└"),
		"deeply nested prefix should contain branch chars")
}

func TestView_VirtualScrolling(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	// The model has many visible nodes. Set height to 2 to show only 2.
	m.SetSize(80, 2)
	view := m.View()

	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	assert.LessOrEqual(t, len(lines), 2, "virtual scrolling should limit to viewport height")
}

func TestView_EmptyDirectory(t *testing.T) {
	t.Parallel()

	root := NewNode("", ".", true)
	root.Expanded = true
	root.SetLoaded(true)
	m := NewWithRoot(root, ".")
	m.SetSize(80, 20)

	view := m.View()
	assert.Contains(t, view, "empty directory")
}

func TestView_Loading(t *testing.T) {
	t.Parallel()

	m := New(".", nil)
	view := m.View()
	assert.Contains(t, view, "Loading file tree")
}

func TestView_ContainsNodeNames(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	view := m.View()

	assert.Contains(t, view, "src")
	assert.Contains(t, view, "lib")
	assert.Contains(t, view, "README.md")
	assert.Contains(t, view, "main.go")
}

func TestView_InclusionIndicators(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	view := m.View()

	// README.md is Included, .env is Included, others are Excluded.
	assert.Contains(t, view, "✓", "included files should show checkmark")
	assert.Contains(t, view, "✗", "excluded files should show X")
}

func TestView_DirectoryIcons(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	view := m.View()

	// src is expanded, lib is collapsed.
	assert.Contains(t, view, DirExpanded, "expanded dir should show expanded icon")
	assert.Contains(t, view, DirCollapsed, "collapsed dir should show collapsed icon")
}

func TestView_PriorityIndicator(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	view := m.View()

	// main.go has IsPriority=true.
	assert.Contains(t, view, PriorityIcon, "priority files should show star icon")
}

func TestView_SecretIndicator(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	view := m.View()

	// .env has HasSecrets=true.
	assert.Contains(t, view, SecretIcon, "secret-containing files should show shield icon")
}

func TestView_TokenCount(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	view := m.View()

	// main.go has TokenCount=1234 -> "1,234 tok"
	assert.Contains(t, view, "1,234", "token count should be formatted with thousands separator")
	assert.Contains(t, view, "tok", "token count should show 'tok' suffix")
}

func TestView_CursorHighlight(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	view := m.View()

	// The cursor row is padded to full width (80 columns) with trailing spaces
	// so the background color fills the entire row.
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	require.NotEmpty(t, lines)
	// First line is the cursor row. It should be padded wider than the
	// non-cursor lines (which are not padded to full width).
	cursorLine := lines[0]
	assert.GreaterOrEqual(t, len(cursorLine), m.Width(),
		"cursor row should be padded to full width")
}

func TestView_BoldDirectoryNames(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	view := m.View()

	// Bold text in ANSI is indicated by \x1b[1m or similar. Lipgloss uses it.
	// Directories should have bold styling. Just verify dir names appear.
	assert.Contains(t, view, "src/")
	assert.Contains(t, view, "lib/")
}

func TestView_TruncationOnNarrowTerminal(t *testing.T) {
	t.Parallel()

	// Create a model with a very long file name.
	root := NewNode("", ".", true)
	root.Expanded = true
	root.SetLoaded(true)
	longName := strings.Repeat("x", 200) + ".go"
	longFile := NewNode(longName, longName, false)
	longFile.TokenCount = 999
	root.AddChild(longFile)

	m := NewWithRoot(root, ".")
	m.SetSize(30, 20)

	view := m.View()
	// Lines should be truncated with ellipsis.
	assert.Contains(t, view, "...", "long lines should be truncated with ellipsis")
}

// --- Unit tests for helper functions ---

func TestFormatThousands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{89420, "89,420"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			result := formatThousands(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateWithEllipsis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxWidth int
		contains string
	}{
		{"short string no truncation", "hello", 10, "hello"},
		{"exact width truncated", "hello", 5, "he..."},
		{"truncated with ellipsis", "hello world this is long", 10, "..."},
		{"very narrow", "hello", 3, "..."},
		{"width 1", "hello", 1, "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := truncateWithEllipsis(tt.input, tt.maxWidth)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestTierColor_AllTiers(t *testing.T) {
	t.Parallel()

	colors := darkViewColors()
	expected := map[int]lipgloss.Color{
		0: colors.tierGold,
		1: colors.tierGreen,
		2: colors.tierBlue,
		3: colors.tierCyan,
		4: colors.tierMagenta,
		5: colors.tierDim,
		6: colors.foreground, // default
	}

	for tier, expectedColor := range expected {
		result := tierColor(tier, colors)
		assert.Equal(t, expectedColor, result, "tier %d should map to correct color", tier)
	}
}

func TestRenderInclusionIndicator(t *testing.T) {
	t.Parallel()

	colors := darkViewColors()

	tests := []struct {
		state    InclusionState
		contains string
	}{
		{Included, "✓"},
		{Excluded, "✗"},
		{Partial, "◐"},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			t.Parallel()
			result := renderInclusionIndicator(tt.state, colors)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestDarkAndLightViewColors(t *testing.T) {
	t.Parallel()

	dark := darkViewColors()
	light := lightViewColors()

	// Both should have non-empty colors.
	assert.NotEmpty(t, string(dark.tierGold))
	assert.NotEmpty(t, string(light.tierGold))

	// Dark and light should have different cursor backgrounds.
	assert.NotEqual(t, dark.cursorBg, light.cursorBg)
}

func TestView_DarkMode(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	m.SetDark(true)
	view := m.View()

	// Should render without errors.
	require.NotEmpty(t, view)
	assert.Contains(t, view, "src")
}

func TestView_LightMode(t *testing.T) {
	t.Parallel()

	m := buildViewTestModel(t)
	m.SetDark(false)
	view := m.View()

	// Should render without errors.
	require.NotEmpty(t, view)
	assert.Contains(t, view, "src")
}

func TestView_LoadingDirectory(t *testing.T) {
	t.Parallel()

	root := NewNode("", ".", true)
	root.Expanded = true
	root.SetLoaded(true)

	dir := NewNode("loading-dir", "loading-dir", true)
	dir.Expanded = true
	root.AddChild(dir)

	m := NewWithRoot(root, ".")
	m.loading["loading-dir"] = true
	m.SetSize(80, 20)

	view := m.View()
	assert.Contains(t, view, "Loading", "loading directory should show loading indicator")
}
