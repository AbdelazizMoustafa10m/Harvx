# Harvx Task Progress Log

## Summary

| Status | Count |
|--------|-------|
| Completed | 81 |
| In Progress | 0 |
| Not Started | 14 |

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

### Phase 8: Workflows (T-066 to T-078)

- **Status:** Completed
- **Date:** 2026-02-25
- **Tasks Completed:** 13 tasks

#### Features Implemented

| Feature | Tasks | Description |
| ------- | ----- | ----------- |
| Pipeline library API | T-066 | `Pipeline.Run()` with 7 stage service interfaces, functional options (`With*`), `RunResult`/`RunStats`/`StageTimings` with JSON serialization |
| Output routing & exit codes | T-067 | `OutputMode` pipe detection via `os.ModeCharDevice`, `--stdout`/`HARVX_STDOUT`, `RunResult.ExitCode` propagated to `os.Exit` at CLI boundary |
| JSON preview metadata | T-068 | `PreviewResult` struct, `BuildPreviewResult`, `--json` flag on `harvx preview`, `PreviewStages` selection helper |
| Assert-include coverage checks | T-069 | `CheckAssertInclude` with doublestar globs, `AssertionError` type, comprehensive `HARVX_*` env var overrides with `parseBoolEnv` |
| Repo brief command | T-070 | `harvx brief` generating stable 1–4K token artifact: README, invariants, ADRs, build commands, module map, token budget enforcement |
| Review slice command | T-071 | `harvx review-slice --base --head` with multi-language import parser (Go/TS/JS/Python), bounded neighbor discovery, budget enforcement |
| Module slice command | T-072 | `harvx slice --path` for targeted module context, multiple paths, reuses review-slice neighbor/budget infrastructure |
| Workspace command | T-073 | `harvx workspace` with `.harvx/workspace.toml` manifest, `DiscoverWorkspaceConfig`, `ValidateWorkspace`, `workspace init` subcommand |
| Session bootstrap docs | T-074 | Guides (session-bootstrap, review-pipeline, workspace-setup), hooks template, CLAUDE.md template, persona recipes |
| Verify command | T-075 | `harvx verify` faithfulness checking: MATCH/REDACTION_DIFF/COMPRESSION_DIFF/UNEXPECTED_DIFF/FILE_CHANGED, `--sample`, unified diff snippets |
| Quality evaluation | T-076 | `harvx quality`/`qa` golden questions harness, coverage analysis, `quality init`, CI-friendly `--json` output |
| MCP server | T-077 | `harvx mcp serve` stdio transport with `brief`/`slice`/`review_slice` tools, typed `mcp.AddTool[In, Out]`, graceful SIGINT/SIGTERM shutdown |
| Integration test suite | T-078 | 40 end-to-end tests via compiled binary: all workflow commands, env var overrides, exit codes, determinism, performance bounds |

#### Key Technical Decisions

1. **Functional options for pipeline stages** -- `PipelineOption` / `With*` constructors allow test doubles without interface indirection at call sites; `RunLegacy` preserves backward compatibility for the existing `generate` command
2. **Pipe detection via `os.ModeCharDevice`** -- `DetectPipe` auto-suppresses progress output and ANSI color when stdout/stderr are piped, eliminating need for explicit `--quiet` in scripts
3. **XXH3 content hashing on all outputs** -- deterministic byte-identical output enables prompt caching across commits; hash changes only when content changes
4. **doublestar glob engine for `assert-include`** -- consistent with relevance tier patterns; avoids regex complexity for path matching on untrusted input
5. **Direct workflow calls in MCP tool handlers** -- no subprocess spawning; each invocation gets independent state making handlers inherently thread-safe
6. **Compiled binary via `TestMain` for integration tests** -- exercises real CLI flag registration, cobra wiring, and exit codes rather than unit-testing internals in isolation

#### Key Files Reference

