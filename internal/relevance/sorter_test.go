package relevance

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/pipeline"
)

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// makeFile is a convenience constructor for test FileDescriptors.
func makeFile(path string, tier, tokens int) *pipeline.FileDescriptor {
	return &pipeline.FileDescriptor{
		Path:       path,
		Tier:       tier,
		TokenCount: tokens,
	}
}

// paths extracts the Path field from each descriptor in order.
func paths(files []*pipeline.FileDescriptor) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Path
	}
	return out
}

// ----------------------------------------------------------------------------
// SortByRelevance
// ----------------------------------------------------------------------------

// TestSortByRelevance_BasicOrdering verifies that files across different tiers
// are returned in ascending tier order.
func TestSortByRelevance_BasicOrdering(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		makeFile("README.md", 4, 0),
		makeFile("go.mod", 0, 0),
		makeFile("src/main.go", 1, 0),
	}

	got := SortByRelevance(input)

	require.Len(t, got, 3)
	assert.Equal(t, []string{"go.mod", "src/main.go", "README.md"}, paths(got))
}

// TestSortByRelevance_AlphabeticalWithinTier verifies secondary alphabetical
// sort when multiple files share the same tier.
func TestSortByRelevance_AlphabeticalWithinTier(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		makeFile("b/bar.go", 1, 0),
		makeFile("a/foo.go", 1, 0),
		makeFile("c/baz.go", 1, 0),
	}

	got := SortByRelevance(input)

	assert.Equal(t, []string{"a/foo.go", "b/bar.go", "c/baz.go"}, paths(got))
}

// TestSortByRelevance_AllSameTier verifies purely alphabetical output when
// every file belongs to the same tier.
func TestSortByRelevance_AllSameTier(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		makeFile("zebra.go", 2, 0),
		makeFile("apple.go", 2, 0),
		makeFile("mango.go", 2, 0),
	}

	got := SortByRelevance(input)

	assert.Equal(t, []string{"apple.go", "mango.go", "zebra.go"}, paths(got))
}

// TestSortByRelevance_Empty verifies that an empty input produces an empty
// (non-nil) output slice.
func TestSortByRelevance_Empty(t *testing.T) {
	t.Parallel()

	got := SortByRelevance([]*pipeline.FileDescriptor{})
	require.NotNil(t, got)
	assert.Empty(t, got)
}

// TestSortByRelevance_SingleFile verifies that a single-element input is
// returned unchanged.
func TestSortByRelevance_SingleFile(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{makeFile("go.mod", 0, 100)}
	got := SortByRelevance(input)

	require.Len(t, got, 1)
	assert.Equal(t, "go.mod", got[0].Path)
}

// TestSortByRelevance_DoesNotMutateInput verifies that the original slice is
// left in its original order after sorting.
func TestSortByRelevance_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		makeFile("z.go", 1, 0),
		makeFile("a.go", 0, 0),
		makeFile("m.go", 3, 0),
	}
	originalPaths := paths(input)

	_ = SortByRelevance(input)

	assert.Equal(t, originalPaths, paths(input),
		"SortByRelevance must not mutate the input slice")
}

// TestSortByRelevance_Deterministic verifies that sorting the same input twice
// produces the same result.
func TestSortByRelevance_Deterministic(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		makeFile("b.go", 2, 0),
		makeFile("a.go", 2, 0),
		makeFile("go.mod", 0, 0),
		makeFile("README.md", 4, 0),
		makeFile("c.go", 2, 0),
	}

	first := paths(SortByRelevance(input))
	second := paths(SortByRelevance(input))
	assert.Equal(t, first, second)
}

// TestSortByRelevance_Stable verifies that descriptors with identical tier and
// path retain their original relative order (stable sort).
func TestSortByRelevance_Stable(t *testing.T) {
	t.Parallel()

	// Two descriptors with identical Tier and Path but different token counts
	// to distinguish them. Stable sort must preserve insertion order.
	a := &pipeline.FileDescriptor{Path: "same.go", Tier: 1, TokenCount: 10}
	b := &pipeline.FileDescriptor{Path: "same.go", Tier: 1, TokenCount: 20}

	input := []*pipeline.FileDescriptor{a, b}
	got := SortByRelevance(input)

	require.Len(t, got, 2)
	assert.Same(t, a, got[0], "first element should be the one inserted first")
	assert.Same(t, b, got[1])
}

