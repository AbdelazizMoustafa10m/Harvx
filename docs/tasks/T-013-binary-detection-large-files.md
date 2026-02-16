# T-013: Binary File Detection & Large File Skipping

**Priority:** Must Have
**Effort:** Small (3-5hrs)
**Dependencies:** T-001, T-003
**Phase:** 1 - Foundation

---

## Description

Implement binary file detection by inspecting the first 8KB of each file for null bytes (matching Git's approach), and implement the `--skip-large-files` feature that skips files exceeding a configurable size threshold (default: 1MB). Both are critical filters in the file discovery pipeline that prevent binary blobs and oversized generated files from polluting the context output.

## User Story

As a developer, I want Harvx to automatically skip binary files like images and compiled outputs, and skip oversized generated files, so that my context output contains only useful text content.

## Acceptance Criteria

- [ ] `internal/discovery/binary.go` defines an `IsBinary(path string) (bool, error)` function
- [ ] Binary detection reads the first 8192 bytes (8KB) of the file
- [ ] A file is considered binary if any null byte (`\x00`) is found in the first 8KB
- [ ] Empty files (0 bytes) are NOT considered binary
- [ ] Files that cannot be read return an error (not silently skipped)
- [ ] `internal/discovery/binary.go` also defines an `IsLargeFile(path string, maxBytes int64) (bool, int64, error)` function
- [ ] `IsLargeFile` uses `os.Stat()` to check file size without reading content
- [ ] The default max size is 1MB (1,048,576 bytes), configurable via `--skip-large-files` flag
- [ ] Both functions are designed for concurrent use (no shared mutable state)
- [ ] Performance: binary detection reads only up to 8KB, not the entire file
- [ ] Edge cases handled: permission denied, file deleted between stat and read, symlink targets
- [ ] Unit tests cover all detection scenarios with real test fixture files

## Technical Notes

- Binary detection approach (same as Git):
  ```go
  func IsBinary(path string) (bool, error) {
      f, err := os.Open(path)
      if err != nil {
          return false, err
      }
      defer f.Close()

      buf := make([]byte, 8192)
      n, err := f.Read(buf)
      if err != nil && err != io.EOF {
          return false, err
      }
      if n == 0 {
          return false, nil // empty file is not binary
      }

      for _, b := range buf[:n] {
          if b == 0 {
              return true, nil
          }
      }
      return false, nil
  }
  ```
- Per PRD Section 5.1: "Binary detection: check first 8KB of each file for null bytes (same approach as Git)."
- Per PRD Section 5.1: "Supports `--skip-large-files <size>` (default: 1MB) to skip oversized generated files."
- The 8KB threshold is sufficient for detecting most binary formats (images, executables, compressed files all contain null bytes in their headers).
- For `IsLargeFile`, use `os.Stat` to avoid reading the file content. This is important for performance when walking large repos.
- Create test fixture files in `testdata/`:
  - A small text file (should pass)
  - A binary file (PNG or similar, should be detected)
  - A file exactly at the size threshold
  - A large file exceeding the threshold
  - An empty file
- Reference: PRD Sections 5.1, 5.9

## Files to Create/Modify

- `internal/discovery/binary.go` - Binary detection and large file check
- `internal/discovery/binary_test.go` - Unit tests
- `testdata/binary-detection/text.txt` - Sample text file
- `testdata/binary-detection/binary.bin` - Sample binary file (small, with null bytes)
- `testdata/binary-detection/empty.txt` - Empty file

## Testing Requirements

- Unit test: text file is not detected as binary
- Unit test: file with null byte in first 8KB is detected as binary
- Unit test: file with null byte after 8KB is NOT detected as binary (only first 8KB checked)
- Unit test: empty file is not binary
- Unit test: PNG/JPEG header bytes are detected as binary
- Unit test: UTF-8 text with multibyte characters is not binary
- Unit test: file with only whitespace is not binary
- Unit test: `IsLargeFile` returns true for files exceeding threshold
- Unit test: `IsLargeFile` returns false for files under threshold
- Unit test: `IsLargeFile` with threshold of 0 returns true for all non-empty files
- Unit test: permission denied returns error, not false
- Benchmark: `IsBinary` on a 10MB file reads only 8KB (verify with read counter or file seek position)
