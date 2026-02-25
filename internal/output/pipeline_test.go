package output

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Shared test fixtures
// ---------------------------------------------------------------------------

// fixedPipelineTimestamp is a deterministic timestamp for all pipeline tests.
var fixedPipelineTimestamp = time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

// sampleFileDescriptors returns a set of 5 pipeline.FileDescriptor values
// matching the golden fixture files. Content is embedded inline for
// deterministic token counts (not read from disk).
func sampleFileDescriptors() []pipeline.FileDescriptor {
	return []pipeline.FileDescriptor{
		{
			Path:       "go.mod",
			AbsPath:    "/tmp/sample/go.mod",
			Size:       39,
			Tier:       0,
			TokenCount: 15,
			Content:    "module example.com/sample\n\ngo 1.24.0\n",
			Language:   "",
		},
		{
			Path:       "README.md",
			AbsPath:    "/tmp/sample/README.md",
			Size:       60,
			Tier:       4,
			TokenCount: 12,
			Content:    "# Sample Project\n\nA sample project for testing output rendering.\n",
			Language:   "markdown",
		},
		{
			Path:       "cmd/main.go",
			AbsPath:    "/tmp/sample/cmd/main.go",
			Size:       68,
			Tier:       1,
			TokenCount: 25,
			Content:    "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello world\")\n}\n",
			Language:   "go",
		},
		{
			Path:       "internal/handler.go",
			AbsPath:    "/tmp/sample/internal/handler.go",
			Size:       138,
			Tier:       1,
			TokenCount: 35,
			Content:    "package internal\n\n// Handler processes requests.\ntype Handler struct {\n\tName string\n}\n\n// Handle processes a single request.\nfunc (h *Handler) Handle() error {\n\treturn nil\n}\n",
			Language:   "go",
		},
		{
			Path:       "internal/handler_test.go",
			AbsPath:    "/tmp/sample/internal/handler_test.go",
			Size:       131,
			Tier:       3,
			TokenCount: 40,
			Content:    "package internal\n\nimport \"testing\"\n\nfunc TestHandler_Handle(t *testing.T) {\n\th := &Handler{Name: \"test\"}\n\tif err := h.Handle(); err != nil {\n\t\tt.Fatal(err)\n\t}\n}\n",
			Language:   "go",
		},
	}
}

// basePipelineConfig returns a minimal OutputConfig for pipeline tests.
func basePipelineConfig(dir string) OutputConfig {
	return OutputConfig{
		Format:        FormatMarkdown,
		Target:        "generic",
		OutputPath:    filepath.Join(dir, "output.md"),
		UseStdout:     false,
		ProjectName:   "sample-project",
		ProfileName:   "default",
		TokenizerName: "cl100k_base",
		Timestamp:     fixedPipelineTimestamp,
	}
}

// stdoutPipelineConfig returns a config that writes to stdout via injectable
// streams, using the provided OutputWriter.
func stdoutPipelineConfig(writer *OutputWriter) OutputConfig {
	return OutputConfig{
		Format:        FormatMarkdown,
		Target:        "generic",
		UseStdout:     true,
		ProjectName:   "sample-project",
		ProfileName:   "default",
		TokenizerName: "cl100k_base",
		Timestamp:     fixedPipelineTimestamp,
		Writer:        writer,
	}
}

// goldenDir returns the absolute path to the pipeline golden test directory.
func goldenDir() string {
	return filepath.Join("testdata", "golden")
}

// ---------------------------------------------------------------------------
// Unit tests for conversion helpers
// ---------------------------------------------------------------------------

func TestToFileRenderEntries(t *testing.T) {
	t.Parallel()

	files := sampleFileDescriptors()
	entries := toFileRenderEntries(files)

	require.Len(t, entries, 5)
	assert.Equal(t, "go.mod", entries[0].Path)
	assert.Equal(t, int64(39), entries[0].Size)
	assert.Equal(t, 0, entries[0].Tier)
	assert.Equal(t, "critical", entries[0].TierLabel)
	assert.Equal(t, 15, entries[0].TokenCount)
	assert.Empty(t, entries[0].Error)
}

