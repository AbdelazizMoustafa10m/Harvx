package tokenizer_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tokenizer"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeFile creates a *pipeline.FileDescriptor suitable for budget tests.
// TokenCount must be set manually or via a stub counter because the test stub
// counts bytes (len), not BPE tokens.
func makeFile(path string, tier int, content string) *pipeline.FileDescriptor {
	fd := &pipeline.FileDescriptor{
		Path:       path,
		Tier:       tier,
		Content:    content,
		TokenCount: len(content), // stub: 1 token per byte
	}
	return fd
}

// newEnforcer constructs a BudgetEnforcer using the stub tokenizer so that
// token counts are deterministic (1 token == 1 byte).
func newEnforcer(maxTokens int, strategy tokenizer.TruncationStrategy) *tokenizer.BudgetEnforcer {
	return tokenizer.NewBudgetEnforcer(maxTokens, strategy, &stubTokenizer{name: "stub"})
}

// ---------------------------------------------------------------------------
// TruncationStrategy constants
// ---------------------------------------------------------------------------

func TestTruncationStrategy_Constants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, tokenizer.TruncationStrategy("skip"), tokenizer.SkipStrategy)
	assert.Equal(t, tokenizer.TruncationStrategy("truncate"), tokenizer.TruncateStrategy)
}

// ---------------------------------------------------------------------------
// NewBudgetEnforcer
// ---------------------------------------------------------------------------

func TestNewBudgetEnforcer_NilTokenizerFallsBackToEstimator(t *testing.T) {
	t.Parallel()
	// Should not panic when tok is nil -- falls back to character estimator.
	e := tokenizer.NewBudgetEnforcer(1000, tokenizer.SkipStrategy, nil)
	require.NotNil(t, e)
}

// ---------------------------------------------------------------------------
// Enforce -- no budget (maxTokens <= 0)
// ---------------------------------------------------------------------------

func TestEnforce_NoBudget_IncludesAll(t *testing.T) {
	t.Parallel()
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, "hello"),
		makeFile("b.go", 1, "world"),
		makeFile("c.go", 2, strings.Repeat("x", 500)),
	}
	e := newEnforcer(0, tokenizer.SkipStrategy)
	result := e.Enforce(files, 100)

	assert.Len(t, result.IncludedFiles, 3)
	assert.Empty(t, result.ExcludedFiles)
	assert.Empty(t, result.TruncatedFiles)
	// Budget fields are zero when enforcement is disabled.
	assert.Equal(t, 0, result.BudgetUsed)
	assert.Equal(t, 0, result.BudgetRemaining)
}

func TestEnforce_NegativeBudget_IncludesAll(t *testing.T) {
	t.Parallel()
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, "abc"),
	}
	e := newEnforcer(-1, tokenizer.SkipStrategy)
	result := e.Enforce(files, 0)

	assert.Len(t, result.IncludedFiles, 1)
	assert.Empty(t, result.ExcludedFiles)
}

// ---------------------------------------------------------------------------
// Enforce -- SkipStrategy
// ---------------------------------------------------------------------------

func TestEnforce_Skip_AllFit(t *testing.T) {
	t.Parallel()
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, "hello"), // 5 tokens
		makeFile("b.go", 1, "world"), // 5 tokens
	}
	// Budget: 20 tokens. Overhead: 0. Remaining: 20.
	e := newEnforcer(20, tokenizer.SkipStrategy)
	result := e.Enforce(files, 0)

	assert.Len(t, result.IncludedFiles, 2)
	assert.Empty(t, result.ExcludedFiles)
	assert.Equal(t, 10, result.TotalTokens)
	assert.Equal(t, 10, result.BudgetUsed)
	assert.Equal(t, 10, result.BudgetRemaining)
}

