# T-027: Glob-Based File-to-Tier Matching

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-026
**Phase:** 2 - Intelligence (Relevance & Tokens)

---

## Description

Implement the pattern-matching engine that assigns each discovered file to exactly one relevance tier. Using `bmatcuk/doublestar/v4` for glob matching, this component takes a list of file paths and a set of tier definitions (either defaults from T-026 or profile-defined overrides) and returns each file tagged with its tier. Files match the first (highest priority) tier where a pattern matches; no file appears in multiple tiers. Unmatched files are assigned to Tier 2.

## User Story

As a developer with a custom profile, I want my relevance tier patterns (e.g., `app/api/**` = tier 1) to be matched against discovered files using standard glob syntax so that the most important files are prioritized in my context output.

## Acceptance Criteria

- [ ] `TierMatcher` struct accepts a slice of `TierDefinition` and provides a `Match(filePath string) int` method
- [ ] Pattern matching uses `doublestar.Match()` or `doublestar.PathMatch()` from `bmatcuk/doublestar/v4`
- [ ] A file matches the **first** (highest priority / lowest tier number) tier where any pattern matches
- [ ] No file appears in multiple tiers
- [ ] Unmatched files return Tier 2 (the source code default)
- [ ] `ClassifyFiles(files []string, tiers []TierDefinition) map[string]int` bulk-classifies files
- [ ] Performance: O(n * m) where n = files, m = total patterns across all tiers -- acceptable for repos up to 50K files
- [ ] Profile-defined tiers replace default tiers entirely (no merging)
- [ ] Handles edge cases: empty pattern list, empty file list, files with special characters, deeply nested paths
- [ ] Unit tests achieve 95%+ coverage

## Technical Notes

- Create in `internal/relevance/matcher.go`
- Use `github.com/bmatcuk/doublestar/v4` -- specifically `doublestar.Match(pattern, name)` for path matching
  - `doublestar.Match()` splits on `/` and works with forward-slash paths
  - For Windows compatibility, consider also exposing `PathMatch()` which uses OS separators
- All file paths passed to the matcher should be relative paths (from the repo root), normalized to forward slashes
- Pattern iteration order matters: iterate tiers 0 through 5, and within each tier iterate patterns in order. First match wins.
- Pre-validate patterns at construction time using `doublestar.ValidatePattern()` if available, or attempt a dry-run match
- The matcher should be constructed once and reused for all files (compile patterns once)
- Consider caching compiled pattern state if doublestar supports it (check library API)

### Key doublestar v4 API

```go
import "github.com/bmatcuk/doublestar/v4"

// Match reports whether name matches the pattern.
// Pattern and name are split on forward slash (/) characters.
matched, err := doublestar.Match(pattern, name)

// PathMatch is like Match but uses OS path separators.
matched, err := doublestar.PathMatch(pattern, name)
```

### Dependencies & Versions

| Package/Library | Version | Purpose |
|-----------------|---------|---------|
| github.com/bmatcuk/doublestar/v4 | v4.7.1+ | Glob pattern matching with ** support |

## Files to Create/Modify

- `internal/relevance/matcher.go` - TierMatcher struct, Match(), ClassifyFiles()
- `internal/relevance/matcher_test.go` - Comprehensive unit tests

## Testing Requirements

- Table-driven tests for individual file matching against default tiers:
  - `go.mod` -> Tier 0
  - `package.json` -> Tier 0
  - `Dockerfile` -> Tier 0
  - `next.config.js` -> Tier 0 (matches *.config.*)
  - `src/main.go` -> Tier 1
  - `internal/server/handler.go` -> Tier 1
  - `components/Button.tsx` -> Tier 2 (secondary source)
  - `utils/helpers.go` -> Tier 2 (unmatched default)
  - `main_test.go` -> Tier 3
  - `src/app.test.ts` -> Tier 3
  - `__tests__/unit/foo.ts` -> Tier 3
  - `README.md` -> Tier 4
  - `docs/architecture.md` -> Tier 4
  - `.github/workflows/ci.yml` -> Tier 5
  - `package-lock.json` -> Tier 5
- Test that first-tier-wins rule is enforced (file matching both tier 1 and tier 3 patterns gets tier 1)
- Test with custom tier definitions (profile override scenario)
- Test with empty tier definitions (all files should get tier 2)
- Test with special characters in file paths
- Benchmark test for ClassifyFiles with 10K files and typical pattern counts (~20 patterns)
- Test Windows-style paths if PathMatch is used