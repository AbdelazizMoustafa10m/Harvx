package workflows

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/harvx/harvx/internal/output"
)

// DefaultBriefMaxTokens is the default token budget for the Repo Brief
// artifact when no profile configuration overrides it.
const DefaultBriefMaxTokens = 4000

// BriefOptions configures the brief generation workflow.
type BriefOptions struct {
	// RootDir is the repository root directory to scan.
	RootDir string

	// MaxTokens is the token budget for the brief output. Zero uses
	// DefaultBriefMaxTokens.
	MaxTokens int

	// Target selects LLM-specific output optimizations (e.g., "claude" for XML).
	Target string

	// AssertInclude is a list of glob patterns that must each match at least
	// one included file in the brief. If any pattern matches zero files,
	// an error is returned.
	AssertInclude []string

	// TokenCounter counts tokens in text. If nil, a character-based estimator
	// is used (len/4).
	TokenCounter func(text string) int
}

// BriefSection represents one section of the Repo Brief output. Sections are
// rendered in fixed priority order. Lower priority numbers are rendered first
// and are the last to be truncated when the token budget is exceeded.
type BriefSection struct {
	// Name is the section heading (e.g., "README", "Build Commands").
	Name string

	// Content is the section body text.
	Content string

	// Priority controls both rendering order and truncation order.
	// Lower values are higher priority (rendered first, truncated last).
	Priority int

	// SourceFiles lists the relative paths of files that contributed to
	// this section's content.
	SourceFiles []string
}

// BriefResult holds the complete output of a brief generation run.
type BriefResult struct {
	// Content is the rendered brief document.
	Content string

	// ContentHash is the XXH3 64-bit hash of the content for caching.
	ContentHash uint64

	// FormattedHash is the hex-formatted content hash string.
	FormattedHash string

	// TokenCount is the number of tokens in the rendered content.
	TokenCount int

	// FilesIncluded lists all source files that contributed to the brief.
	FilesIncluded []string

	// Sections lists the sections that were included in the brief.
	Sections []BriefSection
}

// BriefJSON is the machine-readable metadata output for `harvx brief --json`.
type BriefJSON struct {
	// TokenCount is the number of tokens in the brief.
	TokenCount int `json:"token_count"`

	// ContentHash is the XXH3 content hash formatted as hex.
	ContentHash string `json:"content_hash"`

	// FilesIncluded lists the source files that contributed to the brief.
	FilesIncluded []string `json:"files_included"`

	// Sections lists the section names included in the brief.
	Sections []string `json:"sections"`

	// MaxTokens is the configured token budget.
	MaxTokens int `json:"max_tokens"`
}

// Section priority constants. Lower values are higher priority.
const (
	priorityReadme      = 1
	priorityInvariants  = 2
	priorityArchitecture = 3
	priorityBuildCmds   = 4
	priorityConfigInfo  = 5
	priorityReviewRules = 6
	priorityModuleMap   = 7
)

// readmePatterns lists candidate README file names in priority order.
var readmePatterns = []string{
	"README.md",
	"README.rst",
	"README.txt",
	"README",
	"readme.md",
}

// invariantPatterns lists project invariant/convention files.
var invariantPatterns = []string{
	"CLAUDE.md",
	"agents.md",
	"CONVENTIONS.md",
	"conventions.md",
}

// architecturePatterns lists architecture documentation paths.
var architecturePatterns = []string{
	"docs/architecture.md",
	"docs/ARCHITECTURE.md",
	"architecture.md",
	"ARCHITECTURE.md",
}

// adrGlobPatterns lists glob patterns for ADR files.
var adrGlobPatterns = []string{
	"ADR-*.md",
	"docs/adr/*.md",
	"docs/adr/**/*.md",
}

// buildFileExtractors maps build file names to their content extractors.
var buildFileExtractors = map[string]func(string) string{
	"Makefile":     ExtractMakefileTargets,
	"package.json": ExtractPackageJSONScripts,
}

