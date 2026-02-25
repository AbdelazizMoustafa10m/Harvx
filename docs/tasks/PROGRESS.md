# Harvx Task Progress Log

## Summary

| Status | Count |
|--------|-------|
| Completed | 79 |
| In Progress | 0 |
| Not Started | 16 |

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

### Phase 5: Compression (T-042 to T-050)

- **Status:** Completed
- **Date:** 2026-02-24
- **Tasks Completed:** 9 tasks

#### Features Implemented

| Feature | Tasks | Description |
| ------- | ----- | ----------- |
| WASM Runtime & Grammar Embedding | T-042 | `GrammarRegistry` with wazero v1.11.0, lazy `CompiledModule` compilation, double-checked locking, 8 embedded tree-sitter WASM grammars (~10.4 MB) |
| Language Detection & Compressor Interface | T-043 | `LanguageDetector` (24 extensions → 12 languages), `LanguageCompressor` interface, `SignatureKind` enum (9 kinds), `Signature`/`CompressedOutput` types, `CompressorRegistry` |
| Tier 1 AST Compressors: TypeScript & JavaScript | T-044 | `TypeScriptCompressor` and `JavaScriptCompressor` via shared `jsParser` state machine; extracts functions, classes, interfaces, type aliases, enums, imports, exports, decorators |
| Tier 1 AST Compressor: Go | T-045 | `GoCompressor` with 8-state parser; extracts package/import blocks, func/method signatures, structs with tags, interfaces, type/const/var blocks, generics, doc comments |
| Tier 1 AST Compressors: Python & Rust | T-046 | `PythonCompressor` (indentation-based, 9 states) and `RustCompressor` (brace-tracking, 9 states); extracts all structural declarations with doc comment/attribute/decorator attachment |
| Tier 2 AST Compressors: Java, C, C++ | T-047 | `JavaCompressor` (7 states), `CCompressor` (6 states), `CppCompressor` (8 states extending C); shared `c_base.go` helpers; full declaration extraction excluding bodies |
| Config Compressors & Fallback | T-048 | `JSONCompressor` (depth-2 skeleton, array collapsing), `YAMLCompressor` (line-based, depth ≤ 2 preservation), `TOMLCompressor` (section/comment preservation); `FallbackCompressor` passthrough with `IsFallback()` |
| Compression Orchestrator & Pipeline Integration | T-049 | `Orchestrator` with parallel execution (`errgroup.SetLimit`), per-file timeout, `CompressionStats` (atomic counters), `CompressedMarker`, `ProgressFunc` callback; `--compress`/`--compress-timeout` CLI flags |
| Regex Fallback Engine & E2E Tests | T-050 | `RegexCompressor` with per-language patterns for 8 languages; `CompressEngine` type (`ast`/`regex`/`auto`); auto engine tries AST first, falls back to regex; `--compress-engine` CLI flag; E2E + faithfulness test suites |

#### Key Technical Decisions

1. **Direct wazero over malivvan/tree-sitter** -- malivvan only supports C/C++ and is pre-release (v0.0.1); wazero v1.11.0 is stable and gives full control over WASM instantiation
2. **State machine parsers over tree-sitter WASM** -- Sourcegraph WASM grammars are Emscripten `SIDE_MODULE` builds incompatible with standalone wazero instantiation; line-by-line state machines implement the PRD fallback plan
3. **Shared `jsParser` with `jsParserConfig` flags** -- TypeScript and JavaScript share 95% of extraction logic; booleans control TS-specific features (interfaces, type aliases, enums)
4. **`grammars/` package at module root** -- Go `//go:embed` forbids `..` paths; separate package cleanly exports the embedded FS
5. **Double-checked locking in `GrammarRegistry`** -- `RLock` fast path for cache hits; `Lock` only for compilation with re-check after write lock acquisition
6. **Line-based YAML/TOML compressors without external library** -- avoids new dependencies for config-format structural skeletons
7. **`CompressEngine` auto mode** -- tries AST first, falls back to `RegexCompressor` on parse failure; enables graceful degradation for all supported languages

#### Key Files Reference

