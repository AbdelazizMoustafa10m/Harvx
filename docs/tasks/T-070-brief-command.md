# T-070: Repo Brief Command (`harvx brief`)

**Priority:** Must Have
**Effort:** Large (14-20hrs)
**Dependencies:** T-066 (Pipeline Library API), T-067 (Stdout/Exit Codes), T-068 (JSON Preview/Metadata)
**Phase:** 5 - Workflows

---

## Description

Implement the `harvx brief` command that generates a stable, small "Repo Brief" artifact (~1-4K tokens) containing project-wide invariants: README overview, architecture docs/ADRs, build/test commands, key invariants, and a high-level module map. The brief is designed to be cacheable across commits (content-addressed via hash), deterministic (sorted paths, stable rendering), and suitable for both automated review pipelines and agent session bootstrap.

## User Story

As a developer with a multi-agent review pipeline, I want to generate a stable project brief that provides my AI agents with architectural context (auth patterns, API contracts, build commands) so that they can review code changes with full project awareness.

## Acceptance Criteria

- [ ] `harvx brief` command is registered as a Cobra subcommand
- [ ] Generates a Repo Brief artifact containing:
  - README / high-level product overview (extracted from README.md, README.rst, or similar)
  - Architecture docs/ADRs (from `docs/`, `docs/adr/`, `architecture.md`, `ADR-*.md` patterns)
  - Build/test commands (from Makefile, package.json scripts, Taskfile, justfile, Cargo.toml)
  - Key invariants (from CLAUDE.md, agents.md, CONVENTIONS.md, .github/review/ patterns)
  - High-level module map (top-level directories with 1-line purpose derived from directory name/content)
- [ ] Output budget is configurable via `brief_max_tokens` in profile (default: 4000 tokens)
- [ ] Brief supports `--profile <name>` for project-specific configuration
- [ ] Brief supports `--stdout` for piping and `-o <path>` for file output
- [ ] Brief supports `--json` for machine-readable metadata (token count, content hash, files included)
- [ ] Brief supports `--target claude` for XML-formatted output
- [ ] Brief supports `--assert-include <pattern>` for coverage checks
- [ ] Output is deterministic: same repo state produces identical output (sorted paths, stable rendering, XXH3 content hash)
- [ ] Brief is cacheable: the content hash enables prompt caching on subsequent runs
- [ ] Module map generation is automatic: scans top-level directories and generates 1-line descriptions
- [ ] Brief gracefully handles missing sections (no README -> skip that section, no ADRs -> skip that section)
- [ ] Unit tests verify determinism (two runs produce identical content hash)

## Technical Notes

- Implement in `internal/workflows/brief.go` and `internal/cli/brief.go`
- Brief uses the pipeline library (T-066) with a specialized discovery configuration:
  - Discovery is limited to specific well-known paths (README.md, docs/, Makefile, etc.)
  - No general file walk -- brief has a curated file list
- Source file list for brief (in priority order):
  1. `README.md` / `README.rst` / `README` (first found)
  2. `CLAUDE.md` / `agents.md` / `CONVENTIONS.md` (project invariants)
  3. `docs/architecture.md` / `docs/ARCHITECTURE.md` / `ADR-*.md` / `docs/adr/*.md`
  4. Build files: `Makefile` (targets only), `package.json` (scripts section only), `Taskfile.yml`, `justfile`
  5. Config files: `go.mod` (module name + Go version), `Cargo.toml` (package section), `pyproject.toml` (project section)
  6. Review rules: `.github/review/**/*.md`
- For build files, extract only relevant sections (e.g., Makefile target names, package.json scripts) rather than full content
- Module map: list top directories with descriptions inferred from common conventions (e.g., `cmd/` -> "CLI entry points", `internal/` -> "Private packages", `lib/` -> "Shared libraries")
- Content is rendered in a fixed section order for stability
- Token budget enforcement: if content exceeds `brief_max_tokens`, lower-priority sections are truncated (module map first, then architecture docs)
- The brief output should include a header with content hash and token count
- Reference: PRD Sections 5.11.1 (Artifact 1 - Repo Brief), 5.9 (brief subcommand)

## Files to Create/Modify

- `internal/workflows/brief.go` - Brief generation logic (file discovery, section extraction, rendering)
- `internal/workflows/brief_test.go` - Unit tests for brief generation
- `internal/workflows/module_map.go` - Automatic module map generation
- `internal/workflows/module_map_test.go` - Module map tests
- `internal/workflows/section_extractor.go` - Extract relevant sections from build files (Makefile targets, package.json scripts)
- `internal/cli/brief.go` - Cobra command registration and flag handling
- `internal/cli/brief_test.go` - CLI integration tests
- `internal/config/types.go` - Add `BriefMaxTokens` field to Profile struct
- `testdata/sample-repo/README.md` - Test fixture for brief generation
- `testdata/sample-repo/Makefile` - Test fixture with build targets
- `testdata/expected-output/brief.md` - Golden test output

## Testing Requirements

- Unit test: Brief from sample repo includes all expected sections
- Unit test: Brief output is deterministic (two runs produce identical hash)
- Unit test: Missing README is handled gracefully (section omitted, no error)
- Unit test: Missing architecture docs are handled gracefully
- Unit test: Makefile target extraction works correctly
- Unit test: package.json scripts extraction works correctly
- Unit test: Module map generates correct directory descriptions
- Unit test: Token budget truncates lower-priority sections first
- Unit test: `brief_max_tokens` profile config is respected
- Unit test: `--json` returns valid metadata with token count and hash
- Unit test: `--target claude` produces XML-formatted output
- Golden test: Brief output matches expected fixture for sample repo
- Edge case: Empty repository produces minimal brief (just header)
- Edge case: Very large README is truncated within budget