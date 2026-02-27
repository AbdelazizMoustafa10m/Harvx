package diff

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeebo/xxh3"

	"github.com/harvx/harvx/internal/pipeline"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// createTestFile writes content to a file under the given directory and returns
// its absolute path. It calls t.Helper so failures are reported at the caller.
func createTestFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, content, 0644))
	return path
}

// ---------------------------------------------------------------------------
// TestNewXXH3Hasher
// ---------------------------------------------------------------------------

func TestNewXXH3Hasher(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()
	require.NotNil(t, h, "NewXXH3Hasher must return a non-nil value")
}

// ---------------------------------------------------------------------------
// TestXXH3Hasher_Interface
// ---------------------------------------------------------------------------

func TestXXH3Hasher_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check is already in xxh3.go via:
	//   var _ Hasher = (*XXH3Hasher)(nil)
	// This test verifies it at runtime as well.
	var h Hasher = NewXXH3Hasher()
	require.NotNil(t, h)

	// Verify the interface methods are callable.
	_ = h.HashBytes([]byte("test"))
	_ = h.HashString("test")
}

// ---------------------------------------------------------------------------
// TestXXH3Hasher_HashBytes
// ---------------------------------------------------------------------------

func TestXXH3Hasher_HashBytes(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	tests := []struct {
		name string
		data []byte
		want uint64
	}{
		{
			name: "known string hello world",
			data: []byte("hello world"),
			want: xxh3.Hash([]byte("hello world")),
		},
		{
			name: "empty input",
			data: []byte{},
			want: xxh3.Hash([]byte{}),
		},
		{
			name: "single byte",
			data: []byte{0x42},
			want: xxh3.Hash([]byte{0x42}),
		},
		{
			name: "null bytes",
			data: []byte{0x00, 0x00, 0x00},
			want: xxh3.Hash([]byte{0x00, 0x00, 0x00}),
		},
		{
			name: "binary PNG header",
			data: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			want: xxh3.Hash([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}),
		},
		{
			name: "unicode content",
			data: []byte("cafe\u0301 \xe4\xb8\xad\xe6\x96\x87"),
			want: xxh3.Hash([]byte("cafe\u0301 \xe4\xb8\xad\xe6\x96\x87")),
		},
		{
			name: "UTF-8 BOM prefix",
			data: append([]byte{0xEF, 0xBB, 0xBF}, []byte("hello")...),
			want: xxh3.Hash(append([]byte{0xEF, 0xBB, 0xBF}, []byte("hello")...)),
		},
		{
			name: "large 1KB content",
			data: bytes.Repeat([]byte("abcdefghij"), 100),
			want: xxh3.Hash(bytes.Repeat([]byte("abcdefghij"), 100)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := h.HashBytes(tt.data)
			assert.Equal(t, tt.want, got)

			// Verify determinism: hash the same input again.
			got2 := h.HashBytes(tt.data)
			assert.Equal(t, got, got2, "HashBytes must be deterministic")
		})
	}
}

func TestXXH3Hasher_HashBytes_Determinism(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()
	data := []byte("deterministic content for hashing test")

	first := h.HashBytes(data)
	for i := range 100 {
		got := h.HashBytes(data)
		assert.Equal(t, first, got,
			"HashBytes must be deterministic across 100 calls (iteration %d)", i)
	}
}

func TestXXH3Hasher_HashBytes_DifferentContent(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	hash1 := h.HashBytes([]byte("input one"))
	hash2 := h.HashBytes([]byte("input two"))

	assert.NotEqual(t, hash1, hash2,
		"different content must produce different hashes")
}

func TestXXH3Hasher_HashBytes_SameContentIdentical(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	// Two separate byte slices with identical content.
	a := []byte("identical content")
	b := []byte("identical content")

	assert.Equal(t, h.HashBytes(a), h.HashBytes(b),
		"identical content in different slices must produce identical hashes")
}

func TestXXH3Hasher_HashBytes_EmptyConsistency(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	hash1 := h.HashBytes([]byte{})
	hash2 := h.HashBytes([]byte{})
	hash3 := h.HashBytes(nil)

	assert.Equal(t, hash1, hash2, "empty byte slice hashes must be identical")
	assert.Equal(t, hash1, hash3, "nil and empty byte slice must produce the same hash")
}

