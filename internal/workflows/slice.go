// Package workflows implements high-level workflow commands for harvx.
// This file implements the module slice workflow which generates targeted
// context about specific module(s) or directory paths within a repository.
package workflows

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/harvx/harvx/internal/output"
)

// DefaultModuleSliceMaxTokens is the default token budget for the Module Slice
// artifact when no profile configuration overrides it.
const DefaultModuleSliceMaxTokens = 20000

// ModuleSliceOptions configures the module slice generation workflow.
type ModuleSliceOptions struct {
	// RootDir is the repository root directory.
	RootDir string

	// Paths is the list of relative directory or file paths to slice
	// (e.g., "internal/auth", "internal/middleware/auth.go").
	Paths []string

	// MaxTokens is the token budget. Zero uses DefaultModuleSliceMaxTokens.
	MaxTokens int

	// Depth is the neighborhood discovery depth. -1 means unset (use default).
	Depth int

	// Target selects LLM-specific output format ("claude" for XML).
	Target string

	// AssertInclude is a list of glob patterns for coverage checks.
	AssertInclude []string

	// TokenCounter counts tokens in text. If nil, uses character-based estimator.
	TokenCounter func(string) int

	// Compress enables compression for neighborhood files.
	Compress bool
}

// ModuleSliceResult holds the complete output of a module slice generation run.
type ModuleSliceResult struct {
	// Content is the rendered module slice document.
	Content string

	// ContentHash is the XXH3 64-bit hash.
	ContentHash uint64

	// FormattedHash is the hex-formatted hash string.
	FormattedHash string

	// TokenCount is the number of tokens in the content.
	TokenCount int

	// ModuleFiles lists the file paths within the --path scope.
	ModuleFiles []string

	// NeighborFiles lists the neighbor file paths included.
	NeighborFiles []string

	// TotalFiles is the total number of files in the output.
	TotalFiles int
}

// ModuleSliceJSON is the machine-readable metadata output for
// `harvx slice --json`.
type ModuleSliceJSON struct {
	TokenCount    int      `json:"token_count"`
	ContentHash   string   `json:"content_hash"`
	ModuleFiles   []string `json:"module_files"`
	NeighborFiles []string `json:"neighbor_files"`
	TotalFiles    int      `json:"total_files"`
	MaxTokens     int      `json:"max_tokens"`
	Paths         []string `json:"paths"`
}

