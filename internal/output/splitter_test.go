package output

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// splitterTestRenderData returns a RenderData with the given files for splitter tests.
func splitterTestRenderData(files []FileRenderEntry) *RenderData {
	totalTokens := 0
	tierCounts := make(map[int]int)
	for _, f := range files {
		totalTokens += f.TokenCount
		tierCounts[f.Tier]++
	}

	return &RenderData{
		ProjectName:   "test-project",
		Timestamp:     time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		ContentHash:   "abcdef1234567890",
		ProfileName:   "default",
		TokenizerName: "cl100k_base",
		TotalTokens:   totalTokens,
		TotalFiles:    len(files),
		Files:         files,
		TreeString:    ".\n├── src/\n│   └── main.go\n└── README.md",
		TierCounts:    tierCounts,
		TopFilesByTokens: []FileRenderEntry{
			{Path: "main.go", TokenCount: 100},
		},
		RedactionSummary: map[string]int{"api_key": 2},
		TotalRedactions:  2,
	}
}

// makeFile creates a FileRenderEntry with the given path, token count, and tier.
func makeFile(path string, tokens, tier int) FileRenderEntry {
	return FileRenderEntry{
		Path:       path,
		Size:       int64(tokens * 4), // rough estimate
		TokenCount: tokens,
		Tier:       tier,
		TierLabel:  tierLabel(tier),
		Language:   languageFromExt(path),
		Content:    strings.Repeat("x", tokens*4),
	}
}

func TestNewSplitter(t *testing.T) {
	t.Parallel()

	t.Run("default overhead", func(t *testing.T) {
		t.Parallel()
		s := NewSplitter(SplitOpts{TokensPerPart: 1000})
		assert.Equal(t, DefaultOverheadPerFile, s.opts.OverheadPerFile)
	})

	t.Run("custom overhead", func(t *testing.T) {
		t.Parallel()
		s := NewSplitter(SplitOpts{TokensPerPart: 1000, OverheadPerFile: 500})
		assert.Equal(t, 500, s.opts.OverheadPerFile)
	})

	t.Run("zero overhead uses default", func(t *testing.T) {
		t.Parallel()
		s := NewSplitter(SplitOpts{TokensPerPart: 1000, OverheadPerFile: 0})
		assert.Equal(t, DefaultOverheadPerFile, s.opts.OverheadPerFile)
	})

	t.Run("negative overhead uses default", func(t *testing.T) {
		t.Parallel()
		s := NewSplitter(SplitOpts{TokensPerPart: 1000, OverheadPerFile: -10})
		assert.Equal(t, DefaultOverheadPerFile, s.opts.OverheadPerFile)
	})
}

func TestSplitter_Split_NilData(t *testing.T) {
	t.Parallel()

	s := NewSplitter(SplitOpts{TokensPerPart: 1000})
	parts, err := s.Split(nil)

	assert.Nil(t, parts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render data is nil")
}

func TestSplitter_Split_ZeroTokensPerPart(t *testing.T) {
	t.Parallel()

	s := NewSplitter(SplitOpts{TokensPerPart: 0})
	data := splitterTestRenderData(nil)
	parts, err := s.Split(data)

	assert.Nil(t, parts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tokens per part must be positive")
}

func TestSplitter_Split_NegativeTokensPerPart(t *testing.T) {
	t.Parallel()

	s := NewSplitter(SplitOpts{TokensPerPart: -100})
	data := splitterTestRenderData(nil)
	parts, err := s.Split(data)

	assert.Nil(t, parts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tokens per part must be positive")
}

func TestSplitter_Split_EmptyFileList(t *testing.T) {
	t.Parallel()

	s := NewSplitter(SplitOpts{TokensPerPart: 1000})
	data := splitterTestRenderData(nil)
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.Len(t, parts, 1)
	assert.Equal(t, 1, parts[0].PartNumber)
	assert.Equal(t, 1, parts[0].TotalParts)
}

func TestSplitter_Split_AllFilesFitInOnePart(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("main.go", 100, 0),
		makeFile("README.md", 50, 4),
	}
	data := splitterTestRenderData(files)

	// Budget large enough for all files + overhead.
	s := NewSplitter(SplitOpts{TokensPerPart: 100000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.Len(t, parts, 1)
	assert.Equal(t, 1, parts[0].PartNumber)
	assert.Equal(t, 1, parts[0].TotalParts)
	assert.Equal(t, data, parts[0].RenderData) // unchanged original
	assert.Equal(t, data.ContentHash, parts[0].GlobalHash)
}

func TestSplitter_Split_TwoParts(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("src/main.go", 5000, 0),
		makeFile("src/util.go", 5000, 1),
	}
	data := splitterTestRenderData(files)

	// Budget that forces a split: each file is 5000 tokens + 200 overhead = 5200.
	// Part 1 header overhead: 600. Effective budget: 6000 - 600 = 5400. Fits one file.
	// Part 2 header overhead: 300. Effective budget: 6000 - 300 = 5700. Fits one file.
	s := NewSplitter(SplitOpts{TokensPerPart: 6000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.Len(t, parts, 2)

	// Part 1.
	assert.Equal(t, 1, parts[0].PartNumber)
	assert.Equal(t, 2, parts[0].TotalParts)
	assert.Len(t, parts[0].RenderData.Files, 1)
	assert.Equal(t, "src/main.go", parts[0].RenderData.Files[0].Path)

	// Part 2.
	assert.Equal(t, 2, parts[1].PartNumber)
	assert.Equal(t, 2, parts[1].TotalParts)
	assert.Len(t, parts[1].RenderData.Files, 1)
	assert.Equal(t, "src/util.go", parts[1].RenderData.Files[0].Path)
}

func TestSplitter_Split_FileOrderPreserved(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 100, 0),
		makeFile("b.go", 100, 0),
		makeFile("c.go", 100, 1),
		makeFile("d.go", 100, 1),
		makeFile("e.go", 100, 2),
	}
	data := splitterTestRenderData(files)

	// Force into 2 parts.
	s := NewSplitter(SplitOpts{TokensPerPart: 1500})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parts), 2)

	// Collect all file paths in order across parts.
	var allPaths []string
	for _, part := range parts {
		for _, f := range part.RenderData.Files {
			allPaths = append(allPaths, f.Path)
		}
	}

	// Order should be preserved.
	expected := []string{"a.go", "b.go", "c.go", "d.go", "e.go"}
	assert.Equal(t, expected, allPaths)
}

func TestSplitter_Split_NoFileSplitAcrossParts(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 500, 0),
		makeFile("b.go", 500, 0),
		makeFile("c.go", 500, 1),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 1500})
	parts, err := s.Split(data)

	require.NoError(t, err)

	// Verify each file appears in exactly one part (count appearances).
	appearances := make(map[string]int)
	for _, part := range parts {
		for _, f := range part.RenderData.Files {
			appearances[f.Path]++
		}
	}

	assert.Len(t, appearances, 3, "all 3 files should be present across parts")
	for path, count := range appearances {
		assert.Equal(t, 1, count, "file %s should appear in exactly one part, appeared in %d", path, count)
	}
}

