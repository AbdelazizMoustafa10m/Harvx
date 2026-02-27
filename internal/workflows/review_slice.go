package workflows

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/harvx/harvx/internal/diff"
	"github.com/harvx/harvx/internal/output"
)

// DefaultSliceMaxTokens is the default token budget for the Review Slice.
const DefaultSliceMaxTokens = 20000

// DefaultSliceDepth is the default neighborhood discovery depth.
const DefaultSliceDepth = 1

// ReviewSliceOptions configures the review-slice generation workflow.
type ReviewSliceOptions struct {
	// RootDir is the repository root directory.
	RootDir string

	// BaseRef is the base git ref (e.g., "origin/main", a commit SHA).
	BaseRef string

	// HeadRef is the head git ref (e.g., "HEAD", a branch name).
	HeadRef string

	// MaxTokens is the token budget. Zero uses DefaultSliceMaxTokens.
	MaxTokens int

	// Depth is the neighborhood discovery depth. -1 means unset (use default).
	Depth int

	// Target selects LLM-specific output format ("claude" for XML).
	Target string

	// AssertInclude is a list of glob patterns for coverage checks.
	AssertInclude []string

	// TokenCounter counts tokens in text. If nil, uses character-based estimator.
	TokenCounter func(text string) int

	// Compress enables compression for neighborhood files.
	Compress bool
}

// ReviewSliceResult holds the complete output of a review-slice generation run.
type ReviewSliceResult struct {
	// Content is the rendered review slice document.
	Content string

	// ContentHash is the XXH3 64-bit hash.
	ContentHash uint64

	// FormattedHash is the hex-formatted hash string.
	FormattedHash string

	// TokenCount is the number of tokens in the content.
	TokenCount int

	// ChangedFiles lists the changed file paths.
	ChangedFiles []string

	// NeighborFiles lists the neighbor file paths included.
	NeighborFiles []string

	// DeletedFiles lists files deleted between base and head.
	DeletedFiles []string

	// TotalFiles is the total number of files in the output.
	TotalFiles int
}

// ReviewSliceJSON is the machine-readable metadata output for
// `harvx review-slice --json`.
type ReviewSliceJSON struct {
	TokenCount    int      `json:"token_count"`
	ContentHash   string   `json:"content_hash"`
	ChangedFiles  []string `json:"changed_files"`
	NeighborFiles []string `json:"neighbor_files"`
	DeletedFiles  []string `json:"deleted_files"`
	TotalFiles    int      `json:"total_files"`
	MaxTokens     int      `json:"max_tokens"`
	BaseRef       string   `json:"base_ref"`
	HeadRef       string   `json:"head_ref"`
}

// sliceFile represents a single file entry in the review slice output.
type sliceFile struct {
	Path         string
	Content      string
	Tokens       int
	IsChanged    bool // true for changed files, false for neighbors
	IsCompressed bool
}

// skipDirs lists directory names to skip during repository file collection.
var skipDirs = map[string]bool{
	".git":        true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"__pycache__":  true,
}

