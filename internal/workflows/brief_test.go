package workflows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSampleRepo creates a temporary directory with typical repository files
// for testing brief generation.
func setupSampleRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// README.md
	writeFile(t, dir, "README.md", "# Test Project\n\nA sample project for testing.\n")

	// go.mod
	writeFile(t, dir, "go.mod", "module github.com/test/project\n\ngo 1.24.0\n")

	// Makefile
	writeFile(t, dir, "Makefile", `
.PHONY: build test

build:
	go build ./...

test:
	go test ./...
`)

	// CLAUDE.md
	writeFile(t, dir, "CLAUDE.md", "# Conventions\n\n- Use Go 1.24+\n- All tests must pass\n")

	// Create some directories.
	for _, d := range []string{"cmd", "internal", "docs"} {
		require.NoError(t, os.Mkdir(filepath.Join(dir, d), 0o755))
	}

	return dir
}

// writeFile is a helper that writes content to a file within a directory.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	// Create parent directories if needed.
	parent := filepath.Dir(path)
	require.NoError(t, os.MkdirAll(parent, 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestGenerateBrief_IncludesAllSections(t *testing.T) {
	t.Parallel()
	dir := setupSampleRepo(t)

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.Content)
	assert.Greater(t, result.TokenCount, 0)
	assert.NotZero(t, result.ContentHash)
	assert.NotEmpty(t, result.FormattedHash)
	assert.NotEmpty(t, result.FilesIncluded)

	// Check that expected sections are present.
	sectionNames := make([]string, 0, len(result.Sections))
	for _, sec := range result.Sections {
		sectionNames = append(sectionNames, sec.Name)
	}

	assert.Contains(t, sectionNames, "README")
	assert.Contains(t, sectionNames, "Key Invariants")
	assert.Contains(t, sectionNames, "Build Commands")
	assert.Contains(t, sectionNames, "Project Config")
	assert.Contains(t, sectionNames, "Module Map")
}

func TestGenerateBrief_Deterministic(t *testing.T) {
	t.Parallel()
	dir := setupSampleRepo(t)

	opts := BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	}

	result1, err := GenerateBrief(opts)
	require.NoError(t, err)

	result2, err := GenerateBrief(opts)
	require.NoError(t, err)

	assert.Equal(t, result1.ContentHash, result2.ContentHash,
		"identical repo state must produce identical content hash")
	assert.Equal(t, result1.Content, result2.Content,
		"identical repo state must produce identical content")
	assert.Equal(t, result1.TokenCount, result2.TokenCount,
		"identical repo state must produce identical token count")
}

func TestGenerateBrief_MissingReadme(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Only create a Makefile, no README.
	writeFile(t, dir, "Makefile", "build:\n\tgo build ./...\n")

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should not have README section.
	for _, sec := range result.Sections {
		assert.NotEqual(t, "README", sec.Name, "missing README should be skipped")
	}
}

func TestGenerateBrief_EmptyRepository(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should produce a minimal brief with just the header.
	assert.NotEmpty(t, result.Content)
	assert.NotZero(t, result.ContentHash)
}

func TestGenerateBrief_TokenBudgetTruncation(t *testing.T) {
	t.Parallel()
	dir := setupSampleRepo(t)

	// Use a very small token budget to force truncation.
	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 50,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// With very small budget, some sections should be truncated.
	// The highest-priority sections (README) should survive longest.
	sectionNames := make(map[string]bool)
	for _, sec := range result.Sections {
		sectionNames[sec.Name] = true
	}

	// If Module Map was removed, that's expected (lowest priority).
	// The exact count depends on content sizes.
	assert.Less(t, len(result.Sections), 7,
		"token budget should cause some sections to be removed")
}

func TestGenerateBrief_ClaudeTarget(t *testing.T) {
	t.Parallel()
	dir := setupSampleRepo(t)

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
		Target:    "claude",
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "<repo-brief>")
	assert.Contains(t, result.Content, "</repo-brief>")
	assert.Contains(t, result.Content, "<!-- Repo Brief")
}