func TestToFileRenderEntries_InfersLanguage(t *testing.T) {
	t.Parallel()

	files := []pipeline.FileDescriptor{
		{Path: "go.mod", Language: ""},
	}
	entries := toFileRenderEntries(files)

	// go.mod has no mapping in extToLanguage -- the Language remains
	// whatever languageFromExt returns. Just verify it doesn't panic.
	require.Len(t, entries, 1)
}

func TestToFileRenderEntries_WithError(t *testing.T) {
	t.Parallel()

	files := []pipeline.FileDescriptor{
		{
			Path:  "broken.go",
			Error: os.ErrPermission,
		},
	}
	entries := toFileRenderEntries(files)

	require.Len(t, entries, 1)
	assert.Equal(t, "permission denied", entries[0].Error)
}

func TestToFileEntries(t *testing.T) {
	t.Parallel()

	files := sampleFileDescriptors()
	entries := toFileEntries(files)

	require.Len(t, entries, 5)
	assert.Equal(t, "go.mod", entries[0].Path)
	assert.Equal(t, int64(39), entries[0].Size)
	assert.Equal(t, 0, entries[0].Tier)
}

func TestToFileHashEntries(t *testing.T) {
	t.Parallel()

	files := sampleFileDescriptors()
	entries := toFileHashEntries(files)

	require.Len(t, entries, 5)
	assert.Equal(t, "go.mod", entries[0].Path)
	assert.Equal(t, files[0].Content, entries[0].Content)
}

func TestComputeTierCounts(t *testing.T) {
	t.Parallel()

	entries := toFileRenderEntries(sampleFileDescriptors())
	counts := computeTierCounts(entries)

	assert.Equal(t, 1, counts[0]) // go.mod
	assert.Equal(t, 2, counts[1]) // cmd/main.go, internal/handler.go
	assert.Equal(t, 1, counts[3]) // internal/handler_test.go
	assert.Equal(t, 1, counts[4]) // README.md
}

func TestComputeTotalTokens(t *testing.T) {
	t.Parallel()

	entries := toFileRenderEntries(sampleFileDescriptors())
	total := computeTotalTokens(entries)

	assert.Equal(t, 15+12+25+35+40, total)
}

func TestComputeTotalRedactions(t *testing.T) {
	t.Parallel()

	entries := []FileRenderEntry{
		{Redactions: 3},
		{Redactions: 0},
		{Redactions: 7},
	}
	assert.Equal(t, 10, computeTotalRedactions(entries))
}

func TestComputeTopFiles(t *testing.T) {
	t.Parallel()

	entries := toFileRenderEntries(sampleFileDescriptors())
	top := computeTopFiles(entries, 3)

	require.Len(t, top, 3)
	// Should be sorted descending by token count.
	assert.GreaterOrEqual(t, top[0].TokenCount, top[1].TokenCount)
	assert.GreaterOrEqual(t, top[1].TokenCount, top[2].TokenCount)
}

func TestComputeTopFiles_FewerThanN(t *testing.T) {
	t.Parallel()

	entries := toFileRenderEntries(sampleFileDescriptors()[:2])
	top := computeTopFiles(entries, 5)

	assert.Len(t, top, 2)
}

func TestComputeTopFiles_Empty(t *testing.T) {
	t.Parallel()

	top := computeTopFiles(nil, 5)
	assert.Nil(t, top)
}

// ---------------------------------------------------------------------------
// Integration tests: RenderOutput
// ---------------------------------------------------------------------------

func TestRenderOutput_MarkdownBasic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := basePipelineConfig(dir)
	files := sampleFileDescriptors()

	result, err := RenderOutput(context.Background(), cfg, files)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.Path)
	assert.NotZero(t, result.Hash)
	assert.NotEmpty(t, result.HashHex)
	assert.Equal(t, 16, len(result.HashHex))
	assert.Equal(t, 127, result.TotalTokens) // 15+12+25+35+40
	assert.Greater(t, result.BytesWritten, int64(0))

	// Verify file exists and contains expected content.
	content, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Harvx Context: sample-project")
	assert.Contains(t, string(content), "go.mod")
	assert.Contains(t, string(content), "cmd/main.go")
	assert.Contains(t, string(content), "internal/handler.go")
}