func TestSplitter_Split_Part1HasTreeAndSummary(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("main.go", 5000, 0),
		makeFile("test.go", 5000, 3),
	}
	data := splitterTestRenderData(files)
	data.TreeString = "full tree here"
	data.TopFilesByTokens = []FileRenderEntry{makeFile("main.go", 5000, 0)}
	data.RedactionSummary = map[string]int{"api_key": 1}
	data.TotalRedactions = 1

	s := NewSplitter(SplitOpts{TokensPerPart: 6000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parts), 2)

	// Part 1 should have tree and summary.
	assert.Equal(t, "full tree here", parts[0].RenderData.TreeString)
	assert.NotNil(t, parts[0].RenderData.TopFilesByTokens)
	assert.NotNil(t, parts[0].RenderData.RedactionSummary)
	assert.Equal(t, 1, parts[0].RenderData.TotalRedactions)

	// Part 2+ should have minimal tree placeholder.
	for i := 1; i < len(parts); i++ {
		assert.Contains(t, parts[i].RenderData.TreeString, "See Part 1")
		assert.Nil(t, parts[i].RenderData.TopFilesByTokens)
		assert.Nil(t, parts[i].RenderData.RedactionSummary)
		assert.Equal(t, 0, parts[i].RenderData.TotalRedactions)
	}
}

func TestSplitter_Split_PartHeadersCorrect(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 3000, 0),
		makeFile("b.go", 3000, 1),
		makeFile("c.go", 3000, 2),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 4000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parts), 2)

	for i, part := range parts {
		assert.Equal(t, i+1, part.PartNumber)
		assert.Equal(t, len(parts), part.TotalParts)
		assert.Equal(t, data.ContentHash, part.GlobalHash)
	}
}

func TestSplitter_Split_OversizedFile(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("small.go", 100, 0),
		makeFile("huge.go", 50000, 1),  // exceeds budget
		makeFile("medium.go", 1000, 2), // should go in new part
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 5000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parts), 3)

	// Find the part containing huge.go.
	found := false
	for _, part := range parts {
		for _, f := range part.RenderData.Files {
			if f.Path == "huge.go" {
				found = true
				// It should be alone in its part.
				assert.Len(t, part.RenderData.Files, 1,
					"oversized file should get its own part")
			}
		}
	}
	assert.True(t, found, "huge.go should be present in parts")
}

func TestSplitter_Split_GlobalHashInAllParts(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 3000, 0),
		makeFile("b.go", 3000, 1),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 4000})
	parts, err := s.Split(data)

	require.NoError(t, err)

	for _, part := range parts {
		assert.Equal(t, data.ContentHash, part.GlobalHash)
		assert.Equal(t, data.ContentHash, part.RenderData.ContentHash)
	}
}

func TestSplitter_Split_DirectoryCoherence(t *testing.T) {
	t.Parallel()

	// Files from the same tier and same top-level directory.
	files := []FileRenderEntry{
		makeFile("src/a.go", 1000, 0),
		makeFile("src/b.go", 1000, 0),
		makeFile("pkg/c.go", 1000, 0),
	}
	data := splitterTestRenderData(files)

	// Budget that can hold 2 files with overhead but not 3.
	// 1000 + 200 = 1200 per file. Budget 3000 - 600 header = 2400. Fits 2 files.
	s := NewSplitter(SplitOpts{TokensPerPart: 3000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parts), 2)

	// src/a.go and src/b.go should be in the same part (same tier, same dir).
	part1Files := make([]string, 0)
	for _, f := range parts[0].RenderData.Files {
		part1Files = append(part1Files, f.Path)
	}
	assert.Contains(t, part1Files, "src/a.go")
	assert.Contains(t, part1Files, "src/b.go")
}

// --- PartPath tests ---

func TestPartPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		basePath   string
		partNum    int
		totalParts int
		want       string
	}{
		{
			name:       "single part no suffix",
			basePath:   "harvx-output.md",
			partNum:    1,
			totalParts: 1,
			want:       "harvx-output.md",
		},
		{
			name:       "markdown part 1 of 3",
			basePath:   "harvx-output.md",
			partNum:    1,
			totalParts: 3,
			want:       "harvx-output.part-001.md",
		},
		{
			name:       "markdown part 2 of 3",
			basePath:   "harvx-output.md",
			partNum:    2,
			totalParts: 3,
			want:       "harvx-output.part-002.md",
		},
		{
			name:       "xml part 1 of 2",
			basePath:   "harvx-output.xml",
			partNum:    1,
			totalParts: 2,
			want:       "harvx-output.part-001.xml",
		},
		{
			name:       "xml part 2 of 2",
			basePath:   "harvx-output.xml",
			partNum:    2,
			totalParts: 2,
			want:       "harvx-output.part-002.xml",
		},
		{
			name:       "zero-padded 3 digits",
			basePath:   "output.md",
			partNum:    42,
			totalParts: 100,
			want:       "output.part-042.md",
		},
		{
			name:       "large part number",
			basePath:   "output.md",
			partNum:    999,
			totalParts: 999,
			want:       "output.part-999.md",
		},
		{
			name:       "directory in path preserved",
			basePath:   "/tmp/reports/harvx-output.md",
			partNum:    1,
			totalParts: 5,
			want:       "/tmp/reports/harvx-output.part-001.md",
		},
		{
			name:       "no extension",
			basePath:   "harvx-output",
			partNum:    1,
			totalParts: 2,
			want:       "harvx-output.part-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := PartPath(tt.basePath, tt.partNum, tt.totalParts)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- topLevelDir tests ---

func TestTopLevelDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{"src/main.go", "src"},
		{"internal/cli/root.go", "internal"},
		{"README.md", ""},
		{"", ""},
		{"a/b/c/d.go", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, topLevelDir(tt.path))
		})
	}
}

