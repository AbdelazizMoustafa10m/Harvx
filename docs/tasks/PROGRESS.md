# Harvx Task Progress Log

## Summary

| Status | Count |
|--------|-------|
| Completed | 7 |
| In Progress | 0 |
| Not Started | 88 |

---

## Not Started Tasks

### Phase 1: Foundation (T-001 to T-015)

- **Status:** Not Started
- **Tasks:** 15 (14 Must Have, 1 Should Have)
- **Estimated Effort:** 68-105 hours (~12-18 person-days)
- **PRD Roadmap:** Weeks 1-3

#### Tech Stack (Phase 1)

| Package | Purpose |
|---------|---------|
| Go 1.24+ | Language runtime |
| log/slog (stdlib) | Structured logging |
| spf13/cobra v1.8.x | CLI framework |
| sabhiram/go-gitignore v1.1.0 | .gitignore parsing |
| bmatcuk/doublestar v4.x | Glob pattern matching |
| x/sync/errgroup | Bounded parallel execution |
| stretchr/testify v1.9+ | Testing assertions |

#### Task List

| Task | Name | Priority | Effort | Status |
|------|------|----------|--------|--------|
| T-001 | Go Project Initialization & Directory Structure | Must Have | Small (2-4hrs) | Completed |
| T-002 | Makefile Setup | Must Have | Small (2-4hrs) | Completed |
| T-003 | Central Data Types (FileDescriptor & Pipeline DTOs) | Must Have | Small (2-4hrs) | Completed |
| T-004 | Structured Logging with slog | Must Have | Small (2-4hrs) | Completed |
| T-005 | Cobra CLI Framework & Root Command | Must Have | Medium (6-8hrs) | Completed |
| T-006 | Version Command & Build Info | Must Have | Small (2-4hrs) | Completed |
| T-007 | Global Flags Implementation | Must Have | Medium (6-8hrs) | Completed |
| T-008 | Generate Subcommand (harvx generate / harvx gen) | Must Have | Medium (6-10hrs) | Not Started |
| T-009 | Shell Completions (harvx completion) | Should Have | Small (2-4hrs) | Not Started |
| T-010 | Exit Code Handling | Must Have | Small (2-4hrs) | Not Started |
| T-011 | .gitignore Parsing & Matching | Must Have | Medium (6-10hrs) | Not Started |
| T-012 | Default Ignore Patterns & .harvxignore Support | Must Have | Medium (6-8hrs) | Not Started |
| T-013 | Binary File Detection & Large File Skipping | Must Have | Small (3-5hrs) | Not Started |
| T-014 | Extension/Pattern Filtering, --git-tracked-only & Symlinks | Must Have | Medium (8-12hrs) | Not Started |
| T-015 | Parallel File Discovery Engine (Walker with errgroup) | Must Have | Large (14-20hrs) | Not Started |

**Deliverable:** `harvx` produces correct Markdown output for any repository with default settings.

---

### Phase 2: Intelligence -- Profiles (T-016 to T-025)

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

### Phase 2: Intelligence -- Relevance & Tokens (T-026 to T-033)

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

### Phase 3: Security (T-034 to T-041)

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

### Phase 3: Compression (T-042 to T-050)

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

### Phase 4: Output & Rendering (T-051 to T-058)

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

### Phase 4: State & Diff (T-059 to T-065)

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

### Phase 5: Workflows (T-066 to T-078)

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

### Phase 5: Interactive TUI (T-079 to T-087)

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

### Phase 6: Polish & Distribution (T-088 to T-094)

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

## In Progress Tasks

_None currently_

---

## Completed Tasks

### T-001: Go Project Initialization & Directory Structure

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- Go module initialized (`github.com/harvx/harvx`, Go 1.24)
- Minimal entry point `cmd/harvx/main.go` that prints "harvx" and exits 0
- Full directory structure with `.gitkeep` placeholders for all 13 `internal/` packages
- Support directories: `grammars/`, `templates/`, 4 `testdata/` subdirectories
- `.editorconfig` with tabs for Go, UTF-8, LF line endings
- MIT `LICENSE` (2026 Harvx Contributors)
- `README.md` with project description and "Under Development" badge

**Files created/modified:**

- `go.mod` - Module declaration
- `cmd/harvx/main.go` - Entry point
- `internal/{cli,config,discovery,relevance,tokenizer,security,compression,output,diff,workflows,tui,server,pipeline}/.gitkeep` - Package placeholders
- `grammars/.gitkeep`, `templates/.gitkeep` - Support directories
- `testdata/{sample-repo,secrets,monorepo,expected-output}/.gitkeep` - Test fixture directories
- `.editorconfig` - Editor configuration
- `LICENSE` - MIT license
- `README.md` - Project readme
- `.gitignore` - Added `*.wasm`, `.harvx/`, fixed `/harvx` path-rooted pattern

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass

