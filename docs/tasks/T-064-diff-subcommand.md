# T-064: `harvx diff` Subcommand and `--diff-only` Flag

**Priority:** Should Have
**Effort:** Medium (6-12hrs)
**Dependencies:** T-062 (State Comparison Engine), T-063 (Git-Aware Diffing)
**Phase:** 4 - State & Diff

---

## Description

Implement the `harvx diff` subcommand and the `--diff-only` flag on the root `harvx` / `harvx generate` command. The `diff` subcommand is a dedicated command for generating differential output between the current project state and either a cached state or a git ref. The `--diff-only` flag on the root command modifies standard generation to output only changed files. Both paths converge on the same comparison engine (T-062) and git layer (T-063) but present different UX surfaces.

## User Story

As a developer running iterative code reviews, I want to run `harvx diff --since HEAD~1` so that I get a focused context document containing only the files that changed in the last commit, ready to feed to my AI review agents.

## Acceptance Criteria

- [ ] `internal/cli/diff.go` registers the `harvx diff` subcommand with Cobra
- [ ] `harvx diff` (no flags) compares current state against the cached state for the active profile and outputs changed files
- [ ] `harvx diff --since <ref>` compares current state against the state at the given git ref
- [ ] `harvx diff --since HEAD~1` works (common use case: last commit)
- [ ] `harvx diff --since <sha>` works with a full or short commit SHA
- [ ] `harvx diff --base <ref> --head <ref>` compares two git refs (for PR reviews)
- [ ] `--base` and `--head` flags must be used together; using only one produces a clear error
- [ ] `--since` and `--base/--head` are mutually exclusive; using both produces a clear error
- [ ] When no cached state exists and no git flags are provided, `harvx diff` prints a helpful message: "No cached state found. Run `harvx generate` first, or use `--since <ref>` for git-based diffing."
- [ ] The `--diff-only` flag is added to the root command and `harvx generate` command
- [ ] `harvx generate --diff-only` outputs only changed files (full content for added/modified, deletion markers for deleted)
- [ ] `harvx --diff-only` is equivalent to `harvx generate --diff-only`
- [ ] The diff output includes a change summary header listing added/modified/deleted counts and file paths (rendered by T-065)
- [ ] The diff output format follows the same rendering pipeline as normal output (markdown/XML based on profile) but includes only changed files
- [ ] After a successful `harvx diff`, the current state is NOT saved to cache (diff is a read-only operation)
- [ ] After a successful `harvx generate --diff-only`, the current state IS saved to cache (generate always updates cache)
- [ ] `--profile` flag works with diff to select the correct cached state
- [ ] Exit code 0 when diff succeeds, exit code 1 on error
- [ ] All flags have help text accessible via `harvx diff --help`

## Technical Notes

- **Cobra subcommand registration:**
  ```go
  var diffCmd = &cobra.Command{
      Use:   "diff",
      Short: "Generate differential output showing changes since last run or a git ref",
      Long:  `Compare the current project state against a previous state and output only
  what changed. Supports both cached-state diffing and git-ref diffing.`,
      RunE: runDiff,
  }
  
  func init() {
      diffCmd.Flags().String("since", "", "Git ref to diff against (e.g., HEAD~1, a commit SHA)")
      diffCmd.Flags().String("base", "", "Base git ref for PR review diffing")
      diffCmd.Flags().String("head", "", "Head git ref for PR review diffing")
      rootCmd.AddCommand(diffCmd)
  }
  ```

- **Mode determination logic:**
  ```
  if --since is set:
      use git-based diffing (GetChangedFilesSince)
  else if --base and --head are set:
      use git-based diffing (GetChangedFiles between base and head)
  else:
      use cache-based diffing (CompareStates with loaded cache)
  ```

- **Integration with the generate pipeline:** The `--diff-only` flag does NOT change the file discovery or hashing pipeline. Instead, it adds a filtering step after state comparison: only files in `DiffResult.Added` or `DiffResult.Modified` are passed to the output renderer. Deleted files are listed in the change summary but have no content in the output.

- **Unified diff generation:** For `harvx diff`, optionally generate unified diff format for modified files using `sergi/go-diff` (`diffmatchpatch.DiffMain`). This is a line-by-line diff showing exactly what changed within each file. This is useful for PR review context. Enable via `--unified` flag (default: false; default behavior is to show full content of changed files).
  - Note: `sergi/go-diff` v1.2.0 provides `DiffMain` for character-level diffs. For line-level unified diffs, use `DiffLinesToRunes` + `DiffMainRunes` + `DiffCharsToLines` pipeline, then format with `DiffPrettyText` or a custom unified diff formatter.
  - Add `github.com/sergi/go-diff v1.2.0` to `go.mod`.

- **State save behavior:**
  - `harvx diff` = read-only, no state save (useful for previewing changes)
  - `harvx generate --diff-only` = saves state after generation (standard generate behavior)
  - `harvx generate` (without --diff-only) = saves state after generation (standard behavior)

- **Wire into existing generate pipeline:** The `--diff-only` flag should be read by the pipeline orchestrator (`internal/pipeline/pipeline.go`). After file discovery + hashing, load cached state, compare, and filter the file list before passing to the renderer.

## Files to Create/Modify

- `internal/cli/diff.go` - `harvx diff` subcommand definition and handler
- `internal/cli/root.go` - Add `--diff-only` and `--clear-cache` flags to root/generate commands
- `internal/diff/diff.go` - High-level diff orchestration: mode selection, pipeline integration, unified diff formatting
- `internal/diff/diff_test.go` - Unit tests for orchestration logic
- `internal/cli/diff_test.go` - CLI integration tests (flag parsing, error messages)
- `go.mod` - Add `github.com/sergi/go-diff v1.2.0`

## Testing Requirements

- Unit test: `harvx diff` with no cache returns helpful error message (not a stack trace)
- Unit test: `harvx diff --since HEAD~1` correctly calls git layer and returns diff result
- Unit test: `harvx diff --base main --head feature` calls git layer with correct refs
- Unit test: `--base` without `--head` produces error
- Unit test: `--since` with `--base` produces mutual exclusion error
- Unit test: `--diff-only` flag is recognized on root command
- Unit test: Mode determination logic selects correct diffing strategy
- Unit test: Diff output includes change summary header
- Unit test: Modified files show full content (not just diff hunks) by default
- Unit test: `--unified` flag generates line-level diff output for modified files
- Unit test: After `harvx diff`, no state file is written
- Unit test: After `harvx generate --diff-only`, state file is written
- Integration test: Create test repo, run generate (saves state), modify files, run `harvx diff` and verify correct output

## References

- [spf13/cobra](https://github.com/spf13/cobra) -- CLI framework for subcommands
- [sergi/go-diff v1.2.0](https://github.com/sergi/go-diff/releases/tag/v1.2.0) -- text diffing library
- [sergi/go-diff diffmatchpatch](https://pkg.go.dev/github.com/sergi/go-diff/diffmatchpatch) -- API documentation
- PRD Section 5.8 (harvx diff subcommand, --diff-only flag, --since, --base/--head)
- PRD Section 5.9 (CLI subcommands: `harvx diff`)
