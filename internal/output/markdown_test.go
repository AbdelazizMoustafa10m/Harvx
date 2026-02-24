package output

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/harvx/harvx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// fixedTimestamp is a deterministic timestamp used across all Markdown tests.
var fixedTimestamp = time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

// testRenderData returns a standard *RenderData populated with realistic values
// for reuse across tests. Callers may override individual fields as needed.
func testRenderData() *RenderData {
	return &RenderData{
		ProjectName:   "test-project",
		Timestamp:     fixedTimestamp,
		ContentHash:   "abc123def456",
		ProfileName:   "default",
		TokenizerName: "cl100k_base",
		TotalTokens:   2500,
		TotalFiles:    3,
		TreeString:    "src/\n  main.go\n  util.go\nREADME.md",
		ShowLineNumbers: false,
		TierCounts: map[int]int{
			0: 1,
			1: 1,
			2: 1,
		},
		TopFilesByTokens: []FileRenderEntry{
			{Path: "src/main.go", Size: 2048, TokenCount: 1200, Tier: 0, TierLabel: "critical"},
			{Path: "src/util.go", Size: 1024, TokenCount: 800, Tier: 1, TierLabel: "primary"},
			{Path: "README.md", Size: 512, TokenCount: 500, Tier: 2, TierLabel: "secondary"},
		},
		RedactionSummary: map[string]int{},
		TotalRedactions:  0,
		DiffSummary:      nil,
		Files: []FileRenderEntry{
			{
				Path:       "src/main.go",
				Size:       2048,
				TokenCount: 1200,
				Tier:       0,
				TierLabel:  "critical",
				Language:   "go",
				Content:    "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}",
			},
			{
				Path:       "src/util.go",
				Size:       1024,
				TokenCount: 800,
				Tier:       1,
				TierLabel:  "primary",
				Language:   "go",
				Content:    "package main\n\nfunc add(a, b int) int {\n\treturn a + b\n}",
			},
			{
				Path:       "README.md",
				Size:       512,
				TokenCount: 500,
				Tier:       2,
				TierLabel:  "secondary",
				Language:   "markdown",
				Content:    "# Test Project\n\nThis is a test project.",
			},
		},
	}
}

