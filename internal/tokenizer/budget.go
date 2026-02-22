// Package tokenizer provides token counting implementations for LLM context
// documents. This file implements token budget enforcement with pluggable
// truncation strategies for the pipeline's relevance-aware file inclusion.
package tokenizer

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/harvx/harvx/internal/pipeline"
)

// TruncationStrategy controls how BudgetEnforcer handles files that exceed
// the remaining token budget.
type TruncationStrategy string

const (
	// SkipStrategy skips files that exceed the remaining budget and continues
	// to the next file. Smaller subsequent files may still be included.
	// This is the default strategy.
	SkipStrategy TruncationStrategy = "skip"

	// TruncateStrategy truncates the first file that exceeds the remaining
	// budget to fit exactly, appending a truncation marker. All remaining
	// files after the first truncation are excluded because the budget is
	// fully consumed.
	TruncateStrategy TruncationStrategy = "truncate"
)

// TierStat holds per-tier inclusion statistics produced by BudgetEnforcer.
type TierStat struct {
	// FilesIncluded is the number of files from this tier that were included
	// (fully or truncated) in the output.
	FilesIncluded int

	// FilesExcluded is the number of files from this tier that were excluded
	// because the budget was exhausted.
	FilesExcluded int

	// TokensUsed is the sum of TokenCount for all included files in this tier.
	TokensUsed int
}

// BudgetSummary holds per-tier statistics for a single budget enforcement run.
type BudgetSummary struct {
	// TierStats maps tier number (0-5) to the corresponding TierStat.
	// Only tiers that had at least one file processed are present.
	TierStats map[int]TierStat
}

// BudgetResult is the output of a single BudgetEnforcer.Enforce call. It
// separates files into included, excluded, and truncated buckets and provides
// aggregate token accounting.
type BudgetResult struct {
	// IncludedFiles holds files that were included at their full token count.
	IncludedFiles []*pipeline.FileDescriptor

	// ExcludedFiles holds files that were dropped because the budget was
	// exhausted before they could be included.
	ExcludedFiles []*pipeline.FileDescriptor

	// TruncatedFiles holds files whose Content was shortened to fit the
	// remaining budget. These files also appear in IncludedFiles.
	TruncatedFiles []*pipeline.FileDescriptor

	// TotalTokens is the sum of TokenCount across all IncludedFiles (after
	// truncation adjustments).
	TotalTokens int

	// BudgetUsed is overhead + TotalTokens.
	BudgetUsed int

	// BudgetRemaining is maxTokens - BudgetUsed. May be negative when overhead
	// alone exceeds maxTokens.
	BudgetRemaining int

	// Summary provides per-tier statistics for the enforcement run.
	Summary BudgetSummary
}

// BudgetEnforcer enforces a maximum token budget over an ordered slice of
// FileDescriptors, applying the configured TruncationStrategy when a file
// exceeds the remaining budget. It is safe for sequential use only; do not
// call Enforce from multiple goroutines simultaneously.
type BudgetEnforcer struct {
	maxTokens int
	strategy  TruncationStrategy
	tok       Tokenizer
}

// NewBudgetEnforcer constructs a BudgetEnforcer.
//
// maxTokens is the hard upper bound on total tokens (files + overhead). When
// maxTokens <= 0 all files are included without enforcement.
//
// strategy controls what happens when a file's TokenCount exceeds the remaining
// budget: SkipStrategy moves on to the next file, TruncateStrategy trims the
// file content at a line boundary to fill the remaining budget exactly.
//
// tok is used to count tokens of candidate line subsets during the binary
// search in TruncateStrategy. Pass nil to fall back to the character estimator
// (len/4), which is fast but less accurate.
func NewBudgetEnforcer(maxTokens int, strategy TruncationStrategy, tok Tokenizer) *BudgetEnforcer {
	if tok == nil {
		tok = newEstimatorTokenizer()
	}
	return &BudgetEnforcer{
		maxTokens: maxTokens,
		strategy:  strategy,
		tok:       tok,
	}
}