func TestRenderOutput_XMLBasic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := basePipelineConfig(dir)
	cfg.Format = FormatXML
	cfg.OutputPath = filepath.Join(dir, "output.xml")
	files := sampleFileDescriptors()

	result, err := RenderOutput(context.Background(), cfg, files)
	require.NoError(t, err)
	require.NotNil(t, result)

	content, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "<repository>")
	assert.Contains(t, string(content), "sample-project")
	assert.Contains(t, string(content), "go.mod")
}

func TestRenderOutput_SameContentHash_MarkdownAndXML(t *testing.T) {
	t.Parallel()

	files := sampleFileDescriptors()

	// Render as Markdown.
	var mdStdout, mdStderr bytes.Buffer
	mdWriter := NewOutputWriterWithStreams(&mdStdout, &mdStderr)
	mdCfg := stdoutPipelineConfig(mdWriter)
	mdCfg.Format = FormatMarkdown

	mdResult, err := RenderOutput(context.Background(), mdCfg, files)
	require.NoError(t, err)

	// Render as XML.
	var xmlStdout, xmlStderr bytes.Buffer
	xmlWriter := NewOutputWriterWithStreams(&xmlStdout, &xmlStderr)
	xmlCfg := stdoutPipelineConfig(xmlWriter)
	xmlCfg.Format = FormatXML

	xmlResult, err := RenderOutput(context.Background(), xmlCfg, files)
	require.NoError(t, err)

	// The content hash is computed from the input files, not the rendered
	// output. Both formats should produce the same content hash in the
	// rendered metadata (the TotalTokens from file data).
	assert.Equal(t, mdResult.TotalTokens, xmlResult.TotalTokens)

	// Note: the OutputResult.Hash is the hash of the rendered output bytes,
	// which differs between formats. But the content hash in RenderData
	// (derived from input files) is the same.
	mdContent := mdStdout.String()
	xmlContent := xmlStdout.String()

	// Both should contain the same content hash hex string in their output.
	hasher := NewContentHasher()
	hashEntries := toFileHashEntries(files)
	expectedHash, err := hasher.ComputeContentHash(hashEntries)
	require.NoError(t, err)
	expectedHashHex := FormatHash(expectedHash)

	assert.Contains(t, mdContent, expectedHashHex)
	assert.Contains(t, xmlContent, expectedHashHex)
}

func TestRenderOutput_StdoutEqualsFileOutput(t *testing.T) {
	t.Parallel()

	files := sampleFileDescriptors()
	dir := t.TempDir()

	// Write to file.
	fileCfg := basePipelineConfig(dir)

	fileResult, err := RenderOutput(context.Background(), fileCfg, files)
	require.NoError(t, err)

	fileContent, err := os.ReadFile(fileResult.Path)
	require.NoError(t, err)

	// Write to stdout.
	var stdout, stderr bytes.Buffer
	writer := NewOutputWriterWithStreams(&stdout, &stderr)
	stdoutCfg := stdoutPipelineConfig(writer)

	stdoutResult, err := RenderOutput(context.Background(), stdoutCfg, files)
	require.NoError(t, err)

	// Content should be identical.
	assert.Equal(t, string(fileContent), stdout.String())
	assert.Equal(t, fileResult.Hash, stdoutResult.Hash)
	assert.Equal(t, fileResult.HashHex, stdoutResult.HashHex)
	assert.Equal(t, fileResult.BytesWritten, stdoutResult.BytesWritten)
}

func TestRenderOutput_CorrectOutputResult(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := basePipelineConfig(dir)
	files := sampleFileDescriptors()

	result, err := RenderOutput(context.Background(), cfg, files)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(dir, "output.md"), result.Path)
	assert.NotZero(t, result.Hash)
	assert.Equal(t, 16, len(result.HashHex))
	assert.Equal(t, 127, result.TotalTokens)
	assert.Greater(t, result.BytesWritten, int64(0))
	assert.Nil(t, result.Parts) // No split mode.
}

func TestRenderOutput_ZeroFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := basePipelineConfig(dir)

	result, err := RenderOutput(context.Background(), cfg, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0, result.TotalTokens)
	assert.Greater(t, result.BytesWritten, int64(0)) // Header still renders.

	content, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Harvx Context: sample-project")
	assert.Contains(t, string(content), "Total Files | 0")
}