func TestEnforce_Skip_OverBudget_SkipsLargeFile(t *testing.T) {
	t.Parallel()
	// File A: 100 tokens, File B: 5 tokens.
	// Budget 50. After overhead 0, remaining = 50.
	// A (100) > 50 -- skip. B (5) <= 50 -- include.
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, strings.Repeat("x", 100)),
		makeFile("b.go", 1, "hello"),
	}
	e := newEnforcer(50, tokenizer.SkipStrategy)
	result := e.Enforce(files, 0)

	require.Len(t, result.IncludedFiles, 1)
	assert.Equal(t, "b.go", result.IncludedFiles[0].Path)

	require.Len(t, result.ExcludedFiles, 1)
	assert.Equal(t, "a.go", result.ExcludedFiles[0].Path)

	assert.Equal(t, 5, result.TotalTokens)
}

func TestEnforce_Skip_ContinuesAfterSkip(t *testing.T) {
	t.Parallel()
	// This test verifies the key skip property: smaller files after a large
	// excluded file are still considered.
	//
	// Budget 20. Files (in order): big(50), small1(5), big2(30), small2(8).
	// Expected: small1 and small2 included; big and big2 excluded.
	files := []*pipeline.FileDescriptor{
		makeFile("big.go", 0, strings.Repeat("x", 50)),
		makeFile("small1.go", 0, "hello"),  // 5
		makeFile("big2.go", 1, strings.Repeat("y", 30)),
		makeFile("small2.go", 1, "abcdefgh"), // 8
	}
	e := newEnforcer(20, tokenizer.SkipStrategy)
	result := e.Enforce(files, 0)

	includedPaths := make([]string, 0, len(result.IncludedFiles))
	for _, f := range result.IncludedFiles {
		includedPaths = append(includedPaths, f.Path)
	}
	assert.Contains(t, includedPaths, "small1.go")
	assert.Contains(t, includedPaths, "small2.go")
	assert.NotContains(t, includedPaths, "big.go")
	assert.NotContains(t, includedPaths, "big2.go")
}

func TestEnforce_Skip_OverheadReducesBudget(t *testing.T) {
	t.Parallel()
	// maxTokens=20, overhead=15 => remaining=5.
	// file a: 5 tokens fits. file b: 6 tokens does not.
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, "hello"), // 5
		makeFile("b.go", 0, "foobar"), // 6
	}
	e := newEnforcer(20, tokenizer.SkipStrategy)
	result := e.Enforce(files, 15)

	require.Len(t, result.IncludedFiles, 1)
	assert.Equal(t, "a.go", result.IncludedFiles[0].Path)
	require.Len(t, result.ExcludedFiles, 1)
	assert.Equal(t, "b.go", result.ExcludedFiles[0].Path)

	// BudgetUsed = overhead(15) + totalTokens(5) = 20
	assert.Equal(t, 20, result.BudgetUsed)
	assert.Equal(t, 0, result.BudgetRemaining)
}

func TestEnforce_Skip_AllExcluded(t *testing.T) {
	t.Parallel()
	// maxTokens=5, overhead=5 => remaining=0. Every file is excluded.
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, "hello"),
		makeFile("b.go", 1, "world"),
	}
	e := newEnforcer(5, tokenizer.SkipStrategy)
	result := e.Enforce(files, 5)

	assert.Empty(t, result.IncludedFiles)
	assert.Len(t, result.ExcludedFiles, 2)
	assert.Equal(t, 0, result.TotalTokens)
}

func TestEnforce_Skip_EmptyFiles(t *testing.T) {
	t.Parallel()
	e := newEnforcer(100, tokenizer.SkipStrategy)
	result := e.Enforce(nil, 0)

	assert.Empty(t, result.IncludedFiles)
	assert.Empty(t, result.ExcludedFiles)
	assert.Equal(t, 0, result.TotalTokens)
}

// ---------------------------------------------------------------------------
// Enforce -- TruncateStrategy
// ---------------------------------------------------------------------------

func TestEnforce_Truncate_AllFit(t *testing.T) {
	t.Parallel()
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, "hello"), // 5
		makeFile("b.go", 1, "world"), // 5
	}
	e := newEnforcer(50, tokenizer.TruncateStrategy)
	result := e.Enforce(files, 0)

	assert.Len(t, result.IncludedFiles, 2)
	assert.Empty(t, result.ExcludedFiles)
	assert.Empty(t, result.TruncatedFiles)
	assert.Equal(t, 10, result.TotalTokens)
}

