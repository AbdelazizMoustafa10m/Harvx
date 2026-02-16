package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensureTestdataFixtures creates the binary-detection test fixture files if they
// do not already exist. This is called from individual tests that use fixtures.
// The binary.bin file contains PNG-like header bytes with null bytes, and the
// empty.txt file is a zero-length file.
func ensureTestdataFixtures(t *testing.T) string {
	t.Helper()

	fixtureDir := filepath.Join("..", "..", "testdata", "binary-detection")
	if _, err := os.Stat(fixtureDir); err != nil {
		t.Skipf("testdata fixture directory not found: %s", fixtureDir)
	}

	// Create binary.bin if it does not exist.
	binPath := filepath.Join(fixtureDir, "binary.bin")
	if _, err := os.Stat(binPath); err != nil {
		// PNG-like header: magic bytes with null bytes embedded.
		binContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}
		if writeErr := os.WriteFile(binPath, binContent, 0o644); writeErr != nil {
			t.Logf("could not create binary fixture: %v", writeErr)
		}
	}

	// Create empty.txt if it does not exist.
	emptyPath := filepath.Join(fixtureDir, "empty.txt")
	if _, err := os.Stat(emptyPath); err != nil {
		if writeErr := os.WriteFile(emptyPath, []byte{}, 0o644); writeErr != nil {
			t.Logf("could not create empty fixture: %v", writeErr)
		}
	}

	return fixtureDir
}

// Helper to create a file with the given content in a temp directory.
// Returns the absolute path to the created file.
func createTestFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func TestIsBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content []byte
		wantBin bool
	}{
		{
			name:    "plain text file is not binary",
			content: []byte("Hello, this is a plain text file.\nNo binary content here.\n"),
			wantBin: false,
		},
		{
			name:    "empty file is not binary",
			content: []byte{},
			wantBin: false,
		},
		{
			name:    "file with null byte is binary",
			content: []byte("some text\x00more text"),
			wantBin: true,
		},
		{
			name:    "file starting with null byte is binary",
			content: []byte{0x00, 'h', 'e', 'l', 'l', 'o'},
			wantBin: true,
		},
		{
			name:    "file ending with null byte is binary",
			content: []byte{'h', 'e', 'l', 'l', 'o', 0x00},
			wantBin: true,
		},
		{
			name: "PNG header bytes are binary",
			// PNG magic bytes: 89 50 4E 47 0D 0A 1A 0A
			content: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D},
			wantBin: true,
		},
		{
			name: "JPEG header bytes are binary",
			// JPEG starts with FF D8 FF, contains null bytes in data
			content: []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00},
			wantBin: true,
		},
		{
			name: "ELF binary header is binary",
			// ELF magic: 7F 45 4C 46 followed by binary content
			content: []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x01, 0x01, 0x00, 0x00, 0x00},
			wantBin: true,
		},
		{
			name:    "UTF-8 text with multibyte characters is not binary",
			content: []byte("Hello, 世界! Привет! こんにちは 🌍"),
			wantBin: false,
		},
		{
			name:    "file with only whitespace is not binary",
			content: []byte("   \t\t\n\n\r\n   \t   \n"),
			wantBin: false,
		},
		{
			name:    "file with high-bit bytes but no null is not binary",
			content: []byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA},
			wantBin: false,
		},
		{
			name:    "single newline is not binary",
			content: []byte{'\n'},
			wantBin: false,
		},
		{
			name:    "single null byte is binary",
			content: []byte{0x00},
			wantBin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := createTestFile(t, dir, "testfile", tt.content)

			got, err := IsBinary(path)
			require.NoError(t, err)
			assert.Equal(t, tt.wantBin, got)
		})
	}
}

func TestIsBinary_NullByteAfter8KB(t *testing.T) {
	t.Parallel()

	// Create a file where the null byte appears AFTER the first 8KB.
	// IsBinary only reads the first 8192 bytes, so this should NOT be detected as binary.
	dir := t.TempDir()

	// Fill first 8192 bytes with 'A', then place a null byte at position 8192.
	content := make([]byte, BinaryDetectionBytes+100)
	for i := range content {
		content[i] = 'A'
	}
	content[BinaryDetectionBytes] = 0x00 // null byte at byte 8193

	path := createTestFile(t, dir, "null-after-8kb", content)

	got, err := IsBinary(path)
	require.NoError(t, err)
	assert.False(t, got, "null byte after first 8KB should not be detected")
}

