# Harvx Task Index

> **Total Tasks:** 95 across 10 phases | **Must Have:** 76 | **Should Have:** 16 | **Nice to Have:** 3
>
> **Estimated Total Effort:** ~650-950 hours (~14 weeks at full pace)

This index organizes all implementation tasks for Harvx -- a Go CLI tool that packages codebases into LLM-optimized context documents.

---

## Quick Navigation

- [Phase 1: Foundation](#phase-1-foundation-t-001--t-015)
- [Phase 2: Profiles](#phase-2-profiles-t-016--t-025)
- [Phase 3: Relevance & Tokens](#phase-3-relevance--tokens-t-026--t-033)
- [Phase 4: Security](#phase-4-security-t-034--t-041)
- [Phase 5: Compression](#phase-5-compression-t-042--t-050)
- [Phase 6: Output & Rendering](#phase-6-output--rendering-t-051--t-058)
- [Phase 7: State & Diff](#phase-7-state--diff-t-059--t-065)
- [Phase 8: Workflows](#phase-8-workflows-t-066--t-078)
- [Phase 9: Interactive TUI](#phase-9-interactive-tui-t-079--t-087)
- [Phase 10: Polish & Distribution](#phase-10-polish--distribution-t-088--t-095)

---

## Phase 1: Foundation (T-001 -- T-015)

> Go project scaffolding, CLI framework, file discovery engine

| Task | Name | Priority | Effort | Dependencies |
|------|------|----------|--------|--------------|
| [T-001](T-001-go-project-init.md) | Go Project Initialization & Directory Structure | Must Have | Small (2-4hrs) | None |
| [T-002](T-002-makefile-setup.md) | Makefile Setup | Must Have | Small (2-4hrs) | T-001 |
| [T-003](T-003-central-data-types.md) | Central Data Types (FileDescriptor & Pipeline DTOs) | Must Have | Small (2-4hrs) | T-001 |
| [T-004](T-004-structured-logging.md) | Structured Logging with slog | Must Have | Small (2-4hrs) | T-001 |
| [T-005](T-005-cobra-cli-root-cmd.md) | Cobra CLI Framework & Root Command | Must Have | Medium (6-8hrs) | T-001, T-002, T-003, T-004 |
| [T-006](T-006-version-command.md) | Version Command & Build Info | Must Have | Small (2-4hrs) | T-005 |
| [T-007](T-007-global-flags.md) | Global Flags Implementation | Must Have | Medium (6-8hrs) | T-005, T-004 |
| [T-008](T-008-generate-subcommand.md) | Generate Subcommand (harvx generate / harvx gen) | Must Have | Medium (6-10hrs) | T-005, T-007 |
| [T-009](T-009-shell-completions.md) | Shell Completions (harvx completion) | Should Have | Small (2-4hrs) | T-005 |
| [T-010](T-010-exit-code-handling.md) | Exit Code Handling | Must Have | Small (2-4hrs) | T-003, T-005 |
| [T-011](T-011-gitignore-parsing.md) | .gitignore Parsing & Matching | Must Have | Medium (6-10hrs) | T-001, T-003 |
| [T-012](T-012-default-ignores-harvxignore.md) | Default Ignore Patterns & .harvxignore Support | Must Have | Medium (6-8hrs) | T-011 |
| [T-013](T-013-binary-detection-large-files.md) | Binary File Detection & Large File Skipping | Must Have | Small (3-5hrs) | T-001, T-003 |
| [T-014](T-014-filtering-git-tracked-symlinks.md) | Extension/Pattern Filtering, --git-tracked-only & Symlinks | Must Have | Medium (8-12hrs) | T-011, T-012, T-013 |
| [T-015](T-015-parallel-file-discovery.md) | Parallel File Discovery Engine (Walker with errgroup) | Must Have | Large (14-20hrs) | T-003, T-004, T-007, T-011-T-014 |

**Phase index:** [PHASE-1 details in INDEX.md header section]

---

## Phase 2: Profiles (T-016 -- T-025)

> TOML configuration, profile inheritance, framework templates

| Task | Name | Priority | Effort | Dependencies |
|------|------|----------|--------|--------------|
| [T-016](T-016-config-types-defaults.md) | Configuration Types, Defaults, and TOML Loading | Must Have | Medium (8-12hrs) | None |
| [T-017](T-017-multi-source-config-merging.md) | Multi-Source Configuration Merging and Resolution | Must Have | Large (14-20hrs) | T-016 |
| [T-018](T-018-config-auto-detection.md) | Configuration File Auto-Detection and Discovery | Must Have | Small (3-5hrs) | T-016 |
| [T-019](T-019-profile-inheritance.md) | Profile Inheritance with Deep Merge | Must Have | Medium (8-12hrs) | T-016, T-017 |
| [T-020](T-020-config-validation.md) | Configuration Validation and Lint Engine | Must Have | Medium (8-12hrs) | T-016, T-019 |
| [T-021](T-021-framework-profile-templates.md) | Framework-Specific Profile Templates | Must Have | Medium (6-10hrs) | T-016 |
| [T-022](T-022-profiles-init-list-show.md) | Profile CLI -- init, list, show | Must Have | Medium (8-12hrs) | T-005, T-016, T-021 |
| [T-023](T-023-profiles-lint-explain.md) | Profile CLI -- lint and explain | Should Have | Medium (8-12hrs) | T-020, T-022 |
| [T-024](T-024-config-debug-command.md) | Config Debug Command | Should Have | Small (4-6hrs) | T-005, T-017 |
| [T-025](T-025-profile-integration-tests.md) | Profile Integration Tests and Golden Tests | Must Have | Medium (8-12hrs) | T-016-T-024 |

**Phase index:** [PHASE-2-INDEX.md](PHASE-2-INDEX.md)

---

## Phase 3: Relevance & Tokens (T-026 -- T-033)

> Priority-based file sorting, token counting, budget enforcement

| Task | Name | Priority | Effort | Dependencies |
|------|------|----------|--------|--------------|
| [T-026](T-026-tier-definitions-defaults.md) | Tier Definitions and Default Tier Assignments | Must Have | Medium (6-8hrs) | None |
| [T-027](T-027-glob-tier-matching.md) | Glob-Based File-to-Tier Matching | Must Have | Medium (8-12hrs) | T-026 |
| [T-028](T-028-relevance-sorter.md) | Relevance Sorter -- Sort Files by Tier and Path | Must Have | Small (4-6hrs) | T-026, T-027 |
| [T-029](T-029-tokenizer-interface-impl.md) | Tokenizer Interface and Implementations (cl100k, o200k, none) | Must Have | Medium (8-12hrs) | None |
| [T-030](T-030-parallel-token-counting.md) | Parallel Per-File Token Counting | Must Have | Medium (6-10hrs) | T-029 |
| [T-031](T-031-token-budget-enforcement.md) | Token Budget Enforcement with Truncation Strategies | Must Have | Medium (8-12hrs) | T-028, T-029, T-030 |
| [T-032](T-032-relevance-explain.md) | Relevance Explain and Inclusion Summary | Should Have | Medium (6-8hrs) | T-027, T-028, T-031 |
| [T-033](T-033-token-reporting-cli.md) | Token Reporting CLI Flags and Heatmap | Should Have | Medium (8-12hrs) | T-029-T-032 |

---

## Phase 4: Security (T-034 -- T-041)

> Secret detection, redaction pipeline, regression testing

| Task | Name | Priority | Effort | Dependencies |
|------|------|----------|--------|--------------|
| [T-034](T-034-redaction-types-interfaces.md) | Redaction Core Types, Interfaces, and Pattern Registry | Must Have | Medium (6-8hrs) | None |
| [T-035](T-035-detection-patterns.md) | Gitleaks-Inspired Secret Detection Patterns | Must Have | Large (14-20hrs) | T-034 |
| [T-036](T-036-entropy-analyzer.md) | Shannon Entropy Analyzer for High-Entropy Strings | Must Have | Medium (6-10hrs) | T-034 |
| [T-037](T-037-streaming-redaction-filter.md) | Streaming Redaction Filter Pipeline | Must Have | Large (14-20hrs) | T-035, T-036 |
| [T-038](T-038-sensitive-file-handling.md) | Sensitive File Default Exclusions & Heightened Scanning | Must Have | Small (4-6hrs) | T-037 |
| [T-039](T-039-redaction-report.md) | Redaction Report and Output Summary | Must Have | Medium (6-10hrs) | T-037 |
| [T-040](T-040-cli-redaction-flags.md) | CLI Redaction Flags and Profile Configuration | Must Have | Medium (6-10hrs) | T-037, T-039 |
| [T-041](T-041-regression-test-corpus.md) | Secret Detection Regression Test Corpus & Fuzz Testing | Must Have | Medium (8-12hrs) | T-040 |

**Phase index:** [PHASE-4-INDEX.md](PHASE-4-INDEX.md)

---

## Phase 5: Compression (T-042 -- T-050)

> Tree-sitter WASM code compression via wazero

| Task | Name | Priority | Effort | Dependencies |
|------|------|----------|--------|--------------|
| [T-042](T-042-wazero-wasm-runtime-setup.md) | Wazero WASM Runtime Setup and Grammar Embedding | Must Have | Medium (8-12hrs) | T-001 |
| [T-043](T-043-language-detection-compressor-interface.md) | Language Detection and LanguageCompressor Interface | Must Have | Small (3-4hrs) | T-003 |
| [T-044](T-044-tier1-ts-js-compressor.md) | Tier 1 Compressor -- TypeScript and JavaScript | Must Have | Large (16-20hrs) | T-042, T-043 |
| [T-045](T-045-tier1-go-compressor.md) | Tier 1 Compressor -- Go | Must Have | Medium (8-12hrs) | T-042, T-043 |
| [T-046](T-046-tier1-python-rust-compressor.md) | Tier 1 Compressor -- Python and Rust | Must Have | Large (14-18hrs) | T-042, T-043 |
| [T-047](T-047-tier2-java-c-cpp-compressor.md) | Tier 2 Compressor -- Java, C, and C++ | Should Have | Medium (10-14hrs) | T-042, T-043 |
| [T-048](T-048-tier2-config-compressor-fallback.md) | Tier 2 Config Compressors (JSON/YAML/TOML) & Fallback | Must Have | Small (4-6hrs) | T-043 |
| [T-049](T-049-compression-orchestrator.md) | Compression Orchestrator and Pipeline Integration | Must Have | Medium (10-14hrs) | T-044, T-045, T-046, T-048 |
| [T-050](T-050-regex-fallback-compression-tests.md) | Regex Heuristic Fallback and E2E Compression Tests | Must Have | Medium (10-14hrs) | T-049 |

**Phase index:** [PHASE-5-INDEX.md](PHASE-5-INDEX.md)

---

## Phase 6: Output & Rendering (T-051 -- T-058)

> Markdown/XML output, directory tree, content hashing, splitting

| Task | Name | Priority | Effort | Dependencies |
|------|------|----------|--------|--------------|
| [T-051](T-051-directory-tree-builder.md) | Directory Tree Builder (In-Memory Tree + Rendering) | Must Have | Medium (8-12hrs) | None |
| [T-052](T-052-markdown-renderer.md) | Markdown Output Renderer with Go Templates | Must Have | Large (14-20hrs) | T-051 |
| [T-053](T-053-xml-renderer.md) | XML Output Renderer for Claude Target Preset | Must Have | Medium (8-12hrs) | T-052 |
| [T-054](T-054-content-hash-deterministic.md) | Content Hashing (XXH3) and Deterministic Output | Must Have | Small (3-4hrs) | None |
| [T-055](T-055-output-writer-stdout.md) | Output Writer, File Path Resolution, and Stdout Support | Must Have | Medium (6-8hrs) | T-052, T-054 |
| [T-056](T-056-output-splitter.md) | Output Splitter (Multi-Part File Generation) | Should Have | Medium (8-12hrs) | T-052, T-055 |
| [T-057](T-057-metadata-sidecar.md) | Metadata JSON Sidecar Generation | Should Have | Small (3-4hrs) | T-054, T-055 |
| [T-058](T-058-output-integration-golden-tests.md) | Output Pipeline Integration and Golden Tests | Must Have | Medium (8-12hrs) | T-051-T-057 |

---

## Phase 7: State & Diff (T-059 -- T-065)

> State caching, differential output, git-aware diffing

| Task | Name | Priority | Effort | Dependencies |
|------|------|----------|--------|--------------|
| [T-059](T-059-state-snapshot-types.md) | State Snapshot Types and JSON Serialization | Should Have | Small (2-4hrs) | T-001 |
| [T-060](T-060-content-hashing-xxh3.md) | Content Hashing with XXH3 | Should Have | Small (2-4hrs) | T-059 |
| [T-061](T-061-state-cache-persistence.md) | State Cache Persistence (Read/Write) | Should Have | Medium (6-12hrs) | T-059, T-060 |
| [T-062](T-062-state-comparison-engine.md) | State Comparison Engine (Added/Modified/Deleted) | Should Have | Medium (6-12hrs) | T-059, T-060, T-061 |
| [T-063](T-063-git-aware-diffing.md) | Git-Aware Diffing | Should Have | Medium (6-12hrs) | T-062 |
| [T-064](T-064-diff-subcommand.md) | `harvx diff` Subcommand and `--diff-only` Flag | Should Have | Medium (6-12hrs) | T-062, T-063 |
| [T-065](T-065-cache-subcommands-change-summary.md) | Cache Subcommands and Change Summary Rendering | Should Have | Medium (6-12hrs) | T-061, T-062, T-064 |

---

## Phase 8: Workflows (T-066 -- T-078)

> Pipeline library, review workflows, session bootstrap, workspace

| Task | Name | Priority | Effort | Dependencies |
|------|------|----------|--------|--------------|
| [T-066](T-066-pipeline-library-api.md) | Core Pipeline as Go Library API | Must Have | Large (14-20hrs) | T-003, T-016, T-017 |
| [T-067](T-067-stdout-exit-codes-headless.md) | Stdout Mode, Exit Codes, Non-Interactive Defaults | Must Have | Medium (6-10hrs) | T-066 |
| [T-068](T-068-json-preview-output-metadata.md) | JSON Preview Output and Metadata Sidecar | Must Have | Medium (8-12hrs) | T-066, T-067 |
| [T-069](T-069-assert-include-env-overrides.md) | Assert-Include Coverage Checks & Env Var Overrides | Must Have | Medium (6-10hrs) | T-066 |
| [T-070](T-070-brief-command.md) | Repo Brief Command (`harvx brief`) | Must Have | Large (14-20hrs) | T-066, T-067, T-068 |
| [T-071](T-071-review-slice-command.md) | Review Slice Command (`harvx review-slice`) | Must Have | Large (16-24hrs) | T-070 |
| [T-072](T-072-slice-command.md) | Module Slice Command (`harvx slice`) | Must Have | Medium (8-12hrs) | T-066, T-067 |
| [T-073](T-073-workspace-config-command.md) | Workspace Manifest Config and Command | Must Have | Medium (10-14hrs) | T-016, T-066 |
| [T-074](T-074-session-bootstrap-docs.md) | Session Bootstrap Docs & Claude Code Hooks | Must Have | Medium (6-10hrs) | T-070 |
| [T-075](T-075-verify-command.md) | Verify Command (`harvx verify`) | Must Have | Medium (8-12hrs) | T-066 |
| [T-076](T-076-golden-questions-harness.md) | Golden Questions Harness & Quality Evaluation | Should Have | Medium (8-12hrs) | T-070, T-071, T-075 |
| [T-077](T-077-mcp-server.md) | MCP Server v1.1 (`harvx mcp serve`) | Nice to Have | Large (16-24hrs) | T-066, T-070, T-072 |
| [T-078](T-078-workflow-integration-tests.md) | Workflow Integration Tests (E2E) | Must Have | Medium (10-14hrs) | T-070, T-071, T-072, T-073 |

**Phase index:** [PHASE-8-INDEX.md](PHASE-8-INDEX.md)

---

## Phase 9: Interactive TUI (T-079 -- T-087)

> Bubble Tea terminal UI with file tree, live token counting

| Task | Name | Priority | Effort | Dependencies |
|------|------|----------|--------|--------------|
| [T-079](T-079-bubbletea-app-scaffold.md) | Bubble Tea Application Scaffold & Elm Architecture | Must Have | Medium (8-12hrs) | T-001, T-003 |
| [T-080](T-080-file-tree-model-navigation.md) | File Tree Data Model & Keyboard Navigation | Must Have | Large (14-20hrs) | T-079 |
| [T-081](T-081-file-tree-visual-rendering.md) | File Tree Visual Rendering & Tier Color Coding | Must Have | Medium (8-12hrs) | T-080 |
| [T-082](T-082-stats-panel-live-tokens.md) | Stats Panel with Live Token Counting & Budget Bar | Must Have | Medium (8-12hrs) | T-079, T-029 |
| [T-083](T-083-profile-selector-actions.md) | Profile Selector & Action Keybindings | Must Have | Medium (6-10hrs) | T-079, T-016 |
| [T-084](T-084-lipgloss-styling-layout.md) | Lipgloss Styling, Responsive Layout & Theme Support | Must Have | Medium (8-12hrs) | T-079-T-083 |
| [T-085](T-085-search-filter-help-overlay.md) | Search/Filter, Tier Views & Help Overlay | Must Have | Medium (8-12hrs) | T-080, T-081 |
| [T-086](T-086-tui-state-serialization.md) | TUI State Serialization to Profile TOML & Smart Default | Must Have | Small (4-6hrs) | T-079, T-016 |
| [T-087](T-087-tui-integration-testing.md) | TUI Integration Testing & Pipeline Wiring | Must Have | Medium (6-10hrs) | T-079-T-086 |

---

## Phase 10: Polish & Distribution (T-088 -- T-095)

> Cross-platform builds, testing infrastructure, release automation

| Task | Name | Priority | Effort | Dependencies |
|------|------|----------|--------|--------------|
| [T-088](T-088-goreleaser-cosign-sbom.md) | GoReleaser Config with Cosign Signing & Syft SBOM | Must Have | Medium (8-12hrs) | T-001, T-002 |
| [T-089](T-089-github-release-automation.md) | GitHub Release Automation Workflow | Must Have | Small (4-6hrs) | T-088 |
| [T-090](T-090-shell-completions-man-pages.md) | Shell Completion Generation & Man Pages | Must Have | Medium (6-8hrs) | T-005 |
| [T-091](T-091-performance-benchmarks.md) | Performance Benchmarking Suite | Must Have | Medium (8-12hrs) | T-015, T-031, T-049 |
| [T-092](T-092-integration-tests-oss-repos.md) | Integration Test Suite Against Real OSS Repos | Must Have | Medium (8-12hrs) | T-015, T-031, T-049, T-052 |
| [T-093](T-093-fuzz-testing-redaction-config.md) | Fuzz Testing for Redaction & Config Parsing | Must Have | Medium (6-10hrs) | T-037, T-016 |
| [T-094](T-094-golden-test-infrastructure.md) | Golden Test Infrastructure | Must Have | Medium (8-12hrs) | T-015, T-052 |
| [T-095](T-095-doctor-command-readme-docs.md) | Doctor Command & README Documentation | Should Have | Medium (6-10hrs) | T-005, T-015, T-020 |

---

## Effort Summary

| Effort Level | Count | Hours Range |
|-------------|-------|-------------|
| Small (2-6hrs) | 18 | 36-90 hrs |
| Medium (6-14hrs) | 63 | 378-754 hrs |
| Large (14-24hrs) | 14 | 196-308 hrs |
| **Total** | **95** | **~610-1152 hrs** |

---

## Priority Summary

| Priority | Count |
|----------|-------|
| Must Have | 76 |
| Should Have | 16 |
| Nice to Have | 3 |

---

## PRD Section Mapping

| PRD Section | Tasks |
|-------------|-------|
| 5.1 Core File Discovery | T-011, T-012, T-013, T-014, T-015 |
| 5.2 Profile System | T-016, T-017, T-018, T-019, T-020, T-021, T-022, T-023, T-024, T-025 |
| 5.3 Relevance Sorting | T-026, T-027, T-028, T-032 |
| 5.4 Token Counting & Budgeting | T-029, T-030, T-031, T-033 |
| 5.5 Secret Redaction | T-034, T-035, T-036, T-037, T-038, T-039, T-040, T-041 |
| 5.6 Tree-Sitter Compression | T-042, T-043, T-044, T-045, T-046, T-047, T-048, T-049, T-050 |
| 5.7 Output Rendering | T-052, T-053, T-055, T-056, T-057, T-058 |
| 5.8 State Caching & Diff | T-059, T-060, T-061, T-062, T-063, T-064, T-065 |
| 5.9 CLI Interface | T-005, T-006, T-007, T-008, T-009, T-010, T-090, T-095 |
| 5.10 Pipeline Integration | T-066, T-067, T-068, T-069, T-078 |
| 5.11.1 Review Pipelines | T-070, T-071, T-075, T-076 |
| 5.11.2 Session Bootstrap | T-072, T-074, T-077 |
| 5.11.3 Workspace Manifest | T-073 |
| 5.12 Directory Tree | T-051 |
| 5.13 Interactive TUI | T-079, T-080, T-081, T-082, T-083, T-084, T-085, T-086, T-087 |
| 6.x Architecture | T-001, T-002, T-003, T-004, T-054 |
| 7.x Security | T-038, T-041 |
| 9.x Testing | T-091, T-092, T-093, T-094 |
| 10.x Distribution | T-088, T-089 |

---

## Development Phase Dependency Graph

```
Phase 1: Foundation (Weeks 1-3)
  T-001 -> T-002, T-003, T-004
  T-003 + T-004 + T-002 -> T-005
  T-005 -> T-006, T-007, T-009, T-010
  T-007 -> T-008
  T-011 -> T-012 -> T-014 -> T-015

Phase 2: Profiles (Week 4)
  T-016 -> T-017 -> T-019 -> T-020 -> T-022 -> T-025

Phase 3: Relevance & Tokens (Weeks 5-6)
  Relevance: T-026 -> T-027 -> T-028
  Tokens: T-029 -> T-030 -> T-031

Phase 4: Security (Weeks 7-8)
  T-034 -> T-035 -> T-037 -> T-040 -> T-041

Phase 5: Compression (Week 9)
  T-042 + T-043 -> T-044/T-045/T-046 -> T-049 -> T-050

Phase 6: Output & Rendering (Week 10)
  T-051 -> T-052 -> T-053, T-055 -> T-056, T-057 -> T-058

Phase 7: State & Diff (Week 11)
  T-059 -> T-060 -> T-061 -> T-062 -> T-063 -> T-064 -> T-065

Phase 8: Workflows (Week 12)
  T-066 -> T-067, T-069 -> T-070 -> T-071 -> T-078

Phase 9: Interactive TUI (Week 13)
  T-079 -> T-080 -> T-081, T-082, T-083 -> T-084 -> T-087

Phase 10: Polish & Distribution (Week 14)
  T-088 -> T-089
  T-091, T-092, T-093, T-094, T-095 (parallel)
```

---

_Last updated: 2026-02-16_
