# Harvx Task Progress Log

## Summary

| Status | Count |
|--------|-------|
| Completed | 15 |
| In Progress | 0 |
| Not Started | 80 |

---

## Completed Tasks

### Phase 1: Foundation (T-001 to T-015)

- **Status:** Completed
- **Date:** 2026-02-16
- **Tasks:** 15 (14 Must Have, 1 Should Have)
- **Total Tests:** 200+ passing across 4 packages

#### Tech Stack (Phase 1)

| Package | Purpose |
|---------|---------|
| Go 1.24+ | Language runtime |
| log/slog (stdlib) | Structured logging |
| spf13/cobra v1.10.2 | CLI framework |
| stretchr/testify v1.9+ | Testing assertions |
| sabhiram/go-gitignore v1.1.0 | .gitignore parsing |
| bmatcuk/doublestar v4.10.0 | Glob pattern matching |
| golang.org/x/sync v0.19.0 | Bounded parallel execution (errgroup) |

#### Features Implemented

| Feature | Tasks | Description |
|---------|-------|-------------|
| Project Scaffold | T-001, T-002 | Go module, directory structure (13 `internal/` packages), Makefile with ldflags, `.golangci.yml`, `.editorconfig` |
| Central Data Types | T-003, T-010 | `FileDescriptor` (13 fields), `DiscoveryResult`, `ExitCode`, `OutputFormat`, `LLMTarget`, `HarvxError` custom error type |
| Structured Logging | T-004 | `slog`-based logging with text/JSON handlers, `HARVX_DEBUG`/`HARVX_LOG_FORMAT` env vars, component loggers |
| CLI Framework | T-005, T-006, T-007, T-008, T-009 | Root command, `version` (human + JSON), `generate`/`gen`, `completion` (bash/zsh/fish/powershell), 17 global flags with env var overrides |
| Ignore System | T-011, T-012 | `Ignorer` interface, `GitignoreMatcher` (hierarchical), `HarvxignoreMatcher`, `DefaultIgnoreMatcher` (41 patterns), `CompositeIgnorer` chain |
| File Analysis | T-013, T-014 | Binary detection (8KB null-byte scan), large file skipping, `PatternFilter` (doublestar globs), `GitTrackedFiles`, `SymlinkResolver` (loop detection) |
| Discovery Engine | T-015 | Two-phase parallel `Walker`: Phase 1 walks tree with all filters, Phase 2 reads content via `errgroup` with bounded concurrency; deterministic sorted output |

#### CLI Commands Built

| Command | Flags | Description |
|---------|-------|-------------|
| `harvx` (root) | `--verbose/-v`, `--quiet/-q` | Delegates to generate; logging init in PersistentPreRunE |
| `harvx generate` / `gen` | `--preview`, all 17 global flags | Main generation command (pipeline stub) |
| `harvx version` | `--json` | Build info with ldflags injection |
| `harvx completion <shell>` | -- | Shell completions for bash, zsh, fish, powershell |

**Global Flags (17):** `--dir`, `--output`, `--format` (markdown/xml), `--target` (claude/chatgpt/generic), `--profile`, `--filter`, `--include`, `--exclude`, `--skip-large-files` (human-readable sizes), `--git-tracked-only`, `--compress`, `--redact`, `--interactive/-i`, `--verbose/-v`, `--quiet/-q`, `--no-default-ignore`, `--follow-symlinks`

**Environment Overrides:** `HARVX_DIR`, `HARVX_OUTPUT`, `HARVX_FORMAT`, `HARVX_TARGET`, `HARVX_VERBOSE`, `HARVX_QUIET`, `HARVX_DEBUG`, `HARVX_LOG_FORMAT`

#### Key Types & Interfaces

| Type/Interface | Package | Purpose |
|---------------|---------|---------|
| `FileDescriptor` | `pipeline` | 13-field struct: Path, AbsPath, Size, Tier, TokenCount, ContentHash, Content, IsCompressed, Redactions, Language, IsSymlink, IsBinary, Error |
| `HarvxError` | `pipeline` | Custom error with exit code, `errors.Is`/`errors.As` support, constructors: `NewError`, `NewPartialError`, `NewRedactionError` |
| `FlagValues` | `config` | Collects all 17 global flags with validation and env var resolution |
| `Ignorer` | `discovery` | Interface for all ignore-pattern matchers (`IsIgnored(path, isDir) bool`) |
| `Walker` | `discovery` | Two-phase parallel file discovery engine |
| `PatternFilter` | `discovery` | Include/exclude/extension glob filtering with doublestar |
| `SymlinkResolver` | `discovery` | Thread-safe symlink loop detection with `sync.RWMutex` |