// renderToString is a convenience helper that renders data to a string.
func renderToString(t *testing.T, ctx context.Context, data *RenderData) string {
	t.Helper()
	var buf bytes.Buffer
	r := NewMarkdownRenderer()
	err := r.Render(ctx, &buf, data)
	require.NoError(t, err)
	return buf.String()
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_HeaderBlock
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_HeaderBlock(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	output := renderToString(t, context.Background(), data)

	// Project name in H1 title.
	assert.Contains(t, output, "# Harvx Context: test-project",
		"header should contain the project name in H1")

	// Timestamp formatted as RFC 3339.
	assert.Contains(t, output, "2026-01-15T10:30:00Z",
		"header should contain timestamp in RFC 3339 format")

	// Content hash.
	assert.Contains(t, output, "abc123def456",
		"header should contain the content hash")

	// Profile name.
	assert.Contains(t, output, "default",
		"header should contain the profile name")

	// Tokenizer name.
	assert.Contains(t, output, "cl100k_base",
		"header should contain the tokenizer name")

	// Total tokens (formatted with commas).
	assert.Contains(t, output, "2,500",
		"header should contain total tokens formatted with commas")

	// Total files.
	assert.Contains(t, output, "| Total Files | 3 |",
		"header should contain total file count")

	// Verify the metadata table structure.
	assert.Contains(t, output, "| Field | Value |",
		"header should contain metadata table header")
	assert.Contains(t, output, "|-------|-------|",
		"header should contain metadata table separator")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_FileSummary
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_FileSummary(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	output := renderToString(t, context.Background(), data)

	// Summary section header.
	assert.Contains(t, output, "## File Summary",
		"output should contain the file summary section")

	// Total counts line.
	assert.Contains(t, output, "**Total Files:** 3",
		"summary should show total files")
	assert.Contains(t, output, "**Total Tokens:** 2,500",
		"summary should show total tokens")

	// Tier breakdown.
	assert.Contains(t, output, "### Files by Tier",
		"summary should contain tier breakdown heading")
	assert.Contains(t, output, "| 0 | critical | 1 |",
		"tier 0 (critical) should show count 1")
	assert.Contains(t, output, "| 1 | primary | 1 |",
		"tier 1 (primary) should show count 1")
	assert.Contains(t, output, "| 2 | secondary | 1 |",
		"tier 2 (secondary) should show count 1")

	// Top files section.
	assert.Contains(t, output, "### Top Files by Token Count",
		"summary should contain top files heading")
	assert.Contains(t, output, "src/main.go",
		"top files should list main.go")
	assert.Contains(t, output, "1,200",
		"top files should show formatted token count for main.go")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_FileContents
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_FileContents(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	output := renderToString(t, context.Background(), data)

	// Files section header.
	assert.Contains(t, output, "## Files",
		"output should contain the files section")

	// Each file should have an H3 with backtick-quoted path.
	assert.Contains(t, output, "### `src/main.go`",
		"file should have H3 heading with backtick-quoted path")
	assert.Contains(t, output, "### `src/util.go`",
		"file should have H3 heading with backtick-quoted path")
	assert.Contains(t, output, "### `README.md`",
		"file should have H3 heading with backtick-quoted path")

	// Code fences should have correct language identifiers.
	assert.Contains(t, output, "```go",
		"Go files should have go language in code fence")
	assert.Contains(t, output, "```markdown",
		"Markdown files should have markdown language in code fence")

	// File content should appear.
	assert.Contains(t, output, "package main",
		"file content should be rendered")
	assert.Contains(t, output, "# Test Project",
		"README content should be rendered")

	// Metadata blockquote.
	assert.Contains(t, output, "> **Size:**",
		"each file should have a metadata blockquote")
	assert.Contains(t, output, "**Tokens:**",
		"metadata should include token count")
	assert.Contains(t, output, "**Tier:**",
		"metadata should include tier")
	assert.Contains(t, output, "**Compressed:**",
		"metadata should include compression status")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_FileContents_LanguageFromExt
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_FileContents_LanguageFromExt(t *testing.T) {
	t.Parallel()

	// When Language field is empty, fileLang should fall back to languageFromExt.
	data := testRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:       "app.py",
			Size:       256,
			TokenCount: 100,
			Tier:       1,
			TierLabel:  "primary",
			Language:   "", // empty -- should use extension
			Content:    "print('hello')",
		},
		{
			Path:       "server.rs",
			Size:       512,
			TokenCount: 200,
			Tier:       1,
			TierLabel:  "primary",
			Language:   "rust", // explicitly set
			Content:    "fn main() {}",
		},
	}
	data.TotalFiles = 2

	output := renderToString(t, context.Background(), data)

	assert.Contains(t, output, "```python",
		"Python file with empty Language should use extension-based language")
	assert.Contains(t, output, "```rust",
		"Rust file with explicit Language should use that language")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_LineNumbers
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_LineNumbers(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.ShowLineNumbers = true
	// Use a single file with known content for precise checking.
	data.Files = []FileRenderEntry{
		{
			Path:       "main.go",
			Size:       100,
			TokenCount: 50,
			Tier:       0,
			TierLabel:  "critical",
			Language:   "go",
			Content:    "package main\n\nfunc main() {}",
		},
	}
	data.TotalFiles = 1

	output := renderToString(t, context.Background(), data)

	// Line numbers should be present: "1 | package main"
	assert.Contains(t, output, "1 | package main",
		"line 1 should be numbered")
	assert.Contains(t, output, "2 | ",
		"line 2 (empty line) should be numbered")
	assert.Contains(t, output, "3 | func main() {}",
		"line 3 should be numbered")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_NoLineNumbers
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_NoLineNumbers(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.ShowLineNumbers = false
	data.Files = []FileRenderEntry{
		{
			Path:       "main.go",
			Size:       100,
			TokenCount: 50,
			Tier:       0,
			TierLabel:  "critical",
			Language:   "go",
			Content:    "package main\n\nfunc main() {}",
		},
	}
	data.TotalFiles = 1

	output := renderToString(t, context.Background(), data)

	// Should contain the raw content without line number prefixes.
	assert.Contains(t, output, "package main",
		"content should be present")
	assert.NotContains(t, output, "1 | package main",
		"line numbers should not be present when ShowLineNumbers is false")
	assert.NotContains(t, output, "3 | func main()",
		"line numbers should not be present when ShowLineNumbers is false")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_ChangeSummary
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_ChangeSummary(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.DiffSummary = &DiffSummaryData{
		AddedFiles:    []string{"new_file.go", "another_new.go"},
		ModifiedFiles: []string{"src/main.go"},
		DeletedFiles:  []string{"old_file.go"},
	}

	output := renderToString(t, context.Background(), data)

	// Change summary section.
	assert.Contains(t, output, "## Change Summary",
		"output should contain change summary section when DiffSummary is set")

	// Change type counts table.
	assert.Contains(t, output, "| Added | 2 |",
		"should show added count")
	assert.Contains(t, output, "| Modified | 1 |",
		"should show modified count")
	assert.Contains(t, output, "| Deleted | 1 |",
		"should show deleted count")

	// Added files list.
	assert.Contains(t, output, "### Added Files",
		"should list added files section")
	assert.Contains(t, output, "- new_file.go",
		"should list added file")
	assert.Contains(t, output, "- another_new.go",
		"should list added file")

	// Modified files list.
	assert.Contains(t, output, "### Modified Files",
		"should list modified files section")
	assert.Contains(t, output, "- src/main.go",
		"should list modified file")

	// Deleted files list.
	assert.Contains(t, output, "### Deleted Files",
		"should list deleted files section")
	assert.Contains(t, output, "- old_file.go",
		"should list deleted file")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_NoChangeSummary
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_NoChangeSummary(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.DiffSummary = nil

	output := renderToString(t, context.Background(), data)

	assert.NotContains(t, output, "## Change Summary",
		"change summary should not appear when DiffSummary is nil")
	assert.NotContains(t, output, "### Added Files",
		"added files section should not appear when DiffSummary is nil")
	assert.NotContains(t, output, "### Modified Files",
		"modified files section should not appear when DiffSummary is nil")
	assert.NotContains(t, output, "### Deleted Files",
		"deleted files section should not appear when DiffSummary is nil")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_EmptyFileList
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_EmptyFileList(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.Files = []FileRenderEntry{}
	data.TotalFiles = 0
	data.TotalTokens = 0
	data.TierCounts = map[int]int{}
	data.TopFilesByTokens = nil

	output := renderToString(t, context.Background(), data)

	// Should still produce valid output with all sections.
	assert.Contains(t, output, "# Harvx Context: test-project",
		"header should still render with no files")
	assert.Contains(t, output, "## File Summary",
		"file summary section should still render")
	assert.Contains(t, output, "**Total Files:** 0",
		"total files should show 0")
	assert.Contains(t, output, "## Files",
		"files section heading should still render")

	// Should NOT contain any H3 file entries.
	assert.Equal(t, 0, strings.Count(output, "### `"),
		"should have zero file headings")

	// Top files section should not appear.
	assert.NotContains(t, output, "### Top Files by Token Count",
		"top files section should not appear when empty")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_TripleBacktickEscape
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_TripleBacktickEscape(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:       "README.md",
			Size:       256,
			TokenCount: 100,
			Tier:       2,
			TierLabel:  "secondary",
			Language:   "markdown",
			Content:    "# Hello\n\n```go\nfunc main() {}\n```\n\nEnd.",
		},
	}
	data.TotalFiles = 1

	output := renderToString(t, context.Background(), data)

	// The triple backticks inside the content should be escaped to "`` `".
	assert.Contains(t, output, "`` `go",
		"inner triple backticks should be escaped")
	assert.Contains(t, output, "`` `\n",
		"closing inner triple backticks should be escaped")

	// The outer code fence (wrapping the file content) should still use "```".
	// Count outer triple backticks: there should be exactly 2 for the code fence
	// (opening and closing) per file, but the inner ones are escaped.
	// The file section's opening and closing fences are real ```.
	// Ensure the output does not have broken fences.
	assert.True(t, strings.Contains(output, "```markdown"),
		"outer code fence should have language identifier")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_LongFilePaths
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_LongFilePaths(t *testing.T) {
	t.Parallel()

	// Create a very long path (300+ characters).
	longPath := strings.Repeat("very/deep/nested/directory/", 12) + "file.go"

	data := testRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:       longPath,
			Size:       100,
			TokenCount: 50,
			Tier:       1,
			TierLabel:  "primary",
			Language:   "go",
			Content:    "package main",
		},
	}
	data.TotalFiles = 1

	output := renderToString(t, context.Background(), data)

	// The long path should appear in the H3 heading.
	assert.Contains(t, output, "### `"+longPath+"`",
		"long file path should appear in H3 heading without truncation")

	// Should still produce valid Markdown (no panic, no truncation).
	assert.Contains(t, output, "package main",
		"file content should still render with long path")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_ZeroTokenFiles
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_ZeroTokenFiles(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:       "empty.go",
			Size:       0,
			TokenCount: 0,
			Tier:       5,
			TierLabel:  "low",
			Language:   "go",
			Content:    "",
		},
		{
			Path:       "binary.dat",
			Size:       4096,
			TokenCount: 0,
			Tier:       5,
			TierLabel:  "low",
			Language:   "",
			Content:    "",
		},
	}
	data.TotalFiles = 2
	data.TotalTokens = 0

	output := renderToString(t, context.Background(), data)

	// Files with zero tokens should render without errors.
	assert.Contains(t, output, "### `empty.go`",
		"zero-token file should appear in output")
	assert.Contains(t, output, "### `binary.dat`",
		"zero-token binary file should appear in output")

	// Token count should show "0" in metadata.
	assert.Contains(t, output, "**Tokens:** 0",
		"zero tokens should render as 0")

	// Size should show "0 B" for zero-size file.
	assert.Contains(t, output, "**Size:** 0 B",
		"zero-size file should render as 0 B")

	// Size should show "4.0 KB" for the binary file.
	assert.Contains(t, output, "**Size:** 4.0 KB",
		"4096 bytes should render as 4.0 KB")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_NilData
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_NilData(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewMarkdownRenderer()
	err := r.Render(context.Background(), &buf, nil)

	require.Error(t, err, "nil data should return an error")
	assert.Contains(t, err.Error(), "nil",
		"error message should mention nil")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_CancelledContext
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	var buf bytes.Buffer
	r := NewMarkdownRenderer()
	data := testRenderData()

	err := r.Render(ctx, &buf, data)

	require.Error(t, err, "cancelled context should return an error")
	assert.ErrorIs(t, err, context.Canceled,
		"error should be context.Canceled")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_RedactionSummary
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_RedactionSummary(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.TotalRedactions = 5
	data.RedactionSummary = map[string]int{
		"AWS Key":      2,
		"GitHub Token": 3,
	}

	output := renderToString(t, context.Background(), data)

	// Redaction summary section.
	assert.Contains(t, output, "### Redaction Summary",
		"redaction summary should appear when TotalRedactions > 0")
	assert.Contains(t, output, "| AWS Key | 2 |",
		"AWS Key redaction should appear with count")
	assert.Contains(t, output, "| GitHub Token | 3 |",
		"GitHub Token redaction should appear with count")

	// Sorted keys: AWS Key before GitHub Token (alphabetical).
	awsIdx := strings.Index(output, "AWS Key")
	githubIdx := strings.Index(output, "GitHub Token")
	assert.True(t, awsIdx < githubIdx,
		"redaction types should appear in alphabetical order")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_RedactionSummary_Hidden
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_RedactionSummary_Hidden(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.TotalRedactions = 0
	data.RedactionSummary = map[string]int{}

	output := renderToString(t, context.Background(), data)

	assert.NotContains(t, output, "### Redaction Summary",
		"redaction summary should not appear when TotalRedactions is 0")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_FileWithError
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_FileWithError(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:       "broken.go",
			Size:       1024,
			TokenCount: 0,
			Tier:       1,
			TierLabel:  "primary",
			Language:   "go",
			Content:    "",
			Error:      "permission denied: broken.go",
		},
		{
			Path:       "ok.go",
			Size:       512,
			TokenCount: 100,
			Tier:       1,
			TierLabel:  "primary",
			Language:   "go",
			Content:    "package main",
		},
	}
	data.TotalFiles = 2

	output := renderToString(t, context.Background(), data)

	// Error file should show error message instead of code block.
	assert.Contains(t, output, "**Error:** permission denied: broken.go",
		"error file should display error message")

	// The errored file should NOT have a code fence for content.
	// Find the section for broken.go.
	brokenIdx := strings.Index(output, "### `broken.go`")
	okIdx := strings.Index(output, "### `ok.go`")
	require.True(t, brokenIdx >= 0, "broken.go heading should exist")
	require.True(t, okIdx >= 0, "ok.go heading should exist")

	brokenSection := output[brokenIdx:okIdx]
	assert.NotContains(t, brokenSection, "```go",
		"error file should not have a code fence")

	// The OK file should still render normally.
	okSection := output[okIdx:]
	assert.Contains(t, okSection, "```go",
		"non-error file should have a code fence")
	assert.Contains(t, okSection, "package main",
		"non-error file should have its content")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_Deterministic
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_Deterministic(t *testing.T) {
	t.Parallel()

	data := testRenderData()

	output1 := renderToString(t, context.Background(), data)
	output2 := renderToString(t, context.Background(), data)

	assert.Equal(t, output1, output2,
		"two renders of the same data must produce identical output")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_CompressedIndicator
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_CompressedIndicator(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:         "compressed.go",
			Size:         2048,
			TokenCount:   500,
			Tier:         1,
			TierLabel:    "primary",
			Language:     "go",
			Content:      "package main",
			IsCompressed: true,
		},
		{
			Path:         "uncompressed.go",
			Size:         1024,
			TokenCount:   250,
			Tier:         1,
			TierLabel:    "primary",
			Language:     "go",
			Content:      "package main",
			IsCompressed: false,
		},
	}
	data.TotalFiles = 2

	output := renderToString(t, context.Background(), data)

	// Find each file section and check the compressed indicator.
	compIdx := strings.Index(output, "### `compressed.go`")
	uncompIdx := strings.Index(output, "### `uncompressed.go`")
	require.True(t, compIdx >= 0)
	require.True(t, uncompIdx >= 0)

	compSection := output[compIdx:uncompIdx]
	assert.Contains(t, compSection, "**Compressed:** yes",
		"compressed file should show 'yes'")

	uncompSection := output[uncompIdx:]
	assert.Contains(t, uncompSection, "**Compressed:** no",
		"uncompressed file should show 'no'")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_DirectoryTreeSection
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_DirectoryTreeSection(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.TreeString = "project/\n  src/\n    main.go\n  README.md"

	output := renderToString(t, context.Background(), data)

	assert.Contains(t, output, "## Directory Tree",
		"output should contain directory tree section")
	assert.Contains(t, output, "project/\n  src/\n    main.go\n  README.md",
		"tree string should appear inside code block")

	// Tree should be inside a code block.
	treeIdx := strings.Index(output, "## Directory Tree")
	require.True(t, treeIdx >= 0)
	treeSection := output[treeIdx:]
	// Find the first ``` after the heading.
	assert.Contains(t, treeSection, "```\n",
		"directory tree should be enclosed in code fences")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_ChangeSummary_EmptyLists
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_ChangeSummary_EmptyLists(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.DiffSummary = &DiffSummaryData{
		AddedFiles:    []string{},
		ModifiedFiles: []string{"changed.go"},
		DeletedFiles:  []string{},
	}

	output := renderToString(t, context.Background(), data)

	// Change summary should appear.
	assert.Contains(t, output, "## Change Summary",
		"change summary should appear when DiffSummary is non-nil")

	// Counts should reflect the empty/non-empty lists.
	assert.Contains(t, output, "| Added | 0 |",
		"added count should be 0")
	assert.Contains(t, output, "| Modified | 1 |",
		"modified count should be 1")
	assert.Contains(t, output, "| Deleted | 0 |",
		"deleted count should be 0")

	// Empty list sections should not appear.
	assert.NotContains(t, output, "### Added Files",
		"added files section should not appear when list is empty")
	assert.NotContains(t, output, "### Deleted Files",
		"deleted files section should not appear when list is empty")

	// Non-empty list should appear.
	assert.Contains(t, output, "### Modified Files",
		"modified files section should appear")
	assert.Contains(t, output, "- changed.go",
		"modified file should be listed")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_TierFallback
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_TierFallback(t *testing.T) {
	t.Parallel()

	// When TierLabel is empty, the template should use the tierLabel function.
	data := testRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:       "main.go",
			Size:       100,
			TokenCount: 50,
			Tier:       0,
			TierLabel:  "", // empty -- should fall back to tierLabel(0) = "critical"
			Language:   "go",
			Content:    "package main",
		},
	}
	data.TotalFiles = 1

	output := renderToString(t, context.Background(), data)

	assert.Contains(t, output, "**Tier:** critical",
		"empty TierLabel should fall back to tierLabel function")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_SectionOrdering
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_SectionOrdering(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.DiffSummary = &DiffSummaryData{
		AddedFiles: []string{"new.go"},
	}

	output := renderToString(t, context.Background(), data)

	// Verify sections appear in the correct order.
	headerIdx := strings.Index(output, "# Harvx Context:")
	summaryIdx := strings.Index(output, "## File Summary")
	treeIdx := strings.Index(output, "## Directory Tree")
	filesIdx := strings.Index(output, "## Files\n")
	changeIdx := strings.Index(output, "## Change Summary")

	require.True(t, headerIdx >= 0, "header section must exist")
	require.True(t, summaryIdx >= 0, "summary section must exist")
	require.True(t, treeIdx >= 0, "tree section must exist")
	require.True(t, filesIdx >= 0, "files section must exist")
	require.True(t, changeIdx >= 0, "change summary section must exist")

	assert.True(t, headerIdx < summaryIdx,
		"header should come before summary")
	assert.True(t, summaryIdx < treeIdx,
		"summary should come before tree")
	assert.True(t, treeIdx < filesIdx,
		"tree should come before files")
	assert.True(t, filesIdx < changeIdx,
		"files should come before change summary")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_FileRedactionCount
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_FileRedactionCount(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:       "secrets.go",
			Size:       1024,
			TokenCount: 200,
			Tier:       1,
			TierLabel:  "primary",
			Language:   "go",
			Content:    "var key = \"[REDACTED]\"",
			Redactions: 3,
		},
	}
	data.TotalFiles = 1

	output := renderToString(t, context.Background(), data)

	// The file should render successfully with its content.
	assert.Contains(t, output, "### `secrets.go`",
		"redacted file should have a heading")
	assert.Contains(t, output, "[REDACTED]",
		"redacted content should be rendered")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_SpecialCharactersInProjectName
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_SpecialCharactersInProjectName(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.ProjectName = "my-project_v2.0 (beta)"

	output := renderToString(t, context.Background(), data)

	assert.Contains(t, output, "# Harvx Context: my-project_v2.0 (beta)",
		"special characters in project name should be preserved")
}

// ---------------------------------------------------------------------------
// TestMarkdownRenderer_MultipleFilesCount
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_MultipleFilesCount(t *testing.T) {
	t.Parallel()

	data := testRenderData()

	output := renderToString(t, context.Background(), data)

	// Count file headings -- should match number of files.
	headingCount := strings.Count(output, "### `")
	assert.Equal(t, len(data.Files), headingCount,
		"number of file headings should match number of files in data")
}

// ---------------------------------------------------------------------------
// Golden tests
// ---------------------------------------------------------------------------

func TestMarkdownRenderer_GoldenBasic(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	var buf bytes.Buffer
	r := NewMarkdownRenderer()
	err := r.Render(context.Background(), &buf, data)
	require.NoError(t, err)

	testutil.Golden(t, "markdown-basic", buf.Bytes())
}

func TestMarkdownRenderer_GoldenLineNumbers(t *testing.T) {
	t.Parallel()

	data := testRenderData()
	data.ShowLineNumbers = true

	var buf bytes.Buffer
	r := NewMarkdownRenderer()
	err := r.Render(context.Background(), &buf, data)
	require.NoError(t, err)

	testutil.Golden(t, "markdown-line-numbers", buf.Bytes())
}

// ---------------------------------------------------------------------------
// Benchmark tests
// ---------------------------------------------------------------------------

func BenchmarkMarkdownRenderer(b *testing.B) {
	data := testRenderData()
	r := NewMarkdownRenderer()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = r.Render(ctx, &buf, data)
	}
}

func BenchmarkMarkdownRenderer_LargeFileList(b *testing.B) {
	data := testRenderData()
	// Generate 100 files.
	files := make([]FileRenderEntry, 100)
	for i := range files {
		files[i] = FileRenderEntry{
			Path:       strings.Repeat("pkg/", 3) + "file_" + strings.Repeat("x", 5) + ".go",
			Size:       int64(1000 + i*100),
			TokenCount: 200 + i*10,
			Tier:       i % 5,
			TierLabel:  tierLabel(i % 5),
			Language:   "go",
			Content:    strings.Repeat("line of code\n", 50),
		}
	}
	data.Files = files
	data.TotalFiles = len(files)

	r := NewMarkdownRenderer()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = r.Render(ctx, &buf, data)
	}
}
