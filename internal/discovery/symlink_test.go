package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// skipOnWindows skips the current test on Windows where symlink creation
// requires elevated privileges.
func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests require elevated privileges on Windows")
	}
}

// createSymlink creates a symbolic link at linkPath pointing to target.
func createSymlink(t *testing.T, target, linkPath string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(linkPath), 0o755))
	require.NoError(t, os.Symlink(target, linkPath))
}

// ---------------------------------------------------------------------------
// IsSymlink tests
// ---------------------------------------------------------------------------

func TestIsSymlink(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	dir := t.TempDir()

	// Set up files and directories used across subtests.
	regularFile := createTestFile(t, dir, "regular.txt", []byte("hello"))
	subDir := filepath.Join(dir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	symlinkToFile := filepath.Join(dir, "link-to-file")
	createSymlink(t, regularFile, symlinkToFile)

	symlinkToDir := filepath.Join(dir, "link-to-dir")
	createSymlink(t, subDir, symlinkToDir)

	tests := []struct {
		name     string
		path     string
		wantLink bool
	}{
		{
			name:     "regular file is not a symlink",
			path:     regularFile,
			wantLink: false,
		},
		{
			name:     "directory is not a symlink",
			path:     subDir,
			wantLink: false,
		},
		{
			name:     "symlink to file is detected",
			path:     symlinkToFile,
			wantLink: true,
		},
		{
			name:     "symlink to directory is detected",
			path:     symlinkToDir,
			wantLink: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsSymlink(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.wantLink, got)
		})
	}
}

func TestIsSymlink_NonexistentPath(t *testing.T) {
	t.Parallel()

	_, err := IsSymlink("/nonexistent/path/does/not/exist")
	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestIsSymlink_DanglingSymlink(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	dir := t.TempDir()

	// Create a target, make a symlink to it, then remove the target.
	target := createTestFile(t, dir, "target.txt", []byte("data"))
	link := filepath.Join(dir, "dangling-link")
	createSymlink(t, target, link)
	require.NoError(t, os.Remove(target))

	// Lstat should still succeed on the dangling symlink (it reads link metadata).
	got, err := IsSymlink(link)
	require.NoError(t, err)
	assert.True(t, got, "dangling symlink should still be detected as a symlink by Lstat")
}

// ---------------------------------------------------------------------------
// SymlinkResolver -- NewSymlinkResolver
// ---------------------------------------------------------------------------

func TestNewSymlinkResolver(t *testing.T) {
	t.Parallel()

	sr := NewSymlinkResolver()
	require.NotNil(t, sr)
	assert.Equal(t, 0, sr.VisitedCount(), "new resolver should have zero visited paths")
}

// ---------------------------------------------------------------------------
// SymlinkResolver -- Resolve (regular files and symlinks)
// ---------------------------------------------------------------------------

func TestResolve_RegularFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := createTestFile(t, dir, "plain.txt", []byte("content"))

	sr := NewSymlinkResolver()
	realPath, isLoop, err := sr.Resolve(file)
	require.NoError(t, err)
	assert.False(t, isLoop, "regular file should not be a loop")

	// EvalSymlinks resolves to an absolute path; compare after cleaning.
	expected, err := filepath.EvalSymlinks(file)
	require.NoError(t, err)
	assert.Equal(t, expected, realPath)
}

func TestResolve_SymlinkToFile(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	dir := t.TempDir()
	target := createTestFile(t, dir, "target.txt", []byte("hello"))
	link := filepath.Join(dir, "link.txt")
	createSymlink(t, target, link)

	sr := NewSymlinkResolver()
	realPath, isLoop, err := sr.Resolve(link)
	require.NoError(t, err)
	assert.False(t, isLoop, "first visit should not be a loop")

	// The resolved path should point to the real target.
	expectedReal, err := filepath.EvalSymlinks(target)
	require.NoError(t, err)
	assert.Equal(t, expectedReal, realPath)
}