func TestXXH3Hasher_HashBytes_EmptyNonZero(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	// XXH3 of empty input is a well-defined non-zero value.
	bytesHash := h.HashBytes([]byte{})
	assert.NotEqual(t, uint64(0), bytesHash,
		"XXH3 of empty bytes must be non-zero")
}

// ---------------------------------------------------------------------------
// TestXXH3Hasher_HashString
// ---------------------------------------------------------------------------

func TestXXH3Hasher_HashString(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	tests := []struct {
		name string
		s    string
		want uint64
	}{
		{
			name: "known string hello world",
			s:    "hello world",
			want: xxh3.HashString("hello world"),
		},
		{
			name: "empty string",
			s:    "",
			want: xxh3.HashString(""),
		},
		{
			name: "single character",
			s:    "x",
			want: xxh3.HashString("x"),
		},
		{
			name: "unicode Japanese",
			s:    "\u3053\u3093\u306b\u3061\u306f\u4e16\u754c",
			want: xxh3.HashString("\u3053\u3093\u306b\u3061\u306f\u4e16\u754c"),
		},
		{
			name: "newlines",
			s:    "line1\nline2\nline3",
			want: xxh3.HashString("line1\nline2\nline3"),
		},
		{
			name: "long string",
			s:    string(bytes.Repeat([]byte("repeat"), 500)),
			want: xxh3.HashString(string(bytes.Repeat([]byte("repeat"), 500))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := h.HashString(tt.s)
			assert.Equal(t, tt.want, got)

			// Verify determinism.
			got2 := h.HashString(tt.s)
			assert.Equal(t, got, got2, "HashString must be deterministic")
		})
	}
}

func TestXXH3Hasher_HashString_EmptyNonZero(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	stringHash := h.HashString("")
	assert.NotEqual(t, uint64(0), stringHash,
		"XXH3 of empty string must be non-zero")
}

func TestXXH3Hasher_HashString_DifferentStrings(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	hash1 := h.HashString("alpha")
	hash2 := h.HashString("beta")

	assert.NotEqual(t, hash1, hash2,
		"different strings must produce different hashes")
}

func TestXXH3Hasher_HashString_MatchesHashBytes(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	// HashString("s") must equal HashBytes([]byte("s")) for all inputs.
	tests := []struct {
		name  string
		input string
	}{
		{name: "hello", input: "hello"},
		{name: "empty", input: ""},
		{name: "sentence", input: "the quick brown fox jumps over the lazy dog"},
		{name: "null bytes", input: "\x00\x01\x02"},
		{name: "long repeated", input: string(bytes.Repeat([]byte("abc"), 1000))},
		{name: "go source", input: "package main\n\nfunc main() {}\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hashString := h.HashString(tt.input)
			hashBytes := h.HashBytes([]byte(tt.input))

			assert.Equal(t, hashBytes, hashString,
				"HashString and HashBytes must produce identical hashes")
		})
	}
}

func TestXXH3Hasher_HashBytes_HashString_EmptyConsistency(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	bytesHash := h.HashBytes([]byte{})
	stringHash := h.HashString("")

	assert.Equal(t, bytesHash, stringHash,
		"empty bytes and empty string must hash the same")
}

// ---------------------------------------------------------------------------
// TestXXH3Hasher_HashFile
// ---------------------------------------------------------------------------

func TestXXH3Hasher_HashFile(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "known content matches HashBytes",
			content: []byte("package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"),
		},
		{
			name:    "empty file",
			content: []byte{},
		},
		{
			name:    "binary content",
			content: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE},
		},
		{
			name:    "single byte",
			content: []byte{0x42},
		},
		{
			name:    "1KB file",
			content: bytes.Repeat([]byte("data line\n"), 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := createTestFile(t, dir, "testfile", tt.content)

			got, err := h.HashFile(path)
			require.NoError(t, err)

			// Cross-verify: HashFile must match HashBytes of the same content.
			expected := h.HashBytes(tt.content)
			assert.Equal(t, expected, got,
				"HashFile must produce the same hash as HashBytes for identical content")
		})
	}
}

