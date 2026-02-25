package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/tui/filetree"
)

func TestWalkTree_AllIncluded(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	root.SetLoaded(true)

	for _, name := range []string{"a.go", "b.go", "c.go"} {
		n := filetree.NewNode(name, name, false)
		n.Included = filetree.Included
		n.Tier = 1
		n.TokenCount = 100
		root.AddChild(n)
	}

	result := tokenCountResult{
		TierBreakdown: make(map[int]int),
		TierTokens:    make(map[int]int),
	}
	walkTree(root, &result)

	assert.Equal(t, 300, result.TotalTokens)
	assert.Equal(t, 3, result.SelectedFiles)
	assert.Equal(t, 3, result.TotalFiles)
	assert.Equal(t, 3, result.TierBreakdown[1])
	assert.Equal(t, 300, result.TierTokens[1])
}

func TestWalkTree_MixedInclusion(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	root.SetLoaded(true)

	included := filetree.NewNode("a.go", "a.go", false)
	included.Included = filetree.Included
	included.Tier = 0
	included.TokenCount = 200
	root.AddChild(included)

	excluded := filetree.NewNode("b.go", "b.go", false)
	excluded.Included = filetree.Excluded
	excluded.Tier = 1
	excluded.TokenCount = 150
	root.AddChild(excluded)

	result := tokenCountResult{
		TierBreakdown: make(map[int]int),
		TierTokens:    make(map[int]int),
	}
	walkTree(root, &result)

	assert.Equal(t, 200, result.TotalTokens)
	assert.Equal(t, 1, result.SelectedFiles)
	assert.Equal(t, 2, result.TotalFiles)
}

func TestWalkTree_NestedDirectories(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	root.SetLoaded(true)

	dir := filetree.NewNode("pkg", "pkg", true)
	dir.SetLoaded(true)
	root.AddChild(dir)

	subdir := filetree.NewNode("pkg/sub", "sub", true)
	subdir.SetLoaded(true)
	dir.AddChild(subdir)

	file := filetree.NewNode("pkg/sub/deep.go", "deep.go", false)
	file.Included = filetree.Included
	file.Tier = 1
	file.TokenCount = 75
	subdir.AddChild(file)

	result := tokenCountResult{
		TierBreakdown: make(map[int]int),
		TierTokens:    make(map[int]int),
	}
	walkTree(root, &result)

	assert.Equal(t, 75, result.TotalTokens)
	assert.Equal(t, 1, result.SelectedFiles)
	assert.Equal(t, 1, result.TotalFiles)
	assert.Equal(t, 1, result.TierBreakdown[1])
}

func TestWalkTree_SecretsCount(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	root.SetLoaded(true)

	secret := filetree.NewNode("env.go", "env.go", false)
	secret.Included = filetree.Included
	secret.HasSecrets = true
	secret.TokenCount = 50
	root.AddChild(secret)

	clean := filetree.NewNode("main.go", "main.go", false)
	clean.Included = filetree.Included
	clean.TokenCount = 100
	root.AddChild(clean)

	// Excluded file with secrets should not count.
	excludedSecret := filetree.NewNode("config.go", "config.go", false)
	excludedSecret.Included = filetree.Excluded
	excludedSecret.HasSecrets = true
	excludedSecret.TokenCount = 30
	root.AddChild(excludedSecret)

	result := tokenCountResult{
		TierBreakdown: make(map[int]int),
		TierTokens:    make(map[int]int),
	}
	walkTree(root, &result)

	assert.Equal(t, 1, result.SecretsFound, "only included files with secrets should count")
	assert.Equal(t, 150, result.TotalTokens)
	assert.Equal(t, 2, result.SelectedFiles)
	assert.Equal(t, 3, result.TotalFiles)
}

func TestWalkTree_EmptyTree(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	root.SetLoaded(true)

	result := tokenCountResult{
		TierBreakdown: make(map[int]int),
		TierTokens:    make(map[int]int),
	}
	walkTree(root, &result)

	assert.Equal(t, 0, result.TotalTokens)
	assert.Equal(t, 0, result.SelectedFiles)
	assert.Equal(t, 0, result.TotalFiles)
}

