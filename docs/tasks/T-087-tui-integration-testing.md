# T-087: TUI Integration Testing & Pipeline Wiring

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-079 through T-086 (all TUI components)
**Phase:** 5 - Interactive TUI

---

## Description

Wire all TUI components together into the final integrated application, verify end-to-end flows (launch -> navigate -> select -> generate, launch -> select -> save profile, launch -> search -> select -> export), and create integration tests that exercise the full TUI lifecycle. This task is the "glue" that ensures all independently developed TUI components work together correctly and the TUI-to-pipeline handoff produces correct output.

## User Story

As a developer, I want the TUI to work end-to-end -- from launch to output generation -- without glitches so that I can trust the interactive mode as much as the headless mode.

## Acceptance Criteria

- [ ] Root model correctly composes all sub-models: file tree (T-080), stats panel (T-082), profile selector (T-083), search (T-085), help overlay (T-085), layout (T-084)
- [ ] Full flow: launch TUI -> navigate tree -> toggle files -> verify stats update -> press Enter -> pipeline runs -> output file generated -> TUI exits
- [ ] Full flow: launch TUI -> toggle files -> press `s` -> type profile name -> Enter -> harvx.toml written with new profile
- [ ] Full flow: launch TUI -> press `/` -> type search query -> results filter -> toggle visible -> press `e` -> paths copied to clipboard
- [ ] Full flow: launch TUI -> press Tab -> profile switches -> tree re-evaluates tiers -> stats recalculate
- [ ] File toggle in TUI produces identical file set as equivalent headless `--include`/`--exclude` flags
- [ ] Token counts in TUI match token counts from headless `harvx preview`
- [ ] TUI exits cleanly on Ctrl+C without leaving terminal in bad state (alt screen properly exited)
- [ ] TUI renders correctly with `go test -run TestTUI -v` using `teatest` helper
- [ ] No goroutine leaks after TUI exits (verify with `runtime.NumGoroutine()` before/after)
- [ ] Error conditions handled gracefully: missing files, permission errors, invalid profile
- [ ] Integration test suite with at least 8 scenarios
- [ ] `go build ./cmd/harvx/ && ./harvx -i` works against `testdata/sample-repo/`

## Technical Notes

- Use `charmbracelet/x/exp/teatest` for programmatic TUI testing. This allows sending key sequences and asserting on rendered output without a real terminal.
- `teatest.NewModel()` creates a test model, `teatest.SendMsg()` sends messages, and you can assert on the `View()` output.
- For integration tests that verify pipeline output, use `testdata/sample-repo/` as the target directory. The sample repo should have a known set of files with predictable token counts.
- Goroutine leak detection: capture `runtime.NumGoroutine()` before creating the program and after `p.Run()` returns. Assert they're within a small delta (allow for GC goroutines).
- Terminal state verification: after `tea.Program` exits, verify stdout is not in alt screen mode. This is hard to test programmatically but can be verified by checking the program exited via `tea.Quit` rather than a panic.
- For clipboard integration test, mock the clipboard interface or skip on CI where clipboard is unavailable.
- Reference: PRD Section 5.13 (TUI is a presentation layer only, calls same core pipeline)

## Files to Create/Modify

- `internal/tui/app.go` - Final integration of all sub-models (modify)
- `internal/tui/integration_test.go` - Integration test suite
- `internal/tui/teatest_helpers_test.go` - Test helpers for TUI testing
- `internal/cli/interactive.go` - Final TUI launch logic with error handling (modify)
- `internal/cli/interactive_test.go` - CLI integration tests for `-i` flag

## Testing Requirements

- Integration test: launch with sample-repo, toggle 3 files, verify stats show correct token count
- Integration test: launch, navigate to nested directory, expand it, toggle all children
- Integration test: launch, press Tab to switch profile, verify tier colors change
- Integration test: launch, press `/`, type "README", verify only matching files visible
- Integration test: launch, press `t` twice, verify tier 1 view active
- Integration test: launch, press `a`, verify all files selected, press `n`, verify all deselected
- Integration test: launch, press `?`, verify help overlay visible, press `?` again, verify dismissed
- Integration test: press Enter, verify pipeline runs and output file exists
- Integration test: verify Ctrl+C exits cleanly
- Regression test: TUI file selection matches headless output for same include patterns