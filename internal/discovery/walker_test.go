package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestRepo sets up a synthetic test repository in a temp directory.
// Returns the root path. The caller should defer os.RemoveAll if not using
// t.TempDir().
func createTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Create directory structure.
	dirs := []string{
		"src",
		"docs",
		"build",
		".git/objects", // .git should always be skipped
	}
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(root, d), 0o755))
	}

	// Create text files.
	textFiles := map[string]string{
		"main.go":       "package main\n\nfunc main() {}\n",
		"README.md":     "# Test\n",
		"src/app.go":    "package src\n\nfunc App() {}\n",
		"src/util.go":   "package src\n\nfunc Util() {}\n",
		"docs/guide.md": "# Guide\n",
		".git/HEAD":     "ref: refs/heads/main\n",
	}
	for name, content := range textFiles {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(content), 0o644))
	}

	return root
}

// createBinaryFile writes a file with null bytes to simulate binary content.
func createBinaryFile(t *testing.T, path string) {
	t.Helper()
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00}
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

// createLargeFile writes a file of the given size.
func createLargeFile(t *testing.T, path string, size int64) {
	t.Helper()
	data := make([]byte, size)
	for i := range data {
		data[i] = 'x'
	}
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func TestWalkerBasicDiscovery(t *testing.T) {
	root := createTestRepo(t)

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root: root,
	})
	require.NoError(t, err)

	// Should find: main.go, README.md, src/app.go, src/util.go, docs/guide.md
	// Should NOT find: .git/HEAD
	assert.Len(t, result.Files, 5)

	paths := make([]string, len(result.Files))
	for i, f := range result.Files {
		paths[i] = f.Path
	}

	assert.Contains(t, paths, "main.go")
	assert.Contains(t, paths, "README.md")
	assert.Contains(t, paths, "src/app.go")
	assert.Contains(t, paths, "src/util.go")
	assert.Contains(t, paths, "docs/guide.md")
}

func TestWalkerSortedByPath(t *testing.T) {
	root := createTestRepo(t)

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root: root,
	})
	require.NoError(t, err)

	paths := make([]string, len(result.Files))
	for i, f := range result.Files {
		paths[i] = f.Path
	}

	assert.True(t, sort.SliceIsSorted(paths, func(i, j int) bool {
		return paths[i] < paths[j]
	}), "files should be sorted alphabetically by path")
}

func TestWalkerFileContentLoaded(t *testing.T) {
	root := createTestRepo(t)

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root: root,
	})
	require.NoError(t, err)

	for _, f := range result.Files {
		assert.NotEmpty(t, f.Content, "file %s should have content loaded", f.Path)
		assert.Nil(t, f.Error, "file %s should not have an error", f.Path)
	}

	// Verify specific content.
	for _, f := range result.Files {
		if f.Path == "main.go" {
			assert.Contains(t, f.Content, "package main")
			break
		}
	}
}

func TestWalkerGitDirSkipped(t *testing.T) {
	root := createTestRepo(t)

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root: root,
	})
	require.NoError(t, err)

	for _, f := range result.Files {
		assert.False(t, f.Path == ".git/HEAD" || f.Path == ".git/objects",
			"should not include .git files, got: %s", f.Path)
	}
}

func TestWalkerGitignoreRespected(t *testing.T) {
	root := createTestRepo(t)

	// Create .gitignore that ignores build/ directory.
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("build/\n"), 0o644))

	// Create a file in build/ that should be ignored.
	require.NoError(t, os.WriteFile(filepath.Join(root, "build", "output.js"), []byte("var x=1;\n"), 0o644))

	gitMatcher, err := NewGitignoreMatcher(root)
	require.NoError(t, err)

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:             root,
		GitignoreMatcher: gitMatcher,
	})
	require.NoError(t, err)

	for _, f := range result.Files {
		assert.NotEqual(t, "build/output.js", f.Path, "build/ files should be ignored by .gitignore")
	}
}

func TestWalkerHarvxignoreRespected(t *testing.T) {
	root := createTestRepo(t)

	// Create .harvxignore that ignores docs/.
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harvxignore"), []byte("docs/\n"), 0o644))

	harvxMatcher, err := NewHarvxignoreMatcher(root)
	require.NoError(t, err)

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:               root,
		HarvxignoreMatcher: harvxMatcher,
	})
	require.NoError(t, err)

	for _, f := range result.Files {
		assert.NotEqual(t, "docs/guide.md", f.Path, "docs/ files should be ignored by .harvxignore")
	}
}

