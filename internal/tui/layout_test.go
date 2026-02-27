package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestRenderPanelWithBorder_HasRoundedCorners(t *testing.T) {
	t.Parallel()

	output := RenderPanelWithBorder("hello", "Test", 40, 10, lipgloss.Color("117"), true)

	assert.Contains(t, output, "\u256d", "output should contain top-left rounded corner")
	assert.Contains(t, output, "\u256e", "output should contain top-right rounded corner")
	assert.Contains(t, output, "\u2570", "output should contain bottom-left rounded corner")
	assert.Contains(t, output, "\u256f", "output should contain bottom-right rounded corner")
}

func TestRenderPanelWithBorder_InlineTitle(t *testing.T) {
	t.Parallel()

	output := RenderPanelWithBorder("content", "Files", 40, 10, lipgloss.Color("117"), true)

	// The top border should contain the title inline: "╭─ Files ──...──╮"
	assert.Contains(t, output, "Files", "output should contain the inline title")

	// Verify the top-left corner and dash-space prefix appear before the title.
	lines := strings.Split(output, "\n")
	assert.Greater(t, len(lines), 0, "output should have at least one line")

	topLine := lines[0]
	assert.Contains(t, topLine, "\u256d", "top line should start with rounded corner")
	assert.Contains(t, topLine, "Files", "top line should contain inline title")
}

func TestRenderTooSmall_ContainsMessage(t *testing.T) {
	t.Parallel()

	output := RenderTooSmall(30, 8)

	assert.Contains(t, output, "Terminal too small", "output should contain the warning message")
	assert.Contains(t, output, "40", "output should mention the minimum width")
	assert.Contains(t, output, "12", "output should mention the minimum height")
}

func TestRenderLayout_TooSmallMode(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 30, 8)
	p := LayoutParams{
		Styles:       s,
		FileTreeView: "file tree content here",
		StatsView:    "stats content here",
		TitleBar:     "title",
		StatusBar:    "status",
		Mode:         LayoutTooSmall,
		Width:        30,
		Height:       8,
	}

	result := RenderLayout(p)

	assert.Contains(t, result, "Terminal too small",
		"too-small layout should display the terminal-too-small message")
}

func TestRenderLayout_SinglePanel(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 50, 20)
	p := LayoutParams{
		Styles:       s,
		FileTreeView: "file tree content here",
		StatsView:    "stats content here",
		TitleBar:     RenderTitleBar("v1.0.0", "/my/project", 50, s),
		StatusBar:    RenderStatusBar("default", 50, s),
		Mode:         s.Layout,
		Width:        50,
		Height:       20,
	}

	result := RenderLayout(p)

	assert.Contains(t, result, "file tree content here",
		"single panel layout should contain file tree content")
	assert.NotContains(t, result, "stats content here",
		"single panel layout should not contain stats panel content")
}

func TestRenderLayout_TwoPanel(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 120, 40)
	p := LayoutParams{
		Styles:       s,
		FileTreeView: "file tree content here",
		StatsView:    "stats content here",
		TitleBar:     RenderTitleBar("v1.0.0", "/my/project", 120, s),
		StatusBar:    RenderStatusBar("default", 120, s),
		Mode:         s.Layout,
		Width:        120,
		Height:       40,
	}

	result := RenderLayout(p)

	assert.Contains(t, result, "file tree content here",
		"two-panel layout should contain file tree content")
	assert.Contains(t, result, "stats content here",
		"two-panel layout should contain stats content")
}

func TestRenderStatusBar_ContainsHints(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 120, 40)
	output := RenderStatusBar("default", 120, s)

	assert.Contains(t, output, "q", "status bar should contain quit key")
	assert.Contains(t, output, "quit", "status bar should contain quit hint")
	assert.Contains(t, output, "?", "status bar should contain help key")
	assert.Contains(t, output, "help", "status bar should contain help hint")
	assert.Contains(t, output, "enter", "status bar should contain enter key")
	assert.Contains(t, output, "generate", "status bar should contain generate hint")
	assert.Contains(t, output, "space", "status bar should contain space key")
	assert.Contains(t, output, "toggle", "status bar should contain toggle hint")
	assert.Contains(t, output, "tab", "status bar should contain tab key")
	assert.Contains(t, output, "profile", "status bar should contain profile hint")
}

func TestRenderStatusBar_ContainsProfileName(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 120, 40)
	output := RenderStatusBar("my-custom-profile", 120, s)

	assert.Contains(t, output, "my-custom-profile",
		"status bar should contain the profile name")
}

func TestRenderStatusBar_ZeroWidth(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 120, 40)
	output := RenderStatusBar("default", 0, s)

	assert.Equal(t, "", output, "zero-width status bar should return empty string")
}

func TestRenderTitleBar_ContainsVersion(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 120, 40)
	output := RenderTitleBar("v1.2.3", "/my/project", 120, s)

	assert.Contains(t, output, "Harvx", "title bar should contain the app name")
	assert.Contains(t, output, "v1.2.3", "title bar should contain the version")
}

func TestRenderTitleBar_ContainsDir(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 120, 40)
	output := RenderTitleBar("v1.0.0", "/home/user/my-project", 120, s)

	assert.Contains(t, output, "/home/user/my-project",
		"title bar should contain the directory path")
}

func TestRenderTitleBar_TruncatesLongDir(t *testing.T) {
	t.Parallel()

	longDir := "/very/long/path/that/goes/on/and/on/and/on/forever/deep/nested/directory/structure"
	s := NewStyles(true, 50, 20)
	output := RenderTitleBar("v1.0.0", longDir, 50, s)

	assert.Contains(t, output, "...",
		"title bar should truncate a long directory with ellipsis")
	assert.NotContains(t, output, longDir,
		"the full long directory should not appear when truncated")
}

func TestRenderTitleBar_ZeroWidth(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 120, 40)
	output := RenderTitleBar("v1.0.0", "/my/project", 0, s)

	assert.Equal(t, "", output, "zero-width title bar should return empty string")
}

func TestRenderLayout_Snapshot_120x40(t *testing.T) {
	t.Parallel()

	s := NewStyles(true, 120, 40)
	p := LayoutParams{
		Styles:       s,
		FileTreeView: "main.go\nlib/util.go\nREADME.md",
		StatsView:    "Files: 3\nTokens: 1234",
		TitleBar:     RenderTitleBar("v1.0.0", "/my/project", 120, s),
		StatusBar:    RenderStatusBar("default", 120, s),
		Mode:         s.Layout,
		Width:        120,
		Height:       40,
	}

	result := RenderLayout(p)

	// Verify key structural elements are present in the full render.
	assert.Contains(t, result, "Files",
		"snapshot should contain the Files panel title")
	assert.Contains(t, result, "Harvx",
		"snapshot should contain the app name in the title bar")
	assert.Contains(t, result, "quit",
		"snapshot should contain key hints in the status bar")
	assert.Contains(t, result, "main.go",
		"snapshot should contain file tree content")
	assert.Contains(t, result, "Tokens",
		"snapshot should contain stats content")

	// Verify it has multiple lines (a fully rendered layout).
	lines := strings.Split(result, "\n")
	assert.Greater(t, len(lines), 10,
		"full 120x40 layout should produce more than 10 lines")
}