func TestResolve_SymlinkToDirectory(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	dir := t.TempDir()
	subDir := filepath.Join(dir, "real-dir")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	link := filepath.Join(dir, "link-dir")
	createSymlink(t, subDir, link)

	sr := NewSymlinkResolver()
	realPath, isLoop, err := sr.Resolve(link)
	require.NoError(t, err)
	assert.False(t, isLoop, "symlink to directory should not be a loop on first visit")

	expectedReal, err := filepath.EvalSymlinks(subDir)
	require.NoError(t, err)
	assert.Equal(t, expectedReal, realPath)
}

// ---------------------------------------------------------------------------
// SymlinkResolver -- Resolve (symlink chain)
// ---------------------------------------------------------------------------

func TestResolve_SymlinkChain(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	// Create chain: linkA -> linkB -> linkC -> realFile
	dir := t.TempDir()
	realFile := createTestFile(t, dir, "real.txt", []byte("chain end"))
	linkC := filepath.Join(dir, "linkC")
	linkB := filepath.Join(dir, "linkB")
	linkA := filepath.Join(dir, "linkA")

	createSymlink(t, realFile, linkC)
	createSymlink(t, linkC, linkB)
	createSymlink(t, linkB, linkA)

	sr := NewSymlinkResolver()
	realPath, isLoop, err := sr.Resolve(linkA)
	require.NoError(t, err)
	assert.False(t, isLoop, "chain should resolve without detecting a loop")

	// All links in the chain should resolve to the same real file.
	expectedReal, err := filepath.EvalSymlinks(realFile)
	require.NoError(t, err)
	assert.Equal(t, expectedReal, realPath, "chain should resolve to the real file")
}

// ---------------------------------------------------------------------------
// SymlinkResolver -- Resolve (loop detection)
// ---------------------------------------------------------------------------

func TestResolve_LoopDetection(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	// Cannot create actual filesystem symlink loops. Instead, simulate loop
	// detection by marking a real path as visited, then resolving a symlink
	// that points to it.
	dir := t.TempDir()
	realFile := createTestFile(t, dir, "target.txt", []byte("data"))
	link := filepath.Join(dir, "link.txt")
	createSymlink(t, realFile, link)

	sr := NewSymlinkResolver()

	// First resolve: should succeed without loop.
	realPath, isLoop, err := sr.Resolve(link)
	require.NoError(t, err)
	assert.False(t, isLoop, "first resolve should not detect a loop")

	// Mark the resolved path as visited (simulates walker processing it).
	sr.MarkVisited(realPath)

	// Second resolve of the same symlink: should detect a loop.
	realPath2, isLoop2, err2 := sr.Resolve(link)
	require.NoError(t, err2, "loop detection should not produce an error")
	assert.True(t, isLoop2, "second resolve should detect a loop")
	assert.Equal(t, realPath, realPath2, "resolved path should be the same")
}

func TestResolve_LoopDetection_MultipleLinks(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	// Two different symlinks pointing to the same real file. After marking
	// the real path as visited via one link, the other link should detect a loop.
	dir := t.TempDir()
	realFile := createTestFile(t, dir, "shared-target.txt", []byte("shared"))
	linkA := filepath.Join(dir, "linkA")
	linkB := filepath.Join(dir, "linkB")
	createSymlink(t, realFile, linkA)
	createSymlink(t, realFile, linkB)

	sr := NewSymlinkResolver()

	// Resolve via linkA and mark visited.
	realPath, isLoop, err := sr.Resolve(linkA)
	require.NoError(t, err)
	assert.False(t, isLoop)
	sr.MarkVisited(realPath)

	// Resolve via linkB -- should detect loop because the real path is the same.
	_, isLoop2, err2 := sr.Resolve(linkB)
	require.NoError(t, err2)
	assert.True(t, isLoop2, "different symlink to same target should detect loop")
}

// ---------------------------------------------------------------------------
// SymlinkResolver -- Resolve (dangling symlink)
// ---------------------------------------------------------------------------

