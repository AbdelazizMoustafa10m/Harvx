# Project Name

## Coding Conventions

- Language: Go 1.24+
- Use `internal/` to enforce visibility boundaries
- Errors: `fmt.Errorf("context: %w", err)` for wrapping
- No global mutable state; pass dependencies via constructors
- Prefer `io.Reader`/`io.Writer` interfaces for testability
- Table-driven tests with `testify/assert` and `testify/require`

## Rules

- All exported functions and types have doc comments
- `go vet ./...` must pass with zero warnings
- `go test ./...` must pass before committing
- No `init()` functions except for cobra command registration
- Never commit secrets or credentials

## Quick Reference

- Build: `go build ./cmd/harvx/`
- Test: `go test ./...`
- Lint: `go vet ./...`

## Dynamic Context

Run `harvx brief` for a full project overview including architecture,
module map, and build commands. This file intentionally stays lean—
rules only, not architecture dumps.