#### Key Files Reference

| Purpose | Location |
|---------|----------|
| Entry point | `cmd/harvx/main.go` |
| Build info | `internal/buildinfo/buildinfo.go` |
| CLI commands | `internal/cli/{root,generate,version,completion}.go` |
| Logging | `internal/config/logging.go` |
| Global flags | `internal/config/flags.go` |
| Pipeline types | `internal/pipeline/{types,errors,pipeline}.go` |
| Gitignore matcher | `internal/discovery/gitignore.go` |
| Default patterns | `internal/discovery/defaults.go` |
| Ignorer interface | `internal/discovery/ignore.go` |
| Harvxignore | `internal/discovery/harvxignore.go` |
| Binary detection | `internal/discovery/binary.go` |
| Pattern filter | `internal/discovery/filter.go` |
| Git-tracked files | `internal/discovery/git_tracked.go` |
| Symlink resolver | `internal/discovery/symlink.go` |
| Walker engine | `internal/discovery/walker.go` |
| Makefile | `Makefile` (14 targets, ldflags into `internal/buildinfo`) |
| Linter config | `.golangci.yml` |

#### Test Fixtures

| Directory | Purpose |
|-----------|---------|
| `testdata/sample-repo/` | Representative repo with Go, TypeScript, .gitignore, .harvxignore, build artifacts |
| `testdata/gitignore/` | 5 scenarios: root/nested, negation, comments, deep nesting, empty |
| `testdata/harvxignore/` | 3 scenarios: basic, negation, empty |
| `testdata/binary-detection/` | text.txt, binary.bin, empty.txt |

#### Test Coverage by Feature

| Feature | Tests | Benchmarks |
|---------|-------|------------|
| `cmd/harvx` main | 1 | -- |
| `internal/pipeline` types | 13 | -- |
| `internal/pipeline` errors | 19 | -- |
| `internal/config` logging | 11 | -- |
| `internal/config` flags | 30+ | -- |
| `internal/cli` root + exit codes | 19+ | -- |
| `internal/cli` version | 9 | -- |
| `internal/cli` generate | 11 | -- |
| `internal/cli` completion | 12 | -- |
| `internal/discovery` gitignore | 16 | 1 |
| `internal/discovery` defaults | 13 | 1 |
| `internal/discovery` ignore | 8 | -- |
| `internal/discovery` harvxignore | 15 | 1 |
| `internal/discovery` binary | 10 | 4 |
| `internal/discovery` filter | 14 | 1 |
| `internal/discovery` git_tracked | 13 | -- |
| `internal/discovery` symlink | 19 | 4 |
| `internal/discovery` walker | 24 | 2 |

**Deliverable:** `harvx` CLI with complete file discovery engine -- walks any repository respecting .gitignore, .harvxignore, default patterns, binary detection, large file skipping, extension filtering, symlink handling, and git-tracked-only mode with bounded parallel content loading.

---

## In Progress Tasks

_None currently_

---

## Not Started Tasks

### Phase 2: Profiles (T-016 to T-025)

- **Status:** Not Started
- **Tasks:** 10 (8 Must Have, 2 Should Have)
- **Estimated Effort:** 75-105 hours
- **PRD Roadmap:** Weeks 4-6

#### Task List

| Task | Name | Priority | Effort | Status |
|------|------|----------|--------|--------|
| T-016 | Configuration Types, Defaults, and TOML Loading | Must Have | Medium (8-12hrs) | Not Started |
| T-017 | Multi-Source Configuration Merging and Resolution | Must Have | Large (14-20hrs) | Not Started |
| T-018 | Configuration File Auto-Detection and Discovery | Must Have | Small (3-5hrs) | Not Started |
| T-019 | Profile Inheritance with Deep Merge | Must Have | Medium (8-12hrs) | Not Started |
| T-020 | Configuration Validation and Lint Engine | Must Have | Medium (8-12hrs) | Not Started |
| T-021 | Framework-Specific Profile Templates | Must Have | Medium (6-10hrs) | Not Started |
| T-022 | Profile CLI -- init, list, show | Must Have | Medium (8-12hrs) | Not Started |
| T-023 | Profile CLI -- lint and explain | Should Have | Medium (8-12hrs) | Not Started |
| T-024 | Config Debug Command | Should Have | Small (4-6hrs) | Not Started |
| T-025 | Profile Integration Tests and Golden Tests | Must Have | Medium (8-12hrs) | Not Started |

