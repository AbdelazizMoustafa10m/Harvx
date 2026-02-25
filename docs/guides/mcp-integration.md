# MCP Integration Guide

Harvx includes a built-in Model Context Protocol (MCP) server that exposes
workflow commands as callable tools for coding agents.

## Quick Setup

### Codex CLI

Add to your MCP configuration file (`~/.config/codex/mcp.json` or project-level):

```json
{
  "mcpServers": {
    "harvx": {
      "command": "harvx",
      "args": ["mcp", "serve"],
      "env": {
        "HARVX_PROFILE": "session"
      }
    }
  }
}
```

### Claude Code

Add to `.claude/settings.json` or global settings:

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

## Available Tools

### `brief`

Generate a stable repo brief (~1-4K tokens) with project overview, architecture,
build commands, and key invariants.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `profile` | string | No | Profile name for configuration |

**When to use:** At the start of a coding session to load project context.

### `slice`

Generate targeted context for a specific module or directory path and its
bounded neighborhood (imports, tests).

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | Relative directory or file path |
| `profile` | string | No | Profile name for configuration |
| `max_tokens` | integer | No | Token budget override |

**When to use:** When working on a specific area and you need focused context.

### `review_slice`

Generate PR-specific context containing changed files between two git refs
and their neighborhood.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `base` | string | Yes | Base git ref (e.g., `origin/main`) |
| `head` | string | Yes | Head git ref (e.g., `HEAD`) |
| `profile` | string | No | Profile name for configuration |

**When to use:** For AI-assisted code review.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `HARVX_PROFILE` | Default profile for all tool invocations |
| `HARVX_DIR` | Override working directory |
| `HARVX_VERBOSE` | Enable verbose logging (to stderr) |

## Architecture

The MCP server uses stdio transport (stdin/stdout for JSON-RPC 2.0).
All diagnostic logging goes to stderr. Each tool invocation creates
independent workflow state, so concurrent calls are safe.

```
Agent --stdin--> harvx mcp serve --stdout--> Agent
                      |
                      +-- brief -> workflows.GenerateBrief()
                      +-- slice -> workflows.GenerateModuleSlice()
                      +-- review_slice -> workflows.GenerateReviewSlice()
```

## Troubleshooting

**Server not starting:** Ensure `harvx` is in your PATH. Test with:
```bash
echo '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | harvx mcp serve 2>/dev/null
```

**Tool errors:** Enable verbose logging with `HARVX_VERBOSE=true` in the
MCP server environment. Logs appear on stderr.

**Profile not found:** Ensure the profile exists in `.harvx.toml` or
`~/.config/harvx/config.toml`.
