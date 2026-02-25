package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/harvx/harvx/internal/server"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Model Context Protocol server commands",
	Long: `MCP server commands for exposing harvx workflows as callable tools
for coding agents like Codex CLI and Claude Code.

The MCP server communicates over stdio transport (stdin/stdout) using the
JSON-RPC 2.0 protocol defined by the Model Context Protocol specification.`,
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start an MCP server over stdio",
	Long: `Start an MCP server that exposes harvx workflow commands as callable tools.

Available tools:
  brief          Generate a stable repo brief (~1-4K tokens)
  slice          Generate targeted context for a module or directory
  review_slice   Generate PR-specific context for changed files

Example MCP configuration for Codex CLI or Claude Code:

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
  }`,
	RunE: runMCPServe,
}

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
	rootCmd.AddCommand(mcpCmd)
}

func runMCPServe(cmd *cobra.Command, _ []string) error {
	fv := GlobalFlags()

	rootDir, err := filepath.Abs(fv.Dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	cfg := server.ServerConfig{
		RootDir: rootDir,
		Profile: fv.Profile,
	}

	s := server.NewMCPServer(cfg)
	return server.Serve(cmd.Context(), s)
}
