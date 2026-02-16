# T-051: Directory Tree Builder (In-Memory Tree + Rendering)

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** None (uses file path list; no dependency on discovery package internals)
**Phase:** 4 - Output & Rendering

---

## Description

Build the in-memory directory tree data structure and renderer that produces a Unicode box-drawing character visualization of included project files. This is a self-contained module in `internal/output/tree.go` that accepts a sorted list of relative file paths (with optional metadata) and produces a formatted string. The tree is used both in the main output document and by `harvx preview`.

## User Story

As a developer, I want the context output to include a clear visual representation of my project structure so that the LLM understands the codebase organization at a glance.

## Acceptance Criteria

- [ ] `TreeNode` nested struct representing directories and files is defined in `internal/output/tree.go`
- [ ] `BuildTree(files []FileEntry) *TreeNode` constructs the tree from a flat list of relative paths
- [ ] `RenderTree(root *TreeNode, opts TreeRenderOpts) string` produces Unicode box-drawing output
- [ ] Output uses `├──`, `└──`, `│` connectors correctly (no trailing whitespace)
- [ ] Directories are rendered before files at each level, both sorted alphabetically (case-insensitive)
- [ ] Emoji indicators are used: `📁` for directories, `📄` for files
- [ ] `--tree-depth <n>` support: rendering stops at depth n, showing `...` for truncated branches
- [ ] Empty intermediate directories are collapsed (e.g., `src/utils/` with one child dir becomes `src/utils/helpers/`)
- [ ] Optional file size and token count annotations: `📄 main.go (1.2 KB, 340 tokens)`
- [ ] Tree renders only included files (the input list is already filtered)
- [ ] Unit tests achieve >= 90% coverage for tree building and rendering
- [ ] Golden test comparing rendered output against expected string for a sample project structure

## Technical Notes

- **Data structure**: `TreeNode` has `Name string`, `IsDir bool`, `Children []*TreeNode`, `Size int64`, `TokenCount int`, `Tier int`. Directories hold children; files are leaf nodes.
- **Building algorithm**: For each file path, split by `/`, walk/create directory nodes, insert file at leaf. After all files are inserted, sort each node's children (dirs first, then alphabetical).
- **Collapsing**: After building, walk the tree. If a directory has exactly one child and that child is also a directory, merge them into `parent/child` and recurse.
- **Rendering**: Recursive DFS with a prefix string that accumulates `│   ` or `    ` per depth level. Last child at each level uses `└──`, others use `├──`.
- **Depth limit**: `TreeRenderOpts.MaxDepth int` (0 = unlimited). When depth is reached, append `├── ...` and stop recursing.
- **Metadata display**: Controlled by `TreeRenderOpts.ShowSize bool` and `TreeRenderOpts.ShowTokens bool`.
- Reference: PRD Section 5.12

## Files to Create/Modify

- `internal/output/tree.go` - TreeNode struct, BuildTree, RenderTree, collapse logic
- `internal/output/tree_test.go` - Unit tests (structure building, rendering, collapsing, depth limits, sorting)
- `testdata/expected-output/tree-basic.txt` - Golden test expected output
- `testdata/expected-output/tree-with-metadata.txt` - Golden test with size/tokens
- `testdata/expected-output/tree-collapsed.txt` - Golden test for collapsed dirs
- `testdata/expected-output/tree-depth-limited.txt` - Golden test for depth limit

## Testing Requirements

- Unit test: building tree from flat path list produces correct parent-child hierarchy
- Unit test: directories sort before files, both alphabetically (case-insensitive)
- Unit test: single-child directory chains collapse into combined path
- Unit test: depth limit truncates at correct level with `...` indicator
- Unit test: empty input produces empty string (or just project root)
- Unit test: single file at root level renders correctly
- Unit test: deeply nested paths (10+ levels) render without stack overflow
- Unit test: metadata annotations appear when enabled and are omitted when disabled
- Golden test: compare full render against expected output files
- Edge case: files with Unicode characters in names
- Edge case: paths with leading/trailing slashes are handled gracefully