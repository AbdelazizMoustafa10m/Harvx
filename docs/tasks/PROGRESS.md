# Harvx Task Progress Log

## Summary

| Status | Count |
|--------|-------|
| Completed | 15 |
| In Progress | 0 |
| Not Started | 80 |

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
| T-008 | Generate Subcommand (harvx generate / harvx gen) | Must Have | Medium (6-10hrs) | Completed |
| T-009 | Shell Completions (harvx completion) | Should Have | Small (2-4hrs) | Completed |
| T-010 | Exit Code Handling | Must Have | Small (2-4hrs) | Completed |
| T-011 | .gitignore Parsing & Matching | Must Have | Medium (6-10hrs) | Completed |
| T-012 | Default Ignore Patterns & .harvxignore Support | Must Have | Medium (6-8hrs) | Completed |
| T-013 | Binary File Detection & Large File Skipping | Must Have | Small (3-5hrs) | Completed |
| T-014 | Extension/Pattern Filtering, --git-tracked-only & Symlinks | Must Have | Medium (8-12hrs) | Completed |
| T-015 | Parallel File Discovery Engine (Walker with errgroup) | Must Have | Large (14-20hrs) | Completed |

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

### T-008: Generate Subcommand (harvx generate / harvx gen)

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `harvx generate` subcommand registered with alias `gen`
- Running `harvx` (no subcommand) delegates to the generate logic via `rootCmd.RunE`
- `--preview` flag on generate command (stub for now)
- `internal/pipeline/pipeline.go` with `Run(ctx, cfg)` stub that logs configuration at info/debug levels
- `context.Context` threaded from `cmd.Context()` into the pipeline for cancellation support
- Generate command inherits all global flags from T-007
- `harvx generate --help` and `harvx help generate` show descriptive help text
- 11 unit tests covering: command registration, alias, properties, --preview flag, global flag inheritance, help output, alias resolution, pipeline invocation, root delegation, and context cancellation

**Files created/modified:**

- `internal/pipeline/pipeline.go` - Pipeline Run stub (new)
- `internal/cli/generate.go` - Generate subcommand definition (new)
- `internal/cli/generate_test.go` - 11 unit tests (new)
- `internal/cli/root.go` - Added RunE to delegate to generate when no subcommand is given

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./...` - pass (all packages)
- `go mod tidy` - pass (no drift)

---

### T-009: Shell Completions (harvx completion)

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `harvx completion` subcommand generating shell completion scripts for Bash, Zsh, Fish, and PowerShell
- Rich Long help text with installation instructions for each shell (Bash, Zsh, Fish, PowerShell)
- When run with no arguments, displays help with installation instructions (exit 0)
- Uses `GenBashCompletionV2` (not deprecated GenBashCompletion) for modern Bash completion with descriptions
- `cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs)` for strict argument validation
- Flag completion functions registered on root command for `--format` (markdown, xml) and `--target` (claude, chatgpt, generic)
- Both completion functions return `cobra.ShellCompDirectiveNoFileComp` to suppress file path suggestions
- 12 test functions covering: command registration, properties, ValidArgs, all 4 shell script generation, no-args help display, invalid shell rejection, too many args rejection, Long help content verification, format flag completion, target flag completion, subcommand name registration

**Files created/modified:**

- `internal/cli/completion.go` - Completion subcommand with help text and shell script generation (new)
- `internal/cli/completion_test.go` - 12 test functions (new)
- `internal/cli/root.go` - Added `completeFormat` and `completeTarget` functions, registered flag completion functions in `init()`
- `docs/tasks/PROGRESS.md` - Updated with T-009 completion entry

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./...` - pass (all packages)
- `go mod tidy` - pass (no drift)

---

### T-010: Exit Code Handling

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `HarvxError` custom error type in `internal/pipeline/errors.go` carrying exit code, message, and optional underlying error
- `HarvxError` implements `error` interface via `Error()` and supports `errors.Is`/`errors.As` via `Unwrap()`
- Three convenience constructors: `NewError` (code 1), `NewPartialError` (code 2), `NewRedactionError` (code 1, no underlying error)
- `extractExitCode()` function in `internal/cli/root.go` that determines exit code from error type using `errors.As`
- `Execute()` updated to log errors to stderr via `slog.Error` before returning the extracted exit code
- 19 unit tests for `HarvxError` covering: constructor codes, message formatting, `Unwrap`, `errors.Is`, `errors.As`, error interface compliance, stdlib error wrapping, message preservation
- 10 unit tests for `extractExitCode` covering: nil (0), generic error (1), HarvxError code 1, HarvxError code 2, redaction error, wrapped HarvxError, deeply wrapped HarvxError