// TestSortByRelevance_GoldenOrder is a golden test for 20 files with known
// tiers, verifying the exact deterministic output order.
func TestSortByRelevance_GoldenOrder(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		makeFile("go.sum", 5, 0),
		makeFile("internal/server/handler.go", 1, 0),
		makeFile("cmd/harvx/main.go", 1, 0),
		makeFile("README.md", 4, 0),
		makeFile("go.mod", 0, 0),
		makeFile("Dockerfile", 0, 0),
		makeFile(".github/workflows/ci.yml", 5, 0),
		makeFile("docs/guide.md", 4, 0),
		makeFile("main_test.go", 3, 0),
		makeFile("components/Button.tsx", 2, 0),
		makeFile("utils/helpers.go", 2, 0),
		makeFile("src/index.ts", 1, 0),
		makeFile("CHANGELOG.md", 4, 0),
		makeFile("package-lock.json", 5, 0),
		makeFile("internal/config/loader.go", 1, 0),
		makeFile("__tests__/unit.ts", 3, 0),
		makeFile("api/v1/handler.go", 2, 0),
		makeFile("Makefile", 0, 0),
		makeFile("spec/user_spec.rb", 3, 0),
		makeFile("poetry.lock", 5, 0),
	}

	got := SortByRelevance(input)

	want := []string{
		// Tier 0 (alphabetical)
		"Dockerfile",
		"Makefile",
		"go.mod",
		// Tier 1 (alphabetical)
		"cmd/harvx/main.go",
		"internal/config/loader.go",
		"internal/server/handler.go",
		"src/index.ts",
		// Tier 2 (alphabetical)
		"api/v1/handler.go",
		"components/Button.tsx",
		"utils/helpers.go",
		// Tier 3 (alphabetical)
		"__tests__/unit.ts",
		"main_test.go",
		"spec/user_spec.rb",
		// Tier 4 (alphabetical)
		"CHANGELOG.md",
		"README.md",
		"docs/guide.md",
		// Tier 5 (alphabetical)
		".github/workflows/ci.yml",
		"go.sum",
		"package-lock.json",
		"poetry.lock",
	}

	assert.Equal(t, want, paths(got))
}

// ----------------------------------------------------------------------------
// GroupByTier
// ----------------------------------------------------------------------------

// TestGroupByTier_BasicGrouping verifies correct partitioning across tiers.
func TestGroupByTier_BasicGrouping(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		makeFile("go.mod", 0, 0),
		makeFile("src/main.go", 1, 0),
		makeFile("src/util.go", 1, 0),
		makeFile("README.md", 4, 0),
	}

	got := GroupByTier(input)

	require.Len(t, got, 3)
	assert.Len(t, got[0], 1)
	assert.Len(t, got[1], 2)
	assert.Len(t, got[4], 1)
	assert.Equal(t, "go.mod", got[0][0].Path)
	assert.Equal(t, "README.md", got[4][0].Path)
}

// TestGroupByTier_Empty verifies that an empty input returns an empty map.
func TestGroupByTier_Empty(t *testing.T) {
	t.Parallel()

	got := GroupByTier([]*pipeline.FileDescriptor{})
	assert.Empty(t, got)
}

// TestGroupByTier_AllSameTier verifies a single-bucket grouping.
func TestGroupByTier_AllSameTier(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		makeFile("a.go", 2, 0),
		makeFile("b.go", 2, 0),
	}

	got := GroupByTier(input)

	require.Len(t, got, 1)
	assert.Len(t, got[2], 2)
}

// TestGroupByTier_PreservesOrder verifies that insertion order within each
// group bucket is preserved.
func TestGroupByTier_PreservesOrder(t *testing.T) {
	t.Parallel()

	a := makeFile("z.go", 1, 0)
	b := makeFile("a.go", 1, 0)
	input := []*pipeline.FileDescriptor{a, b}

	got := GroupByTier(input)

	require.Len(t, got[1], 2)
	assert.Same(t, a, got[1][0])
	assert.Same(t, b, got[1][1])
}

