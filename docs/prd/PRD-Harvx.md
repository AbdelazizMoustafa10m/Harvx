# Product Requirements Document: Harvx

**Version:** 2.0
**Author:** Zizo
**Date:** February 16, 2026
**Status:** Draft — Ready for Development Planning

---

## 1. Executive Summary

Harvx is a blazing-fast, Go-based CLI tool that packages codebases into LLM-optimized context documents. It solves the problem of manually assembling project context for AI-assisted development workflows — from feeding code to chat interfaces to powering multi-agent review pipelines to bootstrapping fresh coding-agent sessions with project awareness.

What sets Harvx apart from the 37+ existing tools in this space is its **profile system**, which allows developers to define project-specific context strategies (relevance tiers, priority files, custom ignores) alongside a universal default profile that works on any repository. Combined with **first-class workflow support** (review pipelines, session bootstrap, multi-repo workspaces), **secret redaction**, **tree-sitter code compression via WASM**, **token budgeting**, and **state-based diffing**, Harvx is designed to be both a personal productivity tool and a core component of automated AI code review infrastructure.

Harvx compiles to a single, zero-dependency binary (~15–20MB) that runs on macOS, Linux, and Windows with no CGO requirements. It operates in two modes: **headless** (default, optimized for pipelines and automation) and **interactive TUI** (visual file selection with real-time token counting, optimized for exploration and one-off use).

**Tagline:** *Harvest your context.*

---

## 2. Problem Statement

Developers working with LLMs face a recurring friction: getting the right codebase context into AI tools efficiently, securely, and consistently.

**Current pain points:**

- **Manual copy-pasting is tedious and error-prone.** Developers spend significant time selecting, copying, and formatting code files for LLM prompts. This is especially painful for large codebases with hundreds of files.
- **Existing tools lack project awareness.** Most context builders treat every repository identically — they don't know that `lib/services/` is more important than `public/assets/` in your specific project. This leads to context dilution where the LLM receives irrelevant files that consume precious token budget.
- **No integration with automated review pipelines.** Tools like Repomix and code2prompt are designed for manual, one-off usage. They don't integrate into multi-agent code review or performance audit workflows where context needs to be generated programmatically as part of a CI/CD or review orchestration pipeline.
- **Diff-only reviews miss the big picture.** Automated AI code reviews that only see the git diff and changed files lack project-wide invariants — architecture rules, auth patterns, data ownership, API contracts. They review in isolation, catching style nits but missing cross-module constraint violations.
- **Fresh agent sessions start from zero.** Every new Claude Code or Codex CLI session lacks project context. Developers either write massive `CLAUDE.md` files (degrading agent performance) or repeat the same explanations every session.
- **Multi-repo relationships require constant re-explanation.** Developers working across multiple related repositories spend significant time explaining repo relationships, paths, and integration points in every conversation.
- **Security risks from accidental credential exposure.** Without built-in secret detection, developers risk pasting API keys, tokens, and passwords into LLM prompts — especially dangerous when using cloud-hosted AI services.
- **Distribution friction.** Most competing tools require Node.js, Python, or Rust toolchains. For a tool that should "just work" on any machine, this dependency overhead is unacceptable.

**The opportunity:** A single Go binary with a profile system that serves as a personal productivity tool (default profile for any repo), an infrastructure component (project-specific profiles integrated into automated pipelines), and a session bootstrap engine (hooks and MCP integration for coding agents).

---

## 3. Target Audience

### 3.1 Primary Users

**Solo developers and small teams** who use AI-assisted development workflows daily. They interact with LLMs through chat interfaces (Claude, ChatGPT, Gemini) and coding agents (Claude Code, Cursor, Aider, Copilot). They need a fast, reliable way to package their codebase for these tools.

- Technically proficient (comfortable with CLI tools and configuration files)
- Work across multiple projects (personal, professional, open-source)
- Value privacy and data control — prefer local tooling over cloud services
- Already use `.gitignore` and understand file filtering concepts

**Developers with multi-agent review pipelines** who orchestrate multiple AI agents for code review, performance auditing, and quality assurance. They need context generation as a programmable step in their automation scripts.

- Advanced users who write shell scripts, CI/CD pipelines, and orchestration logic
- Need deterministic, reproducible output for prompt caching
- Require integration with existing tools (git, diff utilities, shell scripts)

**Coding agent users** who rely on Claude Code, Codex CLI, or similar tools and want every session to start with project awareness without bloating instruction files.

- Range from intermediate to advanced developers
- Work across multiple repos that relate to each other
- Want minimal setup and automatic context injection

### 3.2 User Personas

**Persona 1: Zizo (Power User / Pipeline Integrator)**
A developer managing a complex Next.js application (FinVault) with a sophisticated multi-agent review system. Runs 5 AI agents across 2 passes for code review, with 8-category performance audits. Needs Harvx to generate full codebase context that feeds into these pipelines alongside diffs and review prompts. Also uses Harvx in default mode when exploring unfamiliar open-source repos. Works across multiple related repositories at their day job and needs workspace-level context.

**Persona 2: Alex (Daily AI Chat User)**
A full-stack developer who frequently pastes code into Claude or ChatGPT for debugging, refactoring suggestions, and architecture advice. Currently uses manual copy-paste or basic tools like `cat`. Wants a single command to generate a clean, token-efficient context file they can drag into any chat interface. Appreciates an interactive mode for quickly selecting which files to include.

**Persona 3: Jordan (Team Lead / CI Integrator)**
Manages a team and wants to integrate codebase context generation into GitHub Actions workflows. Needs Harvx to run in CI, generate context artifacts, and feed them to automated review bots. Values the single binary distribution for easy CI setup and the `--fail-on-redaction` flag for enforcing secret hygiene.

---

## 4. Product Objectives & Success Metrics

| Objective | Success Metric | Target |
|-----------|---------------|--------|
| Fast context generation | Processing time for 1,000 files | < 1 second |
| Fast context generation | Processing time for 10,000 files | < 3 seconds |
| Token efficiency | Token reduction with compression enabled | 50–70% reduction vs raw |
| Security | Secret detection on curated regression corpus | Zero known misses for high-confidence patterns |
| Security | False positive rate on test fixtures and docs | < 5% of flagged items |
| Usability | Time from install to first output | < 2 minutes |
| Distribution | Binary size | < 20MB |
| Distribution | External dependencies (runtime) | Zero (pure Go + embedded WASM) |
| Pipeline integration | Works as a step in shell script pipelines | stdout/file output, exit codes |
| Profile flexibility | Time to configure a new project profile | < 5 minutes |
| Agent bootstrap | Session startup context injection latency | < 2 seconds |
| Review quality | Improvement in AI review accuracy with Harvx context vs diff-only | Measurable via golden questions harness |
| Interactive UX | TUI responsiveness during file toggling on 1K+ file repos | Token recount completes in < 300ms |

---

## 5. Feature Specifications

### 5.1 Core File Discovery & Processing

**Priority:** Must-Have
**Description:** Recursively walk a directory tree, discover all text-based files, respect ignore patterns, read file contents, and assemble them into a structured output document.

**User Story:** As a developer, I want to run a single command in any project directory and get a complete, well-structured context file so that I can immediately feed it to an LLM.

**Acceptance Criteria:**
- Discovers all text-based files recursively from the target directory
- Respects `.gitignore` rules (including nested `.gitignore` files)
- Respects `.harvxignore` for tool-specific ignore patterns
- Applies a default set of ignore patterns (e.g., `node_modules/`, `.git/`, `dist/`, `coverage/`, `__pycache__/`, `.next/`, `target/`, `vendor/`, binary files)
- Detects and skips binary files automatically (via content inspection, not just extension)
- Supports explicit include patterns (`--include "src/**/*.ts"`) and exclude patterns (`--exclude "**/*.test.ts"`)
- Supports filtering by file extension (`-f ts -f go -f md`)
- Supports `--git-tracked-only` mode that only includes files in the git index (sidesteps gitignore edge cases, ideal for CI)
- Handles symlinks safely (detect and skip loops)
- Processes files in parallel using `x/sync/errgroup` with bounded concurrency for maximum throughput
- Outputs a clean, well-formatted Markdown (default), XML, or plain text file
- Provides a `--preview` mode that shows the file tree and token estimate without generating output
- Supports `--skip-large-files <size>` (default: 1MB) to skip oversized generated files

**Technical Considerations:**
- Use the `sabhiram/go-gitignore` package for `.gitignore` parsing with full pattern support. Must match Git's behavior for a reference test suite (golden tests against `git check-ignore`)
- Use `filepath.WalkDir` with `x/sync/errgroup.WithContext()` and `errgroup.SetLimit(runtime.NumCPU())` for parallel traversal with proper error propagation and cancellation
- Binary detection: check first 8KB of each file for null bytes (same approach as Git)
- File reading should use buffered I/O with configurable max file size limit
- Maintain deterministic file ordering (sorted by path) for reproducible output

---

### 5.2 Profile System

**Priority:** Must-Have
**Description:** A TOML-based configuration system that supports named profiles, profile inheritance, and per-repository auto-detection. Profiles define relevance tiers, priority files, ignore patterns, output settings, and project-specific behavior.

**User Story:** As a developer working on multiple projects, I want to define project-specific context strategies so that each project's context is optimized for its architecture and my workflow needs.