**Files created/modified:**

- `internal/pipeline/errors.go` - HarvxError type and constructors (new)
- `internal/pipeline/errors_test.go` - 19 unit tests for error type (new)
- `internal/cli/root.go` - Updated Execute() with extractExitCode() and slog.Error logging
- `internal/cli/root_test.go` - Added 10 tests for extractExitCode function

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./...` - pass (all packages)
- `go mod tidy` - pass (no drift)

### T-011: .gitignore Parsing & Matching

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `GitignoreMatcher` type in `internal/discovery/gitignore.go` that loads and evaluates `.gitignore` patterns hierarchically
- `NewGitignoreMatcher(rootDir)` constructor that walks the directory tree, discovers all `.gitignore` files, and compiles patterns using `sabhiram/go-gitignore`
- `IsIgnored(path, isDir)` method that checks patterns from root down to the file's parent directory
- Nested `.gitignore` support: each directory's `.gitignore` applies only to files within its subtree
- Parent `.gitignore` rules are inherited by all subdirectories
- Correct handling of negation patterns (`!important.log`), directory-only patterns (`build/`), and doublestar patterns (`**/*.tmp`)
- Graceful handling of missing `.gitignore` files (no error, IsIgnored returns false)
- `.git/` directory is always skipped during discovery
- Path normalization: leading `./`, OS-native separators, and trailing `/` for directories
- `PatternCount()` diagnostic method for logging
- Performance: O(patterns) per path check, not O(files)
- Test fixtures under `testdata/gitignore/` with 5 scenarios: root (nested), negation, comments, deep nesting, empty (no gitignore)
- 16 test functions + 1 benchmark covering all acceptance criteria edge cases

**Files created/modified:**

- `go.mod` - Added `sabhiram/go-gitignore` dependency
- `internal/discovery/gitignore.go` - GitignoreMatcher implementation (new)
- `internal/discovery/gitignore_test.go` - 16 test functions + 1 benchmark (new)
- `testdata/gitignore/root/.gitignore` - Root fixture with wildcard, directory, and doublestar patterns
- `testdata/gitignore/root/src/.gitignore` - Nested fixture with `*.generated.go` and `vendor/`
- `testdata/gitignore/root/README.md` - Sample file for fixture
- `testdata/gitignore/root/src/main.go` - Sample file for fixture
- `testdata/gitignore/negation/.gitignore` - Negation pattern fixture (`!important.log`)
- `testdata/gitignore/comments/.gitignore` - Comments and blank lines fixture
- `testdata/gitignore/deep/.gitignore` - Root of deep nesting fixture
- `testdata/gitignore/deep/a/b/.gitignore` - Deeply nested `.gitignore`
- `testdata/gitignore/deep/a/b/c/file.txt` - Deep nested sample file
- `testdata/gitignore/empty/file.txt` - Empty directory fixture (no .gitignore)
- `docs/tasks/PROGRESS.md` - Updated with T-011 completion entry

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./internal/discovery/...` - pass (16 tests + 1 benchmark)
- `go mod tidy` - pass

### T-012: Default Ignore Patterns & .harvxignore Support

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `Ignorer` interface in `internal/discovery/ignore.go` -- the key abstraction for all ignore-pattern matchers
- `CompositeIgnorer` that chains multiple `Ignorer` implementations (returns true if ANY source matches)
- `DefaultIgnorePatterns` exported slice with all 41 default patterns from the PRD (directories, env files, certificates, sensitive names, lock files, compiled artifacts, OS/editor files)
- `SensitivePatterns` exported subset for security-sensitive override warnings
- `DefaultIgnoreMatcher` compiles default patterns into a gitignore matcher at construction time
- `IsSensitivePath()` utility function with pre-compiled matcher for override warning checks
- `HarvxignoreMatcher` loads and evaluates `.harvxignore` files with full hierarchical support (same model as `GitignoreMatcher`)
- All three matchers (`GitignoreMatcher`, `DefaultIgnoreMatcher`, `HarvxignoreMatcher`) plus `CompositeIgnorer` implement `Ignorer` with compile-time interface compliance checks
- Nil ignorer filtering in `NewCompositeIgnorer` for safe construction
- Test fixtures under `testdata/harvxignore/` (basic, negation, empty)
- Comprehensive test coverage: 13 test functions for defaults, 8 test functions for composite ignorer, 15 test functions for harvxignore, plus 2 benchmarks

**Files created/modified:**

- `internal/discovery/ignore.go` - Ignorer interface + CompositeIgnorer (new)
- `internal/discovery/defaults.go` - DefaultIgnorePatterns, SensitivePatterns, DefaultIgnoreMatcher, IsSensitivePath (new)
- `internal/discovery/harvxignore.go` - HarvxignoreMatcher with hierarchical .harvxignore support (new)
- `internal/discovery/defaults_test.go` - 13 test functions + 1 benchmark (new)
- `internal/discovery/ignore_test.go` - 8 test functions (new)
- `internal/discovery/harvxignore_test.go` - 15 test functions + 1 benchmark (new)
- `internal/discovery/gitignore.go` - Added compile-time Ignorer interface compliance check
- `testdata/harvxignore/basic/.harvxignore` - Basic patterns fixture (new)
- `testdata/harvxignore/negation/.harvxignore` - Negation patterns fixture (new)
- `testdata/harvxignore/empty/file.txt` - Empty directory fixture (new)

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./internal/discovery/...` - pass (all tests)
- `go mod tidy` - pass

### T-013: Binary File Detection & Large File Skipping

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `IsBinary(path string) (bool, error)` function in `internal/discovery/binary.go` that reads the first 8KB of a file and checks for null bytes, matching Git's approach
- `IsLargeFile(path string, maxBytes int64) (bool, int64, error)` function that uses `os.Stat` to check file size without reading content
- `BinaryDetectionBytes` constant (8192) and `DefaultMaxFileSize` constant (1,048,576 bytes = 1MB) exported for use by the pipeline
- Uses `bytes.IndexByte` for assembly-optimized null byte detection
- Both functions are safe for concurrent use (no shared mutable state)
- Proper error wrapping with `fmt.Errorf("context: %w", err)` preserving `os.ErrPermission` and `os.ErrNotExist` sentinels
- Empty files are explicitly not considered binary
- Edge cases handled: permission denied, file not found, symlink targets, null byte boundary at exactly 8KB
- Test fixtures under `testdata/binary-detection/` (text.txt, binary.bin, empty.txt)
- 10 test functions with 13+ table-driven subtests plus 3 testdata fixture subtests
- 4 benchmarks covering large file, small file, binary file, and IsLargeFile

**Files created/modified:**

- `internal/discovery/binary.go` - IsBinary and IsLargeFile implementation (new)
- `internal/discovery/binary_test.go` - Comprehensive tests and benchmarks (new)
- `testdata/binary-detection/text.txt` - Plain text fixture (new)
- `testdata/binary-detection/binary.bin` - Binary fixture with PNG-like header (generated by test helper)
- `testdata/binary-detection/empty.txt` - Empty file fixture (generated by test helper)
- `docs/tasks/PROGRESS.md` - Updated with T-013 completion entry

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./internal/discovery/...` - pass (all tests)
- `go mod tidy` - pass

### T-014: Extension/Pattern Filtering, --git-tracked-only & Symlinks

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `PatternFilter` type in `internal/discovery/filter.go` implementing include/exclude/extension-based filtering with `bmatcuk/doublestar` v4 glob patterns
- Include patterns and extension filters combined with OR logic; exclude patterns take precedence (exclude always wins)
- Extension matching is case-insensitive with leading dot normalization (`.ts` → `ts`)
- `HasFilters()` method for pass-through detection when no filters are configured
- `GitTrackedFiles(root)` function in `internal/discovery/git_tracked.go` that runs `git ls-files` and returns a path set for `--git-tracked-only` mode
- `SymlinkResolver` type in `internal/discovery/symlink.go` with two-step Resolve/MarkVisited design for loop detection
- `IsSymlink()` standalone helper using `os.Lstat` for symlink detection
- Thread-safe `SymlinkResolver` with `sync.RWMutex` for concurrent discovery
- Dangling symlink detection via `filepath.EvalSymlinks` error checking
- Reset capability for restarting discovery passes

**Files created/modified:**

- `go.mod` / `go.sum` - Added `bmatcuk/doublestar/v4` v4.10.0
- `internal/discovery/filter.go` - PatternFilter with doublestar glob matching (new)
- `internal/discovery/filter_test.go` - 14 test functions + 1 benchmark (new)
- `internal/discovery/git_tracked.go` - GitTrackedFiles implementation (new)
- `internal/discovery/git_tracked_test.go` - 13 subtests covering real git repos, empty repos, non-git dirs (new)
- `internal/discovery/symlink.go` - SymlinkResolver and IsSymlink (new)
- `internal/discovery/symlink_test.go` - 19 test functions + 4 benchmarks covering loops, dangling, chains, concurrency (new)

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./...` - pass (all packages)
- `go mod tidy` - pass (no drift)

### T-015: Parallel File Discovery Engine (Walker with errgroup)

- **Status:** Completed
- **Date:** 2026-02-16

**What was built:**

- `Walker` type in `internal/discovery/walker.go` with `Walk(ctx, WalkerConfig)` method implementing two-phase parallel file discovery
- `WalkerConfig` struct accepting: Root, GitignoreMatcher, HarvxignoreMatcher, DefaultIgnorer, PatternFilter, GitTrackedOnly, SkipLargeFiles, Concurrency
- Phase 1 (walking): `filepath.WalkDir` traverses the tree, applying composite ignore rules, binary detection, large file skipping, pattern filters, symlink handling, and git-tracked-only checks; skips directories early via `fs.SkipDir`
- Phase 2 (content loading): `errgroup.WithContext()` with `SetLimit(cfg.Concurrency)` reads file contents in parallel with bounded concurrency; per-file read errors captured in `FileDescriptor.Error` (non-fatal)
- Results sorted alphabetically by path for deterministic output
- `context.Context` cancellation stops walk and all workers promptly
- `readFile()` helper using `os.ReadFile` for simple, correct content loading
- Representative `testdata/sample-repo/` fixture with: Go files, TypeScript files, `.gitignore` (ignores dist/, node_modules/), `.harvxignore` (ignores docs/internal/), build artifacts, and node_modules
- Added `golang.org/x/sync v0.19.0` dependency for errgroup
- 24 unit tests + 2 benchmarks covering: basic discovery, sorted output, content loading, .git skipping, .gitignore respect, .harvxignore respect, default ignorer, binary skipping, large file skipping, extension filter, include/exclude patterns, empty directory, non-existent directory, context cancellation/timeout, per-file read errors, discovery result stats, file descriptor fields, concurrency modes, sample-repo integration, multiple ignore sources, SkipLargeFiles=0 disabled

**Files created/modified:**

- `go.mod` / `go.sum` - Added `golang.org/x/sync v0.19.0`
- `internal/discovery/walker.go` - Walker implementation with two-phase parallel discovery (new)
- `internal/discovery/walker_test.go` - 24 test functions + 2 benchmarks (new)
- `testdata/sample-repo/.gitignore` - Ignores dist/, node_modules/, *.log (new)
- `testdata/sample-repo/.harvxignore` - Ignores docs/internal/ (new)
- `testdata/sample-repo/main.go` - Sample Go file (new)
- `testdata/sample-repo/README.md` - Sample markdown (new)
- `testdata/sample-repo/src/app.ts` - Sample TypeScript (new)
- `testdata/sample-repo/src/utils.ts` - Sample TypeScript (new)
- `testdata/sample-repo/src/test.spec.ts` - Sample test file (new)
- `testdata/sample-repo/dist/bundle.js` - Build artifact to be ignored (new)
- `testdata/sample-repo/node_modules/pkg/index.js` - Dependency to be ignored (new)
- `testdata/sample-repo/docs/internal/notes.md` - Doc to be harvxignored (new)

**Verification:**

- `go build ./cmd/harvx/` - pass
- `go vet ./...` - pass
- `go test ./...` - pass (all packages, 24 walker tests)
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


_Last updated: 2026-02-16 (T-015)_