// GenerateReviewSlice generates a PR-specific context slice containing changed
// files and their bounded neighborhood. The output is deterministic: identical
// repository state and refs produce identical content and content hash.
func GenerateReviewSlice(ctx context.Context, opts ReviewSliceOptions) (*ReviewSliceResult, error) {
	// Validate inputs.
	if opts.RootDir == "" {
		return nil, fmt.Errorf("review-slice: root directory required")
	}
	if opts.BaseRef == "" {
		return nil, fmt.Errorf("review-slice: base ref required")
	}
	if opts.HeadRef == "" {
		return nil, fmt.Errorf("review-slice: head ref required")
	}

	// Resolve defaults.
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = DefaultSliceMaxTokens
	}

	depth := opts.Depth
	if depth < 0 {
		depth = DefaultSliceDepth
	}

	countTokens := opts.TokenCounter
	if countTokens == nil {
		countTokens = estimateTokens
	}

	slog.Debug("generating review slice",
		"root", opts.RootDir,
		"base_ref", opts.BaseRef,
		"head_ref", opts.HeadRef,
		"max_tokens", maxTokens,
		"depth", depth,
		"target", opts.Target,
	)

	// Get changed files via git diff.
	differ := diff.NewGitDiffer()
	changes, err := differ.GetChangedFiles(ctx, opts.RootDir, opts.BaseRef, opts.HeadRef)
	if err != nil {
		return nil, fmt.Errorf("review-slice: getting changed files: %w", err)
	}

	// Separate changes into changed paths and deleted paths.
	var changedPaths []string
	var deletedPaths []string

	for _, c := range changes {
		switch c.Status {
		case diff.GitAdded, diff.GitModified, diff.GitRenamed:
			changedPaths = append(changedPaths, c.Path)
		case diff.GitDeleted:
			deletedPaths = append(deletedPaths, c.Path)
		}
	}

	// Sort for deterministic output.
	sort.Strings(changedPaths)
	sort.Strings(deletedPaths)

	// If no changed files, return early with empty slice.
	if len(changedPaths) == 0 {
		slog.Info("review-slice: no changed files between refs",
			"base_ref", opts.BaseRef,
			"head_ref", opts.HeadRef,
		)

		content := renderEmptySlice(opts)
		tokenCount := countTokens(content)

		hasher := output.NewContentHasher()
		contentHash, hashErr := hasher.ComputeContentHash([]output.FileHashEntry{
			{Path: "review-slice", Content: content},
		})
		if hashErr != nil {
			return nil, fmt.Errorf("review-slice: computing content hash: %w", hashErr)
		}

		return &ReviewSliceResult{
			Content:       content,
			ContentHash:   contentHash,
			FormattedHash: output.FormatHash(contentHash),
			TokenCount:    tokenCount,
			DeletedFiles:  deletedPaths,
		}, nil
	}

	// Collect all repo files for neighbor discovery.
	allFiles, err := collectRepoFiles(opts.RootDir)
	if err != nil {
		return nil, fmt.Errorf("review-slice: collecting repo files: %w", err)
	}

	// Discover neighbor files.
	neighborPaths := discoverNeighbors(opts.RootDir, changedPaths, allFiles, depth)

	slog.Debug("review-slice: discovered neighbors",
		"changed", len(changedPaths),
		"neighbors", len(neighborPaths),
		"deleted", len(deletedPaths),
	)

	// Build file lists with content.
	changedFiles := buildSliceFiles(opts.RootDir, changedPaths, true, countTokens)
	neighborFiles := buildSliceFiles(opts.RootDir, neighborPaths, false, countTokens)

	// Enforce token budget.
	changedFiles, neighborFiles = enforceSliceBudget(changedFiles, neighborFiles, maxTokens)

	// Collect included file paths for assert-include checks.
	var allIncludedPaths []string
	for _, f := range changedFiles {
		allIncludedPaths = append(allIncludedPaths, f.Path)
	}
	for _, f := range neighborFiles {
		allIncludedPaths = append(allIncludedPaths, f.Path)
	}

	// Check assert-include patterns.
	if len(opts.AssertInclude) > 0 {
		if assertErr := checkAssertIncludeBrief(opts.AssertInclude, allIncludedPaths); assertErr != nil {
			return nil, fmt.Errorf("review-slice: %w", assertErr)
		}
	}

	// Compute token count before rendering (for the header).
	preTokenCount := 0
	for _, f := range changedFiles {
		preTokenCount += f.Tokens
	}
	for _, f := range neighborFiles {
		preTokenCount += f.Tokens
	}

	// Compute content hash from all included file contents.
	hashEntries := make([]output.FileHashEntry, 0, len(changedFiles)+len(neighborFiles))
	for _, f := range changedFiles {
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
		return nil, fmt.Errorf("review-slice: computing content hash: %w", err)
	}

	formattedHash := output.FormatHash(contentHash)

	// Render the document.
	content := renderSlice(changedFiles, neighborFiles, deletedPaths, opts, formattedHash, preTokenCount)

	// Final token count after rendering.
	tokenCount := countTokens(content)

	// Collect result paths.
	var resultChangedPaths []string
	for _, f := range changedFiles {
		resultChangedPaths = append(resultChangedPaths, f.Path)
	}
	var resultNeighborPaths []string
	for _, f := range neighborFiles {
		resultNeighborPaths = append(resultNeighborPaths, f.Path)
	}

	totalFiles := len(changedFiles) + len(neighborFiles)

	slog.Info("review-slice generated",
		"token_count", tokenCount,
		"content_hash", formattedHash,
		"changed_files", len(changedFiles),
		"neighbor_files", len(neighborFiles),
		"deleted_files", len(deletedPaths),
		"total_files", totalFiles,
	)

	return &ReviewSliceResult{
		Content:       content,
		ContentHash:   contentHash,
		FormattedHash: formattedHash,
		TokenCount:    tokenCount,
		ChangedFiles:  resultChangedPaths,
		NeighborFiles: resultNeighborPaths,
		DeletedFiles:  deletedPaths,
		TotalFiles:    totalFiles,
	}, nil
}