---

### T-002: Makefile Setup

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- Comprehensive Makefile with all standard targets: `build`, `run`, `test`, `test-verbose`, `test-cover`, `lint`, `fmt`, `vet`, `tidy`, `clean`, `install`, `snapshot`, `help`, `all`
- Build-time metadata injection via `-ldflags` (version, commit, date, goVersion into `main` package)
- `CGO_ENABLED=0` for pure Go cross-compilation
- `golangci-lint` integration with helpful install message if not found
- `.golangci.yml` configuration with sensible defaults (errcheck, staticcheck, unused, govet, ineffassign, gosimple)
- Build metadata variables (`version`, `commit`, `date`, `goVersion`) added to `cmd/harvx/main.go`
- Trivial test (`main_test.go`) verifying ldflags defaults are non-empty
- `.PHONY` declarations for all non-file targets
- `make help` lists all targets with descriptions and build metadata

**Files created/modified:**

- `Makefile` - Updated with all required targets and ldflags injection
- `.golangci.yml` - New linter configuration
- `cmd/harvx/main.go` - Added build-time ldflags variables
- `cmd/harvx/main_test.go` - New test for build metadata defaults

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test -race ./...` - pass (1 test)
- `go mod tidy` - pass (no drift)
- `make build` - produces `bin/harvx` with ldflags embedded
- `make test` - pass
- `make help` - lists all 14 targets
- `make clean` - removes bin/ directory

---

### T-003: Central Data Types (FileDescriptor & Pipeline DTOs)

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `FileDescriptor` struct with all 13 fields: Path, AbsPath, Size, Tier, TokenCount, ContentHash, Content, IsCompressed, Redactions, Language, IsSymlink, IsBinary, Error
- `ExitCode` type with constants: ExitSuccess (0), ExitError (1), ExitPartial (2)
- `OutputFormat` string-based enum: FormatMarkdown ("markdown"), FormatXML ("xml")
- `LLMTarget` string-based enum: TargetClaude ("claude"), TargetChatGPT ("chatgpt"), TargetGeneric ("generic")
- `DiscoveryResult` struct: Files slice, TotalFound, TotalSkipped, SkipReasons map
- `DefaultTier` constant (2) per PRD Section 5.3
- `FileDescriptor.IsValid()` helper method
- All types have JSON struct tags; Error field uses `json:"-"`
- Comprehensive GoDoc comments on all exported types, fields, and methods
- Zero external dependencies (stdlib only)

**Files created/modified:**

- `internal/pipeline/types.go` - Central data types (new)
- `internal/pipeline/types_test.go` - 13 test functions covering constants, zero values, JSON round-trips, validation (new)
- `internal/pipeline/.gitkeep` - To be deleted (superseded by real files)

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./internal/pipeline/...` - pass (13 tests)

---

### T-004: Structured Logging with slog

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `SetupLogging(level, format)` function configures the global slog default logger with text or JSON handler, directing all output to os.Stderr
- `SetupLoggingWithWriter(level, format, writer)` variant for testing with custom writer
- `ResolveLogLevel(verbose, quiet)` resolves log level with priority: HARVX_DEBUG=1 > --verbose > --quiet > default(info)
- `ResolveLogFormat()` reads HARVX_LOG_FORMAT env var, returns "json" or "text"
- `NewLogger(component)` returns child logger with "component" attribute for subsystem identification
- JSON format via `slog.NewJSONHandler`, text format via `slog.NewTextHandler`
- Case-insensitive format matching (e.g., "JSON", "Json", "json" all work)
- All functions are idempotent and safe to call multiple times
- Comprehensive doc comments on all exported functions
- Zero imports from other internal packages (no circular dependencies)
- Added `stretchr/testify v1.9+` as first external dependency in go.mod

**Files created/modified:**