| Purpose | Location |
| ------- | -------- |
| Grammar embedding package | `grammars/embed.go` |
| Tree-sitter WASM grammars (8 languages) | `grammars/tree-sitter-{typescript,javascript,go,python,rust,java,c,cpp}.wasm` |
| Grammar download script | `scripts/fetch-grammars.sh` |
| WASM runtime & GrammarRegistry | `internal/compression/wasm.go` |
| Core types: SignatureKind, Signature, CompressedOutput | `internal/compression/types.go` |
| LanguageCompressor interface | `internal/compression/interface.go` |
| CompressEngine type & ParseCompressEngine | `internal/compression/engine.go` |
| LanguageDetector (24 ext → 12 languages) | `internal/compression/detector.go` |
| CompressorRegistry | `internal/compression/registry.go` |
| Shared JS/TS parsing engine | `internal/compression/js_base.go` |
| TypeScript compressor | `internal/compression/typescript.go` |
| JavaScript compressor | `internal/compression/javascript.go` |
| Go compressor | `internal/compression/golang.go` |
| Python compressor | `internal/compression/python.go` |
| Rust compressor | `internal/compression/rust.go` |
| Java compressor | `internal/compression/java.go` |
| Shared C/C++ helpers | `internal/compression/c_base.go` |
| C compressor | `internal/compression/clang.go` |
| C++ compressor | `internal/compression/cpp.go` |
| JSON compressor | `internal/compression/json_compressor.go` |
| YAML compressor | `internal/compression/yaml_compressor.go` |
| TOML compressor | `internal/compression/toml_compressor.go` |
| Fallback passthrough compressor | `internal/compression/fallback.go` |
| Regex fallback compressor | `internal/compression/regex.go` |
| Per-language regex pattern definitions | `internal/compression/regex_patterns.go` |
| Orchestrator (parallel execution, engine selection) | `internal/compression/orchestrator.go` |
| Atomic-safe CompressionStats | `internal/compression/stats.go` |
| WASM runtime tests & benchmarks | `internal/compression/wasm_test.go`, `internal/compression/wasm_bench_test.go` |
| Detector & registry tests | `internal/compression/detector_test.go`, `internal/compression/registry_test.go` |
| Per-language compressor tests | `internal/compression/{typescript,javascript,golang,python,rust,java,clang,cpp}_test.go` |
| Config compressor tests | `internal/compression/{json_compressor,yaml_compressor,toml_compressor,fallback}_test.go` |
| Orchestrator, stats, regex tests | `internal/compression/{orchestrator,stats,regex}_test.go` |
| E2E, faithfulness, benchmark tests | `internal/compression/{e2e,faithfulness,benchmark}_test.go` |
| Golden test fixtures (11 languages) | `testdata/compression/{typescript,javascript,go,python,rust,c,cpp,json,yaml,toml,e2e}/` |
| CLI flags (--compress, --compress-timeout, --compress-engine) | `internal/config/flags.go` |
| Pipeline compression step | `internal/pipeline/pipeline.go` |

#### Verification

- `go build ./cmd/harvx/` pass
- `go vet ./...` pass
- `go test ./...` pass

---

### Phase 6: Output & Rendering (T-051 to T-058)

- **Status:** Completed
- **Date:** 2026-02-25
- **Tasks Completed:** 8 tasks

#### Features Implemented

| Feature | Tasks | Description |
| ------- | ----- | ----------- |
| Directory Tree Builder | T-051 | `BuildTree`/`RenderTree` with Unicode box-drawing, directory collapsing, depth limits, and size/token annotations |
| Markdown Renderer | T-052 | `MarkdownRenderer` via `text/template` with streaming `io.Writer`, line numbers, conditional diff section, and 60+ language extension map |
| XML Renderer | T-053 | `XMLRenderer` producing Claude-optimized XML with CDATA wrapping, `]]>` boundary splitting, and safe attribute escaping |
| Content Hashing | T-054 | `ContentHasher` (XXH3 64-bit over sorted file collections) and `IncrementalHasher` (`io.Writer`) for streaming hash during output |
| Output Writer & Format Dispatch | T-055 | `OutputWriter` orchestrating renderer, hasher, and destination; atomic file writes; stdout mode with `io.MultiWriter`; 3-tier path resolution |
| Output Splitter | T-056 | `Splitter` with greedy bin-packing respecting tier boundaries and file atomicity; `--split` CLI flag; `PartPath` with `.part-NNN` insertion |
| Metadata JSON Sidecar | T-057 | `OutputMetadata` structs with snake_case JSON; `GenerateMetadata`/`WriteMetadata` with atomic write; `--output-metadata` CLI flag |
| Pipeline Integration | T-058 | `RenderOutput` orchestration function converting `[]pipeline.FileDescriptor` to rendered output; golden test infrastructure with `HARVX_UPDATE_GOLDEN=1` |

#### Key Technical Decisions

1. **`text/template` for XML renderer** -- preserves exact whitespace control; avoids `encoding/xml`'s automatic escaping that would corrupt CDATA sections
2. **Atomic file writes (temp → sync → close → rename)** -- prevents partial or corrupt output files on error in both `OutputWriter` and `WriteMetadata`
3. **Null byte separators in XXH3 hash** -- prevents path/content boundary collisions; case-sensitive byte-order sort ensures platform-independent determinism
4. **Greedy bin-packing with 15% overflow tolerance** -- keeps same-tier files from the same top-level directory together while respecting token budgets and file atomicity
5. **`*float64` for `BudgetUsedPercent`** -- serializes as JSON `null` when no token budget is configured, distinguishing "zero usage" from "no budget"

#### Key Files Reference