// GenerateModuleSlice generates a targeted context slice for the specified
// module path(s). It discovers all files under the given paths, finds their
// neighbors (imports, tests), applies a token budget (module files first,
// neighbors second), and renders a deterministic document.
//
// The output is deterministic: identical repository state and paths produce
// identical content and content hash.
func GenerateModuleSlice(opts ModuleSliceOptions) (*ModuleSliceResult, error) {
	// Validate inputs.
	if opts.RootDir == "" {
		return nil, fmt.Errorf("slice: root directory required")
	}
	if len(opts.Paths) == 0 {
		return nil, fmt.Errorf("slice: at least one --path is required")
	}

	// Resolve defaults.
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = DefaultModuleSliceMaxTokens
	}

	depth := opts.Depth
	if depth < 0 {
		depth = DefaultSliceDepth
	}

	countTokens := opts.TokenCounter
	if countTokens == nil {
		countTokens = estimateTokens
	}

	slog.Debug("generating module slice",
		"root", opts.RootDir,
		"paths", opts.Paths,
		"max_tokens", maxTokens,
		"depth", depth,
		"target", opts.Target,
	)

	// Validate each path exists under RootDir.
	for _, p := range opts.Paths {
		absPath := filepath.Join(opts.RootDir, filepath.FromSlash(p))
		if _, err := os.Stat(absPath); err != nil {
			return nil, fmt.Errorf("slice: path %q does not exist under repository root: %w", p, err)
		}
	}

	// Collect all repo files for filtering and neighbor discovery.
	allFiles, err := collectRepoFiles(opts.RootDir)
	if err != nil {
		return nil, fmt.Errorf("slice: collecting repo files: %w", err)
	}

	// Filter files: a file is a "module file" if it belongs to any of the
	// specified paths. We normalize all paths with forward slashes for
	// cross-platform consistency.
	normalizedPaths := make([]string, len(opts.Paths))
	for i, p := range opts.Paths {
		normalizedPaths[i] = filepath.ToSlash(p)
	}

	moduleFileSet := make(map[string]bool)
	var modulePaths []string

	for _, f := range allFiles {
		if isModuleFile(f, normalizedPaths) {
			if !moduleFileSet[f] {
				moduleFileSet[f] = true
				modulePaths = append(modulePaths, f)
			}
		}
	}

	sort.Strings(modulePaths)

	if len(modulePaths) == 0 {
		pathList := strings.Join(opts.Paths, ", ")
		return nil, fmt.Errorf("slice: no files found under path(s): %s", pathList)
	}

	slog.Debug("slice: found module files",
		"count", len(modulePaths),
		"paths", opts.Paths,
	)

	// Discover neighbors using the same logic as review-slice.
	neighborPaths := discoverNeighbors(opts.RootDir, modulePaths, allFiles, depth)

	// Remove any neighbor paths that are already in the module file set.
	var filteredNeighbors []string
	for _, n := range neighborPaths {
		if !moduleFileSet[n] {
			filteredNeighbors = append(filteredNeighbors, n)
		}
	}
	neighborPaths = filteredNeighbors

	slog.Debug("slice: discovered neighbors",
		"module_files", len(modulePaths),
		"neighbors", len(neighborPaths),
	)

	// Build file lists with content.
	moduleFiles := buildSliceFiles(opts.RootDir, modulePaths, true, countTokens)
	neighborFiles := buildSliceFiles(opts.RootDir, neighborPaths, false, countTokens)

	// Enforce token budget: module files always included first.
	moduleFiles, neighborFiles = enforceSliceBudget(moduleFiles, neighborFiles, maxTokens)

	// Collect included file paths for assert-include checks.
	var allIncludedPaths []string
	for _, f := range moduleFiles {
		allIncludedPaths = append(allIncludedPaths, f.Path)
	}
	for _, f := range neighborFiles {
		allIncludedPaths = append(allIncludedPaths, f.Path)
	}

	// Check assert-include patterns.
	if len(opts.AssertInclude) > 0 {
		if assertErr := checkAssertIncludeBrief(opts.AssertInclude, allIncludedPaths); assertErr != nil {
			return nil, fmt.Errorf("slice: %w", assertErr)
		}
	}

	// Compute pre-render token count for the header.
	preTokenCount := 0
	for _, f := range moduleFiles {
		preTokenCount += f.Tokens
	}
	for _, f := range neighborFiles {
		preTokenCount += f.Tokens
	}

	// Compute content hash from all included file contents.
	hashEntries := make([]output.FileHashEntry, 0, len(moduleFiles)+len(neighborFiles))
	for _, f := range moduleFiles {
		hashEntries = append(hashEntries, output.FileHashEntry{
			Path:    f.Path,
			Content: f.Content,
		})
	}
	for _, f := range neighborFiles {
		hashEntries = append(hashEntries, output.FileHashEntry{
			Path:    f.Path,
			Content: f.Content,
		})
	}

	hasher := output.NewContentHasher()
	contentHash, err := hasher.ComputeContentHash(hashEntries)
	if err != nil {
		return nil, fmt.Errorf("slice: computing content hash: %w", err)
	}

	formattedHash := output.FormatHash(contentHash)

	// Render the document.
	content := renderModuleSlice(moduleFiles, neighborFiles, opts, formattedHash, preTokenCount)

	// Final token count after rendering.
	tokenCount := countTokens(content)

	// Collect result paths.
	var resultModulePaths []string
	for _, f := range moduleFiles {
		resultModulePaths = append(resultModulePaths, f.Path)
	}
	var resultNeighborPaths []string
	for _, f := range neighborFiles {
		resultNeighborPaths = append(resultNeighborPaths, f.Path)
	}

	totalFiles := len(moduleFiles) + len(neighborFiles)

	slog.Info("module slice generated",
		"token_count", tokenCount,
		"content_hash", formattedHash,
		"module_files", len(moduleFiles),
		"neighbor_files", len(neighborFiles),
		"total_files", totalFiles,
	)

	return &ModuleSliceResult{
		Content:       content,
		ContentHash:   contentHash,
		FormattedHash: formattedHash,
		TokenCount:    tokenCount,
		ModuleFiles:   resultModulePaths,
		NeighborFiles: resultNeighborPaths,
		TotalFiles:    totalFiles,
	}, nil
}

