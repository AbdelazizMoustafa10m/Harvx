# T-028: Relevance Sorter -- Sort Files by Tier and Path

**Priority:** Must Have
**Effort:** Small (4-6hrs)
**Dependencies:** T-026, T-027
**Phase:** 2 - Intelligence (Relevance & Tokens)

---

## Description

Implement the sorter that takes a list of `FileDescriptor` structs (each already tagged with a tier from T-027) and produces a deterministic sort order: primary sort by tier (0 first, 5 last), secondary sort alphabetically by relative path within each tier. This sorted list becomes the canonical order for output rendering and token budget enforcement. Also integrates with profile-defined tier overrides, selecting between default and custom tier definitions.

## User Story

As a developer, I want my context output to show configuration files first, then core source code, then tests, then docs, then CI config -- in a deterministic, reproducible order -- so that the LLM always sees the most important files first.

## Acceptance Criteria

- [ ] `SortByRelevance(files []*FileDescriptor) []*FileDescriptor` sorts in-place or returns a new sorted slice
- [ ] Primary sort: ascending tier number (tier 0 first)
- [ ] Secondary sort: alphabetical by `Path` (relative path) within each tier
- [ ] Sort is stable (files with identical tier and path maintain original insertion order -- though this should not happen in practice since paths are unique)
- [ ] Output is deterministic: same input always produces same order
- [ ] `GroupByTier(files []*FileDescriptor) map[int][]*FileDescriptor` groups files by tier for summary reporting
- [ ] `TierSummary(files []*FileDescriptor) []TierStat` returns per-tier counts and total token counts for summary output
- [ ] `TierStat` struct: `Tier int`, `FileCount int`, `TotalTokens int`, `FilePaths []string`
- [ ] Integrates with `TierMatcher` from T-027 via a `ClassifyAndSort` convenience function
- [ ] Unit tests achieve 95%+ coverage

## Technical Notes

- Create in `internal/relevance/sorter.go`
- Use `sort.Slice` or `slices.SortFunc` (Go 1.22+) for the sort implementation
- The `FileDescriptor` type is defined in `internal/pipeline/types.go` (from Phase 1). The `Tier` field should already exist on it. If not yet implemented, define a minimal version here that can be reconciled later.
- The `ClassifyAndSort` function should:
  1. Accept files and tier definitions
  2. Use `TierMatcher` to assign tiers
  3. Sort the result
  4. Return the sorted list
- For summary output, `TierSummary` is used by the output renderer to display included/excluded file counts per tier
- The sorter does NOT handle token budgeting -- that is T-029's responsibility. This task focuses purely on ordering.

### Expected FileDescriptor fields used

```go
type FileDescriptor struct {
    Path       string // Relative path from root (sort key)
    Tier       int    // Relevance tier 0-5 (primary sort key)
    TokenCount int    // Token count (used for summary, set by tokenizer)
    // ... other fields
}
```

## Files to Create/Modify

- `internal/relevance/sorter.go` - SortByRelevance(), GroupByTier(), TierSummary(), ClassifyAndSort()
- `internal/relevance/sorter_test.go` - Unit tests
- `internal/pipeline/types.go` - Ensure FileDescriptor has `Tier` field (create minimal version if not yet existing)

## Testing Requirements

- Test sorting 3 files across different tiers produces correct order
- Test alphabetical sorting within a single tier (e.g., `a/foo.go` before `b/bar.go` in tier 1)
- Test with all files in the same tier (should be purely alphabetical)
- Test with empty input (returns empty slice)
- Test with single file (returns same file)
- Test GroupByTier returns correct grouping
- Test TierSummary returns correct counts and token totals
- Test determinism: sort the same input twice, compare results
- Golden test: a fixed set of 20 files with known tiers, verify exact output order
- Benchmark: sort 10K files with realistic tier distribution