func TestWalkerDefaultIgnorerApplied(t *testing.T) {
	root := createTestRepo(t)

	// Create a node_modules directory (should be caught by default ignorer).
	require.NoError(t, os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "node_modules", "pkg", "index.js"), []byte("module.exports = {}\n"), 0o644))

	defaultIgnorer := NewDefaultIgnoreMatcher()

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:           root,
		DefaultIgnorer: defaultIgnorer,
	})
	require.NoError(t, err)

	for _, f := range result.Files {
		assert.NotContains(t, f.Path, "node_modules",
			"node_modules should be ignored by default ignorer")
	}
}

func TestWalkerBinaryFilesSkipped(t *testing.T) {
	root := createTestRepo(t)

	// Create a binary file.
	createBinaryFile(t, filepath.Join(root, "image.png"))

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root: root,
	})
	require.NoError(t, err)

	for _, f := range result.Files {
		assert.NotEqual(t, "image.png", f.Path, "binary files should be skipped")
	}
	assert.Equal(t, 1, result.SkipReasons["binary"], "should record binary skip reason")
}

func TestWalkerLargeFilesSkipped(t *testing.T) {
	root := createTestRepo(t)

	// Create a large file (> 100 bytes threshold for testing).
	createLargeFile(t, filepath.Join(root, "big.txt"), 200)

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:           root,
		SkipLargeFiles: 100,
	})
	require.NoError(t, err)

	for _, f := range result.Files {
		assert.NotEqual(t, "big.txt", f.Path, "large files should be skipped")
	}
	assert.Equal(t, 1, result.SkipReasons["large_file"], "should record large_file skip reason")
}

func TestWalkerExtensionFilter(t *testing.T) {
	root := createTestRepo(t)

	filter := NewPatternFilter(PatternFilterOptions{
		Extensions: []string{"go"},
	})

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:          root,
		PatternFilter: filter,
	})
	require.NoError(t, err)

	for _, f := range result.Files {
		assert.Equal(t, ".go", filepath.Ext(f.Path), "only .go files should pass extension filter, got: %s", f.Path)
	}
	assert.True(t, len(result.Files) > 0, "should find at least one .go file")
}

func TestWalkerIncludePattern(t *testing.T) {
	root := createTestRepo(t)

	filter := NewPatternFilter(PatternFilterOptions{
		Includes: []string{"src/**"},
	})

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:          root,
		PatternFilter: filter,
	})
	require.NoError(t, err)

	for _, f := range result.Files {
		assert.True(t, len(f.Path) > 4 && f.Path[:4] == "src/",
			"only src/ files should pass include pattern, got: %s", f.Path)
	}
}

func TestWalkerExcludePattern(t *testing.T) {
	root := createTestRepo(t)

	filter := NewPatternFilter(PatternFilterOptions{
		Excludes: []string{"docs/**"},
	})

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:          root,
		PatternFilter: filter,
	})
	require.NoError(t, err)

	for _, f := range result.Files {
		assert.False(t, len(f.Path) > 5 && f.Path[:5] == "docs/",
			"docs/ files should be excluded, got: %s", f.Path)
	}
}

func TestWalkerEmptyDirectory(t *testing.T) {
	root := t.TempDir()

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root: root,
	})
	require.NoError(t, err)

	assert.Empty(t, result.Files, "empty directory should return empty file list")
	assert.Equal(t, 0, result.TotalFound)
	assert.Equal(t, 0, result.TotalSkipped)
}

func TestWalkerNonExistentDirectory(t *testing.T) {
	w := NewWalker()
	_, err := w.Walk(context.Background(), WalkerConfig{
		Root: "/nonexistent/path/that/does/not/exist",
	})
	assert.Error(t, err, "should return error for non-existent directory")
}