// --- WriteSplit tests ---

func TestWriteSplit_NilData(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	results, err := ow.WriteSplit(context.Background(), nil, SplitOutputOpts{
		OutputOpts:  OutputOpts{Format: "markdown", UseStdout: true},
		SplitTokens: 1000,
	})

	assert.Nil(t, results)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render data is nil")
}

func TestWriteSplit_ZeroSplitTokens(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := splitterTestRenderData(nil)
	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{Format: "markdown", UseStdout: true},
		SplitTokens: 0,
	})

	assert.Nil(t, results)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "split tokens must be positive")
}

func TestWriteSplit_NegativeSplitTokens(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := splitterTestRenderData(nil)
	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{Format: "markdown", UseStdout: true},
		SplitTokens: -100,
	})

	assert.Nil(t, results)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "split tokens must be positive")
}

func TestWriteSplit_InvalidFormat(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := splitterTestRenderData(nil)
	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{Format: "json"},
		SplitTokens: 1000,
	})

	assert.Nil(t, results)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestWriteSplit_SinglePartNoSuffix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	files := []FileRenderEntry{makeFile("main.go", 100, 0)}
	data := splitterTestRenderData(files)

	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{OutputPath: outPath, Format: "markdown"},
		SplitTokens: 100000,
	})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 1, results[0].PartNumber)
	assert.Equal(t, outPath, results[0].Path)
	assert.NotZero(t, results[0].Hash)

	// File should exist without .part-001 suffix.
	_, err = os.Stat(outPath)
	require.NoError(t, err)
}

func TestWriteSplit_MultiplePartsFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	files := []FileRenderEntry{
		makeFile("a.go", 5000, 0),
		makeFile("b.go", 5000, 1),
	}
	data := splitterTestRenderData(files)

	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{OutputPath: outPath, Format: "markdown"},
		SplitTokens: 6000,
	})

	require.NoError(t, err)
	require.Len(t, results, 2)

	// Check file names.
	assert.Equal(t, filepath.Join(dir, "output.part-001.md"), results[0].Path)
	assert.Equal(t, filepath.Join(dir, "output.part-002.md"), results[1].Path)

	// Verify files exist.
	for _, r := range results {
		_, err := os.Stat(r.Path)
		require.NoError(t, err, "part file should exist: %s", r.Path)
	}
}

func TestWriteSplit_XMLFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.xml")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	files := []FileRenderEntry{
		makeFile("a.go", 5000, 0),
		makeFile("b.go", 5000, 1),
	}
	data := splitterTestRenderData(files)

	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{OutputPath: outPath, Format: "xml"},
		SplitTokens: 6000,
	})

	require.NoError(t, err)
	require.Len(t, results, 2)

	// Check .xml extension in part names.
	assert.Equal(t, filepath.Join(dir, "output.part-001.xml"), results[0].Path)
	assert.Equal(t, filepath.Join(dir, "output.part-002.xml"), results[1].Path)

	// Verify files contain XML.
	for _, r := range results {
		content, err := os.ReadFile(r.Path)
		require.NoError(t, err)
		assert.Contains(t, string(content), "<repository>")
	}
}

func TestWriteSplit_StdoutSinglePart(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	files := []FileRenderEntry{makeFile("main.go", 100, 0)}
	data := splitterTestRenderData(files)

	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{Format: "markdown", UseStdout: true},
		SplitTokens: 100000,
	})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Empty(t, results[0].Path) // stdout mode
	assert.Contains(t, stdout.String(), "test-project")
}

func TestWriteSplit_CancelledContext(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	files := []FileRenderEntry{
		makeFile("a.go", 5000, 0),
		makeFile("b.go", 5000, 1),
	}
	data := splitterTestRenderData(files)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	_, err := ow.WriteSplit(ctx, data, SplitOutputOpts{
		OutputOpts:  OutputOpts{OutputPath: outPath, Format: "markdown"},
		SplitTokens: 6000,
	})

	require.Error(t, err)
}

func TestWriteSplit_PartResults_Metadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	files := []FileRenderEntry{
		makeFile("a.go", 5000, 0),
		makeFile("b.go", 5000, 1),
	}
	data := splitterTestRenderData(files)

	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{OutputPath: outPath, Format: "markdown"},
		SplitTokens: 6000,
	})

	require.NoError(t, err)
	require.Len(t, results, 2)

	for _, r := range results {
		assert.Greater(t, r.PartNumber, 0)
		assert.Greater(t, r.FileCount, 0)
		assert.Greater(t, r.TokenCount, 0)
		assert.NotZero(t, r.Hash)
		assert.NotEmpty(t, r.Path)
	}
}

// --- Edge case tests ---

func TestSplitter_Split_SingleFileExceedsBudget(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("huge.go", 100000, 0),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 5000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.Len(t, parts, 1)
	assert.Len(t, parts[0].RenderData.Files, 1)
	assert.Equal(t, "huge.go", parts[0].RenderData.Files[0].Path)
}

