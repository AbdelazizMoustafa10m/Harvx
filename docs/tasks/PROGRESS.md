# Harvx Task Progress Log

## Summary

| Status | Count |
|--------|-------|
| Completed | 32 |
| In Progress | 0 |
| Not Started | 63 |

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

### Phase 2: Profiles (T-016 to T-025)

- **Status:** Completed
- **Date:** 2026-02-22
- **Tasks Completed:** 10 tasks

#### Features Implemented

| Feature | Tasks | Description |
| ------- | ----- | ----------- |
| Config types & TOML loading | T-016 | `Config`, `Profile`, `RelevanceConfig`, `RedactionConfig` structs; `LoadFromFile`/`LoadFromString` with unknown-key warnings via `MetaData.Undecoded()` |
| Multi-source config resolution | T-017 | `Resolve()` 5-layer koanf pipeline (defaults → global → repo → env → flags); `Source` iota + `SourceMap` for per-key origin tracking; `ApplyTargetPreset` for claude/chatgpt/generic targets |
| Config file auto-detection | T-018 | `DiscoverRepoConfig` (walks up to `.git` boundary, max 20 levels); `DiscoverGlobalConfig` (XDG-compatible: `$XDG_CONFIG_HOME`, `~/.config`, `%APPDATA%`) |
| Profile inheritance & deep merge | T-019 | `ResolveProfile` with DFS cycle detection and `slog.Warn` at depth > 3; `mergeProfile` per-field merge (strings, ints, bools, slices, `RelevanceConfig`, `RedactionConfig`); inputs never mutated |
| Validation & lint engine | T-020 | `Validate()` collects all hard errors (invalid format/tokenizer/target, bad globs, circular inheritance) and warnings (overlapping tiers, contradictory priority_files); `Lint()` adds codes `unreachable-tier`, `no-ext-match`, `complexity` |
| Framework profile templates | T-021 | 6 TOML templates embedded via `//go:embed`: `base`, `nextjs`, `go-cli`, `python-django`, `rust-cargo`, `monorepo`; `GetTemplate` validates name against allowlist (path traversal prevention); `RenderTemplate` substitutes `{{project_name}}` |
| Profile CLI: init, list, show | T-022 | `harvx profiles list` (tabwriter NAME/SOURCE/EXTENDS/DESCRIPTION); `profiles init` (writes from template, `--template`/`--output`/`--yes`); `profiles show` (annotated TOML with `# source` comments or `--json`); `ShowProfile`/`ShowProfileJSON` in `config/show.go` |
| Profile CLI: lint, explain | T-023 | `profiles lint` (groups by severity, exits 1 on errors, `--profile` filter); `profiles explain` (11-step pipeline simulation via `ExplainFile`: default ignores → profile ignores → include → priority → tiers 0–5; `TraceStep`/`ExplainResult` structs; glob expansion via `doublestar.Glob`) |
| Config debug command | T-024 | `harvx config debug` with `--json`/`--profile`; `BuildDebugOutput` discovers config files, reads all `HARVX_*` env vars, resolves full 5-layer config; `FormatDebugOutput` tabwriter report; `sourceDetailLabel` generates env/flag attribution |
| Integration & golden tests | T-025 | `testutil.Golden` helper with `-update` flag; 8 end-to-end scenario tests; `FuzzConfigParse`/`FuzzValidate` fuzz targets; 6 benchmarks (`BenchmarkConfigResolve/{defaults-only,single-file,multi-source,ten-profiles}`, `BenchmarkConfigValidate/{clean,complex}`); 13 CLI integration tests |

#### Key Technical Decisions

1. **koanf v2 over Viper** -- 313% smaller binary, preserves TOML key casing (Viper force-lowercases keys)
2. **BurntSushi/toml `MetaData.Undecoded()`** -- emits `slog.Warn` for unknown keys without returning an error, allowing forward-compatible configs
3. **Go stdlib `regexp` (RE2)** -- O(n) matching guarantees for untrusted input; all patterns avoid lookaheads
4. **Allowlist in `GetTemplate`** -- validates template name against known set before `embed.FS` access, preventing path traversal

#### Key Files Reference

