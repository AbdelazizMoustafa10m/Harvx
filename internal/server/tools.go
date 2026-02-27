package server

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/harvx/harvx/internal/workflows"
)

// briefInput defines the input schema for the brief tool.
type briefInput struct {
	Profile string `json:"profile,omitempty"`
}

// briefOutput defines the output schema for the brief tool.
type briefOutput struct {
	Content     string `json:"content"`
	TokenCount  int    `json:"token_count"`
	ContentHash string `json:"content_hash"`
}

// sliceInput defines the input schema for the slice tool.
type sliceInput struct {
	Path      string `json:"path"`
	Profile   string `json:"profile,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
}

// sliceOutput defines the output schema for the slice tool.
type sliceOutput struct {
	Content       string   `json:"content"`
	TokenCount    int      `json:"token_count"`
	ContentHash   string   `json:"content_hash"`
	ModuleFiles   []string `json:"module_files"`
	NeighborFiles []string `json:"neighbor_files"`
}

// reviewSliceInput defines the input schema for the review_slice tool.
type reviewSliceInput struct {
	Base    string `json:"base"`
	Head    string `json:"head"`
	Profile string `json:"profile,omitempty"`
}

// reviewSliceOutput defines the output schema for the review_slice tool.
type reviewSliceOutput struct {
	Content       string   `json:"content"`
	TokenCount    int      `json:"token_count"`
	ContentHash   string   `json:"content_hash"`
	ChangedFiles  []string `json:"changed_files"`
	NeighborFiles []string `json:"neighbor_files"`
	DeletedFiles  []string `json:"deleted_files"`
}

// registerTools registers all harvx MCP tools on the server.
func registerTools(s *mcp.Server, cfg ServerConfig) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "brief",
		Description: "Generate a stable repo brief (~1-4K tokens) with project overview, " +
			"architecture, build commands, and key invariants. Use this for initial " +
			"context loading at the start of a coding session.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, makeBriefHandler(cfg))

	mcp.AddTool(s, &mcp.Tool{
		Name: "slice",
		Description: "Generate targeted context for a specific module or directory path " +
			"and its bounded neighborhood (imports, tests). Use this when working on " +
			"a specific area of the codebase and you need focused, deep context.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, makeSliceHandler(cfg))

	mcp.AddTool(s, &mcp.Tool{
		Name: "review_slice",
		Description: "Generate PR-specific context containing changed files between two " +
			"git refs and their neighborhood. Use this for AI-assisted code review " +
			"when you need to understand what changed and its surrounding context.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, makeReviewSliceHandler(cfg))
}

// makeBriefHandler returns a typed tool handler for the brief tool.
func makeBriefHandler(cfg ServerConfig) mcp.ToolHandlerFor[briefInput, briefOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input briefInput) (*mcp.CallToolResult, briefOutput, error) {
		rootDir := cfg.RootDir
		if rootDir == "" {
			var err error
			rootDir, err = filepath.Abs(".")
			if err != nil {
				return nil, briefOutput{}, fmt.Errorf("brief: resolving working directory: %w", err)
			}
		}

		opts := workflows.BriefOptions{
			RootDir:   rootDir,
			MaxTokens: workflows.DefaultBriefMaxTokens,
		}

		result, err := workflows.GenerateBrief(opts)
		if err != nil {
			return nil, briefOutput{}, fmt.Errorf("brief: %w", err)
		}

		return nil, briefOutput{
			Content:     result.Content,
			TokenCount:  result.TokenCount,
			ContentHash: result.FormattedHash,
		}, nil
	}
}

// makeSliceHandler returns a typed tool handler for the slice tool.
func makeSliceHandler(cfg ServerConfig) mcp.ToolHandlerFor[sliceInput, sliceOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input sliceInput) (*mcp.CallToolResult, sliceOutput, error) {
		if input.Path == "" {
			return nil, sliceOutput{}, fmt.Errorf("slice: path parameter is required")
		}

		rootDir := cfg.RootDir
		if rootDir == "" {
			var err error
			rootDir, err = filepath.Abs(".")
			if err != nil {
				return nil, sliceOutput{}, fmt.Errorf("slice: resolving working directory: %w", err)
			}
		}

		maxTokens := input.MaxTokens
		if maxTokens == 0 {
			maxTokens = workflows.DefaultModuleSliceMaxTokens
		}

		opts := workflows.ModuleSliceOptions{
			RootDir:   rootDir,
			Paths:     []string{input.Path},
			MaxTokens: maxTokens,
			Depth:     workflows.DefaultSliceDepth,
		}

		result, err := workflows.GenerateModuleSlice(opts)
		if err != nil {
			return nil, sliceOutput{}, fmt.Errorf("slice: %w", err)
		}

		return nil, sliceOutput{
			Content:       result.Content,
			TokenCount:    result.TokenCount,
			ContentHash:   result.FormattedHash,
			ModuleFiles:   nonNilSlice(result.ModuleFiles),
			NeighborFiles: nonNilSlice(result.NeighborFiles),
		}, nil
	}
}

// makeReviewSliceHandler returns a typed tool handler for the review_slice tool.
func makeReviewSliceHandler(cfg ServerConfig) mcp.ToolHandlerFor[reviewSliceInput, reviewSliceOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input reviewSliceInput) (*mcp.CallToolResult, reviewSliceOutput, error) {
		if input.Base == "" {
			return nil, reviewSliceOutput{}, fmt.Errorf("review_slice: base parameter is required")
		}
		if input.Head == "" {
			return nil, reviewSliceOutput{}, fmt.Errorf("review_slice: head parameter is required")
		}

		rootDir := cfg.RootDir
		if rootDir == "" {
			var err error
			rootDir, err = filepath.Abs(".")
			if err != nil {
				return nil, reviewSliceOutput{}, fmt.Errorf("review_slice: resolving working directory: %w", err)
			}
		}

		opts := workflows.ReviewSliceOptions{
			RootDir: rootDir,
			BaseRef: input.Base,
			HeadRef: input.Head,
			Depth:   workflows.DefaultSliceDepth,
		}

		result, err := workflows.GenerateReviewSlice(ctx, opts)
		if err != nil {
			return nil, reviewSliceOutput{}, fmt.Errorf("review_slice: %w", err)
		}

		return nil, reviewSliceOutput{
			Content:       result.Content,
			TokenCount:    result.TokenCount,
			ContentHash:   result.FormattedHash,
			ChangedFiles:  nonNilSlice(result.ChangedFiles),
			NeighborFiles: nonNilSlice(result.NeighborFiles),
			DeletedFiles:  nonNilSlice(result.DeletedFiles),
		}, nil
	}
}

// nonNilSlice returns s if non-nil, or an empty slice. This prevents JSON
// null in MCP structured output, which some clients handle poorly.
func nonNilSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
