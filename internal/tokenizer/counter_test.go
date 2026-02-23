package tokenizer_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tokenizer"
)

// stubTokenizer is a deterministic, zero-overhead Tokenizer implementation
// used exclusively in tests. Count returns len(text) so that expected totals
// can be computed arithmetically without initialising any BPE encoder.
// It is safe for concurrent use from multiple goroutines.
type stubTokenizer struct {
	name string
}

func (s *stubTokenizer) Count(text string) int { return len(text) }
func (s *stubTokenizer) Name() string          { return s.name }

// newStub returns a *stubTokenizer that satisfies the tokenizer.Tokenizer
// interface. Compile-time assertion is verified below.
func newStub() *stubTokenizer { return &stubTokenizer{name: "stub"} }

// Compile-time interface compliance check.
var _ tokenizer.Tokenizer = (*stubTokenizer)(nil)

// makeDescriptor is a test helper that returns a *pipeline.FileDescriptor
// with Path and Content pre-populated. TokenCount is intentionally left
// as zero to verify that CountFile and CountFiles populate it.
func makeDescriptor(t *testing.T, path, content string) *pipeline.FileDescriptor {
	t.Helper()
	return &pipeline.FileDescriptor{
		Path:    path,
		Content: content,
	}
}

// ---------------------------------------------------------------------------
// TestTokenCounter_CountFile_populated
// ---------------------------------------------------------------------------

// TestTokenCounter_CountFile_populated verifies that CountFile sets TokenCount
// from fd.Content using the supplied Tokenizer.
func TestTokenCounter_CountFile_populated(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "short ascii content",
			content: "hello",
			want:    5,
		},
		{
			name:    "go source snippet",
			content: "package main\n\nfunc main() {}",
			want:    len("package main\n\nfunc main() {}"),
		},
		{
			name:    "multiline content",
			content: "line one\nline two\nline three\n",
			want:    len("line one\nline two\nline three\n"),
		},
		{
			name:    "unicode content",
			content: "こんにちは",
			want:    len("こんにちは"), // stub counts bytes, not runes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tokenizer.NewTokenCounter(newStub())
			fd := makeDescriptor(t, "src/file.go", tt.content)

			c.CountFile(fd)

			assert.Equal(t, tt.want, fd.TokenCount,
				"CountFile must populate TokenCount from fd.Content")
		})
	}
}

// ---------------------------------------------------------------------------
// TestTokenCounter_CountFile_empty
// ---------------------------------------------------------------------------

// TestTokenCounter_CountFile_empty verifies that CountFile sets TokenCount to
// zero when fd.Content is the empty string.
func TestTokenCounter_CountFile_empty(t *testing.T) {
	t.Parallel()
	c := tokenizer.NewTokenCounter(newStub())
	fd := makeDescriptor(t, "empty.go", "")

	c.CountFile(fd)

	assert.Equal(t, 0, fd.TokenCount, "empty content must produce TokenCount == 0")
}

// ---------------------------------------------------------------------------
// TestTokenCounter_CountFiles_zero
// ---------------------------------------------------------------------------

// TestTokenCounter_CountFiles_zero verifies that CountFiles with an empty
// slice returns (0, nil) immediately.
func TestTokenCounter_CountFiles_zero(t *testing.T) {
	t.Parallel()
	c := tokenizer.NewTokenCounter(newStub())

	total, err := c.CountFiles(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, 0, total, "zero files must return total == 0")
}

// TestTokenCounter_CountFiles_emptySlice mirrors TestTokenCounter_CountFiles_zero
// but passes an explicit empty (non-nil) slice.
func TestTokenCounter_CountFiles_emptySlice(t *testing.T) {
	t.Parallel()
	c := tokenizer.NewTokenCounter(newStub())

	total, err := c.CountFiles(context.Background(), []*pipeline.FileDescriptor{})

	require.NoError(t, err)
	assert.Equal(t, 0, total, "empty slice must return total == 0")
}

// ---------------------------------------------------------------------------
// TestTokenCounter_CountFiles_multiple
// ---------------------------------------------------------------------------

// TestTokenCounter_CountFiles_multiple verifies that CountFiles counts tokens
// for every file and returns the correct aggregate total.
func TestTokenCounter_CountFiles_multiple(t *testing.T) {
	t.Parallel()
	contents := []string{
		"abcde",                     // 5
		"1234567890",                // 10
		"",                          // 0
		"hello world",               // 11
		strings.Repeat("x", 1000),   // 1000
	}
	wantTotal := 0
	files := make([]*pipeline.FileDescriptor, len(contents))
	for i, c := range contents {
		wantTotal += len(c)
		files[i] = makeDescriptor(t, "file.go", c)
	}

	counter := tokenizer.NewTokenCounter(newStub())
	total, err := counter.CountFiles(context.Background(), files)

	require.NoError(t, err)
	assert.Equal(t, wantTotal, total,
		"CountFiles must return sum of per-file token counts")

	// Each descriptor must have its own TokenCount populated.
	for i, fd := range files {
		assert.Equal(t, len(contents[i]), fd.TokenCount,
			"fd[%d].TokenCount must equal len(content)", i)
	}
}

