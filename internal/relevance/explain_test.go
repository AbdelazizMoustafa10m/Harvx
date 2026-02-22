// Package relevance — unit tests for explain.go (T-032).
package relevance

import (
	"strings"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tokenizer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// newFD builds a minimal FileDescriptor for use in budget summary tests.
func newFD(path string, tier int, tokens int) *pipeline.FileDescriptor {
	return &pipeline.FileDescriptor{
		Path:       path,
		Tier:       tier,
		TokenCount: tokens,
	}
}

// ----------------------------------------------------------------------------
// TestExplain
// ----------------------------------------------------------------------------

// TestExplainMatchesTier0 verifies a file that matches a Tier 0 pattern.
func TestExplainMatchesTier0(t *testing.T) {
	t.Parallel()

	result := Explain("go.mod", DefaultTierDefinitions())

	require.NotNil(t, result)
	assert.Equal(t, "go.mod", result.FilePath)
	assert.Equal(t, 0, result.AssignedTier)
	assert.Equal(t, "go.mod", result.MatchedPattern)
	assert.False(t, result.IsDefault)
	assert.GreaterOrEqual(t, result.MatchedTierDef, 0)
}

// TestExplainMatchesTier1 verifies a file matching a Tier 1 (src/**) pattern.
func TestExplainMatchesTier1(t *testing.T) {
	t.Parallel()

	result := Explain("src/api/handler.go", DefaultTierDefinitions())

	require.NotNil(t, result)
	assert.Equal(t, 1, result.AssignedTier)
	assert.Equal(t, "src/**", result.MatchedPattern)
	assert.False(t, result.IsDefault)
}

// TestExplainNoMatchIsDefault verifies that an unmatched file gets
// IsDefault=true and tier 2.
func TestExplainNoMatchIsDefault(t *testing.T) {
	t.Parallel()

	result := Explain("mystery/unknown.xyz", DefaultTierDefinitions())

	require.NotNil(t, result)
	assert.Equal(t, int(DefaultUnmatchedTier), result.AssignedTier)
	assert.Equal(t, "", result.MatchedPattern)
	assert.True(t, result.IsDefault)
	assert.Equal(t, -1, result.MatchedTierDef)
	assert.Empty(t, result.AllMatches)
}

// TestExplainAllMatchesContainsEveryOverlap verifies that AllMatches includes
// all patterns that would match, not just the winning one. Uses a custom tier
// definition so that the overlap is deterministic and independent of doublestar
// single-star-vs-separator semantics.
func TestExplainAllMatchesContainsEveryOverlap(t *testing.T) {
	t.Parallel()

	// Define two tiers where "src/app.ts" matches both:
	//   - Tier 1: src/**
	//   - Tier 2: **/*.ts
	defs := []TierDefinition{
		{Tier: Tier1Primary, Patterns: []string{"src/**"}},
		{Tier: Tier2Secondary, Patterns: []string{"**/*.ts"}},
	}

	result := Explain("src/app.ts", defs)

	require.NotNil(t, result)
	// Assigned to tier 1 (first match wins).
	assert.Equal(t, 1, result.AssignedTier)
	assert.False(t, result.IsDefault)

	// AllMatches must contain both the tier-1 and tier-2 matches.
	require.GreaterOrEqual(t, len(result.AllMatches), 2,
		"AllMatches should include both tier-1 and tier-2 entries")

	tiers := make(map[int]bool)
	for _, pm := range result.AllMatches {
		tiers[pm.Tier] = true
	}
	assert.True(t, tiers[1], "AllMatches should include a tier-1 match")
	assert.True(t, tiers[2], "AllMatches should include a tier-2 match")
}

// TestExplainAllMatchesSortedByTier verifies AllMatches is sorted ascending by
// tier then lexicographically by pattern within a tier.
func TestExplainAllMatchesSortedByTier(t *testing.T) {
	t.Parallel()

	// Custom tiers with two patterns in the same tier that both match.
	defs := []TierDefinition{
		{Tier: Tier0Critical, Patterns: []string{"*.go", "main.go"}},
		{Tier: Tier1Primary, Patterns: []string{"*.go"}},
	}

	result := Explain("main.go", defs)
	require.NotNil(t, result)

	for i := 1; i < len(result.AllMatches); i++ {
		prev := result.AllMatches[i-1]
		curr := result.AllMatches[i]
		if prev.Tier == curr.Tier {
			assert.LessOrEqual(t, prev.Pattern, curr.Pattern,
				"patterns within same tier must be sorted lexicographically")
		} else {
			assert.Less(t, prev.Tier, curr.Tier,
				"AllMatches must be sorted by ascending tier")
		}
	}
}

// TestExplainAssignedTierIsLowestMatch confirms that when multiple tiers match,
// the lowest tier number is assigned (first-match-wins).
func TestExplainAssignedTierIsLowestMatch(t *testing.T) {
	t.Parallel()

	// Both patterns match "internal/handler.go":
	//   Tier 1: internal/**
	//   Tier 3: **/*.go
	defs := []TierDefinition{
		{Tier: Tier3Tests, Patterns: []string{"**/*.go"}},
		{Tier: Tier1Primary, Patterns: []string{"internal/**"}},
	}

	result := Explain("internal/handler.go", defs)
	require.NotNil(t, result)
	// Tier 1 must win over Tier 3.
	assert.Equal(t, 1, result.AssignedTier)
	// Both should appear in AllMatches.
	require.GreaterOrEqual(t, len(result.AllMatches), 2)
}

// TestExplainNilTiers treats nil tiers as empty; every file is default.
func TestExplainNilTiers(t *testing.T) {
	t.Parallel()

	result := Explain("src/main.go", nil)
	require.NotNil(t, result)
	assert.True(t, result.IsDefault)
	assert.Equal(t, int(DefaultUnmatchedTier), result.AssignedTier)
}

// TestExplainCallerCanEnrichBudget verifies that the caller can set
// WouldBeIncluded and ExclusionReason after receiving the result.
func TestExplainCallerCanEnrichBudget(t *testing.T) {
	t.Parallel()

	result := Explain("src/api/handler.go", DefaultTierDefinitions())
	require.NotNil(t, result)

	// Simulate budget enrichment by the caller.
	result.WouldBeIncluded = true
	result.TokenCount = 450

	assert.True(t, result.WouldBeIncluded)
	assert.Equal(t, 450, result.TokenCount)
}

// TestExplainPathNormalisation verifies that paths with "./" prefix are
// normalised before matching.
func TestExplainPathNormalisation(t *testing.T) {
	t.Parallel()

	withPrefix := Explain("./src/main.go", DefaultTierDefinitions())
	withoutPrefix := Explain("src/main.go", DefaultTierDefinitions())

	require.NotNil(t, withPrefix)
	require.NotNil(t, withoutPrefix)

	assert.Equal(t, withoutPrefix.AssignedTier, withPrefix.AssignedTier)
	assert.Equal(t, withoutPrefix.MatchedPattern, withPrefix.MatchedPattern)
	assert.Equal(t, withoutPrefix.IsDefault, withPrefix.IsDefault)
}

// TestExplainWindowsPaths verifies that backslash separators are normalised.
func TestExplainWindowsPaths(t *testing.T) {
	t.Parallel()

	result := Explain(`src\api\handler.go`, DefaultTierDefinitions())
	require.NotNil(t, result)
	assert.Equal(t, 1, result.AssignedTier, "backslash path should resolve to tier 1")
}

// TestExplainDoesNotMutateTiers verifies that Explain does not modify the
// caller's tier definitions slice.
func TestExplainDoesNotMutateTiers(t *testing.T) {
	t.Parallel()

	defs := DefaultTierDefinitions()
	original := make([]TierDefinition, len(defs))
	copy(original, defs)

	_ = Explain("src/main.go", defs)

	for i, d := range defs {
		assert.Equal(t, original[i].Tier, d.Tier,
			"tier at index %d should not have been mutated", i)
	}
}

// ----------------------------------------------------------------------------
// TestTierLabel
// ----------------------------------------------------------------------------

// TestTierLabel verifies all six default labels and the unknown-tier fallback.
func TestTierLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tier int
		want string
	}{
		{0, "Config"},
		{1, "Source"},
		{2, "Secondary"},
		{3, "Tests"},
		{4, "Docs"},
		{5, "CI/Lock"},
		{6, "Tier6"},
		{-1, "Tier-1"},
		{100, "Tier100"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			got := TierLabel(tt.tier)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ----------------------------------------------------------------------------
// TestFormatExplain
// ----------------------------------------------------------------------------

// TestFormatExplainMatchedFile verifies output for a file matched by a pattern.
func TestFormatExplainMatchedFile(t *testing.T) {
	t.Parallel()

	result := &ExplainResult{
		FilePath:       "src/api/handler.go",
		AssignedTier:   1,
		MatchedPattern: "src/**",
		MatchedTierDef: 0,
		IsDefault:      false,
		AllMatches: []PatternMatch{
			{Tier: 1, Pattern: "src/**"},
		},
	}

	output := FormatExplain(result)

	assert.Contains(t, output, "File: src/api/handler.go")
	assert.Contains(t, output, "Tier: 1 (Source Code)")
	assert.Contains(t, output, "Matched Pattern: src/** (from tier 1)")
	assert.Contains(t, output, "All matching patterns:")
	assert.Contains(t, output, "- Tier 1: src/**")
	// Budget Status must not appear when WouldBeIncluded is false and
	// ExclusionReason is empty.
	assert.NotContains(t, output, "Budget Status:")
}

// TestFormatExplainDefaultFile verifies output for an unmatched (default) file.
func TestFormatExplainDefaultFile(t *testing.T) {
	t.Parallel()

	result := &ExplainResult{
		FilePath:       "mystery/unknown.xyz",
		AssignedTier:   2,
		MatchedPattern: "",
		MatchedTierDef: -1,
		IsDefault:      true,
		AllMatches:     nil,
	}

	output := FormatExplain(result)

	assert.Contains(t, output, "File: mystery/unknown.xyz")
	assert.Contains(t, output, "Tier: 2 (Secondary)")
	assert.Contains(t, output, "Matched Pattern: (default, unmatched)")
	assert.Contains(t, output, "All matching patterns:")
	// Default entry for unmatched files.
	assert.Contains(t, output, "(default, unmatched)")
	assert.NotContains(t, output, "Budget Status:")
}

// TestFormatExplainWithBudgetIncluded verifies that Budget Status appears when
// WouldBeIncluded is set.
func TestFormatExplainWithBudgetIncluded(t *testing.T) {
	t.Parallel()

	result := &ExplainResult{
		FilePath:        "src/api/handler.go",
		AssignedTier:    1,
		MatchedPattern:  "src/**",
		MatchedTierDef:  0,
		IsDefault:       false,
		WouldBeIncluded: true,
		TokenCount:      450,
		AllMatches: []PatternMatch{
			{Tier: 1, Pattern: "src/**"},
		},
	}

	output := FormatExplain(result)

	assert.Contains(t, output, "Budget Status: Included (tokens: 450)")
}

// TestFormatExplainWithBudgetExcluded verifies Budget Status for excluded files.
func TestFormatExplainWithBudgetExcluded(t *testing.T) {
	t.Parallel()

	result := &ExplainResult{
		FilePath:        "tests/e2e/flow.ts",
		AssignedTier:    3,
		MatchedPattern:  "tests/**",
		MatchedTierDef:  0,
		IsDefault:       false,
		WouldBeIncluded: false,
		ExclusionReason: "budget_exceeded",
		AllMatches: []PatternMatch{
			{Tier: 3, Pattern: "tests/**"},
		},
	}

	output := FormatExplain(result)

	assert.Contains(t, output, "Budget Status: Excluded (budget_exceeded)")
}

// TestFormatExplainMultipleAllMatches verifies that overlapping matches are all
// listed in the All matching patterns section.
func TestFormatExplainMultipleAllMatches(t *testing.T) {
	t.Parallel()

	result := &ExplainResult{
		FilePath:       "internal/server/server_test.go",
		AssignedTier:   1,
		MatchedPattern: "internal/**",
		MatchedTierDef: 0,
		IsDefault:      false,
		AllMatches: []PatternMatch{
			{Tier: 1, Pattern: "internal/**"},
			{Tier: 3, Pattern: "*_test.go"},
		},
	}

	output := FormatExplain(result)

	assert.Contains(t, output, "- Tier 1: internal/**")
	assert.Contains(t, output, "- Tier 3: *_test.go")
}

// ----------------------------------------------------------------------------
// TestGenerateInclusionSummary
// ----------------------------------------------------------------------------

// TestGenerateInclusionSummaryAllIncluded verifies output when every file fits
// within budget (no exclusions) and no budget was set.
func TestGenerateInclusionSummaryAllIncluded(t *testing.T) {
	t.Parallel()

	included := []*pipeline.FileDescriptor{
		newFD("go.mod", 0, 50),
		newFD("src/main.go", 1, 200),
		newFD("README.md", 4, 100),
	}

	br := &tokenizer.BudgetResult{
		IncludedFiles:   included,
		ExcludedFiles:   []*pipeline.FileDescriptor{},
		TruncatedFiles:  []*pipeline.FileDescriptor{},
		TotalTokens:     350,
		BudgetUsed:      0,
		BudgetRemaining: 0,
		Summary: tokenizer.BudgetSummary{
			TierStats: map[int]tokenizer.TierStat{
				0: {FilesIncluded: 1, TokensUsed: 50},
				1: {FilesIncluded: 1, TokensUsed: 200},
				4: {FilesIncluded: 1, TokensUsed: 100},
			},
		},
	}

	output := GenerateInclusionSummary(br)

	assert.Contains(t, output, "Files: 3 included, 0 excluded")
	assert.Contains(t, output, "Tier 0 (Config)")
	assert.Contains(t, output, "Tier 1 (Source)")
	assert.Contains(t, output, "Tier 4 (Docs)")
	// No budget was set; should show "no budget".
	assert.Contains(t, output, "no budget")
	assert.NotContains(t, output, "budget (")
}

// TestGenerateInclusionSummaryWithBudget verifies output when budget is active
// and some files were excluded.
func TestGenerateInclusionSummaryWithBudget(t *testing.T) {
	t.Parallel()

	// Build a scenario with files across all 6 tiers, some excluded.
	// Tier 3: 62 included, 42 excluded.
	// Tier 5: 17 included, 6 excluded.
	br := &tokenizer.BudgetResult{
		IncludedFiles:   make([]*pipeline.FileDescriptor, 0),
		ExcludedFiles:   make([]*pipeline.FileDescriptor, 48),
		TruncatedFiles:  make([]*pipeline.FileDescriptor, 0),
		TotalTokens:     89420,
		BudgetUsed:      89420,
		BudgetRemaining: 110580,
		Summary: tokenizer.BudgetSummary{
			TierStats: map[int]tokenizer.TierStat{
				0: {FilesIncluded: 5, FilesExcluded: 0, TokensUsed: 2100},
				1: {FilesIncluded: 48, FilesExcluded: 0, TokensUsed: 45000},
				2: {FilesIncluded: 180, FilesExcluded: 0, TokensUsed: 35000},
				3: {FilesIncluded: 62, FilesExcluded: 42, TokensUsed: 5000},
				4: {FilesIncluded: 30, FilesExcluded: 0, TokensUsed: 1500},
				5: {FilesIncluded: 17, FilesExcluded: 6, TokensUsed: 820},
			},
		},
	}

	output := GenerateInclusionSummary(br)

	// Header counts.
	assert.Contains(t, output, "Files:")
	assert.Contains(t, output, "included")
	assert.Contains(t, output, "excluded")

	// Tier labels.
	assert.Contains(t, output, "Tier 0 (Config)")
	assert.Contains(t, output, "Tier 1 (Source)")
	assert.Contains(t, output, "Tier 2 (Secondary)")
	assert.Contains(t, output, "Tier 3 (Tests)")
	assert.Contains(t, output, "Tier 4 (Docs)")
	assert.Contains(t, output, "Tier 5 (CI/Lock)")

	// Exclusion annotations appear for tiers with excluded files.
	assert.Contains(t, output, "excluded by budget", "tier 3 should show exclusion count")

	// Budget total line.
	assert.Contains(t, output, "budget (")
	assert.Contains(t, output, "89,420 tokens")
	assert.Contains(t, output, "200,000 budget")
	assert.Contains(t, output, "44%", "~89420/200000 = 44%%")
}

// TestGenerateInclusionSummaryNoExclusions verifies the summary when budget
// enforcement was active but no files were excluded (all fit within budget).
func TestGenerateInclusionSummaryNoExclusions(t *testing.T) {
	t.Parallel()

	br := &tokenizer.BudgetResult{
		IncludedFiles:   make([]*pipeline.FileDescriptor, 5),
		ExcludedFiles:   []*pipeline.FileDescriptor{},
		TruncatedFiles:  []*pipeline.FileDescriptor{},
		TotalTokens:     1000,
		BudgetUsed:      1000,
		BudgetRemaining: 99000,
		Summary: tokenizer.BudgetSummary{
			TierStats: map[int]tokenizer.TierStat{
				1: {FilesIncluded: 5, FilesExcluded: 0, TokensUsed: 1000},
			},
		},
	}

	output := GenerateInclusionSummary(br)

	// BudgetUsed > 0 so budget line should appear.
	assert.Contains(t, output, "budget (")
	assert.NotContains(t, output, "no budget")
	// No exclusion annotations.
	assert.NotContains(t, output, "excluded by budget")
}

// TestGenerateInclusionSummaryAllExcluded verifies the edge case where every
// file was excluded (budget was effectively zero after overhead).
func TestGenerateInclusionSummaryAllExcluded(t *testing.T) {
	t.Parallel()

	excluded := []*pipeline.FileDescriptor{
		newFD("src/main.go", 1, 5000),
		newFD("internal/handler.go", 1, 3000),
	}

	br := &tokenizer.BudgetResult{
		IncludedFiles:   []*pipeline.FileDescriptor{},
		ExcludedFiles:   excluded,
		TruncatedFiles:  []*pipeline.FileDescriptor{},
		TotalTokens:     0,
		BudgetUsed:      0,
		BudgetRemaining: 0,
		Summary: tokenizer.BudgetSummary{
			TierStats: map[int]tokenizer.TierStat{},
		},
	}

	output := GenerateInclusionSummary(br)

	assert.Contains(t, output, "Files: 0 included, 2 excluded")
	// len(ExcludedFiles) > 0 so budget line should appear.
	assert.Contains(t, output, "budget (")
}

// TestGenerateInclusionSummaryTierLabels verifies that each tier row uses the
// TierLabel helper for its human-readable name.
func TestGenerateInclusionSummaryTierLabels(t *testing.T) {
	t.Parallel()

	br := &tokenizer.BudgetResult{
		IncludedFiles:   []*pipeline.FileDescriptor{},
		ExcludedFiles:   []*pipeline.FileDescriptor{},
		TruncatedFiles:  []*pipeline.FileDescriptor{},
		TotalTokens:     0,
		BudgetUsed:      0,
		BudgetRemaining: 0,
		Summary: tokenizer.BudgetSummary{
			TierStats: map[int]tokenizer.TierStat{
				0: {FilesIncluded: 1, TokensUsed: 10},
				1: {FilesIncluded: 1, TokensUsed: 10},
				2: {FilesIncluded: 1, TokensUsed: 10},
				3: {FilesIncluded: 1, TokensUsed: 10},
				4: {FilesIncluded: 1, TokensUsed: 10},
				5: {FilesIncluded: 1, TokensUsed: 10},
			},
		},
	}

	output := GenerateInclusionSummary(br)

	assert.Contains(t, output, "Tier 0 (Config)")
	assert.Contains(t, output, "Tier 1 (Source)")
	assert.Contains(t, output, "Tier 2 (Secondary)")
	assert.Contains(t, output, "Tier 3 (Tests)")
	assert.Contains(t, output, "Tier 4 (Docs)")
	assert.Contains(t, output, "Tier 5 (CI/Lock)")
}

// TestGenerateInclusionSummaryByTierSection verifies the "By Tier:" header
// always appears in the output.
func TestGenerateInclusionSummaryByTierSection(t *testing.T) {
	t.Parallel()

	br := &tokenizer.BudgetResult{
		IncludedFiles:   []*pipeline.FileDescriptor{},
		ExcludedFiles:   []*pipeline.FileDescriptor{},
		TruncatedFiles:  []*pipeline.FileDescriptor{},
		Summary:         tokenizer.BudgetSummary{TierStats: map[int]tokenizer.TierStat{}},
	}

	output := GenerateInclusionSummary(br)

	assert.Contains(t, output, "By Tier:")
}

// ----------------------------------------------------------------------------
// TestFormatInt (internal helper via exported behaviour)
// ----------------------------------------------------------------------------

// TestFormatIntThousandSeparators verifies that formatInt uses comma separators
// and handles edge cases like 0, negative, and sub-1000 values.
func TestFormatIntThousandSeparators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1,000"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{89420, "89,420"},
		{200000, "200,000"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			got := formatInt(tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestFormatExplainTierDisplayLabel verifies that FormatExplain uses "Source Code"
// for tier 1 but the standard label for other tiers.
func TestFormatExplainTierDisplayLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tier      int
		wantLabel string
	}{
		{0, "Config"},
		{1, "Source Code"},
		{2, "Secondary"},
		{3, "Tests"},
		{4, "Docs"},
		{5, "CI/Lock"},
	}

	for _, tt := range tests {
		t.Run(tt.wantLabel, func(t *testing.T) {
			t.Parallel()

			result := &ExplainResult{
				FilePath:       "some/file",
				AssignedTier:   tt.tier,
				MatchedPattern: "some/**",
				MatchedTierDef: 0,
			}

			output := FormatExplain(result)
			wantLine := strings.Contains(output, tt.wantLabel)
			assert.True(t, wantLine,
				"FormatExplain should contain %q for tier %d; got:\n%s",
				tt.wantLabel, tt.tier, output)
		})
	}
}
