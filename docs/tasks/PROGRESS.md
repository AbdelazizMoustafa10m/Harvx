# Harvx Task Progress Log

## Summary

| Status | Count |
|--------|-------|
| Completed | 50 |
| In Progress | 0 |
| Not Started | 45 |

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

### Phase 3: Relevance & Tokens (T-026 to T-033)

- **Status:** Completed
- **Date:** 2026-02-22
- **Tasks Completed:** 8 tasks

#### Features Implemented

| Feature | Tasks | Description |
| ------- | ----- | ----------- |
| Tier type system | T-026 | `Tier` int type with 6 named constants, `TierDefinition` struct with TOML tags, `DefaultTierDefinitions()` |
| Glob-based tier matching | T-027 | `TierMatcher` with allocation-free `Match` and bulk `ClassifyFiles`; first-match-wins on sorted tiers |
| Relevance sorting & grouping | T-028 | `SortByRelevance`, `GroupByTier`, `TierSummary`, `ClassifyAndSort` integrating matcher into pipeline |
| Tokenizer interface & implementations | T-029 | `Tokenizer` interface; cl100k and o200k via `pkoukk/tiktoken-go`; `none` len/4 estimator; `NewTokenizer` factory |
| Parallel token counting | T-030 | `TokenCounter.CountFiles` with `errgroup` bounded by `runtime.NumCPU()`; `EstimateOverhead` formula |
| Budget enforcement | T-031 | `BudgetEnforcer.Enforce` with skip and truncate strategies; binary-search line truncation with 20-token marker reserve |
| Relevance explain & inclusion summary | T-032 | `Explain`, `FormatExplain`, `GenerateInclusionSummary`; all-matches collection with deterministic ordering |
| Token reporting CLI | T-033 | 5 new root flags; `TokenReport`, `TopFilesReport`, `HeatmapReport`; `harvx preview --heatmap` subcommand |

#### Key Technical Decisions

1. **Tokenizer constructed once** -- tiktoken encoding loaded in constructor, not per `Count` call; goroutine-safe and avoids repeated I/O
2. **Truncation uses binary search** -- `enforceWithTruncate` bisects content lines with the actual `Tokenizer` for accurate fit; fixed 20-token reservation guarantees room for the truncation marker
3. **`BudgetRemaining` may be negative** -- when overhead exceeds `maxTokens`, the field reflects the deficit rather than clamping to zero, matching spec behaviour
4. **Input immutability in `TierMatcher`** -- `NewTierMatcher` copies the caller's slice before sorting; caller order is never disturbed
5. **`ErrUnknownTokenizer` via `fmt.Errorf`** -- sentinel supports `errors.Is` unwrapping when callers wrap it with additional context
6. **`previewHeatmap` as package-level var** -- bound after cobra parses flags, not in `init()`, to avoid nil-pointer dereference before `flagValues` is populated
7. **No lipgloss in reports** -- `TokenReport`/`HeatmapReport` use plain text with Unicode `─` box-drawing characters; zero additional binary weight

#### Key Files Reference

| Purpose | Location |
| ------- | -------- |
| Tier type, constants, `DefaultTierDefinitions` | `internal/relevance/tiers.go` |
| `TierMatcher`, `Match`, `ClassifyFiles`, `normalisePath` | `internal/relevance/matcher.go` |
| `SortByRelevance`, `GroupByTier`, `TierStat`, `TierSummary`, `ClassifyAndSort` | `internal/relevance/sorter.go` |
| `Explain`, `FormatExplain`, `GenerateInclusionSummary`, `TierLabel` | `internal/relevance/explain.go` |
| `Tokenizer` interface, `ErrUnknownTokenizer`, `NewTokenizer` factory, name constants | `internal/tokenizer/tokenizer.go` |
| `tiktokenTokenizer` (cl100k, o200k) | `internal/tokenizer/tiktoken.go` |
| `estimatorTokenizer` (none / len÷4) | `internal/tokenizer/estimator.go` |
| `TokenCounter`, `CountFile`, `CountFiles`, `EstimateOverhead` | `internal/tokenizer/counter.go` |
| `BudgetEnforcer`, `Enforce`, skip/truncate algorithms, `BudgetResult`, `BudgetSummary` | `internal/tokenizer/budget.go` |
| `TokenReport`, `TopFilesReport`, `HeatmapReport`, `FormatInt`, `TierLabel` map | `internal/tokenizer/report.go` |
| `PrintTokenReport`, `PrintTopFiles` CLI helpers | `internal/cli/token_report.go` |
| `harvx preview` subcommand with `--heatmap` flag | `internal/cli/preview.go` |
| 5 new persistent flags; `--tokenizer`/`--truncation-strategy` validation | `internal/config/flags.go` |
| Shell completion for `--tokenizer` and `--truncation-strategy` | `internal/cli/root.go`, `internal/cli/generate.go` |

