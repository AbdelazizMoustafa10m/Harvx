# T-061: State Cache Persistence (Read/Write)

**Priority:** Should Have
**Effort:** Medium (6-12hrs)
**Dependencies:** T-059 (State Snapshot Types), T-060 (Content Hashing)
**Phase:** 4 - State & Diff

---

## Description

Implement the state cache layer that persists state snapshots to disk and loads them on subsequent runs. State files are stored at `.harvx/state/<profile-name>.json` and are profile-scoped with branch metadata to avoid stale diffs across branch switches. This task handles directory creation, atomic file writes (write-to-temp-then-rename), cache invalidation on branch change, and the `--clear-cache` reset functionality.

## User Story

As a developer, I want Harvx to automatically save its state after each run so that next time I run it, I get a fast diff of what changed without any manual bookkeeping.

## Acceptance Criteria

- [ ] `internal/diff/cache.go` implements `StateCache` struct with methods for reading and writing state files
- [ ] `SaveState(rootDir string, snapshot *StateSnapshot) error` persists snapshot to `.harvx/state/<profile-name>.json` relative to `rootDir`
- [ ] `LoadState(rootDir, profileName string) (*StateSnapshot, error)` reads the most recent snapshot for a profile
- [ ] `ClearState(rootDir, profileName string) error` deletes the cached state for a specific profile
- [ ] `ClearAllState(rootDir string) error` deletes the entire `.harvx/state/` directory
- [ ] `HasState(rootDir, profileName string) bool` checks if a cached state exists for a profile
- [ ] `GetStatePath(rootDir, profileName string) string` returns the expected file path for a profile's state
- [ ] Directory `.harvx/state/` is created automatically if it does not exist (using `os.MkdirAll` with 0755 permissions)
- [ ] File writes are atomic: write to a temporary file in the same directory, then `os.Rename` to the final path (prevents corruption on crash/interrupt)
- [ ] State files include branch metadata; when loading, if the cached state's `GitBranch` differs from the current branch, return a `ErrBranchMismatch` sentinel error (caller decides whether to use stale state or regenerate)
- [ ] Profile names are sanitized for filesystem safety: only `[a-zA-Z0-9_-]` characters allowed, others replaced with `_`
- [ ] State files use 0644 permissions
- [ ] `.harvx/` directory is in the default ignore list (confirmed by PRD; this task verifies it is present in `internal/config/defaults.go`)
- [ ] Unit tests achieve 90%+ coverage

## Technical Notes

- **Atomic writes pattern:**
  ```go
  tmpFile, err := os.CreateTemp(dir, ".state-*.tmp")
  // write JSON to tmpFile
  tmpFile.Close()
  os.Rename(tmpFile.Name(), finalPath)
  ```
  Using `os.CreateTemp` in the same directory as the final path ensures `os.Rename` is atomic on POSIX systems (same filesystem). On Windows, `os.Rename` may fail if the destination exists; use `os.Remove` + `os.Rename` as fallback.

- **Branch mismatch handling:** The cache stores the branch name at write time. On load, compare the stored branch with the current branch. If they differ, return a typed error `ErrBranchMismatch` with both branch names. The caller (the diff subcommand or the generate pipeline) can then decide to:
  1. Ignore the mismatch and diff anyway (useful for comparing branches)
  2. Clear cache and generate fresh (default behavior when not in diff mode)

- **Profile name sanitization:** Profile names come from TOML config and CLI flags. They should be safe for use as filenames. Use a regexp to replace unsafe characters:
  ```go
  var safeProfileRe = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
  func sanitizeProfileName(name string) string {
      return safeProfileRe.ReplaceAllString(name, "_")
  }
  ```

- **State directory layout:**
  ```
  .harvx/
  └── state/
      ├── default.json
      ├── finvault.json
      └── work.json
  ```

- JSON formatting: Use `json.MarshalIndent` with 2-space indentation for human-readable state files (debugging aid). The overhead is negligible for state files.

- File locking is NOT required for v1: Harvx is a single-process CLI tool. If we later add watch mode, we would need advisory locking.

- Reference: PRD Section 5.8 specifies `.harvx/state/<profile-name>.json`

## Files to Create/Modify

- `internal/diff/cache.go` - StateCache implementation with read/write/clear methods
- `internal/diff/cache_test.go` - Unit tests with temp directory fixtures
- `internal/diff/errors.go` - Sentinel errors: `ErrBranchMismatch`, `ErrNoState`, `ErrInvalidVersion`

## Testing Requirements

- Unit test: Save a snapshot, then load it back and verify all fields match
- Unit test: Save creates `.harvx/state/` directory if it does not exist
- Unit test: Save overwrites existing state file for the same profile
- Unit test: Load returns `ErrNoState` when no state file exists
- Unit test: Load returns `ErrBranchMismatch` when stored branch differs from requested branch
- Unit test: ClearState removes the state file and returns no error
- Unit test: ClearState on non-existent file returns no error (idempotent)
- Unit test: ClearAllState removes the entire state directory
- Unit test: Profile name sanitization: `"my profile!"` becomes `"my_profile_"`
- Unit test: Atomic write -- if write is interrupted (simulated by not renaming), no partial file remains at the final path
- Unit test: Concurrent reads during write do not see partial state (read the old file or the new one, never a half-written one)
- Unit test: HasState returns true after save, false after clear
- Integration test: Full round-trip -- create state with files, save, load, verify hashes match

## References

- [os.CreateTemp](https://pkg.go.dev/os#CreateTemp) -- atomic write pattern
- [os.Rename](https://pkg.go.dev/os#Rename) -- atomic rename on same filesystem
- [os.MkdirAll](https://pkg.go.dev/os#MkdirAll) -- recursive directory creation
- PRD Section 5.8 (State file path: `.harvx/state/<profile-name>.json`)
- PRD Section 5.8 (State files are gitignored; `.harvx/` in default ignore list)