| Purpose | Location |
| ------- | -------- |
| Config, Profile, RelevanceConfig, RedactionConfig struct definitions | `internal/config/types.go` |
| `DefaultProfile()`, `defaultRelevanceTiers()` constructors | `internal/config/defaults.go` |
| `LoadFromFile`, `LoadFromString`, unknown-key warning | `internal/config/loader.go` |
| `Source` iota, `SourceMap` type | `internal/config/sources.go` |
| `HARVX_*` env var constants, `buildEnvMap()` | `internal/config/env.go` |
| `ApplyTargetPreset()` (claude/chatgpt/generic) | `internal/config/target.go` |
| `Resolve()`, `ResolveOptions`, `ResolvedConfig` | `internal/config/resolver.go` |
| `DiscoverRepoConfig()`, `DiscoverGlobalConfig()` | `internal/config/discover.go` |
| `ProfileResolution`, `ResolveProfile()`, `resolveChain()` | `internal/config/profile.go` |
| `mergeProfile()` and per-field merge helpers | `internal/config/merge.go` |
| `ValidationError`, `LintResult` types | `internal/config/errors.go` |
| `Validate()`, `Lint()` and all check helpers | `internal/config/validate.go` |
| `ListTemplates()`, `GetTemplate()`, `RenderTemplate()` | `internal/config/templates.go` |
| `ShowOptions`, `ShowProfile()`, `ShowProfileJSON()` | `internal/config/show.go` |
| `ExplainFile()`, `TraceStep`, `ExplainResult`, pipeline simulation | `internal/config/explain.go` |
| `BuildDebugOutput()`, `FormatDebugOutput()`, `FormatDebugOutputJSON()` | `internal/config/debug.go` |
| `profilesCmd` + `list`/`init`/`show` subcommands | `internal/cli/profiles.go` |
| `profiles lint` subcommand | `internal/cli/profiles_lint.go` |
| `profiles explain` subcommand | `internal/cli/profiles_explain.go` |
| `config debug` subcommand | `internal/cli/config_debug.go` |
| Embedded TOML templates (base, nextjs, go-cli, python-django, rust-cargo, monorepo) | `internal/config/templates/*.toml` |
| `Golden()` test helper with `-update` flag | `internal/testutil/golden.go` |
| 8 end-to-end scenario integration tests | `internal/config/integration_test.go` |
| `FuzzConfigParse`, `FuzzValidate` fuzz targets | `internal/config/fuzz_test.go` |
| `BenchmarkConfigResolve`, `BenchmarkConfigValidate` | `internal/config/benchmark_test.go` |
| TOML test fixtures (valid, minimal, invalid_syntax, unknown_keys, global, repo, inheritance, circular, invalid_format, overlapping_tiers, contradictory) | `testdata/config/*.toml` |
| Integration scenario fixtures (8 scenarios) | `testdata/integration/profiles/scenario-{1-8}/` |

#### Verification

- `go build ./cmd/harvx/` pass
- `go vet ./...` pass
- `go test ./...` pass

---

### T-026: Tier Definitions and Default Tier Assignments

- **Status:** Completed
- **Date:** 2026-02-22
- **What was built:**
  - `Tier` type (`int`) with 6 named constants: `Tier0Critical` through `Tier5Low`
  - `DefaultUnmatchedTier = Tier2Secondary` — fallback for files matching no pattern
  - `String()` method on `Tier` with lowercase labels and numeric fallback
  - `TierDefinition` struct with `toml` struct tags for TOML serialization
  - `DefaultTierDefinitions()` returning all 6 built-in tiers per PRD Section 5.3
  - 10 table-driven test functions with 30+ sub-cases achieving 95%+ coverage
- **Files created/modified:**
  - `internal/relevance/tiers.go` -- Tier type, constants, TierDefinition struct, DefaultTierDefinitions()
  - `internal/relevance/tiers_test.go` -- Unit tests for all tier definitions and defaults
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

---

### T-027: Glob-Based File-to-Tier Matching

- **Status:** Completed
- **Date:** 2026-02-22
- **What was built:**
  - `TierMatcher` struct with private `tierEntry` slice sorted by ascending tier number
  - `NewTierMatcher(defs []TierDefinition) *TierMatcher` -- constructs matcher, sorts tiers, discards invalid patterns via `doublestar.ValidatePattern` at construction time
  - `Match(filePath string) Tier` -- allocation-free per-file matching; evaluates tiers lowest-to-highest, first-match-wins; normalises paths (backslash -> forward slash, strips `./` prefix)
  - `ClassifyFiles(files []string, tiers []TierDefinition) map[string]Tier` -- bulk classification returning original path keys
  - `sortTierDefinitions` (insertion sort on short lists) and `normalisePath` internal helpers
  - Input slice immutability: `NewTierMatcher` copies before sorting so callers' slices are never mutated
  - 28+ table-driven test functions covering all T-027 spec cases, edge cases, and invariants; 97.7% statement coverage
  - 2 benchmarks: `BenchmarkClassifyFiles10K` (~5ms for 10K files), `BenchmarkMatchSingle` (~367ns, 0 allocs)