| Purpose | Location |
| ------- | -------- |
| `Renderer` interface, `RenderData`, `FileRenderEntry`, `DiffSummaryData` | `internal/output/renderer.go` |
| `TreeNode`, `FileEntry`, `TreeRenderOpts`; `BuildTree`, `RenderTree` | `internal/output/tree.go` |
| `MarkdownRenderer` with context cancellation support | `internal/output/markdown.go` |
| `XMLRenderer`, `wrapCDATA`, `xmlEscapeAttr` | `internal/output/xml.go` |
| Markdown and XML template constants and `FuncMap`s | `internal/output/templates.go` |
| `languageFromExt`, `formatBytes`, `addLineNumbers`, `tierLabel` helpers | `internal/output/helpers.go` |
| `ContentHasher`, `IncrementalHasher`, `FileHashEntry`, `FormatHash` | `internal/output/hash.go` |
| `OutputWriter`, `OutputOpts`, `OutputResult`, `countingWriter` | `internal/output/writer.go` |
| Format constants, `NewRenderer` factory, `ResolveOutputPath` | `internal/output/format.go` |
| `Splitter`, `SplitOpts`, `PartData`, `PartResult`, `PartPath`, `WriteSplit` | `internal/output/splitter.go` |
| `OutputMetadata`, `Statistics`, `FileStats`; `GenerateMetadata`, `WriteMetadata` | `internal/output/metadata.go` |
| `OutputConfig`, `RenderOutput` pipeline orchestration | `internal/output/pipeline.go` |
| Golden test helpers (`loadGoldenFile`, `compareGolden`, `writeGoldenFile`) | `internal/output/testutil_test.go` |
| `--split` and `--output-metadata` CLI flag registration | `internal/config/flags.go` |
| Golden test fixture files | `testdata/golden-fixtures/` |
| Golden test reference outputs | `internal/output/testdata/golden/` |

#### Verification

- `go build ./cmd/harvx/` pass
- `go vet ./...` pass
- `go test ./...` pass

---

### Phase 7: State & Diff (T-059 to T-065)

- **Status:** Completed
- **Date:** 2026-02-25
- **Tasks Completed:** 7 tasks

#### Features Implemented

| Feature | Tasks | Description |
| ------- | ----- | ----------- |
| State snapshot types | T-059 | `StateSnapshot` and `FileState` structs with deterministic JSON serialization, schema version validation, and `ErrUnsupportedVersion` sentinel |
| Content hashing | T-060 | `Hasher` interface and `XXH3Hasher` implementation via `zeebo/xxh3`; `HashFile` streaming reader and `HashFileDescriptors` batch helper |
| State cache persistence | T-061 | `StateCache` with atomic read/write/clear via `os.CreateTemp`+`os.Rename`; profile name sanitization; `ErrBranchMismatch`, `ErrNoState`, `ErrInvalidVersion` sentinels |
| State comparison engine | T-062 | `CompareStates` O(n) two-pass algorithm producing `DiffResult` with sorted `Added`/`Modified`/`Deleted` slices; nil-safe; `HasChanges`, `TotalChanged`, `Summary` helpers |
| Git-aware diffing | T-063 | `GitDiffer` with `GetChangedFiles`, `GetChangedFilesSince`, `BuildDiffResultFromGit`; `parseNameStatus` for `git diff --name-status`; `ErrGitNotFound`, `ErrNotGitRepo`, `ErrInvalidRef` sentinels |
| `harvx diff` subcommand | T-064 | Cobra subcommand with `--since`/`--base`/`--head` flags; `DetermineDiffMode` mutual-exclusion validation; `RunDiff` dispatcher; `FormatChangeSummary`; `--diff-only` and `--profile` persistent root flags |
| Cache subcommands & change summary | T-065 | `harvx cache clear` and `harvx cache show` (table/JSON); `RenderChangeSummary` in Markdown and XML; `NewDiffSummaryData` converter; `--clear-cache` wired in `PersistentPreRunE` |

#### Key Technical Decisions

1. **Deterministic JSON via sorted map keys** -- `StateSnapshot.Files` is a `map[string]FileState`; custom marshaler sorts keys before encoding to guarantee byte-identical output across runs
2. **Atomic cache writes via rename** -- `os.CreateTemp` + `os.Rename` prevents torn reads on crash; Windows fallback handles cross-device rename errors
3. **O(n) two-pass comparison** -- `CompareStates` uses hash-map lookups rather than nested loops; verified sub-millisecond on 10,000-file snapshots
4. **Git CLI over libgit2** -- `exec.CommandContext` with `context.Context` keeps CGO disabled and allows cancellation; no C dependency
5. **Three diff modes** -- cache-based, `--since <ref>`, and `--base/--head` PR review; `DetermineDiffMode` validates mutual exclusion at parse time

#### Key Files Reference

