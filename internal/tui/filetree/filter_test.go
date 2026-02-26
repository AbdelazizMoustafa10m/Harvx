package filetree

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildFilterTestTree creates a tree for filter testing:
//
//	root (expanded)
//	  src/ (expanded)
//	    main.go       (tier 0)
//	    middleware.ts  (tier 1)
//	    util.go       (tier 2)
//	  docs/ (expanded)
//	    README.md     (tier 4)
//	    CHANGELOG.md  (tier 4)
//	  tests/ (expanded)
//	    main_test.go  (tier 3)
//	  config.toml     (tier 5)
func buildFilterTestTree(t *testing.T) *Node {
	t.Helper()

	root := NewNode("", ".", true)
	root.Expanded = true
	root.SetLoaded(true)

	src := NewNode("src", "src", true)
	src.Expanded = true
	src.SetLoaded(true)

	mainGo := NewNode("src/main.go", "main.go", false)
	mainGo.Tier = 0

	middleware := NewNode("src/middleware.ts", "middleware.ts", false)
	middleware.Tier = 1

	utilGo := NewNode("src/util.go", "util.go", false)
	utilGo.Tier = 2

	src.AddChild(mainGo)
	src.AddChild(middleware)
	src.AddChild(utilGo)
	src.SortChildren()

	docs := NewNode("docs", "docs", true)
	docs.Expanded = true
	docs.SetLoaded(true)

	readme := NewNode("docs/README.md", "README.md", false)
	readme.Tier = 4

	changelog := NewNode("docs/CHANGELOG.md", "CHANGELOG.md", false)
	changelog.Tier = 4

	docs.AddChild(readme)
	docs.AddChild(changelog)
	docs.SortChildren()

	tests := NewNode("tests", "tests", true)
	tests.Expanded = true
	tests.SetLoaded(true)

	mainTest := NewNode("tests/main_test.go", "main_test.go", false)
	mainTest.Tier = 3

	tests.AddChild(mainTest)

	configToml := NewNode("config.toml", "config.toml", false)
	configToml.Tier = 5

	root.AddChild(src)
	root.AddChild(docs)
	root.AddChild(tests)
	root.AddChild(configToml)
	root.SortChildren()

	return root
}

// --- FilterState tests ---

func TestNewFilterState(t *testing.T) {
	t.Parallel()

	f := NewFilterState()
	assert.Equal(t, -1, f.TierFilter)
	assert.Empty(t, f.SearchQuery)
	assert.False(t, f.HasAnyFilter())
}

func TestFilterState_HasSearchFilter(t *testing.T) {
	t.Parallel()

	f := FilterState{SearchQuery: "main"}
	assert.True(t, f.HasSearchFilter())
	assert.True(t, f.HasAnyFilter())
}

func TestFilterState_HasTierFilter(t *testing.T) {
	t.Parallel()

	f := FilterState{TierFilter: 0}
	assert.True(t, f.HasTierFilter())
	assert.True(t, f.HasAnyFilter())
}

func TestFilterState_CycleTier(t *testing.T) {
	t.Parallel()

	f := NewFilterState()
	assert.Equal(t, -1, f.TierFilter)
	assert.Equal(t, "All", f.TierLabel())

	// Cycle through: All -> 0 -> 1 -> 2 -> 3 -> 4 -> 5 -> All.
	expected := []struct {
		tier  int
		label string
	}{
		{0, "0 (critical)"},
		{1, "1 (primary)"},
		{2, "2 (secondary)"},
		{3, "3 (tests)"},
		{4, "4 (docs)"},
		{5, "5 (low)"},
		{-1, "All"},
	}

	for _, exp := range expected {
		f = f.CycleTier()
		assert.Equal(t, exp.tier, f.TierFilter, "expected tier %d", exp.tier)
		assert.Equal(t, exp.label, f.TierLabel())
	}
}

// --- FilterNodes tests ---

func TestFilterNodes_NoFilter(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()
	filter := NewFilterState()

	result := FilterNodes(nodes, filter)
	assert.Equal(t, len(nodes), len(result), "no filter should return all nodes")
}

func TestFilterNodes_SearchFilter(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()
	filter := FilterState{SearchQuery: "main", TierFilter: -1}

	result := FilterNodes(nodes, filter)

	// Should match: src/main.go, tests/main_test.go
	// Directories src/ and tests/ should be included as ancestors.
	var fileNames []string
	for _, n := range result {
		if !n.IsDir {
			fileNames = append(fileNames, n.Name)
		}
	}
	assert.Contains(t, fileNames, "main.go")
	assert.Contains(t, fileNames, "main_test.go")
	assert.NotContains(t, fileNames, "util.go")
	assert.NotContains(t, fileNames, "README.md")
}

func TestFilterNodes_TierFilter(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()
	filter := FilterState{TierFilter: 4}

	result := FilterNodes(nodes, filter)

	// Should match: docs/README.md, docs/CHANGELOG.md
	// Directory docs/ should be included.
	var fileNames []string
	for _, n := range result {
		if !n.IsDir {
			fileNames = append(fileNames, n.Name)
		}
	}
	assert.Len(t, fileNames, 2)
	assert.Contains(t, fileNames, "README.md")
	assert.Contains(t, fileNames, "CHANGELOG.md")
}