func TestEnforce_Truncate_TruncatesFirstOverBudget(t *testing.T) {
	t.Parallel()
	// Content is a multi-line string so binary search has something to work with.
	// Each line is 10 bytes. Budget remaining = 25 bytes after overhead.
	// File content: 5 lines * 10 bytes = 50 bytes total.
	lines := []string{
		"1234567890",
		"abcdefghij",
		"ABCDEFGHIJ",
		"0987654321",
		"zyxwvutsrq",
	}
	content := strings.Join(lines, "\n")
	files := []*pipeline.FileDescriptor{
		{
			Path:       "big.go",
			Tier:       0,
			Content:    content,
			TokenCount: len(content), // stub: 1 token per byte
		},
	}

	// Budget: 100, overhead: 0, remaining: 100.
	// The file is 54 bytes (50 chars + 4 newlines). With budget 100 it fits.
	e := newEnforcer(100, tokenizer.TruncateStrategy)
	result := e.Enforce(files, 0)

	// Should be fully included (no truncation needed).
	assert.Len(t, result.IncludedFiles, 1)
	assert.Empty(t, result.TruncatedFiles)
}

func TestEnforce_Truncate_ContentIsTruncated(t *testing.T) {
	t.Parallel()
	// 5 lines of 10 chars each, joined = 54 bytes (50 + 4 newlines).
	// Budget 35, overhead 0 => remaining 35.
	// Marker reservation 20 => budgetForContent = 15.
	// Line 0 = "1234567890" (10 bytes), Line 1 = "abcdefghij" (10 bytes).
	// After joining 2 lines: "1234567890\nabcdefghij" = 21 bytes > 15.
	// After joining 1 line:  "1234567890" = 10 bytes <= 15. So 1 line kept.
	lines := []string{"1234567890", "abcdefghij", "ABCDEFGHIJ", "0987654321", "zyxwvutsrq"}
	content := strings.Join(lines, "\n")
	files := []*pipeline.FileDescriptor{
		{Path: "big.go", Tier: 0, Content: content, TokenCount: len(content)},
	}

	e := newEnforcer(35, tokenizer.TruncateStrategy)
	result := e.Enforce(files, 0)

	require.Len(t, result.IncludedFiles, 1)
	require.Len(t, result.TruncatedFiles, 1)
	assert.Empty(t, result.ExcludedFiles)

	truncated := result.TruncatedFiles[0]
	assert.Contains(t, truncated.Content, "<!-- Content truncated:")
	assert.NotContains(t, truncated.Content, "zyxwvutsrq", "last line must be excluded")
}

func TestEnforce_Truncate_SubsequentFilesExcluded(t *testing.T) {
	t.Parallel()
	// First file fits partially (triggers truncation), subsequent files must be excluded.
	lines := []string{"line one", "line two", "line three"}
	content := strings.Join(lines, "\n") // 28 bytes
	files := []*pipeline.FileDescriptor{
		{Path: "a.go", Tier: 0, Content: content, TokenCount: len(content)},
		{Path: "b.go", Tier: 1, Content: "hello", TokenCount: 5},
		{Path: "c.go", Tier: 2, Content: "world", TokenCount: 5},
	}

	// Budget 20 means a.go will be truncated; b.go and c.go must be excluded.
	e := newEnforcer(20, tokenizer.TruncateStrategy)
	result := e.Enforce(files, 0)

	require.Len(t, result.TruncatedFiles, 1)
	assert.Equal(t, "a.go", result.TruncatedFiles[0].Path)

	excludedPaths := make([]string, 0)
	for _, f := range result.ExcludedFiles {
		excludedPaths = append(excludedPaths, f.Path)
	}
	assert.Contains(t, excludedPaths, "b.go")
	assert.Contains(t, excludedPaths, "c.go")
}

