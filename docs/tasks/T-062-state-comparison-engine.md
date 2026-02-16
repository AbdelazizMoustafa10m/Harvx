# T-062: State Comparison Engine (Added/Modified/Deleted Detection)

**Priority:** Should Have
**Effort:** Medium (6-12hrs)
**Dependencies:** T-059 (State Snapshot Types), T-060 (Content Hashing), T-061 (State Cache Persistence)
**Phase:** 4 - State & Diff

---

## Description

Implement the O(n) state comparison engine that takes two `StateSnapshot` instances (previous and current) and produces a structured diff result identifying added, modified, and deleted files. This is the core diffing logic that powers both the `--diff-only` flag and the `harvx diff` subcommand. The engine uses hash-map-based comparison for O(n) performance on large repositories.

## User Story

As a developer running iterative code reviews, I want Harvx to efficiently detect exactly which files were added, modified, or deleted since the last run so that I can give my AI agents focused, relevant context.

## Acceptance Criteria

- [ ] `internal/diff/compare.go` implements `CompareStates(previous, current *StateSnapshot) *DiffResult`
- [ ] `DiffResult` struct contains: `Added []string` (new files), `Modified []string` (changed files), `Deleted []string` (removed files), `Unchanged int` (count of unchanged files)
- [ ] All file path slices in `DiffResult` are sorted alphabetically for deterministic output
- [ ] A file is classified as **added** if it exists in current but not in previous
- [ ] A file is classified as **deleted** if it exists in previous but not in current
- [ ] A file is classified as **modified** if it exists in both but the `ContentHash` differs
- [ ] A file is classified as **unchanged** if it exists in both and the `ContentHash` matches (counted but not listed)
- [ ] `DiffResult.HasChanges() bool` returns true if any files were added, modified, or deleted
- [ ] `DiffResult.TotalChanged() int` returns `len(Added) + len(Modified) + len(Deleted)`
- [ ] `DiffResult.Summary() string` returns a human-readable summary like `"3 added, 5 modified, 1 deleted (42 unchanged)"`
- [ ] Comparison is O(n) where n = max(len(previous.Files), len(current.Files)) -- iterate current files and check against previous hash map
- [ ] Handles edge cases: both snapshots empty, one snapshot empty, identical snapshots
- [ ] Unit tests achieve 95%+ coverage including edge cases

## Technical Notes

- **O(n) algorithm:**
  ```go
  func CompareStates(prev, curr *StateSnapshot) *DiffResult {
      result := &DiffResult{}
      
      // Pass 1: iterate current files -- detect added and modified
      for path, currFile := range curr.Files {
          prevFile, exists := prev.Files[path]
          if !exists {
              result.Added = append(result.Added, path)
          } else if currFile.ContentHash != prevFile.ContentHash {
              result.Modified = append(result.Modified, path)
          } else {
              result.Unchanged++
          }
      }
      
      // Pass 2: iterate previous files -- detect deleted
      for path := range prev.Files {
          if _, exists := curr.Files[path]; !exists {
              result.Deleted = append(result.Deleted, path)
          }
      }
      
      // Sort for determinism
      sort.Strings(result.Added)
      sort.Strings(result.Modified)
      sort.Strings(result.Deleted)
      
      return result
  }
  ```

- **Performance consideration:** For a 10,000-file repo, this involves two hash map lookups per file -- effectively O(n) with constant-factor overhead from map access. This should complete in well under 1ms even for large repos.

- **ModifiedTime optimization (future):** A potential optimization is to first check `ModifiedTime` and `Size` -- if both match, skip the hash comparison. This is NOT implemented in this task but the data model supports it. Add a `// TODO: optimize with mod-time check` comment for future work.

- **Nil safety:** `CompareStates` should handle nil previous snapshot gracefully (treat as empty -- all current files are "added"). This is the behavior on first run when no cache exists.

- The `DiffResult` is consumed by:
  1. The change summary renderer (T-065) to produce the summary section in output
  2. The `--diff-only` flag handler (T-064) to filter which files are included in output
  3. The `harvx diff` subcommand (T-064) for standalone diff output

## Files to Create/Modify

- `internal/diff/compare.go` - CompareStates function and DiffResult type
- `internal/diff/compare_test.go` - Comprehensive unit tests

## Testing Requirements

- Unit test: Both snapshots empty -- result has no changes, `HasChanges()` returns false
- Unit test: Previous empty, current has 3 files -- all 3 are "added"
- Unit test: Previous has 3 files, current empty -- all 3 are "deleted"
- Unit test: Identical snapshots -- no changes, `Unchanged` count matches file count
- Unit test: Mixed scenario: 2 added, 3 modified, 1 deleted, 4 unchanged -- verify all categories
- Unit test: File paths are sorted alphabetically in each category
- Unit test: `Summary()` produces correct human-readable string
- Unit test: `TotalChanged()` returns correct count
- Unit test: `HasChanges()` returns false for identical snapshots, true for any difference
- Unit test: Nil previous snapshot treated as empty (all files are "added")
- Unit test: Files with same path but different hashes are "modified"
- Unit test: Files with same path and same hash are "unchanged"
- Benchmark: Compare two 10,000-file snapshots with 100 changes -- verify sub-millisecond performance

## References

- PRD Section 5.8 ("On subsequent runs, compares current state to cached state and identifies: added files, modified files, deleted files")
- PRD Section 5.8 ("state comparison should be O(n) -- iterate current files, check against hash map")