| Purpose | Location |
| ------- | -------- |
| StateSnapshot and FileState types, JSON serialization | `internal/diff/state.go` |
| StateSnapshot unit tests and golden fixture validation | `internal/diff/state_test.go` |
| Hasher interface | `internal/diff/hasher.go` |
| XXH3Hasher, HashFile, HashFileDescriptors | `internal/diff/xxh3.go` |
| XXH3 unit tests and benchmarks | `internal/diff/xxh3_test.go` |
| Sentinel errors (all diff package errors) | `internal/diff/errors.go` |
| StateCache atomic read/write/clear | `internal/diff/cache.go` |
| StateCache unit and concurrency tests | `internal/diff/cache_test.go` |
| CompareStates and DiffResult | `internal/diff/compare.go` |
| Comparison unit tests and benchmark | `internal/diff/compare_test.go` |
| GitDiffer, parseNameStatus, BuildDiffResultFromGit | `internal/diff/git.go` |
| Git integration tests with real repos | `internal/diff/git_test.go` |
| DiffMode, DiffOptions, RunDiff, FormatChangeSummary, walkDir | `internal/diff/diff.go` |
| Diff orchestration unit and integration tests | `internal/diff/diff_test.go` |
| `harvx diff` Cobra subcommand | `internal/cli/diff.go` |
| diff CLI tests | `internal/cli/diff_test.go` |
| `harvx cache` / `cache clear` / `cache show` Cobra commands | `internal/cli/cache.go` |
| Cache subcommand tests | `internal/cli/cache_test.go` |
| Root command with --clear-cache PersistentPreRunE | `internal/cli/root.go` |
| DiffOnly and Profile fields, --diff-only / --profile flags | `internal/config/flags.go` |
| RenderChangeSummary (Markdown/XML), NewDiffSummaryData | `internal/output/change_summary.go` |
| Change summary rendering tests | `internal/output/change_summary_test.go` |
| DiffSummaryData with Unchanged field | `internal/output/renderer.go` |
| Golden fixture: populated snapshot | `testdata/state/valid_snapshot.json` |
| Golden fixture: empty snapshot | `testdata/state/empty_snapshot.json` |

#### Verification

- `go build ./cmd/harvx/` pass
- `go vet ./...` pass
- `go test ./...` pass

---

### T-066: Core Pipeline as Go Library API

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `Pipeline` struct with `Run(ctx, RunOptions) (*RunResult, error)` method as the core processing engine
  - 7 stage service interfaces: `DiscoveryService`, `RelevanceService`, `TokenizerService`, `BudgetService`, `RedactionService`, `CompressionService`, `RenderService`
  - Functional options pattern: `WithDiscovery`, `WithRelevance`, `WithTokenizer`, `WithBudget`, `WithRedactor`, `WithCompressor`, `WithRenderer`
  - `RunOptions` with stage selection (`StageSelection`), git ref support, path filtering, and max token overrides
  - `RunResult` with aggregate stats (`RunStats`), per-stage timing (`StageTimings`), content hash, and exit code
  - Composable stage selection: `NewStageSelection()`, `DiscoveryOnly()`, `DiscoveryAndRelevance()`
  - Custom JSON serialization for `StageTimings` with human-readable duration strings and full roundtrip support
  - Pipeline threads `context.Context` through all stages for cancellation support
  - Renamed existing CLI function to `RunLegacy` for backward compatibility
- **Files created/modified:**
  - `internal/pipeline/interfaces.go` -- Stage service interfaces and supporting types (DiscoveryOptions, BudgetResult, RenderOptions, DiffSummaryEntry)
  - `internal/pipeline/options.go` -- PipelineOption type and 7 With* functional option constructors
  - `internal/pipeline/result.go` -- RunOptions, StageSelection, RunResult, RunStats, StageTimings with JSON serialization
  - `internal/pipeline/pipeline.go` -- Pipeline struct, NewPipeline constructor, Run method with 6-stage orchestration, legacy RunLegacy function
  - `internal/pipeline/run_test.go` -- 26 unit tests with mock stages covering all acceptance criteria
  - `internal/pipeline/result_test.go` -- 8 tests for result types, JSON roundtrip, and stage selection constructors
  - `internal/pipeline/pipeline_test.go` -- Updated existing tests to use RunLegacy
  - `internal/cli/generate.go` -- Updated to call RunLegacy
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-067: Clean Stdout Mode, Structured Exit Codes, and Non-Interactive Defaults

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `OutputMode` struct with `StdoutMode`, `IsPiped`, `StderrIsPiped` fields for output routing decisions
  - `DetectPipe(*os.File) bool` using `os.ModeCharDevice` check for terminal vs pipe detection
  - `DetectOutputMode(stdoutFlag bool) OutputMode` combining `--stdout` flag, `HARVX_STDOUT` env var, and live pipe detection
  - `ShouldSuppressProgress()` method: suppresses progress when stdout is piped (normal mode) or stderr is piped (stdout mode)
  - `ShouldDisableColor()` method: disables ANSI color when stderr is piped
  - `MessageWriter()` method: always returns `os.Stderr` for user-facing messages
  - `HARVX_STDOUT=true` env var support in `applyEnvOverrides`
  - Mutual exclusion validation: `--stdout` and `--output` are mutually exclusive (when `--output` is explicitly set)
  - Exit codes returned from pipeline library via `RunResult.ExitCode`, translated to `os.Exit` at CLI boundary in `cmd/harvx/main.go`
  - `--yes` flag accepted as no-op (non-interactive is default), `--stdout` flag already registered
