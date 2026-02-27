package workflows

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/output"
)

// WorkspaceOptions configures the workspace rendering workflow.
type WorkspaceOptions struct {
	// Config is the parsed workspace configuration.
	Config *config.WorkspaceConfig

	// ConfigDir is the directory containing workspace.toml (for relative
	// path resolution).
	ConfigDir string

	// Deep enables expanded mode with directory trees and file previews.
	Deep bool

	// Target selects LLM-specific output format (e.g., "claude" for XML).
	Target string

	// TokenCounter counts tokens in text. If nil, uses estimateTokens (len/4).
	TokenCounter func(text string) int
}

// WorkspaceResult holds the rendered workspace output.
type WorkspaceResult struct {
	// Content is the rendered workspace document.
	Content string

	// ContentHash is the XXH3 64-bit hash of the content for caching.
	ContentHash uint64

	// FormattedHash is the hex-formatted content hash string.
	FormattedHash string

	// TokenCount is the number of tokens in the rendered content.
	TokenCount int

	// RepoCount is the number of repositories in the workspace.
	RepoCount int
}

// WorkspaceJSON is the machine-readable metadata for --json output.
type WorkspaceJSON struct {
	// Name is the workspace display name.
	Name string `json:"name"`

	// Description is the workspace description.
	Description string `json:"description"`

	// RepoCount is the number of repositories in the workspace.
	RepoCount int `json:"repo_count"`

	// TokenCount is the number of tokens in the rendered output.
	TokenCount int `json:"token_count"`

	// ContentHash is the XXH3 content hash formatted as hex.
	ContentHash string `json:"content_hash"`

	// Repos lists the repository names in sorted order.
	Repos []string `json:"repos"`

	// Warnings contains validation warnings, if any.
	Warnings []string `json:"warnings,omitempty"`
}

// maxDirEntries is the maximum number of top-level directory entries shown
// per repo in --deep mode to keep output manageable.
const maxDirEntries = 30

// GenerateWorkspace renders a workspace manifest into structured output.
// The output is deterministic: sorted repos, stable rendering, content hash.
func GenerateWorkspace(opts WorkspaceOptions) (*WorkspaceResult, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("workspace: config is required")
	}

	countTokens := opts.TokenCounter
	if countTokens == nil {
		countTokens = estimateTokens
	}

	cfg := opts.Config

	// Sort repos by name for deterministic output. Work on a copy to avoid
	// mutating the caller's config (DC-1).
	repos := make([]config.WorkspaceRepo, len(cfg.Workspace.Repos))
	copy(repos, cfg.Workspace.Repos)
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	slog.Debug("generating workspace",
		"name", cfg.Workspace.Name,
		"repos", len(repos),
		"deep", opts.Deep,
		"target", opts.Target,
	)

	// Expand repo paths for rendering.
	for i := range repos {
		repos[i].Path = config.ExpandPath(repos[i].Path, opts.ConfigDir)
	}

	// Render the document based on target.
	var content string
	if opts.Target == "claude" {
		content = renderWorkspaceXML(cfg.Workspace.Name, cfg.Workspace.Description, repos, opts.Deep)
	} else {
		content = renderWorkspaceMarkdown(cfg.Workspace.Name, cfg.Workspace.Description, repos, opts.Deep)
	}

	// Compute content hash.
	hashEntries := []output.FileHashEntry{
		{Path: "workspace", Content: content},
	}
	hasher := output.NewContentHasher()
	contentHash, err := hasher.ComputeContentHash(hashEntries)
	if err != nil {
		return nil, fmt.Errorf("workspace: computing content hash: %w", err)
	}

	formattedHash := output.FormatHash(contentHash)
	tokenCount := countTokens(content)

	slog.Info("workspace generated",
		"token_count", tokenCount,
		"content_hash", formattedHash,
		"repo_count", len(repos),
	)

	return &WorkspaceResult{
		Content:       content,
		ContentHash:   contentHash,
		FormattedHash: formattedHash,
		TokenCount:    tokenCount,
		RepoCount:     len(repos),
	}, nil
}