#### Verification

- `go build ./cmd/harvx/` pass
- `go vet ./...` pass
- `go test ./...` pass

---

### Phase 4: Security (T-034 to T-041)

- **Status:** Completed
- **Date:** 2026-02-23
- **Tasks Completed:** 8 tasks

#### Features Implemented

| Feature | Tasks | Description |
| ------- | ----- | ----------- |
| Redaction type system | T-034 | `Confidence`, `RedactionMatch`, `RedactionSummary`, `RedactionConfig`, `RedactionRule`, `Redactor` interface, `PatternRegistry` |
| Built-in detection patterns | T-035 | 19 gitleaks-inspired rules in 3 confidence tiers (6 high, 9 medium, 4 low); structural validators `ValidateJWT`, `ValidateAWSKeyID` |
| Shannon entropy analyzer | T-036 | `EntropyAnalyzer` with per-charset thresholds, `Calculate`, `DetectCharset`, `AnalyzeToken` with suspicious-context boosting |
| Streaming redaction pipeline | T-037 | `StreamRedactor` with keyword pre-filter, entropy gating, multi-line PEM state machine, right-to-left replacement, context cancellation |
| Sensitive file handling | T-038 | 29-pattern `sensitiveFilePatterns`, `SensitiveFilePatterns()`, `WarnIfSensitiveFile()`; extended `DefaultIgnorePatterns` and `SensitivePatterns` in discovery; walker warning integration |
| Redaction reporting | T-039 | `ReportGenerator` with `BuildReport`, `GenerateJSON`, `GenerateText`, `WriteReport` (extension-based format), `FormatInlineSummary`; 18-entry `secretTypeLabels` |
| CLI flags and pipeline wiring | T-040 | `--redaction-report` flag, `HARVX_NO_REDACT`/`HARVX_FAIL_ON_REDACTION` env overrides, `CustomPatternDefinition`, `CompileCustomPattern`, full pipeline redaction integration |
| Regression corpus and fuzz tests | T-041 | 15 fixture files covering all 19 rules, 15 `.expected` JSON files, `TestGoldenCorpus`, `TestFalsePositiveRate`, `TestAllPatternsExercised`, 3 fuzz targets |

#### Key Technical Decisions

1. **RE2-only regex** -- no lookaheads or lookbehinds; O(n) matching guaranteed for untrusted content
2. **Capture group 1 convention** -- all 19 built-in patterns use CG1 for secret value extraction, enabling uniform redactor logic
3. **Right-to-left replacement** -- byte offsets are preserved when multiple matches appear on the same line
4. **`ByConfidence` as `map[string]int`** -- clean JSON serialization with predictable string keys without custom marshaling
5. **`SensitivePatterns ⊆ DefaultIgnorePatterns` invariant** -- enforced by `TestSensitivePatterns_SubsetOfDefaults`; avoids divergence between the two lists
6. **`ConfidenceMedium` as default pipeline threshold** -- safe default that avoids low-confidence noise without discarding real secrets
7. **Golden corpus checks presence, not exclusivity** -- `TestGoldenCorpus` asserts expected matches are found but allows extra entropy-triggered matches, preventing test brittleness

