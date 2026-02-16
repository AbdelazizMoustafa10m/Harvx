# Harvx - Copilot Instructions

## Project Context

Harvx is a Go CLI that packages repositories into LLM-optimized context documents.

Key goals:

- Fast generation at large repository scale
- Deterministic, reproducible output
- Strong secret redaction and safe defaults
- Zero runtime dependencies (single static binary)
- Smooth automation in CI/CD and agent workflows

Primary references:

- `docs/prd/PRD-Harvx.md`
- `AGENTS.md`
- `docs/tasks/INDEX.md`
- `docs/tasks/PROGRESS.md`

## Review Priorities (ordered)

1. Correctness and regressions in discovery/filtering/token budgeting behavior
2. Security and redaction safety (secret leaks, false negatives)
3. Determinism and reproducibility (stable ordering/output/hash behavior)
4. Performance and scalability for large repos
5. CLI UX consistency (flags, exit codes, headless behavior)
6. Test quality and coverage for changed logic

## Go Engineering Rules

- Return errors; do not panic in normal flow
- Wrap errors with context using `%w`
- Keep package boundaries explicit (`internal/` design)
- Avoid mutable global state; pass dependencies via constructors
- Use interfaces where they improve testability (`io.Reader`, `io.Writer`, service interfaces)
- Exported symbols must have doc comments
- Prefer deterministic iteration and sorted outputs for stable artifacts

## CLI and Pipeline Rules

- Respect exit-code contract: `0` success, `1` failure, `2` partial success
- Keep commands non-interactive by default (unless explicitly interactive mode)
- Use `slog` for diagnostics instead of ad-hoc prints
- Preserve backward-compatible flag behavior unless intentionally versioned

## Testing Expectations

- Prefer table-driven tests with `testify/assert` and `testify/require`
- Add/adjust golden tests when output contracts change
- Use `testdata/` fixtures rather than inline mega fixtures
- Cover edge cases and failure paths, not only happy path
- Keep tests deterministic and independent

## What Not To Flag

- Pure formatting nits already handled by `gofmt`
- Issues already guaranteed to be caught by CI (`go vet`, `go test`, linter), unless CI coverage is missing