func TestGenerateBrief_MarkdownTarget(t *testing.T) {
	t.Parallel()
	dir := setupSampleRepo(t)

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "# Repo Brief")
	assert.Contains(t, result.Content, "## README")
}

func TestGenerateBrief_RootDirRequired(t *testing.T) {
	t.Parallel()

	_, err := GenerateBrief(BriefOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "root directory required")
}

func TestGenerateBrief_AssertIncludeSuccess(t *testing.T) {
	t.Parallel()
	dir := setupSampleRepo(t)

	result, err := GenerateBrief(BriefOptions{
		RootDir:       dir,
		MaxTokens:     10000,
		AssertInclude: []string{"README.md"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGenerateBrief_AssertIncludeFailure(t *testing.T) {
	t.Parallel()
	dir := setupSampleRepo(t)

	_, err := GenerateBrief(BriefOptions{
		RootDir:       dir,
		MaxTokens:     10000,
		AssertInclude: []string{"nonexistent-file.xyz"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "assert-include failed")
}

func TestGenerateBrief_WithPackageJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	writeFile(t, dir, "package.json", `{
		"name": "test-project",
		"scripts": {
			"build": "tsc",
			"test": "jest"
		}
	}`)

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "Build Commands")
	assert.Contains(t, result.Content, "test-project")
}

func TestGenerateBrief_WithArchitectureDocs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	writeFile(t, dir, "docs/architecture.md", "# Architecture\n\nMicroservices pattern.\n")

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	sectionNames := make([]string, 0, len(result.Sections))
	for _, sec := range result.Sections {
		sectionNames = append(sectionNames, sec.Name)
	}
	assert.Contains(t, sectionNames, "Architecture")
}

func TestGenerateBrief_WithADRs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	writeFile(t, dir, "docs/adr/001-use-go.md", "# ADR-001: Use Go\n\nDecided to use Go.\n")
	writeFile(t, dir, "docs/adr/002-use-cobra.md", "# ADR-002: Use Cobra\n\nDecided to use Cobra.\n")

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "ADR-001")
	assert.Contains(t, result.Content, "ADR-002")
}

func TestGenerateBrief_WithReviewRules(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	writeFile(t, dir, ".github/review/security.md", "# Security Review Rules\n\nCheck for secrets.\n")

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	sectionNames := make([]string, 0, len(result.Sections))
	for _, sec := range result.Sections {
		sectionNames = append(sectionNames, sec.Name)
	}
	assert.Contains(t, sectionNames, "Review Rules")
}

func TestGenerateBrief_DefaultMaxTokens(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	writeFile(t, dir, "README.md", "# Test\n")

	result, err := GenerateBrief(BriefOptions{
		RootDir: dir,
		// MaxTokens is 0 -- should use DefaultBriefMaxTokens.
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGenerateBrief_FilesIncludedSorted(t *testing.T) {
	t.Parallel()
	dir := setupSampleRepo(t)

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	// Verify files are sorted.
	for i := 1; i < len(result.FilesIncluded); i++ {
		assert.True(t, result.FilesIncluded[i-1] <= result.FilesIncluded[i],
			"files_included must be sorted: %q should come before %q",
			result.FilesIncluded[i-1], result.FilesIncluded[i])
	}
}

func TestGenerateBrief_CustomTokenCounter(t *testing.T) {
	t.Parallel()
	dir := setupSampleRepo(t)

	calls := 0
	counter := func(text string) int {
		calls++
		return len(text) / 4
	}

	result, err := GenerateBrief(BriefOptions{
		RootDir:      dir,
		MaxTokens:    10000,
		TokenCounter: counter,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, calls, 0, "custom token counter should be called")
}

func TestEnforceBudget_NoTruncation(t *testing.T) {
	t.Parallel()

	sections := []BriefSection{
		{Name: "A", Content: "short", Priority: 1},
		{Name: "B", Content: "also short", Priority: 2},
	}

	result := enforceBudget(sections, 10000, estimateTokens)
	assert.Len(t, result, 2, "no sections should be removed when budget is large")
}

func TestEnforceBudget_TruncatesLowPriority(t *testing.T) {
	t.Parallel()

	sections := []BriefSection{
		{Name: "High", Content: "important content", Priority: 1},
		{Name: "Low", Content: "less important content that is much longer than the high section", Priority: 7},
	}

	// "High" content: 17 chars ~5 tokens + 20 overhead = 25 for one section.
	// Both: ~21 tokens content + 40 overhead = 61. Budget of 30 fits one section.
	result := enforceBudget(sections, 30, estimateTokens)

	require.Len(t, result, 1, "low-priority section should be removed")
	assert.Equal(t, "High", result[0].Name)
}

func TestEstimateTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want int
	}{
		{name: "empty", text: "", want: 0},
		{name: "short", text: "hi", want: 1},
		{name: "four chars", text: "abcd", want: 1},
		{name: "five chars", text: "abcde", want: 2},
		{name: "eight chars", text: "abcdefgh", want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := estimateTokens(tt.text)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCollectSourceFiles_Deduplicates(t *testing.T) {
	t.Parallel()

	sections := []BriefSection{
		{SourceFiles: []string{"README.md", "go.mod"}},
		{SourceFiles: []string{"go.mod", "Makefile"}},
	}

	files := collectSourceFiles(sections)
	assert.Equal(t, []string{"Makefile", "README.md", "go.mod"}, files)
}

func TestXmlTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
	}{
		{name: "README", want: "readme"},
		{name: "Build Commands", want: "build-commands"},
		{name: "Module Map", want: "module-map"},
		{name: "Key Invariants", want: "key-invariants"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, xmlTag(tt.name))
		})
	}
}

func TestRenderHeader_Markdown(t *testing.T) {
	t.Parallel()

	header := renderHeader("abcdef0123456789", 500, "generic")
	assert.Contains(t, header, "# Repo Brief")
	assert.Contains(t, header, "abcdef0123456789")
	assert.Contains(t, header, "500")
}

func TestRenderHeader_Claude(t *testing.T) {
	t.Parallel()

	header := renderHeader("abcdef0123456789", 500, "claude")
	assert.Contains(t, header, "<!-- Repo Brief")
	assert.Contains(t, header, "abcdef0123456789")
	assert.Contains(t, header, "500")
}

// ---------------------------------------------------------------------------
// Additional tests for T-070 acceptance criteria coverage
// ---------------------------------------------------------------------------

// TestGenerateBrief_MissingArchitectureDocs verifies that when no architecture
// documentation exists in the repository, the Architecture section is simply
// omitted without error. This covers the acceptance criterion for graceful
// handling of missing architecture docs.
func TestGenerateBrief_MissingArchitectureDocs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create a repo with only a README, no architecture docs or ADRs.
	writeFile(t, dir, "README.md", "# Project\n\nA simple project.\n")

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Architecture section should not be present.
	for _, sec := range result.Sections {
		assert.NotEqual(t, "Architecture", sec.Name,
			"missing architecture docs should be skipped without error")
	}

	// Brief should still succeed with remaining sections.
	assert.NotEmpty(t, result.Content)
	assert.NotZero(t, result.ContentHash)
}

// TestGenerateBrief_MissingAllOptionalSections verifies that a repository
// with none of the optional files (no CLAUDE.md, no Makefile, no go.mod,
// no docs/, no .github/review/) produces a valid brief with only the
// sections that have content.
func TestGenerateBrief_MissingAllOptionalSections(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Only a README exists.
	writeFile(t, dir, "README.md", "# Minimal Project\n")

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	sectionNames := make([]string, 0, len(result.Sections))
	for _, sec := range result.Sections {
		sectionNames = append(sectionNames, sec.Name)
	}

	assert.Contains(t, sectionNames, "README")
	assert.NotContains(t, sectionNames, "Key Invariants")
	assert.NotContains(t, sectionNames, "Architecture")
	assert.NotContains(t, sectionNames, "Build Commands")
	assert.NotContains(t, sectionNames, "Project Config")
	assert.NotContains(t, sectionNames, "Review Rules")
}

// TestGenerateBrief_LargeREADMETruncation verifies that when a very large
// README is present and the token budget is small, lower-priority sections
// are removed first. With a sufficiently large budget, the README survives
// while lower-priority sections are dropped.
func TestGenerateBrief_LargeREADMETruncation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create a moderately large README (~400 chars = ~100 tokens with 4 chars/token).
	largeContent := "# Large Project\n\n" + strings.Repeat("Description. ", 30)
	writeFile(t, dir, "README.md", largeContent)

	// Also create a Makefile and go.mod to have multiple sections.
	writeFile(t, dir, "Makefile", "build:\n\tgo build ./...\n\ntest:\n\tgo test ./...\n")
	writeFile(t, dir, "go.mod", "module example.com/large\n\ngo 1.24.0\n")

	// Create directories for the module map.
	for _, d := range []string{"cmd", "internal", "docs"} {
		require.NoError(t, os.Mkdir(filepath.Join(dir, d), 0o755))
	}

	// Use a budget that can hold README + Build Commands but not everything.
	// README ~100 tokens + Build ~30 tokens + overhead = ~190.
	// Config ~20 tokens + Module Map ~40 tokens would push it over.
	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 200,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	sectionNames := make(map[string]bool)
	for _, sec := range result.Sections {
		sectionNames[sec.Name] = true
	}

	// Lower-priority sections should be removed before higher-priority ones.
	// README (priority 1) should survive while Module Map (priority 7) is dropped.
	assert.True(t, sectionNames["README"],
		"README (priority 1) should survive token budget enforcement")
	assert.Less(t, len(result.Sections), 4,
		"tight budget should cause some sections to be removed")
}