func TestResolve_DanglingSymlink(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	dir := t.TempDir()
	target := createTestFile(t, dir, "will-be-deleted.txt", []byte("temp"))
	link := filepath.Join(dir, "dangling-link")
	createSymlink(t, target, link)

	// Remove the target so the symlink becomes dangling.
	require.NoError(t, os.Remove(target))

	sr := NewSymlinkResolver()
	realPath, isLoop, err := sr.Resolve(link)
	require.Error(t, err, "dangling symlink should return an error")
	assert.Contains(t, err.Error(), "dangling symlink")
	assert.Empty(t, realPath, "realPath should be empty on error")
	assert.False(t, isLoop, "isLoop should be false on error")
}

func TestResolve_NonexistentPath(t *testing.T) {
	t.Parallel()

	sr := NewSymlinkResolver()
	realPath, isLoop, err := sr.Resolve("/nonexistent/path/nowhere")
	require.Error(t, err, "nonexistent path should return an error")
	assert.Empty(t, realPath)
	assert.False(t, isLoop)
}

// ---------------------------------------------------------------------------
// SymlinkResolver -- MarkVisited
// ---------------------------------------------------------------------------

func TestMarkVisited(t *testing.T) {
	t.Parallel()

	sr := NewSymlinkResolver()
	assert.Equal(t, 0, sr.VisitedCount())

	sr.MarkVisited("/a")
	assert.Equal(t, 1, sr.VisitedCount())

	sr.MarkVisited("/b")
	assert.Equal(t, 2, sr.VisitedCount())

	// Marking the same path again should not increase count (map semantics).
	sr.MarkVisited("/a")
	assert.Equal(t, 2, sr.VisitedCount(), "duplicate MarkVisited should not increase count")
}

// ---------------------------------------------------------------------------
// SymlinkResolver -- VisitedCount
// ---------------------------------------------------------------------------

func TestVisitedCount(t *testing.T) {
	t.Parallel()

	sr := NewSymlinkResolver()
	assert.Equal(t, 0, sr.VisitedCount(), "fresh resolver has zero visited")

	for i := 0; i < 10; i++ {
		sr.MarkVisited(fmt.Sprintf("/path/%d", i))
	}
	assert.Equal(t, 10, sr.VisitedCount())
}

// ---------------------------------------------------------------------------
// SymlinkResolver -- Reset
// ---------------------------------------------------------------------------

func TestReset(t *testing.T) {
	t.Parallel()

	sr := NewSymlinkResolver()
	sr.MarkVisited("/a")
	sr.MarkVisited("/b")
	sr.MarkVisited("/c")
	assert.Equal(t, 3, sr.VisitedCount())

	sr.Reset()
	assert.Equal(t, 0, sr.VisitedCount(), "Reset should clear all visited paths")

	// After reset, previously visited paths should no longer be detected as loops.
	dir := t.TempDir()
	file := createTestFile(t, dir, "after-reset.txt", []byte("content"))
	absPath, err := filepath.EvalSymlinks(file)
	require.NoError(t, err)

	sr.MarkVisited(absPath)
	_, isLoop, err := sr.Resolve(file)
	require.NoError(t, err)
	assert.True(t, isLoop, "path marked after reset should be detected as visited")
}

func TestReset_ThenReuse(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	dir := t.TempDir()
	realFile := createTestFile(t, dir, "file.txt", []byte("data"))
	link := filepath.Join(dir, "link")
	createSymlink(t, realFile, link)

	sr := NewSymlinkResolver()

	// First pass: resolve and mark.
	realPath, isLoop, err := sr.Resolve(link)
	require.NoError(t, err)
	assert.False(t, isLoop)
	sr.MarkVisited(realPath)

	// Confirm loop is detected.
	_, isLoop2, _ := sr.Resolve(link)
	assert.True(t, isLoop2)

	// Reset and resolve again: no loop.
	sr.Reset()
	_, isLoop3, err3 := sr.Resolve(link)
	require.NoError(t, err3)
	assert.False(t, isLoop3, "after Reset, previously visited paths should not be loops")
}

// ---------------------------------------------------------------------------
// SymlinkResolver -- Concurrent access
// ---------------------------------------------------------------------------

