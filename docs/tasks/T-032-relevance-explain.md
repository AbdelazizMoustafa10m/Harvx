# T-032: Relevance Explain and Inclusion Summary

**Priority:** Should Have
**Effort:** Medium (6-8hrs)
**Dependencies:** T-027, T-028, T-031
**Phase:** 2 - Intelligence (Relevance & Tokens)

---

## Description

Implement the `explain` functionality that shows which tier/rule/pattern applies to a specific file and why it was included or excluded. Also implement the inclusion summary that is rendered in the output header, showing per-tier file counts, token usage, and excluded files with reasons. This powers both the `harvx profiles explain <filepath>` CLI command and the summary section in generated output.

## User Story

As a developer troubleshooting why a file was excluded or placed in an unexpected tier, I want to run `harvx profiles explain <filepath>` and see exactly which pattern matched (or didn't match), what tier was assigned, and whether it was budget-excluded, so that I can tune my profile configuration.

## Acceptance Criteria

- [ ] `Explain(filePath string, tiers []TierDefinition) *ExplainResult` returns detailed matching info
- [ ] `ExplainResult` contains:
  - `FilePath string` - the queried file
  - `AssignedTier int` - the tier the file was assigned to
  - `MatchedPattern string` - the specific glob pattern that matched (or "" if default tier)
  - `MatchedTierDef int` - which tier definition contained the matching pattern
  - `IsDefault bool` - true if the file matched no pattern and defaulted to tier 2
  - `AllMatches []PatternMatch` - all patterns that would have matched (for debugging overlapping rules)
  - `WouldBeIncluded bool` - whether the file would survive budget enforcement (requires budget context)
  - `ExclusionReason string` - if excluded: "budget_exceeded", "filtered_by_ignore", etc.
- [ ] `PatternMatch` struct: `Tier int`, `Pattern string`
- [ ] `FormatExplain(result *ExplainResult) string` produces a human-readable explanation:
  ```
  File: src/api/handler.go
  Tier: 1 (Source Code)
  Matched Pattern: src/** (from tier 1)
  Budget Status: Included (tokens: 450)
  
  All matching patterns:
    - Tier 1: src/**
    - Tier 2: (default, unmatched)
  ```
- [ ] `GenerateInclusionSummary(result *BudgetResult) string` produces the summary for output:
  ```
  Files: 342 included, 48 excluded
  
  By Tier:
    Tier 0 (Config):      5 files,   2,100 tokens
    Tier 1 (Source):      48 files,  45,000 tokens
    Tier 2 (Secondary):  180 files,  35,000 tokens
    Tier 3 (Tests):       62 files,   5,000 tokens (42 excluded by budget)
    Tier 4 (Docs):        30 files,   1,500 tokens
    Tier 5 (CI/Lock):     17 files,     820 tokens (6 excluded by budget)
  
  Total: 89,420 tokens / 200,000 budget (45%)
  ```
- [ ] Tier labels are human-readable: "Config", "Source", "Secondary", "Tests", "Docs", "CI/Lock"
- [ ] Unit tests achieve 90%+ coverage

## Technical Notes

- Create in `internal/relevance/explain.go`
- The `Explain` function reuses `TierMatcher` from T-027 but iterates **all** tiers and **all** patterns to find every match, not just the first
- `FormatExplain` is used by the `harvx profiles explain` CLI subcommand (wired in a separate CLI task)
- `GenerateInclusionSummary` is used by the output renderer to include tier breakdown in the output header
- Tier labels should be configurable but default to: 0="Config", 1="Source", 2="Secondary", 3="Tests", 4="Docs", 5="CI/Lock"

### Integration with Budget

The `WouldBeIncluded` field requires access to the `BudgetResult` from T-031. Two options:
1. The `Explain` function takes an optional `BudgetResult` parameter
2. The caller enriches the `ExplainResult` after budget enforcement

Option 2 is cleaner: the explain function focuses on tier matching, and the caller adds budget context.

## Files to Create/Modify

- `internal/relevance/explain.go` - Explain(), ExplainResult, PatternMatch, FormatExplain(), GenerateInclusionSummary()
- `internal/relevance/explain_test.go` - Unit tests

## Testing Requirements

- Test: Explain a file that matches tier 0 pattern -> correct tier, pattern, not default
- Test: Explain a file with no pattern match -> tier 2, IsDefault=true, MatchedPattern=""
- Test: Explain a file that matches patterns in multiple tiers -> AssignedTier is lowest number, AllMatches shows all
- Test: FormatExplain produces readable output for matched file
- Test: FormatExplain produces readable output for default/unmatched file
- Test: GenerateInclusionSummary with a BudgetResult containing files across all 6 tiers
- Test: GenerateInclusionSummary with all files included (no exclusions)
- Test: GenerateInclusionSummary with all files excluded (budget = 0 edge case handled specially)
- Test: Tier labels are correct for all 6 tiers