func TestRenderOutput_SingleFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := basePipelineConfig(dir)
	files := sampleFileDescriptors()[:1]

	result, err := RenderOutput(context.Background(), cfg, files)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 15, result.TotalTokens)
	content, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "go.mod")
	assert.Contains(t, string(content), "Total Files | 1")
}

func TestRenderOutput_AllOptionsEnabled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := basePipelineConfig(dir)
	cfg.ShowLineNumbers = true
	cfg.OutputMetadata = true
	cfg.ShowTreeMetadata = true
	cfg.MaxTokens = 1000
	cfg.GenerationTimeMs = 42
	files := sampleFileDescriptors()

	result, err := RenderOutput(context.Background(), cfg, files)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check line numbers in output.
	content, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "1 | ")

	// Check metadata sidecar was written.
	metaPath := MetadataSidecarPath(result.Path)
	metaContent, err := os.ReadFile(metaPath)
	require.NoError(t, err)
	assert.Contains(t, string(metaContent), "\"version\"")
	assert.Contains(t, string(metaContent), "\"total_files\"")
	assert.Contains(t, string(metaContent), "\"total_tokens\"")
}

func TestRenderOutput_ContentHashChanges(t *testing.T) {
	t.Parallel()

	files1 := sampleFileDescriptors()
	files2 := sampleFileDescriptors()
	files2[0].Content = "module example.com/changed\n\ngo 1.24.0\n"

	var stdout1, stderr1 bytes.Buffer
	writer1 := NewOutputWriterWithStreams(&stdout1, &stderr1)
	cfg1 := stdoutPipelineConfig(writer1)

	var stdout2, stderr2 bytes.Buffer
	writer2 := NewOutputWriterWithStreams(&stdout2, &stderr2)
	cfg2 := stdoutPipelineConfig(writer2)

	result1, err := RenderOutput(context.Background(), cfg1, files1)
	require.NoError(t, err)

	result2, err := RenderOutput(context.Background(), cfg2, files2)
	require.NoError(t, err)

	// The rendered output hash should differ because content hash differs.
	assert.NotEqual(t, result1.Hash, result2.Hash)
}

func TestRenderOutput_AddingFileChangesHash(t *testing.T) {
	t.Parallel()

	files1 := sampleFileDescriptors()[:3]
	files2 := sampleFileDescriptors()[:4]

	var stdout1, stderr1 bytes.Buffer
	writer1 := NewOutputWriterWithStreams(&stdout1, &stderr1)
	cfg1 := stdoutPipelineConfig(writer1)

	var stdout2, stderr2 bytes.Buffer
	writer2 := NewOutputWriterWithStreams(&stdout2, &stderr2)
	cfg2 := stdoutPipelineConfig(writer2)

	result1, err := RenderOutput(context.Background(), cfg1, files1)
	require.NoError(t, err)

	result2, err := RenderOutput(context.Background(), cfg2, files2)
	require.NoError(t, err)

	// Adding a file changes everything: tree, content hash, output hash.
	assert.NotEqual(t, result1.Hash, result2.Hash)
}

func TestRenderOutput_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := t.TempDir()
	cfg := basePipelineConfig(dir)

	result, err := RenderOutput(ctx, cfg, sampleFileDescriptors())
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRenderOutput_DefaultTimestamp(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	writer := NewOutputWriterWithStreams(&stdout, &stderr)

	cfg := stdoutPipelineConfig(writer)
	cfg.Timestamp = time.Time{} // Zero value = use time.Now()

	result, err := RenderOutput(context.Background(), cfg, sampleFileDescriptors()[:1])
	require.NoError(t, err)
	require.NotNil(t, result)

	// Output should contain a generated timestamp (not zero time).
	output := stdout.String()
	assert.NotContains(t, output, "0001-01-01")
}

func TestRenderOutput_WithDiffSummary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := basePipelineConfig(dir)
	cfg.DiffSummary = &DiffSummaryData{
		AddedFiles:    []string{"new-file.go"},
		ModifiedFiles: []string{"cmd/main.go"},
		DeletedFiles:  []string{"old-file.go"},
	}

	result, err := RenderOutput(context.Background(), cfg, sampleFileDescriptors())
	require.NoError(t, err)

	content, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Change Summary")
	assert.Contains(t, string(content), "new-file.go")
	assert.Contains(t, string(content), "old-file.go")
}