- `internal/config/logging.go` - Logging setup functions (new)
- `internal/config/logging_test.go` - 11 test functions covering level resolution, format selection, JSON/text output, stderr routing, idempotency, component loggers, level inheritance (new)
- `go.mod` - Added stretchr/testify dependency

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./internal/config/...` - pass (11 test functions)
- `go mod tidy` - pass

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
- [PHASE-3-SECURITY-INDEX.md](PHASE-3-SECURITY-INDEX.md) -- Secret Redaction
- [PHASE-3-COMPRESSION-INDEX.md](PHASE-3-COMPRESSION-INDEX.md) -- Tree-Sitter Compression
- [PHASE-5-INDEX.md](PHASE-5-INDEX.md) -- Workflows

### T-005: Cobra CLI Framework & Root Command

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- Cobra CLI framework integrated with root `harvx` command
- Root command with `Use: "harvx"`, `Short: "Harvest your context."`, and multi-line Long description
- `SilenceUsage: true` and `SilenceErrors: true` for clean error handling
- `PersistentPreRunE` initializes logging via T-004's `config.ResolveLogLevel` / `config.ResolveLogFormat` / `config.SetupLogging`
- `--verbose` / `-v` and `--quiet` / `-q` persistent flags registered on root command
- `Execute()` function returns `pipeline.ExitSuccess` (0) or `pipeline.ExitError` (1) exit codes from T-003
- `RootCmd()` accessor for testing and subcommand registration
- `cmd/harvx/main.go` simplified to `os.Exit(cli.Execute())`
- 9 unit tests covering command properties, flags, help output, unknown flag handling

**Files created/modified:**

- `internal/cli/root.go` - Root command definition with Cobra (new)
- `internal/cli/root_test.go` - 9 unit tests (new)
- `cmd/harvx/main.go` - Rewired to use `cli.Execute()`
- `go.mod` / `go.sum` - Added `spf13/cobra v1.10.2`, `spf13/pflag v1.0.9`, `inconshreveable/mousetrap v1.1.0`

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./...` - pass (9 cli tests + existing tests)
- `go mod tidy` - pass (no drift)

---

### T-006: Version Command & Build Info

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `internal/buildinfo` package with exported `Version`, `Commit`, `Date`, `GoVersion` variables (defaults: "dev"/"unknown") and `OS()`/`Arch()` helpers using `runtime.GOOS`/`runtime.GOARCH`
- `harvx version` subcommand registered on root command via `init()`
- Human-readable output format: version header + indented commit, built, go version, os/arch
- `harvx version --json` outputs pretty-printed JSON with all 6 keys (version, commit, date, goVersion, os, arch)
- Makefile ldflags updated from `main` package to `internal/buildinfo` package, variable names changed from lowercase to exported (e.g., `Version` not `version`)
- `cmd/harvx/main.go` cleaned up -- removed old ldflags variables, now just calls `cli.Execute()`
- `cmd/harvx/main_test.go` updated to reference `buildinfo` package instead of removed `main` vars
- 9 unit tests covering: subcommand registration, properties, --json flag, human output format, JSON output format/keys, OS/arch in output, default values, versionInfo struct round-trip

**Files created/modified:**

- `internal/buildinfo/buildinfo.go` - Build-time variables and OS/Arch helpers (new)
- `internal/cli/version.go` - Version subcommand with human + JSON output (new)
- `internal/cli/version_test.go` - 9 unit tests (new)
- `Makefile` - Updated LDFLAGS_PKG to `internal/buildinfo`, capitalized variable names
- `cmd/harvx/main.go` - Removed old ldflags variables
- `cmd/harvx/main_test.go` - Updated to use `buildinfo` package

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./...` - pass (all packages)
- `go mod tidy` - pass (no drift)
- `make build && bin/harvx version` - shows injected version info
- `bin/harvx version --json` - outputs valid JSON with all expected keys

---

### T-007: Global Flags Implementation

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `FlagValues` struct in `internal/config/flags.go` collecting all 17 global persistent flags
- `BindFlags(cmd)` function that registers all flags as Cobra persistent flags on the root command
- `ValidateFlags(fv, cmd)` function that validates all flag values:
  - `--verbose` and `--quiet` mutual exclusion check
  - `--dir` existence and is-directory validation
  - `--format` allowed values validation (markdown, xml)
  - `--target` allowed values validation (claude, chatgpt, generic)
  - `--filter` leading-dot stripping normalization
  - `--skip-large-files` human-readable size parsing
- `ParseSize()` function handling KB, MB, GB suffixes (case-insensitive), plain byte values, and float values
- Environment variable overrides for key flags (HARVX_DIR, HARVX_OUTPUT, HARVX_FORMAT, HARVX_TARGET, HARVX_VERBOSE, HARVX_QUIET) with explicit flag taking priority
- Root command updated to use `config.BindFlags` for flag registration and `config.ValidateFlags` in `PersistentPreRunE`
- `GlobalFlags()` accessor for subcommands to access shared configuration
- 30+ unit tests covering: defaults, mutual exclusion, path validation, format/target validation, filter normalization, size parsing, env overrides, boolean flags, include/exclude patterns

**Files created/modified:**

- `internal/config/flags.go` - FlagValues struct, BindFlags, ValidateFlags, ParseSize (new)
- `internal/config/flags_test.go` - Comprehensive unit tests (new)
- `internal/cli/root.go` - Rewired to use config.BindFlags and config.ValidateFlags
- `internal/cli/root_test.go` - Updated with tests for all new flags

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./...` - pass (all packages)
- `go mod tidy` - pass (no drift)

---

_Last updated: 2026-02-16 (T-007)_