#### Key Files Reference

| Purpose | Location |
| ------- | -------- |
| Core types: Confidence, RedactionMatch, RedactionSummary, RedactionConfig | `internal/security/types.go` |
| RedactionRule struct, NewRedactionRule constructor, FormatReplacement | `internal/security/rule.go` |
| Redactor interface + StreamRedactor implementation | `internal/security/redactor.go` |
| PatternRegistry, NewDefaultRegistry, NewEmptyRegistry | `internal/security/registry.go` |
| 19 built-in rules, registerBuiltinPatterns | `internal/security/patterns.go` |
| ValidateJWT, ValidateAWSKeyID structural validators | `internal/security/validate.go` |
| EntropyAnalyzer, Calculate, DetectCharset, AnalyzeToken | `internal/security/entropy.go` |
| IsSensitiveFile, SensitiveFilePatterns, WarnIfSensitiveFile | `internal/security/sensitive.go` |
| Report, ReportGenerator, BuildReport, GenerateJSON/Text, FormatInlineSummary | `internal/security/report.go` |
| CompileCustomPattern (config-to-security bridge) | `internal/security/custom.go` |
| CustomPatternDefinition struct, RedactionConfig fields | `internal/config/types.go` |
| --redaction-report flag, env var overrides | `internal/config/flags.go` |
| validateCustomPatterns | `internal/config/validate.go` |
| Extended DefaultIgnorePatterns and SensitivePatterns | `internal/discovery/defaults.go` |
| SuppressSensitiveWarnings field, sensitive-file warning in Walk | `internal/discovery/walker.go` |
| buildRedactionConfig, printRedactionSummary, maybeWriteReport | `internal/pipeline/pipeline.go` |
| 15 secret fixture files + 15 .expected JSON files | `testdata/secrets/` |
| TestGoldenCorpus, TestFalsePositiveRate, TestAllPatternsExercised | `internal/security/golden_test.go` |
| FuzzRedactRandomContent, FuzzRedactEnvFile, FuzzEntropyAnalyzer | `internal/security/fuzz_test.go` |

#### Verification

- `go build ./cmd/harvx/` pass
- `go vet ./...` pass
- `go test ./...` pass

---

### T-042: Wazero WASM Runtime Setup and Grammar Embedding

- **Status:** Completed
- **Date:** 2026-02-24
- **What was built:**
  - `GrammarRegistry` with wazero runtime initialization, lazy WASM module compilation, double-checked locking cache, concurrent access safety
  - Grammar embedding package (`grammars/embed.go`) with `//go:embed *.wasm` for 8 tree-sitter languages (TypeScript, JavaScript, Go, Python, Rust, Java, C, C++)
  - Grammar WASM files from Sourcegraph `tree-sitter-wasms` npm v0.1.13 (~10.4 MB total)
  - `SupportedLanguages()`, `HasLanguage()`, `Close()`, `Runtime()` methods on registry
  - `ErrUnknownLanguage` sentinel error with `errors.Is` support
  - Fetch script (`scripts/fetch-grammars.sh`) with unpkg.com primary CDN and jsDelivr fallback
  - 13 unit tests: language loading, caching, unknown language errors, concurrent access, context cancellation, resource cleanup
  - 3 benchmarks: cold start per-grammar, warm cache retrieval, all-grammars compilation
- **Files created/modified:**
  - `grammars/embed.go` -- Grammar embedding package with `embed.FS` and `GrammarFiles` map
  - `grammars/README.md` -- Grammar documentation with sizes, sources, license info
  - `grammars/tree-sitter-{typescript,javascript,go,python,rust,java,c,cpp}.wasm` -- Embedded WASM grammar files
  - `internal/compression/wasm.go` -- `GrammarRegistry` with lazy compilation, double-checked locking, `slog` logging
  - `internal/compression/wasm_test.go` -- 13 unit tests including concurrent access and context cancellation
  - `internal/compression/wasm_bench_test.go` -- 3 benchmarks (cold start, warm cache, all grammars)
  - `scripts/fetch-grammars.sh` -- Grammar download script with CDN fallback
  - `go.mod` / `go.sum` -- Added `github.com/tetratelabs/wazero v1.11.0`
  - `.gitignore` -- Added `!grammars/*.wasm` negation
