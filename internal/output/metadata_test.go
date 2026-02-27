package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// metadataTestRenderData returns a RenderData with multiple files and redaction
// data, suitable for metadata generation tests.
func metadataTestRenderData() *RenderData {
	return &RenderData{
		ProjectName:   "test-project",
		Timestamp:     time.Date(2026, 2, 16, 10, 30, 0, 0, time.UTC),
		ContentHash:   "a1b2c3d4e5f6a7b8",
		ProfileName:   "finvault",
		TokenizerName: "o200k_base",
		TotalTokens:   1500,
		TotalFiles:    3,
		Files: []FileRenderEntry{
			{
				Path:         "src/main.go",
				Size:         1024,
				TokenCount:   500,
				Tier:         0,
				TierLabel:    "critical",
				Language:     "go",
				Content:      "package main\n\nfunc main() {}",
				IsCompressed: false,
				Redactions:   0,
			},
			{
				Path:         "README.md",
				Size:         512,
				TokenCount:   200,
				Tier:         4,
				TierLabel:    "docs",
				Language:     "markdown",
				Content:      "# Test Project",
				IsCompressed: false,
				Redactions:   0,
			},
			{
				Path:         "internal/config/config.go",
				Size:         2048,
				TokenCount:   800,
				Tier:         1,
				TierLabel:    "primary",
				Language:     "go",
				Content:      "package config",
				IsCompressed: true,
				Redactions:   2,
			},
		},
		TreeString:      ".",
		ShowLineNumbers: false,
		TierCounts: map[int]int{
			0: 1,
			1: 1,
			4: 1,
		},
		TopFilesByTokens: nil,
		RedactionSummary: map[string]int{
			"aws_access_key":    1,
			"connection_string": 1,
		},
		TotalRedactions: 2,
	}
}

// testOutputResult returns an OutputResult suitable for metadata tests.
func testOutputResult() *OutputResult {
	return &OutputResult{
		Path:         "/tmp/test/harvx-output.md",
		Hash:         0xa1b2c3d4e5f6a7b8,
		HashHex:      "a1b2c3d4e5f6a7b8",
		TotalTokens:  1500,
		BytesWritten: 4096,
	}
}

// testMetadataOpts returns a MetadataOpts with typical values.
func testMetadataOpts() MetadataOpts {
	return MetadataOpts{
		RenderData:       metadataTestRenderData(),
		Result:           testOutputResult(),
		Format:           "markdown",
		Target:           "claude",
		MaxTokens:        200000,
		GenerationTimeMs: 850,
	}
}

// ---------------------------------------------------------------------------
// 1. TestGenerateMetadata_BasicFields
// Verify all top-level fields: version, generated_at RFC3339, profile,
// tokenizer, format, target, content_hash.
// ---------------------------------------------------------------------------

func TestGenerateMetadata_BasicFields(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	meta := GenerateMetadata(opts)

	assert.Equal(t, MetadataVersion, meta.Version)
	assert.Equal(t, "1.0.0", meta.Version)

	// GeneratedAt must be valid RFC 3339.
	_, err := time.Parse(time.RFC3339, meta.GeneratedAt)
	require.NoError(t, err, "generated_at must be valid RFC 3339: %s", meta.GeneratedAt)
	assert.Equal(t, "2026-02-16T10:30:00Z", meta.GeneratedAt)

	assert.Equal(t, "finvault", meta.Profile)
	assert.Equal(t, "o200k_base", meta.Tokenizer)
	assert.Equal(t, "markdown", meta.Format)
	assert.Equal(t, "claude", meta.Target)
	assert.Equal(t, "a1b2c3d4e5f6a7b8", meta.ContentHash)
}

// ---------------------------------------------------------------------------
// 2. TestGenerateMetadata_Statistics
// Verify statistics computed correctly: TotalFiles, TotalTokens, TotalBytes,
// FilesByTier, RedactionsTotal, RedactionsByType, CompressedFiles,
// GenerationTimeMs.
// ---------------------------------------------------------------------------

