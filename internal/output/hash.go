package output

import (
	"fmt"
	"io"
	"sort"

	"github.com/zeebo/xxh3"
)

// Compile-time check that IncrementalHasher implements io.Writer.
var _ io.Writer = (*IncrementalHasher)(nil)

// FileHashEntry is the input for content hashing. Each entry represents a
// processed file with its relative path and final content.
type FileHashEntry struct {
	// Path is the relative file path used as part of the hash input.
	Path string

	// Content is the file content after all processing (redaction, compression).
	Content string
}

// ContentHasher computes deterministic XXH3 64-bit hashes over collections of
// files. Files are sorted by path before hashing to guarantee identical output
// regardless of input order.
type ContentHasher struct{}

// NewContentHasher creates a new ContentHasher.
func NewContentHasher() *ContentHasher {
	return &ContentHasher{}
}

// ComputeContentHash computes a deterministic XXH3 64-bit hash over all files.
// Files are sorted by Path in case-sensitive byte order before hashing. For each
// file, the hash input is: path + "\x00" + content. The null byte separator
// prevents collisions where a path suffix matches a content prefix.
//
// Returns a stable hash value: identical inputs always produce identical output
// regardless of the order of files in the input slice.
func (h *ContentHasher) ComputeContentHash(files []FileHashEntry) (uint64, error) {
	// Copy to avoid mutating caller's slice (DC-1).
	sorted := make([]FileHashEntry, len(files))
	copy(sorted, files)

	// Sort by Path using case-sensitive byte order for platform independence.
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	hasher := xxh3.New()

	for _, f := range sorted {
		// Feed path + null separator + content into the hasher.
		if _, err := hasher.Write([]byte(f.Path)); err != nil {
			return 0, fmt.Errorf("hashing path %s: %w", f.Path, err)
		}
		if _, err := hasher.Write([]byte{0x00}); err != nil {
			return 0, fmt.Errorf("hashing separator for %s: %w", f.Path, err)
		}
		if _, err := hasher.Write([]byte(f.Content)); err != nil {
			return 0, fmt.Errorf("hashing content for %s: %w", f.Path, err)
		}
	}

	return hasher.Sum64(), nil
}

// IncrementalHasher provides streaming XXH3 hash computation via the io.Writer
// interface. It is useful when computing a hash during output writing rather
// than as a separate pass over the data.
type IncrementalHasher struct {
	hasher *xxh3.Hasher
}

// NewIncrementalHasher creates a new IncrementalHasher ready for streaming use.
func NewIncrementalHasher() *IncrementalHasher {
	return &IncrementalHasher{
		hasher: xxh3.New(),
	}
}

// Write implements io.Writer by feeding bytes into the running XXH3 hash.
func (h *IncrementalHasher) Write(p []byte) (int, error) {
	return h.hasher.Write(p)
}

// Sum64 returns the current XXH3 64-bit hash value.
func (h *IncrementalHasher) Sum64() uint64 {
	return h.hasher.Sum64()
}

// FormatHash formats a 64-bit hash as a lowercase 16-character zero-padded
// hexadecimal string suitable for inclusion in output headers.
func FormatHash(h uint64) string {
	return fmt.Sprintf("%016x", h)
}