// TestGenerateBrief_VeryLargeREADMEExceedsBudget verifies that when a
// README alone exceeds the budget, all sections are removed gracefully.
func TestGenerateBrief_VeryLargeREADMEExceedsBudget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create an extremely large README (~10K chars = ~2500 tokens).
	largeContent := "# Huge Project\n\n" + strings.Repeat("This is a verbose description of the project. ", 200)
	writeFile(t, dir, "README.md", largeContent)

	// Budget is only 50 tokens -- even the README alone exceeds this.
	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 50,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// The brief should still succeed, but sections may be empty if the
	// single README section exceeds the budget.
	assert.NotEmpty(t, result.Content, "brief should still produce output with header")
	assert.NotZero(t, result.ContentHash)
}

// TestGenerateBrief_Deterministic_ContentStringEquality strengthens the
// determinism test by verifying that the actual rendered content strings
// are byte-for-byte identical, not just that their hashes match.
func TestGenerateBrief_Deterministic_ContentStringEquality(t *testing.T) {
	t.Parallel()
	dir := setupSampleRepo(t)

	opts := BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	}

	// Run 5 times to increase confidence in determinism.
	results := make([]*BriefResult, 5)
	for i := 0; i < 5; i++ {
		var err error
		results[i], err = GenerateBrief(opts)
		require.NoError(t, err)
	}

	for i := 1; i < len(results); i++ {
		assert.Equal(t, results[0].Content, results[i].Content,
			"run %d content must be identical to run 0", i)
		assert.Equal(t, results[0].ContentHash, results[i].ContentHash,
			"run %d content hash must be identical to run 0", i)
		assert.Equal(t, results[0].FormattedHash, results[i].FormattedHash,
			"run %d formatted hash must be identical to run 0", i)
		assert.Equal(t, results[0].TokenCount, results[i].TokenCount,
			"run %d token count must be identical to run 0", i)
		assert.Equal(t, results[0].FilesIncluded, results[i].FilesIncluded,
			"run %d files included must be identical to run 0", i)
	}
}

