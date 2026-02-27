package output

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
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

// xmlFixedTimestamp is a deterministic timestamp used across all XML tests.
var xmlFixedTimestamp = time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

// xmlTestRenderData returns a standard *RenderData populated with realistic
// values for reuse across XML tests. Callers may override individual fields.
func xmlTestRenderData() *RenderData {
	return &RenderData{
		ProjectName:   "test-project",
		Timestamp:     xmlFixedTimestamp,
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

// xmlRenderToString renders data using the XMLRenderer and returns the output.
func xmlRenderToString(t *testing.T, ctx context.Context, data *RenderData) string {
	t.Helper()
	var buf bytes.Buffer
	r := NewXMLRenderer()
	err := r.Render(ctx, &buf, data)
	require.NoError(t, err)
	return buf.String()
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_WellFormedXML
// ---------------------------------------------------------------------------

func TestXMLRenderer_WellFormedXML(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	output := xmlRenderToString(t, context.Background(), data)

	assertWellFormedXML(t, output)
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_XMLDeclaration
// ---------------------------------------------------------------------------

func TestXMLRenderer_XMLDeclaration(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	output := xmlRenderToString(t, context.Background(), data)

	assert.True(t, strings.HasPrefix(output, `<?xml version="1.0" encoding="UTF-8"?>`),
		"output should start with XML declaration")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_RootElement
// ---------------------------------------------------------------------------

func TestXMLRenderer_RootElement(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, "<repository>",
		"output should contain opening repository tag")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(output), "</repository>"),
		"output should end with closing repository tag")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_MetadataSection
// ---------------------------------------------------------------------------

func TestXMLRenderer_MetadataSection(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, "<metadata>",
		"output should contain metadata section")
	assert.Contains(t, output, "<project_name>test-project</project_name>",
		"metadata should contain project name")
	assert.Contains(t, output, "<generated>2026-01-15T10:30:00Z</generated>",
		"metadata should contain RFC 3339 timestamp")
	assert.Contains(t, output, "<content_hash>abc123def456</content_hash>",
		"metadata should contain content hash")
	assert.Contains(t, output, "<profile>default</profile>",
		"metadata should contain profile name")
	assert.Contains(t, output, "<tokenizer>cl100k_base</tokenizer>",
		"metadata should contain tokenizer name")
	assert.Contains(t, output, "<total_tokens>2500</total_tokens>",
		"metadata should contain raw total tokens")
	assert.Contains(t, output, "<total_files>3</total_files>",
		"metadata should contain total files")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_FileSummary
// ---------------------------------------------------------------------------

func TestXMLRenderer_FileSummary(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, "<file_summary>",
		"output should contain file_summary section")

	// Summary uses formatted numbers.
	assert.Contains(t, output, "<total_tokens>2,500</total_tokens>",
		"file_summary total_tokens should be formatted with commas")

	// Tier breakdown.
	assert.Contains(t, output, `<tier number="0" label="critical" count="1"/>`,
		"tier 0 should appear in files_by_tier")
	assert.Contains(t, output, `<tier number="1" label="primary" count="1"/>`,
		"tier 1 should appear in files_by_tier")
	assert.Contains(t, output, `<tier number="2" label="secondary" count="1"/>`,
		"tier 2 should appear in files_by_tier")

	// Top files.
	assert.Contains(t, output, "<top_files>",
		"output should contain top_files section")
	assert.Contains(t, output, `path="src/main.go"`,
		"top files should list main.go")
	assert.Contains(t, output, `tokens="1,200"`,
		"top files should show formatted token count")
	assert.Contains(t, output, `size="2.0 KB"`,
		"top files should show formatted size")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_FileContentCDATA
// ---------------------------------------------------------------------------

func TestXMLRenderer_FileContentCDATA(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	output := xmlRenderToString(t, context.Background(), data)

	// File content should be wrapped in CDATA.
	assert.Contains(t, output, "<![CDATA[package main",
		"file content should be wrapped in CDATA section")
	assert.Contains(t, output, "]]></content>",
		"CDATA section should end before closing content tag")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_FileAttributes
// ---------------------------------------------------------------------------

func TestXMLRenderer_FileAttributes(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	output := xmlRenderToString(t, context.Background(), data)

	// Check first file's attributes.
	assert.Contains(t, output, `path="src/main.go"`,
		"file should have path attribute")
	assert.Contains(t, output, `tokens="1200"`,
		"file should have raw token count as attribute")
	assert.Contains(t, output, `tier="critical"`,
		"file should have tier label as attribute")
	assert.Contains(t, output, `size="2048"`,
		"file should have raw size as attribute")
	assert.Contains(t, output, `language="go"`,
		"file should have language attribute")
	assert.Contains(t, output, `compressed="false"`,
		"file should have compressed attribute")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_DirectoryTree
// ---------------------------------------------------------------------------

func TestXMLRenderer_DirectoryTree(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, "<directory_structure>",
		"output should contain directory_structure element")
	assert.Contains(t, output, "<![CDATA[src/",
		"tree string should be wrapped in CDATA")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_Statistics
// ---------------------------------------------------------------------------

func TestXMLRenderer_Statistics(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, "<statistics>",
		"output should contain statistics section")
	// Statistics uses raw (unformatted) numbers.
	statisticsIdx := strings.Index(output, "<statistics>")
	require.True(t, statisticsIdx >= 0)
	statsSection := output[statisticsIdx:]
	assert.Contains(t, statsSection, "<total_files>3</total_files>",
		"statistics should contain total files")
	assert.Contains(t, statsSection, "<total_tokens>2500</total_tokens>",
		"statistics should contain raw total tokens")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_CDATAEdgeCases
// ---------------------------------------------------------------------------

func TestXMLRenderer_CDATAEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		check   func(t *testing.T, output string)
	}{
		{
			name:    "content with ]]> sequence",
			content: "data with ]]> inside it",
			check: func(t *testing.T, output string) {
				t.Helper()
				// The ]]> should be split across CDATA sections.
				assert.Contains(t, output, "]]]]><![CDATA[>",
					"]]> inside content should be split across CDATA sections")
				// Output should still be well-formed XML.
				assertWellFormedXML(t, output)
			},
		},
		{
			name:    "content with multiple ]]> sequences",
			content: "first ]]> second ]]> third",
			check: func(t *testing.T, output string) {
				t.Helper()
				// Count the CDATA splits.
				splits := strings.Count(output, "]]]]><![CDATA[>")
				assert.Equal(t, 2, splits,
					"two ]]> sequences should produce two CDATA splits")
				assertWellFormedXML(t, output)
			},
		},
		{
			name:    "content with XML special characters",
			content: `<div class="test">&amp; 'hello' < > end`,
			check: func(t *testing.T, output string) {
				t.Helper()
				// Inside CDATA, XML special chars should NOT be escaped.
				assert.Contains(t, output, `<div class="test">`,
					"XML chars inside CDATA should not be escaped")
				assert.Contains(t, output, "&amp;",
					"ampersand inside CDATA should remain literal")
				assertWellFormedXML(t, output)
			},
		},
		{
			name:    "empty content",
			content: "",
			check: func(t *testing.T, output string) {
				t.Helper()
				assert.Contains(t, output, "<![CDATA[]]>",
					"empty content should produce empty CDATA section")
				assertWellFormedXML(t, output)
			},
		},
		{
			name:    "content that is valid XML",
			content: `<?xml version="1.0"?><root><item key="val"/></root>`,
			check: func(t *testing.T, output string) {
				t.Helper()
				// The XML content should be inside CDATA, not parsed.
				assert.Contains(t, output, `<![CDATA[<?xml version="1.0"?><root><item key="val"/></root>]]>`,
					"XML content inside CDATA should be preserved literally")
				assertWellFormedXML(t, output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := xmlTestRenderData()
			data.Files = []FileRenderEntry{
				{
					Path:       "test.txt",
					Size:       int64(len(tt.content)),
					TokenCount: 10,
					Tier:       1,
					TierLabel:  "primary",
					Language:   "",
					Content:    tt.content,
				},
			}
			data.TotalFiles = 1

			output := xmlRenderToString(t, context.Background(), data)
			tt.check(t, output)
		})
	}
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_LineNumbers
// ---------------------------------------------------------------------------

func TestXMLRenderer_LineNumbers(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.ShowLineNumbers = true
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

	output := xmlRenderToString(t, context.Background(), data)

	// Line numbers should be present inside CDATA.
	assert.Contains(t, output, "1 | package main",
		"line 1 should be numbered")
	assert.Contains(t, output, "2 | ",
		"line 2 (empty) should be numbered")
	assert.Contains(t, output, "3 | func main() {}",
		"line 3 should be numbered")
	assertWellFormedXML(t, output)
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_NoLineNumbers
// ---------------------------------------------------------------------------

func TestXMLRenderer_NoLineNumbers(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
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

	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, "package main",
		"content should be present")
	assert.NotContains(t, output, "1 | package main",
		"line numbers should not be present when ShowLineNumbers is false")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_FileWithError
// ---------------------------------------------------------------------------

func TestXMLRenderer_FileWithError(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
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

	output := xmlRenderToString(t, context.Background(), data)

	// Error file should show error element instead of content.
	assert.Contains(t, output, "<error>permission denied: broken.go</error>",
		"error file should display error element")

	// The errored file should NOT have a <content> element.
	brokenIdx := strings.Index(output, `path="broken.go"`)
	okIdx := strings.Index(output, `path="ok.go"`)
	require.True(t, brokenIdx >= 0, "broken.go should exist")
	require.True(t, okIdx >= 0, "ok.go should exist")

	brokenSection := output[brokenIdx:okIdx]
	assert.NotContains(t, brokenSection, "<content>",
		"error file should not have content element")

	// OK file should have content.
	okSection := output[okIdx:]
	assert.Contains(t, okSection, "<content>",
		"non-error file should have content element")

	assertWellFormedXML(t, output)
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_EmptyFileList
// ---------------------------------------------------------------------------

func TestXMLRenderer_EmptyFileList(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.Files = []FileRenderEntry{}
	data.TotalFiles = 0
	data.TotalTokens = 0
	data.TierCounts = map[int]int{}
	data.TopFilesByTokens = nil

	output := xmlRenderToString(t, context.Background(), data)

	// Should still produce valid XML.
	assertWellFormedXML(t, output)

	assert.Contains(t, output, "<total_files>0</total_files>",
		"total files should show 0")
	assert.Contains(t, output, "<files>",
		"files section should still be present")
	assert.Contains(t, output, "</files>",
		"files section should close")

	// Should not contain any file elements.
	assert.NotContains(t, output, `<file path=`,
		"should have zero file elements in files section")

	// top_files should not appear.
	assert.NotContains(t, output, "<top_files>",
		"top_files section should not appear when empty")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_ChangeSummary
// ---------------------------------------------------------------------------

func TestXMLRenderer_ChangeSummary(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.DiffSummary = &DiffSummaryData{
		AddedFiles:    []string{"new_file.go", "another_new.go"},
		ModifiedFiles: []string{"src/main.go"},
		DeletedFiles:  []string{"old_file.go"},
	}

	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, "<change_summary>",
		"output should contain change_summary when DiffSummary is set")
	assert.Contains(t, output, `<added count="2">`,
		"added count should be 2")
	assert.Contains(t, output, `<modified count="1">`,
		"modified count should be 1")
	assert.Contains(t, output, `<deleted count="1">`,
		"deleted count should be 1")
	assert.Contains(t, output, "<file>new_file.go</file>",
		"added file should be listed")
	assert.Contains(t, output, "<file>another_new.go</file>",
		"added file should be listed")
	assert.Contains(t, output, "<file>src/main.go</file>",
		"modified file should be listed")
	assert.Contains(t, output, "<file>old_file.go</file>",
		"deleted file should be listed")

	assertWellFormedXML(t, output)
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_NoChangeSummary
// ---------------------------------------------------------------------------

func TestXMLRenderer_NoChangeSummary(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.DiffSummary = nil

	output := xmlRenderToString(t, context.Background(), data)

	assert.NotContains(t, output, "<change_summary>",
		"change_summary should not appear when DiffSummary is nil")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_ChangeSummary_EmptyLists
// ---------------------------------------------------------------------------

func TestXMLRenderer_ChangeSummary_EmptyLists(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.DiffSummary = &DiffSummaryData{
		AddedFiles:    []string{},
		ModifiedFiles: []string{"changed.go"},
		DeletedFiles:  []string{},
	}

	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, "<change_summary>",
		"change_summary should appear when DiffSummary is non-nil")
	assert.Contains(t, output, `<added count="0">`,
		"added count should be 0")
	assert.Contains(t, output, `<modified count="1">`,
		"modified count should be 1")
	assert.Contains(t, output, `<deleted count="0">`,
		"deleted count should be 0")

	assertWellFormedXML(t, output)
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_RedactionSummary
// ---------------------------------------------------------------------------

func TestXMLRenderer_RedactionSummary(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.TotalRedactions = 5
	data.RedactionSummary = map[string]int{
		"AWS Key":      2,
		"GitHub Token": 3,
	}

	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, `<redaction_summary total="5">`,
		"redaction_summary should show total when > 0")
	assert.Contains(t, output, `<type name="AWS Key" count="2"/>`,
		"AWS Key redaction should appear")
	assert.Contains(t, output, `<type name="GitHub Token" count="3"/>`,
		"GitHub Token redaction should appear")

	// Sorted: AWS Key before GitHub Token.
	awsIdx := strings.Index(output, "AWS Key")
	githubIdx := strings.Index(output, "GitHub Token")
	assert.True(t, awsIdx < githubIdx,
		"redaction types should appear in alphabetical order")

	assertWellFormedXML(t, output)
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_RedactionSummary_Zero
// ---------------------------------------------------------------------------

func TestXMLRenderer_RedactionSummary_Zero(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.TotalRedactions = 0
	data.RedactionSummary = map[string]int{}

	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, `<redaction_summary total="0"/>`,
		"zero redactions should show self-closing element")
	assert.NotContains(t, output, `<type name=`,
		"no type elements when zero redactions")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_NilData
// ---------------------------------------------------------------------------

func TestXMLRenderer_NilData(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewXMLRenderer()
	err := r.Render(context.Background(), &buf, nil)

	require.Error(t, err, "nil data should return an error")
	assert.Contains(t, err.Error(), "nil",
		"error message should mention nil")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_CancelledContext
// ---------------------------------------------------------------------------

func TestXMLRenderer_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	var buf bytes.Buffer
	r := NewXMLRenderer()
	data := xmlTestRenderData()

	err := r.Render(ctx, &buf, data)

	require.Error(t, err, "cancelled context should return an error")
	assert.ErrorIs(t, err, context.Canceled,
		"error should be context.Canceled")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_SpecialCharsInPath
// ---------------------------------------------------------------------------

func TestXMLRenderer_SpecialCharsInPath(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:       `dir&name/file<1>.go`,
			Size:       100,
			TokenCount: 50,
			Tier:       1,
			TierLabel:  "primary",
			Language:   "go",
			Content:    "package main",
		},
	}
	data.TotalFiles = 1

	output := xmlRenderToString(t, context.Background(), data)

	// Path attribute should have XML entities escaped.
	assert.Contains(t, output, `path="dir&amp;name/file&lt;1&gt;.go"`,
		"special characters in path should be XML-escaped in attributes")
	assertWellFormedXML(t, output)
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_SpecialCharsInProjectName
// ---------------------------------------------------------------------------

func TestXMLRenderer_SpecialCharsInProjectName(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.ProjectName = `my-project & "friends"`

	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, `<project_name>my-project &amp; &quot;friends&quot;</project_name>`,
		"special characters in project name should be escaped")
	assertWellFormedXML(t, output)
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_TierFallback
// ---------------------------------------------------------------------------

func TestXMLRenderer_TierFallback(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
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

	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, `tier="critical"`,
		"empty TierLabel should fall back to tierLabel function")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_CompressedIndicator
// ---------------------------------------------------------------------------

func TestXMLRenderer_CompressedIndicator(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
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

	output := xmlRenderToString(t, context.Background(), data)

	compIdx := strings.Index(output, `path="compressed.go"`)
	uncompIdx := strings.Index(output, `path="uncompressed.go"`)
	require.True(t, compIdx >= 0)
	require.True(t, uncompIdx >= 0)

	compSection := output[compIdx:uncompIdx]
	assert.Contains(t, compSection, `compressed="true"`,
		"compressed file should show true")

	uncompSection := output[uncompIdx:]
	assert.Contains(t, uncompSection, `compressed="false"`,
		"uncompressed file should show false")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_Deterministic
// ---------------------------------------------------------------------------

func TestXMLRenderer_Deterministic(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()

	output1 := xmlRenderToString(t, context.Background(), data)
	output2 := xmlRenderToString(t, context.Background(), data)

	assert.Equal(t, output1, output2,
		"two renders of the same data must produce identical output")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_SectionOrdering
// ---------------------------------------------------------------------------

func TestXMLRenderer_SectionOrdering(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.DiffSummary = &DiffSummaryData{
		AddedFiles: []string{"new.go"},
	}

	output := xmlRenderToString(t, context.Background(), data)

	metadataIdx := strings.Index(output, "<metadata>")
	summaryIdx := strings.Index(output, "<file_summary>")
	treeIdx := strings.Index(output, "<directory_structure>")
	filesIdx := strings.Index(output, "<files>")
	statsIdx := strings.Index(output, "<statistics>")
	changeIdx := strings.Index(output, "<change_summary>")

	require.True(t, metadataIdx >= 0, "metadata section must exist")
	require.True(t, summaryIdx >= 0, "summary section must exist")
	require.True(t, treeIdx >= 0, "tree section must exist")
	require.True(t, filesIdx >= 0, "files section must exist")
	require.True(t, statsIdx >= 0, "statistics section must exist")
	require.True(t, changeIdx >= 0, "change_summary section must exist")

	assert.True(t, metadataIdx < summaryIdx,
		"metadata should come before summary")
	assert.True(t, summaryIdx < treeIdx,
		"summary should come before tree")
	assert.True(t, treeIdx < filesIdx,
		"tree should come before files")
	assert.True(t, filesIdx < statsIdx,
		"files should come before statistics")
	assert.True(t, statsIdx < changeIdx,
		"statistics should come before change_summary")
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_ZeroTokenFiles
// ---------------------------------------------------------------------------

func TestXMLRenderer_ZeroTokenFiles(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
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
	}
	data.TotalFiles = 1
	data.TotalTokens = 0

	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, `tokens="0"`,
		"zero-token file should render tokens=0")
	assert.Contains(t, output, `size="0"`,
		"zero-size file should render size=0")
	assertWellFormedXML(t, output)
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_LargeContent
// ---------------------------------------------------------------------------

func TestXMLRenderer_LargeContent(t *testing.T) {
	t.Parallel()

	// Generate a large content string (100KB).
	largeContent := strings.Repeat("line of code with some content here\n", 3000)

	data := xmlTestRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:       "large.go",
			Size:       int64(len(largeContent)),
			TokenCount: 50000,
			Tier:       1,
			TierLabel:  "primary",
			Language:   "go",
			Content:    largeContent,
		},
	}
	data.TotalFiles = 1

	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, "line of code",
		"large content should be rendered")
	assertWellFormedXML(t, output)
}

// ---------------------------------------------------------------------------
// TestXMLRenderer_FileWithErrorAndSpecialChars
// ---------------------------------------------------------------------------

func TestXMLRenderer_FileWithErrorAndSpecialChars(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:       "test.go",
			Size:       100,
			TokenCount: 0,
			Tier:       1,
			TierLabel:  "primary",
			Language:   "go",
			Error:      `error <"reading"> file & stuff`,
		},
	}
	data.TotalFiles = 1

	output := xmlRenderToString(t, context.Background(), data)

	assert.Contains(t, output, `<error>error &lt;&quot;reading&quot;&gt; file &amp; stuff</error>`,
		"error message with special chars should be XML-escaped")
	assertWellFormedXML(t, output)
}

// ---------------------------------------------------------------------------
// TestWrapCDATA
// ---------------------------------------------------------------------------

func TestWrapCDATA(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "empty string",
			content: "",
			want:    "<![CDATA[]]>",
		},
		{
			name:    "simple content",
			content: "hello world",
			want:    "<![CDATA[hello world]]>",
		},
		{
			name:    "content with XML chars",
			content: `<div class="test">&</div>`,
			want:    `<![CDATA[<div class="test">&</div>]]>`,
		},
		{
			name:    "content with ]]>",
			content: "before ]]> after",
			want:    "<![CDATA[before ]]]]><![CDATA[> after]]>",
		},
		{
			name:    "content ending with ]]>",
			content: "data]]>",
			want:    "<![CDATA[data]]]]><![CDATA[>]]>",
		},
		{
			name:    "multiple ]]> sequences",
			content: "a]]>b]]>c",
			want:    "<![CDATA[a]]]]><![CDATA[>b]]]]><![CDATA[>c]]>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := wrapCDATA(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestXmlEscapeAttr
// ---------------------------------------------------------------------------

func TestXmlEscapeAttr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no special chars",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "ampersand",
			input: "a & b",
			want:  "a &amp; b",
		},
		{
			name:  "less than",
			input: "a < b",
			want:  "a &lt; b",
		},
		{
			name:  "greater than",
			input: "a > b",
			want:  "a &gt; b",
		},
		{
			name:  "double quote",
			input: `say "hello"`,
			want:  "say &quot;hello&quot;",
		},
		{
			name:  "single quote",
			input: "it's",
			want:  "it&apos;s",
		},
		{
			name:  "all special chars",
			input: `<>&"'`,
			want:  "&lt;&gt;&amp;&quot;&apos;",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := xmlEscapeAttr(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Golden tests
// ---------------------------------------------------------------------------

func TestXMLRenderer_GoldenBasic(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	var buf bytes.Buffer
	r := NewXMLRenderer()
	err := r.Render(context.Background(), &buf, data)
	require.NoError(t, err)

	testutil.Golden(t, "xml-basic", buf.Bytes())
}

func TestXMLRenderer_GoldenLineNumbers(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.ShowLineNumbers = true

	var buf bytes.Buffer
	r := NewXMLRenderer()
	err := r.Render(context.Background(), &buf, data)
	require.NoError(t, err)

	testutil.Golden(t, "xml-line-numbers", buf.Bytes())
}

func TestXMLRenderer_GoldenCDATAEdge(t *testing.T) {
	t.Parallel()

	data := xmlTestRenderData()
	data.Files = []FileRenderEntry{
		{
			Path:       "edge.txt",
			Size:       100,
			TokenCount: 20,
			Tier:       1,
			TierLabel:  "primary",
			Language:   "",
			Content:    "normal text\nwith ]]> inside\nand <xml> & \"quotes\"",
		},
	}
	data.TotalFiles = 1

	var buf bytes.Buffer
	r := NewXMLRenderer()
	err := r.Render(context.Background(), &buf, data)
	require.NoError(t, err)

	testutil.Golden(t, "xml-cdata-edge", buf.Bytes())
}

// ---------------------------------------------------------------------------
// Benchmark tests
// ---------------------------------------------------------------------------

func BenchmarkXMLRenderer(b *testing.B) {
	data := xmlTestRenderData()
	r := NewXMLRenderer()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = r.Render(ctx, &buf, data)
	}
}

func BenchmarkXMLRenderer_LargeFileList(b *testing.B) {
	data := xmlTestRenderData()
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

	r := NewXMLRenderer()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = r.Render(ctx, &buf, data)
	}
}

// ---------------------------------------------------------------------------
// Helper: assertWellFormedXML
// ---------------------------------------------------------------------------

// assertWellFormedXML verifies that the output is valid, well-formed XML by
// parsing it with the standard encoding/xml decoder.
func assertWellFormedXML(t *testing.T, output string) {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(output))
	for {
		_, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return
			}
			t.Errorf("XML is not well-formed: %v\nOutput:\n%s", err, output)
			return
		}
	}
}