- **Files created/modified:**
  - `internal/cli/output.go` -- OutputMode type, DetectPipe, DetectOutputMode, ShouldSuppressProgress, ShouldDisableColor, MessageWriter
  - `internal/cli/output_test.go` -- 20 unit tests covering pipe detection, output mode detection, progress suppression, color disable, message writer
  - `internal/config/flags.go` -- HARVX_STDOUT env var support and --stdout/--output mutual exclusion validation
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-068: JSON Preview Output and Metadata Sidecar

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `PreviewResult` struct in `internal/pipeline/result.go` with JSON struct tags matching the specified schema (11 fields: `total_files`, `total_tokens`, `tokenizer`, `tiers`, `redactions`, `estimated_time_ms`, `content_hash`, `profile`, `budget_utilization_percent`, `files_truncated`, `files_omitted`)
  - `BuildPreviewResult(result *RunResult, profile string, maxTokens int) *PreviewResult` function that converts pipeline `RunResult` into `PreviewResult` with tier key conversion (`map[int]int` to `map[string]int`), budget percentage calculation (`*float64`, nil when no budget), and content hash as lowercase hex
  - `PreviewStages() *StageSelection` helper returning discovery+relevance+tokenization stage selection for preview mode
  - `--json` local flag on `harvx preview` command that outputs machine-readable JSON to stdout via `json.MarshalIndent`
  - `PreviewJSON bool` field in `FlagValues` struct for future use by `brief` and `review-slice` commands
  - Full pipeline integration: `runPreviewJSON` builds a pipeline, runs with preview stages, and produces valid JSON even when services are not wired or errors occur
  - `writePreviewJSON` helper using `cmd.OutOrStdout()` for testability
  - `buildPreviewPipelineOptions` stub for future service wiring
- **Files created/modified:**
  - `internal/pipeline/result.go` -- Added `PreviewResult` struct, `BuildPreviewResult` function, `PreviewStages` helper
  - `internal/pipeline/result_test.go` -- 11 new tests: JSON schema compliance, roundtrip, budget null, string keys, basic conversion, no budget, zero result, budget utilization table-driven, content hash hex, tier conversion, preview stages
  - `internal/config/flags.go` -- Added `PreviewJSON bool` field to `FlagValues`
  - `internal/cli/preview.go` -- Added `--json` flag, `runPreviewJSON`, `writePreviewJSON`, `buildPreviewPipelineOptions`; updated help text with `--json` example
  - `internal/cli/preview_test.go` -- 10 new tests: flag registration, exits zero, schema validation, struct deserialization, max-tokens integration, null budget, pretty-printing, flag variable setting, stdout routing, help text
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-069: Assert-Include Coverage Checks and Environment Variable Overrides

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `CheckAssertInclude` function for verifying critical files are present in pipeline output via `--assert-include` glob patterns
  - `AssertionError` and `AssertionFailure` types with descriptive multi-failure error messages including pattern, file count, and remediation suggestions
  - `--assert-include` repeatable CLI flag using doublestar glob engine (same as relevance tier patterns)
  - `AssertInclude []string` field on `Profile` struct with TOML config support (`assert_include` key)
  - Profile inheritance merge support for `assert_include` via `mergeSlice`
  - Enhanced `applyEnvOverrides` with comprehensive `HARVX_*` env var support: `HARVX_PROFILE`, `HARVX_MAX_TOKENS` (int), `HARVX_TOKENIZER`, `HARVX_COMPRESS` (flexible bool), `HARVX_REDACT` (inverted bool), `HARVX_STDOUT` (flexible bool)
  - `parseBoolEnv` helper accepting `true/1/yes/false/0/no` (case-insensitive) for boolean env vars
  - Invalid env var values produce `slog.Warn` messages (e.g., "HARVX_MAX_TOKENS must be a positive integer")
- **Files created/modified:**
  - `internal/pipeline/assert.go` -- AssertionFailure, AssertionError, CheckAssertInclude with doublestar matching
  - `internal/pipeline/assert_test.go` -- 18 tests: empty patterns, single match/fail, multiple patterns, glob wildcards, error message format, errors.As unwrapping, profile+CLI merge
  - `internal/config/types.go` -- Added AssertInclude field to Profile struct
  - `internal/config/flags.go` -- Added AssertIncludes field, --assert-include flag, enhanced applyEnvOverrides, parseBoolEnv helper
  - `internal/config/env_test.go` -- 13 new tests: parseBoolEnv (20 cases), env var overrides for profile/max-tokens/tokenizer/compress/redact/stdout, CLI flag precedence
  - `internal/config/merge.go` -- Added AssertInclude to mergeProfile
  - `internal/config/resolver.go` -- Added assert_include to flattenProfileRaw, profileToFlatMap, flatMapToProfile
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-070: Repo Brief Command (`harvx brief`)

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `harvx brief` Cobra subcommand generating a stable, deterministic Repo Brief artifact (~1-4K tokens) with project-wide invariants
  - Brief generation workflow discovering README, invariants (CLAUDE.md, CONVENTIONS.md), architecture docs/ADRs, build commands, config info, review rules, and module map
  - Section extraction for Makefile targets, package.json scripts, go.mod info, Cargo.toml package section, pyproject.toml project section
  - Automatic module map generation from top-level directories with 50+ known directory descriptions and content-based inference fallback
  - Token budget enforcement truncating lower-priority sections first (module map → review rules → config → build → architecture → invariants → README)
  - `--json` flag for machine-readable metadata (token count, content hash, files included, section names, max tokens)
  - `--target claude` producing XML-formatted output with `<repo-brief>` wrapper and XML comment header
  - `--assert-include` coverage checks on brief source files
  - `--stdout` and `-o` output routing with brief-specific default filename (`harvx-brief.md`)
  - Content-addressed output via XXH3 hash enabling prompt caching across commits
  - `BriefMaxTokens` profile configuration field (default: 4000 tokens)
  - Deterministic output: sorted paths, fixed section order, stable rendering