// TestGenerateBrief_SectionPriorityOrder verifies that sections are rendered
// in fixed priority order (lower priority number = higher priority).
func TestGenerateBrief_SectionPriorityOrder(t *testing.T) {
	t.Parallel()
	dir := setupSampleRepo(t)

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	// Verify sections are in ascending priority order.
	for i := 1; i < len(result.Sections); i++ {
		assert.LessOrEqual(t, result.Sections[i-1].Priority, result.Sections[i].Priority,
			"section %q (priority %d) should come before %q (priority %d)",
			result.Sections[i-1].Name, result.Sections[i-1].Priority,
			result.Sections[i].Name, result.Sections[i].Priority)
	}
}

// TestEnforceBudget_RemovesAllSections verifies that when the budget is
// extremely small (e.g., 1 token), all sections are removed gracefully
// without panicking.
func TestEnforceBudget_RemovesAllSections(t *testing.T) {
	t.Parallel()

	sections := []BriefSection{
		{Name: "A", Content: strings.Repeat("word ", 100), Priority: 1},
		{Name: "B", Content: strings.Repeat("word ", 100), Priority: 2},
	}

	result := enforceBudget(sections, 1, estimateTokens)
	assert.Empty(t, result, "budget of 1 token should remove all sections")
}

// TestEnforceBudget_PreservesHighPriority verifies that the budget enforcement
// always removes the highest-numbered priority first.
func TestEnforceBudget_PreservesHighPriority(t *testing.T) {
	t.Parallel()

	sections := []BriefSection{
		{Name: "README", Content: "short", Priority: 1},
		{Name: "Invariants", Content: "short", Priority: 2},
		{Name: "Architecture", Content: "short", Priority: 3},
		{Name: "Build", Content: "short", Priority: 4},
		{Name: "Config", Content: "short", Priority: 5},
		{Name: "Review", Content: "short", Priority: 6},
		{Name: "ModuleMap", Content: strings.Repeat("x", 400), Priority: 7},
	}

	// Budget that can hold ~6 sections but not 7.
	result := enforceBudget(sections, 80, estimateTokens)

	// ModuleMap should be removed first.
	names := make([]string, 0, len(result))
	for _, s := range result {
		names = append(names, s.Name)
	}
	assert.NotContains(t, names, "ModuleMap",
		"lowest-priority section (ModuleMap) should be removed first")
	assert.Contains(t, names, "README",
		"highest-priority section (README) should be preserved")
}

