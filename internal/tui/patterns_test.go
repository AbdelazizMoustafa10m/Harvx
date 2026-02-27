package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/tui/filetree"
)

// buildTree is a test helper that creates a root node and populates it with
// the given children. It sets parent pointers and marks directories as loaded.
func buildTree(t *testing.T, children []*filetree.Node) *filetree.Node {
	t.Helper()
	root := filetree.NewNode(".", ".", true)
	root.Included = filetree.Partial // root defaults to partial
	root.SetLoaded(true)
	for _, c := range children {
		root.AddChild(c)
	}
	return root
}

// makeFile creates a file node with the given path, inclusion state, and tier.
func makeFile(path string, included filetree.InclusionState, tier int) *filetree.Node {
	n := filetree.NewNode(path, path, false)
	n.Included = included
	n.Tier = tier
	return n
}

// makeDir creates a directory node with the given path and inclusion state,
// and populates it with children. Children get their parent set automatically.
func makeDir(path string, state filetree.InclusionState, children []*filetree.Node) *filetree.Node {
	n := filetree.NewNode(path, path, true)
	n.Included = state
	n.SetLoaded(true)
	for _, c := range children {
		n.AddChild(c)
	}
	return n
}

func TestMinimizePatterns_NilRoot(t *testing.T) {
	t.Parallel()

	result := MinimizePatterns(nil)

	assert.Empty(t, result.Include)
	assert.Empty(t, result.Ignore)
	assert.Empty(t, result.PriorityFiles)
	assert.NotNil(t, result.TierFiles)
}

func TestMinimizePatterns_EmptyRoot(t *testing.T) {
	t.Parallel()

	root := buildTree(t, nil)
	result := MinimizePatterns(root)

	assert.Empty(t, result.Include)
	assert.Empty(t, result.Ignore)
	assert.Empty(t, result.PriorityFiles)
}

func TestMinimizePatterns_AllIncludedDirectory(t *testing.T) {
	t.Parallel()

	// A directory where all children are included should produce a single
	// "dir/**" glob instead of listing individual files.
	dir := makeDir("internal", filetree.Included, []*filetree.Node{
		makeFile("internal/a.go", filetree.Included, 1),
		makeFile("internal/b.go", filetree.Included, 2),
		makeFile("internal/c.go", filetree.Included, 0),
	})

	root := buildTree(t, []*filetree.Node{dir})
	root.Included = filetree.Partial

	result := MinimizePatterns(root)

	assert.Equal(t, []string{"internal/**"}, result.Include)
	assert.Empty(t, result.Ignore)
	// Tier info should still be collected from leaves.
	assert.Equal(t, []string{"internal/c.go"}, result.PriorityFiles)
	assert.ElementsMatch(t, []string{"internal/a.go"}, result.TierFiles[1])
	assert.ElementsMatch(t, []string{"internal/b.go"}, result.TierFiles[2])
	assert.ElementsMatch(t, []string{"internal/c.go"}, result.TierFiles[0])
}

func TestMinimizePatterns_AllExcludedDirectory(t *testing.T) {
	t.Parallel()

	// A directory where all children are excluded should produce a single
	// ignore entry "dir/**".
	dir := makeDir("vendor", filetree.Excluded, []*filetree.Node{
		makeFile("vendor/lib.go", filetree.Excluded, 5),
		makeFile("vendor/dep.go", filetree.Excluded, 5),
	})

	root := buildTree(t, []*filetree.Node{dir})
	result := MinimizePatterns(root)

	assert.Empty(t, result.Include)
	assert.Equal(t, []string{"vendor/**"}, result.Ignore)
	assert.Empty(t, result.PriorityFiles)
}

func TestMinimizePatterns_MixedDirectory(t *testing.T) {
	t.Parallel()

	// A directory with mixed inclusion should list individual files.
	dir := makeDir("cmd", filetree.Partial, []*filetree.Node{
		makeFile("cmd/main.go", filetree.Included, 1),
		makeFile("cmd/helper.go", filetree.Excluded, 3),
		makeFile("cmd/util.go", filetree.Included, 2),
	})

	root := buildTree(t, []*filetree.Node{dir})
	result := MinimizePatterns(root)

	assert.ElementsMatch(t, []string{"cmd/main.go", "cmd/util.go"}, result.Include)
	assert.ElementsMatch(t, []string{"cmd/helper.go"}, result.Ignore)
	assert.Empty(t, result.PriorityFiles)
}

