// Package server implements the MCP (Model Context Protocol) server for harvx.
// It exposes workflow commands as callable tools for coding agents like
// Codex CLI and Claude Code. The server communicates over stdio transport
// (stdin/stdout) using the JSON-RPC 2.0 protocol.
package server

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/harvx/harvx/internal/buildinfo"
)

// ServerConfig holds configuration for the MCP server.
type ServerConfig struct {
	// RootDir is the working directory for all tool invocations.
	RootDir string

	// Profile is the optional profile name for config resolution.
	Profile string
}

// NewMCPServer creates a new MCP server with all harvx tools registered.
func NewMCPServer(cfg ServerConfig) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "harvx",
		Version: buildinfo.Version,
	}, nil)

	registerTools(s, cfg)
	return s
}

// Serve runs the MCP server on stdio transport until the client disconnects
// or a termination signal is received. All logging goes to stderr since
// stdout/stdin are reserved for the MCP JSON-RPC protocol.
func Serve(ctx context.Context, s *mcp.Server) error {
	// Set up signal handling for graceful shutdown.
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("starting MCP server",
		"name", "harvx",
		"version", buildinfo.Version,
		"transport", "stdio",
	)

	// Ensure logging goes to stderr (stdout is used by MCP protocol).
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	return s.Run(ctx, &mcp.StdioTransport{})
}
