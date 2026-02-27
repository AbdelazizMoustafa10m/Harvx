# Session Bootstrap Guide

Set up Harvx as a session bootstrap tool for Claude Code so every new session starts with project awareness.

## Strategy

Keep your `CLAUDE.md` lean—rules and conventions only, under 500 tokens. Delegate architecture, structure, and build knowledge to Harvx, which runs automatically at session start via a hook.

```
┌─────────────────────────────────────────────────────┐
│ Session Start                                       │
│                                                     │
│  1. Claude Code reads CLAUDE.md (rules only)        │
│  2. SessionStart hook runs: harvx brief --stdout    │
│  3. Brief output injected as additional context     │
│  4. Agent has rules + project awareness             │
└─────────────────────────────────────────────────────┘
```

## Quick Setup

### 1. Create the hook configuration

Create `.claude/hooks.json` in your project root:

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

### 2. Create a lean CLAUDE.md

See [docs/templates/CLAUDE.md](../templates/CLAUDE.md) for a starter template. Key principles:

- Rules and conventions only (no architecture dumps)
- Under 500 tokens
- Reference Harvx for dynamic context

### 3. Verify the setup

```bash
# Test the hook command manually
harvx brief --profile session --stdout

# Time it to confirm it's under 2 seconds
time harvx brief --profile session --stdout > /dev/null
```

## Hook Configuration

### Placement

| Location | Scope |
|----------|-------|
| `.claude/hooks.json` | Project-specific (checked into repo) |
| `~/.claude/hooks.json` | Global (applies to all projects) |

Project-scoped hooks take precedence over global hooks.

### How it works

1. Claude Code fires the `SessionStart` event when a new session begins
2. The hook receives input data including `session_id`, `cwd`, and `model`
3. The hook command runs and its **stdout** is captured
4. The captured output is injected as `additionalContext` into the session
5. The agent sees the brief alongside your CLAUDE.md instructions

### Hook configuration fields

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | Shell command to execute |
| `timeout` | integer | Max execution time in milliseconds |

### Recommended commands

**Basic project brief:**
```json
{
  "command": "harvx brief --stdout",
  "timeout": 5000
}
```

**Claude-optimized XML output:**
```json
{
  "command": "harvx brief --target claude --stdout",
  "timeout": 5000
}
```

**With a named profile:**
```json
{
  "command": "harvx brief --profile session --stdout",
  "timeout": 5000
}
```

**Brief + workspace context (multi-repo):**
```json
{
  "command": "harvx brief --stdout && harvx workspace --stdout",
  "timeout": 8000
}
```

## `--target claude` Mode

When using `--target claude`, Harvx outputs XML following Anthropic's recommended tag structure:

```xml
<!-- Repo Brief | hash: a1b2c3d4 | tokens: 2800 -->

<repo-brief>
  <readme>
  ...README content...
  </readme>
  <key-invariants>
  ...CLAUDE.md / CONVENTIONS.md content...
  </key-invariants>
  <architecture>
  ...architecture docs...
  </architecture>
  <build-commands>
  ...Makefile targets / scripts...
  </build-commands>
  <project-config>
  ...go.mod / package.json info...
  </project-config>
  <module-map>
  ...top-level directory descriptions...
  </module-map>
</repo-brief>
```

The `--target claude` preset also sets:
- Output format: XML
- Default token budget: 200,000 tokens (for `generate`; brief uses its own budget)

## Performance

`harvx brief` is designed to complete in under 2 seconds for typical repositories:

| Repository Size | Expected Time |
|----------------|---------------|
| Small (<100 files) | <500ms |
| Medium (100-1000 files) | 500ms-1s |
| Large (1000+ files) | 1-2s |

The 5,000ms hook timeout provides comfortable headroom. If your repo is exceptionally large, consider using a session profile with constrained scope:

```toml
# .harvx.toml
[profiles.session]
brief_max_tokens = 3000
include = ["src/**", "docs/**"]
```

## Creating a Session Profile

A session profile optimizes the brief output for agent consumption:

```toml
# .harvx.toml
[profiles.session]
brief_max_tokens = 4000
include = ["src/**", "internal/**", "cmd/**", "docs/**"]
exclude = ["vendor/**", "node_modules/**", "testdata/**"]
```

Use it in the hook:

```json
{
  "command": "harvx brief --profile session --stdout",
  "timeout": 5000
}
```

## Environment Variables for Hooks

These `HARVX_*` environment variables are useful in hook contexts:

| Variable | Description |
|----------|-------------|
| `HARVX_DIR` | Override target directory |
| `HARVX_PROFILE` | Set profile without `--profile` flag |
| `HARVX_TARGET` | Set target (e.g., `claude`) |
| `HARVX_STDOUT` | Force stdout output (`true`/`1`/`yes`) |
| `HARVX_QUIET` | Suppress diagnostic messages (`1`) |
| `HARVX_COMPRESS` | Enable compression (`true`/`1`/`yes`) |

Example with env vars:

```json
{
  "command": "HARVX_TARGET=claude HARVX_QUIET=1 harvx brief --stdout",
  "timeout": 5000
}
```

## On-Demand Context with `slice`

During a session, agents can request targeted context about specific modules:

```bash
# Get context about a specific package
harvx slice --path internal/auth --stdout

# Multiple modules at once
harvx slice --path internal/auth --path internal/middleware --stdout

# With Claude-optimized XML
harvx slice --path internal/auth --target claude --stdout
```

The `slice` command includes the module's files plus related neighbors (tests, importers) within a configurable token budget (default: 20,000 tokens).

## Troubleshooting

### `harvx: command not found`

Ensure Harvx is in your `PATH`. Common solutions:

```bash
# Check that the binary is in PATH
which harvx

# If installed via go install
export PATH="$PATH:$(go env GOPATH)/bin"

# Or use the full path in the hook
{
  "command": "/usr/local/bin/harvx brief --stdout",
  "timeout": 5000
}
```

### Hook times out

If the hook exceeds the timeout:

1. Increase the timeout (e.g., `10000` for 10 seconds)
2. Use a constrained profile to reduce discovery scope
3. Add `--quiet` to suppress diagnostic output
4. Check if `--compress` is enabled (adds latency)

### Profile not found

```bash
# List available profiles
harvx profiles list

# Verify your profile exists in config
harvx config debug --profile session
```

### No output from hook

1. Test the command manually: `harvx brief --stdout`
2. Ensure `--stdout` flag is present (without it, output goes to a file)
3. Check for errors: `harvx brief --stdout --verbose 2>/dev/null`

### Stale context

The brief is regenerated on every session start, so it always reflects the current state of the repo. If you need to verify freshness, check the content hash in the output header.
