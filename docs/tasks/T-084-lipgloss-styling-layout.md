# T-084: Lipgloss Styling, Responsive Layout & Theme Support

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-079 (TUI scaffold), T-081 (file tree view), T-082 (stats panel view)
**Phase:** 5 - Interactive TUI

---

## Description

Implement the complete visual design system for the TUI using Charmbracelet Lipgloss, including the responsive multi-panel layout that adapts to terminal size, the color scheme that works on both light and dark terminals, border styles, and the overall "clean, modern terminal aesthetic" specified in the PRD. This task brings together all visual components into a polished, cohesive layout and ensures the TUI degrades gracefully on small terminals.

## User Story

As a developer using Harvx in different terminal environments, I want the TUI to look polished and readable regardless of my terminal size or color scheme so that the tool feels professional and is pleasant to use.

## Acceptance Criteria

- [ ] `internal/tui/styles.go` defines the complete style system as a `Styles` struct with computed lipgloss styles
- [ ] `NewStyles(isDark bool, width int, height int)` constructor creates styles adapted to terminal theme and size
- [ ] Multi-panel layout: file tree (left, ~65% width) and stats panel (right, ~35% width) with a vertical border separator
- [ ] Bottom status bar shows: current action hint, profile name, and key shortcuts summary
- [ ] Top title bar shows: `Harvx` with version, and the target directory path
- [ ] Layout adapts to terminal resize events:
  - Width >= 100: full two-panel layout with borders
  - Width 60-99: compressed two-panel with narrower stats
  - Width < 60: single-panel mode (file tree only, stats accessible via toggle key)
- [ ] Minimum terminal size: 40x12. If terminal is smaller, display a "terminal too small" message
- [ ] Color scheme auto-detects light/dark terminal using `lipgloss.HasDarkBackground()`
- [ ] Dark theme: dark background colors, bright foreground, muted borders
- [ ] Light theme: light background colors, dark foreground, visible borders
- [ ] All borders use rounded Unicode box-drawing characters (`╭`, `╮`, `╰`, `╯`, `─`, `│`)
- [ ] Panel titles rendered inline with top border: `╭─ Files ─────────────╮`
- [ ] Status bar uses inverse colors for visibility
- [ ] Focus indicator: the active panel (file tree or stats) has a brighter border
- [ ] Padding and margins are consistent (1 char horizontal padding inside panels)
- [ ] `lipgloss.Place()` used for centering content within panels where appropriate
- [ ] Smooth visual transitions are not possible in terminal (no animation), but state changes should feel immediate
- [ ] Unit tests verify layout calculations for various terminal sizes

## Technical Notes

- Use `charmbracelet/lipgloss` v1.x (latest v1.0+). Lipgloss v2 is in beta as of February 2026; stay on v1 for stability.
- Lipgloss provides `lipgloss.JoinHorizontal()` and `lipgloss.JoinVertical()` for composing panels.
- Use `lipgloss.NewStyle().Border(lipgloss.RoundedBorder())` for panel borders.
- Terminal theme detection: `lipgloss.HasDarkBackground()` queries the terminal for its background color. Not all terminals support this; default to dark theme if detection fails.
- Responsive layout: compute panel widths from `tea.WindowSizeMsg` dimensions. Store dimensions in root model and recalculate styles on resize.
- For the title bar inline with border, manually construct the top border string: `"╭─ " + title + " " + strings.Repeat("─", remaining) + "╮"`.
- Status bar key hint format: `q quit | ? help | Enter generate | Tab profile | Space toggle`
- Consider using `lipgloss.Width()` for measuring rendered string widths (accounts for wide characters and ANSI escapes).
- Reference: PRD Section 5.13 (Visual design), Section 8.1 (CLI design philosophy)

## Files to Create/Modify

- `internal/tui/styles.go` - Complete style system (extend from T-081 foundation)
- `internal/tui/layout.go` - Responsive layout composition (panel widths, join operations)
- `internal/tui/statusbar.go` - Status bar component with key hints
- `internal/tui/titlebar.go` - Title bar component with bordered title
- `internal/tui/app.go` - Updated root `View()` to use layout system
- `internal/tui/layout_test.go` - Layout calculation tests for various sizes
- `internal/tui/styles_test.go` - Style construction tests

## Testing Requirements

- Unit test: layout at width 120 produces two panels with correct proportions
- Unit test: layout at width 80 produces compressed two-panel layout
- Unit test: layout at width 50 produces single-panel mode
- Unit test: layout at width 30 produces "terminal too small" message
- Unit test: dark theme constructor produces bright foreground styles
- Unit test: light theme constructor produces dark foreground styles
- Unit test: panel border renders with correct Unicode characters
- Unit test: status bar contains all expected key hints
- Unit test: title bar shows version and directory path
- Unit test: resize event recalculates panel widths correctly
- Snapshot test: full layout render at 120x40 terminal size