// collectRepoFiles walks the directory tree collecting all file paths relative
// to rootDir. Hidden directories and common non-source directories are skipped.
func collectRepoFiles(rootDir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden and non-source directories.
		if d.IsDir() {
			name := d.Name()
			if name != "." && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if skipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, relErr)
		}

		// Normalize to forward slashes for cross-platform consistency.
		relPath = filepath.ToSlash(relPath)

		files = append(files, relPath)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", rootDir, err)
	}

	sort.Strings(files)
	return files, nil
}

// buildSliceFiles reads file content for each path and builds sliceFile entries.
// Files that cannot be read are silently skipped. The isChanged flag marks
// whether these are changed or neighbor files.
func buildSliceFiles(rootDir string, paths []string, isChanged bool, countTokens func(string) int) []sliceFile {
	files := make([]sliceFile, 0, len(paths))

	for _, p := range paths {
		absPath := filepath.Join(rootDir, filepath.FromSlash(p))
		data, err := os.ReadFile(absPath)
		if err != nil {
			slog.Debug("review-slice: skipping unreadable file",
				"path", p,
				"error", err,
			)
			continue
		}

		content := string(data)
		tokens := countTokens(content)

		files = append(files, sliceFile{
			Path:      p,
			Content:   content,
			Tokens:    tokens,
			IsChanged: isChanged,
		})
	}

	return files
}

// enforceSliceBudget enforces the token budget for the review slice. Changed
// files are always included first (highest priority). Neighbor files are added
// in order until the budget is reached. Returns the (unchanged) changed files
// and the possibly truncated neighbor list.
func enforceSliceBudget(changed, neighbors []sliceFile, maxTokens int) ([]sliceFile, []sliceFile) {
	// Sum tokens for changed files -- these are always included.
	usedTokens := 0
	for _, f := range changed {
		usedTokens += f.Tokens
	}

	slog.Debug("review-slice: budget enforcement",
		"changed_tokens", usedTokens,
		"max_tokens", maxTokens,
		"neighbor_count", len(neighbors),
	)

	// If changed files alone exceed the budget, include them all but no neighbors.
	if usedTokens >= maxTokens {
		slog.Debug("review-slice: changed files exceed budget, omitting all neighbors",
			"used_tokens", usedTokens,
			"max_tokens", maxTokens,
		)
		return changed, nil
	}

	// Add neighbors in order until the budget is reached.
	remaining := maxTokens - usedTokens
	var included []sliceFile

	for _, f := range neighbors {
		if f.Tokens > remaining {
			slog.Debug("review-slice: budget reached, truncating neighbors",
				"remaining", remaining,
				"next_file_tokens", f.Tokens,
				"next_file", f.Path,
			)
			break
		}
		included = append(included, f)
		remaining -= f.Tokens
	}

	return changed, included
}