// buildFileFullContent lists build files to include with full content.
var buildFileFullContent = []string{
	"Taskfile.yml",
	"Taskfile.yaml",
	"justfile",
	"Justfile",
}

// configFileExtractors maps config files to their content extractors.
var configFileExtractors = map[string]func(string) string{
	"go.mod":          ExtractGoModInfo,
	"Cargo.toml":      ExtractCargoTomlInfo,
	"pyproject.toml":  ExtractPyprojectInfo,
}

// reviewRulesGlob is the glob pattern for GitHub review rule files.
const reviewRulesGlob = ".github/review/**/*.md"

// GenerateBrief generates a Repo Brief artifact from the repository at the
// given root directory. The brief is a stable, deterministic document containing
// project-wide invariants suitable for LLM context injection.
//
// The output is deterministic: identical repository state produces identical
// content and content hash regardless of filesystem traversal order.
func GenerateBrief(opts BriefOptions) (*BriefResult, error) {
	if opts.RootDir == "" {
		return nil, fmt.Errorf("brief: root directory required")
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = DefaultBriefMaxTokens
	}

	countTokens := opts.TokenCounter
	if countTokens == nil {
		countTokens = estimateTokens
	}

	slog.Debug("generating brief",
		"root", opts.RootDir,
		"max_tokens", maxTokens,
		"target", opts.Target,
	)

	// Discover and build sections.
	var sections []BriefSection

	// 1. README
	if sec := discoverReadme(opts.RootDir); sec != nil {
		sections = append(sections, *sec)
	}

	// 2. Invariants (CLAUDE.md, agents.md, CONVENTIONS.md)
	if sec := discoverInvariants(opts.RootDir); sec != nil {
		sections = append(sections, *sec)
	}

	// 3. Architecture docs / ADRs
	if sec := discoverArchitecture(opts.RootDir); sec != nil {
		sections = append(sections, *sec)
	}

	// 4. Build commands
	if sec := discoverBuildCommands(opts.RootDir); sec != nil {
		sections = append(sections, *sec)
	}

	// 5. Config info
	if sec := discoverConfigInfo(opts.RootDir); sec != nil {
		sections = append(sections, *sec)
	}

	// 6. Review rules
	if sec := discoverReviewRules(opts.RootDir); sec != nil {
		sections = append(sections, *sec)
	}

	// 7. Module map
	if sec := discoverModuleMap(opts.RootDir); sec != nil {
		sections = append(sections, *sec)
	}

	// Sort sections by priority for deterministic rendering.
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Priority < sections[j].Priority
	})

	// Enforce token budget: truncate lower-priority sections first.
	sections = enforceBudget(sections, maxTokens, countTokens)

	// Check assert-include patterns.
	if len(opts.AssertInclude) > 0 {
		allFiles := collectSourceFiles(sections)
		if err := checkAssertIncludeBrief(opts.AssertInclude, allFiles); err != nil {
			return nil, err
		}
	}

	// Render the final document.
	content := renderBrief(sections, opts.Target)

	// Compute content hash.
	hashEntries := []output.FileHashEntry{
		{Path: "brief", Content: content},
	}
	hasher := output.NewContentHasher()
	contentHash, err := hasher.ComputeContentHash(hashEntries)
	if err != nil {
		return nil, fmt.Errorf("brief: computing content hash: %w", err)
	}

	formattedHash := output.FormatHash(contentHash)
	tokenCount := countTokens(content)

	// Prepend header with metadata.
	header := renderHeader(formattedHash, tokenCount, opts.Target)
	content = header + content

	// Recount tokens after adding header.
	tokenCount = countTokens(content)

	filesIncluded := collectSourceFiles(sections)

	slog.Info("brief generated",
		"token_count", tokenCount,
		"content_hash", formattedHash,
		"files_included", len(filesIncluded),
		"sections", len(sections),
	)

	return &BriefResult{
		Content:       content,
		ContentHash:   contentHash,
		FormattedHash: formattedHash,
		TokenCount:    tokenCount,
		FilesIncluded: filesIncluded,
		Sections:      sections,
	}, nil
}