- **Files created/modified:**
  - `internal/workflows/brief.go` -- Brief generation logic: section discovery, budget enforcement, Markdown/XML rendering, content hashing
  - `internal/workflows/brief_test.go` -- 30 unit tests: all sections, determinism (5-run), missing README/architecture, empty repo, token budget, Claude XML, assert-include, README variants/priority, content hash changes
  - `internal/workflows/module_map.go` -- ModuleMapEntry type, GenerateModuleMap (50+ known dirs), describeDirectory, inferDescription, RenderModuleMap
  - `internal/workflows/module_map_test.go` -- 15 tests: known directories, hidden dirs, content inference (Go/TS/Py/Rust/MD), empty dir, deterministic order, render format
  - `internal/workflows/section_extractor.go` -- ExtractMakefileTargets, ExtractPackageJSONScripts, ExtractGoModInfo, ExtractCargoTomlInfo, ExtractPyprojectInfo
  - `internal/workflows/section_extractor_test.go` -- 20 tests: Makefile targets (standard, empty, deps, dedup, hyphens), package.json scripts (standard, no scripts, invalid, sorted, complex), go.mod, Cargo.toml, pyproject.toml, TOML value extraction
  - `internal/cli/brief.go` -- Cobra command registration, --json flag, runBrief, resolveBriefMaxTokens, buildBriefTokenCounter, writeBriefJSON, writeBriefOutput
  - `internal/cli/brief_test.go` -- 13 CLI tests: command registration, properties, --json flag, global flag inheritance, stdout/JSON exit zero, JSON schema, determinism, Claude XML, metadata values, help text
  - `internal/config/types.go` -- Added BriefMaxTokens field to Profile struct
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-071: Review Slice Command (`harvx review-slice`)

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `harvx review-slice --base <ref> --head <ref>` Cobra subcommand generating a PR-specific context slice with changed files and bounded neighborhood
  - Multi-language import parser (Go, TypeScript/JavaScript, Python) for neighborhood discovery via regex-based heuristic parsing
  - Bounded neighborhood discovery finding related test files, importer files, and same-directory neighbors with configurable depth
  - Review slice generation workflow with git diff integration, token budget enforcement (changed files always prioritized), and deterministic output
  - Markdown and XML (Claude-optimized) rendering with code fences, language detection, and content hashing
  - `--json` flag for machine-readable metadata (token count, content hash, file lists, max tokens, refs)
  - `--base`/`--head` required flags, `--stdout`, `-o`, `--target`, `--assert-include`, `--profile` support
  - `SliceMaxTokens` (default: 20000) and `SliceDepth` (default: 1) profile configuration fields with merge/resolver support
  - Empty slice handling when no files changed between refs
  - Budget enforcement: changed files always included first, neighbors added greedily until budget reached
- **Files created/modified:**
  - `internal/workflows/imports.go` -- Multi-language import parser: ParseImports, parseGoImports, parseJSImports, parsePythonImports
  - `internal/workflows/imports_test.go` -- 73 test cases: Go/JS/TS/Python parsing, dedup/sort, normalize, relative imports
  - `internal/workflows/neighbors.go` -- DiscoverNeighbors, findRelatedTests, findImporters, findTestCandidates, resolveImportToPath
  - `internal/workflows/neighbors_test.go` -- Neighbor discovery tests: depth control, test file/importer detection, filtering
  - `internal/workflows/review_slice.go` -- GenerateReviewSlice, collectRepoFiles, buildSliceFiles, enforceSliceBudget, renderSlice, discoverNeighbors
  - `internal/workflows/review_slice_test.go` -- 88 test cases: validation, budget, rendering, language detection, neighbor discovery
  - `internal/cli/review_slice.go` -- Cobra command: --base/--head/--json flags, resolveSliceConfig, writeReviewSliceJSON, writeReviewSliceOutput
  - `internal/cli/review_slice_test.go` -- 15 test functions: registration, flags, required flags, help text, JSON output, config resolution
  - `internal/config/types.go` -- Added SliceMaxTokens, SliceDepth fields to Profile struct
  - `internal/config/merge.go` -- Added SliceMaxTokens, SliceDepth to mergeProfile
  - `internal/config/resolver.go` -- Added slice_max_tokens, slice_depth to flattenProfileRaw, profileToFlatMap, flatMapToProfile
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-072: Module Slice Command (`harvx slice`)

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `harvx slice --path <module>` Cobra subcommand generating targeted context about specific modules or directories
  - Module slice workflow: discovers files under specified paths, finds neighbors (imports, tests) via depth-bounded discovery, applies token budget (module files always prioritized)
  - Supports multiple `--path` flags for slicing multiple modules simultaneously
  - Markdown and XML (Claude-optimized) rendering with code fences, language detection, and XXH3 content hashing
  - `--json` flag for machine-readable metadata (token count, content hash, file lists, max tokens, paths)
  - `--stdout`, `-o`, `--target`, `--assert-include`, `--profile`, `--max-tokens`, `--compress` flag support
  - `isModuleFile` matching handles both directory prefixes (with `/` separator to prevent false positives) and exact single-file paths
  - Deterministic output: sorted paths, stable rendering, content-addressed via XXH3 hash
  - Reuses `collectRepoFiles`, `buildSliceFiles`, `enforceSliceBudget`, `discoverNeighbors`, `resolveSliceConfig`, `buildSliceTokenCounter` from review-slice
