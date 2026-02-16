# T-082: Stats Panel with Live Token Counting & Budget Bar

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-079 (TUI scaffold), T-080 (file tree model for selection data)
**Phase:** 5 - Interactive TUI

---

## Description

Build the stats sidebar panel that displays real-time statistics about the current file selection. This includes a live token count with visual budget bar, file count, estimated output size, compression savings (when enabled), redaction count, and tier breakdown. The panel updates reactively whenever files are toggled in the file tree, using debounced token recalculation to maintain UI responsiveness.

## User Story

As a developer selecting files in the TUI, I want to see a live token count and budget utilization bar so that I know exactly how much of my LLM's context window I'm using before generating output.

## Acceptance Criteria

- [ ] `internal/tui/stats/model.go` defines `Model` implementing `tea.Model`
- [ ] Displays current token count vs budget: `Tokens: 89,420 / 200,000` with thousands separator
- [ ] Visual budget utilization bar using block characters: `[████████████░░░░░░░░] 45%`
- [ ] Budget bar color changes with utilization: green (<70%), yellow (70-90%), red (>90%)
- [ ] File count display: `Files: 342 / 390 selected`
- [ ] Estimated output size: `Size: ~2.4 MB`
- [ ] Compression savings (when `compression = true` in profile): `Compressed: 52% reduction`
- [ ] Redaction count: `Secrets: 3 found` (red if > 0)
- [ ] Tier breakdown table showing file count per tier:
  ```
  Tier 0 (priority)  5 files    12,400 tok
  Tier 1 (core)     48 files    34,200 tok
  Tier 2 (support) 180 files    28,800 tok
  ...
  ```
- [ ] Profile info header: `Profile: finvault | Target: claude`
- [ ] Tokenizer info: `Tokenizer: o200k_base`
- [ ] Token recalculation is debounced at 200ms after the last file toggle to prevent UI jank
- [ ] While recalculating, show a spinner or "calculating..." indicator
- [ ] Panel width is fixed (configurable, default ~35 chars) and content wraps or truncates to fit
- [ ] Stats update when receiving `FileToggledMsg` or `ProfileChangedMsg` from root model
- [ ] Unit tests for all stat calculations and display formatting

## Technical Notes

- Debouncing pattern in Bubble Tea: on receiving `FileToggledMsg`, return a `tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return recalcTickMsg{} })` command. Store a generation counter -- if another toggle arrives before the tick, increment the counter. On tick, only recalculate if the counter matches.
- Token counting should run as a `tea.Cmd` (async) that iterates included files and sums their token counts. Use the same `tokenizer.Tokenizer` interface as headless mode.
- For large repos, token counts per file should be cached in the `Node` struct (computed once on lazy load, updated on toggle).
- Budget bar rendering: use `charmbracelet/bubbles/progress` component or custom implementation with lipgloss block characters (`█`, `░`).
- Thousands separator: use `golang.org/x/text/message` or simple custom formatter for `89,420` style formatting.
- Estimated output size: rough estimate from `tokenCount * 4` bytes (average token = 4 chars), then format as KB/MB.
- Reference: PRD Section 5.13 (Stats panel), Section 4 (TUI responsiveness SLO: token recount < 300ms)

## Files to Create/Modify

- `internal/tui/stats/model.go` - Stats panel model with Init/Update/View
- `internal/tui/stats/view.go` - Stats panel rendering (budget bar, tier table, counts)
- `internal/tui/stats/calculate.go` - Token recalculation command and debounce logic
- `internal/tui/stats/format.go` - Number formatting helpers (thousands separator, size formatting)
- `internal/tui/stats/model_test.go` - Unit tests for update logic and debouncing
- `internal/tui/stats/calculate_test.go` - Unit tests for token calculation
- `internal/tui/stats/format_test.go` - Unit tests for formatting

## Testing Requirements

- Unit test: token count formatted with thousands separator (89420 -> "89,420")
- Unit test: budget bar renders correct fill percentage
- Unit test: budget bar color transitions at 70% and 90% thresholds
- Unit test: file count updates when `FileToggledMsg` received
- Unit test: debounce logic -- rapid toggles only trigger one recalculation
- Unit test: debounce generation counter prevents stale recalculations
- Unit test: tier breakdown sums tokens correctly per tier
- Unit test: compression savings percentage calculated correctly
- Unit test: size estimation from token count
- Unit test: panel content truncates gracefully when terminal is narrow