func TestRenderOutput_TreeDepthLimit(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	writer := NewOutputWriterWithStreams(&stdout, &stderr)

	cfg := stdoutPipelineConfig(writer)
	cfg.TreeMaxDepth = 1

	result, err := RenderOutput(context.Background(), cfg, sampleFileDescriptors())
	require.NoError(t, err)
	require.NotNil(t, result)

	output := stdout.String()
	assert.Contains(t, output, "...")
}

// ---------------------------------------------------------------------------
// Split output integration tests
// ---------------------------------------------------------------------------

func TestRenderOutput_SplitOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := basePipelineConfig(dir)
	cfg.SplitTokens = 50 // Low enough to force multiple parts.
	cfg.OutputPath = filepath.Join(dir, "output.md")

	files := sampleFileDescriptors()

	result, err := RenderOutput(context.Background(), cfg, files)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have parts.
	assert.NotNil(t, result.Parts)
	assert.GreaterOrEqual(t, len(result.Parts), 2, "expected at least 2 parts")

	// Each part file should exist.
	for _, part := range result.Parts {
		_, err := os.Stat(part.Path)
		assert.NoError(t, err, "part file should exist: %s", part.Path)
	}
}

// ---------------------------------------------------------------------------
// Golden tests
// ---------------------------------------------------------------------------

func TestGolden_MarkdownBasic(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	writer := NewOutputWriterWithStreams(&stdout, &stderr)

	cfg := stdoutPipelineConfig(writer)

	_, err := RenderOutput(context.Background(), cfg, sampleFileDescriptors())
	require.NoError(t, err)

	goldenPath := filepath.Join(goldenDir(), "pipeline-markdown-basic.golden")
	compareGolden(t, stdout.Bytes(), goldenPath)
}

func TestGolden_MarkdownWithLineNumbers(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	writer := NewOutputWriterWithStreams(&stdout, &stderr)

	cfg := stdoutPipelineConfig(writer)
	cfg.ShowLineNumbers = true

	_, err := RenderOutput(context.Background(), cfg, sampleFileDescriptors())
	require.NoError(t, err)

	goldenPath := filepath.Join(goldenDir(), "pipeline-markdown-line-numbers.golden")
	compareGolden(t, stdout.Bytes(), goldenPath)
}

func TestGolden_XMLBasic(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	writer := NewOutputWriterWithStreams(&stdout, &stderr)

	cfg := stdoutPipelineConfig(writer)
	cfg.Format = FormatXML

	_, err := RenderOutput(context.Background(), cfg, sampleFileDescriptors())
	require.NoError(t, err)

	goldenPath := filepath.Join(goldenDir(), "pipeline-xml-basic.golden")
	compareGolden(t, stdout.Bytes(), goldenPath)
}

func TestGolden_MarkdownWithDiff(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	writer := NewOutputWriterWithStreams(&stdout, &stderr)

	cfg := stdoutPipelineConfig(writer)
	cfg.DiffSummary = &DiffSummaryData{
		AddedFiles:    []string{"new-feature.go"},
		ModifiedFiles: []string{"cmd/main.go", "internal/handler.go"},
		DeletedFiles:  []string{"deprecated.go"},
	}

	_, err := RenderOutput(context.Background(), cfg, sampleFileDescriptors())
	require.NoError(t, err)

	goldenPath := filepath.Join(goldenDir(), "pipeline-markdown-diff.golden")
	compareGolden(t, stdout.Bytes(), goldenPath)
}

func TestGolden_MetadataSidecar(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := basePipelineConfig(dir)
	cfg.OutputMetadata = true
	cfg.MaxTokens = 1000
	cfg.GenerationTimeMs = 42

	result, err := RenderOutput(context.Background(), cfg, sampleFileDescriptors())
	require.NoError(t, err)

	metaPath := MetadataSidecarPath(result.Path)
	metaContent, err := os.ReadFile(metaPath)
	require.NoError(t, err)

	goldenPath := filepath.Join(goldenDir(), "pipeline-metadata-sidecar.golden")
	compareGolden(t, metaContent, goldenPath)
}