**Deliverable:** `harvx --profile finvault --target claude` produces architecture-aware, token-budgeted output.

---

### Phase 3: Relevance & Tokens (T-026 to T-033)

- **Status:** Not Started
- **Tasks:** 8 (6 Must Have, 2 Should Have)
- **Estimated Effort:** 54-80 hours

#### Task List

| Task | Name | Priority | Effort | Status |
|------|------|----------|--------|--------|
| T-026 | Tier Definitions and Default Tier Assignments | Must Have | Medium (6-8hrs) | Not Started |
| T-027 | Glob-Based File-to-Tier Matching | Must Have | Medium (8-12hrs) | Not Started |
| T-028 | Relevance Sorter -- Sort Files by Tier and Path | Must Have | Small (4-6hrs) | Not Started |
| T-029 | Tokenizer Interface and Implementations (cl100k, o200k, none) | Must Have | Medium (8-12hrs) | Not Started |
| T-030 | Parallel Per-File Token Counting | Must Have | Medium (6-10hrs) | Not Started |
| T-031 | Token Budget Enforcement with Truncation Strategies | Must Have | Medium (8-12hrs) | Not Started |
| T-032 | Relevance Explain and Inclusion Summary | Should Have | Medium (6-8hrs) | Not Started |
| T-033 | Token Reporting CLI Flags and Heatmap | Should Have | Medium (8-12hrs) | Not Started |

---

### Phase 4: Security (T-034 to T-041)

- **Status:** Not Started
- **Tasks:** 8 (8 Must Have)
- **Estimated Effort:** 65-96 hours
- **PRD Roadmap:** Weeks 7-9

#### Task List

| Task | Name | Priority | Effort | Status |
|------|------|----------|--------|--------|
| T-034 | Redaction Core Types, Interfaces, and Pattern Registry | Must Have | Medium (6-8hrs) | Not Started |
| T-035 | Gitleaks-Inspired Secret Detection Patterns | Must Have | Large (14-20hrs) | Not Started |
| T-036 | Shannon Entropy Analyzer | Must Have | Medium (6-10hrs) | Not Started |
| T-037 | Streaming Redaction Filter Pipeline | Must Have | Large (14-20hrs) | Not Started |
| T-038 | Sensitive File Default Exclusions & Heightened Scanning | Must Have | Small (4-6hrs) | Not Started |
| T-039 | Redaction Report and Output Summary | Must Have | Medium (6-10hrs) | Not Started |
| T-040 | CLI Redaction Flags and Profile Configuration | Must Have | Medium (6-10hrs) | Not Started |
| T-041 | Secret Detection Regression Test Corpus & Fuzz Testing | Must Have | Medium (8-12hrs) | Not Started |

**Deliverable:** `harvx --compress --profile finvault` produces compressed, redacted output with zero known secret leaks.

---

### Phase 5: Compression (T-042 to T-050)

- **Status:** Not Started
- **Tasks:** 9 (8 Must Have, 1 Should Have)
- **Estimated Effort:** 83-114 hours

#### Task List

| Task | Name | Priority | Effort | Status |
|------|------|----------|--------|--------|
| T-042 | Wazero WASM Runtime Setup and Grammar Embedding | Must Have | Medium (8-12hrs) | Not Started |
| T-043 | Language Detection and LanguageCompressor Interface | Must Have | Small (3-4hrs) | Not Started |
| T-044 | Tier 1 Compressor -- TypeScript and JavaScript | Must Have | Large (16-20hrs) | Not Started |
| T-045 | Tier 1 Compressor -- Go | Must Have | Medium (8-12hrs) | Not Started |
| T-046 | Tier 1 Compressor -- Python and Rust | Must Have | Large (14-18hrs) | Not Started |
| T-047 | Tier 2 Compressor -- Java, C, and C++ | Should Have | Medium (10-14hrs) | Not Started |
| T-048 | Tier 2 Config Compressors & Fallback | Must Have | Small (4-6hrs) | Not Started |
| T-049 | Compression Orchestrator and Pipeline Integration | Must Have | Medium (10-14hrs) | Not Started |
| T-050 | Regex Heuristic Fallback and E2E Compression Tests | Must Have | Medium (10-14hrs) | Not Started |

---