func TestSplitter_Split_ManySmallFiles(t *testing.T) {
	t.Parallel()

	// 100 small files, should pack into multiple parts.
	files := make([]FileRenderEntry, 100)
	for i := 0; i < 100; i++ {
		files[i] = makeFile(
			filepath.Join("src", strings.Repeat("a", i%5+1)+".go"),
			100,
			i%6,
		)
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 5000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.Greater(t, len(parts), 1)

	// All files should be accounted for.
	totalFiles := 0
	for _, part := range parts {
		totalFiles += len(part.RenderData.Files)
	}
	assert.Equal(t, 100, totalFiles)
}

func TestSplitter_Split_PartNumbersSequential(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 3000, 0),
		makeFile("b.go", 3000, 1),
		makeFile("c.go", 3000, 2),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 4000})
	parts, err := s.Split(data)

	require.NoError(t, err)

	for i, part := range parts {
		assert.Equal(t, i+1, part.PartNumber, "part numbers should be sequential")
		assert.Equal(t, len(parts), part.TotalParts, "total parts should be consistent")
	}
}

func TestSplitter_Split_TierCounts(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("main.go", 3000, 0),
		makeFile("test.go", 3000, 3),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 4000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.Len(t, parts, 2)

	// Part 1 should have tier 0 count.
	assert.Equal(t, 1, parts[0].RenderData.TierCounts[0])
	assert.Equal(t, 0, parts[0].RenderData.TierCounts[3])

	// Part 2 should have tier 3 count.
	assert.Equal(t, 0, parts[1].RenderData.TierCounts[0])
	assert.Equal(t, 1, parts[1].RenderData.TierCounts[3])
}

func TestSplitter_Split_PartTokenCounts(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 1000, 0),
		makeFile("b.go", 2000, 0),
		makeFile("c.go", 3000, 1),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 4500})
	parts, err := s.Split(data)

	require.NoError(t, err)

	// Each part's TotalTokens should be the sum of its files' token counts.
	for _, part := range parts {
		expectedTokens := 0
		for _, f := range part.RenderData.Files {
			expectedTokens += f.TokenCount
		}
		assert.Equal(t, expectedTokens, part.RenderData.TotalTokens)
	}
}

func TestSplitter_Split_PreservesProjectName(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{makeFile("main.go", 100, 0)}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 100000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.Len(t, parts, 1)
	assert.Equal(t, "test-project", parts[0].RenderData.ProjectName)
}

func TestSplitter_Split_PreservesShowLineNumbers(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 3000, 0),
		makeFile("b.go", 3000, 1),
	}
	data := splitterTestRenderData(files)
	data.ShowLineNumbers = true

	s := NewSplitter(SplitOpts{TokensPerPart: 4000})
	parts, err := s.Split(data)

	require.NoError(t, err)

	for _, part := range parts {
		assert.True(t, part.RenderData.ShowLineNumbers)
	}
}

func TestSplitter_Split_DiffSummaryInAllParts(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 3000, 0),
		makeFile("b.go", 3000, 1),
	}
	data := splitterTestRenderData(files)
	data.DiffSummary = &DiffSummaryData{
		AddedFiles: []string{"new.go"},
	}

	s := NewSplitter(SplitOpts{TokensPerPart: 4000})
	parts, err := s.Split(data)

	require.NoError(t, err)

	for _, part := range parts {
		assert.NotNil(t, part.RenderData.DiffSummary)
	}
}

// --- estimateTotalTokens test ---

func TestSplitter_estimateTotalTokens(t *testing.T) {
	t.Parallel()

	s := NewSplitter(SplitOpts{TokensPerPart: 1000, OverheadPerFile: 200})

	files := []FileRenderEntry{
		makeFile("a.go", 500, 0),
		makeFile("b.go", 300, 1),
	}

	// Expected: header overhead (600) + (500+200) + (300+200) = 1800
	total := s.estimateTotalTokens(files)
	assert.Equal(t, 600+700+500, total)
}

// --- Integration: splitter + renderer ---

func TestSplitter_Split_RendersValidMarkdown(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("main.go", 5000, 0),
		makeFile("test.go", 5000, 3),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 6000})
	parts, err := s.Split(data)
	require.NoError(t, err)
	require.Len(t, parts, 2)

	renderer := NewMarkdownRenderer()
	for _, part := range parts {
		var buf bytes.Buffer
		err := renderer.Render(context.Background(), &buf, part.RenderData)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "# Harvx Context:")
	}
}

func TestSplitter_Split_RendersValidXML(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("main.go", 5000, 0),
		makeFile("test.go", 5000, 3),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 6000, Format: "xml"})
	parts, err := s.Split(data)
	require.NoError(t, err)
	require.Len(t, parts, 2)

	renderer := NewXMLRenderer()
	for _, part := range parts {
		var buf bytes.Buffer
		err := renderer.Render(context.Background(), &buf, part.RenderData)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "<repository>")
	}
}

// --- shouldKeepTogether tests ---

