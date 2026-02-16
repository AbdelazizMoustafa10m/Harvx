# Harvx Patterns

Reference patterns for reviewing Harvx changes.

## Go and Error Handling

- Wrap errors with context: `fmt.Errorf("context: %w", err)`.
- Avoid panic except unrecoverable startup failures.
- Prefer explicit dependency injection; avoid global mutable state.
- Use `io.Reader`/`io.Writer` interfaces for testable units.

## Cobra CLI Contracts

- Command flags must have deterministic defaults and clear help text.
- CLI output contracts should keep machine-readable output stable.
- Diagnostics should use structured logging (`slog`) rather than ad-hoc prints.
- Respect documented exit codes: success=0, error=1, partial=2.

## Config and Profiles

- TOML parsing should reject unknown keys where required.
- Merging behavior must be predictable and documented.
- Environment overrides must not silently break profile invariants.

## Determinism and Reproducibility

- Sort map/file outputs before rendering.
- Keep token counts and file ordering stable for same input.
- Preserve deterministic hashing behavior across platforms.

## Security and Safety

- Secret redaction paths must fail closed, not fail open.
- Regex and parsing logic must avoid unbounded behaviors.
- Script/tool invocations must avoid shell injection vectors.
- Avoid logging secrets, tokens, auth headers, or raw credentials.

## Testing Standards

- Prefer table-driven tests for behavior matrices.
- Add golden tests for stable output formats.
- Use `t.TempDir()` and fixture data in `testdata/`.
- Cover negative/error paths, not only happy paths.