- **Key decisions:**
  - **Direct wazero over malivvan/tree-sitter** -- malivvan only supports C/C++, is pre-release (v0.0.1, 2 stars); direct wazero v1.11.0 is stable and gives full control
  - **`CompiledModule` (not `api.Module`)** -- Registry compiles and caches WASM; instantiation with host functions deferred to T-043+ parser layer
  - **`grammars/` package at module root** -- Go `//go:embed` forbids `..` paths; separate package cleanly exports embedded FS
  - **Double-checked locking** -- RLock fast path for cache hits, Lock only for compilation, re-check after write lock acquisition
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-043: Language Detection and LanguageCompressor Interface

- **Status:** Completed
- **Date:** 2026-02-24
- **What was built:**
  - `LanguageCompressor` interface with `Compress`, `Language`, `SupportedNodeTypes` methods
  - `SignatureKind` enum (9 kinds: function, class, struct, interface, type, import, export, constant, doc_comment) with `String()` method
  - `Signature` struct capturing extracted code elements with source-order line numbers
  - `CompressedOutput` struct with `Render()` (joins signatures with blank-line separators) and `CompressionRatio()` methods
  - `LanguageDetector` mapping 24 file extensions to 12 language identifiers across Tier 1 (typescript, javascript, go, python, rust) and Tier 2 (java, c, cpp, json, yaml, toml)
  - `CompressorRegistry` with `Register`, `Get`, `GetByLanguage`, `IsSupported`, `Languages` methods
  - 21 detector tests: all extensions, unknown extensions, case sensitivity, ambiguous `.h`, nested paths, defensive copy, count
  - 14 registry/types tests: registry CRUD, replacement, multi-compressor dispatch, `SignatureKind.String()`, `Render()`, `CompressionRatio()`, source ordering
- **Files created/modified:**
  - `internal/compression/types.go` -- `SignatureKind`, `Signature`, `CompressedOutput` types with `Render()` and `CompressionRatio()` methods
  - `internal/compression/interface.go` -- `LanguageCompressor` interface definition
  - `internal/compression/detector.go` -- `LanguageDetector` with 24 extension-to-language mappings and `SupportedExtensions()` copy method
  - `internal/compression/registry.go` -- `CompressorRegistry` with register/lookup/dispatch by file path or language
  - `internal/compression/detector_test.go` -- 7 test functions with 65+ subtests for language detection
  - `internal/compression/registry_test.go` -- 14 test functions covering registry operations and type behavior
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-044: Tier 1 Compressor -- TypeScript and JavaScript

- **Status:** Completed
- **Date:** 2026-02-24
- **What was built:**
  - `TypeScriptCompressor` and `JavaScriptCompressor` implementing `LanguageCompressor` interface
  - Shared `jsParser` line-by-line state machine parser extracting structural signatures (functions, classes, interfaces, type aliases, enums, imports, exports, constants, doc comments, decorators)
  - TypeScript-specific extraction: interfaces, type aliases, enums, type annotations
  - JavaScript extraction: functions, classes, arrow functions, imports, exports, constants, doc comments
  - Brace depth tracking with string-awareness (ignores braces in quotes/backticks)
  - Doc comment and decorator attachment to following declarations
  - Class body member extraction (field declarations + method signatures, no bodies)
  - 10 TypeScript fixture files + 10 expected outputs for golden tests
  - 5 JavaScript fixture files + 4 expected outputs for golden tests
  - 9 TypeScript golden tests + 20+ unit tests + 2 benchmarks
  - 4 JavaScript golden tests + 14+ unit tests + 1 benchmark