// Enforce applies the token budget to files and returns a BudgetResult.
//
// files must already be sorted by tier then path (as produced by T-028); they
// are processed in the provided order without re-sorting.
//
// overhead is the estimated token cost of output document structure (headers,
// file tree, section markers). It is subtracted from maxTokens before
// evaluating individual files.
//
// When maxTokens <= 0 all files are included, overhead is ignored, and the
// result reports zero budget fields.
func (e *BudgetEnforcer) Enforce(files []*pipeline.FileDescriptor, overhead int) *BudgetResult {
	result := &BudgetResult{
		IncludedFiles: make([]*pipeline.FileDescriptor, 0, len(files)),
		ExcludedFiles: make([]*pipeline.FileDescriptor, 0),
		TruncatedFiles: make([]*pipeline.FileDescriptor, 0),
		Summary: BudgetSummary{
			TierStats: make(map[int]TierStat),
		},
	}

	// When no budget is configured, include everything.
	if e.maxTokens <= 0 {
		result.IncludedFiles = append(result.IncludedFiles, files...)
		for _, fd := range files {
			result.TotalTokens += fd.TokenCount
			stat := result.Summary.TierStats[fd.Tier]
			stat.FilesIncluded++
			stat.TokensUsed += fd.TokenCount
			result.Summary.TierStats[fd.Tier] = stat
		}
		return result
	}

	remaining := e.maxTokens - overhead
	slog.Debug("budget enforcement started",
		"maxTokens", e.maxTokens,
		"overhead", overhead,
		"remaining", remaining,
		"strategy", string(e.strategy),
		"fileCount", len(files),
	)

	switch e.strategy {
	case TruncateStrategy:
		e.enforceWithTruncate(files, remaining, result)
	default:
		// SkipStrategy is the default for any unrecognised value.
		e.enforceWithSkip(files, remaining, result)
	}

	result.BudgetUsed = overhead + result.TotalTokens
	result.BudgetRemaining = e.maxTokens - result.BudgetUsed

	slog.Debug("budget enforcement complete",
		"included", len(result.IncludedFiles),
		"excluded", len(result.ExcludedFiles),
		"truncated", len(result.TruncatedFiles),
		"totalTokens", result.TotalTokens,
		"budgetUsed", result.BudgetUsed,
		"budgetRemaining", result.BudgetRemaining,
	)

	return result
}

// enforceWithSkip runs the skip strategy: files that exceed remaining budget
// are skipped, but iteration continues so that smaller later files may still
// fit within the remaining budget.
func (e *BudgetEnforcer) enforceWithSkip(
	files []*pipeline.FileDescriptor,
	remaining int,
	result *BudgetResult,
) {
	for _, fd := range files {
		if fd.TokenCount <= remaining {
			result.IncludedFiles = append(result.IncludedFiles, fd)
			result.TotalTokens += fd.TokenCount
			remaining -= fd.TokenCount

			stat := result.Summary.TierStats[fd.Tier]
			stat.FilesIncluded++
			stat.TokensUsed += fd.TokenCount
			result.Summary.TierStats[fd.Tier] = stat

			slog.Debug("file included",
				"path", fd.Path,
				"tier", fd.Tier,
				"tokens", fd.TokenCount,
				"remaining", remaining,
			)
		} else {
			result.ExcludedFiles = append(result.ExcludedFiles, fd)

			stat := result.Summary.TierStats[fd.Tier]
			stat.FilesExcluded++
			result.Summary.TierStats[fd.Tier] = stat

			slog.Debug("file skipped (exceeds budget)",
				"path", fd.Path,
				"tier", fd.Tier,
				"tokens", fd.TokenCount,
				"remaining", remaining,
			)
		}
	}
}