- **Files created/modified:**
  - `internal/relevance/matcher.go` -- TierMatcher, NewTierMatcher, Match, ClassifyFiles, normalisePath
  - `internal/relevance/matcher_test.go` -- Comprehensive unit tests and benchmarks
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓  `go mod tidy` ✓  `go test -race` ✓

---

### T-029: Tokenizer Interface and Implementations (cl100k, o200k, none)

- **Status:** Completed
- **Date:** 2026-02-22
- **What was built:**
  - `Tokenizer` interface with `Count(text string) int` and `Name() string` methods
  - `tiktokenTokenizer` struct for `cl100k_base` and `o200k_base` using `pkoukk/tiktoken-go`; encoding loaded once on construction, goroutine-safe via tiktoken-go's immutable encode state
  - `estimatorTokenizer` struct for `"none"` mode: returns `len(text) / 4`; zero-allocation, no I/O
  - `NewTokenizer(name string) (Tokenizer, error)` factory; empty string defaults to `cl100k_base`; returns `ErrUnknownTokenizer` (sentinel) for unrecognised names
  - Exported name constants: `NameCL100K`, `NameO200K`, `NameNone`
  - `TIKTOKEN_CACHE_DIR` env var respected via tiktoken-go's built-in support
  - 3 benchmark sets (1KB/10KB/100KB) × 3 implementations = 9 benchmarks total
  - Concurrent safety test: 10 goroutines × 50 iterations for all three implementations
- **Files created/modified:**
  - `internal/tokenizer/tokenizer.go` -- Tokenizer interface, name constants, ErrUnknownTokenizer sentinel, NewTokenizer factory
  - `internal/tokenizer/tiktoken.go` -- tiktokenTokenizer (cl100k_base and o200k_base)
  - `internal/tokenizer/estimator.go` -- estimatorTokenizer (none/len-div-4)
  - `internal/tokenizer/tokenizer_test.go` -- factory tests, interface compliance, concurrent safety, name constants
  - `internal/tokenizer/tiktoken_test.go` -- cl100k/o200k correctness, Unicode, large text, benchmarks
  - `internal/tokenizer/estimator_test.go` -- len/4 formula table tests, large text, consistency, benchmarks
  - `go.mod` / `go.sum` -- added `github.com/pkoukk/tiktoken-go v0.1.8` (upgraded from v0.1.7 via mod tidy)
- **Key decisions:**
  - Encoding initialised once in constructor (not per `Count` call) per spec
  - `errors.Is`-compatible sentinel `ErrUnknownTokenizer` using `fmt.Errorf` (not `errors.New`) to allow wrapping with additional context
  - `estimatorTokenizer` holds no state so goroutine safety is trivially guaranteed
  - `tiktokenTokenizer.Count` short-circuits on empty string to avoid BPE call overhead
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓  `go mod tidy` ✓

---

### T-028: Relevance Sorter -- Sort Files by Tier and Path

- **Status:** Completed
- **Date:** 2026-02-22
- **What was built:**
  - `SortByRelevance(files []*pipeline.FileDescriptor) []*pipeline.FileDescriptor` -- returns a new sorted slice (input never mutated); primary sort by ascending `Tier`, secondary sort alphabetically by `Path`; uses `slices.SortStableFunc` + `cmp.Compare` (Go 1.22+)
  - `GroupByTier(files []*pipeline.FileDescriptor) map[int][]*pipeline.FileDescriptor` -- partitions files into a map keyed by tier number; preserves insertion order within each bucket
  - `TierStat` struct -- `Tier int`, `FileCount int`, `TotalTokens int`, `FilePaths []string`
  - `TierSummary(files []*pipeline.FileDescriptor) []TierStat` -- per-tier counts and total token sums; only populated tiers included; result sorted by ascending Tier; `FilePaths` sorted alphabetically within each stat
  - `ClassifyAndSort(files []*pipeline.FileDescriptor, tiers []TierDefinition) []*pipeline.FileDescriptor` -- integrates with `TierMatcher` from T-027: assigns `Tier` field on each descriptor then returns `SortByRelevance` output
  - 25 table-driven test functions covering all spec cases, golden 20-file order test, stability, determinism, mutation safety, and edge cases (empty, single, nil tiers)
  - 1 benchmark: `BenchmarkSortByRelevance10K` (10 000 files, realistic tier distribution)