- **Files created/modified:**
  - `internal/compression/js_base.go` -- Shared JS/TS parsing engine with state machine, detection helpers, extraction helpers, brace counting
  - `internal/compression/typescript.go` -- `TypeScriptCompressor` with full TS extraction (interfaces, types, enums, type annotations)
  - `internal/compression/javascript.go` -- `JavaScriptCompressor` with JS-only extraction (no TS-specific features)
  - `internal/compression/typescript_test.go` -- 9 golden tests, 20+ unit tests, 2 benchmarks
  - `internal/compression/javascript_test.go` -- 4 golden tests, 14+ unit tests, 1 benchmark
  - `testdata/compression/typescript/` -- 10 input fixtures + 10 expected outputs
  - `testdata/compression/javascript/` -- 5 input fixtures + 4 expected outputs
- **Key decisions:**
  - **State machine over tree-sitter WASM** -- Sourcegraph WASM grammars are Emscripten SIDE_MODULE builds incompatible with standalone wazero instantiation; implemented robust line-by-line parser per PRD fallback plan
  - **Shared `jsParser` with config flags** -- TypeScript and JavaScript share 95% of extraction logic; `jsParserConfig` booleans control TS-specific features
  - **Separate class doc/decorator tracking** -- `classDocComment`/`classDecorators` fields prevent member-level doc comments from overwriting class-level ones
  - **Verbatim extraction** -- all source text preserved exactly as written; function/method bodies replaced with `{ ... }` markers
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-045: Tier 1 Compressor -- Go

- **Status:** Completed
- **Date:** 2026-02-24
- **What was built:**
  - `GoCompressor` implementing `LanguageCompressor` interface with line-by-line state machine parser
  - 8-state parser (`goStateTopLevel`, `goStateInLineComment`, `goStateInBlockComment`, `goStateInImport`, `goStateInType`, `goStateInConst`, `goStateInVar`, `goStateInFunc`)
  - Extracts: package clauses, import blocks, function/method signatures (excluding bodies), struct declarations (with tags), interface declarations, type aliases/definitions, const/var blocks (including iota), doc comments
  - Go-specific `goCountBraces` handling double-quoted strings, backtick raw strings (struct tags), rune literals, line comments, block comments
  - `findFuncBodyBrace` correctly identifies function body `{` vs `interface{}` braces in parameter types
  - Grouped declaration support: `type (...)`, `const (...)`, `var (...)` blocks using `trimmed == ")"` termination to avoid false positives from `)` inside expressions
  - 8 golden test fixtures covering: simple functions, methods with receivers, structs with tags/embedding, interfaces with embedding, generics (Go 1.18+), const/iota blocks, import patterns, full realistic file
  - 43+ unit tests covering all node types, edge cases, doc comments, build constraints, context cancellation
  - 1 benchmark for compression throughput
- **Files created/modified:**
  - `internal/compression/golang.go` -- GoCompressor with 8-state line-by-line parser, signature extraction, Go-specific brace counting
  - `internal/compression/golang_test.go` -- 8 golden tests, 43+ unit tests, 1 benchmark
  - `testdata/compression/go/simple_func.go` + `.expected` -- Basic function declarations
  - `testdata/compression/go/methods.go` + `.expected` -- Methods with receivers
  - `testdata/compression/go/structs.go` + `.expected` -- Struct declarations with tags and embedding
  - `testdata/compression/go/interfaces.go` + `.expected` -- Interface declarations with embedding
  - `testdata/compression/go/generics.go` + `.expected` -- Generic types and functions
  - `testdata/compression/go/const_iota.go` + `.expected` -- Const blocks with iota, var blocks
  - `testdata/compression/go/imports.go` + `.expected` -- Various import patterns
  - `testdata/compression/go/full_file.go` + `.expected` -- Realistic complete Go file
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-046: Tier 1 Compressor -- Python and Rust

