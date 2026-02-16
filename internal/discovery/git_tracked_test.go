package discovery

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gitInit initialises a new git repository in dir with a minimal config so
// that commits can be created without a global user.name / user.email.
func gitInit(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
}

// runGit executes a git command in the given directory and fails the test on
// error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, string(out))
}

// gitAddCommit stages all files and creates a commit in the given directory.
func gitAddCommit(t *testing.T, dir, msg string) {
	t.Helper()
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", msg, "--allow-empty")
}

// projectRoot returns the root of the harvx repository by walking up from the
// test file location until a .git directory is found.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")

	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "could not find .git directory above test file")
		dir = parent
	}
}

func TestGitTrackedFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns tracked files from a real repository", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		gitInit(t, dir)

		// Create some files and track them.
		createTestFile(t, dir, "main.go", []byte("package main"))
		createTestFile(t, dir, "README.md", []byte("# Test"))
		createTestFile(t, dir, "src/util.go", []byte("package src"))
		gitAddCommit(t, dir, "initial commit")

		files, err := GitTrackedFiles(dir)
		require.NoError(t, err)

		assert.True(t, files["main.go"], "main.go should be tracked")
		assert.True(t, files["README.md"], "README.md should be tracked")
		assert.True(t, files["src/util.go"], "src/util.go should be tracked")
		assert.Len(t, files, 3, "should have exactly 3 tracked files")
	})

	t.Run("non-git directory returns descriptive error", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		files, err := GitTrackedFiles(dir)
		assert.Nil(t, files, "files should be nil on error")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git ls-files failed")
		assert.Contains(t, err.Error(), "is this a git repository?")
	})

	t.Run("empty repo returns empty set", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		gitInit(t, dir)

		files, err := GitTrackedFiles(dir)
		require.NoError(t, err)
		assert.NotNil(t, files, "files map should not be nil")
		assert.Empty(t, files, "empty repo should have no tracked files")
	})

	t.Run("file paths are relative to root", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		gitInit(t, dir)

		// Create nested files.
		createTestFile(t, dir, "a/b/c/deep.txt", []byte("deep"))
		createTestFile(t, dir, "top.txt", []byte("top"))
		gitAddCommit(t, dir, "nested files")

		files, err := GitTrackedFiles(dir)
		require.NoError(t, err)

		// Verify all paths are relative (no absolute path prefixes).
		for path := range files {
			assert.False(t, filepath.IsAbs(path),
				"path %q should be relative, not absolute", path)
			assert.NotContains(t, path, dir,
				"path %q should not contain root directory %q", path, dir)
		}

		assert.True(t, files["a/b/c/deep.txt"], "deeply nested file should use relative path")
		assert.True(t, files["top.txt"], "top-level file should use relative path")
	})

	t.Run("untracked files are excluded", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		gitInit(t, dir)

		// Create and commit one file.
		createTestFile(t, dir, "tracked.go", []byte("package main"))
		gitAddCommit(t, dir, "add tracked file")

		// Create another file but do NOT stage/commit it.
		createTestFile(t, dir, "untracked.go", []byte("package main"))

		files, err := GitTrackedFiles(dir)
		require.NoError(t, err)

		assert.True(t, files["tracked.go"], "committed file should be tracked")
		assert.False(t, files["untracked.go"], "uncommitted file should not be tracked")
		assert.Len(t, files, 1)
	})

	t.Run("staged but uncommitted files are included", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		gitInit(t, dir)

		// Create an initial commit so the repo is not empty.
		createTestFile(t, dir, "initial.txt", []byte("init"))
		gitAddCommit(t, dir, "initial")

		// Stage a file without committing.
		createTestFile(t, dir, "staged.go", []byte("package main"))
		runGit(t, dir, "add", "staged.go")

		files, err := GitTrackedFiles(dir)
		require.NoError(t, err)

		assert.True(t, files["staged.go"],
			"staged (but uncommitted) file should appear in git ls-files")
		assert.True(t, files["initial.txt"])
	})

	t.Run("deleted files are not returned", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		gitInit(t, dir)

		createTestFile(t, dir, "keep.go", []byte("keep"))
		createTestFile(t, dir, "remove.go", []byte("remove"))
		gitAddCommit(t, dir, "two files")

		// Remove the file and stage the deletion.
		runGit(t, dir, "rm", "remove.go")

		files, err := GitTrackedFiles(dir)
		require.NoError(t, err)

		assert.True(t, files["keep.go"], "kept file should be tracked")
		assert.False(t, files["remove.go"], "git-rm'd file should not be tracked")
	})

	t.Run("nonexistent directory returns error", func(t *testing.T) {
		t.Parallel()

		files, err := GitTrackedFiles("/nonexistent/path/does/not/exist")
		assert.Nil(t, files)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git ls-files failed")
	})

	t.Run("works against project root", func(t *testing.T) {
		t.Parallel()

		root := projectRoot(t)
		files, err := GitTrackedFiles(root)
		require.NoError(t, err)

		// The project itself should have tracked files.
		assert.NotEmpty(t, files, "project root should have tracked files")

		// Verify some known files exist.
		assert.True(t, files["go.mod"], "go.mod should be tracked in project")
		assert.True(t, files["CLAUDE.md"], "CLAUDE.md should be tracked in project")

		// All paths should be relative.
		for path := range files {
			assert.False(t, filepath.IsAbs(path),
				"path %q in project should be relative", path)
		}
	})

	t.Run("handles files with spaces in names", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		gitInit(t, dir)

		createTestFile(t, dir, "file with spaces.txt", []byte("content"))
		createTestFile(t, dir, "dir with space/nested file.go", []byte("package main"))
		gitAddCommit(t, dir, "files with spaces")

		files, err := GitTrackedFiles(dir)
		require.NoError(t, err)

		assert.True(t, files["file with spaces.txt"],
			"file with spaces should be tracked")
		assert.True(t, files["dir with space/nested file.go"],
			"nested file with spaces in path should be tracked")
	})

	t.Run("many files are all returned", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		gitInit(t, dir)

		const fileCount = 100
		expectedPaths := make([]string, 0, fileCount)
		for i := 0; i < fileCount; i++ {
			relName := fmt.Sprintf("pkg/file_%03d.go", i)
			createTestFile(t, dir, relName, []byte(fmt.Sprintf("package pkg // file %d", i)))
			expectedPaths = append(expectedPaths, relName)
		}
		gitAddCommit(t, dir, "bulk files")

		files, err := GitTrackedFiles(dir)
		require.NoError(t, err)
		assert.Len(t, files, fileCount, "all %d files should be tracked", fileCount)

		// Verify every expected path is present.
		for _, p := range expectedPaths {
			assert.True(t, files[p], "file %q should be tracked", p)
		}
	})

	t.Run("empty lines in output are skipped", func(t *testing.T) {
		t.Parallel()

		// This test verifies the line != "" guard in the implementation.
		// git ls-files may produce a trailing newline which creates an empty
		// final line from the scanner. We verify the map has no empty-string key.
		dir := t.TempDir()
		gitInit(t, dir)

		createTestFile(t, dir, "only.txt", []byte("only"))
		gitAddCommit(t, dir, "single file")

		files, err := GitTrackedFiles(dir)
		require.NoError(t, err)

		assert.False(t, files[""], "empty string should not be a key in the map")
		assert.Len(t, files, 1)
	})

	t.Run("forward slashes on all platforms", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		gitInit(t, dir)

		createTestFile(t, dir, "a/b/c.txt", []byte("nested"))
		gitAddCommit(t, dir, "nested")

		files, err := GitTrackedFiles(dir)
		require.NoError(t, err)

		// git ls-files always uses forward slashes regardless of OS.
		var paths []string
		for p := range files {
			paths = append(paths, p)
		}
		sort.Strings(paths)

		assert.Equal(t, "a/b/c.txt", paths[0],
			"git ls-files should use forward slashes")
	})
}