### Phase 6: Output & Rendering (T-051 to T-058)

- **Status:** Not Started
- **Tasks:** 8 (6 Must Have, 2 Should Have)
- **Estimated Effort:** 54-80 hours
- **PRD Roadmap:** Weeks 10-11

#### Task List

| Task | Name | Priority | Effort | Status |
|------|------|----------|--------|--------|
| T-051 | Directory Tree Builder | Must Have | Medium (8-12hrs) | Not Started |
| T-052 | Markdown Output Renderer with Go Templates | Must Have | Large (14-20hrs) | Not Started |
| T-053 | XML Output Renderer for Claude Target | Must Have | Medium (8-12hrs) | Not Started |
| T-054 | Content Hashing (XXH3) and Deterministic Output | Must Have | Small (3-4hrs) | Not Started |
| T-055 | Output Writer, File Path Resolution, Stdout Support | Must Have | Medium (6-8hrs) | Not Started |
| T-056 | Output Splitter (Multi-Part File Generation) | Should Have | Medium (8-12hrs) | Not Started |
| T-057 | Metadata JSON Sidecar Generation | Should Have | Small (3-4hrs) | Not Started |
| T-058 | Output Pipeline Integration and Golden Tests | Must Have | Medium (8-12hrs) | Not Started |

**Deliverable:** Full output rendering with Markdown, XML, splitting, and metadata sidecar support.

---

### Phase 7: State & Diff (T-059 to T-065)

- **Status:** Not Started
- **Tasks:** 7 (all Should Have)
- **Estimated Effort:** 38-64 hours

#### Task List

| Task | Name | Priority | Effort | Status |
|------|------|----------|--------|--------|
| T-059 | State Snapshot Types and JSON Serialization | Should Have | Small (2-4hrs) | Not Started |
| T-060 | Content Hashing with XXH3 | Should Have | Small (2-4hrs) | Not Started |
| T-061 | State Cache Persistence (Read/Write) | Should Have | Medium (6-12hrs) | Not Started |
| T-062 | State Comparison Engine | Should Have | Medium (6-12hrs) | Not Started |
| T-063 | Git-Aware Diffing | Should Have | Medium (6-12hrs) | Not Started |
| T-064 | `harvx diff` Subcommand and `--diff-only` Flag | Should Have | Medium (6-12hrs) | Not Started |
| T-065 | Cache Subcommands and Change Summary Rendering | Should Have | Medium (6-12hrs) | Not Started |

---

### Phase 8: Workflows (T-066 to T-078)

- **Status:** Not Started
- **Tasks:** 13 (11 Must Have, 1 Should Have, 1 Nice to Have)
- **Estimated Effort:** 116-180 hours
- **PRD Roadmap:** Weeks 10-11

#### Task List

| Task | Name | Priority | Effort | Status |
|------|------|----------|--------|--------|
| T-066 | Core Pipeline as Go Library API | Must Have | Large (14-20hrs) | Not Started |
| T-067 | Stdout Mode, Exit Codes, Non-Interactive Defaults | Must Have | Medium (6-10hrs) | Not Started |
| T-068 | JSON Preview Output and Metadata Sidecar | Must Have | Medium (8-12hrs) | Not Started |
| T-069 | Assert-Include Coverage Checks & Env Var Overrides | Must Have | Medium (6-10hrs) | Not Started |
| T-070 | Repo Brief Command (`harvx brief`) | Must Have | Large (14-20hrs) | Not Started |
| T-071 | Review Slice Command (`harvx review-slice`) | Must Have | Large (16-24hrs) | Not Started |
| T-072 | Module Slice Command (`harvx slice`) | Must Have | Medium (8-12hrs) | Not Started |
| T-073 | Workspace Manifest Config and Command | Must Have | Medium (10-14hrs) | Not Started |
| T-074 | Session Bootstrap Docs & Claude Code Hooks | Must Have | Medium (6-10hrs) | Not Started |
| T-075 | Verify Command (`harvx verify`) | Must Have | Medium (8-12hrs) | Not Started |
| T-076 | Golden Questions Harness & Quality Evaluation | Should Have | Medium (8-12hrs) | Not Started |
| T-077 | MCP Server v1.1 (`harvx mcp serve`) | Nice to Have | Large (16-24hrs) | Not Started |
| T-078 | Workflow Integration Tests (E2E) | Must Have | Medium (10-14hrs) | Not Started |

**Deliverable:** `harvx brief && harvx review-slice --base main --head HEAD` enriches review pipelines.

