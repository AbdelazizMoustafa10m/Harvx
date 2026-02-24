package output

import (
	"fmt"
	"strings"
	"testing"

	"github.com/harvx/harvx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// sampleProjectFiles returns a representative set of file entries for a sample
// Go project. Used across multiple tests and golden tests.
func sampleProjectFiles() []FileEntry {
	return []FileEntry{
		{Path: "cmd/harvx/main.go", Size: 245, TokenCount: 60, Tier: 1},
		{Path: "internal/cli/root.go", Size: 2150, TokenCount: 450, Tier: 1},
		{Path: "internal/cli/version.go", Size: 1024, TokenCount: 200, Tier: 2},
		{Path: "internal/config/types.go", Size: 3584, TokenCount: 800, Tier: 1},
		{Path: "internal/config/loader.go", Size: 2048, TokenCount: 500, Tier: 2},
		{Path: "internal/discovery/walker.go", Size: 4096, TokenCount: 950, Tier: 1},
		{Path: "go.mod", Size: 512, TokenCount: 100, Tier: 3},
		{Path: "go.sum", Size: 8192, TokenCount: 0, Tier: 3},
		{Path: "README.md", Size: 1536, TokenCount: 300, Tier: 2},
	}
}

// collapsibleFiles returns files that create collapsible single-child
// directory chains, in addition to the sample project files.
func collapsibleFiles() []FileEntry {
	files := sampleProjectFiles()
	files = append(files,
		FileEntry{Path: "pkg/utils/helpers/format.go", Size: 512, TokenCount: 120, Tier: 2},
		FileEntry{Path: "pkg/utils/helpers/strings.go", Size: 384, TokenCount: 90, Tier: 2},
		FileEntry{Path: "docs/api/v1/endpoints.md", Size: 2048, TokenCount: 400, Tier: 2},
	)
	return files
}

// collectAllNodes performs a DFS over the tree and returns all nodes.
func collectAllNodes(root *TreeNode) []*TreeNode {
	if root == nil {
		return nil
	}
	var nodes []*TreeNode
	var walk func(n *TreeNode)
	walk = func(n *TreeNode) {
		nodes = append(nodes, n)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return nodes
}

// findChild returns the direct child with the given name, or nil.
func findChild(node *TreeNode, name string) *TreeNode {
	if node == nil {
		return nil
	}
	for _, c := range node.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// childNames returns a slice of Name fields from direct children.
func childNames(node *TreeNode) []string {
	names := make([]string, len(node.Children))
	for i, c := range node.Children {
		names[i] = c.Name
	}
	return names
}

// ---------------------------------------------------------------------------
// TestBuildTree_FlatPathList
// ---------------------------------------------------------------------------

func TestBuildTree_FlatPathList(t *testing.T) {
	files := []FileEntry{
		{Path: "src/main.go", Size: 100, TokenCount: 20},
		{Path: "src/util.go", Size: 200, TokenCount: 40},
		{Path: "README.md", Size: 50, TokenCount: 10},
		{Path: "internal/pkg/types.go", Size: 300, TokenCount: 60},
	}

	root := BuildTree(files)
	require.NotNil(t, root, "root should not be nil")
	assert.True(t, root.IsDir, "root should be a directory")

	// Root should have children representing: internal/pkg, src, README.md
	// Note: internal/pkg may be collapsed since it is a single-child chain.
	allNodes := collectAllNodes(root)
	assert.True(t, len(allNodes) > 0, "tree should contain nodes")

	// Verify that src/ exists as a directory with two file children.
	// src has 2 children so it should NOT be collapsed.
	src := findChild(root, "src")
	require.NotNil(t, src, "src directory should exist as child of root")
	assert.True(t, src.IsDir, "src should be a directory")
	assert.Len(t, src.Children, 2, "src should have 2 children (main.go, util.go)")

	// Verify that types.go exists somewhere in the tree with correct metadata.
	// The internal/pkg chain may be collapsed, so we search all nodes.
	var foundTypes bool
	for _, n := range allNodes {
		if n.Name == "types.go" && !n.IsDir {
			foundTypes = true
			assert.Equal(t, int64(300), n.Size)
			assert.Equal(t, 60, n.TokenCount)
			break
		}
	}
	assert.True(t, foundTypes, "types.go should exist in the tree")

	// Verify README.md is a direct child of root.
	readme := findChild(root, "README.md")
	require.NotNil(t, readme, "README.md should be a direct child of root")
	assert.False(t, readme.IsDir)
}

// ---------------------------------------------------------------------------
// TestBuildTree_Empty
// ---------------------------------------------------------------------------

func TestBuildTree_Empty(t *testing.T) {
	root := BuildTree(nil)
	require.NotNil(t, root, "root should not be nil even for empty input")
	assert.True(t, root.IsDir, "root should be a directory")
	assert.Empty(t, root.Children, "root should have no children for empty input")

	// Also test with an empty (non-nil) slice.
	root2 := BuildTree([]FileEntry{})
	require.NotNil(t, root2, "root should not be nil for empty slice")
	assert.Empty(t, root2.Children, "root should have no children for empty slice")
}

// ---------------------------------------------------------------------------
// TestBuildTree_SingleFile
// ---------------------------------------------------------------------------

func TestBuildTree_SingleFile(t *testing.T) {
	files := []FileEntry{
		{Path: "README.md", Size: 1024, TokenCount: 200, Tier: 2},
	}

	root := BuildTree(files)
	require.NotNil(t, root)
	require.Len(t, root.Children, 1, "root should have exactly one child")

	child := root.Children[0]
	assert.Equal(t, "README.md", child.Name)
	assert.False(t, child.IsDir)
	assert.Equal(t, int64(1024), child.Size)
	assert.Equal(t, 200, child.TokenCount)
	assert.Equal(t, 2, child.Tier)
}

// ---------------------------------------------------------------------------
// TestBuildTree_DeepNesting
// ---------------------------------------------------------------------------

func TestBuildTree_DeepNesting(t *testing.T) {
	// Create deeply nested paths with 15 levels to verify no stack overflow.
	// We add files at multiple depths so the tree has branching points and
	// does not fully collapse into a single node.
	parts := make([]string, 15)
	for i := 0; i < 14; i++ {
		parts[i] = fmt.Sprintf("level%d", i)
	}
	parts[14] = "deep_file.go"
	deepPath := strings.Join(parts, "/")

	// Add a sibling at level 5 to prevent full collapse.
	siblingParts := make([]string, 6)
	copy(siblingParts, parts[:5])
	siblingParts[5] = "sibling.go"
	siblingPath := strings.Join(siblingParts, "/")

	files := []FileEntry{
		{Path: deepPath, Size: 100, TokenCount: 25},
		{Path: siblingPath, Size: 50, TokenCount: 10},
	}

	root := BuildTree(files)
	require.NotNil(t, root, "root should not be nil for deeply nested paths")

	// The tree should build without panic or stack overflow.
	// Verify both files exist somewhere in the tree.
	allNodes := collectAllNodes(root)
	var foundDeep, foundSibling bool
	for _, n := range allNodes {
		if n.Name == "deep_file.go" {
			foundDeep = true
		}
		if n.Name == "sibling.go" {
			foundSibling = true
		}
	}
	assert.True(t, foundDeep, "deep_file.go should exist in the tree")
	assert.True(t, foundSibling, "sibling.go should exist in the tree")

	// Verify rendering also works without panic on deep trees.
	result := RenderTree(root, TreeRenderOpts{})
	assert.Contains(t, result, "deep_file.go")
	assert.Contains(t, result, "sibling.go")
}

// ---------------------------------------------------------------------------
// TestBuildTree_PathCleaning
// ---------------------------------------------------------------------------

func TestBuildTree_PathCleaning(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantName string
	}{
		{
			name:     "leading slash",
			path:     "/src/main.go",
			wantName: "main.go",
		},
		{
			name:     "trailing slash on file treated as dir component",
			path:     "src/main.go/",
			wantName: "main.go",
		},
		{
			name:     "double slashes",
			path:     "src//main.go",
			wantName: "main.go",
		},
		{
			name:     "dot segments",
			path:     "src/./main.go",
			wantName: "main.go",
		},
		{
			name:     "leading dot-slash",
			path:     "./src/main.go",
			wantName: "main.go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := []FileEntry{
				{Path: tt.path, Size: 100, TokenCount: 20},
			}
			root := BuildTree(files)
			require.NotNil(t, root)

			// Walk to find the leaf file regardless of how many directories exist.
			allNodes := collectAllNodes(root)
			var found bool
			for _, n := range allNodes {
				if !n.IsDir && n.Name == tt.wantName {
					found = true
					break
				}
			}
			assert.True(t, found, "expected to find file named %q in tree", tt.wantName)
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildTree_UnicodeNames
// ---------------------------------------------------------------------------

func TestBuildTree_UnicodeNames(t *testing.T) {
	files := []FileEntry{
		{Path: "docs/README.md", Size: 100, TokenCount: 20},
		{Path: "docs/guide-\u65e5\u672c\u8a9e.md", Size: 200, TokenCount: 50},
		{Path: "src/caf\u00e9.go", Size: 150, TokenCount: 30},
		{Path: "\u4e2d\u6587/\u6587\u4ef6.txt", Size: 80, TokenCount: 15},
	}

	root := BuildTree(files)
	require.NotNil(t, root)

	allNodes := collectAllNodes(root)
	nodeNames := make([]string, len(allNodes))
	for i, n := range allNodes {
		nodeNames[i] = n.Name
	}

	// Verify Unicode file names are preserved.
	assert.Contains(t, nodeNames, "guide-\u65e5\u672c\u8a9e.md",
		"Japanese characters should be preserved in file names")
	assert.Contains(t, nodeNames, "caf\u00e9.go",
		"Accented characters should be preserved in file names")
	assert.Contains(t, nodeNames, "\u6587\u4ef6.txt",
		"Chinese characters should be preserved in file names")
	assert.Contains(t, nodeNames, "\u4e2d\u6587",
		"Chinese directory names should be preserved")
}

// ---------------------------------------------------------------------------
// TestSortTree_DirsBeforeFiles
// ---------------------------------------------------------------------------

func TestSortTree_DirsBeforeFiles(t *testing.T) {
	files := []FileEntry{
		{Path: "zebra.txt", Size: 10, TokenCount: 2},
		{Path: "alpha/file.go", Size: 20, TokenCount: 5},
		{Path: "beta.go", Size: 30, TokenCount: 8},
		{Path: "Alpha.md", Size: 15, TokenCount: 3},
		{Path: "zeta/deep.go", Size: 25, TokenCount: 6},
		{Path: "alpha/other.go", Size: 22, TokenCount: 5},
	}

	root := BuildTree(files)
	require.NotNil(t, root)
	require.True(t, len(root.Children) > 0, "root should have children")

	// At root level, directories should come before files.
	var seenFile bool
	for _, child := range root.Children {
		if child.IsDir {
			assert.False(t, seenFile,
				"directory %q appeared after a file; dirs must come first", child.Name)
		} else {
			seenFile = true
		}
	}

	// Verify case-insensitive alphabetical ordering within dirs and within files.
	var dirs, fileNodes []string
	for _, child := range root.Children {
		if child.IsDir {
			dirs = append(dirs, child.Name)
		} else {
			fileNodes = append(fileNodes, child.Name)
		}
	}

	// Directories should be sorted case-insensitively: alpha, zeta
	assert.Equal(t, []string{"alpha", "zeta"}, dirs,
		"directories should be sorted alphabetically case-insensitive")

	// Files should be sorted case-insensitively: Alpha.md, beta.go, zebra.txt
	assert.Equal(t, []string{"Alpha.md", "beta.go", "zebra.txt"}, fileNodes,
		"files should be sorted alphabetically case-insensitive")

	// Also verify children within subdirectories are sorted.
	alphaDir := findChild(root, "alpha")
	require.NotNil(t, alphaDir)
	alphaChildNames := childNames(alphaDir)
	assert.Equal(t, []string{"file.go", "other.go"}, alphaChildNames,
		"files within alpha/ should be sorted")
}

// ---------------------------------------------------------------------------
// TestCollapseTree_SingleChildChains
// ---------------------------------------------------------------------------

func TestCollapseTree_SingleChildChains(t *testing.T) {
	// Create a structure where a -> b -> c -> file.go
	// with a, b, c each having one child.
	files := []FileEntry{
		{Path: "a/b/c/file.go", Size: 100, TokenCount: 20},
	}

	root := BuildTree(files)
	require.NotNil(t, root)

	// After collapsing, the single-child chain a -> b -> c should be
	// collapsed into a single directory node named "a/b/c".
	require.Len(t, root.Children, 1, "root should have one collapsed directory")

	collapsed := root.Children[0]
	assert.True(t, collapsed.IsDir, "collapsed node should be a directory")
	assert.Contains(t, collapsed.Name, "a",
		"collapsed name should contain 'a'")
	assert.Contains(t, collapsed.Name, "b",
		"collapsed name should contain 'b'")
	assert.Contains(t, collapsed.Name, "c",
		"collapsed name should contain 'c'")

	// The collapsed node should have one file child.
	require.Len(t, collapsed.Children, 1)
	assert.Equal(t, "file.go", collapsed.Children[0].Name)
	assert.False(t, collapsed.Children[0].IsDir)
}

// ---------------------------------------------------------------------------
// TestCollapseTree_NoCollapse
// ---------------------------------------------------------------------------

func TestCollapseTree_NoCollapse(t *testing.T) {
	// Directories with multiple children should NOT be collapsed.
	files := []FileEntry{
		{Path: "src/main.go", Size: 100, TokenCount: 20},
		{Path: "src/util.go", Size: 200, TokenCount: 40},
		{Path: "lib/helper.go", Size: 150, TokenCount: 30},
	}

	root := BuildTree(files)
	require.NotNil(t, root)

	// Root has two children: lib, src -- neither should be collapsed.
	srcDir := findChild(root, "src")
	require.NotNil(t, srcDir, "src directory should exist")
	assert.True(t, srcDir.IsDir)
	assert.Len(t, srcDir.Children, 2,
		"src with 2 children should not be collapsed")

	libDir := findChild(root, "lib")
	require.NotNil(t, libDir, "lib directory should exist")
	assert.True(t, libDir.IsDir)
	// lib has one child, but that child is a file (not a directory),
	// so it should NOT be collapsed.
	assert.Len(t, libDir.Children, 1)
	assert.Equal(t, "helper.go", libDir.Children[0].Name)
	assert.Equal(t, "lib", libDir.Name,
		"lib should keep its original name (child is a file, not dir)")
}

// ---------------------------------------------------------------------------
// TestCollapseTree_PartialCollapse
// ---------------------------------------------------------------------------

func TestCollapseTree_PartialCollapse(t *testing.T) {
	// a -> b -> c where c has two files should collapse a/b but not c.
	files := []FileEntry{
		{Path: "a/b/c/file1.go", Size: 100, TokenCount: 20},
		{Path: "a/b/c/file2.go", Size: 100, TokenCount: 20},
	}

	root := BuildTree(files)
	require.NotNil(t, root)

	// a and b are single-child directories pointing to c, which has 2 children.
	// So a/b should collapse, but c should remain because it has multiple children.
	require.Len(t, root.Children, 1)
	collapsed := root.Children[0]
	assert.True(t, collapsed.IsDir)

	// The collapsed chain could be "a/b/c" (if c is merged because a->b->c is
	// still a single-child chain up to c) or "a/b" with c as a child.
	// Both are valid depending on implementation. Check both scenarios.
	if len(collapsed.Children) == 2 {
		// a/b/c was fully collapsed.
		assert.Contains(t, collapsed.Name, "c")
	} else {
		// a/b was collapsed, c is a separate directory child.
		cDir := collapsed.Children[0]
		assert.True(t, cDir.IsDir)
		assert.Len(t, cDir.Children, 2, "c should have 2 file children")
	}
}

// ---------------------------------------------------------------------------
// TestRenderTree_BasicStructure
// ---------------------------------------------------------------------------

func TestRenderTree_BasicStructure(t *testing.T) {
	files := []FileEntry{
		{Path: "src/main.go", Size: 100, TokenCount: 20},
		{Path: "src/util.go", Size: 200, TokenCount: 40},
		{Path: "README.md", Size: 50, TokenCount: 10},
	}

	root := BuildTree(files)
	result := RenderTree(root, TreeRenderOpts{})

	assert.NotEmpty(t, result, "rendered tree should not be empty")

	// Verify box-drawing characters are present.
	assert.True(t,
		strings.Contains(result, "\u251c\u2500\u2500") || strings.Contains(result, "\u2514\u2500\u2500"),
		"output should contain Unicode box-drawing connectors")

	// Verify file names appear.
	assert.Contains(t, result, "main.go")
	assert.Contains(t, result, "util.go")
	assert.Contains(t, result, "README.md")
	assert.Contains(t, result, "src")
}

// ---------------------------------------------------------------------------
// TestRenderTree_NoTrailingWhitespace
// ---------------------------------------------------------------------------

func TestRenderTree_NoTrailingWhitespace(t *testing.T) {
	files := sampleProjectFiles()
	root := BuildTree(files)

	result := RenderTree(root, TreeRenderOpts{})
	lines := strings.Split(result, "\n")

	for i, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		assert.Equal(t, trimmed, line,
			"line %d has trailing whitespace: %q", i+1, line)
	}
}

// ---------------------------------------------------------------------------
// TestRenderTree_DepthLimit
// ---------------------------------------------------------------------------

func TestRenderTree_DepthLimit(t *testing.T) {
	tests := []struct {
		name           string
		maxDepth       int
		shouldContain  []string
		shouldNotExist []string
	}{
		{
			name:     "depth 1 shows only top-level",
			maxDepth: 1,
			shouldContain: []string{
				"...",
			},
		},
		{
			name:     "depth 2 shows two levels",
			maxDepth: 2,
			shouldContain: []string{
				"...",
			},
		},
		{
			name:          "depth 0 unlimited",
			maxDepth:      0,
			shouldContain: []string{"main.go", "walker.go", "root.go"},
		},
	}

	files := sampleProjectFiles()
	root := BuildTree(files)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderTree(root, TreeRenderOpts{MaxDepth: tt.maxDepth})

			for _, s := range tt.shouldContain {
				assert.Contains(t, result, s,
					"depth %d output should contain %q", tt.maxDepth, s)
			}

			// When depth is limited, the deepest files should not appear.
			if tt.maxDepth > 0 && tt.maxDepth < 4 {
				// Files at depth > maxDepth should be truncated.
				// walker.go is at depth 3 (internal/discovery/walker.go).
				if tt.maxDepth < 3 {
					assert.NotContains(t, result, "walker.go",
						"walker.go at depth 3 should not appear with maxDepth=%d", tt.maxDepth)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRenderTree_DepthLimitTruncationIndicator
// ---------------------------------------------------------------------------

func TestRenderTree_DepthLimitTruncationIndicator(t *testing.T) {
	files := []FileEntry{
		{Path: "a/b/c/deep.go", Size: 100, TokenCount: 20},
		{Path: "a/b/other.go", Size: 100, TokenCount: 20},
		{Path: "top.go", Size: 100, TokenCount: 20},
	}

	root := BuildTree(files)
	result := RenderTree(root, TreeRenderOpts{MaxDepth: 1})

	assert.Contains(t, result, "...",
		"truncated branches should show '...' indicator")
	assert.Contains(t, result, "top.go",
		"top-level file should appear at depth 1")
}

// ---------------------------------------------------------------------------
// TestRenderTree_MetadataAnnotations
// ---------------------------------------------------------------------------

func TestRenderTree_MetadataAnnotations(t *testing.T) {
	files := []FileEntry{
		{Path: "main.go", Size: 1234, TokenCount: 340},
		{Path: "lib/util.go", Size: 5678, TokenCount: 1200},
	}

	root := BuildTree(files)

	t.Run("size and tokens enabled", func(t *testing.T) {
		result := RenderTree(root, TreeRenderOpts{
			ShowSize:   true,
			ShowTokens: true,
		})

		// Size should appear with human-readable formatting.
		assert.Contains(t, result, "340",
			"token count should appear when ShowTokens is true")
		assert.Contains(t, result, "1200",
			"token count should appear when ShowTokens is true")
		// Size annotation should be present in some form.
		assert.True(t,
			strings.Contains(result, "1234") || strings.Contains(result, "1.2") || strings.Contains(result, "KB"),
			"file size should appear when ShowSize is true")
	})

	t.Run("size only", func(t *testing.T) {
		result := RenderTree(root, TreeRenderOpts{
			ShowSize:   true,
			ShowTokens: false,
		})

		assert.True(t,
			strings.Contains(result, "1234") || strings.Contains(result, "1.2") || strings.Contains(result, "KB"),
			"file size should appear when ShowSize is true")
	})

	t.Run("tokens only", func(t *testing.T) {
		result := RenderTree(root, TreeRenderOpts{
			ShowSize:   false,
			ShowTokens: true,
		})

		assert.Contains(t, result, "340",
			"token count should appear when ShowTokens is true")
	})

	t.Run("no metadata", func(t *testing.T) {
		result := RenderTree(root, TreeRenderOpts{
			ShowSize:   false,
			ShowTokens: false,
		})

		// When metadata is disabled, the output should not contain size or
		// token annotations. We check that the specific numbers do not appear
		// as annotations (they might appear as part of file names, but in this
		// case our file names don't contain those numbers).
		assert.NotContains(t, result, "340",
			"token count should not appear when ShowTokens is false")
		assert.NotContains(t, result, "1200",
			"token count should not appear when ShowTokens is false")
	})
}

// ---------------------------------------------------------------------------
// TestRenderTree_EmptyTree
// ---------------------------------------------------------------------------

func TestRenderTree_EmptyTree(t *testing.T) {
	root := BuildTree(nil)
	result := RenderTree(root, TreeRenderOpts{})

	// An empty tree should render as just the root line or an empty string.
	// Either is acceptable. It should NOT contain box-drawing for children.
	assert.NotContains(t, result, "\u251c\u2500\u2500",
		"empty tree should not have child connectors")
	assert.NotContains(t, result, "\u2514\u2500\u2500",
		"empty tree should not have child connectors")
}

// ---------------------------------------------------------------------------
// TestRenderTree_Emojis
// ---------------------------------------------------------------------------

func TestRenderTree_Emojis(t *testing.T) {
	files := []FileEntry{
		{Path: "src/main.go", Size: 100, TokenCount: 20},
		{Path: "README.md", Size: 50, TokenCount: 10},
	}

	root := BuildTree(files)
	result := RenderTree(root, TreeRenderOpts{})

	// Verify directory emoji.
	assert.Contains(t, result, "\U0001f4c1",
		"directories should have folder emoji (U+1F4C1)")

	// Verify file emoji.
	assert.Contains(t, result, "\U0001f4c4",
		"files should have page emoji (U+1F4C4)")
}

// ---------------------------------------------------------------------------
// TestRenderTree_ConsistentConnectors
// ---------------------------------------------------------------------------

func TestRenderTree_ConsistentConnectors(t *testing.T) {
	files := []FileEntry{
		{Path: "a/file1.go", Size: 100, TokenCount: 20},
		{Path: "a/file2.go", Size: 100, TokenCount: 20},
		{Path: "a/file3.go", Size: 100, TokenCount: 20},
		{Path: "b/file4.go", Size: 100, TokenCount: 20},
	}

	root := BuildTree(files)
	result := RenderTree(root, TreeRenderOpts{})
	lines := strings.Split(result, "\n")

	// The last child at any level should use the end connector.
	var hasCorner bool
	var hasTee bool
	for _, line := range lines {
		if strings.Contains(line, "\u2514\u2500\u2500") {
			hasCorner = true
		}
		if strings.Contains(line, "\u251c\u2500\u2500") {
			hasTee = true
		}
	}

	assert.True(t, hasCorner,
		"tree should contain corner connector (U+2514) for last children")
	assert.True(t, hasTee,
		"tree should contain tee connector (U+251C) for non-last children")
}

// ---------------------------------------------------------------------------
// TestRenderTree_PipeConnector
// ---------------------------------------------------------------------------

func TestRenderTree_PipeConnector(t *testing.T) {
	// Ensure the pipe connector is used for continuation.
	files := []FileEntry{
		{Path: "a/sub/file1.go", Size: 100, TokenCount: 20},
		{Path: "b/file2.go", Size: 100, TokenCount: 20},
	}

	root := BuildTree(files)
	result := RenderTree(root, TreeRenderOpts{})

	// When "a" is not the last child, the pipe connector should continue.
	assert.True(t,
		strings.Contains(result, "\u2502") || strings.Contains(result, "|"),
		"tree should contain pipe connector for continuation lines")
}

// ---------------------------------------------------------------------------
// TestBuildTree_DuplicatePaths
// ---------------------------------------------------------------------------

func TestBuildTree_DuplicatePaths(t *testing.T) {
	// Duplicate paths should not cause panic or duplicate nodes.
	files := []FileEntry{
		{Path: "src/main.go", Size: 100, TokenCount: 20},
		{Path: "src/main.go", Size: 100, TokenCount: 20},
	}

	root := BuildTree(files)
	require.NotNil(t, root, "root should not be nil even with duplicates")

	// The tree should handle duplicates gracefully -- either deduplicate or
	// include both without crashing.
	assert.NotNil(t, root.Children, "root should have children")
}

// ---------------------------------------------------------------------------
// TestBuildTree_RootLevelFilesOnly
// ---------------------------------------------------------------------------

func TestBuildTree_RootLevelFilesOnly(t *testing.T) {
	files := []FileEntry{
		{Path: "go.mod", Size: 512, TokenCount: 100},
		{Path: "go.sum", Size: 8192, TokenCount: 0},
		{Path: "README.md", Size: 1536, TokenCount: 300},
		{Path: "main.go", Size: 245, TokenCount: 60},
	}

	root := BuildTree(files)
	require.NotNil(t, root)

	// All children should be files, none should be directories.
	for _, child := range root.Children {
		assert.False(t, child.IsDir,
			"child %q should be a file, not a directory", child.Name)
	}

	assert.Len(t, root.Children, 4, "should have 4 root-level files")
}

// ---------------------------------------------------------------------------
// TestBuildTree_PreservesMetadata
// ---------------------------------------------------------------------------

func TestBuildTree_PreservesMetadata(t *testing.T) {
	files := []FileEntry{
		{Path: "main.go", Size: 245, TokenCount: 60, Tier: 1},
		{Path: "lib/util.go", Size: 4096, TokenCount: 950, Tier: 3},
	}

	root := BuildTree(files)
	require.NotNil(t, root)

	allNodes := collectAllNodes(root)
	for _, node := range allNodes {
		if node.Name == "main.go" {
			assert.Equal(t, int64(245), node.Size)
			assert.Equal(t, 60, node.TokenCount)
			assert.Equal(t, 1, node.Tier)
		}
		if node.Name == "util.go" {
			assert.Equal(t, int64(4096), node.Size)
			assert.Equal(t, 950, node.TokenCount)
			assert.Equal(t, 3, node.Tier)
		}
	}
}

// ---------------------------------------------------------------------------
// TestRenderTree_MultipleDepthLevels
// ---------------------------------------------------------------------------

func TestRenderTree_MultipleDepthLevels(t *testing.T) {
	files := sampleProjectFiles()
	root := BuildTree(files)

	// Depth 0 (unlimited) should render all files.
	unlimited := RenderTree(root, TreeRenderOpts{MaxDepth: 0})
	for _, f := range files {
		parts := strings.Split(f.Path, "/")
		fileName := parts[len(parts)-1]
		assert.Contains(t, unlimited, fileName,
			"unlimited depth should contain all files including %s", fileName)
	}
}

// ---------------------------------------------------------------------------
// TestRenderTree_DeterministicOutput
// ---------------------------------------------------------------------------

func TestRenderTree_DeterministicOutput(t *testing.T) {
	// Rendering the same tree twice should produce identical output.
	files := sampleProjectFiles()

	root1 := BuildTree(files)
	result1 := RenderTree(root1, TreeRenderOpts{ShowSize: true, ShowTokens: true})

	root2 := BuildTree(files)
	result2 := RenderTree(root2, TreeRenderOpts{ShowSize: true, ShowTokens: true})

	assert.Equal(t, result1, result2,
		"rendering the same input twice should produce identical output")
}

// ---------------------------------------------------------------------------
// Golden tests
// ---------------------------------------------------------------------------

func TestRenderTree_Golden_Basic(t *testing.T) {
	files := sampleProjectFiles()
	root := BuildTree(files)
	result := RenderTree(root, TreeRenderOpts{})

	testutil.Golden(t, "tree-basic", []byte(result))
}

func TestRenderTree_Golden_WithMetadata(t *testing.T) {
	files := sampleProjectFiles()
	root := BuildTree(files)
	result := RenderTree(root, TreeRenderOpts{
		ShowSize:   true,
		ShowTokens: true,
	})

	testutil.Golden(t, "tree-with-metadata", []byte(result))
}

func TestRenderTree_Golden_Collapsed(t *testing.T) {
	files := collapsibleFiles()
	root := BuildTree(files)
	result := RenderTree(root, TreeRenderOpts{})

	testutil.Golden(t, "tree-collapsed", []byte(result))
}

func TestRenderTree_Golden_DepthLimited(t *testing.T) {
	files := sampleProjectFiles()
	root := BuildTree(files)
	result := RenderTree(root, TreeRenderOpts{MaxDepth: 2})

	testutil.Golden(t, "tree-depth-limited", []byte(result))
}

// ---------------------------------------------------------------------------
// Benchmark tests
// ---------------------------------------------------------------------------

func BenchmarkBuildTree(b *testing.B) {
	// Generate 1000 file entries across a realistic directory structure.
	files := make([]FileEntry, 0, 1000)
	packages := []string{
		"cmd/app", "cmd/tool",
		"internal/api", "internal/config", "internal/discovery",
		"internal/output", "internal/security", "internal/tokenizer",
		"pkg/utils", "pkg/models", "pkg/middleware",
		"test/integration", "test/e2e",
	}
	idx := 0
	for idx < 1000 {
		for _, pkg := range packages {
			if idx >= 1000 {
				break
			}
			files = append(files, FileEntry{
				Path:       fmt.Sprintf("%s/file_%04d.go", pkg, idx),
				Size:       int64(100 + idx%5000),
				TokenCount: 20 + idx%1000,
				Tier:       1 + idx%3,
			})
			idx++
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildTree(files)
	}
}

func BenchmarkRenderTree(b *testing.B) {
	// Pre-build a large tree, then benchmark rendering only.
	files := make([]FileEntry, 0, 1000)
	packages := []string{
		"cmd/app", "cmd/tool",
		"internal/api", "internal/config", "internal/discovery",
		"internal/output", "internal/security", "internal/tokenizer",
		"pkg/utils", "pkg/models", "pkg/middleware",
		"test/integration", "test/e2e",
	}
	idx := 0
	for idx < 1000 {
		for _, pkg := range packages {
			if idx >= 1000 {
				break
			}
			files = append(files, FileEntry{
				Path:       fmt.Sprintf("%s/file_%04d.go", pkg, idx),
				Size:       int64(100 + idx%5000),
				TokenCount: 20 + idx%1000,
				Tier:       1 + idx%3,
			})
			idx++
		}
	}

	root := BuildTree(files)
	opts := TreeRenderOpts{ShowSize: true, ShowTokens: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RenderTree(root, opts)
	}
}

func BenchmarkRenderTree_DepthLimited(b *testing.B) {
	files := make([]FileEntry, 0, 1000)
	for i := 0; i < 1000; i++ {
		files = append(files, FileEntry{
			Path:       fmt.Sprintf("level0/level1/level2/level3/level4/file_%04d.go", i),
			Size:       int64(100 + i%5000),
			TokenCount: 20 + i%1000,
		})
	}

	root := BuildTree(files)
	opts := TreeRenderOpts{MaxDepth: 2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RenderTree(root, opts)
	}
}