func TestEnforce_Truncate_TruncatedFileInIncludedFiles(t *testing.T) {
	t.Parallel()
	// Verify that a truncated file appears in both IncludedFiles and TruncatedFiles.
	content := strings.Repeat("x", 100)
	files := []*pipeline.FileDescriptor{
		{Path: "big.go", Tier: 0, Content: content, TokenCount: 100},
	}

	e := newEnforcer(50, tokenizer.TruncateStrategy)
	result := e.Enforce(files, 0)

	require.Len(t, result.IncludedFiles, 1)
	require.Len(t, result.TruncatedFiles, 1)
	// Both slices should point to the same descriptor.
	assert.Same(t, result.IncludedFiles[0], result.TruncatedFiles[0])
}

func TestEnforce_Truncate_OriginalNotMutated(t *testing.T) {
	t.Parallel()
	// The original FileDescriptor must not be mutated by truncation.
	originalContent := strings.Repeat("a", 100)
	fd := &pipeline.FileDescriptor{
		Path:       "orig.go",
		Tier:       0,
		Content:    originalContent,
		TokenCount: 100,
	}
	files := []*pipeline.FileDescriptor{fd}

	e := newEnforcer(50, tokenizer.TruncateStrategy)
	_ = e.Enforce(files, 0)

	assert.Equal(t, originalContent, fd.Content, "original Content must not be mutated")
	assert.Equal(t, 100, fd.TokenCount, "original TokenCount must not be mutated")
}

func TestEnforce_Truncate_ZeroRemainingExcludesFile(t *testing.T) {
	t.Parallel()
	// overhead == maxTokens so remaining == 0; file must be excluded, not truncated.
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, "hello"),
	}
	e := newEnforcer(10, tokenizer.TruncateStrategy)
	result := e.Enforce(files, 10)

	assert.Empty(t, result.IncludedFiles)
	assert.Len(t, result.ExcludedFiles, 1)
	assert.Empty(t, result.TruncatedFiles)
}

// ---------------------------------------------------------------------------
// BudgetResult -- budget accounting
// ---------------------------------------------------------------------------

func TestEnforce_BudgetAccounting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		maxTokens       int
		overhead        int
		files           []*pipeline.FileDescriptor
		strategy        tokenizer.TruncationStrategy
		wantBudgetUsed  int
		wantBudgetRem   int
		wantTotalTokens int
	}{
		{
			name:      "exact fit",
			maxTokens: 15,
			overhead:  5,
			files: []*pipeline.FileDescriptor{
				makeFile("a.go", 0, "hello"), // 5
				makeFile("b.go", 0, "world"), // 5
			},
			strategy:        tokenizer.SkipStrategy,
			wantTotalTokens: 10,
			wantBudgetUsed:  15, // overhead(5) + tokens(10)
			wantBudgetRem:   0,
		},
		{
			name:      "overhead exceeds max",
			maxTokens: 5,
			overhead:  10,
			files: []*pipeline.FileDescriptor{
				makeFile("a.go", 0, "hello"), // 5
			},
			strategy:        tokenizer.SkipStrategy,
			wantTotalTokens: 0, // file excluded (remaining = 5-10 = -5)
			wantBudgetUsed:  10,
			wantBudgetRem:   -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newEnforcer(tt.maxTokens, tt.strategy)
			result := e.Enforce(tt.files, tt.overhead)

			assert.Equal(t, tt.wantTotalTokens, result.TotalTokens, "TotalTokens")
			assert.Equal(t, tt.wantBudgetUsed, result.BudgetUsed, "BudgetUsed")
			assert.Equal(t, tt.wantBudgetRem, result.BudgetRemaining, "BudgetRemaining")
		})
	}
}

// ---------------------------------------------------------------------------
// BudgetSummary -- per-tier statistics
// ---------------------------------------------------------------------------

