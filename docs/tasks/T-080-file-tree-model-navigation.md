# T-080: File Tree Data Model & Keyboard Navigation

**Priority:** Must Have
**Effort:** Large (14-20hrs)
**Dependencies:** T-079 (TUI scaffold must exist)
**Phase:** 5 - Interactive TUI

---

## Description

Build the interactive file tree component that forms the primary panel of the TUI. This includes the tree data model (nodes representing files and directories with inclusion/exclusion state), keyboard navigation (arrow keys, j/k, space to toggle, enter to expand/collapse directories), and lazy directory loading for large repositories. The file tree is the core interaction surface where users select which files to include in context generation.

## User Story

As a developer exploring a codebase in the TUI, I want to navigate a file tree with arrow keys and toggle file inclusion with spacebar so that I can precisely control which files go into my context output.

## Acceptance Criteria

- [ ] `internal/tui/filetree/model.go` defines `Model` implementing `tea.Model` with a tree of `Node` structs
- [ ] `Node` struct contains: `Path` (relative), `Name`, `IsDir`, `Included` (tri-state: included, excluded, partial), `Expanded` (for dirs), `Tier` int, `HasSecrets` bool, `IsPriority` bool, `TokenCount` int, `Children` []*Node, `Parent` *Node
- [ ] `InclusionState` type with constants: `Included`, `Excluded`, `Partial`
- [ ] Arrow Up / `k` moves cursor up one visible node
- [ ] Arrow Down / `j` moves cursor down one visible node
- [ ] Arrow Right / `l` expands a collapsed directory; on a file, no-op
- [ ] Arrow Left / `h` collapses an expanded directory; on a file, jumps to parent directory
- [ ] Space toggles inclusion state for files and directories (toggling a directory toggles all children recursively)
- [ ] When a directory has mixed included/excluded children, its state shows as `Partial` (◐)
- [ ] Enter on a directory expands/collapses it
- [ ] Page Up / Page Down scroll by viewport height
- [ ] Home / End jump to first / last visible node
- [ ] Visible node list is computed from the tree by filtering only expanded branches
- [ ] Cursor position clamps within visible bounds
- [ ] Viewport scrolls to keep cursor visible (scroll offset management)
- [ ] Lazy directory loading: directories are populated on first expand, not at startup. Uses `discovery.Walker` to scan contents on demand.
- [ ] Root directory nodes are pre-loaded (top-level entries only) during `Init()`
- [ ] Toggling a node sends a `FileToggledMsg` to parent model (for stats recalculation)
- [ ] Tree construction respects existing ignore patterns (`.gitignore`, `.harvxignore`, default ignores) -- ignored files never appear in the tree
- [ ] Binary files are excluded from the tree
- [ ] Unit tests for tree construction, navigation, toggling, viewport scrolling, and lazy loading

## Technical Notes

- There is no official tree component in `charmbracelet/bubbles` as of February 2026 (PR #639 is still unmerged). Build a custom implementation.
- Third-party `mariusor/bubbles-tree` exists but is not well-maintained. Build custom for full control over tri-state toggling, lazy loading, and tier-aware rendering.
- Tri-state propagation: when toggling a directory, set all descendants to the same state. Then walk up from the toggled node to root, recalculating each ancestor's state based on children (all included = included, all excluded = excluded, mixed = partial).
- Lazy loading pattern: each `Node` has a `loaded` bool. On first expand, fire a `tea.Cmd` that scans the directory and returns a `DirLoadedMsg` with the children. Show a spinner or "loading..." indicator while scanning.
- For large repos (10K+ files), limit initial tree depth to 2 levels and load deeper levels on demand.
- Use `filepath.WalkDir` (same as headless discovery) but scoped to the specific directory being expanded.
- Reference: PRD Section 5.13 (File tree panel), Section 6.2 (`internal/tui/file_tree.go`)

## Files to Create/Modify

- `internal/tui/filetree/model.go` - File tree model, Node struct, tree operations
- `internal/tui/filetree/node.go` - Node type definition, tri-state logic, tree traversal helpers
- `internal/tui/filetree/update.go` - Update handler for key events and loaded directory messages
- `internal/tui/filetree/view.go` - View rendering (placeholder; full styling in T-081)
- `internal/tui/filetree/lazy.go` - Lazy directory loading commands using discovery package
- `internal/tui/filetree/model_test.go` - Unit tests for navigation
- `internal/tui/filetree/node_test.go` - Unit tests for tri-state propagation
- `internal/tui/filetree/lazy_test.go` - Unit tests for lazy loading

## Testing Requirements

- Unit test: navigating down through a flat file list moves cursor correctly
- Unit test: expanding a directory makes children visible, collapsing hides them
- Unit test: toggling a directory sets all descendants to same state
- Unit test: mixed children causes parent to show `Partial` state
- Unit test: tri-state propagation walks up to root correctly for deeply nested changes
- Unit test: Page Up / Page Down moves cursor by viewport height
- Unit test: cursor clamps to valid range on tree changes
- Unit test: lazy loading produces `DirLoadedMsg` with correct children
- Unit test: ignored files (from `.gitignore`) are excluded from tree
- Unit test: binary files are excluded from tree
- Table-driven tests for all key bindings