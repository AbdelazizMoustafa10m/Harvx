# T-065: Cache Subcommands and Change Summary Rendering

**Priority:** Should Have
**Effort:** Medium (6-12hrs)
**Dependencies:** T-061 (State Cache Persistence), T-062 (State Comparison Engine), T-064 (Diff Subcommand)
**Phase:** 4 - State & Diff

---

## Description

Implement the `harvx cache` subcommands (`cache clear`, `cache show`) and the change summary section that is rendered in the output document when diff mode is active. The cache subcommands provide visibility and control over Harvx's persistent state. The change summary section is a formatted block (in markdown or XML depending on output format) that lists added, modified, and deleted files with counts, appearing in the output document after the file summary section.

## User Story

As a developer, I want to inspect and manage Harvx's cached state so that I can debug stale diffs, clear cache when switching contexts, and understand what state Harvx is tracking. When I run in diff mode, I want a clear summary of what changed at the top of the output.

## Acceptance Criteria

### Cache Subcommands

- [ ] `internal/cli/cache.go` registers the `harvx cache` parent command and its subcommands
- [ ] `harvx cache clear` clears all cached state files (entire `.harvx/state/` directory)
- [ ] `harvx cache clear --profile <name>` clears cached state for a specific profile only
- [ ] `harvx cache clear` prints confirmation: `"Cleared all cached state from .harvx/state/"`
- [ ] `harvx cache clear --profile finvault` prints: `"Cleared cached state for profile 'finvault'"`
- [ ] `harvx cache clear` on non-existent cache prints: `"No cached state found."` (not an error, exit code 0)
- [ ] `harvx cache show` displays a summary of all cached state files with: profile name, last generated timestamp, git branch, HEAD SHA, file count, and file path
- [ ] `harvx cache show` with no cached state prints: `"No cached state found. Run 'harvx generate' to create state."`
- [ ] `harvx cache show --json` outputs the summary as JSON for programmatic consumption
- [ ] All cache subcommands respect the `-d/--dir` flag for targeting a specific directory

### Change Summary Rendering

- [ ] `internal/output/change_summary.go` implements `RenderChangeSummary(result *DiffResult, format string) string`
- [ ] Markdown format renders a section like:
  ```markdown
  ## Changes Since Last Run
  
  **3 added** | **5 modified** | **1 deleted** (42 unchanged)
  
  ### Added Files
  - `src/new-feature.go`
  - `src/helper.go`
  - `tests/new_test.go`
  
  ### Modified Files
  - `src/main.go`
  - `src/config.go`
  - `internal/handler.go`
  - `README.md`
  - `go.mod`
  
  ### Deleted Files
  - `src/deprecated.go`
  ```
- [ ] XML format renders equivalent structure using XML tags:
  ```xml
  <change_summary>
    <counts added="3" modified="5" deleted="1" unchanged="42"/>
    <added_files>
      <file path="src/new-feature.go"/>
      ...
    </added_files>
    ...
  </change_summary>
  ```
- [ ] When there are no changes, the summary reads: `"No changes detected since last run."`
- [ ] The change summary is integrated into the main output renderer (injected between the file summary and directory tree sections)
- [ ] The change summary is only rendered when diff mode is active (`--diff-only` or `harvx diff`)
- [ ] `--clear-cache` flag on root command calls `ClearState` before running the generate pipeline
- [ ] Unit tests achieve 90%+ coverage

### Integration with Pipeline

- [ ] The pipeline orchestrator (`internal/pipeline/pipeline.go`) is updated to:
  1. After file discovery + hashing, build current state snapshot
  2. If diff mode: load cached state, compare, produce DiffResult, filter files
  3. If `--diff-only`: pass only added/modified files to renderer
  4. Pass DiffResult to renderer for change summary section
  5. After successful generation: save current state to cache
- [ ] The `--clear-cache` flag triggers cache clearing before any pipeline steps

## Technical Notes