- **Status:** Completed
- **Date:** 2026-02-24
- **What was built:**
  - `PythonCompressor` implementing `LanguageCompressor` interface with indentation-based state machine parser (9 states)
  - Python extraction: imports, function/async function signatures, class declarations with method signatures and fields, decorators (@dataclass, @property, @staticmethod, @classmethod), docstrings (module/class/function), type-annotated assignments, `__all__` lists, Protocol classes
  - Indentation-based scope tracking for Python's whitespace-significant syntax
  - `RustCompressor` implementing `LanguageCompressor` interface with brace-tracking state machine parser (9 states)
  - Rust extraction: use declarations, fn signatures (pub/pub(crate)/pub(super)/async/unsafe/const), structs (regular/tuple/unit with derive macros), enums with variants, traits with method signatures and associated types, impl blocks with method signatures, type aliases, const/static items, mod declarations, macro_rules! names, extern "C" blocks
  - `rustCountBraces` handling raw strings (`r#"..."#`), string/char literals, line/block comments
  - Doc comment (`///`, `//!`) and attribute (`#[...]`) attachment to declarations
  - 8 Python golden test fixtures + 8 Rust golden test fixtures
  - 35+ Python unit tests + 40+ Rust unit tests + 16 golden tests + 2 benchmarks
- **Files created/modified:**
  - `internal/compression/python.go` -- PythonCompressor with 9-state indentation-based parser
  - `internal/compression/python_test.go` -- 35+ unit tests, 8 golden tests, 1 benchmark
  - `internal/compression/python_golden_gen_test.go` -- Golden file regeneration helper
  - `internal/compression/rust.go` -- RustCompressor with 9-state brace-tracking parser
  - `internal/compression/rust_test.go` -- 40+ unit tests, 8 golden tests, 1 benchmark
  - `testdata/compression/python/` -- 8 input fixtures + 8 expected outputs
  - `testdata/compression/rust/` -- 8 input fixtures + 8 expected outputs
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-047: Tier 2 Compressor -- Java, C, and C++

- **Status:** Completed
- **Date:** 2026-02-24
- **What was built:**
  - `JavaCompressor` implementing `LanguageCompressor` with 7-state line-by-line parser extracting: package/import declarations, class/interface/enum/annotation-type/record declarations, method/constructor signatures (bodies excluded), Javadoc and annotation attachment, nested class headers
  - `CCompressor` implementing `LanguageCompressor` with 6-state parser extracting: `#include`/`#define` directives, function definitions and prototypes (bodies excluded), struct/enum declarations with fields, typedef statements, forward declarations, global variable declarations
  - `CppCompressor` implementing `LanguageCompressor` with 8-state parser extending C extraction with: class declarations (with access specifiers, member extraction), template declarations, namespace definitions (with nested extraction), using declarations, enum class/struct, virtual/override/const/noexcept qualifiers, operator overloading, multiple inheritance
  - Shared `c_base.go` with `cCountBraces`, `cExtractIdentifier`, detection helpers for preprocessor directives, function definitions, struct/enum/typedef, doc comment accumulation
  - 4 C golden test fixtures + 4 C++ golden test fixtures
  - Comprehensive test suites: 30+ Java tests, 25+ C tests, 25+ C++ tests, 2 benchmarks
- **Files created/modified:**
  - `internal/compression/java.go` -- Java compressor with 7-state parser
  - `internal/compression/java_test.go` -- 30+ unit tests covering all Java declaration types
  - `internal/compression/c_base.go` -- Shared C/C++ helpers (brace counting, detection, doc comments)
  - `internal/compression/clang.go` -- C compressor with 6-state parser
  - `internal/compression/clang_test.go` -- 25+ unit tests and 4 golden tests
  - `internal/compression/cpp.go` -- C++ compressor with 8-state parser extending C
  - `internal/compression/cpp_test.go` -- 25+ unit tests and 4 golden tests
  - `testdata/compression/c/` -- 4 input fixtures + 4 expected outputs
  - `testdata/compression/cpp/` -- 4 input fixtures + 4 expected outputs
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