// enforceWithTruncate runs the truncate strategy: the first file that exceeds
// the remaining budget is truncated at a line boundary to consume exactly
// `remaining` tokens. All subsequent files are excluded because the budget is
// now fully consumed after the truncation.
func (e *BudgetEnforcer) enforceWithTruncate(
	files []*pipeline.FileDescriptor,
	remaining int,
	result *BudgetResult,
) {
	budgetExhausted := false

	for _, fd := range files {
		if budgetExhausted {
			result.ExcludedFiles = append(result.ExcludedFiles, fd)

			stat := result.Summary.TierStats[fd.Tier]
			stat.FilesExcluded++
			result.Summary.TierStats[fd.Tier] = stat
			continue
		}

		if fd.TokenCount <= remaining {
			// File fits fully within the remaining budget.
			result.IncludedFiles = append(result.IncludedFiles, fd)
			result.TotalTokens += fd.TokenCount
			remaining -= fd.TokenCount

			stat := result.Summary.TierStats[fd.Tier]
			stat.FilesIncluded++
			stat.TokensUsed += fd.TokenCount
			result.Summary.TierStats[fd.Tier] = stat

			slog.Debug("file included",
				"path", fd.Path,
				"tier", fd.Tier,
				"tokens", fd.TokenCount,
				"remaining", remaining,
			)
		} else if remaining > 0 {
			// File exceeds budget; truncate it to fit.
			truncated := e.truncateToFit(fd, remaining)

			result.IncludedFiles = append(result.IncludedFiles, truncated)
			result.TruncatedFiles = append(result.TruncatedFiles, truncated)
			result.TotalTokens += truncated.TokenCount

			stat := result.Summary.TierStats[fd.Tier]
			stat.FilesIncluded++
			stat.TokensUsed += truncated.TokenCount
			result.Summary.TierStats[fd.Tier] = stat

			slog.Debug("file truncated",
				"path", fd.Path,
				"tier", fd.Tier,
				"originalTokens", fd.TokenCount,
				"truncatedTokens", truncated.TokenCount,
				"remaining", remaining,
			)

			remaining = 0
			budgetExhausted = true
		} else {
			// remaining == 0: budget is already fully consumed.
			result.ExcludedFiles = append(result.ExcludedFiles, fd)

			stat := result.Summary.TierStats[fd.Tier]
			stat.FilesExcluded++
			result.Summary.TierStats[fd.Tier] = stat

			budgetExhausted = true
		}
	}
}

// truncateToFit creates a shallow copy of fd with Content and TokenCount
// adjusted so that the content fits within remaining tokens. It finds the
// maximum number of lines whose joined token count is <= remaining via binary
// search, then appends a truncation marker.
//
// The original fd is never mutated; the returned descriptor is a new value.
func (e *BudgetEnforcer) truncateToFit(fd *pipeline.FileDescriptor, remaining int) *pipeline.FileDescriptor {
	lines := strings.Split(fd.Content, "\n")
	n := len(lines)

	// Reserve tokens for the truncation marker itself.
	// "<!-- Content truncated: X of Y tokens shown -->" is at most ~60 chars.
	// We use a small fixed reservation so the marker always fits.
	const markerReservation = 20
	budgetForContent := remaining - markerReservation
	if budgetForContent <= 0 {
		budgetForContent = 0
	}

	// Binary search for the maximum k (number of lines) such that the joined
	// content of lines[0:k] has a token count <= budgetForContent.
	//
	// Invariant: lines[0:lo] always fits, lines[0:hi] may not.
	lo, hi := 0, n
	for lo < hi {
		mid := (lo + hi + 1) / 2 // round up to avoid infinite loop when hi = lo+1
		candidate := strings.Join(lines[:mid], "\n")
		if e.tok.Count(candidate) <= budgetForContent {
			lo = mid
		} else {
			hi = mid - 1
		}
	}

	// lo is now the maximum number of lines that fit.
	keptLines := lines[:lo]
	keptContent := strings.Join(keptLines, "\n")

	// Build the truncation marker.
	shownTokens := e.tok.Count(keptContent)
	marker := fmt.Sprintf("<!-- Content truncated: %d of %d tokens shown -->", shownTokens, fd.TokenCount)

	var truncatedContent string
	if keptContent == "" {
		truncatedContent = marker
	} else {
		truncatedContent = keptContent + "\n" + marker
	}

	// Count the actual tokens in the final truncated content to set TokenCount
	// accurately (includes the marker).
	actualTokens := e.tok.Count(truncatedContent)

	// Shallow-copy the descriptor; only Content and TokenCount differ.
	truncated := *fd
	truncated.Content = truncatedContent
	truncated.TokenCount = actualTokens

	slog.Debug("truncation result",
		"path", fd.Path,
		"linesKept", lo,
		"linesTotal", n,
		"shownTokens", shownTokens,
		"totalTokens", fd.TokenCount,
		"actualTokens", actualTokens,
	)

	return &truncated
}

// SortedTierKeys returns the tier numbers present in the BudgetSummary,
// sorted in ascending order. This is a convenience helper for deterministic
// reporting and testing.
func (s *BudgetSummary) SortedTierKeys() []int {
	keys := make([]int, 0, len(s.TierStats))
	for k := range s.TierStats {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}
