# Harvx Task Progress Log

## Summary

| Status | Count |
|--------|-------|
| Completed | 19 |
| In Progress | 0 |
| Not Started | 76 |

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

### Phase 2: Profiles (T-016, T-017)

- **Status:** Partially Complete (T-016 and T-017 done)
- **Date:** 2026-02-22
- **Tasks:** 2

#### Features Implemented

| Feature | Task | Description |
|---------|------|-------------|
| Config Types | T-016 | `Config`, `Profile`, `RelevanceConfig`, `RedactionConfig` structs with `toml` tags |
| Default Profile | T-016 | `DefaultProfile()` with PRD Section 5.2 values; `defaultRelevanceTiers()` with PRD Section 5.3 glob patterns |
| TOML Loader | T-016 | `LoadFromFile()` and `LoadFromString()` via `BurntSushi/toml` v1.5.0; unknown-key warnings via `MetaData.Undecoded()` |
| Multi-Source Resolver | T-017 | 5-layer merge pipeline: defaults → global → repo/profile-file → env vars → CLI flags |
| Source Tracking | T-017 | `SourceMap` tracking which layer provided each config value |
| Target Presets | T-017 | `ApplyTargetPreset()` for claude/chatgpt/generic LLM targets |
| Env Var Parsing | T-017 | `buildEnvMap()` reads all `HARVX_*` env vars into koanf-compatible flat map |

#### Key Files Added

| Purpose | Location |
|---------|----------|
| Config struct types | `internal/config/types.go` |
| Default profile | `internal/config/defaults.go` |
| TOML loader | `internal/config/loader.go` |
| Source constants | `internal/config/sources.go` |
| Target presets | `internal/config/target.go` |
| Env var mapping | `internal/config/env.go` |
| Multi-source resolver | `internal/config/resolver.go` |
| Resolver tests | `internal/config/resolver_test.go` |
| Target tests | `internal/config/target_test.go` |
| Env tests | `internal/config/env_test.go` |
| Sources tests | `internal/config/sources_test.go` |
| Test fixtures | `testdata/config/{valid,minimal,invalid_syntax,unknown_keys,global,repo}.toml` |

#### New Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/BurntSushi/toml` | v1.5.0 | TOML v1.0 parsing with `MetaData.Undecoded()` |
| `github.com/knadh/koanf/v2` | v2.3.2 | Multi-source config merging engine |
| `github.com/knadh/koanf/providers/confmap` | v1.0.0 | In-memory map provider for koanf |

### T-016: Configuration Types, Defaults, and TOML Loading

- **Status:** Completed
- **Date:** 2026-02-22
- **What was built:**
  - `Config`, `Profile`, `RelevanceConfig`, `RedactionConfig` structs with `toml` struct tags
  - `DefaultProfile()` constructor with all PRD Section 5.2 defaults (output, format, max_tokens, tokenizer, compression, redaction, ignore)
  - `defaultRelevanceTiers()` with 6-tier glob patterns per PRD Section 5.3
  - `LoadFromFile(path string) (*Config, error)` and `LoadFromString(data, name string) (*Config, error)` using `BurntSushi/toml` v1.5.0
  - Unknown-key warning via `MetaData.Undecoded()` logged through slog (no error returned)
  - 4 test fixture TOML files (valid, minimal, invalid_syntax, unknown_keys)
  - 32+ new test functions across types_test.go, loader_test.go, and defaults_test.go
- **Files created/modified:**
  - `internal/config/types.go` -- Config, Profile, RelevanceConfig, RedactionConfig struct definitions
  - `internal/config/defaults.go` -- DefaultProfile() and defaultRelevanceTiers() constructors
  - `internal/config/loader.go` -- LoadFromFile, LoadFromString, warnUndecodedKeys
  - `internal/config/types_test.go` -- struct defaults and field validation tests
  - `internal/config/loader_test.go` -- TOML loading tests (all acceptance criteria)
  - `internal/config/defaults_test.go` -- exhaustive default value and tier pattern tests
  - `testdata/config/valid.toml` -- PRD example with default + finvault profiles
  - `testdata/config/minimal.toml` -- minimal [profile.default] fixture
  - `testdata/config/invalid_syntax.toml` -- malformed TOML for error testing
  - `testdata/config/unknown_keys.toml` -- extra keys for warning testing
  - `go.mod` / `go.sum` -- added github.com/BurntSushi/toml v1.5.0
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-017: Multi-Source Configuration Merging and Resolution

- **Status:** Completed
- **Date:** 2026-02-22
- **What was built:**
  - `Source` type (iota) with 5 levels: SourceDefault, SourceGlobal, SourceRepo, SourceEnv, SourceFlag
  - `SourceMap` type tracking per-key config origin
  - `ApplyTargetPreset(p *Profile, target string) error` -- applies claude/chatgpt/generic presets
  - `buildEnvMap()` -- reads HARVX_FORMAT, HARVX_MAX_TOKENS, HARVX_TOKENIZER, HARVX_OUTPUT, HARVX_TARGET, HARVX_COMPRESS, HARVX_REDACT env vars into flat koanf map
  - `Resolve(opts ResolveOptions) (*ResolvedConfig, error)` -- 5-layer pipeline with koanf confmap provider
  - `loadFileLayer()` / `extractProfileFlat()` / `flattenProfileRaw()` -- TOML raw-map parsing (only explicitly-set fields) for correct source attribution
  - `profileToFlatMap()` / `flatMapToProfile()` -- bidirectional Profile ↔ flat map conversion (used for defaults and preset layers)
  - `loadLayer()` -- merges flat map into koanf and marks all provided keys with their source
  - 9 env var constants (EnvProfile, EnvMaxTokens, EnvFormat, etc.)
  - Error for non-default profiles not found in any config file
  - 50+ test functions across 4 test files
