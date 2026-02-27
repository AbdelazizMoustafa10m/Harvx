package diff

import (
	"fmt"
	"io"
	"os"

	"github.com/zeebo/xxh3"

	"github.com/harvx/harvx/internal/pipeline"
)

// Compile-time interface check: XXH3Hasher must satisfy Hasher.
var _ Hasher = (*XXH3Hasher)(nil)

// XXH3Hasher implements the Hasher interface using the zeebo/xxh3 library.
// It provides deterministic XXH3 64-bit hashing for byte slices, strings,
// files, and pipeline FileDescriptor slices. The struct is stateless; all
// hashing state is created per-call.
type XXH3Hasher struct{}

// NewXXH3Hasher creates a new XXH3Hasher.
func NewXXH3Hasher() *XXH3Hasher {
	return &XXH3Hasher{}
}

// HashBytes computes the XXH3 64-bit hash of a byte slice.
func (h *XXH3Hasher) HashBytes(data []byte) uint64 {
	return xxh3.Hash(data)
}

// HashString computes the XXH3 64-bit hash of a string without allocating
// a copy of the string data.
func (h *XXH3Hasher) HashString(s string) uint64 {
	return xxh3.HashString(s)
}

// HashFile opens the file at path and computes its XXH3 64-bit hash using
// streaming I/O. The file is read in 32KB chunks via io.Copy, so arbitrarily
// large files can be hashed without loading them entirely into memory.
func (h *XXH3Hasher) HashFile(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("hashing file %s: %w", path, err)
	}
	defer f.Close()

	hasher := xxh3.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return 0, fmt.Errorf("hashing file %s: %w", path, err)
	}

	return hasher.Sum64(), nil
}

// HashFileDescriptors iterates the slice and populates each element's
// ContentHash field. If a descriptor has non-empty Content, the Content string
// is hashed directly. Otherwise, if AbsPath is non-empty, the file at AbsPath
// is hashed from disk using streaming I/O. Descriptors with both Content and
// AbsPath empty are skipped. The slice is modified in place via index access.
// Returns the first error encountered, wrapped with context.
func (h *XXH3Hasher) HashFileDescriptors(fds []pipeline.FileDescriptor) error {
	for i := range fds {
		if fds[i].Content != "" {
			fds[i].ContentHash = h.HashString(fds[i].Content)
			continue
		}

		if fds[i].AbsPath != "" {
			hash, err := h.HashFile(fds[i].AbsPath)
			if err != nil {
				return fmt.Errorf("hashing descriptor %s: %w", fds[i].Path, err)
			}
			fds[i].ContentHash = hash
			continue
		}
	}

	return nil
}