// discoverReadme finds and reads the first available README file.
func discoverReadme(rootDir string) *BriefSection {
	for _, pattern := range readmePatterns {
		path := filepath.Join(rootDir, pattern)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		relPath := pattern
		slog.Debug("brief: found README", "path", relPath)

		return &BriefSection{
			Name:        "README",
			Content:     string(content),
			Priority:    priorityReadme,
			SourceFiles: []string{relPath},
		}
	}
	return nil
}

// discoverInvariants finds and reads project invariant/convention files.
func discoverInvariants(rootDir string) *BriefSection {
	var content strings.Builder
	var sourceFiles []string

	for _, pattern := range invariantPatterns {
		path := filepath.Join(rootDir, pattern)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		relPath := pattern
		slog.Debug("brief: found invariant file", "path", relPath)

		if content.Len() > 0 {
			content.WriteString("\n---\n\n")
		}
		content.WriteString(fmt.Sprintf("### %s\n\n", relPath))
		content.Write(data)
		sourceFiles = append(sourceFiles, relPath)
	}

	if content.Len() == 0 {
		return nil
	}

	return &BriefSection{
		Name:        "Key Invariants",
		Content:     content.String(),
		Priority:    priorityInvariants,
		SourceFiles: sourceFiles,
	}
}

// discoverArchitecture finds and reads architecture documentation and ADRs.
func discoverArchitecture(rootDir string) *BriefSection {
	var content strings.Builder
	var sourceFiles []string

	// Check fixed-path architecture docs.
	for _, pattern := range architecturePatterns {
		path := filepath.Join(rootDir, pattern)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		relPath := pattern
		slog.Debug("brief: found architecture doc", "path", relPath)

		if content.Len() > 0 {
			content.WriteString("\n---\n\n")
		}
		content.WriteString(fmt.Sprintf("### %s\n\n", relPath))
		content.Write(data)
		sourceFiles = append(sourceFiles, relPath)
	}

	// Check ADR glob patterns.
	for _, glob := range adrGlobPatterns {
		matches, err := doublestar.Glob(os.DirFS(rootDir), glob)
		if err != nil {
			slog.Debug("brief: ADR glob error", "pattern", glob, "error", err)
			continue
		}

		// Sort matches for deterministic output.
		sort.Strings(matches)

		for _, match := range matches {
			// Skip if already included.
			if containsString(sourceFiles, match) {
				continue
			}

			path := filepath.Join(rootDir, match)
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			slog.Debug("brief: found ADR", "path", match)

			if content.Len() > 0 {
				content.WriteString("\n---\n\n")
			}
			content.WriteString(fmt.Sprintf("### %s\n\n", match))
			content.Write(data)
			sourceFiles = append(sourceFiles, match)
		}
	}

	if content.Len() == 0 {
		return nil
	}

	return &BriefSection{
		Name:        "Architecture",
		Content:     content.String(),
		Priority:    priorityArchitecture,
		SourceFiles: sourceFiles,
	}
}

// discoverBuildCommands finds and extracts relevant sections from build files.
func discoverBuildCommands(rootDir string) *BriefSection {
	var content strings.Builder
	var sourceFiles []string

	// Files with section extraction. Process in sorted order for determinism.
	buildFiles := make([]string, 0, len(buildFileExtractors))
	for file := range buildFileExtractors {
		buildFiles = append(buildFiles, file)
	}
	sort.Strings(buildFiles)

	for _, file := range buildFiles {
		extractor := buildFileExtractors[file]
		path := filepath.Join(rootDir, file)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		extracted := extractor(string(data))
		if extracted == "" {
			continue
		}

		slog.Debug("brief: found build file", "path", file)

		if content.Len() > 0 {
			content.WriteString("\n")
		}
		content.WriteString(fmt.Sprintf("### %s\n\n", file))
		content.WriteString(extracted)
		sourceFiles = append(sourceFiles, file)
	}

	// Files included with full content.
	for _, file := range buildFileFullContent {
		path := filepath.Join(rootDir, file)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		slog.Debug("brief: found build file (full)", "path", file)

		if content.Len() > 0 {
			content.WriteString("\n")
		}
		content.WriteString(fmt.Sprintf("### %s\n\n", file))
		content.WriteString("```\n")
		content.Write(data)
		if !strings.HasSuffix(string(data), "\n") {
			content.WriteString("\n")
		}
		content.WriteString("```\n")
		sourceFiles = append(sourceFiles, file)
	}

	if content.Len() == 0 {
		return nil
	}

	// Sort source files for deterministic output.
	sort.Strings(sourceFiles)

	return &BriefSection{
		Name:        "Build Commands",
		Content:     content.String(),
		Priority:    priorityBuildCmds,
		SourceFiles: sourceFiles,
	}
}