// renderSlice renders the review slice document. The format is selected based
// on the Target field in options: "claude" renders XML, anything else renders
// Markdown.
func renderSlice(changed, neighbors []sliceFile, deleted []string, opts ReviewSliceOptions, contentHash string, tokenCount int) string {
	if opts.Target == "claude" {
		return renderSliceXML(changed, neighbors, deleted, opts, contentHash, tokenCount)
	}
	return renderSliceMarkdown(changed, neighbors, deleted, opts, contentHash, tokenCount)
}

// renderSliceMarkdown renders the review slice as a Markdown document.
func renderSliceMarkdown(changed, neighbors []sliceFile, deleted []string, opts ReviewSliceOptions, contentHash string, tokenCount int) string {
	var b strings.Builder

	// Header.
	b.WriteString("# Review Slice\n\n")
	fmt.Fprintf(&b, "> base: `%s` → head: `%s` | hash: `%s` | tokens: %d\n\n",
		opts.BaseRef, opts.HeadRef, contentHash, tokenCount)

	// Changed files section.
	if len(changed) > 0 {
		b.WriteString("## Changed Files\n\n")
		for _, f := range changed {
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

	// Neighborhood section.
	if len(neighbors) > 0 {
		b.WriteString("## Neighborhood Context\n\n")
		for _, f := range neighbors {
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

	// Deleted files section.
	if len(deleted) > 0 {
		b.WriteString("## Deleted Files\n\n")
		for _, p := range deleted {
			fmt.Fprintf(&b, "- `%s`\n", p)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderSliceXML renders the review slice as XML for Claude-optimized
// consumption.
func renderSliceXML(changed, neighbors []sliceFile, deleted []string, opts ReviewSliceOptions, contentHash string, tokenCount int) string {
	var b strings.Builder

	// XML comment header.
	fmt.Fprintf(&b, "<!-- Review Slice | base: %s → head: %s | hash: %s | tokens: %d -->\n",
		opts.BaseRef, opts.HeadRef, contentHash, tokenCount)
	b.WriteString("<review-slice>\n")

	// Changed files.
	if len(changed) > 0 {
		b.WriteString("<changed-files>\n")
		for _, f := range changed {
			fmt.Fprintf(&b, "<file path=%q>\n", f.Path)
			b.WriteString("<content>\n")
			b.WriteString(f.Content)
			if !strings.HasSuffix(f.Content, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("</content>\n")
			b.WriteString("</file>\n")
		}
		b.WriteString("</changed-files>\n")
	}

	// Neighborhood.
	if len(neighbors) > 0 {
		b.WriteString("<neighborhood>\n")
		for _, f := range neighbors {
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

	// Deleted files.
	if len(deleted) > 0 {
		b.WriteString("<deleted-files>\n")
		for _, p := range deleted {
			fmt.Fprintf(&b, "<file path=%q/>\n", p)
		}
		b.WriteString("</deleted-files>\n")
	}

	b.WriteString("</review-slice>\n")

	return b.String()
}

// renderEmptySlice renders an empty review slice with a message indicating
// no files changed between the base and head refs.
func renderEmptySlice(opts ReviewSliceOptions) string {
	if opts.Target == "claude" {
		return fmt.Sprintf(
			"<!-- Review Slice | base: %s → head: %s | No changes detected -->\n"+
				"<review-slice>\n"+
				"<message>No files changed between %s and %s.</message>\n"+
				"</review-slice>\n",
			opts.BaseRef, opts.HeadRef, opts.BaseRef, opts.HeadRef,
		)
	}
	return fmt.Sprintf(
		"# Review Slice\n\n"+
			"> base: `%s` → head: `%s`\n\n"+
			"No files changed between `%s` and `%s`.\n",
		opts.BaseRef, opts.HeadRef, opts.BaseRef, opts.HeadRef,
	)
}

// discoverNeighbors finds files that are related to the changed files via
// import relationships and co-location heuristics. The depth parameter controls
// how many levels of transitive imports are followed (0 means no neighbors,
// 1 means direct imports only, etc.).
//
// The algorithm:
//  1. For each changed file, parse its imports to find direct dependencies.
//  2. For each repo file, parse its imports to find reverse dependencies
//     (files that import a changed file).
//  3. Include test files matching changed file patterns.
//  4. For unsupported languages, include same-directory files as neighbors.
//  5. Repeat for additional depth levels.
func discoverNeighbors(rootDir string, changedPaths, allFiles []string, depth int) []string {
	if depth <= 0 {
		return nil
	}

	changedSet := make(map[string]bool, len(changedPaths))
	for _, p := range changedPaths {
		changedSet[p] = true
	}

	// Build a set of all repo files for quick lookup.
	allFileSet := make(map[string]bool, len(allFiles))
	for _, f := range allFiles {
		allFileSet[f] = true
	}

	neighborSet := make(map[string]bool)

	// Current frontier: files we're discovering neighbors for.
	frontier := make(map[string]bool, len(changedPaths))
	for _, p := range changedPaths {
		frontier[p] = true
	}

	for level := 0; level < depth; level++ {
		var nextFrontier []string

		for p := range frontier {
			// Read file content for import parsing.
			absPath := filepath.Join(rootDir, filepath.FromSlash(p))
			data, err := os.ReadFile(absPath)
			if err != nil {
				continue
			}

			// Forward imports: files this file imports.
			imports := ParseImports(p, string(data))
			for _, imp := range imports {
				resolved := resolveImportPath(p, imp, allFileSet)
				for _, r := range resolved {
					if !changedSet[r] && !neighborSet[r] {
						neighborSet[r] = true
						nextFrontier = append(nextFrontier, r)
					}
				}
			}

			// Test file discovery: find test files for changed files.
			testFiles := findTestFiles(p, allFileSet)
			for _, tf := range testFiles {
				if !changedSet[tf] && !neighborSet[tf] {
					neighborSet[tf] = true
					nextFrontier = append(nextFrontier, tf)
				}
			}
		}

		// Reverse imports: find files in the repo that import any frontier file.
		for _, repoFile := range allFiles {
			if changedSet[repoFile] || neighborSet[repoFile] {
				continue
			}

			absPath := filepath.Join(rootDir, filepath.FromSlash(repoFile))
			data, err := os.ReadFile(absPath)
			if err != nil {
				continue
			}

			imports := ParseImports(repoFile, string(data))
			for _, imp := range imports {
				resolved := resolveImportPath(repoFile, imp, allFileSet)
				for _, r := range resolved {
					if frontier[r] {
						neighborSet[repoFile] = true
						nextFrontier = append(nextFrontier, repoFile)
						break
					}
				}
				if neighborSet[repoFile] {
					break
				}
			}
		}

		// Same-directory heuristic for files without import support.
		for p := range frontier {
			ext := strings.ToLower(filepath.Ext(p))
			// Only apply same-dir heuristic for unsupported languages.
			if ext == ".go" || ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".py" {
				continue
			}

			dir := filepath.Dir(p)
			for _, f := range allFiles {
				if filepath.Dir(f) == dir && !changedSet[f] && !neighborSet[f] {
					neighborSet[f] = true
					nextFrontier = append(nextFrontier, f)
				}
			}
		}

		// Prepare the next frontier.
		frontier = make(map[string]bool, len(nextFrontier))
		for _, f := range nextFrontier {
			frontier[f] = true
		}

		if len(frontier) == 0 {
			break
		}
	}

	// Convert to sorted slice.
	result := make([]string, 0, len(neighborSet))
	for p := range neighborSet {
		result = append(result, p)
	}
	sort.Strings(result)

	return result
}

// resolveImportPath attempts to resolve an import path to one or more concrete
// file paths that exist in the repository. It handles both relative imports
// (resolved against the importing file's directory) and package-level imports.
func resolveImportPath(importingFile, importPath string, allFiles map[string]bool) []string {
	var candidates []string

	// For relative imports, resolve against the importing file's directory.
	if strings.HasPrefix(importPath, "../") || strings.HasPrefix(importPath, "./") {
		dir := filepath.Dir(importingFile)
		resolved := filepath.ToSlash(filepath.Clean(filepath.Join(dir, importPath)))
		candidates = append(candidates, resolved)
	} else {
		// Absolute/package import: use as-is.
		candidates = append(candidates, importPath)
	}

	var results []string
	for _, candidate := range candidates {
		// Direct match.
		if allFiles[candidate] {
			results = append(results, candidate)
			continue
		}

		// Try common extensions.
		extensions := []string{".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs"}
		for _, ext := range extensions {
			withExt := candidate + ext
			if allFiles[withExt] {
				results = append(results, withExt)
			}
		}

		// Try as directory with index file.
		indexFiles := []string{
			candidate + "/index.ts",
			candidate + "/index.tsx",
			candidate + "/index.js",
			candidate + "/index.jsx",
			candidate + "/mod.rs",
			candidate + "/__init__.py",
		}
		for _, idx := range indexFiles {
			if allFiles[idx] {
				results = append(results, idx)
			}
		}

		// For Go package paths, find all .go files in the directory.
		if allFiles != nil {
			prefix := candidate + "/"
			for f := range allFiles {
				if strings.HasPrefix(f, prefix) && !strings.Contains(f[len(prefix):], "/") && strings.HasSuffix(f, ".go") {
					results = append(results, f)
				}
			}
		}
	}

	return results
}

// findTestFiles returns test file paths that correspond to a given source file.
// It uses language-specific conventions to find matching test files.
func findTestFiles(filePath string, allFiles map[string]bool) []string {
	ext := strings.ToLower(filepath.Ext(filePath))
	base := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	dir := filepath.Dir(filePath)

	var testPatterns []string

	switch ext {
	case ".go":
		// Go: foo.go -> foo_test.go
		testPatterns = append(testPatterns, base+"_test.go")
	case ".ts", ".tsx":
		// TypeScript: foo.ts -> foo.test.ts, foo.spec.ts, __tests__/foo.ts
		testPatterns = append(testPatterns,
			base+".test.ts",
			base+".spec.ts",
			base+".test.tsx",
			base+".spec.tsx",
			filepath.ToSlash(filepath.Join(dir, "__tests__", filepath.Base(filePath))),
		)
	case ".js", ".jsx":
		// JavaScript: foo.js -> foo.test.js, foo.spec.js, __tests__/foo.js
		testPatterns = append(testPatterns,
			base+".test.js",
			base+".spec.js",
			base+".test.jsx",
			base+".spec.jsx",
			filepath.ToSlash(filepath.Join(dir, "__tests__", filepath.Base(filePath))),
		)
	case ".py":
		// Python: foo.py -> test_foo.py, foo_test.py, tests/test_foo.py
		fileName := filepath.Base(base)
		testPatterns = append(testPatterns,
			filepath.ToSlash(filepath.Join(dir, "test_"+fileName+".py")),
			base+"_test.py",
			filepath.ToSlash(filepath.Join(dir, "tests", "test_"+fileName+".py")),
		)
	case ".rs":
		// Rust: files with mod tests inside are the tests themselves;
		// also check for tests/ directory.
		testPatterns = append(testPatterns,
			filepath.ToSlash(filepath.Join(dir, "tests", filepath.Base(filePath))),
		)
	}

	var results []string
	for _, pattern := range testPatterns {
		if allFiles[pattern] {
			results = append(results, pattern)
		}
	}

	return results
}

// languageFromPath returns the code fence language identifier for a file based
// on its extension. Returns an empty string for unrecognized extensions.
func languageFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc":
		return "cpp"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".md":
		return "markdown"
	case ".sh", ".bash":
		return "bash"
	case ".rb":
		return "ruby"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".sql":
		return "sql"
	case ".css":
		return "css"
	case ".html", ".htm":
		return "html"
	case ".xml":
		return "xml"
	default:
		return ""
	}
}