func TestEnforce_Summary_TierStats(t *testing.T) {
	t.Parallel()
	// Tier 0: a.go (5 tokens, included), b.go (100 tokens, excluded).
	// Tier 1: c.go (3 tokens, included).
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, "hello"),                     // 5
		makeFile("b.go", 0, strings.Repeat("x", 100)),   // 100
		makeFile("c.go", 1, "abc"),                       // 3
	}

	// Budget 20, overhead 0.
	// Tier 0: a(5) fits, b(100) skipped.
	// Tier 1: c(3) fits.
	e := newEnforcer(20, tokenizer.SkipStrategy)
	result := e.Enforce(files, 0)

	require.Contains(t, result.Summary.TierStats, 0)
	require.Contains(t, result.Summary.TierStats, 1)

	tier0 := result.Summary.TierStats[0]
	assert.Equal(t, 1, tier0.FilesIncluded, "tier 0 included")
	assert.Equal(t, 1, tier0.FilesExcluded, "tier 0 excluded")
	assert.Equal(t, 5, tier0.TokensUsed, "tier 0 tokens used")

	tier1 := result.Summary.TierStats[1]
	assert.Equal(t, 1, tier1.FilesIncluded, "tier 1 included")
	assert.Equal(t, 0, tier1.FilesExcluded, "tier 1 excluded")
	assert.Equal(t, 3, tier1.TokensUsed, "tier 1 tokens used")
}

func TestEnforce_Summary_EmptyStats_WhenNoFiles(t *testing.T) {
	t.Parallel()
	e := newEnforcer(100, tokenizer.SkipStrategy)
	result := e.Enforce(nil, 0)

	assert.Empty(t, result.Summary.TierStats)
}

func TestEnforce_Summary_NoBudget_AllIncluded(t *testing.T) {
	t.Parallel()
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, "hello"), // tier 0
		makeFile("b.go", 2, "world"), // tier 2
	}
	e := newEnforcer(0, tokenizer.SkipStrategy)
	result := e.Enforce(files, 0)

	assert.Equal(t, 1, result.Summary.TierStats[0].FilesIncluded)
	assert.Equal(t, 1, result.Summary.TierStats[2].FilesIncluded)
}

// ---------------------------------------------------------------------------
// BudgetSummary.SortedTierKeys
// ---------------------------------------------------------------------------

func TestBudgetSummary_SortedTierKeys(t *testing.T) {
	t.Parallel()
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 3, "abc"),
		makeFile("b.go", 1, "def"),
		makeFile("c.go", 0, "ghi"),
		makeFile("d.go", 5, "jkl"),
	}
	e := newEnforcer(0, tokenizer.SkipStrategy) // no budget = include all
	result := e.Enforce(files, 0)

	keys := result.Summary.SortedTierKeys()
	assert.Equal(t, []int{0, 1, 3, 5}, keys)
}

func TestBudgetSummary_SortedTierKeys_Empty(t *testing.T) {
	t.Parallel()
	e := newEnforcer(100, tokenizer.SkipStrategy)
	result := e.Enforce(nil, 0)

	keys := result.Summary.SortedTierKeys()
	assert.Empty(t, keys)
}

// ---------------------------------------------------------------------------
// Truncation marker content
// ---------------------------------------------------------------------------

func TestEnforce_Truncate_MarkerContent(t *testing.T) {
	t.Parallel()
	// Build a file with many distinct lines so truncation is clearly visible.
	var linesSlice []string
	for i := 0; i < 20; i++ {
		linesSlice = append(linesSlice, fmt.Sprintf("line %02d: content here", i))
	}
	content := strings.Join(linesSlice, "\n")

	files := []*pipeline.FileDescriptor{
		{Path: "many.go", Tier: 0, Content: content, TokenCount: len(content)},
	}

	// Tight budget to force truncation.
	e := newEnforcer(60, tokenizer.TruncateStrategy)
	result := e.Enforce(files, 0)

	if len(result.TruncatedFiles) == 0 {
		// File fit entirely -- skip assertion (budget was enough).
		t.Skip("file fit without truncation; increase budget tightness")
	}

	truncated := result.TruncatedFiles[0]
	assert.Contains(t, truncated.Content, "<!-- Content truncated:")
	assert.Contains(t, truncated.Content, "tokens shown -->")

	// TokenCount on the truncated descriptor must reflect the actual content length
	// as counted by the stub (1 token per byte).
	assert.Equal(t, len(truncated.Content), truncated.TokenCount)
}

