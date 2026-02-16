# T-072: Module Slice Command (`harvx slice`)

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-066 (Pipeline Library API), T-071 (Review Slice -- shares neighbor logic)
**Phase:** 5 - Workflows

---

## Description

Implement the `harvx slice --path <module>` command for generating targeted context about a specific module or directory. This command enables on-demand context retrieval -- an agent working on a specific area can request a focused slice without loading the entire codebase. The slice includes the module's files, its public interfaces, tests, and immediate dependency neighbors.

## User Story

As a coding agent user, I want to request targeted context about a specific module (e.g., `lib/services/auth`) so that I get deep, focused context for the area I am working in without consuming my entire context window on unrelated code.

## Acceptance Criteria

- [ ] `harvx slice --path <module>` command is registered as a Cobra subcommand
- [ ] `--path` accepts a relative directory path within the repository (e.g., `internal/auth`, `lib/services`, `src/components/dashboard`)
- [ ] Multiple `--path` flags are supported for slicing multiple modules: `--path internal/auth --path internal/middleware`
- [ ] Generated slice includes:
  - All files within the specified path(s) (respecting ignore patterns and profile rules)
  - Public interfaces exported by the module (function signatures, type declarations)
  - Test files associated with the module
  - Dependency neighbors (files that import or are imported by the module, depth 1)
- [ ] Supports all standard flags: `--profile`, `--stdout`, `-o <path>`, `--json`, `--target`, `--max-tokens`, `--compress`
- [ ] Output is deterministic: sorted paths, stable rendering, XXH3 content hash
- [ ] Token budget defaults to `slice_max_tokens` from profile (default: 20000)
- [ ] Invalid or non-existent paths produce clear error messages
- [ ] Module files are prioritized over neighbor files when budget is tight
- [ ] Unit tests verify correct module boundary discovery

## Technical Notes

- Implement in `internal/workflows/slice.go` and `internal/cli/slice.go`
- Reuses neighbor discovery logic from T-071 (`internal/workflows/neighbors.go`)
- Slice is essentially a scoped version of the full pipeline:
  1. Run discovery but filter to only include files under the specified path(s)
  2. Run relevance sorting within the slice scope
  3. Discover neighbors (imports going out, imports coming in) limited to depth 1
  4. Apply token budget: module files first, then neighbors
- Path resolution: `--path` is resolved relative to the repository root (same as `--dir`)
- The slice should include a header section identifying the module scope and its relationship to the broader project
- For neighbor discovery, reuse the import parsing logic from T-071
- Compression can be applied differently to module files (verbatim) vs neighbors (compressed) -- this is a nice optimization but not required for v1
- Reference: PRD Sections 5.11.2 (harvx slice --path), 5.9 (slice subcommand)

## Files to Create/Modify

- `internal/workflows/slice.go` - Module slice generation logic
- `internal/workflows/slice_test.go` - Unit tests
- `internal/cli/slice.go` - Cobra command registration and flag handling
- `internal/cli/slice_test.go` - CLI integration tests

## Testing Requirements

- Unit test: Slice includes all files under specified path
- Unit test: Slice excludes files outside specified path (except neighbors)
- Unit test: Multiple `--path` flags union the file sets
- Unit test: Neighbor discovery finds files importing the sliced module
- Unit test: Neighbor discovery finds files imported by the sliced module
- Unit test: Token budget prioritizes module files over neighbors
- Unit test: Invalid path returns clear error message
- Unit test: `--json` returns valid metadata
- Unit test: Output is deterministic (two runs produce identical hash)
- Edge case: Path with no files (empty directory) returns clear message
- Edge case: Path that is a single file (not directory) includes that file plus neighbors
- Edge case: Deeply nested path works correctly (e.g., `src/components/dashboard/widgets`)