func TestSymlinkResolver_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	dir := t.TempDir()

	// Create a set of real files and symlinks.
	const numFiles = 50
	realPaths := make([]string, numFiles)
	linkPaths := make([]string, numFiles)

	for i := 0; i < numFiles; i++ {
		realPaths[i] = createTestFile(t, dir, fmt.Sprintf("file-%d.txt", i), []byte(fmt.Sprintf("content %d", i)))
		linkPaths[i] = filepath.Join(dir, fmt.Sprintf("link-%d", i))
		createSymlink(t, realPaths[i], linkPaths[i])
	}

	sr := NewSymlinkResolver()

	var wg sync.WaitGroup

	// Concurrently resolve symlinks.
	for i := 0; i < numFiles; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			realPath, _, err := sr.Resolve(linkPaths[idx])
			if err != nil {
				t.Errorf("Resolve(%d) failed: %v", idx, err)
				return
			}
			sr.MarkVisited(realPath)
		}(i)
	}

	// Concurrently read visited count.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = sr.VisitedCount()
		}()
	}

	wg.Wait()

	assert.Equal(t, numFiles, sr.VisitedCount(),
		"all files should be marked as visited after concurrent access")
}

func TestSymlinkResolver_ConcurrentResolveAndReset(t *testing.T) {
	t.Parallel()
	skipOnWindows(t)

	dir := t.TempDir()
	realFile := createTestFile(t, dir, "target.txt", []byte("data"))
	link := filepath.Join(dir, "link")
	createSymlink(t, realFile, link)

	sr := NewSymlinkResolver()

	// Run concurrent Resolve, MarkVisited, Reset, and VisitedCount operations.
	// The goal is to verify no data race occurs (detected by -race flag).
	var wg sync.WaitGroup
	const iterations = 100

	for i := 0; i < iterations; i++ {
		idx := i // capture for goroutine
		wg.Add(4)

		go func() {
			defer wg.Done()
			realPath, _, err := sr.Resolve(link)
			if err == nil {
				sr.MarkVisited(realPath)
			}
		}()

		go func() {
			defer wg.Done()
			_ = sr.VisitedCount()
		}()

		go func() {
			defer wg.Done()
			sr.MarkVisited("/some/path")
		}()

		go func() {
			defer wg.Done()
			if idx%10 == 0 {
				sr.Reset()
			}
		}()
	}

	wg.Wait()
	// No assertions on final state -- the purpose is race detection.
}

// ---------------------------------------------------------------------------
// SymlinkResolver -- Resolve does NOT auto-mark visited
// ---------------------------------------------------------------------------

func TestResolve_DoesNotAutoMarkVisited(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := createTestFile(t, dir, "file.txt", []byte("data"))

	sr := NewSymlinkResolver()

	// Resolve should not mark the path as visited.
	_, _, err := sr.Resolve(file)
	require.NoError(t, err)
	assert.Equal(t, 0, sr.VisitedCount(),
		"Resolve should not auto-mark the path as visited")

	// Resolve again: still no loop since nothing was marked.
	_, isLoop, err := sr.Resolve(file)
	require.NoError(t, err)
	assert.False(t, isLoop, "second Resolve without MarkVisited should not be a loop")
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkResolve_RegularFile(b *testing.B) {
	dir := b.TempDir()
	file := filepath.Join(dir, "bench.txt")
	if err := os.WriteFile(file, []byte("bench"), 0o644); err != nil {
		b.Fatal(err)
	}

	sr := NewSymlinkResolver()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := sr.Resolve(file)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolve_Symlink(b *testing.B) {
	if runtime.GOOS == "windows" {
		b.Skip("symlink benchmarks require elevated privileges on Windows")
	}

	dir := b.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("bench"), 0o644); err != nil {
		b.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		b.Fatal(err)
	}

	sr := NewSymlinkResolver()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := sr.Resolve(link)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIsSymlink(b *testing.B) {
	if runtime.GOOS == "windows" {
		b.Skip("symlink benchmarks require elevated privileges on Windows")
	}

	dir := b.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("bench"), 0o644); err != nil {
		b.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := IsSymlink(link)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarkVisited(b *testing.B) {
	sr := NewSymlinkResolver()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sr.MarkVisited(fmt.Sprintf("/path/%d", i))
	}
}