**Acceptance Criteria:**
- Profiles are defined in TOML format
- Supports three configuration scopes, merged in order of precedence:
  1. **Global config:** `~/.config/harvx/config.toml` (user defaults)
  2. **Repository config:** `harvx.toml` at repository root (project-specific)
  3. **CLI flags:** Always override config file values
- Profiles support an `extends` field for inheritance (e.g., `extends = "default"`)
- The `default` profile is built-in and always available as a baseline
- Auto-detection: if a `harvx.toml` exists in the current directory or any parent directory, it is automatically loaded
- Named profiles are selected via `--profile <name>` flag
- Supports `--profile-file <path>` for loading standalone profile files (team distribution)
- Profile configuration includes:
  - `output`: file path, format (markdown/xml/plain), max tokens
  - `tokenizer`: tokenizer encoding (`cl100k_base` or `o200k_base`)
  - `relevance`: tier-based file prioritization (tiers 0–5)
  - `ignore`: additional ignore patterns
  - `priority_files`: files always included first (tier 0)
  - `include`: explicit include patterns
  - `compression`: enable/disable tree-sitter compression
  - `redaction`: enable/disable secret scanning, path exclusions, confidence threshold
  - `target`: LLM target preset (`claude`, `chatgpt`, `generic`)
- Ships with **framework-specific profile templates** for quick setup:
  - `harvx profiles init --template nextjs`
  - `harvx profiles init --template go-cli`
  - `harvx profiles init --template python-django`
  - `harvx profiles init --template rust-cargo`
  - `harvx profiles init --template monorepo`
- `harvx profiles list` subcommand shows all available profiles
- `harvx profiles init` generates a starter `harvx.toml` in the current directory
- `harvx profiles show <name>` shows resolved configuration for a profile
- `harvx profiles lint` validates patterns, warns on overlapping tier rules or unreachable tiers
- `harvx profiles explain <filepath>` shows which profile/tier/rule applies to a specific file and why it's included/excluded

**Technical Considerations:**
- Use `BurntSushi/toml` for TOML parsing
- Use `spf13/viper` for multi-source config merging (CLI flags → env vars → config file → defaults), or evaluate `koanf/koanf` as a lighter alternative to reduce binary size
- Profile inheritance uses deep merge: child profile values override parent, arrays are replaced (not concatenated)
- Validate profiles at load time and provide clear error messages for invalid configurations
- Warn if profile inheritance exceeds 3 levels deep ("consider flattening")

**Example Configuration:**

```toml
# harvx.toml (at repository root)

[profile.default]
output = "harvx-output.md"
format = "markdown"
max_tokens = 128000
tokenizer = "cl100k_base"
compression = false
redaction = true
ignore = ["node_modules", "dist", ".git", "coverage", "__pycache__"]

[profile.finvault]
extends = "default"
output = ".harvx/finvault-context.md"
max_tokens = 200000
tokenizer = "o200k_base"
compression = true
target = "claude"

# Files always included first (tier 0)
priority_files = [
  "CLAUDE.md",
  "prisma/schema.prisma",
  ".github/review/rules/finvault-patterns.md",
  ".github/review/rules/review-checklist.md",
]

# Architecture-aware relevance tiers
[profile.finvault.relevance]
tier_0 = ["CLAUDE.md", "prisma/schema.prisma", "*.config.*"]
tier_1 = ["app/api/**", "lib/services/**", "middleware.ts"]
tier_2 = ["components/**", "hooks/**", "lib/**"]
tier_3 = ["__tests__/**"]
tier_4 = ["docs/**", "prompts/**"]
tier_5 = [".github/**", "*.lock"]

# Redaction with path exclusions
[profile.finvault.redaction]
enabled = true
exclude_paths = ["**/*test*/**", "**/fixtures/**", "docs/**/*.md"]
confidence_threshold = "high"

# Extra ignores
ignore = [
  "reports/",
  ".review-workspace/",
  ".harvx/",
  ".next/",
]

[profile.work]
extends = "default"
output = ".harvx/work-context.md"
max_tokens = 150000
target = "claude"
priority_files = ["README.md", "docs/architecture.md"]

[profile.session]
extends = "default"
output = ".harvx/session-brief.md"
max_tokens = 8000
compression = true
# Lean profile for agent bootstrap — only the essentials
```

---

### 5.3 Relevance-Based File Sorting

**Priority:** Must-Have
**Description:** Sort discovered files into priority tiers so that the most important files appear first in the output. When token budget is limited, lower-tier files are truncated or omitted first.

**User Story:** As a developer, I want my configuration files and core source code to always appear before tests and CI config in the context output so that the LLM sees the most important context first within its token limit.

**Acceptance Criteria:**
- Files are sorted into 6 priority tiers (0 = highest, 5 = lowest)
- Default tier assignments (when no profile-specific relevance is configured):
  - **Tier 0:** Configuration files (package.json, tsconfig.json, Cargo.toml, go.mod, Makefile, Dockerfile, *.config.*)
  - **Tier 1:** Source code in primary language directories (src/, lib/, app/, cmd/, internal/, pkg/)
  - **Tier 2:** Secondary source files, components, utilities
  - **Tier 3:** Test files (*_test.go, *.test.ts, *.spec.js, __tests__/)
  - **Tier 4:** Documentation (*.md, docs/, README*)
  - **Tier 5:** CI/CD (.github/, .gitlab-ci.yml), lock files (*.lock, package-lock.json)
- Profile-defined relevance tiers override the defaults entirely
- Within each tier, files are sorted alphabetically by path for deterministic output
- When `max_tokens` is set, files are included tier-by-tier until the budget is exhausted
- Files that exceed the remaining budget are either truncated (with a marker) or skipped, based on a `--truncation-strategy` flag (`truncate` | `skip`, default: `skip`)
- The output includes a summary of included/excluded files with tier information

**Technical Considerations:**
- Relevance matching uses glob patterns (same engine as ignore patterns via `bmatcuk/doublestar`)
- File-to-tier matching should be O(n × m) where n = files, m = patterns — acceptable for typical repos
- A file matches the first (highest priority) tier where a pattern matches; no file appears in multiple tiers
- Unmatched files go to tier 2 (source code default) to avoid excluding unexpected but important files

---

### 5.4 Token Counting & Budgeting

**Priority:** Must-Have
**Description:** Accurately count tokens in the output using a real tokenizer, and enforce a configurable token budget that truncates or omits lower-priority content when the limit is reached.

**User Story:** As a developer, I want to know exactly how many tokens my context file will consume so that I can stay within my LLM's context window and avoid wasted tokens.

**Acceptance Criteria:**
- Supports multiple tokenizer encodings via `--tokenizer` flag and profile config:
  - `cl100k_base` (GPT-4, Claude compatible) — default
  - `o200k_base` (GPT-4o, o1, newer OpenAI models)
  - `none` (fallback to character/byte estimates for speed)
- Supports a `--max-tokens` flag and profile-level `max_tokens` setting
- When budget is exceeded, lower-tier files are omitted first (respecting relevance order)
- Provides per-file token counts in the output header/summary section
- Provides total token count for the complete output
- Supports a `--token-count` mode that only reports token counts without generating output
- Supports a `--top-files <n>` flag to show the N largest files by token count
- Supports a `--heatmap` flag on the preview command that shows files sorted by token density (tokens per line of code) to identify context-bloat files
- Labels the tokenizer used in output metadata for pipeline debuggability

**Technical Considerations:**
- Use `pkoukk/tiktoken-go` for tokenization in Go
- Design a `Tokenizer` interface so additional encodings can be added later
- Token counting runs in parallel with file reading (count as files are loaded, not as a separate pass)
- The summary section itself consumes tokens — account for this overhead when budgeting
- For compressed output (tree-sitter), count tokens on the compressed content, not the original

---

### 5.5 Secret & Credential Redaction

**Priority:** Must-Have
**Description:** Automatically detect and redact sensitive information (API keys, tokens, passwords, private keys, connection strings) before they appear in the output. Security-first: this should be on by default.

**User Story:** As a developer, I want Harvx to automatically detect and redact secrets in my codebase so that I never accidentally expose credentials when sharing context with cloud-hosted LLMs.