// TestTokenCounter_CountFiles_singleFile exercises the single-file path to
// ensure no edge cases exist around channel buffering with len==1.
func TestTokenCounter_CountFiles_singleFile(t *testing.T) {
	t.Parallel()
	content := "hello"
	fd := makeDescriptor(t, "single.go", content)

	c := tokenizer.NewTokenCounter(newStub())
	total, err := c.CountFiles(context.Background(), []*pipeline.FileDescriptor{fd})

	require.NoError(t, err)
	assert.Equal(t, len(content), total)
	assert.Equal(t, len(content), fd.TokenCount)
}

// ---------------------------------------------------------------------------
// TestTokenCounter_CountFiles_cancellation
// ---------------------------------------------------------------------------

// TestTokenCounter_CountFiles_cancellation verifies that CountFiles respects
// context cancellation. When the context is already cancelled before the call,
// CountFiles must return a non-nil error that wraps context.Canceled.
func TestTokenCounter_CountFiles_cancellation(t *testing.T) {
	t.Parallel()

	// Build a large batch so that at least some goroutines are queued behind
	// NumCPU and will observe the cancelled gctx before calling CountFile.
	const fileCount = 500
	files := make([]*pipeline.FileDescriptor, fileCount)
	for i := range files {
		files[i] = makeDescriptor(t, "file.go", strings.Repeat("a", 128))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before calling CountFiles

	c := tokenizer.NewTokenCounter(newStub())
	_, err := c.CountFiles(ctx, files)

	require.Error(t, err, "CountFiles must return an error when context is cancelled")
}

// TestTokenCounter_CountFiles_cancellationMidFlight cancels the context after
// the call has been started to exercise mid-flight cancellation.
func TestTokenCounter_CountFiles_cancellationMidFlight(t *testing.T) {
	t.Parallel()

	const fileCount = 200
	files := make([]*pipeline.FileDescriptor, fileCount)
	for i := range files {
		files[i] = makeDescriptor(t, "file.go", strings.Repeat("b", 256))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel after a minimal delay so the errgroup goroutines have started.
	go func() { cancel() }()

	c := tokenizer.NewTokenCounter(newStub())
	_, err := c.CountFiles(ctx, files)
	// err may or may not be non-nil depending on timing; we only assert no
	// panic and that if an error is returned it wraps context.Canceled.
	if err != nil {
		assert.ErrorContains(t, err, "cancelled",
			"error message must mention cancellation")
	}
}

// ---------------------------------------------------------------------------
// TestTokenCounter_EstimateOverhead_zero
// ---------------------------------------------------------------------------

// TestTokenCounter_EstimateOverhead_zero verifies that EstimateOverhead(0, 0)
// returns the base overhead of 200 tokens.
func TestTokenCounter_EstimateOverhead_zero(t *testing.T) {
	t.Parallel()
	c := tokenizer.NewTokenCounter(newStub())

	got := c.EstimateOverhead(0, 0)

	assert.Equal(t, 200, got,
		"EstimateOverhead(0, 0) must return the base overhead of 200")
}

// ---------------------------------------------------------------------------
// TestTokenCounter_EstimateOverhead_values
// ---------------------------------------------------------------------------

// TestTokenCounter_EstimateOverhead_values is a table-driven test for the
// overhead formula: overhead = 200 + (fileCount * 35).
func TestTokenCounter_EstimateOverhead_values(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		fileCount int
		treeSize  int // currently unused by the formula but forwarded
		want      int
	}{
		{
			name:      "zero files zero tree",
			fileCount: 0,
			treeSize:  0,
			want:      200, // 200 + (0 * 35)
		},
		{
			name:      "ten files zero tree",
			fileCount: 10,
			treeSize:  0,
			want:      550, // 200 + (10 * 35)
		},
		{
			name:      "one file",
			fileCount: 1,
			treeSize:  0,
			want:      235, // 200 + (1 * 35)
		},
		{
			name:      "tree size ignored",
			fileCount: 10,
			treeSize:  9999,
			want:      550, // treeSize has no effect per spec
		},
		{
			name:      "100 files",
			fileCount: 100,
			treeSize:  0,
			want:      3700, // 200 + (100 * 35)
		},
		{
			name:      "1000 files",
			fileCount: 1000,
			treeSize:  0,
			want:      35200, // 200 + (1000 * 35)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := tokenizer.NewTokenCounter(newStub())
			got := c.EstimateOverhead(tt.fileCount, tt.treeSize)
			assert.Equal(t, tt.want, got,
				"EstimateOverhead(%d, %d) = %d, want %d",
				tt.fileCount, tt.treeSize, got, tt.want)
		})
	}
}

