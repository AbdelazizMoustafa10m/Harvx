# T-014: Extension/Pattern Filtering, --git-tracked-only & Symlink Handling

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-001, T-003, T-011, T-012, T-013
**Phase:** 1 - Foundation

---

## Description

Implement three related filtering capabilities: (1) extension and glob pattern filtering via `--include`, `--exclude`, and `-f` flags using the `bmatcuk/doublestar` package; (2) `--git-tracked-only` mode that restricts discovery to files in the Git index; and (3) safe symlink handling that follows symlinks but detects and breaks loops. These are the final filtering components before the full walker is assembled.

## User Story

As a developer, I want to use `--include "src/**/*.ts"` and `-f go` to focus my context on specific file types, and I want `--git-tracked-only` for CI environments where I know the git index is authoritative.

## Acceptance Criteria

### Extension and Pattern Filtering
- [ ] `bmatcuk/doublestar` v4 is added to `go.mod`
- [ ] `internal/discovery/filter.go` defines a `PatternFilter` type
- [ ] `PatternFilter` supports:
  - Include patterns (`--include "src/**/*.ts"`): if any include patterns are set, only files matching at least one are kept
  - Exclude patterns (`--exclude "**/*.test.ts"`): files matching any exclude pattern are removed
  - Extension filters (`-f ts -f go`): shorthand for include by extension
- [ ] Include and extension filters are combined with OR logic (file must match at least one)
- [ ] Exclude patterns take precedence over includes (exclude always wins)
- [ ] Extension matching is case-insensitive (`.TS` matches `-f ts`)
- [ ] Doublestar patterns work correctly: `**/*.ts` matches `src/deep/nested/file.ts`
- [ ] `PatternFilter.Matches(path string) bool` returns true if the file should be included
- [ ] When no include/filter patterns are set, all files pass (no filtering)

### --git-tracked-only Mode
- [ ] `internal/discovery/git_tracked.go` defines a `GitTrackedFiles(root string) (map[string]bool, error)` function
- [ ] Implementation runs `git ls-files` in the root directory and parses its output
- [ ] Returns a set (map) of file paths relative to the root
- [ ] Handles the case where the directory is not a git repo (returns error with clear message)
- [ ] Handles empty repos (no tracked files) gracefully
- [ ] Works correctly in subdirectories of a git repo
- [ ] The result is used as a whitelist: only files in the set pass through

### Symlink Handling
- [ ] `internal/discovery/symlink.go` defines symlink detection and resolution logic
- [ ] Symlinks are followed (resolved to their targets) for file content reading
- [ ] Symlink loops are detected by tracking visited real paths (via `filepath.EvalSymlinks`)
- [ ] When a loop is detected, the symlink is skipped with a debug-level log message
- [ ] Dangling symlinks (pointing to non-existent targets) are skipped with a warning
- [ ] Symlink detection works on all platforms (macOS, Linux, Windows)

### Integration
- [ ] All three filters can be composed together in the discovery pipeline
- [ ] Filter order: git-tracked-only whitelist -> ignore patterns -> pattern filter -> binary/size check

## Technical Notes

- `bmatcuk/doublestar` v4 is the current major version (ref: https://github.com/bmatcuk/doublestar). Install: `go get github.com/bmatcuk/doublestar/v4`
- Pattern matching with doublestar:
  ```go
  import "github.com/bmatcuk/doublestar/v4"

  matched, err := doublestar.Match(pattern, path)
  ```
- For `--git-tracked-only`, use `exec.Command("git", "ls-files")`:
  ```go
  func GitTrackedFiles(root string) (map[string]bool, error) {
      cmd := exec.Command("git", "ls-files")
      cmd.Dir = root
      output, err := cmd.Output()
      if err != nil {
          return nil, fmt.Errorf("git ls-files failed: %w (is this a git repository?)", err)
      }
      files := make(map[string]bool)
      scanner := bufio.NewScanner(bytes.NewReader(output))
      for scanner.Scan() {
          files[scanner.Text()] = true
      }
      return files, scanner.Err()
  }
  ```
- Per PRD Section 5.1: "Supports `--git-tracked-only` mode that only includes files in the git index (sidesteps gitignore edge cases, ideal for CI)."
- Per PRD Section 5.1: "Handles symlinks safely (detect and skip loops)."
- Symlink loop detection:
  ```go
  func isSymlinkLoop(path string, visited map[string]bool) (bool, string, error) {
      realPath, err := filepath.EvalSymlinks(path)
      if err != nil {
          return false, "", err
      }
      if visited[realPath] {
          return true, realPath, nil
      }
      return false, realPath, nil
  }
  ```
- Reference: PRD Sections 5.1, 5.9

## Files to Create/Modify

- `go.mod` / `go.sum` - Add `bmatcuk/doublestar/v4`
- `internal/discovery/filter.go` - PatternFilter implementation
- `internal/discovery/filter_test.go` - Unit tests
- `internal/discovery/git_tracked.go` - Git tracked files implementation
- `internal/discovery/git_tracked_test.go` - Unit tests
- `internal/discovery/symlink.go` - Symlink handling
- `internal/discovery/symlink_test.go` - Unit tests
- `testdata/symlinks/` - Test fixtures with symlink structures

## Testing Requirements

### Pattern Filtering Tests
- Unit test: `--include "**/*.ts"` keeps only `.ts` files
- Unit test: `--exclude "**/*.test.ts"` removes test files
- Unit test: `-f ts -f go` keeps `.ts` and `.go` files
- Unit test: `-f .ts` (with dot) is normalized to `ts`
- Unit test: exclude takes precedence over include
- Unit test: no filters set allows all files through
- Unit test: case-insensitive extension matching (`.TS` matches `-f ts`)
- Unit test: doublestar patterns match deeply nested paths

### Git Tracked Tests
- Unit test: returns file set from `git ls-files` output (mock command execution)
- Unit test: non-git directory returns descriptive error
- Unit test: empty repo returns empty set
- Unit test: file paths are relative to root

### Symlink Tests
- Unit test: regular file is not a symlink
- Unit test: symlink to file is followed
- Unit test: symlink to directory is followed
- Unit test: symlink loop is detected and skipped
- Unit test: dangling symlink is detected and skipped
- Unit test: chain of symlinks (A -> B -> C) resolves correctly