- **Files created/modified:**
  - `internal/relevance/sorter.go` -- SortByRelevance, GroupByTier, TierStat, TierSummary, ClassifyAndSort
  - `internal/relevance/sorter_test.go` -- comprehensive unit tests and benchmark
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

---

### T-030: Parallel Per-File Token Counting

- **Status:** Completed
- **Date:** 2026-02-22
- **What was built:**
  - `TokenCounter` struct wrapping a `Tokenizer` for parallel per-file token counting
  - `CountFile(fd *pipeline.FileDescriptor)` -- populates `fd.TokenCount` from `fd.Content`, goroutine-safe
  - `CountFiles(ctx context.Context, files []*pipeline.FileDescriptor) (int, error)` -- parallel counting via `errgroup` with `SetLimit(runtime.NumCPU())` bounded concurrency, returns total token count, supports context cancellation
  - `EstimateOverhead(fileCount int, treeSize int) int` -- estimates output structure overhead using formula `200 + (fileCount * 35)`
  - 15 unit tests + 2 benchmarks covering all acceptance criteria, edge cases (empty, zero files, cancellation, processed content, field mutation safety)
- **Files created/modified:**
  - `internal/tokenizer/counter.go` -- TokenCounter, NewTokenCounter, CountFile, CountFiles, EstimateOverhead
  - `internal/tokenizer/counter_test.go` -- comprehensive unit tests and benchmarks
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

---

### T-031: Token Budget Enforcement with Truncation Strategies

- **Status:** Completed
- **Date:** 2026-02-22
- **What was built:**
  - `TruncationStrategy` string type with `SkipStrategy` ("skip") and `TruncateStrategy` ("truncate") constants
  - `TierStat` struct: `FilesIncluded`, `FilesExcluded`, `TokensUsed` per tier
  - `BudgetSummary` struct: `TierStats map[int]TierStat` + `SortedTierKeys()` convenience helper
  - `BudgetResult` struct: `IncludedFiles`, `ExcludedFiles`, `TruncatedFiles`, `TotalTokens`, `BudgetUsed`, `BudgetRemaining`, `Summary`
  - `BudgetEnforcer` struct with `maxTokens`, `strategy`, and `tok Tokenizer` fields
  - `NewBudgetEnforcer(maxTokens int, strategy TruncationStrategy, tok Tokenizer) *BudgetEnforcer` constructor; nil tok falls back to character estimator
  - `Enforce(files []*pipeline.FileDescriptor, overhead int) *BudgetResult` main method
  - Skip algorithm: iterates all files; smaller files after a large excluded file are still considered
  - Truncate algorithm: binary search over lines using the Tokenizer to find max lines fitting in budget; reserves 20 tokens for marker; appends `<!-- Content truncated: X of Y tokens shown -->` marker; all subsequent files excluded
  - Original `FileDescriptor` is never mutated; truncation returns a shallow copy with updated `Content` and `TokenCount`
  - `maxTokens <= 0` disables enforcement entirely (pass-through mode)
  - 25+ table-driven tests + 2 benchmarks covering all strategies, edge cases (empty, no budget, zero remaining, overhead exceeds max, skip-continues-after-skip, original-not-mutated), invariants (included+excluded==total, truncated⊆included, TotalTokens==sum), and per-tier summary accuracy
- **Files created:**
  - `internal/tokenizer/budget.go` -- TruncationStrategy, TierStat, BudgetSummary, BudgetResult, BudgetEnforcer, NewBudgetEnforcer, Enforce, enforceWithSkip, enforceWithTruncate, truncateToFit, SortedTierKeys
  - `internal/tokenizer/budget_test.go` -- comprehensive unit tests and benchmarks
- **Key decisions:**
  - Accepted `Tokenizer` in constructor for accurate binary search during truncation (not just char-based heuristic)
  - Fixed 20-token marker reservation to always leave room for the truncation comment
  - `TruncatedFiles` is a subset of `IncludedFiles` (same pointer), not a separate copy
  - `BudgetRemaining` can be negative when overhead > maxTokens (spec requirement)
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

---