func TestXXH3Hasher_HashFile_Determinism(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()
	dir := t.TempDir()
	path := createTestFile(t, dir, "stable.txt", []byte("stable content for determinism"))

	first, err := h.HashFile(path)
	require.NoError(t, err)

	for i := range 50 {
		got, err := h.HashFile(path)
		require.NoError(t, err)
		assert.Equal(t, first, got,
			"HashFile must be deterministic (iteration %d)", i)
	}
}

func TestXXH3Hasher_HashFile_NonExistent(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	_, err := h.HashFile("/tmp/nonexistent-file-xxh3-test-abc123")
	require.Error(t, err, "HashFile must return an error for a non-existent file")
	assert.Contains(t, err.Error(), "hashing file")
}

func TestXXH3Hasher_HashFile_EmptyFile(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()
	dir := t.TempDir()
	path := createTestFile(t, dir, "empty.txt", []byte{})

	got, err := h.HashFile(path)
	require.NoError(t, err)

	// Must match HashBytes of empty input and produce a non-zero hash.
	expected := h.HashBytes([]byte{})
	assert.Equal(t, expected, got,
		"HashFile on empty file must match HashBytes of empty input")
	assert.NotZero(t, got,
		"hash of empty file must be non-zero")
}

func TestXXH3Hasher_HashFile_LargeFile(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()
	dir := t.TempDir()

	// Create a 1MB file with random data to verify streaming works.
	data := make([]byte, 1<<20) // 1 MiB
	_, err := rand.Read(data)
	require.NoError(t, err)

	path := createTestFile(t, dir, "large.bin", data)

	got, err := h.HashFile(path)
	require.NoError(t, err)

	want := xxh3.Hash(data)
	assert.Equal(t, want, got,
		"HashFile on a large file must match xxh3.Hash of the same data")
}

func TestXXH3Hasher_HashFile_Directory(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()
	dir := t.TempDir()

	_, err := h.HashFile(dir)
	require.Error(t, err, "HashFile must return an error when given a directory path")
}

// ---------------------------------------------------------------------------
// TestXXH3Hasher_HashFileDescriptors
// ---------------------------------------------------------------------------

func TestXXH3Hasher_HashFileDescriptors_WithContent(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	fds := []pipeline.FileDescriptor{
		{
			Path:    "main.go",
			Content: "package main",
		},
		{
			Path:    "README.md",
			Content: "# Hello",
		},
	}

	err := h.HashFileDescriptors(fds)
	require.NoError(t, err)

	assert.Equal(t, xxh3.HashString("package main"), fds[0].ContentHash)
	assert.Equal(t, xxh3.HashString("# Hello"), fds[1].ContentHash)
}

func TestXXH3Hasher_HashFileDescriptors_WithAbsPath(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()
	dir := t.TempDir()

	fileContent := []byte("file on disk")
	path := createTestFile(t, dir, "disk.txt", fileContent)

	fds := []pipeline.FileDescriptor{
		{
			Path:    "disk.txt",
			AbsPath: path,
			// Content is empty, should fall back to reading AbsPath.
		},
	}

	err := h.HashFileDescriptors(fds)
	require.NoError(t, err)

	assert.Equal(t, xxh3.Hash(fileContent), fds[0].ContentHash,
		"ContentHash must be computed from file when Content is empty")
}

func TestXXH3Hasher_HashFileDescriptors_ContentTakesPrecedence(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()
	dir := t.TempDir()

	diskContent := []byte("disk content")
	path := createTestFile(t, dir, "file.txt", diskContent)

	memoryContent := "memory content"
	fds := []pipeline.FileDescriptor{
		{
			Path:    "file.txt",
			AbsPath: path,
			Content: memoryContent,
		},
	}

	err := h.HashFileDescriptors(fds)
	require.NoError(t, err)

	// Content field must take precedence over AbsPath.
	assert.Equal(t, xxh3.HashString(memoryContent), fds[0].ContentHash,
		"Content field must take precedence over AbsPath")
	assert.NotEqual(t, xxh3.Hash(diskContent), fds[0].ContentHash,
		"hash must not come from disk when Content is populated")
}