| Purpose | Location |
| ------- | -------- |
| Stage interfaces and supporting types | `internal/pipeline/interfaces.go` |
| Functional option constructors | `internal/pipeline/options.go` |
| `RunOptions`, `RunResult`, `RunStats`, `StageTimings`, `PreviewResult` | `internal/pipeline/result.go` |
| `Pipeline` struct, `Run`, `RunLegacy` | `internal/pipeline/pipeline.go` |
| `CheckAssertInclude`, `AssertionError` | `internal/pipeline/assert.go` |
| `OutputMode`, `DetectPipe`, `DetectOutputMode` | `internal/cli/output.go` |
| `FlagValues`, `applyEnvOverrides`, `parseBoolEnv`, `--assert-include` | `internal/config/flags.go` |
| `Profile` struct (all new fields: `AssertInclude`, `BriefMaxTokens`, `SliceMaxTokens`, `SliceDepth`) | `internal/config/types.go` |
| Profile merge logic | `internal/config/merge.go` |
| Profile flatten/resolve | `internal/config/resolver.go` |
| Workspace manifest types, discovery, validation | `internal/config/workspace.go` |
| Golden questions types, discovery, validation | `internal/config/quality.go` |
| Brief generation: section discovery, budget, rendering | `internal/workflows/brief.go` |
| Module map generation (50+ known dirs) | `internal/workflows/module_map.go` |
| Makefile/package.json/go.mod/Cargo/pyproject extractors | `internal/workflows/section_extractor.go` |
| Multi-language import parser (Go/TS/JS/Python) | `internal/workflows/imports.go` |
| Bounded neighbor discovery | `internal/workflows/neighbors.go` |
| `GenerateReviewSlice`, budget enforcement, rendering | `internal/workflows/review_slice.go` |
| `GenerateModuleSlice`, `isModuleFile` | `internal/workflows/slice.go` |
| `GenerateWorkspace`, Markdown+XML renderers | `internal/workflows/workspace.go` |
| `ParseOutput`, Markdown+XML extractors | `internal/workflows/output_parser.go` |
| `VerifyOutput`, redaction detection, simple diff | `internal/workflows/verify.go` |
| `EvaluateQuality`, coverage analysis | `internal/workflows/quality.go` |
| MCP server creation and lifecycle | `internal/server/mcp.go` |
| MCP tool input/output types and handlers | `internal/server/tools.go` |
| `harvx brief` Cobra command | `internal/cli/brief.go` |
| `harvx review-slice` Cobra command | `internal/cli/review_slice.go` |
| `harvx slice` Cobra command | `internal/cli/slice.go` |
| `harvx workspace` / `workspace init` Cobra commands | `internal/cli/workspace.go` |
| `harvx verify` Cobra command | `internal/cli/verify.go` |
| `harvx quality` / `quality init` Cobra commands | `internal/cli/quality.go` |
| `harvx mcp serve` Cobra command | `internal/cli/mcp.go` |
| Integration test infrastructure (binary build, helpers) | `tests/integration/setup_test.go` |
| Workflow end-to-end tests | `tests/integration/workflow_test.go` |
| Pipeline composition tests | `tests/integration/pipeline_test.go` |
| Env var override tests | `tests/integration/env_test.go` |
| Exit code validation tests | `tests/integration/exitcode_test.go` |
| Determinism and performance tests | `tests/integration/determinism_test.go` |
| Session bootstrap guide | `docs/guides/session-bootstrap.md` |
| Review pipeline guide | `docs/guides/review-pipeline.md` |
| Workspace setup guide | `docs/guides/workspace-setup.md` |
| Golden questions methodology | `docs/guides/golden-questions.md` |
| MCP integration guide | `docs/guides/mcp-integration.md` |
| Lean CLAUDE.md template | `docs/templates/CLAUDE.md` |
| Claude Code hooks reference | `docs/templates/hooks.json` |
| Sample golden questions TOML | `docs/templates/golden-questions.toml` |
| LLM A/B evaluation shell script | `docs/templates/evaluate.sh` |

#### Verification

- `go build ./cmd/harvx/` pass
- `go vet ./...` pass
- `go test ./...` pass
- `go test -tags integration` pass (40/40)

---