// ----------------------------------------------------------------------------
// TierSummary
// ----------------------------------------------------------------------------

// TestTierSummary_Basic verifies counts, token totals, and sorted file paths.
func TestTierSummary_Basic(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		makeFile("go.mod", 0, 50),
		makeFile("src/b.go", 1, 200),
		makeFile("src/a.go", 1, 100),
		makeFile("README.md", 4, 30),
	}

	got := TierSummary(input)

	require.Len(t, got, 3)

	// Tier 0
	assert.Equal(t, 0, got[0].Tier)
	assert.Equal(t, 1, got[0].FileCount)
	assert.Equal(t, 50, got[0].TotalTokens)
	assert.Equal(t, []string{"go.mod"}, got[0].FilePaths)

	// Tier 1
	assert.Equal(t, 1, got[1].Tier)
	assert.Equal(t, 2, got[1].FileCount)
	assert.Equal(t, 300, got[1].TotalTokens)
	assert.Equal(t, []string{"src/a.go", "src/b.go"}, got[1].FilePaths)

	// Tier 4
	assert.Equal(t, 4, got[2].Tier)
	assert.Equal(t, 1, got[2].FileCount)
	assert.Equal(t, 30, got[2].TotalTokens)
	assert.Equal(t, []string{"README.md"}, got[2].FilePaths)
}

// TestTierSummary_Empty verifies that an empty input returns an empty slice.
func TestTierSummary_Empty(t *testing.T) {
	t.Parallel()

	got := TierSummary([]*pipeline.FileDescriptor{})
	assert.Empty(t, got)
}

// TestTierSummary_SingleFile verifies a single-file summary.
func TestTierSummary_SingleFile(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{makeFile("go.mod", 0, 100)}
	got := TierSummary(input)

	require.Len(t, got, 1)
	assert.Equal(t, 0, got[0].Tier)
	assert.Equal(t, 1, got[0].FileCount)
	assert.Equal(t, 100, got[0].TotalTokens)
	assert.Equal(t, []string{"go.mod"}, got[0].FilePaths)
}

// TestTierSummary_OnlyPopulatedTiers verifies that empty tiers are omitted.
func TestTierSummary_OnlyPopulatedTiers(t *testing.T) {
	t.Parallel()

	// Tiers 1, 3, 5 are absent from input.
	input := []*pipeline.FileDescriptor{
		makeFile("go.mod", 0, 10),
		makeFile("README.md", 4, 20),
	}

	got := TierSummary(input)

	require.Len(t, got, 2)
	assert.Equal(t, 0, got[0].Tier)
	assert.Equal(t, 4, got[1].Tier)
}

// TestTierSummary_SortedByTier verifies that the result slice is always in
// ascending tier order regardless of input order.
func TestTierSummary_SortedByTier(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		makeFile("poetry.lock", 5, 0),
		makeFile("go.mod", 0, 0),
		makeFile("README.md", 4, 0),
		makeFile("src/main.go", 1, 0),
	}

	got := TierSummary(input)

	for i := 1; i < len(got); i++ {
		assert.Less(t, got[i-1].Tier, got[i].Tier,
			"tier stats must be in ascending order")
	}
}

// TestTierSummary_FilePathsSorted verifies alphabetical ordering of FilePaths
// within each TierStat.
func TestTierSummary_FilePathsSorted(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		makeFile("z/last.go", 1, 0),
		makeFile("a/first.go", 1, 0),
		makeFile("m/middle.go", 1, 0),
	}

	got := TierSummary(input)

	require.Len(t, got, 1)
	assert.Equal(t, []string{"a/first.go", "m/middle.go", "z/last.go"}, got[0].FilePaths)
}

// ----------------------------------------------------------------------------
// ClassifyAndSort
// ----------------------------------------------------------------------------