func TestGenerateMetadata_Statistics(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	meta := GenerateMetadata(opts)

	stats := meta.Statistics

	// TotalFiles from RenderData.TotalFiles.
	assert.Equal(t, 3, stats.TotalFiles)

	// TotalTokens from RenderData.TotalTokens.
	assert.Equal(t, 1500, stats.TotalTokens)

	// TotalBytes: sum of File.Size (1024 + 512 + 2048 = 3584).
	assert.Equal(t, int64(1024+512+2048), stats.TotalBytes)

	// MaxTokens from opts.
	assert.Equal(t, 200000, stats.MaxTokens)

	// FilesByTier with string keys from TierCounts.
	assert.Equal(t, map[string]int{"0": 1, "1": 1, "4": 1}, stats.FilesByTier)

	// RedactionsTotal from TotalRedactions.
	assert.Equal(t, 2, stats.RedactionsTotal)

	// RedactionsByType from RedactionSummary.
	assert.Equal(t, map[string]int{
		"aws_access_key":    1,
		"connection_string": 1,
	}, stats.RedactionsByType)

	// CompressedFiles count: only internal/config/config.go is compressed.
	assert.Equal(t, 1, stats.CompressedFiles)

	// GenerationTimeMs from opts.
	assert.Equal(t, int64(850), stats.GenerationTimeMs)
}

// ---------------------------------------------------------------------------
// 3. TestGenerateMetadata_BudgetUsedPercent (table-driven)
// ---------------------------------------------------------------------------