**Acceptance Criteria:**
- Enabled by default (can be disabled with `--no-redact` or profile setting `redaction.enabled = false`)
- Detection patterns are modeled after the Gitleaks ruleset (proven, versioned, well-tested) and include:
  - **API keys & tokens:** AWS access keys (`AKIA...`), GitHub tokens (ghp_*, gho_*, github_pat_*), Stripe keys (sk_live_*, pk_live_*), OpenAI keys (sk-*), generic patterns (api_key=, api-key=, apikey=)
  - **Private keys:** RSA/DSA/EC/Ed25519 private key blocks (-----BEGIN * PRIVATE KEY-----)
  - **Connection strings:** Database URLs (postgres://, mysql://, mongodb://), Redis URLs
  - **Passwords & secrets:** password=, secret=, token= assignments in config files
  - **Cloud provider credentials:** AWS secret access keys, GCP service account JSON, Azure connection strings
  - **JWT tokens:** eyJ pattern with valid base64 segments
- Redacted content is replaced with `[REDACTED:type]` (e.g., `[REDACTED:aws_access_key]`, `[REDACTED:private_key_block]`)
- Supports **path exclusions** in profile config to reduce false positives:
  ```toml
  [profile.default.redaction]
  enabled = true
  exclude_paths = ["**/*test*/**", "**/fixtures/**", "**/examples/**", "docs/**/*.md"]
  confidence_threshold = "high"  # Skip low-confidence matches
  ```
- Produces a redaction summary in the output: number of redactions by type
- Supports a `--redaction-report` flag that outputs a detailed report of what was redacted and where (file + line number, without showing the actual secret)
- Supports a `--fail-on-redaction` flag for CI: exit code 1 if any secrets detected (team enforcement)
- Supports custom redaction patterns via profile configuration
- Aims for zero known misses against a curated regression test corpus for high-confidence patterns (private key blocks, AWS key IDs, Stripe live keys)
- Minimizes false positives via path exclusions, confidence thresholds, and entropy analysis

**Technical Considerations:**
- Implement as a streaming filter: each file's content passes through the redaction pipeline before being written to output
- Use compiled regular expressions for performance (compile once, apply to all files)
- Pattern matching should be case-insensitive for key names but case-sensitive for key values
- Entropy analysis as a secondary signal for detecting high-entropy strings (configurable threshold)
- `.env` files and `*.pem` files trigger heightened scanning regardless of content
- Redaction runs before token counting (count the redacted output, not the original)
- Maintain a versioned test corpus of realistic secret formats for regression testing

---

### 5.6 Tree-Sitter Code Compression (via WASM)

**Priority:** Must-Have
**Description:** Use tree-sitter AST parsing to extract structural signatures (function signatures, class declarations, type definitions, import statements) from source files, producing a compressed representation that preserves code structure while reducing token count by 50–70%.

**Critical Design Decision:** Tree-sitter grammars are executed as **WebAssembly modules** via the `wazero` runtime (pure Go, zero CGO). Grammar `.wasm` files are embedded in the binary using `//go:embed`. This preserves the zero-dependency, single-binary distribution model while enabling real AST parsing.

**User Story:** As a developer working with token-limited LLMs, I want to compress my codebase to its structural essence so that I can fit more of my project into a single context window while the LLM still understands the architecture.

**Acceptance Criteria:**
- Activated via `--compress` flag or profile setting `compression = true`
- Supports the following languages at launch:
  - **Tier 1 (full support):** TypeScript, JavaScript, Go, Python, Rust
  - **Tier 2 (basic support):** Java, C, C++, JSON, YAML, TOML
- For supported languages, extracts:
  - Function/method signatures (name, parameters, return type)
  - Class/struct/interface declarations with field types
  - Type aliases and enum definitions
  - Import/require statements
  - Export statements
  - Top-level constant declarations
  - Doc comments only (skip inline comments)
- For unsupported languages, falls back to full file content (no compression)
- Compressed output is clearly marked: `<!-- Compressed: signatures only -->` header per file
- Achieves 50–70% token reduction on typical source files
- Compression **never alters semantics** — it extracts verbatim source text at AST node boundaries, it does not "summarize" or rewrite code
- Supports a `--compress-timeout <ms>` flag to abandon slow parsing operations (default: 5000ms per file)

**Technical Considerations:**
- Use `wazero` (github.com/tetratelabs/wazero) as the WebAssembly runtime — pure Go, zero CGO, no system dependencies
- Embed `.wasm` grammar files using `//go:embed` directives
- Use `tree-sitter-grammars` WASM builds (available from tree-sitter's official releases)
- If WASM approach proves inadequate, fallback plan: regex-based heuristic signature extraction per language
- Language detection by file extension (reliable for most cases)
- Compression is applied per-file — each file's compressed output is independent
- Lazy loading: only parse files that survive token budget filtering (don't compress files that will be omitted)
- Define a `LanguageCompressor` interface internally for future extensibility (new languages via build tags or external WASM files)

---

### 5.7 Output Rendering

**Priority:** Must-Have
**Description:** Generate well-structured output documents in Markdown format (v1) with metadata headers, file tree visualizations, and clear file separators optimized for LLM consumption.

**User Story:** As a developer, I want the output to be clean, well-organized, and easy for LLMs to parse so that I get better responses when I share my codebase context.

**Acceptance Criteria:**
- **v1 formats:** Markdown (default) and XML (for Claude target preset)
- **v1.1 formats:** Plain text, JSON
- Output structure (Markdown example):
  1. **Header block**: Project name, generation timestamp, content hash, profile used, tokenizer used, token count
  2. **File summary**: Total files, total tokens, files by tier, top N largest files, redaction summary
  3. **Directory tree**: Visual tree structure (├── / └──) of included files
  4. **File contents**: Each file with path, size, token count, tier, and content (or compressed signatures)
  5. **Change summary** (if diff mode): Added, modified, deleted files since last run
- **LLM target presets** via `--target` flag:
  - `claude` → XML format following Anthropic's recommended XML tag structure, 200K default budget
  - `chatgpt` → Markdown format, 128K default budget
  - `generic` → Markdown format, no preset budget (default)
- Output file path is configurable via `--output` / `-o` flag or profile setting
- Default output file: `harvx-output.md` in the current directory
- Supports `--stdout` for piping output to other tools
- Supports `--split <tokens-per-part>` producing `harvx-output.part-001.md`, `part-002.md`, etc. Respects tier boundaries (no file split across parts)
- Deterministic output: uses content hash (XXH3) instead of timestamps for the content hash field, enabling LLM prompt caching
- Line numbers can be optionally added to code blocks via `--line-numbers`
- **Metadata JSON sidecar** via `--output-metadata` flag: produces a `.meta.json` file alongside the context file containing version, profile, tokenizer, content hash, per-file stats (path, tier, tokens, redacted), and aggregate statistics

**Technical Considerations:**
- Use Go `text/template` for output rendering with embedded templates
- Directory tree generation: recursive walk with depth tracking and Unicode box-drawing characters
- Content hash: compute XXH3 hash over all file contents (sorted by path) using `cespare/xxhash`
- XML output follows Repomix's proven XML structure and Anthropic's XML tag best practices
- Streaming output: write to output file as files are processed (don't hold entire output in memory)
- Split output groups files by top-level directory to maintain context; a single file is never split across parts

---

### 5.8 State Caching & Differential Output

**Priority:** Should-Have
**Description:** Cache project state (file paths, sizes, content hashes) between runs and generate diffs showing what changed. This enables incremental reviews where the LLM only needs to see what's new. Supports both filesystem-based and git-based diffing.

**User Story:** As a developer running iterative code reviews, I want to see only what changed since my last context generation so that I can give my AI agents focused, relevant context instead of the full codebase every time.

**Acceptance Criteria:**
- After each run, saves a state snapshot to `.harvx/state/<profile-name>.json`
- State includes: file paths, file sizes, content hashes (XXH3), generation timestamp, git branch and HEAD SHA (if in a git repo)
- On subsequent runs, compares current state to cached state and identifies: added files, modified files, deleted files
- Supports `--diff-only` flag that outputs only changed files (with full content for modified files)
- Supports **git-aware diffing**:
  - `harvx diff --since HEAD~1` — diff against a specific git ref
  - `harvx diff --since <sha>` — diff against a specific commit
  - `harvx diff --base <ref> --head <ref>` — diff between two refs (for PR reviews)
- Supports `--clear-cache` flag to reset state and force a full generation
- The change summary section shows: files added (count), modified (count), deleted (count), with file paths
- State files are gitignored by default (Harvx adds `.harvx/` to its default ignore list)
- State files are profile-scoped and include branch metadata to avoid stale diffs on branch switches

**Technical Considerations:**
- State storage: JSON file with map of `filepath -> {size, hash, modified_time}` plus git metadata
- Content hashing: XXH3 for speed (same as output content hash)
- Diff generation: use `sergi/go-diff` for unified diffs between cached and current content
- For large repos, state comparison should be O(n) — iterate current files, check against hash map

---

### 5.9 CLI Interface & Subcommands

**Priority:** Must-Have
**Description:** A clean, intuitive CLI with sensible defaults, subcommands for different operations, comprehensive help text, and intelligent shell completions.

**User Story:** As a developer, I want to run `harvx` with zero configuration and get useful output immediately, while having the option to customize everything when I need to.

**Acceptance Criteria:**
- Root command: `harvx` (runs context generation with auto-detected or default profile)
- **Core subcommands:**
  - `harvx generate` (alias: `harvx gen`) — Explicit generation command (same as root)
  - `harvx brief` — Generate a stable, small "Repo Brief" for session bootstrap and review pipelines
  - `harvx review-slice --base <ref> --head <ref>` — Generate PR-specific context slice with bounded neighborhood
  - `harvx slice --path <module>` — Generate targeted context for a specific module/directory
  - `harvx preview` — Show file tree, token estimates, tier breakdown, and optional heatmap
  - `harvx diff` — Generate differential output (replaces `--diff-only` flag on root)
- **Profile management:**
  - `harvx profiles list` — Show all available profiles
  - `harvx profiles init [--template <framework>]` — Generate a starter `harvx.toml`
  - `harvx profiles show <name>` — Show resolved configuration for a profile
  - `harvx profiles lint` — Validate profiles, warn on issues
  - `harvx profiles explain <filepath>` — Show which rules apply to a file
- **Config & diagnostics:**
  - `harvx config debug` — Show resolved config with source annotations (which value came from CLI vs config vs default)
  - `harvx doctor` — Check for repo issues (large binaries not ignored, huge files, config problems)
  - `harvx cache clear [--profile <name>]` — Clear cached state
  - `harvx cache show` — Show cached state summary
  - `harvx version` — Show version, build info, supported languages, and tokenizer info
- **Workspace:**
  - `harvx workspace` — Render workspace manifest (multi-repo context)
- **MCP server (v1.1):**
  - `harvx mcp serve` — Start MCP server for coding agent integration
- **Shell completions:**
  - `harvx completion <shell>` — Generate completions for bash, zsh, fish, PowerShell
  - Intelligent completions: `harvx --profile <TAB>` lists available profiles, `--format <TAB>` lists formats, `--target <TAB>` lists presets
- **Global flags:**
  - `-d, --dir <path>` — Target directory (default: current directory)
  - `-o, --output <path>` — Output file path
  - `-p, --profile <name>` — Profile to use
  - `--profile-file <path>` — Load profile from external file
  - `-f, --filter <ext>` — Filter by file extension (repeatable)
  - `--include <pattern>` — Include glob pattern (repeatable)
  - `--exclude <pattern>` — Exclude glob pattern (repeatable)
  - `--format <type>` — Output format: markdown, xml (default: markdown)
  - `--target <preset>` — LLM target: claude, chatgpt, generic
  - `--max-tokens <n>` — Token budget
  - `--tokenizer <encoding>` — Tokenizer: cl100k_base, o200k_base, none
  - `--compress` — Enable tree-sitter compression
  - `--no-redact` — Disable secret redaction
  - `--fail-on-redaction` — Exit 1 if secrets detected (CI mode)
  - `--git-tracked-only` — Only include files in git index
  - `--line-numbers` — Add line numbers to code blocks
  - `--stdout` — Output to stdout instead of file
  - `--split <size>` — Split output into chunks (e.g., `1mb`, `50000` tokens)
  - `--output-metadata` — Generate `.meta.json` sidecar
  - `--skip-large-files <size>` — Skip files larger than threshold (default: 1MB)
  - `--assert-include <pattern>` — Fail if pattern matches no included files (coverage check)
  - `--yes` — Skip confirmation prompts
  - `-i, --interactive` — Launch interactive TUI mode
  - `--verbose` — Verbose logging (debug level, per-stage timings)
  - `--quiet` — Suppress all output except errors
  - `--clear-cache` — Clear cached state before running
- **Exit codes:** 0 (success), 1 (error or `--fail-on-redaction` triggered), 2 (partial — some files failed but output was generated)
- **Canonical recipes in documentation** per persona:
  - Alex (quick use): `harvx` (launches TUI if no config) or `harvx -i` or `harvx --compress`
  - Zizo (pipeline): `harvx brief --profile finvault && harvx review-slice --base main --head HEAD`
  - Jordan (CI): `harvx --profile ci --fail-on-redaction --output-metadata --quiet`

**Technical Considerations:**
- Use `spf13/cobra` for CLI framework — provides subcommands, auto-generated help, shell completions, and man pages
- Use `spf13/viper` (or `koanf`) for configuration management with env var support (prefix: `HARVX_`)
- Progress output and logs go to stderr so stdout remains clean for piping
- Color output via `charmbracelet/lipgloss` (declarative terminal styling, auto-disabled when piped)
- Progress bars via `schollz/progressbar` for long operations on large repos
- Environment variable overrides: `HARVX_PROFILE`, `HARVX_MAX_TOKENS`, `HARVX_LOG_FORMAT=json`, etc.

---

### 5.10 Multi-Agent Pipeline Integration

**Priority:** Must-Have
**Description:** Design Harvx's output and behavior to integrate seamlessly with multi-agent code review and audit pipelines, specifically supporting shell script orchestration and programmatic consumption.

**User Story:** As a developer with a multi-agent review pipeline, I want Harvx to generate context that my review scripts can consume alongside diffs and prompts so that my AI agents have full codebase awareness when reviewing changes.

**Acceptance Criteria:**
- Clean stdout output mode (`--stdout`) for piping into other tools
- Structured exit codes for programmatic error handling
- Machine-readable output via `--format xml` or `--output-metadata`
- Can be invoked as a single step in shell scripts:
  ```bash
  # In code-review.sh, before running agents:
  harvx brief --profile finvault -o .review-workspace/repo-brief.md
  harvx review-slice --base main --head HEAD -o .review-workspace/review-slice.md
  ```
- Supports environment variable overrides for all config (prefix: `HARVX_`)
- JSON metadata output for orchestration decisions:
  ```bash
  harvx preview --json
  ```
  Returns:
  ```json
  {
    "total_files": 342,
    "total_tokens": 89420,
    "tokenizer": "o200k_base",
    "tiers": {"0": 5, "1": 48, "2": 180, "3": 62, "4": 30, "5": 17},
    "redactions": 3,
    "estimated_time_ms": 850
  }
  ```
- Non-interactive by default (no prompts when used in scripts)
- Supports `--assert-include` patterns for coverage checks (fail if critical files are missing)

**Technical Considerations:**
- All user-facing messages go to stderr; only the context output goes to stdout (when `--stdout` is used)
- JSON metadata output uses Go's `encoding/json` with proper struct tags
- Design the core pipeline as a **Go library** (not just CLI glue) so it can also be invoked programmatically — this enables the future MCP server to use the same engine
- Integration testing should cover: shell script invocation, pipe chains, exit code handling, env var overrides

---

### 5.11 Workflow Integrations: Review Pipelines & Fresh-Session Bootstrap

**Priority:** Must-Have
**Description:** Define first-class workflows and artifacts so Harvx can (1) enrich automated AI code review pipelines with repository-level context beyond diffs, (2) bootstrap fresh coding-agent sessions with minimal always-on instructions, and (3) manage multi-repo workspace context.

**Why this matters:** Diff-only reviews and "fresh session" agent workflows frequently miss project-wide invariants (architecture, conventions, contracts, auth rules, config defaults). Harvx fills that gap by generating small, deterministic context artifacts that can be injected automatically at run time.

#### 5.11.1 Workflow A — Automated AI Review (Diff + Repo Context)

**Goal:** Improve automated reviews by combining PR-specific inputs (diff, commit message) with stable, repo-wide grounding ("the project brain").

**The two-artifact approach:**

**Artifact 1 — Repo Brief** (stable, small, always included):
- README / high-level product overview
- Architecture docs/ADRs (if present)
- Build/test commands (Makefile, package scripts, task runners)
- Key invariants: auth/permissions, error handling, logging, data ownership, API contracts
- High-level module map (top directories + purpose)
- Budget: ~1K–4K tokens

**Artifact 2 — Review Slice** (dynamic, per PR/run):
- Changed files (verbatim or compressed per profile)
- Bounded neighborhood context:
  - Public interfaces touched (exported funcs/types, schemas)
  - Config defaults referenced by changes
  - Related tests and fixtures for the changed module(s)
  - Dependency "neighbors" (imports/entrypoints) within a configured limit
- Budget: variable, typically 4K–20K tokens depending on diff size

**Acceptance Criteria:**
- `harvx brief` generates the Repo Brief (cacheable; stable across commits)
- `harvx review-slice --base <ref> --head <ref>` generates the PR-specific slice
- Both artifacts are deterministic (sorted paths, stable rendering, content hash)
- Supports token budgets per artifact (`brief_max_tokens`, `slice_max_tokens` in profile)
- The Review Slice **never summarizes semantics** — it extracts verbatim source at AST node boundaries to avoid creating wrong conclusions
- Supports `--assert-include <pattern>` to verify critical files are present (e.g., auth middleware, schema)
- Both commands support `--json` for machine-readable metadata

**Pipeline integration example:**
```bash
#!/bin/bash
# In code-review.sh, before running agents:

# 1. Generate stable repo context (cacheable)
harvx brief --profile finvault -o .review-workspace/repo-brief.md

# 2. Generate PR-specific slice
harvx review-slice --base origin/main --head HEAD \
  --profile finvault \
  -o .review-workspace/review-slice.md

# 3. Each agent now gets:
#    - repo-brief.md (project invariants, architecture)
#    - review-slice.md (changed code + neighborhood)
#    - git diff (existing)
#    - review prompts (existing)
```

#### 5.11.2 Workflow B — Fresh-Session Bootstrap for Coding Agents

**Goal:** Start every new agent session with enough project context to reduce repetitive explanations, while keeping always-loaded instruction files (e.g., `CLAUDE.md`, `agents.md`) short and stable.

**Strategy:** Keep `CLAUDE.md` / `agents.md` minimal (rules only). Use hooks or tool integrations to run Harvx at session start and inject the generated context.

**Claude Code integration (hooks):**
- Claude Code supports lifecycle hooks (e.g., `SessionStart`) that can run shell commands and inject context at session start
- Best practices recommend keeping `CLAUDE.md` concise to avoid instruction bloat that degrades agent performance

**Acceptance Criteria (Claude Code):**
- Document a reference `SessionStart` hook setup:
  ```json
  {
    "hooks": {
      "SessionStart": [
        {
          "command": "harvx brief --profile session --stdout",
          "timeout": 5000
        }
      ]
    }
  }
  ```
- Provide a "lean baseline" `CLAUDE.md` template in docs (principles only, not architecture dumps)
- Support `--target claude` rendering mode that defaults to XML-friendly structure
- Support `harvx slice --path <module>` for on-demand context (agent calls Harvx when it needs more detail about a specific area)

**Codex CLI integration (MCP):**
- Codex CLI supports Model Context Protocol (MCP) for connecting models to tools/context providers

**Acceptance Criteria (Codex CLI — v1.1):**
- `harvx mcp serve` exposes tools:
  - `harvx.brief(profile=...)` — Generate repo brief
  - `harvx.slice(path=..., profile=...)` — Targeted module context
  - `harvx.review_slice(base=..., head=..., profile=...)` — PR-specific slice
- Agents can request targeted slices on demand rather than loading a massive bundle up front

#### 5.11.3 Workflow C — Multi-Repo Workspace Manifest

**Goal:** Eliminate repeated explanations of how multiple repos relate (paths, responsibilities, integration edges).

**Concept:** A Workspace Manifest describing a set of repositories and their relationships. Harvx renders this into a small "workspace context" section.

**Acceptance Criteria:**
- Support a workspace config file: `.harvx/workspace.toml`
- `harvx workspace` outputs:
  - Repo list with local paths
  - 1–3 line description per repo
  - Key integration edges (API calls, shared schemas/libraries)
  - "Where to look" hints (entrypoints, docs)
- Workspace output is small by default (≤ 1–2K tokens), with optional `--deep` mode
- Workspace context can be included in session bootstrap hooks

**Example workspace config:**
```toml
# .harvx/workspace.toml

[workspace]
name = "MyOrg Platform"
description = "Microservices platform with shared UI library"

[[workspace.repos]]
name = "api-gateway"
path = "~/work/api-gateway"
description = "Express.js API gateway, handles auth and routing"
entrypoints = ["src/server.ts", "src/routes/"]
integrates_with = ["user-service", "billing-service"]

[[workspace.repos]]
name = "user-service"
path = "~/work/user-service"
description = "User management microservice (Go)"
entrypoints = ["cmd/server/main.go", "internal/handlers/"]
integrates_with = ["api-gateway"]
shared_schemas = ["proto/user.proto"]

[[workspace.repos]]
name = "shared-ui"
path = "~/work/shared-ui"
description = "React component library consumed by all frontends"
entrypoints = ["src/index.ts"]
```

#### 5.11.4 Output Quality Evaluation

**Goal:** Provide practical, automatable ways to judge whether Harvx output is helping or hurting LLM task success.

**Quality dimensions:**

1. **Coverage** — Did Harvx include the right files? Missing the file that defines actual behavior (e.g., auth middleware) means the LLM can confidently answer wrong.
2. **Faithfulness** — Did compression or redaction distort meaning? Verbatim extraction at AST boundaries is safe; "summarizing" is not.
3. **Structure & clarity** — Can the model navigate the output? Clear file boundaries, paths, and headings improve retrieval within the prompt.
4. **Budget accuracy** — Does the output stay within token limits without silent truncation?

**Acceptance Criteria:**
- **Coverage checks:** `--assert-include` patterns fail the run if critical files are omitted
- **Faithfulness verification:** `harvx verify [--sample <n>]` command compares packed output to source for N sampled files (or specific paths), reporting any differences beyond expected compression/redaction
- **Golden questions harness:** Ship documentation and a template for maintaining a set of repo-specific questions with known answers (e.g., "Where is JWT validated?", "What's the retry default?"). Compare LLM accuracy with diff-only context vs diff + Harvx context. Store results in CI artifacts for tracking over time
- **Budget reporting:** Output metadata always includes tokenizer used, total tokens, budget utilization percentage, and whether any files were truncated or omitted

---

### 5.12 Directory Tree Visualization

**Priority:** Must-Have
**Description:** Generate a visual directory tree of included files using Unicode box-drawing characters, with file metadata (size, token count) and tier indicators.

**User Story:** As a developer, I want the context output to include a clear visual representation of my project structure so that the LLM understands the codebase organization at a glance.

**Acceptance Criteria:**
- Generates a tree using Unicode box-drawing characters (├──, └──, │)
- Uses folder and file emoji indicators (📁, 📄) for visual clarity
- Shows only included files (respects all filters and ignores)
- Optionally shows file sizes and token counts next to each file
- Respects depth limits (`--tree-depth <n>`, default: unlimited)
- Can be generated independently via `harvx preview`

**Technical Considerations:**
- Build tree in memory as a nested struct, then render to string
- Sort directories before files, both alphabetically
- Collapse empty intermediate directories (e.g., `src/utils/` if it only contains one subdirectory)

---

### 5.13 Interactive TUI Mode

**Priority:** Must-Have
**Description:** A terminal user interface that lets users visually browse their project tree and toggle file/directory inclusion before generating output. Harvx operates in two modes: **headless** (default, optimized for pipelines and scripts) and **interactive** (TUI, optimized for exploration and one-off use). Both modes share the same core pipeline — the TUI is a presentation layer only. A polished, beautiful UI/UX is a core brand differentiator for Harvx.

**User Story:** As a developer who wants quick, one-off context generation, I want to visually select which files to include so that I don't have to remember glob patterns or write a config file.

**Acceptance Criteria:**
- Launched via `harvx --interactive` or `harvx -i`
- Also launches when running `harvx` with no arguments and no `harvx.toml` detected (smart default: first-timers get the TUI)
- **File tree panel:**
  - Navigable file tree (arrow keys to move, space to toggle file/directory inclusion)
  - Visual indicators for included (✓), excluded (✗), and partially-included directories (◐)
  - Color-coded tier assignments (tier 0 = gold, tier 1 = green, tier 2 = blue, etc.)
  - Highlights priority files and secret-containing files with distinct icons
  - Supports expand/collapse directories
  - Keyboard shortcuts: `/` to search/filter files, `a` to select all, `n` to select none, `t` to cycle tier views
- **Stats panel (live-updating sidebar):**
  - Real-time token count (updates as files are toggled): `Tokens: 89,420 / 200,000`
  - Budget utilization bar (visual progress)
  - File count: `Files: 342 / 390`
  - Estimated output size
  - Compression savings (if enabled): `Compressed: 52% reduction`
  - Redaction count: `Secrets: 3 found`
- **Profile selector:**
  - Quick-switch between profiles (Tab key)
  - Shows resolved profile name and key settings
- **Actions:**
  - Enter: Generate output with current selection
  - `p`: Preview output (summary only)
  - `s`: Save current selection as a new profile
  - `e`: Export selection to clipboard
  - `q` / Esc: Quit without generating
  - `?`: Help overlay with all keybindings
- **Visual design:**
  - Clean, modern terminal aesthetic using lipgloss styled components
  - Responsive layout that adapts to terminal size
  - Smooth animations for toggling and panel transitions
  - High-contrast color scheme that works on both light and dark terminals

**Technical Considerations:**
- Use `charmbracelet/bubbletea` for the TUI framework (Elm architecture for Go)
- Use `charmbracelet/lipgloss` for declarative, CSS-like terminal styling
- Use `charmbracelet/bubbles` for reusable components (file tree, text input, viewport, progress)
- The TUI is a presentation layer only — it calls the same core pipeline library as the CLI
- Token counting in the TUI should be debounced (recalculate 200ms after last toggle) to keep the UI responsive
- File tree should lazy-load directory contents for large repos (don't scan everything upfront)
- TUI state can be serialized to a profile TOML for reuse in headless mode

---

## 6. Technical Architecture

### 6.1 Recommended Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| Language | Go 1.22+ | Single binary, goroutine parallelism, fast compilation, mature CLI ecosystem |
| CLI Framework | spf13/cobra | Industry standard for Go CLIs (kubectl, gh, docker) |
| Configuration | spf13/viper + BurntSushi/toml | Multi-source config resolution with TOML parsing. Evaluate koanf/koanf as lighter alternative |
| Token Counting | pkoukk/tiktoken-go | Go port of OpenAI's tiktoken, cl100k_base + o200k_base |
| Tree-Sitter | tetratelabs/wazero + embedded WASM grammars | Pure Go WASM runtime, zero CGO, cross-platform binary |
| Hashing | cespare/xxhash | XXH3 implementation for fast content hashing |
| Diff | sergi/go-diff | Unified diff generation |
| Terminal Styling | charmbracelet/lipgloss | Declarative terminal styling, auto-detection |
| TUI Framework | charmbracelet/bubbletea | Interactive terminal UI (v1.1) |
| Progress Bars | schollz/progressbar | Animated progress for large repos |
| Gitignore | sabhiram/go-gitignore | .gitignore pattern parsing |
| Glob Matching | bmatcuk/doublestar | Doublestar glob pattern matching |
| Logging | log/slog (stdlib) | Structured logging with levels (Go 1.21+ stdlib) |
| Concurrency | x/sync/errgroup | Bounded parallel execution with error propagation |
| Testing | stretchr/testify | Assertions and test suites |
| Releases | goreleaser + cosign + syft | Cross-platform builds, signing, SBOM |

### 6.2 Project Structure

```
harvx/
├── cmd/
│   └── harvx/
│       └── main.go                  # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go                  # Root command
│   │   ├── generate.go              # Generate subcommand
│   │   ├── brief.go                 # Brief subcommand
│   │   ├── review_slice.go          # Review slice subcommand
│   │   ├── slice.go                 # Module slice subcommand
│   │   ├── preview.go               # Preview subcommand
│   │   ├── diff.go                  # Diff subcommand
│   │   ├── profiles.go              # Profile management subcommands
│   │   ├── workspace.go             # Workspace subcommand
│   │   ├── config_debug.go          # Config debug subcommand
│   │   ├── doctor.go                # Doctor subcommand
│   │   ├── verify.go                # Verify subcommand
│   │   └── cache.go                 # Cache management subcommands
│   ├── config/
│   │   ├── config.go                # Configuration types and loading
│   │   ├── profile.go               # Profile resolution and inheritance
│   │   ├── workspace.go             # Workspace manifest parsing
│   │   ├── defaults.go              # Built-in default values
│   │   ├── templates.go             # Framework-specific profile templates
│   │   └── validate.go              # Configuration validation + lint
│   ├── discovery/
│   │   ├── walker.go                # Parallel file discovery (errgroup)
│   │   ├── gitignore.go             # .gitignore parsing and matching
│   │   ├── filter.go                # Extension and pattern filtering
│   │   ├── git_tracked.go           # --git-tracked-only implementation
│   │   └── binary.go                # Binary file detection
│   ├── relevance/
│   │   ├── sorter.go                # Priority-based file sorting
│   │   ├── tiers.go                 # Tier definitions and matching
│   │   └── explain.go               # Explain which rule applies to a file
│   ├── tokenizer/
│   │   ├── tokenizer.go             # Tokenizer interface
│   │   ├── tiktoken.go              # tiktoken implementation (cl100k, o200k)
│   │   ├── estimator.go             # Byte-based fallback estimator
│   │   └── budget.go                # Token budget enforcement
│   ├── security/
│   │   ├── redactor.go              # Secret detection and redaction
│   │   ├── patterns.go              # Gitleaks-inspired patterns (compiled regexes)
│   │   ├── entropy.go               # Entropy analysis for high-entropy strings
│   │   └── report.go                # Redaction report generation
│   ├── compression/
│   │   ├── compressor.go            # Compression orchestrator + LanguageCompressor interface
│   │   ├── wasm.go                  # Wazero WASM runtime management
│   │   ├── languages.go             # Language-specific extraction rules
│   │   └── fallback.go              # Fallback for unsupported languages
│   ├── output/
│   │   ├── renderer.go              # Output format dispatcher
│   │   ├── markdown.go              # Markdown renderer
│   │   ├── xml.go                   # XML renderer (Anthropic conventions)
│   │   ├── splitter.go              # Output chunking/splitting
│   │   ├── metadata.go              # JSON metadata sidecar generation
│   │   └── tree.go                  # Directory tree visualization
│   ├── diff/
│   │   ├── state.go                 # Project state snapshots
│   │   ├── cache.go                 # State caching (read/write)
│   │   ├── git.go                   # Git-aware diffing (--since, --base/--head)
│   │   └── diff.go                  # Diff generation between states
│   ├── workflows/
│   │   ├── brief.go                 # Repo Brief generation logic
│   │   ├── review_slice.go          # Review Slice generation logic
│   │   └── workspace.go             # Workspace manifest rendering
│   ├── tui/                         # Interactive TUI mode (Bubble Tea)
│   │   ├── app.go                   # Bubble Tea application
│   │   ├── file_tree.go             # Interactive file tree component
│   │   └── styles.go                # Lipgloss styles
│   ├── server/                      # (v1.1) MCP server
│   │   └── mcp.go                   # MCP server implementation
│   └── pipeline/
│       ├── pipeline.go              # Main processing pipeline orchestration
│       └── types.go                 # FileDescriptor and shared DTOs
├── grammars/                        # Embedded WASM grammar files
│   ├── tree-sitter-typescript.wasm
│   ├── tree-sitter-javascript.wasm
│   ├── tree-sitter-go.wasm
│   ├── tree-sitter-python.wasm
│   └── tree-sitter-rust.wasm
├── templates/                       # Profile templates
│   ├── nextjs.toml
│   ├── go-cli.toml
│   ├── python-django.toml
│   ├── rust-cargo.toml
│   └── monorepo.toml
├── testdata/                        # Test fixtures
│   ├── sample-repo/
│   ├── secrets/                     # Regression corpus for redaction
│   ├── monorepo/
│   └── expected-output/             # Golden test outputs
├── .goreleaser.yml                  # Cross-platform release builds + signing + SBOM
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── LICENSE
```

### 6.3 Processing Pipeline

```
CLI Input (flags, config, subcommand)
       │
       ▼
┌──────────────┐
│ Config        │ ← Merge: CLI flags + env vars + harvx.toml + ~/.config/harvx/config.toml + defaults
│ Resolution    │ ← Profile selection and inheritance resolution
│               │ ← LLM target preset application (claude/chatgpt/generic)
│               │ ← Validation + lint warnings
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ File          │ ← Parallel directory walk (errgroup with SetLimit)
│ Discovery     │ ← .gitignore + .harvxignore + default ignores + custom patterns
│               │ ← Binary file detection, --git-tracked-only mode
│               │ ← Extension/pattern filtering, --skip-large-files
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Relevance     │ ← Match files to tiers (0-5) using glob patterns
│ Sorting       │ ← Sort by tier, then alphabetically within each tier
│               │ ← --assert-include coverage checks
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Content       │ ← Parallel file reading (errgroup, bounded concurrency)
│ Loading       │ ← Apply secret redaction (streaming filter, Gitleaks-inspired patterns)
│               │ ← Apply tree-sitter WASM compression (if enabled, lazy — only for budget survivors)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Token         │ ← Count tokens per file (configurable tokenizer: cl100k/o200k/none)
│ Budgeting     │ ← Enforce max_tokens: omit lower-tier files first
│               │ ← Generate heatmap data (tokens per line)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ State         │ ← Compare with cached state or git refs (if diff mode)
│ Comparison    │ ← Generate diff summary
│               │ ← Save current state to cache (with branch metadata)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Output        │ ← Render header + summary + tree + files + diff
│ Rendering     │ ← Write to file / stdout / split into chunks
│               │ ← Compute content hash (XXH3) for deterministic fingerprinting
│               │ ← Generate metadata JSON sidecar (if --output-metadata)
│               │ ← Log per-stage timings (if --verbose)
└──────────────┘
```

### 6.4 Concurrency Model

Harvx uses `x/sync/errgroup.WithContext()` with `SetLimit(runtime.NumCPU())` for bounded parallelism with proper error propagation and cancellation via `context.Context`.

```
                    ┌─── Worker 1 (walk subtree) ──┐
                    │                                │
Directory Root ────►├─── Worker 2 (walk subtree) ──►├──► Collector (sorted by path)
                    │                                │
                    └─── Worker N (walk subtree) ──┘
                    (errgroup with SetLimit)

                    ┌─── Worker 1 (read+redact+compress) ──┐
                    │                                        │
Sorted Files ──────►├─── Worker 2 (read+redact+compress) ──►├──► Renderer (tier order)
                    │                                        │
                    └─── Worker N (read+redact+compress) ──┘
                    (errgroup with SetLimit)
```

`context.Context` is threaded through all long-running operations so CLIs are cancellable in both CI and interactive use (Ctrl+C).

### 6.5 Central Data Types

Define a `FileDescriptor` struct as the central DTO passed between pipeline stages:

```go
type FileDescriptor struct {
    Path         string   // Relative path from root
    AbsPath      string   // Absolute filesystem path
    Size         int64    // File size in bytes
    Tier         int      // Relevance tier (0-5)
    TokenCount   int      // Token count (after compression/redaction)
    ContentHash  uint64   // XXH3 hash of processed content
    Content      string   // Processed content (redacted, optionally compressed)
    IsCompressed bool     // Whether tree-sitter compression was applied
    Redactions   int      // Number of secrets redacted in this file
    Language     string   // Detected language (for compression)
}
```

### 6.6 Logging & Diagnostics

- Use Go's stdlib `log/slog` (available since Go 1.21) for structured logging
- Log level mapping:
  - Default: `info` (summary output only)
  - `--verbose`: `debug` (per-stage timings, per-file decisions, config resolution)
  - `--quiet`: `error` (only fatal issues)
- `HARVX_LOG_FORMAT=json` env var enables JSON-structured logs for CI pipeline parsing
- `HARVX_DEBUG=1` dumps: effective resolved config, top N slowest files, per-stage duration breakdown
- Only `internal/pipeline` talks to multiple layers; each lower layer exposes small, composable interfaces
- Viper/config does not leak past `internal/config` — lower layers receive a concrete config struct

### 6.7 Internal API Boundaries

- Only `internal/pipeline` orchestrates multiple layers
- Each layer exposes composable interfaces: `DiscoveryService`, `TokenBudgeter`, `Redactor`, `Compressor`, `Renderer`
- The core pipeline is also usable as a **Go library** (not just CLI glue), enabling future MCP server and programmatic usage to share the same engine
- Redaction patterns are loadable from TOML sections (pluggable per category)
- `LanguageCompressor` interface for compression; core languages built-in, additional languages via external WASM files in future
- Output rendering uses an internal template abstraction so adding new formats is straightforward

---

## 7. Security & Compliance

### 7.1 Secret Redaction (Primary Security Feature)

Secret redaction is Harvx's primary security mechanism and is enabled by default. See Feature 5.5 for full specification. Patterns are modeled after the Gitleaks ruleset for proven, real-world coverage.

### 7.2 Data Handling Principles

- **All processing is local.** Harvx never sends data to external services. All file reading, parsing, and output generation happens on the user's machine.
- **No telemetry.** Harvx collects no usage data, analytics, or crash reports.
- **No network access.** The binary makes zero network requests. (Remote repo support may be added in a future version but will always be opt-in.)
- **Output files are the user's responsibility.** Harvx generates files locally; the user decides where to send them.

### 7.3 Sensitive File Handling

- `.env` files are excluded by default (in the built-in ignore list)
- `.pem`, `.key`, `.p12`, `.pfx` files are excluded by default
- Files matching `*secret*`, `*credential*`, `*password*` patterns are excluded by default
- These defaults can be overridden in profile configuration, but a warning is emitted

### 7.4 Supply Chain Security

- All releases include SHA-256 checksums
- Binaries are signed using Cosign (keyless signing via Sigstore)
- SBOM (Software Bill of Materials) generated via Syft and included in every release
- These are generated automatically by GoReleaser

---

## 8. UI/UX Principles

### 8.1 CLI Design Philosophy

- **Zero-config first run.** `harvx` in any directory should produce useful output with sensible defaults.
- **Progressive disclosure.** Simple commands for simple needs, advanced flags for power users.
- **Predictable behavior.** Same input always produces same output (deterministic content hash).
- **Helpful errors.** Every error message should include what went wrong, why, and how to fix it. Example: "Failed to parse TypeScript: tree-sitter grammar not found. Run `harvx doctor` for diagnostics."
- **Minimal output by default.** Progress information goes to stderr, results to stdout/file. No unnecessary banners or decorations.
- **Fail fast.** Invalid config exits immediately with a clear message. Partial success (5/1000 files failed) returns exit code 2 with a summary.

### 8.2 Output Formatting

- Declarative terminal styling via `charmbracelet/lipgloss` (auto-disabled when piped)
- Animated progress bars via `schollz/progressbar` for large repos:
  ```
  Processing files... ━━━━━━━━━━━━━━╸━━━━━━━ 67% (2340/3500) 1.2s
  Compressing...      ━━━━━━━━━╸━━━━━━━━━━━━ 45% (890/1980)
  ```
- Summary table after generation:
  ```
  ✓ Generated finvault-context.md

    Profile:     finvault
    Target:      claude (XML)
    Tokenizer:   o200k_base
    Files:       342 included (48 excluded)
    Tokens:      89,420 / 200,000 budget (45%)
    Compressed:  yes (52% reduction)
    Redactions:  3 (2 API keys, 1 connection string)
    Time:        0.8s

    Top 5 files by tokens:
    1. prisma/schema.prisma          4,210 tokens
    2. lib/services/transaction.ts   3,890 tokens
    3. app/api/transactions/route.ts 2,340 tokens
    4. CLAUDE.md                     2,100 tokens
    5. middleware.ts                  1,850 tokens
  ```

---

## 9. Testing Strategy

### 9.1 Unit Tests

- **discovery:** ignore patterns, binary detection, symlink handling, `--git-tracked-only`
- **relevance:** tier matching, tie-breaking, unmatched file default behavior
- **tokenizer:** budget enforcement, truncation strategies, cl100k vs o200k consistency
- **redaction:** per-pattern test cases including edge cases (fake tokens in test fixtures, keys in docs, high-entropy non-secrets)
- **compression:** per-language extraction correctness (compare extracted signatures to expected output)
- **config:** profile inheritance, merge behavior, validation, lint warnings

### 9.2 Golden Tests

Small representative repos under `testdata/`:
- `sample-repo/` with TypeScript/Go/Python, docs, CI, config files
- `secrets/` with safe mock secrets (realistic AWS/GCP/Stripe formats) — the regression corpus
- `monorepo/` with many packages and build artifacts

Golden outputs for:
- Default profile, finvault profile, work profile
- Compressed vs non-compressed runs
- Markdown vs XML format
- `--git-tracked-only` vs standard discovery
- Diff-only mode with known state changes

### 9.3 Performance Benchmarks

- Generate context for 1K, 10K, 50K file repos
- Run discovery + tokenization + compression with different profiles
- Track P50/P95 times and compare to SLOs (< 1s for 1K files, < 3s for 10K files)
- Memory usage tracking for streaming output verification

### 9.4 Fuzz & Property-Based Tests

- Fuzz redaction on random high-entropy strings and synthetic `.env` content
- Property: redaction + tokenization must not produce invalid UTF-8
- Fuzz config parsing with malformed TOML

### 9.5 Integration Tests

- Test against real open-source repos (cloned in CI):
  - Next.js (TypeScript, large codebase)
  - Kubernetes (Go, massive scale)
  - Django (Python)
- Shell script invocation, pipe chains, exit code handling, env var overrides
- End-to-end: config → discovery → redaction → compression → output → verify

### 9.6 Performance Diagnostics

- `--verbose` mode includes per-stage timing breakdown
- `HARVX_DEBUG=1` outputs detailed profiling information

---

## 10. Development Roadmap

### Phase 1: Foundation (Weeks 1–3)

**Goal:** Working CLI that can walk a directory, discover files, and produce basic Markdown output.

- Project scaffolding (Go modules, Cobra CLI, Makefile, CI, structured logging with slog)
- File discovery with .gitignore support and `--git-tracked-only`
- Default ignore patterns and binary detection
- Basic Markdown output renderer
- Directory tree visualization
- Extension and pattern filtering
- `--preview` mode with token heatmap
- Errgroup-based parallel processing
- Basic benchmarks on sample repos
- Unit tests + golden tests for discovery and filtering

**Deliverable:** `harvx` produces correct Markdown output for any repository with default settings.

### Phase 2: Intelligence (Weeks 4–6)

**Goal:** Add profile system, relevance sorting, and token budgeting.

- TOML configuration loading and validation
- Profile system with inheritance and framework templates
- Profile management subcommands (list, init, show, lint, explain)
- Config debug command
- Relevance-based file sorting (6 tiers)
- Token counting with configurable tokenizer (cl100k_base + o200k_base)
- Token budgeting with truncation strategies
- LLM target presets (claude, chatgpt, generic)
- `--assert-include` coverage checks
- Integration tests for profile resolution

**Deliverable:** `harvx --profile finvault --target claude` produces architecture-aware, token-budgeted output.

### Phase 3: Security & Compression (Weeks 7–9)

**Goal:** Add secret redaction and tree-sitter code compression via WASM.

- Gitleaks-inspired secret detection patterns (compiled regexes)
- Redaction pipeline (streaming filter) with path exclusions and confidence thresholds
- Redaction reporting and `--fail-on-redaction`
- Entropy analysis module
- Wazero WASM runtime integration
- Embed tree-sitter grammar WASM files (TS, JS, Go, Python, Rust)
- LanguageCompressor interface + language-specific extraction rules
- Compression with `--compress-timeout`
- `harvx verify` command for faithfulness checking
- Regression test corpus for secrets

**Deliverable:** `harvx --compress --profile finvault` produces compressed, redacted output with zero known secret leaks.

### Phase 4: Workflows & Diff (Weeks 10–11)

**Goal:** Add workflow commands, state caching, and differential output.

- `harvx brief` command (Repo Brief generation)
- `harvx review-slice` command (PR-specific context)
- `harvx slice` command (targeted module context)
- `harvx workspace` command (multi-repo manifest)
- State snapshot serialization (JSON with git metadata)
- Git-aware diffing (`--since`, `--base/--head`)
- `harvx diff` subcommand
- Cache management subcommands
- Output splitting (`--split`)
- Metadata JSON sidecar (`--output-metadata`)
- XML output format (for claude target)
- Doctor command

**Deliverable:** Full workflow support — `harvx brief && harvx review-slice --base main --head HEAD` enriches review pipelines.

### Phase 5: Interactive TUI (Weeks 12–13)

**Goal:** Build a polished interactive terminal interface that makes Harvx approachable for exploration and one-off use.

- Bubble Tea application scaffold with Elm architecture
- File tree component (navigable, expandable, toggleable)
- Live token count sidebar with budget utilization bar
- Profile selector (Tab to switch)
- Lipgloss styling: color-coded tiers, responsive layout, light/dark terminal support
- Keyboard shortcuts (search, select all/none, tier views, help overlay)
- Save-to-profile flow (serialize TUI selections to TOML)
- Smart default: `harvx` with no args and no `harvx.toml` launches TUI
- Performance: debounced token recalculation, lazy directory loading for large repos
- Unit tests for TUI state management

**Deliverable:** `harvx -i` launches a beautiful, responsive TUI for visual file selection with real-time token counting.

### Phase 6: Polish & Distribution (Week 14)

**Goal:** Release-ready binary with cross-platform builds, documentation, and completions.

- GoReleaser configuration: macOS (amd64/arm64), Linux (amd64/arm64), Windows (amd64)
- Cosign signing + Syft SBOM generation
- Shell completion generation (bash, zsh, fish, PowerShell)
- Comprehensive README with usage examples and canonical recipes per persona
- Man page generation
- Performance benchmarking and optimization pass
- End-to-end integration test suite against real OSS repos
- Claude Code hook setup documentation + lean CLAUDE.md template
- GitHub Release automation

**Deliverable:** Published v1.0.0 with signed binaries for all platforms.

### v1.1 (Post-Release)

| Feature | Description | Priority |
|---------|-------------|----------|
| MCP Server | `harvx mcp serve` for Claude Code / Codex CLI integration | High |
| Remote Repo Support | `harvx --remote github.com/user/repo` | High |
| Plain text + JSON output | Additional output formats | Medium |
| Clipboard support | `--clipboard` flag | Medium |
| Compression levels | `signatures` vs `structure` vs `full` | Medium |
| Homebrew/Scoop Formula | `brew install harvx` | Medium |
| Watch Mode | `harvx --watch` for continuous regeneration | Medium |
| Tier 2 compression languages | Java, C, C++ tree-sitter grammars | Medium |
| Custom Output Templates | User-defined Go templates | Low |
| Semantic Chunking | AI-aware chunking across token boundaries | Low |
| VS Code Extension | GUI for profile management | Low |

---

## 11. Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| WASM tree-sitter grammars have parsing gaps vs native | Medium | Medium | Start with well-tested languages (TS, Go, Python). Maintain fallback to regex-based extraction. Benchmark against native tree-sitter output. |
| Token counting accuracy varies across LLM providers | Low | Medium | Support both cl100k_base and o200k_base. Design tokenizer as pluggable interface. Document that counts are approximate. |
| False positives in secret redaction annoy users | Medium | Low | Gitleaks-inspired patterns, path exclusions, confidence thresholds, `--no-redact` escape hatch, redaction report for verification. |
| WASM grammar files increase binary size beyond target | Medium | Low | Compress WASM files before embedding. Evaluate lazy-loading grammars from a companion archive if binary exceeds 25MB. |
| Large monorepos (50K+ files) cause performance issues | Low | Medium | Bounded concurrency, streaming output, file size limits, `--skip-large-files`. Profile-based filtering reduces effective file count. `--git-tracked-only` avoids traversing untracked build artifacts. |
| Profile configuration complexity deters casual users | Low | Low | Zero-config default works great. Framework templates. `harvx profiles init`, `lint`, `explain` commands. Canonical recipes in docs. |
| Config mis-tuning leads to surprising omissions | Medium | Medium | `harvx profiles explain <path>`, `harvx config debug`, `--assert-include` coverage checks. |
| Competing tools (Repomix) add profile features | Medium | Low | Harvx's Go binary + pipeline integration + workflow commands are structural advantages. Focus on the automated review and agent bootstrap use cases. |
| v1 scope creep delays initial release | Medium | High | Moved MCP server, clipboard, compression levels, and additional formats to v1.1. Core v1 is tightly scoped to 14 weeks with dedicated TUI phase. |
| Git-aware diffing edge cases (rebases, force pushes, merge commits) | Medium | Low | Document supported git scenarios. Default to filesystem-based state comparison when git metadata is ambiguous. |

---

## 12. Open Questions (with Proposed Directions)

| # | Question | Proposed Direction |
|---|---------|-------------------|
| 1 | **Tokenizer default:** Should Harvx default to cl100k_base or o200k_base? | Start with cl100k_base as default (widest compatibility). Design tokenizer as pluggable abstraction. Add `tokenizer_model = "gpt-4"` auto-selection in future. |
| 2 | **Output size limits:** Should there be a max output file size independent of token budget? | Yes. Soft warning at 20–30MB, hard cap configurable via profile (`max_output_size`). |
| 3 | **Profile sharing:** Is `--profile-file` sufficient for team distribution? | Yes — both `--profile-file <path>` for standalone sharing AND `harvx.toml` at repo root for committed configs. |
| 4 | **Incremental compression:** When using `harvx diff` with `--compress`, show diff of full content or compressed? | Show diff of full content (compressed diffs are confusing and lose meaning). |
| 5 | **CI/CD integration:** Should Harvx provide a GitHub Action? | Yes — official Action increases adoption significantly. Add in v1.1 or shortly after v1.0. |
| 6 | **License:** MIT vs Apache 2.0? | MIT for maximum adoption. |
| 7 | **Review Slice neighborhood depth:** How many levels of imports/dependencies should the review slice include? | Start with 1 level (direct imports only). Make configurable via `slice_depth` in profile. |
| 8 | **Workspace cross-repo slicing:** Should `harvx workspace` allow slicing into other repos? | v1: render manifest only. v1.1: allow `harvx slice --repo <name> --path <module>` for cross-repo context. |

---

## 13. Appendix

### A. Competitive Landscape Reference

| Tool | Stars | Language | Key Differentiator | Harvx Advantage |
|------|-------|----------|--------------------|-----------------|
| Repomix | 20K+ | TypeScript | Most polished, MCP server, Claude Code plugins | Profile system, Go binary, workflow commands, pipeline integration |
| code2prompt | 6.9K | Rust | Handlebars prompt templating | Profiles, target presets, review workflows |
| gpt-repository-loader | 3K | Python | First mover (Mar 2023) | Everything — gpt-repo-loader is minimal |
| files-to-prompt | 2.5K | Python | Unix philosophy, composable | Profiles, compression, redaction, budgeting |
| yek | 2.4K | Rust | Pure speed, smart chunking | Profiles, workflows, agent bootstrap |
| CTX (context-hub) | ~500 | PHP/Go | YAML config, closest to profile concept | Go binary, better DX, workflow commands |
| context-builder (igorls) | Newer | Rust | Tree-sitter AST, relevance tiers, state caching | Go binary, profiles, workflows, WASM portability |
| CodeContexter | Newer | Rust | Security-first, secret redaction | Profiles, pipeline integration, broader feature set |

### B. Reference Links

- [Wazero — Zero-dependency WASM runtime for Go](https://github.com/tetratelabs/wazero)
- [Cobra CLI framework](https://github.com/spf13/cobra)
- [Viper configuration](https://github.com/spf13/viper)
- [tiktoken-go](https://github.com/pkoukk/tiktoken-go)
- [Charmbracelet ecosystem](https://charm.sh/)
- [Gitleaks — Secret detection rules](https://github.com/gitleaks/gitleaks)
- [GoReleaser](https://goreleaser.com/) — Cross-platform binary releases
- [Cosign — Container/artifact signing](https://github.com/sigstore/cosign)
- [Syft — SBOM generation](https://github.com/anchore/syft)
- [Repomix output format reference](https://repomix.com/)
- [Anthropic XML tag best practices](https://docs.anthropic.com/en/docs/build-with-claude/prompt-engineering/use-xml-tags)
- [Claude Code hooks guide](https://docs.anthropic.com/en/docs/claude-code/hooks)
- [OpenAI Codex CLI MCP](https://platform.openai.com/docs/guides/tools-mcp)
- [context-builder crate reference](https://crates.io/crates/context-builder)

### C. Glossary

| Term | Definition |
|------|-----------|
| **Context window** | The maximum number of tokens an LLM can process in a single interaction |
| **Token** | The basic unit of text that LLMs process; roughly 4 characters or ¾ of a word |
| **Tree-sitter** | A parser generator tool that builds fast, incremental parsers for source code |
| **WASM** | WebAssembly — a portable binary format that can be executed in sandboxed runtimes like wazero |
| **Wazero** | A zero-dependency WebAssembly runtime for Go, enabling tree-sitter without CGO |
| **AST** | Abstract Syntax Tree — a structured representation of source code |
| **cl100k_base** | OpenAI's tokenizer encoding used by GPT-4, compatible with Claude's token counting |
| **o200k_base** | OpenAI's newer tokenizer encoding used by GPT-4o, o1, and newer models |
| **XXH3** | An extremely fast non-cryptographic hash function |
| **MCP** | Model Context Protocol — Anthropic's standard for AI tool integration |
| **Profile** | A named configuration preset that defines how Harvx processes a specific project |
| **Relevance tier** | A priority level (0–5) assigned to files that determines output order and budget priority |
| **Redaction** | Detecting and replacing sensitive information with placeholder text |
| **Repo Brief** | A stable, small context artifact containing project-wide invariants and architecture |
| **Review Slice** | A dynamic, PR-specific context artifact with changed files and bounded neighborhood |
| **Workspace Manifest** | A config file describing multiple related repos, their paths, and integration edges |
| **Gitleaks** | A popular open-source secrets scanner whose pattern ruleset Harvx's redaction is modeled after |
| **errgroup** | Go's `x/sync/errgroup` package for bounded parallel execution with error propagation |
| **GoReleaser** | A tool for automating cross-platform Go binary releases |
| **Cosign** | A tool for signing and verifying software artifacts (part of Sigstore) |
| **SBOM** | Software Bill of Materials — a manifest of all dependencies in a software artifact |
