# Harvx - LLM Context Generator

## Project Overview

Harvx is a Go CLI tool that packages codebases into LLM-optimized context documents.
Single binary (~15-20MB), zero runtime dependencies, cross-platform (macOS/Linux/Windows).

**PRD:** `docs/prd/PRD-Harvx.md`
**Tasks:** `docs/tasks/T-XXX-*.md` (95 tasks across 6 phases)
**Progress:** `docs/tasks/PROGRESS.md`
**Index:** `docs/tasks/INDEX.md`

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.24+ |
| CLI Framework | spf13/cobra v1.8.x |
| Logging | log/slog (stdlib) |
| Config | BurntSushi/toml v1.5.0 + koanf/v2 |
| Testing | stretchr/testify v1.9+ |
| Glob Matching | bmatcuk/doublestar v4.x |
| Gitignore | sabhiram/go-gitignore v1.1.0 |
| Parallelism | x/sync/errgroup |
| Token Counting | pkoukk/tiktoken-go |
| Hashing | zeebo/xxh3 |
| TUI | charmbracelet/bubbletea v1.x + lipgloss |
| WASM Runtime | tetratelabs/wazero |
| Distribution | GoReleaser v2 + Cosign + Syft |

## Project Structure

```
cmd/harvx/            # Entry point (main.go)
internal/
  cli/                # Cobra commands (root, generate, version, etc.)
  config/             # TOML config, profiles, validation
  discovery/          # File walking, filtering, binary detection
  relevance/          # Tier sorting, token budgets
  tokenizer/          # Token counting (cl100k, o200k)
  security/           # Secret redaction, entropy analysis
  compression/        # Tree-sitter WASM compression
  output/             # Markdown/XML rendering, tree builder
  diff/               # State snapshots, diffing
  workflows/          # brief, slice, review-slice, verify
  tui/                # Bubble Tea interactive UI
  server/             # MCP server
  pipeline/           # Core pipeline, DTOs, exit codes
grammars/             # Embedded WASM grammars
templates/            # Go text/template files
testdata/             # Test fixtures
  sample-repo/        # Curated sample repository
  secrets/            # Mock secrets for redaction tests
  monorepo/           # Multi-package test repo
  expected-output/    # Golden test reference files
docs/
  prd/                # Product Requirements Document
  tasks/              # Task specs, progress, index
scripts/              # Automation scripts (ralph loop)
```

## Go Conventions

### Code Style
- `go vet ./...` must pass with zero warnings
- `go test ./...` must pass
- All exported functions/types have doc comments
- Use `internal/` to enforce visibility boundaries
- Errors: `fmt.Errorf("context: %w", err)` for wrapping
- No global mutable state; pass dependencies via constructors
- No `init()` functions except for cobra command registration
- Prefer `io.Reader`/`io.Writer` interfaces for testability

### Naming
- Package names: lowercase, single word (`discovery`, `config`, `pipeline`)
- Interfaces: `-er` suffix (`Walker`, `Tokenizer`, `Redactor`, `Renderer`)
- Test files: `*_test.go` in same package
- Test helpers: `testutil` package or `_test.go` helpers
- Constants: `CamelCase` not `SCREAMING_SNAKE`

### Testing
- Table-driven tests with `testify/assert` and `testify/require`
- Golden test pattern for output verification (see T-094)
- Use `t.Helper()` in test helpers
- Use `testdata/` for test fixtures
- Use `t.TempDir()` for temporary directories
- Subtests with `t.Run("descriptive name", ...)`

### Error Handling
- Return errors, never panic (except truly unrecoverable)
- Exit codes: 0 (success), 1 (error), 2 (partial success)
- Use `slog` for structured logging, never `fmt.Printf` for diagnostics
- Wrap errors with context: `fmt.Errorf("walking %s: %w", path, err)`

### Verification Commands

```bash
go build ./cmd/harvx/    # Compilation check
go vet ./...             # Static analysis
go test ./...            # All tests
go mod tidy              # Module hygiene (no drift)
```

## Phase Structure

| Phase | Tasks | Description |
|-------|-------|-------------|
| 1 | T-001 to T-015 | Foundation: CLI skeleton, file discovery engine |
| 2a | T-016 to T-025 | Intelligence: TOML profiles, config merging |
| 2b | T-026 to T-033 | Intelligence: Relevance tiers, token counting |
| 3a | T-034 to T-041 | Security: Secret redaction, entropy analysis |
| 3b | T-042 to T-050 | Compression: Tree-sitter WASM, language compressors |
| 4a | T-051 to T-058 | Output: Markdown/XML rendering, tree builder |
| 4b | T-059 to T-065 | State: Snapshots, diffing, caching |
| 5a | T-066 to T-078 | Workflows: brief, slice, review-slice, MCP |
| 5b | T-079 to T-087 | TUI: Bubble Tea interactive file selector |
| 6 | T-088 to T-095 | Polish: GoReleaser, benchmarks, golden tests |

## Key Technical Decisions

1. **koanf v2 over Viper** -- 313% smaller binary, doesn't force-lowercase TOML keys
2. **zeebo/xxh3 over cespare/xxhash** -- Proper XXH3 in pure Go with SIMD
3. **Go stdlib regexp only** -- RE2 engine guarantees O(n) matching for untrusted input
4. **BurntSushi/toml v1.5.0** -- MetaData.Undecoded() for unknown-key detection
5. **Bubble Tea v1.x** -- v2 still in RC; stable v1.2+ for production
6. **CGO_ENABLED=0** -- Pure Go for cross-compilation, no C dependencies

## Task Workflow

When implementing a task:
1. Read the task spec: `docs/tasks/T-XXX-*.md`
2. Check dependencies are complete in `docs/tasks/PROGRESS.md`
3. Implement following acceptance criteria
4. Write tests (table-driven, golden where appropriate)
5. Verify: `go build ./cmd/harvx/ && go vet ./... && go test ./...`
6. Update `docs/tasks/PROGRESS.md` with completion entry