- **Cobra command structure:**
  ```go
  var cacheCmd = &cobra.Command{
      Use:   "cache",
      Short: "Manage Harvx state cache",
  }
  
  var cacheClearCmd = &cobra.Command{
      Use:   "clear",
      Short: "Clear cached state",
      RunE:  runCacheClear,
  }
  
  var cacheShowCmd = &cobra.Command{
      Use:   "show",
      Short: "Show cached state summary",
      RunE:  runCacheShow,
  }
  
  func init() {
      cacheClearCmd.Flags().StringP("profile", "p", "", "Clear state for a specific profile")
      cacheShowCmd.Flags().Bool("json", false, "Output as JSON")
      cacheCmd.AddCommand(cacheClearCmd, cacheShowCmd)
      rootCmd.AddCommand(cacheCmd)
  }
  ```

- **`cache show` output format (table):**
  ```
  Cached State Summary (.harvx/state/):
  
  PROFILE      GENERATED             BRANCH    HEAD     FILES
  default      2026-02-15 14:30:00   main      a1b2c3d  342
  finvault     2026-02-15 14:25:00   main      a1b2c3d  342
  work         2026-02-14 09:00:00   develop   e5f6g7h  128
  
  Total: 3 profiles cached
  ```

- **`cache show --json` output:**
  ```json
  {
    "cache_dir": ".harvx/state",
    "profiles": [
      {
        "name": "default",
        "generated_at": "2026-02-15T14:30:00Z",
        "git_branch": "main",
        "git_head_sha": "a1b2c3d",
        "file_count": 342,
        "state_file": ".harvx/state/default.json"
      }
    ]
  }
  ```

- **Change summary placement in output:** The change summary appears after the file summary section (counts, tokens, tiers) and before the directory tree. This gives the LLM immediate awareness of what changed before seeing the file contents.

- **Table formatting:** Use `text/tabwriter` for aligned table output in `cache show`. For terminal styling, use `charmbracelet/lipgloss` if it is already available in the project.

- **The `--clear-cache` flag** should be added to the root command (T-064 adds it as a flag; this task implements the handler that calls `ClearAllState` or `ClearState` depending on whether a profile is specified).

## Files to Create/Modify

- `internal/cli/cache.go` - `harvx cache`, `harvx cache clear`, `harvx cache show` subcommands
- `internal/cli/cache_test.go` - CLI tests for cache subcommands
- `internal/output/change_summary.go` - Change summary rendering for markdown and XML formats
- `internal/output/change_summary_test.go` - Unit tests for rendering
- `internal/pipeline/pipeline.go` - Integrate state snapshot creation, comparison, and filtering into the generation pipeline
- `internal/cli/root.go` - Wire `--clear-cache` handler

## Testing Requirements

### Cache Subcommand Tests
- Unit test: `cache clear` removes all state files from `.harvx/state/`
- Unit test: `cache clear --profile finvault` removes only `finvault.json`
- Unit test: `cache clear` on empty/non-existent directory prints friendly message, exits 0
- Unit test: `cache show` lists all cached profiles with correct metadata
- Unit test: `cache show` with no cache prints helpful message
- Unit test: `cache show --json` produces valid JSON matching expected schema
- Unit test: `-d` flag changes the root directory for cache operations

### Change Summary Rendering Tests
- Unit test: Markdown rendering with all three change types (added/modified/deleted)
- Unit test: Markdown rendering with only additions (no modified/deleted sections)
- Unit test: Markdown rendering with no changes produces "No changes detected" message
- Unit test: XML rendering produces valid XML structure
- Unit test: File paths in summary are sorted alphabetically
- Unit test: Counts in summary header match actual file lists
- Golden test: Render a known DiffResult to markdown and compare against fixture

### Pipeline Integration Tests
- Integration test: Run generate twice (first creates state, second loads and compares), verify second run correctly identifies changes
- Integration test: `--clear-cache` flag clears state before generate
- Integration test: `--diff-only` filters output to only changed files

## References

- [spf13/cobra subcommands](https://github.com/spf13/cobra) -- parent + child command pattern
- [text/tabwriter](https://pkg.go.dev/text/tabwriter) -- aligned table output
- PRD Section 5.8 (Change summary section, cache management)
- PRD Section 5.7 (Output structure: change summary section placement)
- PRD Section 5.9 (`harvx cache clear`, `harvx cache show` subcommands)
