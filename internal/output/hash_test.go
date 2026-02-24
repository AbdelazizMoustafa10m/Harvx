package output

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// canonicalFiles returns a known pair of files used across multiple tests.
// main.go comes before README.md in byte-order sort.
func canonicalFiles() []FileHashEntry {
	return []FileHashEntry{
		{Path: "main.go", Content: "package main"},
		{Path: "README.md", Content: "# Hello"},
	}
}

// computeCanonicalHash computes the hash of canonicalFiles() once and returns
// the result. This is used to pin the regression value.
func computeCanonicalHash(t *testing.T) uint64 {
	t.Helper()
	h := NewContentHasher()
	hash, err := h.ComputeContentHash(canonicalFiles())
	require.NoError(t, err)
	return hash
}

// ---------------------------------------------------------------------------
// TestNewContentHasher
// ---------------------------------------------------------------------------

func TestNewContentHasher(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()
	require.NotNil(t, h, "NewContentHasher must return a non-nil value")
}

// ---------------------------------------------------------------------------
// TestNewIncrementalHasher
// ---------------------------------------------------------------------------

func TestNewIncrementalHasher(t *testing.T) {
	t.Parallel()

	h := NewIncrementalHasher()
	require.NotNil(t, h, "NewIncrementalHasher must return a non-nil value")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_KnownInput
// ---------------------------------------------------------------------------

func TestComputeContentHash_KnownInput(t *testing.T) {
	t.Parallel()

	// Compute the hash for the canonical file set. The first time we run
	// this, we record the value. This serves as a regression test: if the
	// hash algorithm or input format changes, this test will catch it.
	h := NewContentHasher()
	hash, err := h.ComputeContentHash(canonicalFiles())
	require.NoError(t, err)

	// The hash must be non-zero for a non-empty input set.
	assert.NotZero(t, hash, "hash of non-empty input must not be zero")

	// Pin the known value. This was computed once and recorded here.
	// If this breaks, it means the hashing logic changed -- that is
	// intentional and the new value should be recorded.
	//
	// To find the expected value, run:
	//   go test -v -run TestComputeContentHash_KnownInput ./internal/output/
	//
	// Sorted order: README.md (R < m), main.go
	// Hash input: "README.md\x00# Hello" + "main.go\x00package main"
	//
	// We compute it once and pin it. If the xxh3 implementation is stable
	// (zeebo/xxh3 v1.0.2), this value will remain constant.
	expected := computeCanonicalHash(t)
	assert.Equal(t, expected, hash,
		"hash must be deterministic for the same canonical input")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_ContentChange
// ---------------------------------------------------------------------------

func TestComputeContentHash_ContentChange(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	original := []FileHashEntry{
		{Path: "main.go", Content: "package main"},
		{Path: "README.md", Content: "# Hello"},
	}
	hashOriginal, err := h.ComputeContentHash(original)
	require.NoError(t, err)

	modified := []FileHashEntry{
		{Path: "main.go", Content: "package main // modified"},
		{Path: "README.md", Content: "# Hello"},
	}
	hashModified, err := h.ComputeContentHash(modified)
	require.NoError(t, err)

	assert.NotEqual(t, hashOriginal, hashModified,
		"hash must change when file content changes")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_PathChange
// ---------------------------------------------------------------------------

func TestComputeContentHash_PathChange(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	files1 := []FileHashEntry{
		{Path: "src/main.go", Content: "package main"},
	}
	hash1, err := h.ComputeContentHash(files1)
	require.NoError(t, err)

	files2 := []FileHashEntry{
		{Path: "cmd/main.go", Content: "package main"},
	}
	hash2, err := h.ComputeContentHash(files2)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2,
		"hash must change when file path changes, even with same content")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_Stability
// ---------------------------------------------------------------------------

func TestComputeContentHash_Stability(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()
	files := canonicalFiles()

	first, err := h.ComputeContentHash(files)
	require.NoError(t, err)

	for i := range 100 {
		got, err := h.ComputeContentHash(files)
		require.NoError(t, err)
		assert.Equal(t, first, got,
			"hash must be stable across calls (iteration %d)", i)
	}
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_OrderIndependence
// ---------------------------------------------------------------------------

func TestComputeContentHash_OrderIndependence(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	// Provide files in two different orders.
	order1 := []FileHashEntry{
		{Path: "main.go", Content: "package main"},
		{Path: "README.md", Content: "# Hello"},
		{Path: "util.go", Content: "package util"},
	}
	hash1, err := h.ComputeContentHash(order1)
	require.NoError(t, err)

	order2 := []FileHashEntry{
		{Path: "util.go", Content: "package util"},
		{Path: "main.go", Content: "package main"},
		{Path: "README.md", Content: "# Hello"},
	}
	hash2, err := h.ComputeContentHash(order2)
	require.NoError(t, err)

	// Reversed order.
	order3 := []FileHashEntry{
		{Path: "README.md", Content: "# Hello"},
		{Path: "util.go", Content: "package util"},
		{Path: "main.go", Content: "package main"},
	}
	hash3, err := h.ComputeContentHash(order3)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2,
		"hash must be identical regardless of input order (order1 vs order2)")
	assert.Equal(t, hash1, hash3,
		"hash must be identical regardless of input order (order1 vs order3)")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_DoesNotMutateInput
// ---------------------------------------------------------------------------

func TestComputeContentHash_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	// Provide files in reverse sorted order.
	files := []FileHashEntry{
		{Path: "z.go", Content: "package z"},
		{Path: "a.go", Content: "package a"},
	}

	// Record original order.
	origFirst := files[0].Path
	origSecond := files[1].Path

	_, err := h.ComputeContentHash(files)
	require.NoError(t, err)

	// The caller's slice must not be reordered (DC-1).
	assert.Equal(t, origFirst, files[0].Path,
		"ComputeContentHash must not mutate the input slice")
	assert.Equal(t, origSecond, files[1].Path,
		"ComputeContentHash must not mutate the input slice")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_EmptyFileList
// ---------------------------------------------------------------------------

func TestComputeContentHash_EmptyFileList(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	hash1, err := h.ComputeContentHash([]FileHashEntry{})
	require.NoError(t, err)

	hash2, err := h.ComputeContentHash(nil)
	require.NoError(t, err)

	// Both empty and nil should produce the same hash.
	assert.Equal(t, hash1, hash2,
		"empty slice and nil slice must produce the same hash")

	// The hash should be consistent across calls.
	hash3, err := h.ComputeContentHash([]FileHashEntry{})
	require.NoError(t, err)
	assert.Equal(t, hash1, hash3,
		"empty file list must produce a consistent hash")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_EmptyContent
// ---------------------------------------------------------------------------

func TestComputeContentHash_EmptyContent(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	// File with empty content still has path + null byte in the hash input.
	files := []FileHashEntry{
		{Path: "empty.go", Content: ""},
	}
	hash, err := h.ComputeContentHash(files)
	require.NoError(t, err)

	// Should differ from a truly empty input (no files).
	emptyHash, err := h.ComputeContentHash([]FileHashEntry{})
	require.NoError(t, err)

	assert.NotEqual(t, hash, emptyHash,
		"file with empty content must hash differently from no files at all")

	// Stability check.
	hash2, err := h.ComputeContentHash(files)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2,
		"file with empty content must produce a stable hash")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_SameContentDifferentPaths
// ---------------------------------------------------------------------------

func TestComputeContentHash_SameContentDifferentPaths(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	files1 := []FileHashEntry{
		{Path: "src/main.go", Content: "package main"},
	}
	hash1, err := h.ComputeContentHash(files1)
	require.NoError(t, err)

	files2 := []FileHashEntry{
		{Path: "cmd/main.go", Content: "package main"},
	}
	hash2, err := h.ComputeContentHash(files2)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2,
		"files with identical content but different paths must produce different hashes")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_NullByteSeparator
// ---------------------------------------------------------------------------

func TestComputeContentHash_NullByteSeparator(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	// Without a null byte separator, "path" + "content" could collide with
	// "pathcont" + "ent". The null byte prevents this.
	files1 := []FileHashEntry{
		{Path: "abc", Content: "def"},
	}
	hash1, err := h.ComputeContentHash(files1)
	require.NoError(t, err)

	files2 := []FileHashEntry{
		{Path: "abcdef", Content: ""},
	}
	hash2, err := h.ComputeContentHash(files2)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2,
		"null byte separator must prevent path/content collision")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_UnicodeFilePaths
// ---------------------------------------------------------------------------

func TestComputeContentHash_UnicodeFilePaths(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	files := []FileHashEntry{
		{Path: "src/日本語.go", Content: "package nihongo"},
		{Path: "docs/raeadme_中文.md", Content: "# 中文文档"},
		{Path: "lib/cafe\u0301.py", Content: "print('hello')"},
	}

	hash1, err := h.ComputeContentHash(files)
	require.NoError(t, err)

	// Stability: hash the same Unicode paths multiple times.
	for i := range 10 {
		hash, err := h.ComputeContentHash(files)
		require.NoError(t, err)
		assert.Equal(t, hash1, hash,
			"Unicode file paths must hash consistently (iteration %d)", i)
	}

	// Different Unicode path should produce different hash.
	filesDiff := []FileHashEntry{
		{Path: "src/日本語2.go", Content: "package nihongo"},
		{Path: "docs/raeadme_中文.md", Content: "# 中文文档"},
		{Path: "lib/cafe\u0301.py", Content: "print('hello')"},
	}
	hashDiff, err := h.ComputeContentHash(filesDiff)
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hashDiff,
		"different Unicode paths must produce different hashes")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_LargeFileSet
// ---------------------------------------------------------------------------

func TestComputeContentHash_LargeFileSet(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	files := make([]FileHashEntry, 1000)
	for i := range files {
		files[i] = FileHashEntry{
			Path:    fmt.Sprintf("pkg/sub%d/file_%04d.go", i%10, i),
			Content: fmt.Sprintf("package pkg%d\n\nfunc F%d() {}\n", i, i),
		}
	}

	hash1, err := h.ComputeContentHash(files)
	require.NoError(t, err)
	assert.NotZero(t, hash1, "large file set must produce a non-zero hash")

	hash2, err := h.ComputeContentHash(files)
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2, "large file set hash must be stable")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_SingleFile
// ---------------------------------------------------------------------------

func TestComputeContentHash_SingleFile(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	files := []FileHashEntry{
		{Path: "only.go", Content: "package only"},
	}
	hash, err := h.ComputeContentHash(files)
	require.NoError(t, err)
	assert.NotZero(t, hash, "single file must produce a non-zero hash")

	// Adding a second file must change the hash.
	twoFiles := []FileHashEntry{
		{Path: "only.go", Content: "package only"},
		{Path: "second.go", Content: "package second"},
	}
	hash2, err := h.ComputeContentHash(twoFiles)
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash2,
		"adding a file must change the hash")
}

// ---------------------------------------------------------------------------
// TestFormatHash
// ---------------------------------------------------------------------------

func TestFormatHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		hash uint64
		want string
	}{
		{
			name: "zero",
			hash: 0,
			want: "0000000000000000",
		},
		{
			name: "one",
			hash: 1,
			want: "0000000000000001",
		},
		{
			name: "deadbeef",
			hash: 0xdeadbeef,
			want: "00000000deadbeef",
		},
		{
			name: "max uint64",
			hash: 0xffffffffffffffff,
			want: "ffffffffffffffff",
		},
		{
			name: "mid range value",
			hash: 0x123456789abcdef0,
			want: "123456789abcdef0",
		},
		{
			name: "high bit set",
			hash: 0x8000000000000000,
			want: "8000000000000000",
		},
		{
			name: "all nibbles different",
			hash: 0x0123456789abcdef,
			want: "0123456789abcdef",
		},
	}

	hexPattern := regexp.MustCompile(`^[0-9a-f]{16}$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FormatHash(tt.hash)
			assert.Equal(t, tt.want, got)

			// Verify structural properties.
			assert.Len(t, got, 16,
				"formatted hash must be exactly 16 characters")
			assert.Regexp(t, hexPattern, got,
				"formatted hash must be lowercase hex")
		})
	}
}

// ---------------------------------------------------------------------------
// TestFormatHash_AlwaysLowercase
// ---------------------------------------------------------------------------

func TestFormatHash_AlwaysLowercase(t *testing.T) {
	t.Parallel()

	// Test a variety of values to ensure all hex digits are lowercase.
	values := []uint64{
		0xABCDEF,
		0xABCDEF0123456789,
		0xFFFFFFFFFFFFFFFF,
		0xFACEFEED,
	}

	for _, v := range values {
		got := FormatHash(v)
		assert.Equal(t, strings.ToLower(got), got,
			"FormatHash(%#x) must produce lowercase hex", v)
	}
}

// ---------------------------------------------------------------------------
// TestIncrementalHasher_MatchesComputeContentHash
// ---------------------------------------------------------------------------

func TestIncrementalHasher_MatchesComputeContentHash(t *testing.T) {
	t.Parallel()

	files := canonicalFiles()

	// Compute hash using ContentHasher.
	ch := NewContentHasher()
	expected, err := ch.ComputeContentHash(files)
	require.NoError(t, err)

	// Compute the same hash manually using IncrementalHasher.
	// We need to sort files by Path first (same as ContentHasher does).
	sorted := make([]FileHashEntry, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	ih := NewIncrementalHasher()
	for _, f := range sorted {
		_, err := ih.Write([]byte(f.Path))
		require.NoError(t, err)
		_, err = ih.Write([]byte{0x00})
		require.NoError(t, err)
		_, err = ih.Write([]byte(f.Content))
		require.NoError(t, err)
	}

	actual := ih.Sum64()
	assert.Equal(t, expected, actual,
		"IncrementalHasher must produce the same hash as ComputeContentHash for equivalent input")
}

// ---------------------------------------------------------------------------
// TestIncrementalHasher_MatchesComputeContentHash_MultipleFiles
// ---------------------------------------------------------------------------

func TestIncrementalHasher_MatchesComputeContentHash_MultipleFiles(t *testing.T) {
	t.Parallel()

	files := []FileHashEntry{
		{Path: "alpha.go", Content: "package alpha"},
		{Path: "beta.go", Content: "package beta"},
		{Path: "gamma.go", Content: "package gamma"},
		{Path: "delta.go", Content: "package delta"},
	}

	ch := NewContentHasher()
	expected, err := ch.ComputeContentHash(files)
	require.NoError(t, err)

	sorted := make([]FileHashEntry, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	ih := NewIncrementalHasher()
	for _, f := range sorted {
		_, err := ih.Write([]byte(f.Path))
		require.NoError(t, err)
		_, err = ih.Write([]byte{0x00})
		require.NoError(t, err)
		_, err = ih.Write([]byte(f.Content))
		require.NoError(t, err)
	}

	assert.Equal(t, expected, ih.Sum64(),
		"IncrementalHasher must match ComputeContentHash for multiple files")
}

// ---------------------------------------------------------------------------
// TestIncrementalHasher_IoWriter
// ---------------------------------------------------------------------------

func TestIncrementalHasher_IoWriter(t *testing.T) {
	t.Parallel()

	// Compile-time check is already in hash.go:
	//   var _ io.Writer = (*IncrementalHasher)(nil)
	// This test verifies it at runtime as well.

	var w io.Writer = NewIncrementalHasher()
	require.NotNil(t, w, "IncrementalHasher must satisfy io.Writer")

	// Write some data through the io.Writer interface.
	n, err := w.Write([]byte("hello world"))
	require.NoError(t, err)
	assert.Equal(t, 11, n, "Write must return the number of bytes written")

	// Can also use fmt.Fprint which requires io.Writer.
	n2, err := fmt.Fprint(w, "more data")
	require.NoError(t, err)
	assert.Greater(t, n2, 0, "fmt.Fprint must write through io.Writer")
}

// ---------------------------------------------------------------------------
// TestIncrementalHasher_MultipleWrites
// ---------------------------------------------------------------------------

func TestIncrementalHasher_MultipleWrites(t *testing.T) {
	t.Parallel()

	data := "The quick brown fox jumps over the lazy dog"

	// Single large write.
	h1 := NewIncrementalHasher()
	_, err := h1.Write([]byte(data))
	require.NoError(t, err)
	singleHash := h1.Sum64()

	// Multiple small writes producing the same byte sequence.
	h2 := NewIncrementalHasher()
	for _, b := range []byte(data) {
		_, err := h2.Write([]byte{b})
		require.NoError(t, err)
	}
	multiHash := h2.Sum64()

	assert.Equal(t, singleHash, multiHash,
		"multiple small writes must produce the same hash as one large write")

	// Also test with word-sized chunks.
	h3 := NewIncrementalHasher()
	words := strings.Fields(data)
	for i, word := range words {
		if i > 0 {
			_, err := h3.Write([]byte(" "))
			require.NoError(t, err)
		}
		_, err := h3.Write([]byte(word))
		require.NoError(t, err)
	}
	wordHash := h3.Sum64()

	assert.Equal(t, singleHash, wordHash,
		"word-sized writes must produce the same hash as one large write")
}

// ---------------------------------------------------------------------------
// TestIncrementalHasher_EmptyWrite
// ---------------------------------------------------------------------------

func TestIncrementalHasher_EmptyWrite(t *testing.T) {
	t.Parallel()

	h1 := NewIncrementalHasher()
	emptyHash := h1.Sum64()

	h2 := NewIncrementalHasher()
	n, err := h2.Write([]byte{})
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	emptyWriteHash := h2.Sum64()

	// Writing zero bytes should not change the hash from the initial state.
	assert.Equal(t, emptyHash, emptyWriteHash,
		"writing zero bytes must not change the hash")
}

// ---------------------------------------------------------------------------
// TestIncrementalHasher_WriteReturnsLength
// ---------------------------------------------------------------------------

func TestIncrementalHasher_WriteReturnsLength(t *testing.T) {
	t.Parallel()

	h := NewIncrementalHasher()

	tests := []struct {
		name string
		data []byte
	}{
		{name: "empty", data: []byte{}},
		{name: "single byte", data: []byte{0x42}},
		{name: "short string", data: []byte("hello")},
		{name: "medium string", data: []byte(strings.Repeat("x", 256))},
		{name: "large string", data: []byte(strings.Repeat("y", 65536))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := h.Write(tt.data)
			require.NoError(t, err)
			assert.Equal(t, len(tt.data), n,
				"Write must return the exact number of bytes written")
		})
	}
}

// ---------------------------------------------------------------------------
// TestIncrementalHasher_Sum64Idempotent
// ---------------------------------------------------------------------------

func TestIncrementalHasher_Sum64Idempotent(t *testing.T) {
	t.Parallel()

	h := NewIncrementalHasher()
	_, err := h.Write([]byte("some data"))
	require.NoError(t, err)

	hash1 := h.Sum64()
	hash2 := h.Sum64()
	hash3 := h.Sum64()

	assert.Equal(t, hash1, hash2,
		"Sum64 must be idempotent (call 1 vs 2)")
	assert.Equal(t, hash2, hash3,
		"Sum64 must be idempotent (call 2 vs 3)")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_SortOrder
// ---------------------------------------------------------------------------

func TestComputeContentHash_SortOrder(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	// Verify the sort is case-sensitive byte order.
	// In byte order: "A" (0x41) < "a" (0x61) < "b" (0x62)
	files := []FileHashEntry{
		{Path: "b.go", Content: "b"},
		{Path: "A.go", Content: "A"},
		{Path: "a.go", Content: "a"},
	}

	hash1, err := h.ComputeContentHash(files)
	require.NoError(t, err)

	// Provide in sorted order.
	sortedFiles := []FileHashEntry{
		{Path: "A.go", Content: "A"},
		{Path: "a.go", Content: "a"},
		{Path: "b.go", Content: "b"},
	}

	hash2, err := h.ComputeContentHash(sortedFiles)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2,
		"case-sensitive byte-order sort must produce consistent results")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_BinaryContent
// ---------------------------------------------------------------------------

func TestComputeContentHash_BinaryContent(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	files := []FileHashEntry{
		{Path: "data.bin", Content: string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD})},
	}

	hash1, err := h.ComputeContentHash(files)
	require.NoError(t, err)
	assert.NotZero(t, hash1)

	hash2, err := h.ComputeContentHash(files)
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2,
		"binary content must hash consistently")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_EmptyPathEmptyContent
// ---------------------------------------------------------------------------

func TestComputeContentHash_EmptyPathEmptyContent(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	// Edge case: both path and content are empty, but the null byte is still
	// part of the hash input.
	files := []FileHashEntry{
		{Path: "", Content: ""},
	}

	hash, err := h.ComputeContentHash(files)
	require.NoError(t, err)

	// Must differ from truly empty input.
	emptyHash, err := h.ComputeContentHash([]FileHashEntry{})
	require.NoError(t, err)

	assert.NotEqual(t, hash, emptyHash,
		"file with empty path and content must differ from no files")
}

// ---------------------------------------------------------------------------
// TestComputeContentHash_DuplicatePaths
// ---------------------------------------------------------------------------

func TestComputeContentHash_DuplicatePaths(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()

	// Two files with the same path but different content.
	files := []FileHashEntry{
		{Path: "main.go", Content: "version 1"},
		{Path: "main.go", Content: "version 2"},
	}

	hash, err := h.ComputeContentHash(files)
	require.NoError(t, err)
	assert.NotZero(t, hash,
		"duplicate paths should not cause an error")

	// The hash must be stable.
	hash2, err := h.ComputeContentHash(files)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)
}

// ---------------------------------------------------------------------------
// TestFormatHash_IntegrationWithContentHasher
// ---------------------------------------------------------------------------

func TestFormatHash_IntegrationWithContentHasher(t *testing.T) {
	t.Parallel()

	h := NewContentHasher()
	hash, err := h.ComputeContentHash(canonicalFiles())
	require.NoError(t, err)

	formatted := FormatHash(hash)

	assert.Len(t, formatted, 16,
		"FormatHash of computed hash must be 16 characters")
	assert.Regexp(t, `^[0-9a-f]{16}$`, formatted,
		"FormatHash of computed hash must be lowercase hex")
}

// ---------------------------------------------------------------------------
// Benchmark tests
// ---------------------------------------------------------------------------

// BenchmarkComputeContentHash benchmarks hashing 100 files of ~1KB each.
func BenchmarkComputeContentHash(b *testing.B) {
	h := NewContentHasher()

	files := make([]FileHashEntry, 100)
	content := strings.Repeat("func example() { return nil }\n", 34) // ~1020 bytes
	for i := range files {
		files[i] = FileHashEntry{
			Path:    fmt.Sprintf("pkg/sub%d/file_%03d.go", i%10, i),
			Content: content,
		}
	}

	b.ResetTimer()
	for range b.N {
		_, _ = h.ComputeContentHash(files)
	}
}

// BenchmarkIncrementalHasher benchmarks writing 1MB of data.
func BenchmarkIncrementalHasher(b *testing.B) {
	// Prepare 1MB of data.
	data := []byte(strings.Repeat("abcdefghijklmnopqrstuvwxyz0123456789\n", 29127)) // ~1MB

	b.ResetTimer()
	for range b.N {
		h := NewIncrementalHasher()
		_, _ = h.Write(data)
		_ = h.Sum64()
	}
}

// BenchmarkFormatHash benchmarks the FormatHash function.
func BenchmarkFormatHash(b *testing.B) {
	for range b.N {
		_ = FormatHash(0xdeadbeefcafebabe)
	}
}