func TestWalkerContextCancellation(t *testing.T) {
	root := createTestRepo(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	w := NewWalker()
	_, err := w.Walk(ctx, WalkerConfig{
		Root: root,
	})

	assert.Error(t, err, "should return error when context is cancelled")
}

func TestWalkerContextTimeout(t *testing.T) {
	// Create a repo with many files to increase chance of timeout hitting during walk.
	root := t.TempDir()
	for i := 0; i < 100; i++ {
		require.NoError(t, os.WriteFile(
			filepath.Join(root, fmt.Sprintf("file_%03d.txt", i)),
			[]byte(fmt.Sprintf("content %d\n", i)),
			0o644,
		))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give it a moment for the timeout to fire.
	time.Sleep(1 * time.Millisecond)

	w := NewWalker()
	_, err := w.Walk(ctx, WalkerConfig{
		Root: root,
	})

	assert.Error(t, err, "should return error when context times out")
}

func TestWalkerPerFileReadErrors(t *testing.T) {
	root := t.TempDir()

	// Create a readable file.
	require.NoError(t, os.WriteFile(filepath.Join(root, "good.txt"), []byte("good content\n"), 0o644))

	// Create a file and then make it unreadable.
	badPath := filepath.Join(root, "bad.txt")
	require.NoError(t, os.WriteFile(badPath, []byte("bad content\n"), 0o644))
	require.NoError(t, os.Chmod(badPath, 0o000))

	// Ensure we restore permissions for cleanup.
	t.Cleanup(func() {
		os.Chmod(badPath, 0o644) //nolint:errcheck
	})

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root: root,
	})
	require.NoError(t, err, "walk should succeed even with per-file errors")

	assert.Len(t, result.Files, 2, "should include both files")

	var goodFile, badFile bool
	for _, f := range result.Files {
		if f.Path == "good.txt" {
			goodFile = true
			assert.NotEmpty(t, f.Content, "good file should have content")
			assert.Nil(t, f.Error, "good file should not have error")
		}
		if f.Path == "bad.txt" {
			badFile = true
			assert.Error(t, f.Error, "bad file should have error")
			assert.Empty(t, f.Content, "bad file should not have content")
		}
	}
	assert.True(t, goodFile, "should find good.txt")
	assert.True(t, badFile, "should find bad.txt")
}

func TestWalkerDiscoveryResult(t *testing.T) {
	root := createTestRepo(t)

	// Add a binary file.
	createBinaryFile(t, filepath.Join(root, "image.png"))

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root: root,
	})
	require.NoError(t, err)

	assert.Greater(t, result.TotalFound, 0, "TotalFound should be > 0")
	assert.Greater(t, result.TotalSkipped, 0, "TotalSkipped should be > 0 (binary file)")
	assert.NotNil(t, result.SkipReasons, "SkipReasons should not be nil")
	assert.Equal(t, 5, len(result.Files), "should find 5 text files")
}

func TestWalkerFileDescriptorFields(t *testing.T) {
	root := createTestRepo(t)

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root: root,
	})
	require.NoError(t, err)

	for _, f := range result.Files {
		assert.NotEmpty(t, f.Path, "Path should not be empty")
		assert.NotEmpty(t, f.AbsPath, "AbsPath should not be empty")
		assert.True(t, filepath.IsAbs(f.AbsPath), "AbsPath should be absolute: %s", f.AbsPath)
		assert.Greater(t, f.Size, int64(0), "Size should be > 0 for %s", f.Path)
		assert.Equal(t, 2, f.Tier, "Tier should be DefaultTier (2) for %s", f.Path)
	}
}

func TestWalkerConcurrencyDefault(t *testing.T) {
	root := createTestRepo(t)

	// Test with default concurrency (0 should resolve to runtime.NumCPU()).
	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:        root,
		Concurrency: 0,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Files, "should find files with default concurrency")
}

func TestWalkerConcurrencyOne(t *testing.T) {
	root := createTestRepo(t)

	// Single-worker mode should still work.
	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:        root,
		Concurrency: 1,
	})
	require.NoError(t, err)
	assert.Len(t, result.Files, 5, "should find all files with single worker")
}

func TestWalkerSampleRepo(t *testing.T) {
	// Integration test using the testdata/sample-repo fixture.
	sampleRepo := filepath.Join("testdata", "sample-repo")

	// Find testdata relative to the project root.
	// Walk up from the test directory to find testdata.
	projectRoot := findProjectRoot(t)
	sampleRepo = filepath.Join(projectRoot, "testdata", "sample-repo")

	if _, err := os.Stat(sampleRepo); os.IsNotExist(err) {
		t.Skip("testdata/sample-repo not found, skipping integration test")
	}

	gitMatcher, err := NewGitignoreMatcher(sampleRepo)
	require.NoError(t, err)

	harvxMatcher, err := NewHarvxignoreMatcher(sampleRepo)
	require.NoError(t, err)

	defaultIgnorer := NewDefaultIgnoreMatcher()

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:               sampleRepo,
		GitignoreMatcher:   gitMatcher,
		HarvxignoreMatcher: harvxMatcher,
		DefaultIgnorer:     defaultIgnorer,
	})
	require.NoError(t, err)

	paths := make([]string, len(result.Files))
	for i, f := range result.Files {
		paths[i] = f.Path
	}

	// Should include these text files.
	assert.Contains(t, paths, "main.go")
	assert.Contains(t, paths, "README.md")
	assert.Contains(t, paths, "src/app.ts")
	assert.Contains(t, paths, "src/utils.ts")
	assert.Contains(t, paths, "src/test.spec.ts")
	// .gitignore and .harvxignore themselves are text files and should be included.
	assert.Contains(t, paths, ".gitignore")
	assert.Contains(t, paths, ".harvxignore")

	// Should NOT include ignored files.
	assert.NotContains(t, paths, "dist/bundle.js", "dist/ should be ignored by .gitignore")
	assert.NotContains(t, paths, "node_modules/pkg/index.js", "node_modules/ should be ignored")
	assert.NotContains(t, paths, "docs/internal/notes.md", "docs/internal/ should be ignored by .harvxignore")
}