// discoverConfigInfo finds and extracts info from project config files.
func discoverConfigInfo(rootDir string) *BriefSection {
	var content strings.Builder
	var sourceFiles []string

	// Process config files in sorted order for determinism.
	configFiles := make([]string, 0, len(configFileExtractors))
	for file := range configFileExtractors {
		configFiles = append(configFiles, file)
	}
	sort.Strings(configFiles)

	for _, file := range configFiles {
		extractor := configFileExtractors[file]
		path := filepath.Join(rootDir, file)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		extracted := extractor(string(data))
		if extracted == "" {
			continue
		}

		slog.Debug("brief: found config file", "path", file)

		if content.Len() > 0 {
			content.WriteString("\n")
		}
		content.WriteString(fmt.Sprintf("### %s\n\n", file))
		content.WriteString(extracted)
		sourceFiles = append(sourceFiles, file)
	}

	if content.Len() == 0 {
		return nil
	}

	return &BriefSection{
		Name:        "Project Config",
		Content:     content.String(),
		Priority:    priorityConfigInfo,
		SourceFiles: sourceFiles,
	}
}

// discoverReviewRules finds and reads GitHub review rule markdown files.
func discoverReviewRules(rootDir string) *BriefSection {
	matches, err := doublestar.Glob(os.DirFS(rootDir), reviewRulesGlob)
	if err != nil {
		slog.Debug("brief: review rules glob error", "error", err)
		return nil
	}

	if len(matches) == 0 {
		return nil
	}

	// Sort for deterministic output.
	sort.Strings(matches)

	var content strings.Builder
	var sourceFiles []string

	for _, match := range matches {
		path := filepath.Join(rootDir, match)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		slog.Debug("brief: found review rule", "path", match)

		if content.Len() > 0 {
			content.WriteString("\n---\n\n")
		}
		content.WriteString(fmt.Sprintf("### %s\n\n", match))
		content.Write(data)
		sourceFiles = append(sourceFiles, match)
	}

	if content.Len() == 0 {
		return nil
	}

	return &BriefSection{
		Name:        "Review Rules",
		Content:     content.String(),
		Priority:    priorityReviewRules,
		SourceFiles: sourceFiles,
	}
}

// discoverModuleMap generates a module map from top-level directories.
func discoverModuleMap(rootDir string) *BriefSection {
	entries, err := GenerateModuleMap(rootDir)
	if err != nil {
		slog.Debug("brief: module map generation error", "error", err)
		return nil
	}

	if len(entries) == 0 {
		return nil
	}

	content := RenderModuleMap(entries)

	return &BriefSection{
		Name:     "Module Map",
		Content:  content,
		Priority: priorityModuleMap,
	}
}