// TestClassifyAndSort_Basic verifies end-to-end classification and sorting
// using the default tier definitions.
func TestClassifyAndSort_Basic(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		{Path: "README.md"},
		{Path: "go.mod"},
		{Path: "src/main.go"},
		{Path: "main_test.go"},
	}

	got := ClassifyAndSort(input, DefaultTierDefinitions())

	// After classification: go.mod=0, src/main.go=1, main_test.go=3, README.md=4
	require.Len(t, got, 4)
	assert.Equal(t, "go.mod", got[0].Path)
	assert.Equal(t, "src/main.go", got[1].Path)
	assert.Equal(t, "main_test.go", got[2].Path)
	assert.Equal(t, "README.md", got[3].Path)
}

// TestClassifyAndSort_SetsCorrectTiers verifies that Tier fields are mutated
// on each descriptor after classification.
func TestClassifyAndSort_SetsCorrectTiers(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		{Path: "go.mod"},
		{Path: "src/main.go"},
		{Path: "README.md"},
	}

	got := ClassifyAndSort(input, DefaultTierDefinitions())

	require.Len(t, got, 3)
	assert.Equal(t, int(Tier0Critical), got[0].Tier)
	assert.Equal(t, int(Tier1Primary), got[1].Tier)
	assert.Equal(t, int(Tier4Docs), got[2].Tier)
}

// TestClassifyAndSort_CustomTiers verifies that custom tier definitions are
// applied instead of defaults.
func TestClassifyAndSort_CustomTiers(t *testing.T) {
	t.Parallel()

	customTiers := []TierDefinition{
		{Tier: Tier0Critical, Patterns: []string{"important/**"}},
		{Tier: Tier5Low, Patterns: []string{"low/**"}},
	}

	input := []*pipeline.FileDescriptor{
		{Path: "low/junk.txt"},
		{Path: "important/key.go"},
		{Path: "other/file.go"}, // unmatched -> DefaultUnmatchedTier (2)
	}

	got := ClassifyAndSort(input, customTiers)

	require.Len(t, got, 3)
	assert.Equal(t, "important/key.go", got[0].Path)
	assert.Equal(t, int(Tier0Critical), got[0].Tier)

	assert.Equal(t, "other/file.go", got[1].Path)
	assert.Equal(t, int(DefaultUnmatchedTier), got[1].Tier)

	assert.Equal(t, "low/junk.txt", got[2].Path)
	assert.Equal(t, int(Tier5Low), got[2].Tier)
}

// TestClassifyAndSort_Empty verifies an empty input returns an empty slice.
func TestClassifyAndSort_Empty(t *testing.T) {
	t.Parallel()

	got := ClassifyAndSort([]*pipeline.FileDescriptor{}, DefaultTierDefinitions())
	require.NotNil(t, got)
	assert.Empty(t, got)
}

// TestClassifyAndSort_NilTiers verifies that nil tier definitions cause all
// files to be assigned DefaultUnmatchedTier and sorted alphabetically.
func TestClassifyAndSort_NilTiers(t *testing.T) {
	t.Parallel()

	input := []*pipeline.FileDescriptor{
		{Path: "z.go"},
		{Path: "a.go"},
	}

	got := ClassifyAndSort(input, nil)

	require.Len(t, got, 2)
	// Both land in DefaultUnmatchedTier; secondary sort is alphabetical.
	assert.Equal(t, "a.go", got[0].Path)
	assert.Equal(t, "z.go", got[1].Path)
	assert.Equal(t, int(DefaultUnmatchedTier), got[0].Tier)
	assert.Equal(t, int(DefaultUnmatchedTier), got[1].Tier)
}

// ----------------------------------------------------------------------------
// Benchmark
// ----------------------------------------------------------------------------

// BenchmarkSortByRelevance10K measures sort throughput on 10 000 pre-classified
// files with a realistic tier distribution.
func BenchmarkSortByRelevance10K(b *testing.B) {
	tiers := []int{0, 1, 1, 1, 2, 2, 3, 4, 5, 5}
	files := make([]*pipeline.FileDescriptor, 10000)
	for i := range files {
		files[i] = &pipeline.FileDescriptor{
			Path: fmt.Sprintf("dir%d/file%d.go", i%100, i),
			Tier: tiers[i%len(tiers)],
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		_ = SortByRelevance(files)
	}
}