func TestWalkerMultipleIgnoreSources(t *testing.T) {
	root := createTestRepo(t)

	// Create files that should be ignored by different sources.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "vendor"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "vendor", "lib.go"), []byte("package vendor\n"), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("build/\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "build", "out.js"), []byte("var x;\n"), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(root, ".harvxignore"), []byte("vendor/\n"), 0o644))

	gitMatcher, err := NewGitignoreMatcher(root)
	require.NoError(t, err)

	harvxMatcher, err := NewHarvxignoreMatcher(root)
	require.NoError(t, err)

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:               root,
		GitignoreMatcher:   gitMatcher,
		HarvxignoreMatcher: harvxMatcher,
	})
	require.NoError(t, err)

	paths := make([]string, len(result.Files))
	for i, f := range result.Files {
		paths[i] = f.Path
	}

	assert.NotContains(t, paths, "build/out.js", "build/ should be ignored by .gitignore")
	assert.NotContains(t, paths, "vendor/lib.go", "vendor/ should be ignored by .harvxignore")
}

func TestWalkerSkipLargeFilesZeroDisabled(t *testing.T) {
	root := t.TempDir()

	// Create a "large" file.
	createLargeFile(t, filepath.Join(root, "big.txt"), 10000)

	w := NewWalker()
	result, err := w.Walk(context.Background(), WalkerConfig{
		Root:           root,
		SkipLargeFiles: 0, // disabled
	})
	require.NoError(t, err)

	assert.Len(t, result.Files, 1, "large file should be included when SkipLargeFiles=0")
}

// BenchmarkWalker1000Files benchmarks walking a directory with 1000 files.
func BenchmarkWalker1000Files(b *testing.B) {
	root := b.TempDir()

	// Create 1000 files.
	for i := 0; i < 1000; i++ {
		err := os.WriteFile(
			filepath.Join(root, fmt.Sprintf("file_%04d.go", i)),
			[]byte(fmt.Sprintf("package main\n\nfunc f%d() {}\n", i)),
			0o644,
		)
		if err != nil {
			b.Fatal(err)
		}
	}

	w := NewWalker()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := w.Walk(ctx, WalkerConfig{Root: root})
		if err != nil {
			b.Fatal(err)
		}
		if len(result.Files) != 1000 {
			b.Fatalf("expected 1000 files, got %d", len(result.Files))
		}
	}
}

// BenchmarkWalkerWithFilters benchmarks walking with all filter types enabled.
func BenchmarkWalkerWithFilters(b *testing.B) {
	root := b.TempDir()

	// Create files in multiple directories.
	dirs := []string{"src", "test", "docs", "vendor", "build"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			b.Fatal(err)
		}
	}

	for i := 0; i < 200; i++ {
		for _, d := range dirs {
			err := os.WriteFile(
				filepath.Join(root, d, fmt.Sprintf("file_%04d.go", i)),
				[]byte(fmt.Sprintf("package %s\n\nfunc f%d() {}\n", d, i)),
				0o644,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	}

	// Add .gitignore.
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("vendor/\nbuild/\n"), 0o644); err != nil {
		b.Fatal(err)
	}

	gitMatcher, err := NewGitignoreMatcher(root)
	if err != nil {
		b.Fatal(err)
	}

	defaultIgnorer := NewDefaultIgnoreMatcher()

	filter := NewPatternFilter(PatternFilterOptions{
		Extensions: []string{"go"},
	})

	w := NewWalker()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := w.Walk(ctx, WalkerConfig{
			Root:             root,
			GitignoreMatcher: gitMatcher,
			DefaultIgnorer:   defaultIgnorer,
			PatternFilter:    filter,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