func TestWalkTree_MultipleTiers(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	root.SetLoaded(true)

	files := []struct {
		name   string
		tier   int
		tokens int
	}{
		{"go.mod", 0, 10},
		{"main.go", 1, 200},
		{"util.go", 1, 150},
		{"test.go", 3, 80},
		{"readme.md", 4, 30},
	}

	for _, f := range files {
		n := filetree.NewNode(f.name, f.name, false)
		n.Included = filetree.Included
		n.Tier = f.tier
		n.TokenCount = f.tokens
		root.AddChild(n)
	}

	result := tokenCountResult{
		TierBreakdown: make(map[int]int),
		TierTokens:    make(map[int]int),
	}
	walkTree(root, &result)

	assert.Equal(t, 470, result.TotalTokens)
	assert.Equal(t, 5, result.SelectedFiles)

	// Per-tier breakdown.
	assert.Equal(t, 1, result.TierBreakdown[0])
	assert.Equal(t, 2, result.TierBreakdown[1])
	assert.Equal(t, 1, result.TierBreakdown[3])
	assert.Equal(t, 1, result.TierBreakdown[4])

	// Per-tier tokens.
	assert.Equal(t, 10, result.TierTokens[0])
	assert.Equal(t, 350, result.TierTokens[1])
	assert.Equal(t, 80, result.TierTokens[3])
	assert.Equal(t, 30, result.TierTokens[4])
}

func TestWalkTree_AllExcluded(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	root.SetLoaded(true)

	for _, name := range []string{"a.go", "b.go", "c.go"} {
		n := filetree.NewNode(name, name, false)
		n.Included = filetree.Excluded
		n.Tier = 1
		n.TokenCount = 100
		root.AddChild(n)
	}

	result := tokenCountResult{
		TierBreakdown: make(map[int]int),
		TierTokens:    make(map[int]int),
	}
	walkTree(root, &result)

	assert.Equal(t, 0, result.TotalTokens, "excluded files should contribute zero tokens")
	assert.Equal(t, 0, result.SelectedFiles, "excluded files should not be counted as selected")
	assert.Equal(t, 3, result.TotalFiles, "all files should still be counted in total")
	assert.Equal(t, 0, result.SecretsFound)
}

func TestWalkTree_LargeTokenCounts(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	root.SetLoaded(true)

	// Simulate a large repo with many tokens.
	for i := 0; i < 100; i++ {
		n := filetree.NewNode("file.go", "file.go", false)
		n.Included = filetree.Included
		n.Tier = 1
		n.TokenCount = 10000
		root.AddChild(n)
	}

	result := tokenCountResult{
		TierBreakdown: make(map[int]int),
		TierTokens:    make(map[int]int),
	}
	walkTree(root, &result)

	assert.Equal(t, 1000000, result.TotalTokens)
	assert.Equal(t, 100, result.SelectedFiles)
	assert.Equal(t, 100, result.TotalFiles)
	assert.Equal(t, 100, result.TierBreakdown[1])
	assert.Equal(t, 1000000, result.TierTokens[1])
}

func TestWalkTree_ZeroTokenFiles(t *testing.T) {
	t.Parallel()

	root := filetree.NewNode("", "root", true)
	root.SetLoaded(true)

	// File with zero tokens (e.g., empty file).
	n := filetree.NewNode("empty.go", "empty.go", false)
	n.Included = filetree.Included
	n.Tier = 2
	n.TokenCount = 0
	root.AddChild(n)

	result := tokenCountResult{
		TierBreakdown: make(map[int]int),
		TierTokens:    make(map[int]int),
	}
	walkTree(root, &result)

	assert.Equal(t, 0, result.TotalTokens)
	assert.Equal(t, 1, result.SelectedFiles, "empty file should still count as selected")
	assert.Equal(t, 1, result.TierBreakdown[2])
	assert.Equal(t, 0, result.TierTokens[2])
}

func TestScheduleDebounce_ReturnsCmd(t *testing.T) {
	t.Parallel()

	cmd := scheduleDebounce(42)
	require.NotNil(t, cmd, "scheduleDebounce should return a non-nil tea.Cmd")
}