// TestGenerateBrief_ReadmeVariants verifies that different README file names
// are found (README.md, README.rst, README.txt, README, readme.md).
func TestGenerateBrief_ReadmeVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
	}{
		{name: "README.md", filename: "README.md"},
		{name: "README.rst", filename: "README.rst"},
		{name: "README.txt", filename: "README.txt"},
		{name: "README (no ext)", filename: "README"},
		{name: "readme.md (lowercase)", filename: "readme.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()

			writeFile(t, dir, tt.filename, "# My Project\n")

			result, err := GenerateBrief(BriefOptions{
				RootDir:   dir,
				MaxTokens: 10000,
			})
			require.NoError(t, err)

			sectionNames := make([]string, 0)
			for _, sec := range result.Sections {
				sectionNames = append(sectionNames, sec.Name)
			}
			assert.Contains(t, sectionNames, "README",
				"%s should be detected as README", tt.filename)
		})
	}
}

// TestGenerateBrief_ReadmePriority verifies that README.md takes precedence
// over README.rst when both exist.
func TestGenerateBrief_ReadmePriority(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	writeFile(t, dir, "README.md", "# Markdown README\n")
	writeFile(t, dir, "README.rst", "RST README\n=========\n")

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	// Should pick README.md first.
	for _, sec := range result.Sections {
		if sec.Name == "README" {
			assert.Contains(t, sec.Content, "Markdown README",
				"README.md should take priority over README.rst")
			assert.Equal(t, []string{"README.md"}, sec.SourceFiles)
			break
		}
	}
}

// TestContainsString verifies the containsString helper.
func TestContainsString(t *testing.T) {
	t.Parallel()

	assert.True(t, containsString([]string{"a", "b", "c"}, "b"))
	assert.False(t, containsString([]string{"a", "b", "c"}, "d"))
	assert.False(t, containsString(nil, "a"))
	assert.False(t, containsString([]string{}, "a"))
}