// renderWorkspaceMarkdown renders the workspace as a Markdown document.
func renderWorkspaceMarkdown(name, description string, repos []config.WorkspaceRepo, deep bool) string {
	var b strings.Builder

	// Header.
	wsName := name
	if wsName == "" {
		wsName = "Workspace"
	}
	b.WriteString(fmt.Sprintf("# Workspace: %s\n\n", wsName))

	if description != "" {
		b.WriteString(fmt.Sprintf("> %s\n\n", description))
	}

	// Repositories section.
	if len(repos) > 0 {
		b.WriteString("## Repositories\n\n")

		for _, repo := range repos {
			b.WriteString(fmt.Sprintf("### %s\n", repo.Name))

			if repo.Path != "" {
				pathExists := pathExistsOnDisk(repo.Path)
				if pathExists {
					b.WriteString(fmt.Sprintf("- **Path:** %s\n", repo.Path))
				} else {
					b.WriteString(fmt.Sprintf("- **Path:** %s (not found)\n", repo.Path))
				}
			}

			if repo.Description != "" {
				b.WriteString(fmt.Sprintf("- **Description:** %s\n", repo.Description))
			}

			if len(repo.Entrypoints) > 0 {
				formatted := formatCodeList(repo.Entrypoints)
				b.WriteString(fmt.Sprintf("- **Entrypoints:** %s\n", formatted))
			}

			if len(repo.Docs) > 0 {
				formatted := formatCodeList(repo.Docs)
				b.WriteString(fmt.Sprintf("- **Docs:** %s\n", formatted))
			}

			if len(repo.IntegratesWith) > 0 {
				b.WriteString(fmt.Sprintf("- **Integrates with:** %s\n", strings.Join(repo.IntegratesWith, ", ")))
			}

			// Deep mode: directory listing.
			if deep && repo.Path != "" && pathExistsOnDisk(repo.Path) {
				listing := readDirListing(repo.Path, maxDirEntries)
				if listing != "" {
					b.WriteString("\n**Directory listing:**\n```\n")
					b.WriteString(listing)
					b.WriteString("```\n")
				}
			}

			b.WriteString("\n")
		}
	}

	// Integration graph section.
	integrations := buildIntegrationEdges(repos)
	if len(integrations) > 0 {
		b.WriteString("## Integration Graph\n\n")
		for _, edge := range integrations {
			b.WriteString(fmt.Sprintf("- %s\n", edge))
		}
		b.WriteString("\n")
	}

	// Shared schemas section.
	schemas := buildSharedSchemas(repos)
	if len(schemas) > 0 {
		b.WriteString("## Shared Schemas\n\n")
		for _, schema := range schemas {
			b.WriteString(fmt.Sprintf("- %s\n", schema))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderWorkspaceXML renders the workspace as XML for Claude-optimized consumption.
func renderWorkspaceXML(name, description string, repos []config.WorkspaceRepo, deep bool) string {
	var b strings.Builder

	wsName := name
	if wsName == "" {
		wsName = "Workspace"
	}
	b.WriteString(fmt.Sprintf("<workspace name=%q>\n", wsName))

	if description != "" {
		b.WriteString(fmt.Sprintf("<description>%s</description>\n", xmlEscape(description)))
	}

	if len(repos) > 0 {
		b.WriteString("<repos>\n")

		for _, repo := range repos {
			pathAttr := ""
			if repo.Path != "" {
				pathAttr = fmt.Sprintf(" path=%q", repo.Path)
			}
			b.WriteString(fmt.Sprintf("<repo name=%q%s>\n", repo.Name, pathAttr))

			if repo.Description != "" {
				b.WriteString(fmt.Sprintf("<description>%s</description>\n", xmlEscape(repo.Description)))
			}

			if len(repo.Entrypoints) > 0 {
				b.WriteString(fmt.Sprintf("<entrypoints>%s</entrypoints>\n", strings.Join(repo.Entrypoints, ", ")))
			}

			if len(repo.Docs) > 0 {
				b.WriteString(fmt.Sprintf("<docs>%s</docs>\n", strings.Join(repo.Docs, ", ")))
			}

			if len(repo.IntegratesWith) > 0 {
				b.WriteString(fmt.Sprintf("<integrates-with>%s</integrates-with>\n", strings.Join(repo.IntegratesWith, ", ")))
			}

			// Deep mode: directory listing.
			if deep && repo.Path != "" && pathExistsOnDisk(repo.Path) {
				listing := readDirListing(repo.Path, maxDirEntries)
				if listing != "" {
					b.WriteString("<directory-listing>\n")
					b.WriteString(listing)
					b.WriteString("</directory-listing>\n")
				}
			}

			b.WriteString("</repo>\n")
		}

		b.WriteString("</repos>\n")
	}

	// Integration graph.
	integrations := buildIntegrationEdges(repos)
	if len(integrations) > 0 {
		b.WriteString("<integrations>\n")
		for _, edge := range integrations {
			b.WriteString(edge)
			b.WriteString("\n")
		}
		b.WriteString("</integrations>\n")
	}

	// Shared schemas.
	schemas := buildSharedSchemas(repos)
	if len(schemas) > 0 {
		b.WriteString("<shared-schemas>\n")
		for _, schema := range schemas {
			b.WriteString(schema)
			b.WriteString("\n")
		}
		b.WriteString("</shared-schemas>\n")
	}

	b.WriteString("</workspace>\n")

	return b.String()
}

// buildIntegrationEdges constructs sorted integration edge strings from repos.
// Each edge has the format: "source → target1, target2".
func buildIntegrationEdges(repos []config.WorkspaceRepo) []string {
	var edges []string

	for _, repo := range repos {
		if len(repo.IntegratesWith) == 0 {
			continue
		}

		// Sort integration targets for determinism.
		targets := make([]string, len(repo.IntegratesWith))
		copy(targets, repo.IntegratesWith)
		sort.Strings(targets)

		edges = append(edges, fmt.Sprintf("%s \u2192 %s", repo.Name, strings.Join(targets, ", ")))
	}

	// Edges are already sorted because repos are sorted by name.
	return edges
}

// buildSharedSchemas constructs sorted shared schema strings from repos.
// Each entry has the format: "`schema` (repo-name)".
func buildSharedSchemas(repos []config.WorkspaceRepo) []string {
	var schemas []string

	for _, repo := range repos {
		for _, schema := range repo.SharedSchemas {
			schemas = append(schemas, fmt.Sprintf("`%s` (%s)", schema, repo.Name))
		}
	}

	// Sort for determinism.
	sort.Strings(schemas)
	return schemas
}

// formatCodeList formats a slice of strings as backtick-delimited code items
// joined by commas (e.g., "`src/main.ts`, `src/routes/`").
func formatCodeList(items []string) string {
	formatted := make([]string, len(items))
	for i, item := range items {
		formatted[i] = fmt.Sprintf("`%s`", item)
	}
	return strings.Join(formatted, ", ")
}

// pathExistsOnDisk reports whether the given path exists on the filesystem.
func pathExistsOnDisk(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readDirListing reads the top-level directory entries at path and returns
// them as a newline-separated listing. Directories get a trailing /.
// At most maxEntries entries are shown; excess is indicated with "...".
func readDirListing(path string, maxEntries int) string {
	entries, err := os.ReadDir(path)
	if err != nil {
		slog.Debug("workspace: could not read directory",
			"path", path,
			"err", err,
		)
		return ""
	}

	var b strings.Builder
	count := 0
	for _, entry := range entries {
		if count >= maxEntries {
			b.WriteString("... (truncated)\n")
			break
		}

		name := entry.Name()
		// Skip hidden files/dirs for cleaner output.
		if strings.HasPrefix(name, ".") {
			continue
		}

		if entry.IsDir() {
			b.WriteString(name + "/\n")
		} else {
			b.WriteString(name + "\n")
		}
		count++
	}

	return b.String()
}

// xmlEscape escapes special XML characters in text. This is a minimal
// escaper for use within XML tag content.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