// ---------------------------------------------------------------------------
// TestTokenCounter_CountFile_processedContent
// ---------------------------------------------------------------------------

// TestTokenCounter_CountFile_processedContent verifies that CountFile operates
// on fd.Content as-is (the already-processed, post-redaction content). The
// test simulates a redacted secret by injecting a "[REDACTED:key]" marker
// directly into Content, then asserts that TokenCount reflects the length of
// that processed string rather than any hypothetical original.
func TestTokenCounter_CountFile_processedContent(t *testing.T) {
	t.Parallel()
	// Simulate content after the security stage has replaced a real key value
	// with a placeholder.
	processed := "hello [REDACTED:key]"
	fd := makeDescriptor(t, "config.go", processed)

	c := tokenizer.NewTokenCounter(newStub())
	c.CountFile(fd)

	assert.Equal(t, len(processed), fd.TokenCount,
		"CountFile must count tokens on fd.Content (post-redaction), not any original value")
}

// TestTokenCounter_CountFile_doesNotMutateOtherFields verifies that CountFile
// only modifies TokenCount and leaves all other fields of the descriptor intact.
func TestTokenCounter_CountFile_doesNotMutateOtherFields(t *testing.T) {
	t.Parallel()
	fd := &pipeline.FileDescriptor{
		Path:         "src/main.go",
		AbsPath:      "/repo/src/main.go",
		Size:         1024,
		Tier:         1,
		Content:      "package main",
		Language:     "go",
		IsCompressed: true,
		Redactions:   3,
		IsBinary:     false,
		IsSymlink:    false,
		ContentHash:  0xDEADBEEF,
	}

	c := tokenizer.NewTokenCounter(newStub())
	c.CountFile(fd)

	assert.Equal(t, "src/main.go", fd.Path)
	assert.Equal(t, "/repo/src/main.go", fd.AbsPath)
	assert.Equal(t, int64(1024), fd.Size)
	assert.Equal(t, 1, fd.Tier)
	assert.Equal(t, "package main", fd.Content)
	assert.Equal(t, "go", fd.Language)
	assert.True(t, fd.IsCompressed)
	assert.Equal(t, 3, fd.Redactions)
	assert.False(t, fd.IsBinary)
	assert.False(t, fd.IsSymlink)
	assert.Equal(t, uint64(0xDEADBEEF), fd.ContentHash)
	// Only TokenCount should have been updated.
	assert.Equal(t, len("package main"), fd.TokenCount)
}

// ---------------------------------------------------------------------------
// BenchmarkTokenCounter_CountFiles_1K
// ---------------------------------------------------------------------------

// BenchmarkTokenCounter_CountFiles_1K measures CountFiles throughput against
// 1 000 FileDescriptors each carrying ~1 KB of content. The stub tokenizer is
// used to isolate the dispatch and concurrency overhead from BPE encoding cost.
func BenchmarkTokenCounter_CountFiles_1K(b *testing.B) {
	const fileCount = 1000
	const contentSize = 1024 // 1 KB per file

	content := strings.Repeat("x", contentSize)
	files := make([]*pipeline.FileDescriptor, fileCount)
	for i := range files {
		files[i] = &pipeline.FileDescriptor{
			Path:    "file.go",
			Content: content,
		}
	}

	c := tokenizer.NewTokenCounter(newStub())
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		// Reset TokenCount so each iteration starts from a clean state.
		for _, fd := range files {
			fd.TokenCount = 0
		}
		_, err := c.CountFiles(ctx, files)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTokenCounter_CountFile_single measures per-file overhead when
// CountFile is called in a tight loop without any parallelism.
func BenchmarkTokenCounter_CountFile_single(b *testing.B) {
	content := strings.Repeat("The quick brown fox. ", 50) // ~1 KB
	fd := &pipeline.FileDescriptor{
		Path:    "file.go",
		Content: content,
	}
	c := tokenizer.NewTokenCounter(newStub())

	b.ResetTimer()
	for range b.N {
		fd.TokenCount = 0
		c.CountFile(fd)
	}
}
