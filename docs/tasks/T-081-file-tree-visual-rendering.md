# T-081: File Tree Visual Rendering & Tier Color Coding

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-080 (file tree data model), T-084 (lipgloss styles -- can develop in parallel if style constants are agreed upon)
**Phase:** 5 - Interactive TUI

---

## Description

Implement the visual rendering layer for the file tree component, including Unicode tree-drawing characters, inclusion/exclusion indicators, color-coded tier assignments, priority file and secret-containing file highlights, and directory expand/collapse markers. This makes the file tree visually informative and aligns with the PRD's "clean, modern terminal aesthetic" requirement.

## User Story

As a developer using the TUI, I want to see at a glance which files are included, what tier they belong to, and which contain secrets so that I can make informed decisions about my context selection.

## Acceptance Criteria

- [ ] Each node renders with proper Unicode tree-drawing characters: `├──`, `└──`, `│   ` for indentation based on depth and sibling position
- [ ] Inclusion indicators display next to each node: `[✓]` included (green), `[✗]` excluded (dim/gray), `[◐]` partial (yellow)
- [ ] Directories show expand/collapse indicators: `▸` (collapsed) / `▾` (expanded) before the name
- [ ] Files show a file icon or type indicator
- [ ] Tier color coding applied to file names: Tier 0 = gold/yellow, Tier 1 = green, Tier 2 = blue, Tier 3 = cyan, Tier 4 = magenta, Tier 5 = dim gray
- [ ] Priority files (tier 0 / `priority_files` in profile) display a star icon `★` suffix
- [ ] Files containing detected secrets display a shield icon `🛡` or lock icon suffix in red
- [ ] Currently selected node (cursor row) has a highlighted background
- [ ] Directory names are bold, file names are regular weight
- [ ] Token count displays right-aligned for each file: `(1,234 tok)` in dim text
- [ ] Nodes beyond the viewport are not rendered (virtual scrolling for performance)
- [ ] Tree renders correctly with terminals of various widths (truncates long paths with ellipsis)
- [ ] High-contrast colors work on both light and dark terminal backgrounds
- [ ] Unit tests verify rendering output for various tree states

## Technical Notes

- Use `charmbracelet/lipgloss` v1.x for all styling. Define style constants in `internal/tui/styles.go` (shared with T-084).
- Tier colors should use ANSI 256-color palette for broad terminal compatibility, with TrueColor fallback: Gold = `lipgloss.Color("220")`, Green = `lipgloss.Color("34")`, Blue = `lipgloss.Color("33")`, Cyan = `lipgloss.Color("36")`, Magenta = `lipgloss.Color("133")`, Dim = `lipgloss.Color("240")`.
- Use `lipgloss.HasDarkBackground()` (lipgloss v0.10+) to detect terminal theme and adjust colors accordingly.
- Virtual scrolling: only render nodes from `scrollOffset` to `scrollOffset + viewportHeight`. This is critical for repos with thousands of visible nodes.
- Tree-drawing character generation: maintain a stack of "is last child" booleans as you walk the visible node list. Each level contributes either `│   ` (not last) or `    ` (last sibling) to the prefix.
- Right-align token counts using `lipgloss.Width()` to measure available space, then `lipgloss.PlaceHorizontal()` or manual padding.
- Reference: PRD Section 5.13 (visual indicators, color-coded tiers, highlights)

## Files to Create/Modify

- `internal/tui/filetree/view.go` - Complete rendering implementation (replaces placeholder from T-080)
- `internal/tui/filetree/icons.go` - Icon and indicator constants
- `internal/tui/filetree/view_test.go` - Rendering snapshot tests
- `internal/tui/styles.go` - Shared style definitions (created here, extended in T-084)

## Testing Requirements

- Snapshot test: tree with mixed included/excluded/partial nodes renders correct indicators
- Snapshot test: tier colors applied correctly to node names
- Snapshot test: priority file shows star icon
- Snapshot test: secret-containing file shows shield icon
- Unit test: tree-drawing prefix generation for nested directories (correct `├──`, `└──`, `│` placement)
- Unit test: long file paths truncated with ellipsis when terminal is narrow
- Unit test: virtual scrolling renders only visible range
- Unit test: cursor highlight applied to correct row
- Unit test: token count right-aligned correctly