// enforceBudget truncates sections to fit within the token budget.
// Sections are sorted by priority (lower = higher priority). When the budget
// is exceeded, lower-priority sections (higher priority numbers) are truncated
// first by removing them entirely. If removing a section still doesn't fit,
// the next lowest-priority section is removed.
func enforceBudget(sections []BriefSection, maxTokens int, countTokens func(string) int) []BriefSection {
	// Calculate total tokens.
	totalTokens := 0
	for _, sec := range sections {
		totalTokens += countTokens(sec.Content)
	}

	// Add overhead for headers and formatting.
	overhead := len(sections) * 20 // Approximate overhead per section header
	totalTokens += overhead

	if totalTokens <= maxTokens {
		return sections
	}

	slog.Debug("brief: token budget exceeded, truncating",
		"total_tokens", totalTokens,
		"max_tokens", maxTokens,
	)

	// Remove sections from the end (lowest priority) until within budget.
	result := make([]BriefSection, len(sections))
	copy(result, sections)

	for len(result) > 0 {
		overhead = len(result) * 20
		totalTokens = overhead
		for _, sec := range result {
			totalTokens += countTokens(sec.Content)
		}

		if totalTokens <= maxTokens {
			break
		}

		removed := result[len(result)-1]
		slog.Debug("brief: removing section to fit budget",
			"section", removed.Name,
			"priority", removed.Priority,
		)
		result = result[:len(result)-1]
	}

	return result
}

// renderBrief renders all sections into the final document. When target is
// "claude", sections are wrapped in XML tags for optimal Claude parsing.
func renderBrief(sections []BriefSection, target string) string {
	if target == "claude" {
		return renderBriefXML(sections)
	}
	return renderBriefMarkdown(sections)
}

// renderBriefMarkdown renders sections as a Markdown document.
func renderBriefMarkdown(sections []BriefSection) string {
	var b strings.Builder

	for i, sec := range sections {
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		b.WriteString("## ")
		b.WriteString(sec.Name)
		b.WriteString("\n\n")
		b.WriteString(sec.Content)
		if !strings.HasSuffix(sec.Content, "\n") {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderBriefXML renders sections as XML for Claude-optimized consumption.
func renderBriefXML(sections []BriefSection) string {
	var b strings.Builder

	b.WriteString("<repo-brief>\n")
	for _, sec := range sections {
		tag := xmlTag(sec.Name)
		b.WriteString(fmt.Sprintf("<%s>\n", tag))
		b.WriteString(sec.Content)
		if !strings.HasSuffix(sec.Content, "\n") {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("</%s>\n", tag))
	}
	b.WriteString("</repo-brief>\n")

	return b.String()
}

// renderHeader creates the brief header with metadata.
func renderHeader(contentHash string, tokenCount int, target string) string {
	if target == "claude" {
		return fmt.Sprintf("<!-- Repo Brief | hash: %s | tokens: %d -->\n\n", contentHash, tokenCount)
	}
	return fmt.Sprintf("# Repo Brief\n\n> hash: `%s` | tokens: %d\n\n", contentHash, tokenCount)
}

// xmlTag converts a section name to a valid XML tag name.
func xmlTag(name string) string {
	tag := strings.ToLower(name)
	tag = strings.ReplaceAll(tag, " ", "-")
	return tag
}

// estimateTokens provides a simple character-based token estimate.
// Approximately 4 characters per token.
func estimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4
}

// collectSourceFiles gathers all unique source file paths from sections,
// sorted for deterministic output.
func collectSourceFiles(sections []BriefSection) []string {
	seen := make(map[string]bool)
	var files []string

	for _, sec := range sections {
		for _, f := range sec.SourceFiles {
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}

	sort.Strings(files)
	return files
}

// containsString reports whether s is in the slice.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// checkAssertIncludeBrief checks that each assert-include pattern matches at
// least one file in the brief's included files.
func checkAssertIncludeBrief(patterns, files []string) error {
	var failures []string

	for _, pattern := range patterns {
		matched := false
		for _, f := range files {
			ok, err := doublestar.Match(pattern, f)
			if err != nil {
				failures = append(failures, fmt.Sprintf("pattern %q: invalid glob: %s", pattern, err))
				matched = true
				break
			}
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			failures = append(failures, fmt.Sprintf("pattern %q matched 0 files in brief", pattern))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("brief assert-include failed: %s", strings.Join(failures, "; "))
	}
	return nil
}