func TestIsBinary_NullByteAtEnd8KB(t *testing.T) {
	t.Parallel()

	// Create a file where the null byte is at the last position within the first 8KB.
	dir := t.TempDir()

	content := make([]byte, BinaryDetectionBytes)
	for i := range content {
		content[i] = 'A'
	}
	content[BinaryDetectionBytes-1] = 0x00 // null byte at last byte of 8KB

	path := createTestFile(t, dir, "null-at-end-8kb", content)

	got, err := IsBinary(path)
	require.NoError(t, err)
	assert.True(t, got, "null byte at end of first 8KB should be detected")
}

func TestIsBinary_PermissionDenied(t *testing.T) {
	t.Parallel()

	// Skip on systems where we might be running as root.
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	path := createTestFile(t, dir, "no-perms", []byte("content"))
	require.NoError(t, os.Chmod(path, 0o000))
	t.Cleanup(func() {
		// Restore permissions for cleanup.
		os.Chmod(path, 0o644)
	})

	_, err := IsBinary(path)
	assert.Error(t, err, "should return error for permission denied")
	assert.ErrorIs(t, err, os.ErrPermission)
}

func TestIsBinary_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := IsBinary("/nonexistent/path/to/file")
	assert.Error(t, err, "should return error for missing file")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestIsBinary_WithTestdataFixtures(t *testing.T) {
	t.Parallel()

	fixtureDir := ensureTestdataFixtures(t)

	t.Run("text fixture is not binary", func(t *testing.T) {
		t.Parallel()
		textPath := filepath.Join(fixtureDir, "text.txt")
		got, err := IsBinary(textPath)
		require.NoError(t, err)
		assert.False(t, got, "testdata text.txt should not be binary")
	})

	t.Run("binary fixture is binary", func(t *testing.T) {
		t.Parallel()
		binPath := filepath.Join(fixtureDir, "binary.bin")
		got, err := IsBinary(binPath)
		require.NoError(t, err)
		assert.True(t, got, "testdata binary.bin should be binary")
	})

	t.Run("empty fixture is not binary", func(t *testing.T) {
		t.Parallel()
		emptyPath := filepath.Join(fixtureDir, "empty.txt")
		got, err := IsBinary(emptyPath)
		require.NoError(t, err)
		assert.False(t, got, "testdata empty.txt should not be binary")
	})
}

func TestIsBinary_Symlink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create a text file and a symlink pointing to it.
	textPath := createTestFile(t, dir, "original.txt", []byte("plain text content"))
	symlinkPath := filepath.Join(dir, "link.txt")
	require.NoError(t, os.Symlink(textPath, symlinkPath))

	got, err := IsBinary(symlinkPath)
	require.NoError(t, err)
	assert.False(t, got, "symlink to text file should not be binary")

	// Create a binary file and a symlink pointing to it.
	binPath := createTestFile(t, dir, "original.bin", []byte{0x89, 0x50, 0x4E, 0x47, 0x00})
	binSymlinkPath := filepath.Join(dir, "link.bin")
	require.NoError(t, os.Symlink(binPath, binSymlinkPath))

	got, err = IsBinary(binSymlinkPath)
	require.NoError(t, err)
	assert.True(t, got, "symlink to binary file should be binary")
}

func TestIsBinary_LargeTextFile(t *testing.T) {
	t.Parallel()

	// Create a large text file (10MB) to verify only first 8KB is read.
	dir := t.TempDir()
	content := make([]byte, 10*1024*1024) // 10MB
	for i := range content {
		content[i] = 'A'
	}
	path := createTestFile(t, dir, "large.txt", content)

	got, err := IsBinary(path)
	require.NoError(t, err)
	assert.False(t, got, "large text file should not be binary")
}

func TestIsLargeFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		size      int
		maxBytes  int64
		wantLarge bool
		wantSize  int64
	}{
		{
			name:      "file under threshold",
			size:      100,
			maxBytes:  DefaultMaxFileSize,
			wantLarge: false,
			wantSize:  100,
		},
		{
			name:      "file exactly at threshold",
			size:      int(DefaultMaxFileSize),
			maxBytes:  DefaultMaxFileSize,
			wantLarge: false,
			wantSize:  DefaultMaxFileSize,
		},
		{
			name:      "file over threshold",
			size:      int(DefaultMaxFileSize) + 1,
			maxBytes:  DefaultMaxFileSize,
			wantLarge: true,
			wantSize:  DefaultMaxFileSize + 1,
		},
		{
			name:      "empty file with zero threshold",
			size:      0,
			maxBytes:  0,
			wantLarge: false,
			wantSize:  0,
		},
		{
			name:      "non-empty file with zero threshold",
			size:      1,
			maxBytes:  0,
			wantLarge: true,
			wantSize:  1,
		},
		{
			name:      "small file with small threshold",
			size:      100,
			maxBytes:  50,
			wantLarge: true,
			wantSize:  100,
		},
		{
			name:      "empty file is not large",
			size:      0,
			maxBytes:  DefaultMaxFileSize,
			wantLarge: false,
			wantSize:  0,
		},
		{
			name:      "1KB file with 1KB threshold",
			size:      1024,
			maxBytes:  1024,
			wantLarge: false,
			wantSize:  1024,
		},
		{
			name:      "1KB+1 file with 1KB threshold",
			size:      1025,
			maxBytes:  1024,
			wantLarge: true,
			wantSize:  1025,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()

			content := make([]byte, tt.size)
			for i := range content {
				content[i] = 'x'
			}
			path := createTestFile(t, dir, "testfile", content)

			large, size, err := IsLargeFile(path, tt.maxBytes)
			require.NoError(t, err)
			assert.Equal(t, tt.wantLarge, large)
			assert.Equal(t, tt.wantSize, size)
		})
	}
}

func TestIsLargeFile_PermissionDenied(t *testing.T) {
	t.Parallel()

	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()

	// Remove all permissions from the parent directory so Stat fails.
	// On macOS/Linux, os.Stat on the file itself still works with 000 perms
	// on the file. We need to make the containing directory unreadable.
	subDir := filepath.Join(dir, "restricted")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	restrictedFile := createTestFile(t, subDir, "file.txt", []byte("content"))
	require.NoError(t, os.Chmod(subDir, 0o000))
	t.Cleanup(func() {
		os.Chmod(subDir, 0o755)
	})

	_, _, err := IsLargeFile(restrictedFile, DefaultMaxFileSize)
	assert.Error(t, err, "should return error for permission denied")
}

func TestIsLargeFile_FileNotFound(t *testing.T) {
	t.Parallel()

	_, _, err := IsLargeFile("/nonexistent/path/to/file", DefaultMaxFileSize)
	assert.Error(t, err, "should return error for missing file")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestIsLargeFile_Symlink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create a small file and a symlink.
	content := []byte(strings.Repeat("x", 100))
	origPath := createTestFile(t, dir, "original.txt", content)
	symlinkPath := filepath.Join(dir, "link.txt")
	require.NoError(t, os.Symlink(origPath, symlinkPath))

	// IsLargeFile follows symlinks (os.Stat follows them).
	large, size, err := IsLargeFile(symlinkPath, DefaultMaxFileSize)
	require.NoError(t, err)
	assert.False(t, large)
	assert.Equal(t, int64(100), size)
}

func TestConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 8192, BinaryDetectionBytes, "BinaryDetectionBytes should be 8192")
	assert.Equal(t, int64(1_048_576), DefaultMaxFileSize, "DefaultMaxFileSize should be 1MB (1,048,576 bytes)")
}

// BenchmarkIsBinary_LargeFile verifies that IsBinary reads only the first 8KB
// of a large file, not the entire content.
func BenchmarkIsBinary_LargeFile(b *testing.B) {
	dir := b.TempDir()

	// Create a 10MB text file.
	content := make([]byte, 10*1024*1024)
	for i := range content {
		content[i] = 'A'
	}
	path := filepath.Join(dir, "large.txt")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := IsBinary(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIsBinary_SmallFile benchmarks binary detection on a small file.
func BenchmarkIsBinary_SmallFile(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "small.txt")
	if err := os.WriteFile(path, []byte("hello world\n"), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := IsBinary(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIsBinary_BinaryFile benchmarks binary detection on a file with null bytes.
func BenchmarkIsBinary_BinaryFile(b *testing.B) {
	dir := b.TempDir()
	content := make([]byte, 4096)
	for i := range content {
		content[i] = 'A'
	}
	content[2048] = 0x00 // null byte in the middle
	path := filepath.Join(dir, "binary.bin")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := IsBinary(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIsLargeFile benchmarks the large file size check.
func BenchmarkIsLargeFile(b *testing.B) {
	dir := b.TempDir()
	content := make([]byte, 2*1024*1024) // 2MB
	path := filepath.Join(dir, "large.bin")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := IsLargeFile(path, DefaultMaxFileSize)
		if err != nil {
			b.Fatal(err)
		}
	}
}