// isModuleFile reports whether the given file path belongs to any of the
// specified module paths. A file matches if:
//   - The file path equals a module path exactly (single file case)
//   - The file path starts with modulePath + "/" (directory case)
func isModuleFile(filePath string, modulePaths []string) bool {
	for _, mp := range modulePaths {
		if filePath == mp {
			return true
		}
		if strings.HasPrefix(filePath, mp+"/") {
			return true
		}
	}
	return false
}

// renderModuleSlice renders the module slice document. The format is selected
// based on the Target field in options: "claude" renders XML, anything else
// renders Markdown.
func renderModuleSlice(moduleFiles, neighborFiles []sliceFile, opts ModuleSliceOptions, contentHash string, tokenCount int) string {
	if opts.Target == "claude" {
		return renderModuleSliceXML(moduleFiles, neighborFiles, opts, contentHash, tokenCount)
	}
	return renderModuleSliceMarkdown(moduleFiles, neighborFiles, opts, contentHash, tokenCount)
}

// renderModuleSliceMarkdown renders the module slice as a Markdown document.
func renderModuleSliceMarkdown(moduleFiles, neighborFiles []sliceFile, opts ModuleSliceOptions, contentHash string, tokenCount int) string {
	var b strings.Builder

	// Header.
	b.WriteString("# Module Slice\n\n")

	// Build paths display.
	pathParts := make([]string, len(opts.Paths))
	for i, p := range opts.Paths {
		pathParts[i] = fmt.Sprintf("`%s`", p)
	}
	pathsDisplay := strings.Join(pathParts, ", ")

	fmt.Fprintf(&b, "> paths: %s | hash: `%s` | tokens: %d\n\n",
		pathsDisplay, contentHash, tokenCount)

	// Module files section.
	if len(moduleFiles) > 0 {
		b.WriteString("## Module Files\n\n")
		for _, f := range moduleFiles {
			fmt.Fprintf(&b, "### `%s`\n\n", f.Path)
			lang := languageFromPath(f.Path)
			fmt.Fprintf(&b, "```%s\n", lang)
			b.WriteString(f.Content)
			if !strings.HasSuffix(f.Content, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("```\n\n")
		}
	}

	// Neighborhood context section.
	if len(neighborFiles) > 0 {
		b.WriteString("## Neighborhood Context\n\n")
		for _, f := range neighborFiles {
			fmt.Fprintf(&b, "### `%s`\n\n", f.Path)
			lang := languageFromPath(f.Path)
			fmt.Fprintf(&b, "```%s\n", lang)
			b.WriteString(f.Content)
			if !strings.HasSuffix(f.Content, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("```\n\n")
		}
	}

	return b.String()
}

// renderModuleSliceXML renders the module slice as XML for Claude-optimized
// consumption.
func renderModuleSliceXML(moduleFiles, neighborFiles []sliceFile, opts ModuleSliceOptions, contentHash string, tokenCount int) string {
	var b strings.Builder

	// XML comment header.
	pathsDisplay := strings.Join(opts.Paths, ", ")
	fmt.Fprintf(&b, "<!-- Module Slice | paths: %s | hash: %s | tokens: %d -->\n",
		pathsDisplay, contentHash, tokenCount)
	b.WriteString("<module-slice>\n")

	// Module files.
	if len(moduleFiles) > 0 {
		b.WriteString("<module-files>\n")
		for _, f := range moduleFiles {
			fmt.Fprintf(&b, "<file path=%q>\n", f.Path)
			b.WriteString("<content>\n")
			b.WriteString(f.Content)
			if !strings.HasSuffix(f.Content, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("</content>\n")
			b.WriteString("</file>\n")
		}
		b.WriteString("</module-files>\n")
	}

	// Neighborhood.
	if len(neighborFiles) > 0 {
		b.WriteString("<neighborhood>\n")
		for _, f := range neighborFiles {
			fmt.Fprintf(&b, "<file path=%q>\n", f.Path)
			b.WriteString("<content>\n")
			b.WriteString(f.Content)
			if !strings.HasSuffix(f.Content, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("</content>\n")
			b.WriteString("</file>\n")
		}
		b.WriteString("</neighborhood>\n")
	}

	b.WriteString("</module-slice>\n")

	return b.String()
}
