# T-063: Git-Aware Diffing

**Priority:** Should Have
**Effort:** Medium (6-12hrs)
**Dependencies:** T-062 (State Comparison Engine)
**Phase:** 4 - State & Diff

---

## Description

Implement git-aware diffing that allows Harvx to generate diffs based on git history rather than cached filesystem state. This supports three modes: `--since <ref>` (diff against a specific git ref like `HEAD~1` or a SHA), `--base <ref> --head <ref>` (diff between two arbitrary refs, designed for PR reviews), and automatic detection of changed files using `git diff --name-status`. This layer shells out to the `git` CLI for ref resolution and file listing, then delegates to the state comparison engine for producing the `DiffResult`.

## User Story

As a developer reviewing a pull request, I want to run `harvx diff --base main --head feature-branch` so that I can generate context containing only the files that changed in the PR, along with their full content for the AI reviewer.

## Acceptance Criteria

- [ ] `internal/diff/git.go` implements the `GitDiffer` struct with methods for git-based diffing
- [ ] `GetCurrentBranch(rootDir string) (string, error)` returns the current git branch name (or empty string if detached HEAD)
- [ ] `GetHeadSHA(rootDir string) (string, error)` returns the current HEAD commit SHA (short, 7 chars)
- [ ] `GetChangedFiles(rootDir, baseRef, headRef string) ([]GitFileChange, error)` returns the list of changed files between two refs with their change type (added/modified/deleted/renamed)
- [ ] `GetChangedFilesSince(rootDir, sinceRef string) ([]GitFileChange, error)` returns files changed since a ref (equivalent to `git diff --name-status <ref>..HEAD`)
- [ ] `GitFileChange` struct: `Path string`, `OldPath string` (for renames), `Status GitChangeType` (Added/Modified/Deleted/Renamed)
- [ ] `BuildDiffResultFromGit(changes []GitFileChange) *DiffResult` converts git changes into the standard `DiffResult` format
- [ ] Handles renamed files: treated as a delete of old path + add of new path
- [ ] Handles the case where `git` is not installed: returns a clear `ErrGitNotFound` error
- [ ] Handles the case where the directory is not a git repository: returns `ErrNotGitRepo`
- [ ] Handles invalid refs: returns `ErrInvalidRef` with the ref string in the error message
- [ ] All git commands use `--no-pager` to prevent interactive output
- [ ] All git commands respect the `rootDir` parameter via `exec.Cmd.Dir`
- [ ] Git commands use `context.Context` for cancellation support
- [ ] Unit tests achieve 90%+ coverage (using a test git repo created in `t.TempDir()`)

## Technical Notes

- **Shelling out to git:** Use `os/exec` to call git commands. This is intentional -- the `git` CLI is nearly universal on developer machines and CI environments, and it handles all edge cases (submodules, worktrees, partial clones) that Go git libraries struggle with. The PRD's zero-dependency goal refers to Go runtime dependencies, not host tools.

- **Key git commands:**
  ```bash
  # Current branch
  git -C <rootDir> rev-parse --abbrev-ref HEAD
  
  # HEAD SHA (short)
  git -C <rootDir> rev-parse --short HEAD
  
  # Changed files between refs (for --base/--head)
  git -C <rootDir> diff --name-status <base>..<head>
  
  # Changed files since ref (for --since)
  git -C <rootDir> diff --name-status <ref>..HEAD
  
  # Verify ref exists
  git -C <rootDir> rev-parse --verify <ref>
  ```

- **Parsing `--name-status` output:** Each line is `<status>\t<path>` or `<status>\t<old-path>\t<new-path>` (for renames). Status codes:
  - `A` = Added
  - `M` = Modified  
  - `D` = Deleted
  - `R<score>` = Renamed (e.g., `R100`)
  - `C<score>` = Copied (treat as Added)

- **Error handling pattern:**
  ```go
  func runGit(ctx context.Context, dir string, args ...string) (string, error) {
      cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir, "--no-pager"}, args...)...)
      var stdout, stderr bytes.Buffer
      cmd.Stdout = &stdout
      cmd.Stderr = &stderr
      err := cmd.Run()
      if err != nil {
          return "", fmt.Errorf("git %s: %w: %s", args[0], err, stderr.String())
      }
      return strings.TrimSpace(stdout.String()), nil
  }
  ```

- **Ref validation:** Before running a diff, validate that the ref exists using `git rev-parse --verify <ref>`. This provides a clear error instead of a cryptic git error message.

- **Context threading:** All git operations accept `context.Context` as their first parameter for cancellation via Ctrl+C (PRD Section 6.4).

- **Not using go-git:** The PRD specifies `os/exec` + git CLI rather than a Go git library. This avoids adding a large dependency (go-git is ~50 packages) and ensures compatibility with all git features.

## Files to Create/Modify

- `internal/diff/git.go` - GitDiffer implementation with git CLI interaction
- `internal/diff/git_test.go` - Unit tests using real git repos created in temp dirs
- `internal/diff/errors.go` - Add `ErrGitNotFound`, `ErrNotGitRepo`, `ErrInvalidRef` (extend file from T-061)

## Testing Requirements

- Unit test: `GetCurrentBranch` returns correct branch name from a test git repo
- Unit test: `GetCurrentBranch` returns empty string for detached HEAD
- Unit test: `GetHeadSHA` returns a 7-character hex string
- Unit test: `GetChangedFiles` between two commits correctly identifies added, modified, deleted files
- Unit test: `GetChangedFiles` handles renamed files (maps to delete + add)
- Unit test: `GetChangedFilesSince` with `HEAD~1` correctly identifies the last commit's changes
- Unit test: `GetChangedFilesSince` with a specific SHA works correctly
- Unit test: Invalid ref returns `ErrInvalidRef`
- Unit test: Non-git directory returns `ErrNotGitRepo`
- Unit test: `BuildDiffResultFromGit` correctly maps `GitFileChange` slice to `DiffResult`
- Unit test: Empty diff (no changes between refs) returns `DiffResult` with `HasChanges() == false`
- Unit test: Context cancellation stops a running git command
- Integration test: Create a test repo with multiple commits, verify diff between first and last commit

**Test setup helper:**
```go
func setupTestRepo(t *testing.T) string {
    t.Helper()
    dir := t.TempDir()
    runCmd(t, dir, "git", "init")
    runCmd(t, dir, "git", "config", "user.email", "test@test.com")
    runCmd(t, dir, "git", "config", "user.name", "Test")
    // Create initial commit
    writeFile(t, filepath.Join(dir, "file1.go"), "package main")
    runCmd(t, dir, "git", "add", ".")
    runCmd(t, dir, "git", "commit", "-m", "initial")
    return dir
}
```

## References

- [os/exec package](https://pkg.go.dev/os/exec) -- running external commands
- [git-diff documentation](https://git-scm.com/docs/git-diff) -- `--name-status` output format
- [git-rev-parse documentation](https://git-scm.com/docs/git-rev-parse) -- ref resolution
- PRD Section 5.8 (Git-aware diffing: `--since`, `--base/--head`)
- PRD Section 6.4 (context.Context threading for cancellation)