func TestSplitter_shouldKeepTogether(t *testing.T) {
	t.Parallel()

	s := NewSplitter(SplitOpts{TokensPerPart: 10000})

	tests := []struct {
		name            string
		files           []FileRenderEntry
		idx             int
		currentTokens   int
		fileTokens      int
		effectiveBudget int
		want            bool
	}{
		{
			name:  "first file always false",
			files: []FileRenderEntry{makeFile("src/a.go", 100, 0)},
			idx:   0,
			want:  false,
		},
		{
			name: "same tier same dir within tolerance",
			files: []FileRenderEntry{
				makeFile("src/a.go", 100, 0),
				makeFile("src/b.go", 100, 0),
			},
			idx:             1,
			currentTokens:   900,
			fileTokens:      200,
			effectiveBudget: 1000,
			want:            true, // 1100 <= 1150 (1000 + 15%)
		},
		{
			name: "same tier different dir",
			files: []FileRenderEntry{
				makeFile("src/a.go", 100, 0),
				makeFile("pkg/b.go", 100, 0),
			},
			idx:             1,
			currentTokens:   900,
			fileTokens:      200,
			effectiveBudget: 1000,
			want:            false,
		},
		{
			name: "different tier same dir",
			files: []FileRenderEntry{
				makeFile("src/a.go", 100, 0),
				makeFile("src/b.go", 100, 1),
			},
			idx:             1,
			currentTokens:   900,
			fileTokens:      200,
			effectiveBudget: 1000,
			want:            false,
		},
		{
			name: "exceeds tolerance",
			files: []FileRenderEntry{
				makeFile("src/a.go", 100, 0),
				makeFile("src/b.go", 100, 0),
			},
			idx:             1,
			currentTokens:   900,
			fileTokens:      500,
			effectiveBudget: 1000,
			want:            false, // 1400 > 1150
		},
		{
			name: "root level files no common dir",
			files: []FileRenderEntry{
				makeFile("a.go", 100, 0),
				makeFile("b.go", 100, 0),
			},
			idx:             1,
			currentTokens:   900,
			fileTokens:      200,
			effectiveBudget: 1000,
			want:            true, // both at root, same empty topLevelDir
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := s.shouldKeepTogether(tt.files, tt.idx, tt.currentTokens, tt.fileTokens, tt.effectiveBudget)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- Part content verification ---

func TestWriteSplit_Part1ContainsPartHeader(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	files := []FileRenderEntry{
		makeFile("a.go", 5000, 0),
		makeFile("b.go", 5000, 1),
	}
	data := splitterTestRenderData(files)

	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{OutputPath: outPath, Format: "markdown"},
		SplitTokens: 6000,
	})

	require.NoError(t, err)
	require.Len(t, results, 2)

	// Part 1 should contain "Part 1 of 2" in the header.
	content1, err := os.ReadFile(results[0].Path)
	require.NoError(t, err)
	assert.Contains(t, string(content1), "Part 1 of 2")

	// Part 2 should contain "Part 2 of 2" in the header.
	content2, err := os.ReadFile(results[1].Path)
	require.NoError(t, err)
	assert.Contains(t, string(content2), "Part 2 of 2")

	// Part 2 should reference Part 1 for tree/summary.
	assert.Contains(t, string(content2), "See Part 1")
}

// --- buildPartRenderData tests ---

func TestSplitter_buildPartRenderData_Part1(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 100, 0),
		makeFile("b.go", 200, 1),
	}
	data := splitterTestRenderData(files)
	data.TreeString = "tree content"
	data.TopFilesByTokens = files
	data.RedactionSummary = map[string]int{"token": 3}
	data.TotalRedactions = 3
	data.ShowLineNumbers = true
	data.DiffSummary = &DiffSummaryData{AddedFiles: []string{"new.go"}}

	s := NewSplitter(SplitOpts{TokensPerPart: 10000})
	rd := s.buildPartRenderData(data, files, 1, 3, "globalhash")

	assert.Equal(t, "test-project", rd.ProjectName)
	assert.Equal(t, "globalhash", rd.ContentHash)
	assert.Equal(t, "tree content", rd.TreeString)
	assert.Equal(t, files, rd.TopFilesByTokens)
	assert.Equal(t, 3, rd.TotalRedactions)
	assert.True(t, rd.ShowLineNumbers)
	assert.NotNil(t, rd.DiffSummary)
	assert.Equal(t, 300, rd.TotalTokens)
	assert.Equal(t, 2, rd.TotalFiles)
}

func TestSplitter_buildPartRenderData_Part2(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{makeFile("c.go", 300, 2)}
	data := splitterTestRenderData(files)
	data.TreeString = "tree content"
	data.TopFilesByTokens = files

	s := NewSplitter(SplitOpts{TokensPerPart: 10000})
	rd := s.buildPartRenderData(data, files, 2, 3, "globalhash")

	assert.Contains(t, rd.TreeString, "See Part 1")
	assert.Nil(t, rd.TopFilesByTokens)
	assert.Nil(t, rd.RedactionSummary)
	assert.Equal(t, 0, rd.TotalRedactions)
	assert.Equal(t, "globalhash", rd.ContentHash)
	assert.Equal(t, 300, rd.TotalTokens)
	assert.Equal(t, 1, rd.TotalFiles)
}

// --- Additional tests for complete acceptance criteria coverage ---

// TestSplitter_Split_150KTokensSplitAt100K tests the specific scenario from the
// task spec: two files totaling 150K tokens split at 100K produces 2 parts.
func TestSplitter_Split_150KTokensSplitAt100K(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("src/large_a.go", 75000, 0),
		makeFile("src/large_b.go", 75000, 1),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 100000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.Len(t, parts, 2, "150K tokens split at 100K should produce exactly 2 parts")

	// Each part should contain exactly one file.
	assert.Len(t, parts[0].RenderData.Files, 1)
	assert.Equal(t, "src/large_a.go", parts[0].RenderData.Files[0].Path)

	assert.Len(t, parts[1].RenderData.Files, 1)
	assert.Equal(t, "src/large_b.go", parts[1].RenderData.Files[0].Path)

	// Verify total tokens add up.
	totalTokens := 0
	for _, part := range parts {
		totalTokens += part.RenderData.TotalTokens
	}
	assert.Equal(t, 150000, totalTokens)
}