func TestXXH3Hasher_HashFileDescriptors_Mixed(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()
	dir := t.TempDir()

	diskContent := []byte("from disk")
	diskPath := createTestFile(t, dir, "disk.go", diskContent)

	fds := []pipeline.FileDescriptor{
		{
			Path:    "mem.go",
			Content: "from memory",
		},
		{
			Path:    "disk.go",
			AbsPath: diskPath,
		},
		{
			Path: "nothing.go",
			// Neither Content nor AbsPath set.
		},
	}

	err := h.HashFileDescriptors(fds)
	require.NoError(t, err)

	assert.Equal(t, xxh3.HashString("from memory"), fds[0].ContentHash)
	assert.Equal(t, xxh3.Hash(diskContent), fds[1].ContentHash)
	assert.Equal(t, uint64(0), fds[2].ContentHash,
		"descriptor with no Content and no AbsPath should keep zero hash")
}

func TestXXH3Hasher_HashFileDescriptors_EmptySlice(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	err := h.HashFileDescriptors(nil)
	require.NoError(t, err, "nil slice must not cause an error")

	err = h.HashFileDescriptors([]pipeline.FileDescriptor{})
	require.NoError(t, err, "empty slice must not cause an error")
}

func TestXXH3Hasher_HashFileDescriptors_NonExistentAbsPath(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	fds := []pipeline.FileDescriptor{
		{
			Path:    "missing.go",
			AbsPath: "/tmp/nonexistent-file-xxh3-test-abc123",
			// Content is empty, so it tries to read AbsPath.
		},
	}

	err := h.HashFileDescriptors(fds)
	require.Error(t, err, "non-existent AbsPath with empty Content must return an error")
	assert.Contains(t, err.Error(), "hashing descriptor")
	assert.Contains(t, err.Error(), "missing.go")
}

func TestXXH3Hasher_HashFileDescriptors_EmptyContentEmptyFile(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()
	dir := t.TempDir()
	absPath := createTestFile(t, dir, "empty.go", []byte{})

	fds := []pipeline.FileDescriptor{
		{
			Path:    "empty.go",
			AbsPath: absPath,
			Content: "",
		},
	}

	err := h.HashFileDescriptors(fds)
	require.NoError(t, err)

	// Hash should match HashBytes of empty content (via streaming the empty file).
	expected := h.HashBytes([]byte{})
	assert.Equal(t, expected, fds[0].ContentHash,
		"empty file on disk should produce hash matching HashBytes of empty input")
}

func TestXXH3Hasher_HashFileDescriptors_PreservesOtherFields(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	fds := []pipeline.FileDescriptor{
		{
			Path:       "main.go",
			AbsPath:    "/repo/main.go",
			Size:       42,
			Tier:       1,
			Language:   "go",
			Content:    "package main",
			TokenCount: 2,
			IsBinary:   false,
			IsSymlink:  true,
			Redactions: 3,
		},
	}

	err := h.HashFileDescriptors(fds)
	require.NoError(t, err)

	// ContentHash should be set.
	assert.NotZero(t, fds[0].ContentHash)

	// All other fields must remain unchanged.
	assert.Equal(t, "main.go", fds[0].Path)
	assert.Equal(t, "/repo/main.go", fds[0].AbsPath)
	assert.Equal(t, int64(42), fds[0].Size)
	assert.Equal(t, 1, fds[0].Tier)
	assert.Equal(t, "go", fds[0].Language)
	assert.Equal(t, "package main", fds[0].Content)
	assert.Equal(t, 2, fds[0].TokenCount)
	assert.False(t, fds[0].IsBinary)
	assert.True(t, fds[0].IsSymlink)
	assert.Equal(t, 3, fds[0].Redactions)
}

