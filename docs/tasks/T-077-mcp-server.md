# T-077: MCP Server v1.1 (`harvx mcp serve`)

**Priority:** Nice to Have
**Effort:** Large (16-24hrs)
**Dependencies:** T-066 (Pipeline Library API), T-070 (Brief Command), T-071 (Review Slice), T-072 (Slice Command)
**Phase:** 5 - Workflows

---

## Description

Implement `harvx mcp serve` as a Model Context Protocol (MCP) server that exposes Harvx's workflow commands as callable tools for coding agents like Codex CLI and Claude Code. The MCP server allows agents to request targeted context on demand rather than loading a massive bundle up front -- an agent working on auth can call `harvx.slice(path="internal/auth")` to get focused context.

## User Story

As a Codex CLI user, I want Harvx to run as an MCP server so that my coding agent can request targeted project context on demand, getting deep detail only for the areas it is actually working on.

## Acceptance Criteria

- [ ] `harvx mcp serve` command starts an MCP server using stdio transport
- [ ] Server exposes three tools via MCP protocol:
  - `harvx.brief` - Generate repo brief
    - Parameters: `profile` (optional string)
    - Returns: brief content as text, plus metadata (tokens, hash)
  - `harvx.slice` - Generate targeted module context
    - Parameters: `path` (required string), `profile` (optional string), `max_tokens` (optional int)
    - Returns: slice content as text, plus metadata
  - `harvx.review_slice` - Generate PR-specific context
    - Parameters: `base` (required string), `head` (required string), `profile` (optional string)
    - Returns: review slice content as text, plus metadata
- [ ] Server implements MCP specification version 2025-11-25
- [ ] Server uses the official Go MCP SDK (`github.com/modelcontextprotocol/go-sdk`)
- [ ] Each tool invocation uses the core pipeline library (T-066) -- no subprocess spawning
- [ ] Server includes proper error handling: invalid parameters return MCP error responses
- [ ] Server includes tool descriptions that help LLMs understand when to call each tool
- [ ] Server supports concurrent tool invocations (thread-safe pipeline)
- [ ] Server gracefully shuts down on SIGINT/SIGTERM
- [ ] Documentation for Codex CLI MCP configuration:
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

## Technical Notes

- Use the official Go MCP SDK: `github.com/modelcontextprotocol/go-sdk/mcp`
- Server creation pattern:
  ```go
  server := mcp.NewServer(&mcp.Implementation{
      Name:    "harvx",
      Version: version.Version,
  }, nil)
  mcp.AddTool(server, &mcp.Tool{
      Name:        "brief",
      Description: "Generate a stable repo brief with project overview, architecture, and key invariants",
  }, handleBrief)
  ```
- Each tool handler constructs a `Pipeline` with appropriate options and calls `Run()`
- Tool responses should include both the content and metadata (token count, content hash) as structured output
- Use stdio transport (`mcp.StdioTransport`) for communication -- this is the standard for local MCP servers
- The MCP server should log to stderr (not stdout) since stdio transport uses stdout/stdin for the protocol
- Thread safety: each tool invocation creates a new `Pipeline` instance, so concurrent calls are safe
- Consider caching brief results since they are stable across calls (invalidate on file changes)
- Reference: PRD Sections 5.11.2 (MCP server v1.1), 5.9 (mcp serve subcommand)
- Official MCP Go SDK: https://github.com/modelcontextprotocol/go-sdk
- MCP Specification: https://modelcontextprotocol.io/specification/2025-11-25

## Files to Create/Modify

- `internal/server/mcp.go` - MCP server setup and lifecycle
- `internal/server/tools.go` - Tool handlers (brief, slice, review_slice)
- `internal/server/mcp_test.go` - Server setup and tool registration tests
- `internal/server/tools_test.go` - Tool handler tests with mock pipeline
- `internal/cli/mcp.go` - Cobra command registration (`mcp serve`)
- `docs/guides/mcp-integration.md` - MCP setup documentation for Codex CLI

## Testing Requirements

- Unit test: Server registers all three tools with correct names and descriptions
- Unit test: `brief` tool handler returns content and metadata
- Unit test: `slice` tool handler validates required `path` parameter
- Unit test: `review_slice` tool handler validates required `base` and `head` parameters
- Unit test: Invalid parameters return MCP error responses
- Unit test: Tool handlers use pipeline library (not subprocess)
- Unit test: Concurrent tool invocations produce correct results
- Integration test: MCP server starts, responds to tool list request, and handles tool calls
- Edge case: Tool call with unknown profile name returns descriptive error
- Edge case: Server gracefully handles pipeline errors (returns MCP error, does not crash)