func TestGenerateMetadata_BudgetUsedPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		maxTokens   int
		totalTokens int
		wantNil     bool
		wantPercent float64
	}{
		{
			name:        "MaxTokens=200000 TotalTokens=89420 -> ~44.71",
			maxTokens:   200000,
			totalTokens: 89420,
			wantNil:     false,
			wantPercent: 44.71,
		},
		{
			name:        "MaxTokens=0 -> nil (no budget)",
			maxTokens:   0,
			totalTokens: 89420,
			wantNil:     true,
		},
		{
			name:        "MaxTokens=100 TotalTokens=100 -> 100.0",
			maxTokens:   100,
			totalTokens: 100,
			wantNil:     false,
			wantPercent: 100.0,
		},
		{
			name:        "MaxTokens=100 TotalTokens=0 -> 0.0",
			maxTokens:   100,
			totalTokens: 0,
			wantNil:     false,
			wantPercent: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := testMetadataOpts()
			opts.MaxTokens = tt.maxTokens
			opts.RenderData.TotalTokens = tt.totalTokens

			meta := GenerateMetadata(opts)

			if tt.wantNil {
				assert.Nil(t, meta.Statistics.BudgetUsedPercent,
					"budget_used_percent should be nil when MaxTokens=0")
			} else {
				require.NotNil(t, meta.Statistics.BudgetUsedPercent)
				assert.InDelta(t, tt.wantPercent, *meta.Statistics.BudgetUsedPercent, 0.01)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 4. TestGenerateMetadata_FilesSortedByPath
// Files with paths z.go, a.go, m.go -> sorted a.go, m.go, z.go.
// ---------------------------------------------------------------------------

func TestGenerateMetadata_FilesSortedByPath(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	opts.RenderData.Files = []FileRenderEntry{
		{Path: "z.go", Size: 100, TokenCount: 10, Tier: 0, Language: "go"},
		{Path: "a.go", Size: 200, TokenCount: 20, Tier: 1, Language: "go"},
		{Path: "m.go", Size: 150, TokenCount: 15, Tier: 2, Language: "go"},
	}
	opts.RenderData.TotalFiles = 3

	meta := GenerateMetadata(opts)

	require.Len(t, meta.Files, 3)
	assert.Equal(t, "a.go", meta.Files[0].Path)
	assert.Equal(t, "m.go", meta.Files[1].Path)
	assert.Equal(t, "z.go", meta.Files[2].Path)
}

// ---------------------------------------------------------------------------
// 5. TestGenerateMetadata_FilesFields
// Verify each FileStats field is mapped correctly from FileRenderEntry.
// ---------------------------------------------------------------------------

func TestGenerateMetadata_FilesFields(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	opts.RenderData.Files = []FileRenderEntry{
		{
			Path:         "src/handler.go",
			Size:         2048,
			TokenCount:   500,
			Tier:         1,
			TierLabel:    "primary",
			Language:     "go",
			Content:      "package handler",
			IsCompressed: true,
			Redactions:   3,
			Error:        "",
		},
	}
	opts.RenderData.TotalFiles = 1

	meta := GenerateMetadata(opts)
	require.Len(t, meta.Files, 1)

	fs := meta.Files[0]
	assert.Equal(t, "src/handler.go", fs.Path)
	assert.Equal(t, 1, fs.Tier)
	assert.Equal(t, 500, fs.Tokens)
	assert.Equal(t, int64(2048), fs.Bytes)
	assert.Equal(t, 3, fs.Redactions)
	assert.True(t, fs.Compressed)
	assert.Equal(t, "go", fs.Language)
}

// ---------------------------------------------------------------------------
// 6. TestGenerateMetadata_EmptyFiles
// Empty file list produces valid metadata with zero counts and empty but
// non-nil maps.
// ---------------------------------------------------------------------------

func TestGenerateMetadata_EmptyFiles(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	opts.RenderData.Files = nil
	opts.RenderData.TotalFiles = 0
	opts.RenderData.TotalTokens = 0
	opts.RenderData.TierCounts = nil
	opts.RenderData.RedactionSummary = nil
	opts.RenderData.TotalRedactions = 0

	meta := GenerateMetadata(opts)

	// Zero counts.
	assert.Equal(t, 0, meta.Statistics.TotalFiles)
	assert.Equal(t, 0, meta.Statistics.TotalTokens)
	assert.Equal(t, int64(0), meta.Statistics.TotalBytes)
	assert.Equal(t, 0, meta.Statistics.CompressedFiles)
	assert.Equal(t, 0, meta.Statistics.RedactionsTotal)

	// Files slice is empty but not nil for JSON [].
	assert.Empty(t, meta.Files)

	// Maps are non-nil for JSON {}.
	require.NotNil(t, meta.Statistics.FilesByTier)
	assert.Empty(t, meta.Statistics.FilesByTier)
	require.NotNil(t, meta.Statistics.RedactionsByType)
	assert.Empty(t, meta.Statistics.RedactionsByType)

	// Verify JSON is valid.
	data, err := json.Marshal(meta)
	require.NoError(t, err)

	var parsed OutputMetadata
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, 0, parsed.Statistics.TotalFiles)
}

// ---------------------------------------------------------------------------
// 7. TestGenerateMetadata_NoRedactions
// redactions_total=0, redactions_by_type=empty map (not nil).
// ---------------------------------------------------------------------------

func TestGenerateMetadata_NoRedactions(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	opts.RenderData.RedactionSummary = nil
	opts.RenderData.TotalRedactions = 0

	// Clear file-level redactions.
	for i := range opts.RenderData.Files {
		opts.RenderData.Files[i].Redactions = 0
	}

	meta := GenerateMetadata(opts)

	assert.Equal(t, 0, meta.Statistics.RedactionsTotal)
	require.NotNil(t, meta.Statistics.RedactionsByType)
	assert.Empty(t, meta.Statistics.RedactionsByType)

	// JSON must produce {} not null.
	data, err := json.Marshal(meta.Statistics)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"redactions_by_type":{}`)
	assert.Contains(t, string(data), `"redactions_total":0`)
}

// ---------------------------------------------------------------------------
// 8. TestGenerateMetadata_EmptyMaps
// FilesByTier and RedactionsByType are empty maps not nil.
// JSON should be {} not null.
// ---------------------------------------------------------------------------

func TestGenerateMetadata_EmptyMaps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		tierCounts       map[int]int
		redactionSummary map[string]int
	}{
		{
			name:             "nil maps",
			tierCounts:       nil,
			redactionSummary: nil,
		},
		{
			name:             "empty maps",
			tierCounts:       map[int]int{},
			redactionSummary: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := testMetadataOpts()
			opts.RenderData.TierCounts = tt.tierCounts
			opts.RenderData.RedactionSummary = tt.redactionSummary
			opts.RenderData.TotalRedactions = 0

			meta := GenerateMetadata(opts)

			require.NotNil(t, meta.Statistics.FilesByTier,
				"files_by_tier must be non-nil for JSON {}")
			assert.Empty(t, meta.Statistics.FilesByTier)
			require.NotNil(t, meta.Statistics.RedactionsByType,
				"redactions_by_type must be non-nil for JSON {}")
			assert.Empty(t, meta.Statistics.RedactionsByType)

			// Verify JSON serialization produces {} not null.
			data, err := json.Marshal(meta)
			require.NoError(t, err)
			jsonStr := string(data)

			assert.Contains(t, jsonStr, `"files_by_tier":{}`)
			assert.Contains(t, jsonStr, `"redactions_by_type":{}`)
			assert.NotContains(t, jsonStr, `"files_by_tier":null`)
			assert.NotContains(t, jsonStr, `"redactions_by_type":null`)
		})
	}
}

// ---------------------------------------------------------------------------
// 9. TestWriteMetadata_ValidJSON
// Writes valid JSON, parse it back, verify fields.
// ---------------------------------------------------------------------------

func TestWriteMetadata_ValidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	opts := testMetadataOpts()
	meta := GenerateMetadata(opts)

	err := WriteMetadata(meta, outPath)
	require.NoError(t, err)

	sidecarPath := MetadataSidecarPath(outPath)
	data, err := os.ReadFile(sidecarPath)
	require.NoError(t, err)

	// Parse it back.
	var parsed OutputMetadata
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "written metadata must be valid JSON")

	// Verify key fields survived the write.
	assert.Equal(t, meta.Version, parsed.Version)
	assert.Equal(t, meta.Profile, parsed.Profile)
	assert.Equal(t, meta.Tokenizer, parsed.Tokenizer)
	assert.Equal(t, meta.Format, parsed.Format)
	assert.Equal(t, meta.Target, parsed.Target)
	assert.Equal(t, meta.ContentHash, parsed.ContentHash)
	assert.Equal(t, meta.Statistics.TotalFiles, parsed.Statistics.TotalFiles)
	assert.Equal(t, meta.Statistics.TotalTokens, parsed.Statistics.TotalTokens)
	assert.Len(t, parsed.Files, len(meta.Files))
}

// ---------------------------------------------------------------------------
// 10. TestWriteMetadata_PrettyPrinted
// Output has 2-space indentation.
// ---------------------------------------------------------------------------

func TestWriteMetadata_PrettyPrinted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	meta := GenerateMetadata(testMetadataOpts())
	err := WriteMetadata(meta, outPath)
	require.NoError(t, err)

	content, err := os.ReadFile(MetadataSidecarPath(outPath))
	require.NoError(t, err)

	output := string(content)
	lines := strings.Split(output, "\n")

	// Verify multi-line output (not compact single-line).
	assert.Greater(t, len(lines), 1, "pretty-printed JSON should be multi-line")

	// Verify 2-space indentation is present.
	foundTwoSpace := false
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") {
			foundTwoSpace = true
			break
		}
	}
	assert.True(t, foundTwoSpace, "output should have lines with 2-space indent prefix")

	// Verify no tab indentation.
	for _, line := range lines {
		assert.False(t, strings.HasPrefix(line, "\t"),
			"output should not use tab indentation, found: %q", line)
	}

	// Verify specific indented fields.
	assert.Contains(t, output, "\n  \"version\"")
	assert.Contains(t, output, "\n    \"total_files\"")

	// Verify trailing newline (POSIX compliance).
	assert.True(t, strings.HasSuffix(output, "}\n"))
}

// ---------------------------------------------------------------------------
// 11. TestWriteMetadata_AtomicWrite
// Verify no temp files remain after write.
// ---------------------------------------------------------------------------

func TestWriteMetadata_AtomicWrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	meta := GenerateMetadata(testMetadataOpts())
	err := WriteMetadata(meta, outPath)
	require.NoError(t, err)

	// Verify no temp files remain.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, e := range entries {
		assert.False(t, strings.HasPrefix(e.Name(), ".harvx-meta-"),
			"temp file should be cleaned up: %s", e.Name())
		assert.False(t, strings.HasSuffix(e.Name(), ".tmp"),
			"temp file should be cleaned up: %s", e.Name())
	}

	// Verify only the sidecar file exists.
	assert.Len(t, entries, 1)
	assert.Equal(t, "output.md.meta.json", entries[0].Name())
}

// ---------------------------------------------------------------------------
// 12. TestWriteMetadata_SidecarPath
// Verify path is <output>.meta.json.
// ---------------------------------------------------------------------------

func TestWriteMetadata_SidecarPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	meta := GenerateMetadata(testMetadataOpts())
	err := WriteMetadata(meta, outPath)
	require.NoError(t, err)

	expectedPath := outPath + ".meta.json"
	_, err = os.Stat(expectedPath)
	require.NoError(t, err, "sidecar file should exist at %s", expectedPath)
}

// ---------------------------------------------------------------------------
// 13. TestMetadataSidecarPath (table-driven)
// ---------------------------------------------------------------------------

func TestMetadataSidecarPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		outputPath string
		want       string
	}{
		{
			name:       "output.md -> output.md.meta.json",
			outputPath: "output.md",
			want:       "output.md.meta.json",
		},
		{
			name:       "my.project.output.md -> my.project.output.md.meta.json",
			outputPath: "my.project.output.md",
			want:       "my.project.output.md.meta.json",
		},
		{
			name:       "/path/to/output.xml -> /path/to/output.xml.meta.json",
			outputPath: "/path/to/output.xml",
			want:       "/path/to/output.xml.meta.json",
		},
		{
			name:       "no extension",
			outputPath: "output",
			want:       "output.meta.json",
		},
		{
			name:       "harvx-output.md",
			outputPath: "harvx-output.md",
			want:       "harvx-output.md.meta.json",
		},
		{
			name:       "nested directory",
			outputPath: "/home/user/project/dist/harvx-output.md",
			want:       "/home/user/project/dist/harvx-output.md.meta.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := MetadataSidecarPath(tt.outputPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// 14. TestWriteMetadata_OutputDirNotExist
// Error when output dir doesn't exist.
// ---------------------------------------------------------------------------

func TestWriteMetadata_OutputDirNotExist(t *testing.T) {
	t.Parallel()

	meta := GenerateMetadata(testMetadataOpts())

	err := WriteMetadata(meta, "/nonexistent/dir/output.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "writing metadata")
}

// ---------------------------------------------------------------------------
// 15. TestGenerateMetadata_LargeFileList
// 10K+ files produces valid metadata without truncation.
// ---------------------------------------------------------------------------

func TestGenerateMetadata_LargeFileList(t *testing.T) {
	t.Parallel()

	const numFiles = 10001

	opts := testMetadataOpts()

	files := make([]FileRenderEntry, numFiles)
	tierCounts := make(map[int]int)
	for i := range files {
		tier := i % 6
		files[i] = FileRenderEntry{
			Path:         fmt.Sprintf("pkg/sub%03d/file_%05d.go", i%100, i),
			Size:         int64(100 + i),
			TokenCount:   50 + i%100,
			Tier:         tier,
			Language:     "go",
			IsCompressed: i%3 == 0,
			Redactions:   i % 5,
		}
		tierCounts[tier]++
	}
	opts.RenderData.Files = files
	opts.RenderData.TotalFiles = numFiles
	opts.RenderData.TierCounts = tierCounts

	meta := GenerateMetadata(opts)

	// All files present, no truncation.
	assert.Len(t, meta.Files, numFiles)

	// Files are sorted.
	for i := 1; i < len(meta.Files); i++ {
		assert.True(t, meta.Files[i-1].Path <= meta.Files[i].Path,
			"files should be sorted: %q > %q at index %d",
			meta.Files[i-1].Path, meta.Files[i].Path, i)
	}

	// Verify JSON is valid and round-trips.
	data, err := json.Marshal(meta)
	require.NoError(t, err)
	assert.Greater(t, len(data), 0)

	var parsed OutputMetadata
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Len(t, parsed.Files, numFiles, "all files must survive JSON round-trip")

	// Also write to disk and read back.
	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")
	require.NoError(t, WriteMetadata(meta, outPath))

	diskData, err := os.ReadFile(MetadataSidecarPath(outPath))
	require.NoError(t, err)

	var diskParsed OutputMetadata
	require.NoError(t, json.Unmarshal(diskData, &diskParsed))
	assert.Len(t, diskParsed.Files, numFiles, "all files must survive disk round-trip")
}

// ---------------------------------------------------------------------------
// 16. TestWriteMetadata_RoundTrip
// Generate -> write -> read -> unmarshal -> compare.
// ---------------------------------------------------------------------------

func TestWriteMetadata_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	opts := testMetadataOpts()
	original := GenerateMetadata(opts)

	// Write.
	err := WriteMetadata(original, outPath)
	require.NoError(t, err)

	// Read.
	content, err := os.ReadFile(MetadataSidecarPath(outPath))
	require.NoError(t, err)

	// Unmarshal.
	var parsed OutputMetadata
	require.NoError(t, json.Unmarshal(content, &parsed))

	// Compare top-level fields.
	assert.Equal(t, original.Version, parsed.Version)
	assert.Equal(t, original.GeneratedAt, parsed.GeneratedAt)
	assert.Equal(t, original.Profile, parsed.Profile)
	assert.Equal(t, original.Tokenizer, parsed.Tokenizer)
	assert.Equal(t, original.Format, parsed.Format)
	assert.Equal(t, original.Target, parsed.Target)
	assert.Equal(t, original.ContentHash, parsed.ContentHash)

	// Compare statistics.
	assert.Equal(t, original.Statistics.TotalFiles, parsed.Statistics.TotalFiles)
	assert.Equal(t, original.Statistics.TotalTokens, parsed.Statistics.TotalTokens)
	assert.Equal(t, original.Statistics.TotalBytes, parsed.Statistics.TotalBytes)
	assert.Equal(t, original.Statistics.MaxTokens, parsed.Statistics.MaxTokens)
	assert.Equal(t, original.Statistics.RedactionsTotal, parsed.Statistics.RedactionsTotal)
	assert.Equal(t, original.Statistics.CompressedFiles, parsed.Statistics.CompressedFiles)
	assert.Equal(t, original.Statistics.GenerationTimeMs, parsed.Statistics.GenerationTimeMs)

	require.NotNil(t, parsed.Statistics.BudgetUsedPercent)
	assert.InDelta(t, *original.Statistics.BudgetUsedPercent,
		*parsed.Statistics.BudgetUsedPercent, 0.001)

	assert.Equal(t, original.Statistics.FilesByTier, parsed.Statistics.FilesByTier)
	assert.Equal(t, original.Statistics.RedactionsByType, parsed.Statistics.RedactionsByType)

	// Compare files.
	require.Len(t, parsed.Files, len(original.Files))
	for i := range original.Files {
		assert.Equal(t, original.Files[i].Path, parsed.Files[i].Path)
		assert.Equal(t, original.Files[i].Tier, parsed.Files[i].Tier)
		assert.Equal(t, original.Files[i].Tokens, parsed.Files[i].Tokens)
		assert.Equal(t, original.Files[i].Bytes, parsed.Files[i].Bytes)
		assert.Equal(t, original.Files[i].Redactions, parsed.Files[i].Redactions)
		assert.Equal(t, original.Files[i].Compressed, parsed.Files[i].Compressed)
		assert.Equal(t, original.Files[i].Language, parsed.Files[i].Language)
	}
}

// ---------------------------------------------------------------------------
// 17. TestGenerateMetadata_GenerationTime
// Verify generation_time_ms is set from opts.
// ---------------------------------------------------------------------------

func TestGenerateMetadata_GenerationTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		generationTimeMs int64
	}{
		{name: "typical time", generationTimeMs: 850},
		{name: "zero time", generationTimeMs: 0},
		{name: "very large time", generationTimeMs: 999999},
		{name: "one millisecond", generationTimeMs: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := testMetadataOpts()
			opts.GenerationTimeMs = tt.generationTimeMs

			meta := GenerateMetadata(opts)
			assert.Equal(t, tt.generationTimeMs, meta.Statistics.GenerationTimeMs)
		})
	}
}

// ---------------------------------------------------------------------------
// Additional coverage tests
// ---------------------------------------------------------------------------

// TestGenerateMetadata_ContentHashFromResult verifies that the content hash
// is taken from OutputResult.HashHex, not from RenderData.ContentHash.
func TestGenerateMetadata_ContentHashFromResult(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	opts.RenderData.ContentHash = "renderdatahash"
	opts.Result.HashHex = "outputresulthash"

	meta := GenerateMetadata(opts)
	assert.Equal(t, "outputresulthash", meta.ContentHash)
}

// TestGenerateMetadata_GeneratedAtRFC3339 verifies the timestamp is
// correctly formatted as RFC 3339 from the Timestamp field.
func TestGenerateMetadata_GeneratedAtRFC3339(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	meta := GenerateMetadata(opts)

	parsed, err := time.Parse(time.RFC3339, meta.GeneratedAt)
	require.NoError(t, err)
	assert.Equal(t, opts.RenderData.Timestamp.UTC(), parsed.UTC())
}

// TestGenerateMetadata_TotalBytesComputed verifies TotalBytes is the sum
// of all file sizes.
func TestGenerateMetadata_TotalBytesComputed(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	meta := GenerateMetadata(opts)

	var expectedTotal int64
	for _, f := range opts.RenderData.Files {
		expectedTotal += f.Size
	}
	assert.Equal(t, expectedTotal, meta.Statistics.TotalBytes)
}

// TestGenerateMetadata_CompressedFilesCount verifies the compressed files
// counter is computed from IsCompressed flags.
func TestGenerateMetadata_CompressedFilesCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		files          []FileRenderEntry
		wantCompressed int
	}{
		{
			name: "no compressed files",
			files: []FileRenderEntry{
				{Path: "a.go", IsCompressed: false},
				{Path: "b.go", IsCompressed: false},
			},
			wantCompressed: 0,
		},
		{
			name: "all compressed",
			files: []FileRenderEntry{
				{Path: "a.go", IsCompressed: true},
				{Path: "b.go", IsCompressed: true},
			},
			wantCompressed: 2,
		},
		{
			name: "mixed",
			files: []FileRenderEntry{
				{Path: "a.go", IsCompressed: true},
				{Path: "b.go", IsCompressed: false},
				{Path: "c.go", IsCompressed: true},
			},
			wantCompressed: 2,
		},
		{
			name:           "empty list",
			files:          []FileRenderEntry{},
			wantCompressed: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := testMetadataOpts()
			opts.RenderData.Files = tt.files
			opts.RenderData.TotalFiles = len(tt.files)

			meta := GenerateMetadata(opts)
			assert.Equal(t, tt.wantCompressed, meta.Statistics.CompressedFiles)
		})
	}
}

// TestGenerateMetadata_JSONSnakeCaseFields verifies all JSON field names
// use snake_case.
func TestGenerateMetadata_JSONSnakeCaseFields(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	meta := GenerateMetadata(opts)

	data, err := json.Marshal(meta)
	require.NoError(t, err)
	jsonStr := string(data)

	expectedFields := []string{
		`"version"`, `"generated_at"`, `"profile"`, `"tokenizer"`,
		`"format"`, `"target"`, `"content_hash"`, `"statistics"`,
		`"total_files"`, `"total_tokens"`, `"total_bytes"`,
		`"budget_used_percent"`, `"max_tokens"`, `"files_by_tier"`,
		`"redactions_total"`, `"redactions_by_type"`,
		`"compressed_files"`, `"generation_time_ms"`,
		`"files"`, `"path"`, `"tier"`, `"tokens"`, `"bytes"`,
		`"redactions"`, `"compressed"`, `"language"`,
	}
	for _, field := range expectedFields {
		assert.Contains(t, jsonStr, field, "missing JSON field: %s", field)
	}
}

// TestGenerateMetadata_BudgetUsedPercentNull verifies that the JSON output
// contains null for budget_used_percent when MaxTokens is 0.
func TestGenerateMetadata_BudgetUsedPercentNull(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	opts.MaxTokens = 0
	meta := GenerateMetadata(opts)

	data, err := json.Marshal(meta)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"budget_used_percent":null`)
}

// TestGenerateMetadata_FilesByTierStringKeys verifies that the FilesByTier map
// uses string keys in JSON (e.g., "0", "1") not integer keys.
func TestGenerateMetadata_FilesByTierStringKeys(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	opts.RenderData.TierCounts = map[int]int{0: 5, 1: 48, 2: 180, 3: 62, 4: 30, 5: 17}

	meta := GenerateMetadata(opts)

	expected := map[string]int{
		"0": 5, "1": 48, "2": 180, "3": 62, "4": 30, "5": 17,
	}
	assert.Equal(t, expected, meta.Statistics.FilesByTier)

	// Verify JSON uses quoted string keys.
	data, err := json.Marshal(meta.Statistics.FilesByTier)
	require.NoError(t, err)
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"0"`)
	assert.Contains(t, jsonStr, `"5"`)
}

// TestGenerateMetadata_FilesSortedWithNestedPaths verifies sorting with
// mixed-depth paths.
func TestGenerateMetadata_FilesSortedWithNestedPaths(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	opts.RenderData.Files = []FileRenderEntry{
		{Path: "src/z/deep.go"},
		{Path: "README.md"},
		{Path: "internal/cli/root.go"},
		{Path: "cmd/main.go"},
		{Path: "a.go"},
	}
	opts.RenderData.TotalFiles = 5

	meta := GenerateMetadata(opts)

	paths := make([]string, len(meta.Files))
	for i, f := range meta.Files {
		paths[i] = f.Path
	}

	expected := []string{
		"README.md",
		"a.go",
		"cmd/main.go",
		"internal/cli/root.go",
		"src/z/deep.go",
	}
	assert.Equal(t, expected, paths)
}

// TestGenerateMetadata_FormatAndTargetPassthrough verifies format and target
// are passed through from opts to metadata.
func TestGenerateMetadata_FormatAndTargetPassthrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format string
		target string
	}{
		{name: "markdown/claude", format: "markdown", target: "claude"},
		{name: "xml/gpt4", format: "xml", target: "gpt-4"},
		{name: "markdown/empty target", format: "markdown", target: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := testMetadataOpts()
			opts.Format = tt.format
			opts.Target = tt.target

			meta := GenerateMetadata(opts)
			assert.Equal(t, tt.format, meta.Format)
			assert.Equal(t, tt.target, meta.Target)
		})
	}
}

// TestGenerateMetadata_MaxTokensInStatistics verifies MaxTokens is included.
func TestGenerateMetadata_MaxTokensInStatistics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		maxTokens int
	}{
		{name: "zero", maxTokens: 0},
		{name: "small", maxTokens: 100},
		{name: "large", maxTokens: 200000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := testMetadataOpts()
			opts.MaxTokens = tt.maxTokens

			meta := GenerateMetadata(opts)
			assert.Equal(t, tt.maxTokens, meta.Statistics.MaxTokens)
		})
	}
}

// TestWriteMetadata_MultipleDots verifies sidecar naming with multiple dots
// in the output filename.
func TestWriteMetadata_MultipleDots(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "my.project.output.md")

	meta := GenerateMetadata(testMetadataOpts())

	err := WriteMetadata(meta, outPath)
	require.NoError(t, err)

	expectedSidecar := filepath.Join(dir, "my.project.output.md.meta.json")
	_, err = os.Stat(expectedSidecar)
	require.NoError(t, err, "sidecar file should exist at %s", expectedSidecar)
}

// TestWriteMetadata_Overwrite verifies that writing metadata to the same
// path overwrites the previous content.
func TestWriteMetadata_Overwrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")
	sidecarPath := MetadataSidecarPath(outPath)

	// Write initial metadata.
	opts1 := testMetadataOpts()
	opts1.RenderData.ProfileName = "initial"
	meta1 := GenerateMetadata(opts1)
	require.NoError(t, WriteMetadata(meta1, outPath))

	// Write updated metadata.
	opts2 := testMetadataOpts()
	opts2.RenderData.ProfileName = "updated"
	meta2 := GenerateMetadata(opts2)
	require.NoError(t, WriteMetadata(meta2, outPath))

	// Read and verify the overwritten file.
	content, err := os.ReadFile(sidecarPath)
	require.NoError(t, err)

	var parsed OutputMetadata
	require.NoError(t, json.Unmarshal(content, &parsed))
	assert.Equal(t, "updated", parsed.Profile)
}

// TestMetadataVersion verifies the constant value.
func TestMetadataVersion(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.0.0", MetadataVersion)
}

// TestGenerateMetadata_MultipleRedactionTypes verifies all redaction types
// are correctly propagated.
func TestGenerateMetadata_MultipleRedactionTypes(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	opts.RenderData.TotalRedactions = 10
	opts.RenderData.RedactionSummary = map[string]int{
		"aws_access_key":    2,
		"connection_string": 1,
		"api_key":           5,
		"private_key":       2,
	}

	meta := GenerateMetadata(opts)

	assert.Equal(t, 10, meta.Statistics.RedactionsTotal)
	assert.Len(t, meta.Statistics.RedactionsByType, 4)
	assert.Equal(t, 2, meta.Statistics.RedactionsByType["aws_access_key"])
	assert.Equal(t, 1, meta.Statistics.RedactionsByType["connection_string"])
	assert.Equal(t, 5, meta.Statistics.RedactionsByType["api_key"])
	assert.Equal(t, 2, meta.Statistics.RedactionsByType["private_key"])
}

// TestGenerateMetadata_FilesPreserveLanguage verifies language values are
// correctly mapped including empty strings.
func TestGenerateMetadata_FilesPreserveLanguage(t *testing.T) {
	t.Parallel()

	opts := testMetadataOpts()
	opts.RenderData.Files = []FileRenderEntry{
		{Path: "main.go", Language: "go"},
		{Path: "app.py", Language: "python"},
		{Path: "unknown.xyz", Language: ""},
	}
	opts.RenderData.TotalFiles = 3

	meta := GenerateMetadata(opts)

	langMap := make(map[string]string)
	for _, f := range meta.Files {
		langMap[f.Path] = f.Language
	}

	assert.Equal(t, "go", langMap["main.go"])
	assert.Equal(t, "python", langMap["app.py"])
	assert.Equal(t, "", langMap["unknown.xyz"])
}

// ---------------------------------------------------------------------------
// Benchmark tests
// ---------------------------------------------------------------------------

func BenchmarkGenerateMetadata(b *testing.B) {
	files := make([]FileRenderEntry, 1000)
	for i := 0; i < 1000; i++ {
		files[i] = FileRenderEntry{
			Path:         filepath.Join("src", fmt.Sprintf("pkg%d", i%26), "file.go"),
			Size:         int64(100 + i%500),
			TokenCount:   50 + i%200,
			Tier:         i % 6,
			Language:     "go",
			IsCompressed: i%3 == 0,
			Redactions:   i % 5,
		}
	}

	opts := MetadataOpts{
		RenderData: &RenderData{
			Timestamp:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			ProfileName:      "default",
			TokenizerName:    "cl100k_base",
			TotalTokens:      150000,
			TotalFiles:       1000,
			Files:            files,
			TierCounts:       map[int]int{0: 100, 1: 200, 2: 300, 3: 200, 4: 100, 5: 100},
			TotalRedactions:  500,
			RedactionSummary: map[string]int{"api_key": 300, "password": 200},
		},
		Result: &OutputResult{
			HashHex: "abcdef0123456789",
		},
		Format:           "markdown",
		Target:           "claude",
		MaxTokens:        200000,
		GenerationTimeMs: 850,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateMetadata(opts)
	}
}

func BenchmarkWriteMetadata(b *testing.B) {
	meta := GenerateMetadata(testMetadataOpts())
	dir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outPath := filepath.Join(dir, fmt.Sprintf("output_%d.md", i))
		_ = WriteMetadata(meta, outPath)
	}
}
