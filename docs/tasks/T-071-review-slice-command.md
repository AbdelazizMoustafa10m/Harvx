# T-071: Review Slice Command (`harvx review-slice`)

**Priority:** Must Have
**Effort:** Large (16-24hrs)
**Dependencies:** T-066 (Pipeline Library API), T-070 (Brief Command), T-069 (Assert-Include)
**Phase:** 5 - Workflows

---

## Description

Implement the `harvx review-slice --base <ref> --head <ref>` command that generates a PR-specific context slice containing changed files and their bounded neighborhood (public interfaces, config defaults, related tests, dependency neighbors). This is the dynamic companion to the stable `brief` artifact, providing reviewers with the specific code context needed to understand a change within the broader project architecture.

## User Story

As a developer running automated AI code reviews, I want to generate a PR-specific context slice that includes changed files plus their architectural neighborhood so that AI reviewers can catch cross-module constraint violations, not just style nits.

## Acceptance Criteria

- [ ] `harvx review-slice --base <ref> --head <ref>` command is registered as a Cobra subcommand
- [ ] Accepts git refs for both `--base` and `--head` (branch names, tags, commit SHAs, `HEAD`, `origin/main`)
- [ ] Generates a Review Slice artifact containing:
  - Changed files: verbatim content (or compressed if profile enables compression)
  - Bounded neighborhood context:
    - Public interfaces touched (exported funcs/types from modified files, schema definitions)
    - Config defaults referenced by changes (config files that modified code reads from)
    - Related tests and fixtures for the changed module(s)
    - Dependency neighbors: files that import modified files, within a configurable depth limit
- [ ] Changed files are identified using `git diff --name-only --diff-filter=ACMR <base>...<head>`
- [ ] Neighborhood depth is configurable via `slice_depth` in profile (default: 1 -- direct imports only)
- [ ] Output budget is configurable via `slice_max_tokens` in profile (default: 20000 tokens)
- [ ] Supports `--profile <name>`, `--stdout`, `-o <path>`, `--json`, `--target`, `--assert-include`
- [ ] Output is deterministic: sorted paths, stable rendering, XXH3 content hash
- [ ] The slice **never summarizes semantics** -- it extracts verbatim source at AST node boundaries
- [ ] Compressed content (via tree-sitter) is clearly marked per file
- [ ] When no files are changed between base and head, outputs an empty slice with a clear message
- [ ] Unit tests verify correct neighbor discovery and budget enforcement

## Technical Notes

- Implement in `internal/workflows/review_slice.go` and `internal/cli/review_slice.go`
- Git diff execution: shell out to `git diff --name-only --diff-filter=ACMR <base>...<head>` to get changed file list
  - `ACMR` filters: Added, Copied, Modified, Renamed (exclude Deleted since we can't include their content)
  - Validate that the current directory is a git repo before proceeding
  - Validate that both refs exist before running diff
- Neighborhood discovery algorithm:
  1. Start with set of changed files
  2. For each changed file, find its imports/dependencies (language-aware via file extension heuristics)
  3. For each import, check if the imported file exports public interfaces that changed
  4. Include test files matching patterns: `*_test.go` for `*.go`, `*.test.ts` for `*.ts`, `__tests__/` for the module
  5. Include config files referenced by changed code (heuristic: scan for config key references)
- Import resolution strategy (v1 -- heuristic, not full AST):
  - Go: parse `import` blocks, resolve relative to module root
  - TypeScript/JavaScript: parse `import`/`require` statements, resolve relative paths
  - Python: parse `from X import` / `import X`, resolve relative paths
  - For other languages: include files in the same directory as a simple heuristic
- Budget enforcement: changed files always included first (highest priority), then neighborhood files by proximity, then truncate
- Content for neighborhood files can be compressed (signatures only) even if the changed files are shown verbatim
- Reference: PRD Sections 5.11.1 (Artifact 2 - Review Slice), 5.9 (review-slice subcommand), Open Question #7 (neighborhood depth)

## Files to Create/Modify

- `internal/workflows/review_slice.go` - Review slice generation logic
- `internal/workflows/review_slice_test.go` - Unit tests
- `internal/workflows/neighbors.go` - Bounded neighborhood discovery
- `internal/workflows/neighbors_test.go` - Neighbor discovery tests
- `internal/workflows/imports.go` - Import/dependency parsing (multi-language)
- `internal/workflows/imports_test.go` - Import parsing tests
- `internal/diff/git.go` - Git diff execution and changed file list extraction
- `internal/diff/git_test.go` - Git operations tests
- `internal/cli/review_slice.go` - Cobra command registration and flag handling
- `internal/config/types.go` - Add `SliceMaxTokens`, `SliceDepth` fields to Profile struct
- `testdata/sample-repo/` - Add test files with import relationships for neighbor tests

## Testing Requirements

- Unit test: Changed files between two known refs are correctly identified
- Unit test: Neighborhood includes test files matching changed modules
- Unit test: Neighborhood includes files that import changed files (depth 1)
- Unit test: Neighborhood respects `slice_depth` limit (depth 0 = no neighbors)
- Unit test: Go import parsing extracts correct file paths
- Unit test: TypeScript import parsing extracts correct file paths
- Unit test: Python import parsing extracts correct file paths
- Unit test: Token budget truncates neighborhood before changed files
- Unit test: Output is deterministic (two runs produce identical hash)
- Unit test: `--json` returns valid metadata with file list and token count
- Unit test: Invalid git refs produce clear error message
- Unit test: Non-git directory produces clear error message
- Golden test: Review slice output matches expected fixture for sample repo
- Edge case: No files changed produces empty slice with message
- Edge case: All changed files exceed budget -- neighborhood is omitted entirely
- Edge case: Deleted files are excluded from output but mentioned in metadata