func TestXXH3Hasher_HashFileDescriptors_MultipleDistinctHashes(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	fds := []pipeline.FileDescriptor{
		{Path: "a.go", Content: "package a"},
		{Path: "b.go", Content: "package b"},
		{Path: "c.go", Content: "package c"},
		{Path: "d.go", Content: "package d"},
		{Path: "e.go", Content: "package e"},
	}

	err := h.HashFileDescriptors(fds)
	require.NoError(t, err)

	// All hashes should be set and each should match individual HashString.
	for i, fd := range fds {
		expected := h.HashString(fd.Content)
		assert.Equal(t, expected, fd.ContentHash,
			"FileDescriptor[%d] (%s) hash mismatch", i, fd.Path)
	}

	// All hashes should be distinct since content differs.
	seen := make(map[uint64]string, len(fds))
	for _, fd := range fds {
		if existing, ok := seen[fd.ContentHash]; ok {
			t.Errorf("duplicate hash: %s and %s both hash to %d",
				fd.Path, existing, fd.ContentHash)
		}
		seen[fd.ContentHash] = fd.Path
	}
}

func TestXXH3Hasher_HashFileDescriptors_LargeSlice(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	// Hash 1000 descriptors with distinct content.
	fds := make([]pipeline.FileDescriptor, 1000)
	for i := range fds {
		fds[i] = pipeline.FileDescriptor{
			Path:    fmt.Sprintf("pkg/sub%d/file_%04d.go", i%10, i),
			Content: fmt.Sprintf("package pkg%d\n\nfunc F%d() {}\n", i, i),
		}
	}

	err := h.HashFileDescriptors(fds)
	require.NoError(t, err)

	for i, fd := range fds {
		assert.NotZero(t, fd.ContentHash,
			"FileDescriptor[%d] ContentHash must be non-zero", i)
	}
}

// ---------------------------------------------------------------------------
// TestXXH3Hasher_CollisionResistance
// ---------------------------------------------------------------------------

func TestXXH3Hasher_CollisionResistance(t *testing.T) {
	t.Parallel()

	h := NewXXH3Hasher()

	// Hash many similar but distinct inputs and verify no collisions.
	hashes := make(map[uint64]string, 10000)
	for i := range 10000 {
		input := fmt.Sprintf("input-%d-suffix", i)
		hash := h.HashString(input)
		if existing, ok := hashes[hash]; ok {
			t.Errorf("collision detected: %q and %q both hash to %d",
				input, existing, hash)
		}
		hashes[hash] = input
	}
}

// ---------------------------------------------------------------------------
// Benchmark tests
// ---------------------------------------------------------------------------

func BenchmarkHashBytes(b *testing.B) {
	h := NewXXH3Hasher()

	sizes := []struct {
		name string
		size int
	}{
		{name: "1KB", size: 1024},
		{name: "64KB", size: 64 * 1024},
		{name: "1MB", size: 1024 * 1024},
	}

	for _, s := range sizes {
		data := make([]byte, s.size)
		_, _ = rand.Read(data)

		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(s.size))
			b.ResetTimer()
			for range b.N {
				h.HashBytes(data)
			}
		})
	}
}

func BenchmarkHashString(b *testing.B) {
	h := NewXXH3Hasher()

	data := make([]byte, 1024)
	_, _ = rand.Read(data)
	s := string(data)

	b.SetBytes(1024)
	b.ResetTimer()
	for range b.N {
		h.HashString(s)
	}
}

func BenchmarkHashFile(b *testing.B) {
	h := NewXXH3Hasher()

	// Create a 1MB temp file.
	data := make([]byte, 1024*1024)
	_, _ = rand.Read(data)

	dir := b.TempDir()
	path := filepath.Join(dir, "bench.bin")
	require.NoError(b, os.WriteFile(path, data, 0644))

	b.SetBytes(1024 * 1024)
	b.ResetTimer()
	for range b.N {
		_, err := h.HashFile(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashFileDescriptors(b *testing.B) {
	h := NewXXH3Hasher()

	// 100 descriptors with ~1KB content each.
	content := string(bytes.Repeat([]byte("Z"), 1024))
	fds := make([]pipeline.FileDescriptor, 100)
	for i := range fds {
		fds[i] = pipeline.FileDescriptor{
			Path:    fmt.Sprintf("file_%03d.go", i),
			Content: content,
		}
	}

	b.ResetTimer()
	for range b.N {
		// Reset hashes before each iteration.
		for i := range fds {
			fds[i].ContentHash = 0
		}
		err := h.HashFileDescriptors(fds)
		if err != nil {
			b.Fatal(err)
		}
	}
}