func TestMinimizePatterns_NestedDirectories(t *testing.T) {
	t.Parallel()

	// Nested structure:
	//   internal/ (partial)
	//     config/ (included) -- should produce internal/config/**
	//       types.go (included, tier 0)
	//       load.go (included, tier 1)
	//     cli/ (excluded) -- should produce ignore internal/cli/**
	//       root.go (excluded)
	inner1 := makeDir("internal/config", filetree.Included, []*filetree.Node{
		makeFile("internal/config/types.go", filetree.Included, 0),
		makeFile("internal/config/load.go", filetree.Included, 1),
	})
	inner2 := makeDir("internal/cli", filetree.Excluded, []*filetree.Node{
		makeFile("internal/cli/root.go", filetree.Excluded, 2),
	})
	outer := makeDir("internal", filetree.Partial, []*filetree.Node{inner1, inner2})

	root := buildTree(t, []*filetree.Node{outer})
	result := MinimizePatterns(root)

	assert.ElementsMatch(t, []string{"internal/config/**"}, result.Include)
	assert.ElementsMatch(t, []string{"internal/cli/**"}, result.Ignore)
	assert.Equal(t, []string{"internal/config/types.go"}, result.PriorityFiles)
	assert.ElementsMatch(t, []string{"internal/config/types.go"}, result.TierFiles[0])
	assert.ElementsMatch(t, []string{"internal/config/load.go"}, result.TierFiles[1])
}

func TestMinimizePatterns_TopLevelFiles(t *testing.T) {
	t.Parallel()

	// Top-level files (direct children of root) should be listed individually.
	root := buildTree(t, []*filetree.Node{
		makeFile("go.mod", filetree.Included, 0),
		makeFile("go.sum", filetree.Excluded, 5),
		makeFile("README.md", filetree.Included, 1),
	})

	result := MinimizePatterns(root)

	assert.ElementsMatch(t, []string{"README.md", "go.mod"}, result.Include)
	assert.ElementsMatch(t, []string{"go.sum"}, result.Ignore)
	assert.Equal(t, []string{"go.mod"}, result.PriorityFiles)
}

func TestMinimizePatterns_DeeplyNested(t *testing.T) {
	t.Parallel()

	// Three-level nesting with partial at each intermediate level.
	// a/ (partial)
	//   b/ (partial)
	//     c/ (included) -- should emit a/b/c/**
	//       x.go (included, tier 2)
	//     d.go (excluded)
	leaf := makeFile("a/b/c/x.go", filetree.Included, 2)
	cDir := makeDir("a/b/c", filetree.Included, []*filetree.Node{leaf})
	dFile := makeFile("a/b/d.go", filetree.Excluded, 3)
	bDir := makeDir("a/b", filetree.Partial, []*filetree.Node{cDir, dFile})
	aDir := makeDir("a", filetree.Partial, []*filetree.Node{bDir})

	root := buildTree(t, []*filetree.Node{aDir})
	result := MinimizePatterns(root)

	assert.ElementsMatch(t, []string{"a/b/c/**"}, result.Include)
	assert.ElementsMatch(t, []string{"a/b/d.go"}, result.Ignore)
	assert.ElementsMatch(t, []string{"a/b/c/x.go"}, result.TierFiles[2])
}

func TestMinimizePatterns_AllIncludedRoot(t *testing.T) {
	t.Parallel()

	// When every child directory is fully included, each gets its own glob.
	dir1 := makeDir("src", filetree.Included, []*filetree.Node{
		makeFile("src/app.go", filetree.Included, 1),
	})
	dir2 := makeDir("lib", filetree.Included, []*filetree.Node{
		makeFile("lib/util.go", filetree.Included, 2),
	})

	root := buildTree(t, []*filetree.Node{dir1, dir2})
	root.Included = filetree.Included

	result := MinimizePatterns(root)

	assert.ElementsMatch(t, []string{"lib/**", "src/**"}, result.Include)
	assert.Empty(t, result.Ignore)
}

