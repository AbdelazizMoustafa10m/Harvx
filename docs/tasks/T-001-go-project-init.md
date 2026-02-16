# T-001: Go Project Initialization & Directory Structure

**Priority:** Must Have
**Effort:** Small (2-4hrs)
**Dependencies:** None
**Phase:** 1 - Foundation

---

## Description

Initialize the Go module, create the canonical directory structure from PRD Section 6.2, set up the entry point at `cmd/harvx/main.go`, and configure essential project files (`.gitignore`, `.editorconfig`, `LICENSE`). This task establishes the skeleton that every other task builds upon.

## User Story

As a developer joining the Harvx project, I want a clean, well-structured Go project so that I can immediately begin implementing features without debating layout conventions.

## Acceptance Criteria

- [ ] `go.mod` is initialized with module path `github.com/yourusername/harvx` (adjust to actual org) and Go 1.24 as the minimum version
- [ ] `cmd/harvx/main.go` exists with a minimal `func main()` that prints "harvx" and exits 0
- [ ] All directories from PRD Section 6.2 are created (with `.gitkeep` files where empty): `cmd/harvx/`, `internal/cli/`, `internal/config/`, `internal/discovery/`, `internal/relevance/`, `internal/tokenizer/`, `internal/security/`, `internal/compression/`, `internal/output/`, `internal/diff/`, `internal/workflows/`, `internal/tui/`, `internal/server/`, `internal/pipeline/`, `grammars/`, `templates/`, `testdata/sample-repo/`, `testdata/secrets/`, `testdata/monorepo/`, `testdata/expected-output/`
- [ ] `.gitignore` includes Go-specific ignores (`/harvx`, `*.exe`, `/dist/`, `/bin/`, `.harvx/`, `*.wasm` build artifacts, OS files)
- [ ] `.editorconfig` enforces consistent formatting (tabs for Go, UTF-8, LF line endings)
- [ ] `LICENSE` file with MIT license text
- [ ] `README.md` with project name, one-line description, and "Under Development" badge
- [ ] `go build ./cmd/harvx/` compiles successfully
- [ ] `go vet ./...` passes with zero warnings

## Technical Notes

- Use Go 1.24 (latest stable as of February 2026, ref: https://go.dev/doc/go1.24). The PRD specifies Go 1.22+ but targeting 1.24 gives access to the latest stdlib features including all slog improvements.
- Module path should match the planned GitHub repository URL.
- The `internal/` directory enforces Go's visibility rules -- packages under `internal/` are not importable by external consumers, which is correct for a CLI tool.
- Do NOT add any third-party dependencies yet -- those come in subsequent tasks.
- Reference: PRD Sections 6.1, 6.2

## Files to Create/Modify

- `go.mod` - Module initialization
- `cmd/harvx/main.go` - Entry point (minimal)
- `internal/cli/.gitkeep` - Placeholder
- `internal/config/.gitkeep` - Placeholder
- `internal/discovery/.gitkeep` - Placeholder
- `internal/relevance/.gitkeep` - Placeholder
- `internal/tokenizer/.gitkeep` - Placeholder
- `internal/security/.gitkeep` - Placeholder
- `internal/compression/.gitkeep` - Placeholder
- `internal/output/.gitkeep` - Placeholder
- `internal/diff/.gitkeep` - Placeholder
- `internal/workflows/.gitkeep` - Placeholder
- `internal/tui/.gitkeep` - Placeholder
- `internal/server/.gitkeep` - Placeholder
- `internal/pipeline/.gitkeep` - Placeholder
- `grammars/.gitkeep` - Placeholder
- `templates/.gitkeep` - Placeholder
- `testdata/sample-repo/.gitkeep` - Test fixtures directory
- `testdata/secrets/.gitkeep` - Secret regression test fixtures
- `testdata/monorepo/.gitkeep` - Monorepo test fixtures
- `testdata/expected-output/.gitkeep` - Golden test outputs
- `.gitignore` - Go + project-specific ignores
- `.editorconfig` - Editor configuration
- `LICENSE` - MIT license
- `README.md` - Project readme

## Testing Requirements

- `go build ./cmd/harvx/` succeeds
- `go vet ./...` passes
- Running the compiled binary outputs "harvx" and exits with code 0