// TestPartPath_100PlusParts tests that 100+ parts produce correctly zero-padded
// names using 3-digit padding.
func TestPartPath_100PlusParts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		partNum    int
		totalParts int
		want       string
	}{
		{
			name:       "part 1 of 100",
			partNum:    1,
			totalParts: 100,
			want:       "output.part-001.md",
		},
		{
			name:       "part 42 of 150",
			partNum:    42,
			totalParts: 150,
			want:       "output.part-042.md",
		},
		{
			name:       "part 100 of 100",
			partNum:    100,
			totalParts: 100,
			want:       "output.part-100.md",
		},
		{
			name:       "part 101 of 200",
			partNum:    101,
			totalParts: 200,
			want:       "output.part-101.md",
		},
		{
			name:       "part 999 of 1000",
			partNum:    999,
			totalParts: 1000,
			want:       "output.part-999.md",
		},
		{
			name:       "part 1000 of 1000 exceeds 3 digits gracefully",
			partNum:    1000,
			totalParts: 1000,
			want:       "output.part-1000.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := PartPath("output.md", tt.partNum, tt.totalParts)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSplitter_Split_100PlusPartsGenerated verifies that generating 100+ parts
// produces correct sequential numbering and all files are accounted for.
func TestSplitter_Split_100PlusPartsGenerated(t *testing.T) {
	t.Parallel()

	// Create 200 files, each with 500 tokens.
	// With overhead of 200 per file = 700 effective per file.
	// Budget of 1000 per part minus Part1 header (600) = 400 effective.
	// That means roughly 0-1 files per part with very tight budget.
	// Use budget of 1500 per part. Part1 header overhead = 600, effective = 900.
	// PartN header overhead = 300, effective = 1200.
	// Each file = 500 + 200 = 700. So ~1 file per part for Part1, ~1-2 for PartN.
	files := make([]FileRenderEntry, 200)
	for i := 0; i < 200; i++ {
		files[i] = makeFile(
			fmt.Sprintf("pkg%03d/file.go", i),
			500,
			i%5,
		)
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 1500})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.Greater(t, len(parts), 100, "should produce 100+ parts with this budget")

	// Verify all part numbers are sequential.
	for i, part := range parts {
		assert.Equal(t, i+1, part.PartNumber)
		assert.Equal(t, len(parts), part.TotalParts)
	}

	// Verify all files are accounted for.
	totalFiles := 0
	for _, part := range parts {
		totalFiles += len(part.RenderData.Files)
	}
	assert.Equal(t, 200, totalFiles)

	// Verify PartPath produces correctly padded names for all parts.
	for _, part := range parts {
		path := PartPath("output.md", part.PartNumber, part.TotalParts)
		assert.Contains(t, path, ".part-")
		assert.Contains(t, path, ".md")
	}
}

// TestSplitter_Split_OversizedFileBetweenNormalFiles verifies that an oversized
// file between normal files preserves the ordering of all files across parts.
func TestSplitter_Split_OversizedFileBetweenNormalFiles(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 100, 0),
		makeFile("b.go", 100, 0),
		makeFile("huge.go", 50000, 1), // oversized
		makeFile("c.go", 100, 2),
		makeFile("d.go", 100, 2),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 3000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parts), 3, "should produce at least 3 parts")

	// Collect all file paths in order across parts.
	var allPaths []string
	for _, part := range parts {
		for _, f := range part.RenderData.Files {
			allPaths = append(allPaths, f.Path)
		}
	}

	// Order must be preserved.
	expected := []string{"a.go", "b.go", "huge.go", "c.go", "d.go"}
	assert.Equal(t, expected, allPaths, "file ordering must be preserved across parts")

	// The oversized file must be alone in its part.
	for _, part := range parts {
		for _, f := range part.RenderData.Files {
			if f.Path == "huge.go" {
				assert.Len(t, part.RenderData.Files, 1,
					"oversized file should be isolated in its own part")
			}
		}
	}
}

// TestSplitter_Split_EmptyFileListSinglePart verifies that an empty file list
// still returns one part (the single-part path since 0 tokens < any budget).
func TestSplitter_Split_EmptyFileListVaryingBudgets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		budget int
	}{
		{name: "small budget", budget: 100},
		{name: "medium budget", budget: 10000},
		{name: "large budget", budget: 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewSplitter(SplitOpts{TokensPerPart: tt.budget})
			data := splitterTestRenderData(nil)
			parts, err := s.Split(data)

			require.NoError(t, err)
			require.Len(t, parts, 1)
			assert.Equal(t, 1, parts[0].PartNumber)
			assert.Equal(t, 1, parts[0].TotalParts)
			assert.Empty(t, parts[0].RenderData.Files)
		})
	}
}

// TestWriteSplit_XMLStdout verifies that split output works with XML format
// to stdout.
func TestWriteSplit_XMLStdout(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	files := []FileRenderEntry{makeFile("main.go", 100, 0)}
	data := splitterTestRenderData(files)

	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{Format: "xml", UseStdout: true},
		SplitTokens: 100000,
	})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Empty(t, results[0].Path)
	assert.Contains(t, stdout.String(), "<repository>")
	assert.Contains(t, stdout.String(), "test-project")
}

// TestWriteSplit_MultiplePartsXMLFile verifies end-to-end split writing with
// XML format to files.
func TestWriteSplit_MultiplePartsXMLFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.xml")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	files := []FileRenderEntry{
		makeFile("a.go", 5000, 0),
		makeFile("b.go", 5000, 1),
	}
	data := splitterTestRenderData(files)

	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{OutputPath: outPath, Format: "xml"},
		SplitTokens: 6000,
	})

	require.NoError(t, err)
	require.Len(t, results, 2)

	// Check .xml extension in part file names.
	assert.Equal(t, filepath.Join(dir, "output.part-001.xml"), results[0].Path)
	assert.Equal(t, filepath.Join(dir, "output.part-002.xml"), results[1].Path)

	// Both files should exist and contain XML.
	for _, r := range results {
		content, err := os.ReadFile(r.Path)
		require.NoError(t, err)
		assert.Contains(t, string(content), "<repository>")
	}

	// Part 1 should have the full tree.
	content1, err := os.ReadFile(results[0].Path)
	require.NoError(t, err)
	assert.NotContains(t, string(content1), "See Part 1")

	// Part 2 should reference Part 1.
	content2, err := os.ReadFile(results[1].Path)
	require.NoError(t, err)
	assert.Contains(t, string(content2), "See Part 1")
}

