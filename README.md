# Harvx

> Harvest your context.

[![CI](https://github.com/harvx/harvx/actions/workflows/ci.yml/badge.svg)](https://github.com/harvx/harvx/actions/workflows/ci.yml)
[![Release](https://github.com/harvx/harvx/actions/workflows/release.yml/badge.svg)](https://github.com/harvx/harvx/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/harvx/harvx)](https://github.com/harvx/harvx/releases/latest)
![Go 1.24+](https://img.shields.io/badge/go-1.24%2B-blue)
![License: MIT](https://img.shields.io/badge/license-MIT-green)

Harvx is a single-binary CLI tool that packages source code repositories into context documents optimized for large language models. It handles file discovery, relevance tiering, token budgeting, secret redaction, and code compression -- producing clean, structured output ready for Claude, ChatGPT, and other LLMs.

**Single binary (~15-20MB). Zero runtime dependencies. macOS, Linux, Windows.**

## Installation

### Go Install

```bash
go install github.com/harvx/harvx/cmd/harvx@latest
```

### Binary Download

Download the latest release for your platform from [GitHub Releases](https://github.com/harvx/harvx/releases/latest).

```bash
# macOS (Apple Silicon)
curl -Lo harvx.tar.gz https://github.com/harvx/harvx/releases/latest/download/harvx_darwin_arm64.tar.gz
tar xzf harvx.tar.gz
sudo mv harvx /usr/local/bin/

# Linux (amd64)
curl -Lo harvx.tar.gz https://github.com/harvx/harvx/releases/latest/download/harvx_linux_amd64.tar.gz
tar xzf harvx.tar.gz
sudo mv harvx /usr/local/bin/
```

### Homebrew (planned)

```bash
brew install harvx/tap/harvx
```

### Verify Installation

```bash
harvx version
```

## Quickstart

Run `harvx` in any repository with zero configuration:

```bash
# Generate context (outputs to harvx-output.md)
harvx

# Launch interactive TUI for file selection
harvx -i

# Output to stdout for piping
harvx --stdout | pbcopy

# Generate compressed context (50-70% token reduction)
harvx --compress

# Generate XML format for Claude
harvx --format xml --target claude
```

That's it. Harvx uses sensible defaults: it respects `.gitignore`, redacts secrets, sorts files by relevance, and produces Markdown output optimized for LLM context windows.

## Key Features

| Feature | Description |
|---------|-------------|
| **File Discovery** | Parallel walker respecting `.gitignore`, `.harvxignore`, binary detection, symlink handling |
| **Relevance Tiering** | 6-tier system (T0-T5) sorting files by importance using glob patterns |
| **Token Budgeting** | cl100k/o200k tokenizers with budget enforcement (skip or truncate strategies) |
| **Secret Redaction** | 19 built-in detection patterns, Shannon entropy analysis, PEM block handling |
| **Code Compression** | AST-aware compression for 12 languages via state-machine parsers, regex fallback |
| **Profile System** | TOML-based project profiles with inheritance, 6 framework templates |
| **Interactive TUI** | Bubble Tea file selector with real-time token counting and vim keybindings |
| **State & Diff** | Cache-based diffing, git-aware diffs, differential context generation |
| **Workflows** | `brief`, `slice`, `review-slice`, `workspace`, `verify`, `quality` commands |
| **MCP Server** | Stdio-based MCP server for coding agent integration |

## Configuration

### Zero Config

Harvx works out of the box with sensible defaults. Just run `harvx` in any directory.

### harvx.toml

Create a `harvx.toml` in your project root for project-specific settings:

```toml
[profile.default]
format = "markdown"
target = "claude"
max_tokens = 128000
compression = true
redaction = true

ignore = [
  "**/*.test.*",
  "fixtures/**",
]

priority_files = [
  "README.md",
  "ARCHITECTURE.md",
]

[profile.default.relevance]
tier_0 = ["README.md", "CLAUDE.md", "ARCHITECTURE.md"]
tier_1 = ["**/*.go", "**/*.ts", "**/*.py"]
tier_2 = ["**/*.json", "**/*.yaml", "**/*.toml"]
tier_5 = ["**/*.md", "**/LICENSE"]
```

### Profile Templates

Initialize a project config from a framework template:

```bash
harvx profiles init --template go-cli
harvx profiles init --template nextjs
harvx profiles init --template python-django
harvx profiles init --template rust-cargo
harvx profiles init --template monorepo
```

### Profile Inheritance

Profiles can extend other profiles:

```toml
[profile.base]
format = "markdown"
max_tokens = 128000

[profile.ci]
extends = "base"
redaction = true

[profile.review]
extends = "base"
compression = true
max_tokens = 50000
```

## Persona Recipes

### Alex -- Daily AI Chat User

Quick context generation for pasting into Claude, ChatGPT, or any chat interface:

```bash
# Zero-config: launches TUI if no harvx.toml exists
harvx

# Interactive file selection with real-time token counting
harvx -i

# Generate compressed context and copy to clipboard
harvx --compress --stdout | pbcopy

# Quick XML output for Claude
harvx --format xml --target claude --stdout
```

### Zizo -- Pipeline Integrator

Multi-step workflows for code review pipelines and session bootstrap:

```bash
# Generate a brief project summary (1-4K tokens)
harvx brief --profile finvault

# Review slice: context for a PR review
harvx review-slice --base main --head HEAD

# Module slice: targeted context for a specific module
harvx slice --path internal/auth

# Workspace context across multiple repos
harvx workspace

# Session bootstrap for Claude Code
harvx brief --stdout >> .claude/context.md
```

### Jordan -- CI Integrator

Automated context generation in GitHub Actions and CI/CD pipelines:

```bash
# CI mode: strict redaction, metadata sidecar, quiet output
harvx --profile ci --fail-on-redaction --output-metadata --quiet

# Verify output faithfulness
harvx verify output.md

# Quality evaluation with golden questions
harvx quality --json

# Doctor check in CI
harvx doctor --json
```

## Command Reference

### Core Commands

| Command | Description |
|---------|-------------|
| `harvx` / `harvx generate` | Generate context document (default command) |
| `harvx -i` / `harvx --interactive` | Launch interactive TUI file selector |
| `harvx preview` | Show file tree, token estimates, tier breakdown |
| `harvx preview --heatmap` | Token heatmap visualization |

### Workflow Commands

| Command | Description |
|---------|-------------|
| `harvx brief` | Generate 1-4K token project summary |
| `harvx slice --path <path>` | Module-targeted context slice |
| `harvx review-slice --base <ref> --head <ref>` | PR review context with import neighbors |
| `harvx workspace` | Multi-repo workspace context |
| `harvx verify <file>` | Verify output faithfulness |
| `harvx quality` | Golden questions evaluation harness |

### Profile Management

| Command | Description |
|---------|-------------|
| `harvx profiles list` | List all available profiles |
| `harvx profiles init [--template <name>]` | Generate starter `harvx.toml` |
| `harvx profiles show <name>` | Show resolved profile config |
| `harvx profiles lint` | Validate profiles, warn on issues |
| `harvx profiles explain <filepath>` | Show which rules apply to a file |

### Diagnostics

| Command | Description |
|---------|-------------|
| `harvx doctor` | Check for repo issues (binaries, oversized files, config) |
| `harvx doctor --fix` | Auto-fix detected issues |
| `harvx doctor --json` | Machine-readable diagnostic output |
| `harvx config debug` | Show resolved config with source annotations |
| `harvx cache show` | Show cached state summary |
| `harvx cache clear` | Clear cached state |
| `harvx version` | Show version and build info |

### MCP Server

| Command | Description |
|---------|-------------|
| `harvx mcp serve` | Start MCP server (stdio transport) for agent integration |

### Shell Completions

```bash
# Bash
harvx completion bash > /etc/bash_completion.d/harvx

# Zsh
harvx completion zsh > "${fpath[1]}/_harvx"

# Fish
harvx completion fish > ~/.config/fish/completions/harvx.fish
```

## Global Flags

| Flag | Env Var | Description |
|------|---------|-------------|
| `-d, --dir` | `HARVX_DIR` | Target directory (default: `.`) |
| `-o, --output` | `HARVX_OUTPUT` | Output file path |
| `-p, --profile` | | Profile name to use |
| `-f, --filter` | | Filter by file extension |
| `--include` | | Include glob pattern |
| `--exclude` | | Exclude glob pattern |
| `--format` | `HARVX_FORMAT` | Output format: `markdown`, `xml` |
| `--target` | `HARVX_TARGET` | LLM target: `claude`, `chatgpt`, `generic` |
| `--max-tokens` | `HARVX_MAX_TOKENS` | Token budget |
| `--tokenizer` | | Tokenizer: `cl100k_base`, `o200k_base`, `none` |
| `--compress` | | Enable code compression |
| `--compress-engine` | | Compression engine: `ast`, `regex`, `auto` |
| `--no-redact` | `HARVX_NO_REDACT` | Disable secret redaction |
| `--fail-on-redaction` | `HARVX_FAIL_ON_REDACTION` | Exit 1 if secrets detected |
| `--git-tracked-only` | | Only include git-tracked files |
| `--skip-large-files` | | Skip files larger than threshold |
| `--stdout` | `HARVX_STDOUT` | Output to stdout |
| `--split` | | Split output into chunks |
| `--output-metadata` | | Generate `.meta.json` sidecar |
| `-i, --interactive` | | Launch TUI mode |
| `-v, --verbose` | `HARVX_VERBOSE` | Debug-level logging |
| `-q, --quiet` | `HARVX_QUIET` | Suppress non-error output |
| `--clear-cache` | | Clear state cache before running |

## Claude Code Integration

### Hooks Setup

Add a session bootstrap hook in `.claude/hooks.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": ".*",
        "command": "harvx brief --stdout 2>/dev/null || true"
      }
    ]
  }
}
```

### MCP Integration

Configure Harvx as an MCP tool server:

```json
{
  "mcpServers": {
    "harvx": {
      "command": "harvx",
      "args": ["mcp", "serve"]
    }
  }
}
```

This exposes `brief`, `slice`, and `review_slice` as MCP tools that coding agents can invoke directly.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (or `--fail-on-redaction` triggered) |
| `2` | Partial success (some files failed but output was generated) |

## Comparison with Alternatives

| Feature | Harvx | Repomix | code2prompt |
|---------|-------|---------|-------------|
| Language | Go (single binary) | Node.js | Python |
| Runtime deps | None | npm | pip |
| Profile system | Yes (TOML, inheritance) | No | No |
| Relevance tiering | 6-tier glob matching | No | No |
| Token budgeting | cl100k/o200k + budget enforcement | Approximate | No |
| Secret redaction | 19 patterns + entropy analysis | No | No |
| Code compression | AST-aware (12 languages) | No | No |
| Diff-based context | Yes (cache + git) | No | No |
| Workflow commands | brief, slice, review-slice | No | No |
| Interactive TUI | Bubble Tea + vim keys | No | No |
| MCP server | Yes (stdio) | No | No |
| CI integration | Exit codes, `--quiet`, `--json` | Limited | Limited |
| Cross-platform | macOS, Linux, Windows | macOS, Linux, Windows | macOS, Linux, Windows |

## Development

```bash
# Build
go build ./cmd/harvx/

# Test
go test ./...

# Lint
go vet ./...

# Benchmarks
make bench

# Golden test updates
make golden-update

# Release snapshot (local)
make release-snapshot
```

## License

[MIT](LICENSE)
