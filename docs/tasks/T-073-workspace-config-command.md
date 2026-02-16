# T-073: Workspace Manifest Config and Command (`harvx workspace`)

**Priority:** Must Have
**Effort:** Medium (10-14hrs)
**Dependencies:** T-016 (Config Types), T-066 (Pipeline Library API), T-067 (Stdout/Exit Codes)
**Phase:** 5 - Workflows

---

## Description

Implement the workspace manifest system: a `.harvx/workspace.toml` configuration file that describes multiple related repositories and their relationships, and a `harvx workspace` command that renders this manifest into a small, structured context section. This eliminates repeated explanations of how repos relate when working across a multi-repo codebase.

## User Story

As a developer working across multiple related repositories (API gateway, user service, shared UI library), I want to describe their relationships once in a config file so that every AI session starts with an understanding of the workspace structure without me repeating it.

## Acceptance Criteria

- [ ] `.harvx/workspace.toml` configuration file is parsed using `BurntSushi/toml`
- [ ] Workspace config supports the schema from PRD Section 5.11.3:
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
  entrypoints = ["cmd/server/main.go", "internal/handlers/"]
  integrates_with = ["api-gateway"]
  shared_schemas = ["proto/user.proto"]
  ```
- [ ] `harvx workspace` command renders the manifest into structured output:
  - Repo list with local paths (with `~` expansion)
  - 1-3 line description per repo
  - Key integration edges (which repos talk to which, shared schemas/libraries)
  - "Where to look" hints (entrypoints, docs)
- [ ] Default output is small: <= 1-2K tokens
- [ ] `--deep` mode includes expanded details: directory trees of each repo, key file previews
- [ ] Supports `--stdout`, `-o <path>`, `--json`, `--target`
- [ ] Workspace config auto-detection: looks for `.harvx/workspace.toml` in current directory and parent directories
- [ ] `harvx workspace init` generates a starter workspace config with placeholder entries
- [ ] Validates repo paths exist (warning if a repo path is not found, not fatal)
- [ ] Validates integration edges reference known repo names (warning if unknown)
- [ ] Output is deterministic: sorted repos, stable rendering, content hash
- [ ] Workspace output can be included in session bootstrap hooks (composable with `brief`)

## Technical Notes

- Implement config parsing in `internal/config/workspace.go` and rendering in `internal/workflows/workspace.go`
- CLI command in `internal/cli/workspace.go`
- Workspace config struct:
  ```go
  type WorkspaceConfig struct {
      Workspace struct {
          Name        string            `toml:"name"`
          Description string            `toml:"description"`
          Repos       []WorkspaceRepo   `toml:"repos"`
      } `toml:"workspace"`
  }
  type WorkspaceRepo struct {
      Name           string   `toml:"name"`
      Path           string   `toml:"path"`
      Description    string   `toml:"description"`
      Entrypoints    []string `toml:"entrypoints"`
      IntegratesWith []string `toml:"integrates_with"`
      SharedSchemas  []string `toml:"shared_schemas"`
      Docs           []string `toml:"docs"`
  }
  ```
- Path expansion: resolve `~` to `$HOME`, resolve relative paths relative to workspace.toml location
- `--deep` mode runs a lightweight discovery on each repo (directory tree only, no content loading) and adds it to the output
- Integration edges can be rendered as a simple graph section in the output
- For `--target claude`, render workspace as XML with `<workspace>`, `<repo>`, `<integrations>` tags
- The workspace command does NOT run the full pipeline -- it reads config and renders, making it very fast
- Consider future cross-repo slicing (PRD Open Question #8) but do NOT implement it now
- Reference: PRD Sections 5.11.3 (Workspace Manifest), 5.9 (workspace subcommand)

## Files to Create/Modify

- `internal/config/workspace.go` - WorkspaceConfig types and TOML parsing
- `internal/config/workspace_test.go` - Config parsing tests
- `internal/workflows/workspace.go` - Workspace rendering logic
- `internal/workflows/workspace_test.go` - Rendering tests
- `internal/cli/workspace.go` - Cobra command registration (`workspace`, `workspace init`)
- `internal/cli/workspace_test.go` - CLI tests
- `testdata/config/workspace.toml` - Test fixture workspace config
- `testdata/expected-output/workspace.md` - Golden test output

## Testing Requirements

- Unit test: Valid workspace.toml parses correctly with all fields
- Unit test: Minimal workspace.toml (just name and one repo) parses correctly
- Unit test: Path expansion resolves `~` to home directory
- Unit test: Missing repo path produces warning (not error)
- Unit test: Unknown integration target produces warning
- Unit test: Default output is within 2K token budget
- Unit test: `--deep` mode includes directory trees
- Unit test: Output is deterministic (two runs produce identical hash)
- Unit test: `workspace init` generates valid TOML with placeholder entries
- Unit test: `--json` returns structured metadata
- Golden test: Workspace output matches expected fixture
- Edge case: Empty workspace config (no repos) produces minimal output
- Edge case: Workspace.toml not found produces helpful error with creation instructions
- Edge case: All repo paths invalid produces warnings but still renders descriptions