# Harvx Task Progress Log

## Summary

| Status | Count |
|--------|-------|
| Completed | 58 |
| In Progress | 0 |
| Not Started | 37 |

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

### T-051: Directory Tree Builder (In-Memory Tree + Rendering)

- **Status:** Completed
- **Date:** 2026-02-24
- **What was built:**
  - `TreeNode` nested struct with `BuildTree` (flat path list → in-memory tree) and `RenderTree` (Unicode box-drawing output with emoji indicators)
  - Directory collapsing: single-child directory chains merge into combined paths (e.g., `src/utils/helpers/`)
  - Sorting: directories before files, both alphabetically case-insensitive
  - `TreeRenderOpts` with `MaxDepth` (truncates with `...`), `ShowSize`, and `ShowTokens` metadata annotations
  - Human-readable size formatting (B, KB, MB, GB)
- **Files created/modified:**
  - `internal/output/tree.go` -- TreeNode, FileEntry, TreeRenderOpts types; BuildTree, RenderTree, collapseTree, sortTree, humanizeSize functions
  - `internal/output/tree_test.go` -- 27 unit tests + 4 golden tests + 3 benchmarks covering hierarchy, sorting, collapsing, depth limits, metadata, Unicode, edge cases
  - `internal/output/testdata/golden/tree-basic.golden` -- Golden test: basic tree rendering
  - `internal/output/testdata/golden/tree-with-metadata.golden` -- Golden test: size and token annotations
  - `internal/output/testdata/golden/tree-collapsed.golden` -- Golden test: collapsed directory chains
  - `internal/output/testdata/golden/tree-depth-limited.golden` -- Golden test: MaxDepth=2 truncation
  - `testdata/expected-output/tree-basic.txt` -- Reference output: basic tree
  - `testdata/expected-output/tree-with-metadata.txt` -- Reference output: metadata annotations
  - `testdata/expected-output/tree-collapsed.txt` -- Reference output: collapsed dirs
  - `testdata/expected-output/tree-depth-limited.txt` -- Reference output: depth limit
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-052: Markdown Output Renderer with Go Templates

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `Renderer` interface with `Render(ctx, w, data)` for format dispatch
  - `RenderData` and `FileRenderEntry` structs holding all pipeline output for rendering
  - `MarkdownRenderer` implementation using Go `text/template` with streaming output to `io.Writer`
  - Template system with named sub-templates: header, summary, tree, files, changeSummary composed via `markdown-root`
  - `template.FuncMap` helpers: formatBytes, formatNumber, languageFromExt (60+ extensions), addLineNumbers, escapeTripleBackticks, tierLabel, sortedKeys
  - Line numbers support via `ShowLineNumbers` flag with right-aligned numbering
  - Conditional change summary section (diff mode) with added/modified/deleted file lists
  - Deterministic output: same input always produces byte-identical output
- **Files created/modified:**
  - `internal/output/renderer.go` -- Renderer interface, RenderData, FileRenderEntry, DiffSummaryData types
  - `internal/output/markdown.go` -- MarkdownRenderer implementation with context cancellation support
  - `internal/output/templates.go` -- Markdown template constants (header, summary, tree, files, changeSummary, root) and FuncMap
  - `internal/output/helpers.go` -- languageFromExt (60+ ext map), formatBytes, formatNumber, addLineNumbers, repeatString, tierLabel, escapeTripleBackticks
  - `internal/output/helpers_test.go` -- 60+ unit tests for all helper functions
  - `internal/output/markdown_test.go` -- 25 unit tests + 2 golden tests + 2 benchmarks covering all acceptance criteria
  - `internal/output/testdata/golden/markdown-basic.golden` -- Golden test: basic Markdown rendering
  - `internal/output/testdata/golden/markdown-line-numbers.golden` -- Golden test: line-numbered rendering
  - `testdata/expected-output/markdown-basic.md` -- Reference output: basic Markdown
  - `testdata/expected-output/markdown-line-numbers.md` -- Reference output: line numbers
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-053: XML Output Renderer for Claude Target Preset

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `XMLRenderer` implementing `Renderer` interface, producing Claude-optimized XML with semantic tags (`<repository>`, `<metadata>`, `<file_summary>`, `<directory_structure>`, `<files>`, `<statistics>`)
  - CDATA section wrapping for file content and directory tree with proper `]]>` splitting across CDATA boundaries
  - `xmlEscapeAttr` for safe XML attribute/element encoding of user data (paths, project names, error messages)
  - XML template system with 7 named sub-templates using `text/template` (not `encoding/xml`)
  - `<file>` elements with `path`, `tokens`, `tier`, `size`, `language`, `compressed` attributes
  - Line numbers support within CDATA content via `--line-numbers` flag
  - Optional `<change_summary>` section for diff mode with added/modified/deleted file lists
  - Well-formed XML output validated by `encoding/xml` decoder in tests
  - Deterministic output: same input always produces byte-identical output
