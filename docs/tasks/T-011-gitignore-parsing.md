# T-011: .gitignore Parsing & Matching

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-001, T-003
**Phase:** 1 - Foundation

---

## Description

Implement `.gitignore` file parsing and pattern matching using the `sabhiram/go-gitignore` package. Support nested `.gitignore` files (each directory can have its own), merge patterns hierarchically, and provide a clean interface for the file discovery walker to check whether a given path should be ignored. This is a core filtering component that must match Git's behavior faithfully.

## User Story

As a developer, I want Harvx to respect my `.gitignore` rules exactly like Git does so that I never see `node_modules`, `dist`, or other ignored files in my context output.

## Acceptance Criteria

- [ ] `sabhiram/go-gitignore` package is added to `go.mod`
- [ ] `internal/discovery/gitignore.go` defines a `GitignoreMatcher` type that:
  - Loads and parses `.gitignore` files at any directory level
  - Supports nested `.gitignore` files (a `.gitignore` in `src/` applies only to files under `src/`)
  - Merges patterns from parent directories (root `.gitignore` applies everywhere, nested ones add to it)
  - Correctly handles negation patterns (`!important.log`)
  - Correctly handles directory-only patterns (`build/` matches directories named `build`)
  - Correctly handles wildcard patterns (`*.log`, `**/*.tmp`)
- [ ] `GitignoreMatcher` exposes an `IsIgnored(path string, isDir bool) bool` method
- [ ] Paths passed to `IsIgnored` are relative to the root directory
- [ ] The matcher is initialized with a root directory and recursively discovers all `.gitignore` files during construction
- [ ] Handles missing `.gitignore` gracefully (no error if no `.gitignore` exists)
- [ ] Golden tests compare `GitignoreMatcher.IsIgnored()` results against `git check-ignore` for a reference test suite
- [ ] Performance: matching a single path is O(patterns), not O(files)
- [ ] Unit tests cover edge cases: empty `.gitignore`, comments, blank lines, trailing whitespace, escaped characters

## Technical Notes

- `sabhiram/go-gitignore` v1.1.0 is the latest version. Install: `go get github.com/sabhiram/go-gitignore`
- The package provides:
  ```go
  ignore, err := gitignore.CompileIgnoreFile(".gitignore")
  isMatch := ignore.MatchesPath("some/file.txt")
  ```
- For nested `.gitignore` support, we need to build a hierarchical matcher:
  ```go
  type GitignoreMatcher struct {
      root     string
      matchers map[string]*gitignore.GitIgnore  // dir path -> compiled patterns
  }
  ```
- When checking if a path is ignored, iterate from the root down to the file's parent directory, checking each level's matcher.
- Per PRD Section 5.1: "Respects `.gitignore` rules (including nested `.gitignore` files)." and "Use the `sabhiram/go-gitignore` package for `.gitignore` parsing with full pattern support. Must match Git's behavior for a reference test suite (golden tests against `git check-ignore`)."
- Create a `testdata/gitignore/` directory with sample repo structures and `.gitignore` files for golden testing.
- The golden test approach: create a small test repo under `testdata/`, run `git check-ignore` against known paths, store results, and verify our matcher produces identical results.
- This module does NOT handle `.harvxignore` -- that is T-012.
- Reference: PRD Section 5.1

## Files to Create/Modify

- `go.mod` / `go.sum` - Add `sabhiram/go-gitignore`
- `internal/discovery/gitignore.go` - GitignoreMatcher implementation
- `internal/discovery/gitignore_test.go` - Unit tests
- `testdata/gitignore/` - Test fixtures with sample `.gitignore` files and directory structures

## Testing Requirements

- Unit test: basic pattern matching (`*.log` matches `error.log`)
- Unit test: directory pattern (`build/` matches `build/` directory but not `build` file)
- Unit test: negation pattern (`!important.log` overrides `*.log`)
- Unit test: doublestar pattern (`**/*.tmp` matches `deep/nested/file.tmp`)
- Unit test: nested `.gitignore` applies only to subdirectory
- Unit test: parent `.gitignore` rules are inherited by subdirectories
- Unit test: no `.gitignore` file present returns false for all paths
- Unit test: comments and blank lines are skipped
- Unit test: trailing whitespace is handled per Git spec
- Golden test: compare matcher output against `git check-ignore` for reference paths
- Benchmark: matching 10,000 paths against a complex `.gitignore` completes in < 100ms