- **Files created/modified:**
  - `internal/config/sources.go` -- Source iota + SourceMap type
  - `internal/config/target.go` -- ApplyTargetPreset() with claude/chatgpt/generic presets
  - `internal/config/env.go` -- env var constants + buildEnvMap()
  - `internal/config/resolver.go` -- Resolve(), ResolveOptions, ResolvedConfig, helpers
  - `internal/config/sources_test.go` -- Source.String() and precedence tests
  - `internal/config/target_test.go` -- preset tests for all valid targets + error cases
  - `internal/config/env_test.go` -- env var parsing tests for all HARVX_ vars
  - `internal/config/resolver_test.go` -- 25+ integration tests covering all 5 layers
  - `testdata/config/global.toml` -- test global config fixture
  - `testdata/config/repo.toml` -- test repo config fixture
  - `go.mod` -- promoted koanf/v2 and koanf/providers/confmap to direct dependencies
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-019: Profile Inheritance with Deep Merge

- **Status:** Completed
- **Date:** 2026-02-22
- **What was built:**
  - `ProfileResolution` struct with `Profile *Profile` and `Chain []string` fields for inheritance chain debugging
  - `ResolveProfile(name string, profiles map[string]*Profile) (*ProfileResolution, error)` — public entry point; resolves full inheritance chain, emits slog.Warn when depth > 3, emits slog.Debug on successful resolution
  - `resolveChain(name, profiles, visited)` — recursive DFS helper; detects circular/self-referential inheritance by tracking visited set; supports fresh-visited for implicit default base (avoids false positives when "default" appears in chain)
  - `lookupProfile(name, profiles)` — synthesizes built-in `DefaultProfile()` for "default" when absent from map
  - `mergeProfile(base, override *Profile) *Profile` — explicit per-field merge: strings (non-empty wins), ints (non-zero wins), booleans (override always wins), slices (non-empty override replaces base entirely), RelevanceConfig (per-tier), RedactionConfig (field-by-field); Extends always cleared; no mutation of inputs
  - `mergeString`, `mergeInt`, `mergeSlice`, `mergeRelevance`, `mergeRedactionConfig` — unexported merge helpers
  - Profiles without `extends` automatically get built-in defaults applied for unset fields
  - 30+ table-driven tests covering: base cases, multi-level chains (1-3 levels), chain tracking, error cases (missing profile, missing parent, circular 2-way, circular 3-way, self-referential), slice semantics, relevance tier merge, boolean override, RedactionConfig field merge, TOML fixture integration, immutability
- **Files created/modified:**
  - `internal/config/profile.go` -- ProfileResolution, ResolveProfile, resolveChain, lookupProfile
  - `internal/config/merge.go` -- mergeProfile and all merge helpers
  - `internal/config/profile_test.go` -- 30+ tests
  - `testdata/config/inheritance.toml` -- multi-level fixture (default, base, child, grandchild, deep, no_extends, custom_tiers, custom_redaction)
  - `testdata/config/circular.toml` -- circular and self-referential fixture (a -> b -> a, self-ref)
  - `docs/tasks/PROGRESS.md` -- updated summary
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-018: Configuration File Auto-Detection and Discovery

- **Status:** Completed
- **Date:** 2026-02-22
- **What was built:**
  - `DiscoverRepoConfig(startDir string) (string, error)` — walks up the directory tree from `startDir` resolving symlinks first, checking for `harvx.toml` at each level, stopping at `.git` boundary or after 20 levels (max depth), returning the first config found or empty string
  - `DiscoverGlobalConfig() (string, error)` — returns XDG-compatible global config path: `$XDG_CONFIG_HOME/harvx/config.toml`, `~/.config/harvx/config.toml` (Linux/macOS), or `%APPDATA%\harvx\config.toml` (Windows); returns empty string (no error) when file absent
  - `globalConfigDir()` unexported helper isolating platform-specific path logic
  - Resolver Layer 2 updated to use `DiscoverGlobalConfig()` instead of hard-coded `os.UserHomeDir()` + `filepath.Join`
  - Resolver Layer 3 updated to use `DiscoverRepoConfig(targetDir)` instead of direct `filepath.Join(targetDir, "harvx.toml")`
  - 29 tests covering all acceptance criteria, edge cases, and resolver integration
- **Files created/modified:**
  - `internal/config/discover.go` -- `DiscoverRepoConfig`, `DiscoverGlobalConfig`, `globalConfigDir`
  - `internal/config/discover_test.go` -- 29 tests: start dir, parent dir, two levels up, max depth, .git boundary, symlink resolution, permission-denied, XDG env, table-driven, resolver integration
  - `internal/config/resolver.go` -- Layer 2 and Layer 3 replaced to call discovery functions
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

---

## In Progress Tasks

_None currently_

---

## Not Started Tasks

### Phase 2: Profiles (T-016 to T-025)

- **Status:** In Progress
- **Tasks:** 10 (8 Must Have, 2 Should Have)
- **Estimated Effort:** 75-105 hours
- **PRD Roadmap:** Weeks 4-6

#### Task List

| Task | Name | Priority | Effort | Status |
|------|------|----------|--------|--------|
| T-016 | Configuration Types, Defaults, and TOML Loading | Must Have | Medium (8-12hrs) | Completed |
| T-017 | Multi-Source Configuration Merging and Resolution | Must Have | Large (14-20hrs) | Completed |
| T-018 | Configuration File Auto-Detection and Discovery | Must Have | Small (3-5hrs) | Completed |
| T-019 | Profile Inheritance with Deep Merge | Must Have | Medium (8-12hrs) | Completed |
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
