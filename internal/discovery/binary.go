package discovery

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// BinaryDetectionBytes is the number of bytes read from the beginning of a
// file to detect binary content. This matches Git's approach of checking the
// first 8KB for null bytes. Reading only 8KB ensures performance remains
// constant regardless of file size.
const BinaryDetectionBytes = 8192

// DefaultMaxFileSize is the default maximum file size in bytes (1MB). Files
// exceeding this threshold are skipped by the discovery pipeline when the
// --skip-large-files feature is enabled. The value is configurable via the
// --skip-large-files flag.
const DefaultMaxFileSize int64 = 1_048_576

// IsBinary reports whether the file at the given path contains binary content.
// Binary detection reads the first 8192 bytes (8KB) of the file and checks
// for the presence of any null byte (\x00), matching Git's approach.
//
// An empty file (0 bytes) is NOT considered binary.
// Files that cannot be opened or read return an error.
//
// This function is safe for concurrent use -- it has no shared mutable state.
// Each call opens and closes its own file handle.
func IsBinary(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("opening %s for binary detection: %w", path, err)
	}
	defer f.Close()

	buf := make([]byte, BinaryDetectionBytes)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("reading %s for binary detection: %w", path, err)
	}

	// Empty file is not binary.
	if n == 0 {
		return false, nil
	}

	// Use bytes.IndexByte for efficient null byte detection (assembly-optimized).
	return bytes.IndexByte(buf[:n], 0) != -1, nil
}

// IsLargeFile reports whether the file at the given path exceeds the specified
// maximum size in bytes. It uses os.Stat to check the file size without reading
// content, making it efficient for large files.
//
// Returns:
//   - large: true if the file size exceeds maxBytes
//   - size: the actual file size in bytes
//   - err: any error from os.Stat (permission denied, file not found, etc.)
//
// A maxBytes of 0 means all non-empty files are considered large.
// This function is safe for concurrent use -- it has no shared mutable state.
func IsLargeFile(path string, maxBytes int64) (large bool, size int64, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, 0, fmt.Errorf("stat %s for size check: %w", path, err)
	}

	size = info.Size()
	return size > maxBytes, size, nil
}
