# T-085: Search/Filter, Tier Views & Help Overlay

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-080 (file tree model), T-084 (styling)
**Phase:** 5 - Interactive TUI

---

## Description

Implement the search/filter functionality (`/` key), tier view cycling (`t` key), select all/none shortcuts (`a`/`n` keys), and the help overlay (`?` key) for the TUI. Search enables fuzzy file path filtering within the tree, tier views allow filtering the tree to show only files of a specific tier, and the help overlay displays all available keybindings in a formatted reference card.

## User Story

As a developer working in a large repository, I want to search for files by name and filter by tier so that I can quickly find and select specific files without scrolling through hundreds of entries.

## Acceptance Criteria

- [ ] `/` key activates search mode: a text input appears at the bottom of the file tree panel
- [ ] Search performs fuzzy matching on relative file paths (e.g., typing "midl" matches "middleware.ts")
- [ ] Search results filter the visible tree in real-time as the user types
- [ ] Matched characters in file paths are highlighted (bold or underlined)
- [ ] Esc or Enter exits search mode: Enter keeps the filter applied, Esc clears it
- [ ] When filter is active, a "Filter: <query>" indicator shows in the panel header
- [ ] `Ctrl+L` or second press of `/` clears the current filter
- [ ] `t` key cycles tier views: All -> Tier 0 only -> Tier 1 only -> ... -> Tier 5 only -> All
- [ ] When a tier view is active, only files matching that tier are visible in the tree
- [ ] Tier view indicator shows in panel header: `Tier: 0 (priority)` or `Tier: All`
- [ ] `a` key selects all currently visible files (respects search filter and tier view)
- [ ] `n` key deselects all currently visible files (sets to excluded)
- [ ] `?` key toggles a help overlay that covers the center of the screen
- [ ] Help overlay displays all keybindings organized by category:
  - Navigation: arrows, j/k, h/l, PgUp/PgDn, Home/End
  - Selection: Space (toggle), a (all), n (none)
  - Filtering: / (search), t (tier view), Ctrl+L (clear)
  - Profiles: Tab/Shift+Tab (cycle)
  - Actions: Enter (generate), p (preview), s (save profile), e (export), q (quit)
- [ ] Help overlay is dismissible by pressing `?` again or Esc
- [ ] Help overlay is styled with a bordered box, centered on screen
- [ ] Fuzzy matching uses a simple substring or Levenshtein-like algorithm (no external dependency needed)
- [ ] Unit tests for search filtering, tier cycling, select all/none, and help overlay toggle

## Technical Notes

- For search text input, use `charmbracelet/bubbles/textinput` component. Configure it with `Placeholder: "Search files..."`, `Prompt: "/"`, and `CharLimit: 100`.
- Fuzzy matching options: (1) simple case-insensitive substring match (fast, easy), (2) `sahilm/fuzzy` package for ranked fuzzy matching. Recommend starting with simple substring and upgrading to fuzzy if needed.
- Search filter is applied as a predicate on the visible node list. The underlying tree data is not modified -- only the view changes.
- Tier view filter is also a predicate. When both search and tier filters are active, they compose (intersection).
- Select all/none only affects currently visible nodes (after filters). This allows workflows like: filter to tier 0, select all, clear filter, then toggle individual files.
- Help overlay: render as a lipgloss-styled box with `lipgloss.Place()` centered over the main layout. Use `lipgloss.JoinVertical()` to stack category sections.
- The help text should be generated from the key bindings defined in `keys.go` (T-079) to stay in sync.
- Reference: PRD Section 5.13 (keyboard shortcuts: / search, a select all, n select none, t cycle tier views, ? help overlay)

## Files to Create/Modify

- `internal/tui/search/model.go` - Search component with text input and filter logic
- `internal/tui/search/fuzzy.go` - Fuzzy/substring matching implementation
- `internal/tui/search/view.go` - Search input and match highlight rendering
- `internal/tui/filetree/filter.go` - Filter predicates (search + tier view) applied to visible nodes
- `internal/tui/help/model.go` - Help overlay model
- `internal/tui/help/view.go` - Help overlay rendering with categorized keybindings
- `internal/tui/app.go` - Updated root model to handle `/`, `t`, `a`, `n`, `?` keys
- `internal/tui/search/model_test.go` - Search logic tests
- `internal/tui/search/fuzzy_test.go` - Fuzzy matching tests
- `internal/tui/filetree/filter_test.go` - Filter predicate tests
- `internal/tui/help/model_test.go` - Help overlay tests

## Testing Requirements

- Unit test: search "midl" matches "middleware.ts" and "src/middleware/auth.ts"
- Unit test: search is case-insensitive
- Unit test: search filter reduces visible node count correctly
- Unit test: Esc in search mode clears filter and restores full tree
- Unit test: Enter in search mode keeps filter active
- Unit test: tier view `t` cycles through All -> 0 -> 1 -> 2 -> 3 -> 4 -> 5 -> All
- Unit test: tier view filters tree to only matching tier
- Unit test: `a` selects all visible nodes (after search filter)
- Unit test: `n` deselects all visible nodes
- Unit test: combined search + tier filter works (intersection)
- Unit test: `?` toggles help overlay visibility
- Unit test: help overlay contains all expected key categories
- Unit test: help overlay Esc dismisses it