- **Files created/modified:**
  - `internal/workflows/slice.go` -- GenerateModuleSlice, ModuleSliceOptions/Result/JSON types, isModuleFile, renderModuleSlice (Markdown + XML)
  - `internal/workflows/slice_test.go` -- 23+ test cases: validation, single dir/file, multiple paths, neighbors, budget, determinism, formats, assert-include, edge cases
  - `internal/cli/slice.go` -- Cobra command: --path/--json flags, runSlice, writeSliceJSON, writeSliceOutput
  - `internal/cli/slice_test.go` -- 11 test functions: registration, flags, required flags, help text, JSON output, global flag inheritance
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-073: Workspace Manifest Config and Command (`harvx workspace`)

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `.harvx/workspace.toml` manifest parsing using BurntSushi/toml with unknown-key warnings for forward compatibility
  - `WorkspaceConfig`, `WorkspaceManifest`, `WorkspaceRepo` types with TOML struct tags matching PRD Section 5.11.3 schema
  - `DiscoverWorkspaceConfig` auto-detection walking up parent directories, stopping at `.git` boundary or maxSearchDepth (20)
  - `ValidateWorkspace` producing warnings (not errors) for missing repo paths, unknown integration targets, duplicate repo names
  - `ExpandPath` resolving `~` to `$HOME` and relative paths relative to workspace.toml location
  - `GenerateWorkspaceInit` producing valid starter TOML with placeholder entries
  - `harvx workspace` command rendering manifests as Markdown or XML with repo list, integration graph, shared schemas
  - `--deep` mode including top-level directory listings per repo (max 30 entries, hidden files skipped)
  - `--json` flag for machine-readable metadata (name, description, repo count, token count, content hash, warnings)
  - `--target claude` producing XML output with `<workspace>`, `<repo>`, `<integrations>`, `<shared-schemas>` tags
  - `harvx workspace init` generating `.harvx/workspace.toml` with overwrite protection via `--yes`
  - `--stdout`, `-o`, `--target` output routing with `harvx-workspace.md` default filename
  - Deterministic output: repos sorted by name, integration edges sorted, shared schemas sorted, XXH3 content hash
- **Files created/modified:**
  - `internal/config/workspace.go` -- WorkspaceConfig/WorkspaceManifest/WorkspaceRepo types, LoadWorkspaceConfig, DiscoverWorkspaceConfig, ValidateWorkspace, ExpandPath, GenerateWorkspaceInit
  - `internal/config/workspace_test.go` -- 30 tests: TOML parsing, auto-detection, validation warnings, path expansion, init generation
  - `internal/workflows/workspace.go` -- GenerateWorkspace, WorkspaceOptions/Result/JSON types, Markdown+XML renderers, integration edges, shared schemas, directory listings
  - `internal/workflows/workspace_test.go` -- 28 tests: rendering, determinism, XML mode, deep mode, path existence, content hashing, helper functions
  - `internal/cli/workspace.go` -- Cobra command: workspace + workspace init subcommands, --json/--deep flags, output routing
  - `internal/cli/workspace_test.go` -- 11 tests: registration, flags, JSON output, stdout output, missing config, init creation, init overwrite, global flag inheritance
  - `testdata/config/workspace.toml` -- Test fixture with 3 repos demonstrating all fields
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-074: Session Bootstrap Documentation and Claude Code Hooks Integration

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - Session bootstrap guide with Claude Code `SessionStart` hook configuration, `--target claude` XML rendering documentation, performance guidance, and troubleshooting
  - Review pipeline guide with shell script examples, GitHub Actions CI workflow, environment variable reference, and JSON metadata usage for automation
  - Workspace setup guide for multi-repo session bootstrap with manifest schema documentation and hook integration examples
  - Lean CLAUDE.md template under 500 tokens with rules-only philosophy and Harvx dynamic context reference
  - Reference `hooks.json` template with valid JSON structure for Claude Code hooks
  - Three persona-specific recipes: quick context (Alex), pipeline review (Zizo), CI integration (Jordan)
  - Validation test suite verifying JSON validity, token budget, command references, YAML structure, and file existence