func TestMinimizePatterns_DeterministicOutput(t *testing.T) {
	t.Parallel()

	// Run minimization multiple times and confirm output is identical.
	dir := makeDir("pkg", filetree.Partial, []*filetree.Node{
		makeFile("pkg/z.go", filetree.Included, 1),
		makeFile("pkg/a.go", filetree.Included, 2),
		makeFile("pkg/m.go", filetree.Excluded, 3),
	})
	root := buildTree(t, []*filetree.Node{dir})

	result1 := MinimizePatterns(root)
	result2 := MinimizePatterns(root)

	assert.Equal(t, result1.Include, result2.Include)
	assert.Equal(t, result1.Ignore, result2.Ignore)
	assert.Equal(t, result1.PriorityFiles, result2.PriorityFiles)
}

func TestMinimizePatterns_PriorityFilesFromMultipleDirs(t *testing.T) {
	t.Parallel()

	dir1 := makeDir("src", filetree.Included, []*filetree.Node{
		makeFile("src/critical.go", filetree.Included, 0),
		makeFile("src/normal.go", filetree.Included, 2),
	})
	dir2 := makeDir("docs", filetree.Partial, []*filetree.Node{
		makeFile("docs/README.md", filetree.Included, 0),
		makeFile("docs/draft.md", filetree.Excluded, 4),
	})

	root := buildTree(t, []*filetree.Node{dir1, dir2})
	result := MinimizePatterns(root)

	assert.ElementsMatch(t, []string{"src/critical.go", "docs/README.md"}, result.PriorityFiles)
}

func TestMinimizedPatterns_HasManualIncludes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		includes []string
		want     bool
	}{
		{
			name:     "only directory globs",
			includes: []string{"src/**", "lib/**"},
			want:     false,
		},
		{
			name:     "has individual file",
			includes: []string{"src/**", "go.mod"},
			want:     true,
		},
		{
			name:     "empty includes",
			includes: nil,
			want:     false,
		},
		{
			name:     "only individual files",
			includes: []string{"main.go", "util.go"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := MinimizedPatterns{Include: tt.includes}
			assert.Equal(t, tt.want, p.HasManualIncludes())
		})
	}
}

func TestMinimizePatterns_ExcludedFilesNotDuplicatedUnderExcludedDir(t *testing.T) {
	t.Parallel()

	// When a directory is fully excluded, individual file excludes should NOT
	// appear in the Ignore list -- only the directory glob.
	dir := makeDir("vendor", filetree.Excluded, []*filetree.Node{
		makeFile("vendor/a.go", filetree.Excluded, 5),
		makeFile("vendor/b.go", filetree.Excluded, 5),
		makeFile("vendor/c.go", filetree.Excluded, 5),
	})

	root := buildTree(t, []*filetree.Node{dir})
	result := MinimizePatterns(root)

	// Should only have the directory glob, not individual files.
	require.Len(t, result.Ignore, 1)
	assert.Equal(t, "vendor/**", result.Ignore[0])
}

func TestMinimizePatterns_IncludedDirWithNestedDirs(t *testing.T) {
	t.Parallel()

	// Fully included directory containing subdirectories should emit a single
	// glob and still collect tier info from all nested leaves.
	inner := makeDir("pkg/sub", filetree.Included, []*filetree.Node{
		makeFile("pkg/sub/deep.go", filetree.Included, 0),
	})
	// Mark inner as included since parent is included.
	outer := makeDir("pkg", filetree.Included, []*filetree.Node{
		makeFile("pkg/top.go", filetree.Included, 1),
		inner,
	})

	root := buildTree(t, []*filetree.Node{outer})
	result := MinimizePatterns(root)

	assert.Equal(t, []string{"pkg/**"}, result.Include)
	assert.Empty(t, result.Ignore)
	// Tier info collected from both levels.
	assert.ElementsMatch(t, []string{"pkg/sub/deep.go"}, result.TierFiles[0])
	assert.ElementsMatch(t, []string{"pkg/top.go"}, result.TierFiles[1])
	assert.Equal(t, []string{"pkg/sub/deep.go"}, result.PriorityFiles)
}