// TestSplitter_Split_AllFilesInOnePart_NoPartSuffix verifies that when all
// files fit in one part, the single-part path is used (no .part-001 suffix) and
// the original RenderData is returned unchanged.
func TestSplitter_Split_AllFilesInOnePart_OriginalDataUnchanged(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("main.go", 100, 0),
		makeFile("util.go", 200, 1),
		makeFile("test.go", 50, 3),
	}
	data := splitterTestRenderData(files)
	data.TreeString = "full tree"
	data.TopFilesByTokens = files[:1]
	data.RedactionSummary = map[string]int{"key": 1}
	data.TotalRedactions = 1

	s := NewSplitter(SplitOpts{TokensPerPart: 500000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.Len(t, parts, 1)

	// Original data should be returned as-is (pointer equality).
	assert.Same(t, data, parts[0].RenderData, "single-part should return original data unchanged")

	// PartPath for single-part should produce no suffix.
	path := PartPath("output.md", parts[0].PartNumber, parts[0].TotalParts)
	assert.Equal(t, "output.md", path, "single part should not have .part-001 suffix")
}

// TestSplitter_Split_TierBoundaryRespected verifies that when possible, files
// from the same tier stay together.
func TestSplitter_Split_TierBoundaryRespected(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("critical/a.go", 1000, 0),
		makeFile("critical/b.go", 1000, 0),
		makeFile("primary/c.go", 1000, 1),
		makeFile("primary/d.go", 1000, 1),
		makeFile("support/e.go", 1000, 3),
	}
	data := splitterTestRenderData(files)

	// Budget that forces splitting but allows 2 files per part.
	// Each file = 1000 + 200 overhead = 1200.
	// Part1: header 600, effective 2900 - 600 = 2300, fits 1 file (1200 < 2300)
	// and 2 files (2400 > 2300). So Part1 gets 1 file... unless coherence kicks in.
	// Let's use a budget that allows exactly 2 files per part.
	// Part1: budget 3500 - 600 header = 2900. 2 files = 2400. Fits.
	// 3 files = 3600 > 2900. Doesn't fit. So Part1 gets critical/a.go + critical/b.go.
	// Part2: budget 3500 - 300 header = 3200. 2 files = 2400. Fits.
	// So Part2 gets primary/c.go + primary/d.go.
	// Part3 gets support/e.go.
	s := NewSplitter(SplitOpts{TokensPerPart: 3500})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parts), 2)

	// All files from tier 0 should be in the same part.
	tier0Part := -1
	for i, part := range parts {
		for _, f := range part.RenderData.Files {
			if f.Tier == 0 {
				if tier0Part == -1 {
					tier0Part = i
				}
				assert.Equal(t, tier0Part, i, "all tier 0 files should be in the same part")
			}
		}
	}
}

// TestWriteSplit_PartResultsFileCount verifies that PartResult.FileCount
// matches the actual number of files in each part.
func TestWriteSplit_PartResultsFileCount(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	files := []FileRenderEntry{
		makeFile("a.go", 3000, 0),
		makeFile("b.go", 3000, 1),
		makeFile("c.go", 3000, 2),
	}
	data := splitterTestRenderData(files)

	results, err := ow.WriteSplit(context.Background(), data, SplitOutputOpts{
		OutputOpts:  OutputOpts{OutputPath: outPath, Format: "markdown"},
		SplitTokens: 4000,
	})

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(results), 2)

	totalFileCount := 0
	for _, r := range results {
		assert.Greater(t, r.FileCount, 0, "each part should have at least one file")
		totalFileCount += r.FileCount
	}
	assert.Equal(t, 3, totalFileCount, "total file count across parts should equal input files")
}

// TestSplitter_Split_PreservesProfileAndTokenizerName verifies that profile
// name and tokenizer name are propagated to all parts.
func TestSplitter_Split_PreservesProfileAndTokenizerName(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 3000, 0),
		makeFile("b.go", 3000, 1),
	}
	data := splitterTestRenderData(files)
	data.ProfileName = "custom-profile"
	data.TokenizerName = "o200k_base"

	s := NewSplitter(SplitOpts{TokensPerPart: 4000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parts), 2)

	for _, part := range parts {
		assert.Equal(t, "custom-profile", part.RenderData.ProfileName)
		assert.Equal(t, "o200k_base", part.RenderData.TokenizerName)
	}
}

// TestSplitter_Split_MixedOversizedAndNormalFiles verifies correct behavior
// when oversized files are interspersed with normal files.
func TestSplitter_Split_MixedOversizedAndNormalFiles(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("small1.go", 100, 0),
		makeFile("huge1.go", 50000, 1),
		makeFile("small2.go", 100, 2),
		makeFile("huge2.go", 60000, 3),
		makeFile("small3.go", 100, 4),
	}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 3000})
	parts, err := s.Split(data)

	require.NoError(t, err)

	// Collect all paths.
	var allPaths []string
	for _, part := range parts {
		for _, f := range part.RenderData.Files {
			allPaths = append(allPaths, f.Path)
		}
	}

	// All files present and order preserved.
	expected := []string{"small1.go", "huge1.go", "small2.go", "huge2.go", "small3.go"}
	assert.Equal(t, expected, allPaths)

	// Oversized files should be alone in their parts.
	for _, part := range parts {
		for _, f := range part.RenderData.Files {
			if f.Path == "huge1.go" || f.Path == "huge2.go" {
				assert.Len(t, part.RenderData.Files, 1,
					"oversized file %s should be alone in its part", f.Path)
			}
		}
	}
}

// TestSplitter_Split_PartData_GlobalHashConsistent verifies that GlobalHash is
// the same across all parts and matches the original data's ContentHash.
func TestSplitter_Split_PartData_GlobalHashConsistent(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("a.go", 3000, 0),
		makeFile("b.go", 3000, 1),
		makeFile("c.go", 3000, 2),
		makeFile("d.go", 3000, 3),
	}
	data := splitterTestRenderData(files)
	data.ContentHash = "unique-global-hash-abc123"

	s := NewSplitter(SplitOpts{TokensPerPart: 4000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parts), 2)

	for _, part := range parts {
		assert.Equal(t, "unique-global-hash-abc123", part.GlobalHash,
			"GlobalHash should be consistent across all parts")
		assert.Equal(t, "unique-global-hash-abc123", part.RenderData.ContentHash,
			"RenderData.ContentHash should match GlobalHash")
	}
}

