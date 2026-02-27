# Workspace Setup Guide

Configure Harvx for multi-repo session bootstrap using workspace manifests.

## Overview

A workspace manifest (`.harvx/workspace.toml`) describes how multiple repositories relate to each other. The `harvx workspace` command produces a cross-repo context document for agents working across service boundaries.

## Creating a Workspace Manifest

### Quick start

```bash
# Generate a starter workspace.toml
harvx workspace init
```

This creates `.harvx/workspace.toml` with placeholder entries. Edit it to describe your repositories.

### Manual setup

Create `.harvx/workspace.toml`:

```toml
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
entrypoints = ["cmd/server/main.go"]
shared_schemas = ["proto/user.proto"]
integrates_with = ["api-gateway"]

[[workspace.repos]]
name = "billing-service"
path = "../billing-service"
description = "Billing and subscription management"
entrypoints = ["src/main.py"]
integrates_with = ["api-gateway"]
```

### Manifest fields

**Workspace level:**

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Workspace display name |
| `description` | No | Short description |

**Repo level:**

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Repo display name |
| `path` | Yes | Path to repo (absolute, `~`, or relative to workspace.toml) |
| `description` | No | Short description of the repo's role |
| `entrypoints` | No | Key entry files or directories |
| `docs` | No | Path to repo-specific documentation |
| `integrates_with` | No | Names of repos this repo communicates with |
| `shared_schemas` | No | Shared schema files (protobuf, OpenAPI, etc.) |

### Path resolution

Paths support:
- Absolute paths: `/home/user/work/api-gateway`
- Home directory expansion: `~/work/api-gateway`
- Relative paths: `../api-gateway` (resolved relative to workspace.toml location)

## Using Workspace in Session Bootstrap

### Hook configuration

Add workspace output to your session bootstrap hook:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "command": "harvx brief --stdout && echo '---' && harvx workspace --stdout",
        "timeout": 8000
      }
    ]
  }
}
```

This gives the agent both project-level context (brief) and cross-repo awareness (workspace).

### With Claude-optimized XML

```json
{
  "command": "harvx brief --target claude --stdout && harvx workspace --target claude --stdout",
  "timeout": 8000
}
```

XML output wraps workspace data in structured tags:

```xml
<workspace name="MyOrg Platform">
  <description>Microservices platform...</description>
  <repos>
    <repo name="api-gateway" path="/home/user/work/api-gateway">
      <description>Express.js API gateway...</description>
      <entrypoints>src/server.ts, src/routes/</entrypoints>
      <integrates-with>user-service, billing-service</integrates-with>
    </repo>
  </repos>
  <integrations>
    api-gateway → billing-service, user-service
  </integrations>
  <shared-schemas>
    `proto/user.proto` (user-service)
  </shared-schemas>
</workspace>
```

### Deep mode

Use `--deep` to include directory listings per repo (useful for initial project onboarding):

```bash
harvx workspace --deep --stdout
```

This adds a directory listing (up to 30 entries) for each repo, helping the agent understand project structure without reading every file.

## `workspace` Command Reference

```bash
harvx workspace [flags]
harvx workspace init [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | `false` | Output machine-readable JSON metadata |
| `--deep` | `false` | Include directory listings per repo |
| `--target` | `generic` | LLM target: `claude`, `chatgpt`, `generic` |
| `--output` | `harvx-workspace.md` | Output file path |
| `--stdout` | `false` | Output to stdout |

**Init flags:**

| Flag | Description |
|------|-------------|
| `--yes` | Overwrite existing workspace.toml without confirmation |

### JSON output

```bash
harvx workspace --json
```

```json
{
  "name": "MyOrg Platform",
  "description": "Microservices platform...",
  "repo_count": 3,
  "token_count": 850,
  "content_hash": "a1b2c3d4e5f6a7b8",
  "repos": ["api-gateway", "billing-service", "user-service"],
  "warnings": []
}
```

## Auto-Detection

The `harvx workspace` command automatically searches for `.harvx/workspace.toml` by walking up from the current directory (or `--dir`), stopping at the `.git` boundary. You don't need to specify the config path explicitly.

## Validation

Workspace manifests are validated on load. Warnings (not errors) are emitted for:
- Missing repo paths (directory doesn't exist)
- Unknown integration targets (repo referenced in `integrates_with` but not defined)
- Duplicate repo names