---

### Phase 9: Interactive TUI (T-079 to T-087)

- **Status:** Not Started
- **Tasks:** 9 (9 Must Have)
- **Estimated Effort:** 72-104 hours
- **PRD Roadmap:** Weeks 12-13

#### Task List

| Task | Name | Priority | Effort | Status |
|------|------|----------|--------|--------|
| T-079 | Bubble Tea Application Scaffold & Elm Architecture | Must Have | Medium (8-12hrs) | Not Started |
| T-080 | File Tree Data Model & Keyboard Navigation | Must Have | Large (14-20hrs) | Not Started |
| T-081 | File Tree Visual Rendering & Tier Color Coding | Must Have | Medium (8-12hrs) | Not Started |
| T-082 | Stats Panel with Live Token Counting & Budget Bar | Must Have | Medium (8-12hrs) | Not Started |
| T-083 | Profile Selector & Action Keybindings | Must Have | Medium (6-10hrs) | Not Started |
| T-084 | Lipgloss Styling, Responsive Layout & Theme Support | Must Have | Medium (8-12hrs) | Not Started |
| T-085 | Search/Filter, Tier Views & Help Overlay | Must Have | Medium (8-12hrs) | Not Started |
| T-086 | TUI State Serialization to Profile TOML & Smart Default | Must Have | Small (4-6hrs) | Not Started |
| T-087 | TUI Integration Testing & Pipeline Wiring | Must Have | Medium (6-10hrs) | Not Started |

**Deliverable:** `harvx -i` launches a beautiful, responsive TUI for visual file selection with real-time token counting.

---

### Phase 10: Polish & Distribution (T-088 to T-095)

- **Status:** Not Started
- **Tasks:** 7 (7 Must Have)
- **Estimated Effort:** 48-72 hours
- **PRD Roadmap:** Week 14

#### Task List

| Task | Name | Priority | Effort | Status |
|------|------|----------|--------|--------|
| T-088 | GoReleaser Config with Cosign Signing & Syft SBOM | Must Have | Medium (8-12hrs) | Not Started |
| T-089 | GitHub Release Automation Workflow | Must Have | Small (4-6hrs) | Not Started |
| T-090 | Shell Completion Generation & Man Pages | Must Have | Medium (6-8hrs) | Not Started |
| T-091 | Performance Benchmarking Suite | Must Have | Medium (8-12hrs) | Not Started |
| T-092 | Integration Test Suite Against Real OSS Repos | Must Have | Medium (8-12hrs) | Not Started |
| T-093 | Fuzz Testing for Redaction & Config Parsing | Must Have | Medium (6-10hrs) | Not Started |
| T-094 | Golden Test Infrastructure | Must Have | Medium (8-12hrs) | Not Started |

**Deliverable:** Published v1.0.0 with signed binaries for all platforms.

---

## Notes

### Key Technical Decisions (from agent research)

1. **koanf v2 over Viper** -- Produces 313% smaller binary, doesn't force-lowercase keys (which breaks TOML spec). Better fit for single-binary distribution.
2. **zeebo/xxh3 over cespare/xxhash** -- PRD specifies XXH3, but cespare/xxhash implements XXH64. zeebo/xxh3 provides proper XXH3 in pure Go with SIMD optimizations.
3. **Go stdlib regexp only** -- RE2 engine guarantees O(n) matching time for untrusted input. All Gitleaks patterns adapted without lookaheads.
4. **malivvan/tree-sitter vs direct wazero** -- Needs evaluation. malivvan provides higher-level API but is pre-release (Jan 2025). Decision documented in T-042.
5. **BurntSushi/toml v1.5.0** -- Latest stable. MetaData.Undecoded() enables unknown-key detection.
6. **Bubble Tea v1.x** -- v2 still in RC as of Feb 2026. Stable v1.2+ recommended for production.

### Phase Index Files

Detailed phase-level documentation with Mermaid dependency graphs, implementation order, and tech stack summaries:
- [PHASE-2-INDEX.md](PHASE-2-INDEX.md) -- Profile System
- [PHASE-4-INDEX.md](PHASE-4-INDEX.md) -- Secret Redaction
- [PHASE-5-INDEX.md](PHASE-5-INDEX.md) -- Tree-Sitter Compression
- [PHASE-8-INDEX.md](PHASE-8-INDEX.md) -- Workflows


_Last updated: 2026-02-16 (Phase 1 complete)_