// TestCheckAssertIncludeBrief_PatternMatching verifies that assert-include
// patterns correctly match against the files list.
func TestCheckAssertIncludeBrief_PatternMatching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		patterns []string
		files    []string
		wantErr  bool
	}{
		{
			name:     "exact match",
			patterns: []string{"README.md"},
			files:    []string{"README.md", "go.mod"},
			wantErr:  false,
		},
		{
			name:     "glob match",
			patterns: []string{"*.md"},
			files:    []string{"README.md", "go.mod"},
			wantErr:  false,
		},
		{
			name:     "no match",
			patterns: []string{"nonexistent.xyz"},
			files:    []string{"README.md"},
			wantErr:  true,
		},
		{
			name:     "multiple patterns all match",
			patterns: []string{"README.md", "go.mod"},
			files:    []string{"README.md", "go.mod", "Makefile"},
			wantErr:  false,
		},
		{
			name:     "one pattern fails",
			patterns: []string{"README.md", "missing.txt"},
			files:    []string{"README.md"},
			wantErr:  true,
		},
		{
			name:     "empty patterns",
			patterns: []string{},
			files:    []string{"README.md"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkAssertIncludeBrief(tt.patterns, tt.files)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRenderBriefMarkdown_MultipleSection verifies that markdown rendering
// includes separators between sections and proper heading levels.
func TestRenderBriefMarkdown_MultipleSections(t *testing.T) {
	t.Parallel()

	sections := []BriefSection{
		{Name: "README", Content: "Hello world\n", Priority: 1},
		{Name: "Build Commands", Content: "make build\n", Priority: 4},
	}

	output := renderBrief(sections, "generic")

	assert.Contains(t, output, "## README")
	assert.Contains(t, output, "## Build Commands")
	assert.Contains(t, output, "---")
	assert.Contains(t, output, "Hello world")
	assert.Contains(t, output, "make build")
}

// TestRenderBriefXML_MultipleSections verifies that XML rendering wraps
// sections in appropriate XML tags.
func TestRenderBriefXML_MultipleSections(t *testing.T) {
	t.Parallel()

	sections := []BriefSection{
		{Name: "README", Content: "Hello world\n", Priority: 1},
		{Name: "Build Commands", Content: "make build\n", Priority: 4},
	}

	output := renderBrief(sections, "claude")

	assert.Contains(t, output, "<repo-brief>")
	assert.Contains(t, output, "</repo-brief>")
	assert.Contains(t, output, "<readme>")
	assert.Contains(t, output, "</readme>")
	assert.Contains(t, output, "<build-commands>")
	assert.Contains(t, output, "</build-commands>")
	assert.Contains(t, output, "Hello world")
	assert.Contains(t, output, "make build")
}

// TestGenerateBrief_WithTaskfile verifies that a Taskfile.yml is included
// in the Build Commands section with full content.
func TestGenerateBrief_WithTaskfile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	writeFile(t, dir, "Taskfile.yml", "version: '3'\ntasks:\n  build:\n    cmds:\n      - go build ./...\n")

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "Build Commands")
	assert.Contains(t, result.Content, "Taskfile.yml")
	assert.Contains(t, result.Content, "go build")
}

// TestGenerateBrief_WithJustfile verifies that a justfile is included
// in the Build Commands section with full content.
func TestGenerateBrief_WithJustfile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	writeFile(t, dir, "justfile", "build:\n    go build ./...\n\ntest:\n    go test ./...\n")

	result, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "Build Commands")
	assert.Contains(t, result.Content, "justfile")
}

// TestGenerateBrief_ContentHashChangesWithContent verifies that modifying
// the repository content produces a different content hash.
func TestGenerateBrief_ContentHashChangesWithContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	writeFile(t, dir, "README.md", "# Version 1\n")

	result1, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	// Modify the README.
	writeFile(t, dir, "README.md", "# Version 2\n")

	result2, err := GenerateBrief(BriefOptions{
		RootDir:   dir,
		MaxTokens: 10000,
	})
	require.NoError(t, err)

	assert.NotEqual(t, result1.ContentHash, result2.ContentHash,
		"different content must produce different content hash")
	assert.NotEqual(t, result1.Content, result2.Content,
		"different content must produce different output")
}