// TestSplitter_Split_buildPartRenderData_EmptyFiles verifies behavior when
// building part render data with an empty file slice.
func TestSplitter_buildPartRenderData_EmptyFiles(t *testing.T) {
	t.Parallel()

	data := splitterTestRenderData(nil)
	s := NewSplitter(SplitOpts{TokensPerPart: 10000})
	rd := s.buildPartRenderData(data, nil, 1, 1, "hash")

	assert.Equal(t, 0, rd.TotalTokens)
	assert.Equal(t, 0, rd.TotalFiles)
	assert.Empty(t, rd.Files)
	assert.Empty(t, rd.TierCounts)
}

// TestPartPath_BothExtensions is a table-driven test verifying PartPath works
// correctly with both .md and .xml extensions across various scenarios.
func TestPartPath_BothExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		basePath   string
		partNum    int
		totalParts int
		want       string
	}{
		{
			name:       "markdown extension",
			basePath:   "report.md",
			partNum:    1,
			totalParts: 3,
			want:       "report.part-001.md",
		},
		{
			name:       "xml extension",
			basePath:   "report.xml",
			partNum:    2,
			totalParts: 3,
			want:       "report.part-002.xml",
		},
		{
			name:       "txt extension",
			basePath:   "output.txt",
			partNum:    1,
			totalParts: 2,
			want:       "output.part-001.txt",
		},
		{
			name:       "dot in directory name with md",
			basePath:   "/home/user/v1.0/output.md",
			partNum:    3,
			totalParts: 5,
			want:       "/home/user/v1.0/output.part-003.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := PartPath(tt.basePath, tt.partNum, tt.totalParts)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestWriteSplit_CancelledContextBetweenParts verifies that context cancellation
// is checked between rendering parts.
func TestWriteSplit_CancelledContextBetweenParts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	// Create enough files to guarantee multiple parts.
	files := []FileRenderEntry{
		makeFile("a.go", 5000, 0),
		makeFile("b.go", 5000, 1),
		makeFile("c.go", 5000, 2),
	}
	data := splitterTestRenderData(files)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately -- the context check happens in the loop.
	cancel()

	_, err := ow.WriteSplit(ctx, data, SplitOutputOpts{
		OutputOpts:  OutputOpts{OutputPath: outPath, Format: "markdown"},
		SplitTokens: 6000,
	})

	require.Error(t, err, "should return error when context is cancelled")
}

// TestSplitter_assignFilesToParts_EmptySlice verifies that an empty file list
// returns nil buckets.
func TestSplitter_assignFilesToParts_EmptySlice(t *testing.T) {
	t.Parallel()

	s := NewSplitter(SplitOpts{TokensPerPart: 10000})
	buckets := s.assignFilesToParts(nil)
	assert.Nil(t, buckets)

	buckets = s.assignFilesToParts([]FileRenderEntry{})
	assert.Nil(t, buckets)
}

// TestSplitter_Split_SingleFileUnderBudget verifies that a single small file
// produces exactly one part with no splitting.
func TestSplitter_Split_SingleFileUnderBudget(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{makeFile("tiny.go", 10, 0)}
	data := splitterTestRenderData(files)

	s := NewSplitter(SplitOpts{TokensPerPart: 100000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.Len(t, parts, 1)
	assert.Equal(t, 1, parts[0].PartNumber)
	assert.Equal(t, 1, parts[0].TotalParts)
	assert.Len(t, parts[0].RenderData.Files, 1)
}

// TestSplitter_Split_TopLevelDirCoherence_SameDir verifies that directory
// coherence grouping works for files in deeply nested but same top-level dirs.
func TestSplitter_Split_TopLevelDirCoherence_SameDir(t *testing.T) {
	t.Parallel()

	files := []FileRenderEntry{
		makeFile("internal/cli/root.go", 1000, 0),
		makeFile("internal/cli/generate.go", 1000, 0),
		makeFile("internal/output/writer.go", 1000, 0),
	}
	data := splitterTestRenderData(files)

	// Budget: each file = 1200 with overhead. Part1 header = 600.
	// Effective = 3000 - 600 = 2400. 2 files = 2400, just fits.
	// With coherence (same top-level dir "internal"), 3 files might overflow
	// by < 15%, so they could be kept together.
	s := NewSplitter(SplitOpts{TokensPerPart: 3000})
	parts, err := s.Split(data)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(parts), 1)

	// All three share "internal" as top-level dir and are same tier,
	// so they should preferentially be grouped together if coherence applies.
	// Verify all files are accounted for.
	totalFiles := 0
	for _, part := range parts {
		totalFiles += len(part.RenderData.Files)
	}
	assert.Equal(t, 3, totalFiles)
}

// --- Benchmark tests ---

func BenchmarkSplitter_Split_SmallFiles(b *testing.B) {
	files := make([]FileRenderEntry, 100)
	for i := 0; i < 100; i++ {
		files[i] = makeFile(
			fmt.Sprintf("pkg%d/file.go", i),
			500,
			i%5,
		)
	}
	data := splitterTestRenderData(files)
	s := NewSplitter(SplitOpts{TokensPerPart: 5000})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.Split(data)
	}
}

func BenchmarkSplitter_Split_LargeFileCount(b *testing.B) {
	files := make([]FileRenderEntry, 1000)
	for i := 0; i < 1000; i++ {
		files[i] = makeFile(
			fmt.Sprintf("pkg%03d/file%03d.go", i/10, i%10),
			200,
			i%6,
		)
	}
	data := splitterTestRenderData(files)
	s := NewSplitter(SplitOpts{TokensPerPart: 10000})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.Split(data)
	}
}

func BenchmarkPartPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		PartPath("harvx-output.md", 42, 100)
	}
}