func TestFilterNodes_CombinedSearchAndTier(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()
	// Search for "main" AND tier 0.
	filter := FilterState{SearchQuery: "main", TierFilter: 0}

	result := FilterNodes(nodes, filter)

	// Only src/main.go is tier 0 and matches "main".
	// tests/main_test.go matches "main" but is tier 3.
	var fileNames []string
	for _, n := range result {
		if !n.IsDir {
			fileNames = append(fileNames, n.Name)
		}
	}
	assert.Len(t, fileNames, 1)
	assert.Contains(t, fileNames, "main.go")
}

func TestFilterNodes_SearchCaseInsensitive(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()
	filter := FilterState{SearchQuery: "MAIN", TierFilter: -1}

	result := FilterNodes(nodes, filter)

	var fileNames []string
	for _, n := range result {
		if !n.IsDir {
			fileNames = append(fileNames, n.Name)
		}
	}
	assert.Contains(t, fileNames, "main.go")
	assert.Contains(t, fileNames, "main_test.go")
}

func TestFilterNodes_NoMatches(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()
	filter := FilterState{SearchQuery: "zzznomatch", TierFilter: -1}

	result := FilterNodes(nodes, filter)
	assert.Empty(t, result)
}

func TestFilterNodes_TierNoMatches(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()
	// Tier 0 + search for something that does not match any tier-0 file.
	filter := FilterState{SearchQuery: "changelog", TierFilter: 0}

	result := FilterNodes(nodes, filter)
	assert.Empty(t, result)
}

func TestFilterNodes_DirectoryIncludedOnlyWithMatchingDescendant(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()
	// Filter for tier 3 only. Only tests/main_test.go matches.
	filter := FilterState{TierFilter: 3}

	result := FilterNodes(nodes, filter)

	var dirNames []string
	for _, n := range result {
		if n.IsDir {
			dirNames = append(dirNames, n.Name)
		}
	}
	// Only tests/ directory should appear, not src/ or docs/.
	assert.Contains(t, dirNames, "tests")
	assert.NotContains(t, dirNames, "src")
	assert.NotContains(t, dirNames, "docs")
}

// --- SelectAll / DeselectAll tests ---

func TestSelectAll(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()

	// All start as Excluded.
	for _, n := range nodes {
		if !n.IsDir {
			assert.Equal(t, Excluded, n.Included)
		}
	}

	SelectAll(nodes)

	// All files should now be Included.
	for _, n := range nodes {
		if !n.IsDir {
			assert.Equal(t, Included, n.Included, "file %s should be included", n.Name)
		}
	}
}

func TestDeselectAll(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()

	// First include everything.
	SelectAll(nodes)
	for _, n := range nodes {
		if !n.IsDir {
			require.Equal(t, Included, n.Included)
		}
	}

	// Then deselect all.
	DeselectAll(nodes)
	for _, n := range nodes {
		if !n.IsDir {
			assert.Equal(t, Excluded, n.Included, "file %s should be excluded", n.Name)
		}
	}
}

func TestSelectAll_FilteredSubset(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()

	// Filter to tier 4 only.
	filter := FilterState{TierFilter: 4}
	filtered := FilterNodes(nodes, filter)

	// Select all filtered nodes.
	SelectAll(filtered)

	// Only tier 4 files should be included.
	readme := root.FindByPath("docs/README.md")
	changelog := root.FindByPath("docs/CHANGELOG.md")
	mainGo := root.FindByPath("src/main.go")
	require.NotNil(t, readme)
	require.NotNil(t, changelog)
	require.NotNil(t, mainGo)

	assert.Equal(t, Included, readme.Included)
	assert.Equal(t, Included, changelog.Included)
	assert.Equal(t, Excluded, mainGo.Included, "non-filtered file should remain excluded")
}

// --- VisibleFiles tests ---

func TestVisibleFiles(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()

	files := VisibleFiles(nodes)
	for _, f := range files {
		assert.False(t, f.IsDir, "VisibleFiles should only return files, got dir %s", f.Name)
	}
	assert.NotEmpty(t, files)
}

func TestSelectAll_PropagatesParentState(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()

	SelectAll(nodes)

	// Directories should be Included (all children included).
	src := root.FindByPath("src")
	require.NotNil(t, src)
	assert.Equal(t, Included, src.Included, "src directory should be Included when all children are")
}

func TestDeselectAll_PropagatesParentState(t *testing.T) {
	t.Parallel()

	root := buildFilterTestTree(t)
	nodes := root.VisibleNodes()

	SelectAll(nodes)
	DeselectAll(nodes)

	// Directories should be Excluded (all children excluded).
	src := root.FindByPath("src")
	require.NotNil(t, src)
	assert.Equal(t, Excluded, src.Included)
}