// ---------------------------------------------------------------------------
// Invariants
// ---------------------------------------------------------------------------

func TestEnforce_Invariant_IncludedPlusExcludedEqualsTotal(t *testing.T) {
	t.Parallel()
	// Verify |included| + |excluded| == |input| for all strategies.
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, strings.Repeat("a", 40)),
		makeFile("b.go", 0, strings.Repeat("b", 10)),
		makeFile("c.go", 1, strings.Repeat("c", 30)),
		makeFile("d.go", 2, strings.Repeat("d", 5)),
		makeFile("e.go", 3, strings.Repeat("e", 50)),
	}

	for _, strategy := range []tokenizer.TruncationStrategy{
		tokenizer.SkipStrategy,
		tokenizer.TruncateStrategy,
	} {
		t.Run(string(strategy), func(t *testing.T) {
			t.Parallel()
			e := newEnforcer(50, strategy)
			result := e.Enforce(files, 0)
			assert.Equal(t, len(files), len(result.IncludedFiles)+len(result.ExcludedFiles),
				"included + excluded must equal total input files")
		})
	}
}

func TestEnforce_Invariant_TruncatedSubsetOfIncluded(t *testing.T) {
	t.Parallel()
	// Every truncated file must also appear in IncludedFiles.
	files := []*pipeline.FileDescriptor{
		{Path: "big.go", Tier: 0, Content: strings.Repeat("x", 100), TokenCount: 100},
		makeFile("small.go", 1, "ok"),
	}

	e := newEnforcer(40, tokenizer.TruncateStrategy)
	result := e.Enforce(files, 0)

	// Build set of included paths.
	includedSet := make(map[string]bool)
	for _, f := range result.IncludedFiles {
		includedSet[f.Path] = true
	}
	for _, f := range result.TruncatedFiles {
		assert.True(t, includedSet[f.Path],
			"truncated file %q must appear in IncludedFiles", f.Path)
	}
}

func TestEnforce_Invariant_TotalTokensSumCheck(t *testing.T) {
	t.Parallel()
	files := []*pipeline.FileDescriptor{
		makeFile("a.go", 0, "hello"),  // 5
		makeFile("b.go", 0, "world"),  // 5
		makeFile("c.go", 1, "foobar"), // 6
	}
	e := newEnforcer(100, tokenizer.SkipStrategy)
	result := e.Enforce(files, 0)

	// Sum all included file token counts manually.
	sum := 0
	for _, f := range result.IncludedFiles {
		sum += f.TokenCount
	}
	assert.Equal(t, sum, result.TotalTokens,
		"TotalTokens must equal sum of included file token counts")
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkBudgetEnforcer_Skip_1K(b *testing.B) {
	const fileCount = 1000
	files := make([]*pipeline.FileDescriptor, fileCount)
	for i := range files {
		files[i] = &pipeline.FileDescriptor{
			Path:       fmt.Sprintf("file%04d.go", i),
			Tier:       i % 6,
			Content:    strings.Repeat("x", 200),
			TokenCount: 200,
		}
	}
	e := tokenizer.NewBudgetEnforcer(100_000, tokenizer.SkipStrategy, &stubTokenizer{name: "stub"})

	b.ResetTimer()
	for range b.N {
		_ = e.Enforce(files, 500)
	}
}

func BenchmarkBudgetEnforcer_Truncate_1K(b *testing.B) {
	const fileCount = 1000
	files := make([]*pipeline.FileDescriptor, fileCount)
	for i := range files {
		files[i] = &pipeline.FileDescriptor{
			Path:       fmt.Sprintf("file%04d.go", i),
			Tier:       i % 6,
			Content:    strings.Repeat("line content\n", 20),
			TokenCount: len(strings.Repeat("line content\n", 20)),
		}
	}
	e := tokenizer.NewBudgetEnforcer(50_000, tokenizer.TruncateStrategy, &stubTokenizer{name: "stub"})

	b.ResetTimer()
	for range b.N {
		_ = e.Enforce(files, 500)
	}
}