- **Files created/modified:**
  - `internal/output/xml.go` -- XMLRenderer struct, wrapCDATA, xmlEscapeAttr functions
  - `internal/output/templates.go` -- XML template constants (xmlHeaderTmpl, xmlSummaryTmpl, xmlTreeTmpl, xmlFilesTmpl, xmlStatisticsTmpl, xmlChangeSummaryTmpl, xmlRootTmpl) and xmlFuncMap
  - `internal/output/xml_test.go` -- 28+ unit tests + 3 golden tests + 2 benchmarks covering well-formedness, CDATA edge cases, special chars, section ordering
  - `internal/output/testdata/golden/xml-basic.golden` -- Golden test: basic XML rendering
  - `internal/output/testdata/golden/xml-line-numbers.golden` -- Golden test: line-numbered XML rendering
  - `internal/output/testdata/golden/xml-cdata-edge.golden` -- Golden test: CDATA edge case with ]]> splitting
  - `testdata/expected-output/xml-basic.xml` -- Reference output: basic XML
  - `testdata/expected-output/xml-cdata-edge.xml` -- Reference output: CDATA edge cases
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-054: Content Hashing (XXH3) and Deterministic Output

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `ContentHasher` with `ComputeContentHash` computing deterministic XXH3 64-bit hash over sorted file collections using `zeebo/xxh3`
  - `IncrementalHasher` implementing `io.Writer` for streaming hash computation during output writing
  - `FormatHash` producing 16-character zero-padded lowercase hex string for output headers
  - `FileHashEntry` type with path + null byte separator convention preventing path/content boundary collisions
  - Defensive input copy before sorting to avoid mutating caller's slice
  - Case-sensitive byte-order sorting for platform-independent determinism
- **Files created/modified:**
  - `internal/output/hash.go` -- ContentHasher, IncrementalHasher, FileHashEntry, FormatHash
  - `internal/output/hash_test.go` -- 29 unit tests + 3 benchmarks covering determinism, order independence, stability, null byte separation, Unicode paths, empty inputs, incremental equivalence
  - `go.mod` -- Added `github.com/zeebo/xxh3 v1.1.0` dependency
  - `go.sum` -- Updated with xxh3 and klauspost/cpuid transitive dependencies
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

### T-055: Output Writer, File Path Resolution, and Stdout Support

- **Status:** Completed
- **Date:** 2026-02-25
- **What was built:**
  - `OutputWriter` orchestration layer coordinating renderer, content hasher, and output destination (file or stdout)
  - `OutputOpts` and `OutputResult` structs for write configuration and structured results
  - Atomic file writes: `os.CreateTemp` → render → `Sync` → `Close` → `os.Rename` with deferred cleanup on error
  - Stdout mode with `io.MultiWriter` for simultaneous streaming and XXH3 hash computation
  - Format dispatch factory (`NewRenderer`) returning `MarkdownRenderer` or `XMLRenderer`
  - Output path resolution with 3-tier precedence: CLI flag → profile config → default path
  - Automatic file extension appending (`.md`/`.xml`) when path has no extension
  - `countingWriter` helper for tracking bytes written during streaming
- **Files created/modified:**
  - `internal/output/writer.go` -- OutputWriter, OutputOpts, OutputResult, countingWriter, Write/writeStdout/writeFile methods
  - `internal/output/writer_test.go` -- 20+ unit tests covering stdout/file modes, atomic writes, hash consistency, path resolution, error cases
  - `internal/output/format.go` -- Format constants, NewRenderer factory, ExtensionForFormat, DefaultOutputPath, ResolveOutputPath
  - `internal/output/format_test.go` -- Table-driven tests for renderer factory, extension mapping, path resolution
- **Verification:** `go build` ✓  `go vet` ✓  `go test` ✓

