# T-031: Token Budget Enforcement with Truncation Strategies

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-028, T-029, T-030
**Phase:** 2 - Intelligence (Relevance & Tokens)

---

## Description

Implement the token budget enforcement engine that, given a `max_tokens` limit, includes files tier-by-tier (highest priority first) until the budget is exhausted. When a file exceeds the remaining budget, it is either truncated with a marker or skipped entirely, based on the `--truncation-strategy` flag. This task produces the final list of included/excluded files along with a summary of what was included, excluded, and why.

## User Story

As a developer with a 128K token budget, I want Harvx to automatically include my most important files first and gracefully handle the budget limit so that I get the maximum useful context within my LLM's context window.

## Acceptance Criteria

- [ ] `BudgetEnforcer` struct accepts `maxTokens int` and `strategy TruncationStrategy`
- [ ] `TruncationStrategy` is an enum: `TruncateStrategy` and `SkipStrategy`
- [ ] Default strategy is `SkipStrategy` (skip files that exceed remaining budget)
- [ ] `Enforce(files []*FileDescriptor, overhead int) *BudgetResult` processes sorted files tier-by-tier
- [ ] Files are included in tier order (0 first), alphabetically within each tier
- [ ] For `SkipStrategy`: if a file's TokenCount exceeds remaining budget, skip it entirely and continue to next file
- [ ] For `TruncateStrategy`: if a file's TokenCount exceeds remaining budget, truncate its Content to fit and add a `<!-- Content truncated: X of Y tokens shown -->` marker
- [ ] Truncation cuts at a line boundary (never mid-line) to preserve readability
- [ ] `BudgetResult` contains:
  - `IncludedFiles []*FileDescriptor` - files that made the cut
  - `ExcludedFiles []*FileDescriptor` - files that were omitted
  - `TruncatedFiles []*FileDescriptor` - files that were truncated (subset of IncludedFiles)
  - `TotalTokens int` - total tokens of included content
  - `BudgetUsed int` - total including overhead
  - `BudgetRemaining int` - remaining budget
  - `Summary BudgetSummary` - human-readable summary
- [ ] `BudgetSummary` includes per-tier stats: files included, files excluded, tokens used per tier
- [ ] When `maxTokens` is 0 or negative, all files are included (no budget enforcement)
- [ ] Overhead (from EstimateOverhead in T-030) is subtracted from budget before file processing
- [ ] Unit tests achieve 95%+ coverage

## Technical Notes

- Create in `internal/tokenizer/budget.go`
- The enforcer receives files already sorted by T-028's `SortByRelevance`. It does NOT re-sort.
- Budget enforcement is the last gate before output rendering. After this step, the `IncludedFiles` list is final.

### Truncation Algorithm (TruncateStrategy)

```
remaining_budget = max_tokens - overhead
for each file in sorted order:
    if file.TokenCount <= remaining_budget:
        include file fully
        remaining_budget -= file.TokenCount
    else if remaining_budget > 0 and strategy == truncate:
        truncate file content to fit remaining_budget tokens
        add truncation marker
        include truncated file
        remaining_budget = 0
    else:
        exclude file
```

For truncation, use a binary search approach:
1. Split content into lines
2. Binary search for the number of lines whose token count fits the remaining budget
3. Join those lines and append the truncation marker

### Skip Algorithm (SkipStrategy)

```
remaining_budget = max_tokens - overhead
for each file in sorted order:
    if file.TokenCount <= remaining_budget:
        include file fully
        remaining_budget -= file.TokenCount
    else:
        exclude file (but continue checking smaller files in same/lower tiers)
```

Note: with SkipStrategy, a large file in tier 1 that doesn't fit might be skipped, but smaller files in tier 2 could still be included. This maximizes content coverage.

### CLI Integration

The `--truncation-strategy` flag will be wired in T-033. This task only implements the enforcement logic.

- `--max-tokens` and profile `max_tokens` feed into `BudgetEnforcer.maxTokens`
- `--truncation-strategy truncate|skip` feeds into `BudgetEnforcer.strategy`

## Files to Create/Modify

- `internal/tokenizer/budget.go` - BudgetEnforcer, TruncationStrategy, BudgetResult, BudgetSummary, Enforce()
- `internal/tokenizer/budget_test.go` - Comprehensive unit tests

## Testing Requirements

- Test: 5 files, budget of 1000 tokens, all fit -> all included
- Test: 5 files (200 tokens each), budget of 600 tokens -> first 3 included, last 2 excluded
- Test: Skip strategy with mixed-size files in multiple tiers -> verify tier priority
- Test: Skip strategy skips large file but includes smaller files after it
- Test: Truncate strategy truncates a file that partially fits
- Test: Truncation cuts at line boundary (content with 10 lines, budget allows ~5 lines)
- Test: Truncation marker is appended to truncated files
- Test: Budget of 0 -> all files included (no enforcement)
- Test: Budget of -1 -> all files included
- Test: Overhead is subtracted: budget=100, overhead=50, file=60 -> excluded (60 > 50 remaining)
- Test: BudgetResult.Summary has correct per-tier stats
- Test: ExcludedFiles list is correct
- Test: TruncatedFiles is a subset of IncludedFiles
- Golden test: fixed set of 15 files across 4 tiers with known token counts and budget=5000 -> verify exact include/exclude list
- Benchmark: Enforce on 10K files with realistic token distribution