- **Files created/modified:**
  - `docs/guides/session-bootstrap.md` -- Session bootstrap guide with hook config, target claude docs, troubleshooting
  - `docs/guides/review-pipeline.md` -- End-to-end review pipeline with shell script and GitHub Actions examples
  - `docs/guides/workspace-setup.md` -- Multi-repo workspace documentation with manifest schema and hook integration
  - `docs/templates/CLAUDE.md` -- Lean baseline template under 500 tokens
  - `docs/templates/hooks.json` -- Reference Claude Code hooks configuration (valid JSON)
  - `docs/recipes/quick-context.md` -- Alex persona: quick chat context recipes
  - `docs/recipes/pipeline-review.md` -- Zizo persona: pipeline integration recipes
  - `docs/recipes/ci-integration.md` -- Jordan persona: CI/CD setup with GitHub Actions
  - `internal/cli/docs_test.go` -- 5 test functions validating documentation completeness and correctness
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-075: Verify Command and Faithfulness Checking (`harvx verify`)

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `harvx verify` Cobra subcommand comparing packed output to original source files for faithfulness checking
  - Output file parser supporting both Markdown and XML formats with auto-detection, extracting file blocks (path, content, metadata)
  - Verification workflow with 5 status types: MATCH, REDACTION_DIFF, COMPRESSION_DIFF, UNEXPECTED_DIFF, FILE_CHANGED
  - `--sample <n>` flag for reproducible random sampling using FNV-1a hash seed with Fisher-Yates shuffle (default: 10 files)
  - `--path <file>` repeatable flag for verifying specific files
  - `--json` flag for structured JSON output suitable for CI pipelines
  - `--profile` support for resolving correct output path and settings
  - Budget reporting from `.meta.json` sidecar: tokenizer, total tokens, budget utilization %, compressed files, redactions
  - Human-readable report with [PASS]/[WARN] labels, aligned file paths, and unified diff snippets for unexpected differences
  - Exit code 0 when all files pass, exit code 2 (partial) when any have unexpected differences
  - Redaction detection heuristic: line-by-line comparison identifying `[REDACTED:type]` placeholder substitutions
  - Simple line-based diff with configurable max lines (default: 10) for debugging unexpected differences
  - Trailing newline normalization for robust content comparison across output formats
- **Files created/modified:**
  - `internal/workflows/output_parser.go` -- ParseOutput, ParseMarkdownOutput, ParseXMLOutput with CDATA unwrap and backtick unescape
  - `internal/workflows/output_parser_test.go` -- 30+ tests: format detection, single/multi file extraction, compressed/redaction metadata, CDATA split, XML entities
  - `internal/workflows/verify.go` -- VerifyOutput workflow, file selection, verification logic, redaction detection, simple diff, budget info
  - `internal/workflows/verify_test.go` -- 14+ tests: exact match, redaction diff, compression diff, unexpected diff, file changed, sampling, reproducibility, paths, budget info
  - `internal/cli/verify.go` -- Cobra command: --sample/--path/--json flags, human-readable report, JSON output, budget line
  - `internal/cli/verify_test.go` -- 9 tests: registration, properties, flags, global flag inheritance, help, status labels, formatNumber, pluralS
  - `testdata/expected-output/verify-pass.md` -- Test fixture with matching output (3 files)
  - `testdata/expected-output/verify-fail.md` -- Test fixture with intentional mismatch in config.go
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-076: Golden Questions Harness and Quality Evaluation Framework

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - Golden questions TOML types (`GoldenQuestion`, `GoldenQuestionsConfig`) with BurntSushi/toml parsing, unknown-key warnings, validation, and auto-discovery
  - `harvx quality` command (alias: `qa`) evaluating critical file coverage against golden questions using doublestar glob matching
  - `harvx quality init` subcommand generating starter `.harvx/golden-questions.toml` with 3 example questions
  - Coverage analysis workflow walking repo files, matching critical_files patterns, computing per-question and aggregate coverage
  - `--json` flag for structured CI-friendly output, `--questions` flag for custom questions path
  - Human-readable report with `[PASS]`/`[MISS]` labels and coverage summary
  - Evaluation methodology guide, starter TOML template, and shell script template for LLM A/B testing
  - 5 recognized categories: architecture, configuration, security, conventions, integration
- **Files created/modified:**
  - `internal/config/quality.go` -- GoldenQuestion, GoldenQuestionsConfig types, LoadGoldenQuestions, ValidateGoldenQuestions, DiscoverGoldenQuestions, GenerateGoldenQuestionsInit
  - `internal/config/quality_test.go` -- 16 tests: TOML parsing, validation, discovery, init generation, categories, uniqueness
  - `internal/workflows/quality.go` -- QualityOptions, QuestionResult, QualityResult, EvaluateQuality, evaluateQuestion, patternMatchesAny
  - `internal/workflows/quality_test.go` -- 20 tests: full coverage, partial coverage, glob patterns, empty files, auto-discovery, sorting, hidden dirs
  - `internal/cli/quality.go` -- Cobra quality/qa command, quality init subcommand, --json/--questions/--output/--yes flags
  - `internal/cli/quality_test.go` -- 9 tests: registration, alias, flags, init flags, help, status labels, global flag inheritance
  - `docs/guides/golden-questions.md` -- Evaluation methodology documentation with writing tips, CI workflow, scoring guide
  - `docs/templates/golden-questions.toml` -- Starter template with 5 example questions across all categories
  - `docs/templates/evaluate.sh` -- Shell script template for LLM A/B evaluation (diff-only vs Harvx context)
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓
