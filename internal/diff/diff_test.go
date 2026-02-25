package diff

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- DiffMode tests ---

func TestDiffMode_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode DiffMode
		want string
	}{
		{name: "cache", mode: DiffModeCache, want: "cache"},
		{name: "since", mode: DiffModeSince, want: "since"},
		{name: "base-head", mode: DiffModeBaseHead, want: "base-head"},
		{name: "unknown", mode: DiffMode(99), want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.mode.String())
		})
	}
}

// --- DetermineDiffMode tests ---

func TestDetermineDiffMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		since    string
		base     string
		head     string
		wantMode DiffMode
		wantErr  string
	}{
		{
			name:     "no flags defaults to cache mode",
			wantMode: DiffModeCache,
		},
		{
			name:     "since flag selects since mode",
			since:    "HEAD~1",
			wantMode: DiffModeSince,
		},
		{
			name:     "base and head select base-head mode",
			base:     "main",
			head:     "feature",
			wantMode: DiffModeBaseHead,
		},
		{
			name:    "since with base is mutually exclusive",
			since:   "HEAD~1",
			base:    "main",
			wantErr: "--since and --base/--head are mutually exclusive",
		},
		{
			name:    "since with head is mutually exclusive",
			since:   "HEAD~1",
			head:    "feature",
			wantErr: "--since and --base/--head are mutually exclusive",
		},
		{
			name:    "since with base and head is mutually exclusive",
			since:   "HEAD~1",
			base:    "main",
			head:    "feature",
			wantErr: "--since and --base/--head are mutually exclusive",
		},
		{
			name:    "base without head is an error",
			base:    "main",
			wantErr: "--base requires --head",
		},
		{
			name:    "head without base is an error",
			head:    "feature",
			wantErr: "--head requires --base",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mode, err := DetermineDiffMode(tt.since, tt.base, tt.head)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMode, mode)
		})
	}
}

// --- FormatChangeSummary tests ---

func TestFormatChangeSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result *DiffResult
		checks []string // substrings that must be present
	}{
		{
			name:   "nil result",
			result: nil,
			checks: []string{"No changes detected."},
		},
		{
			name:   "no changes",
			result: &DiffResult{Unchanged: 5},
			checks: []string{"No changes detected."},
		},
		{
			name: "only added",
			result: &DiffResult{
				Added: []string{"new.go", "util.go"},
			},
			checks: []string{
				"Change Summary:",
				"2 added",
				"+ new.go",
				"+ util.go",
			},
		},
		{
			name: "only modified",
			result: &DiffResult{
				Modified: []string{"main.go"},
			},
			checks: []string{
				"Change Summary:",
				"1 modified",
				"~ main.go",
			},
		},
		{
			name: "only deleted",
			result: &DiffResult{
				Deleted: []string{"old.go"},
			},
			checks: []string{
				"Change Summary:",
				"1 deleted",
				"- old.go",
			},
		},
		{
			name: "mixed changes",
			result: &DiffResult{
				Added:     []string{"a.go"},
				Modified:  []string{"b.go"},
				Deleted:   []string{"c.go"},
				Unchanged: 10,
			},
			checks: []string{
				"Change Summary:",
				"1 added",
				"1 modified",
				"1 deleted",
				"10 unchanged",
				"+ a.go",
				"~ b.go",
				"- c.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := FormatChangeSummary(tt.result)
			for _, check := range tt.checks {
				assert.Contains(t, result, check, "expected substring %q in summary", check)
			}
		})
	}
}

// --- RunDiff tests ---

func TestRunDiff_EmptyRootDir(t *testing.T) {
	t.Parallel()

	_, err := RunDiff(context.Background(), DiffOptions{
		Mode: DiffModeCache,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "root directory required")
}

func TestRunDiff_UnknownMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := RunDiff(context.Background(), DiffOptions{
		Mode:    DiffMode(99),
		RootDir: dir,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown diff mode")
}

func TestRunDiff_CacheMode_NoState(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := RunDiff(context.Background(), DiffOptions{
		Mode:        DiffModeCache,
		RootDir:     dir,
		ProfileName: "test",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoState)
}

func TestRunDiff_SinceMode_RequiresRef(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := RunDiff(context.Background(), DiffOptions{
		Mode:    DiffModeSince,
		RootDir: dir,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "since ref required")
}

func TestRunDiff_BaseHeadMode_RequiresBothRefs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	t.Run("missing head", func(t *testing.T) {
		t.Parallel()
		_, err := RunDiff(context.Background(), DiffOptions{
			Mode:    DiffModeBaseHead,
			RootDir: dir,
			BaseRef: "main",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "both base and head refs required")
	})

	t.Run("missing base", func(t *testing.T) {
		t.Parallel()
		_, err := RunDiff(context.Background(), DiffOptions{
			Mode:    DiffModeBaseHead,
			RootDir: dir,
			HeadRef: "feature",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "both base and head refs required")
	})
}

// --- Git-based diff integration tests ---

func TestRunDiff_SinceMode_Integration(t *testing.T) {
	t.Parallel()

	dir := setupDiffTestRepo(t)

	// Get initial SHA.
	initialSHA := runGitCmd(t, dir, "rev-parse", "--short", "HEAD")

	// Create a second commit with a new file.
	writeTestFile(t, filepath.Join(dir, "second.go"), "package main\n// second")
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "second commit")

	output, err := RunDiff(context.Background(), DiffOptions{
		Mode:     DiffModeSince,
		RootDir:  dir,
		SinceRef: initialSHA,
	})
	require.NoError(t, err)
	require.NotNil(t, output)
	require.NotNil(t, output.Result)

	assert.Contains(t, output.Result.Added, "second.go")
	assert.Contains(t, output.Summary, "1 added")
}

func TestRunDiff_SinceMode_HEAD1(t *testing.T) {
	t.Parallel()

	dir := setupDiffTestRepo(t)

	// Create a second commit.
	writeTestFile(t, filepath.Join(dir, "new_file.go"), "package main")
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "second commit")

	output, err := RunDiff(context.Background(), DiffOptions{
		Mode:     DiffModeSince,
		RootDir:  dir,
		SinceRef: "HEAD~1",
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	assert.Contains(t, output.Result.Added, "new_file.go")
}

func TestRunDiff_BaseHeadMode_Integration(t *testing.T) {
	t.Parallel()

	dir := setupDiffTestRepo(t)

	// Get SHA of initial commit on main.
	initialSHA := runGitCmd(t, dir, "rev-parse", "HEAD")

	// Create a feature branch with changes.
	runGitCmd(t, dir, "checkout", "-b", "feature")
	writeTestFile(t, filepath.Join(dir, "feature.go"), "package feature")
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "feature commit")

	featureSHA := runGitCmd(t, dir, "rev-parse", "HEAD")

	output, err := RunDiff(context.Background(), DiffOptions{
		Mode:    DiffModeBaseHead,
		RootDir: dir,
		BaseRef: initialSHA[:7],
		HeadRef: featureSHA[:7],
	})
	require.NoError(t, err)
	require.NotNil(t, output)

	assert.Contains(t, output.Result.Added, "feature.go")
}

func TestRunDiff_CacheMode_Integration(t *testing.T) {
	t.Parallel()

	dir := setupDiffTestRepo(t)

	// Create initial state and save it to cache.
	profileName := "test-cache-diff"
	cache := NewStateCache(profileName)

	previousSnap := NewStateSnapshot(profileName, dir, "", "")
	previousSnap.AddFile("file1.go", FileState{
		Size:         12,
		ContentHash:  NewXXH3Hasher().HashString("package main"),
		ModifiedTime: "2026-01-01T00:00:00Z",
	})

	require.NoError(t, cache.SaveState(dir, previousSnap))

	// Modify file1.go to trigger a change.
	writeTestFile(t, filepath.Join(dir, "file1.go"), "package main\n// modified")

	output, err := RunDiff(context.Background(), DiffOptions{
		Mode:        DiffModeCache,
		RootDir:     dir,
		ProfileName: profileName,
	})
	require.NoError(t, err)
	require.NotNil(t, output)
	assert.True(t, output.Result.HasChanges())
}

// --- walkDir tests ---

func TestWalkDir_SimpleDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.go"), "package a")
	writeTestFile(t, filepath.Join(dir, "b.go"), "package b")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0755))
	writeTestFile(t, filepath.Join(dir, "sub", "c.go"), "package c")

	var files []string
	err := walkDir(context.Background(), dir, func(relPath, absPath string, size int64, modTime string) error {
		files = append(files, relPath)
		return nil
	})
	require.NoError(t, err)

	assert.Contains(t, files, "a.go")
	assert.Contains(t, files, "b.go")
	assert.Contains(t, files, filepath.Join("sub", "c.go"))
}

func TestWalkDir_SkipsHiddenDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "visible.go"), "visible")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".hidden"), 0755))
	writeTestFile(t, filepath.Join(dir, ".hidden", "secret.go"), "hidden")

	var files []string
	err := walkDir(context.Background(), dir, func(relPath, absPath string, size int64, modTime string) error {
		files = append(files, relPath)
		return nil
	})
	require.NoError(t, err)

	assert.Contains(t, files, "visible.go")
	for _, f := range files {
		assert.NotContains(t, f, ".hidden")
	}
}

func TestWalkDir_SkipsVendorAndNodeModules(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "main.go"), "package main")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "vendor"), 0755))
	writeTestFile(t, filepath.Join(dir, "vendor", "lib.go"), "vendor code")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules"), 0755))
	writeTestFile(t, filepath.Join(dir, "node_modules", "pkg.js"), "node code")

	var files []string
	err := walkDir(context.Background(), dir, func(relPath, absPath string, size int64, modTime string) error {
		files = append(files, relPath)
		return nil
	})
	require.NoError(t, err)

	assert.Len(t, files, 1)
	assert.Equal(t, "main.go", files[0])
}

func TestWalkDir_ContextCancellation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "file.go"), "package main")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := walkDir(ctx, dir, func(relPath, absPath string, size int64, modTime string) error {
		return nil
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// --- isSkippedDir tests ---

func TestIsSkippedDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{name: "node_modules", dir: "node_modules", want: true},
		{name: "vendor", dir: "vendor", want: true},
		{name: "__pycache__", dir: "__pycache__", want: true},
		{name: "dist", dir: "dist", want: true},
		{name: "build", dir: "build", want: true},
		{name: "target", dir: "target", want: true},
		{name: "src is not skipped", dir: "src", want: false},
		{name: "internal is not skipped", dir: "internal", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isSkippedDir(tt.dir))
		})
	}
}

// --- State save behavior tests ---

func TestRunDiff_DoesNotSaveState(t *testing.T) {
	t.Parallel()

	dir := setupDiffTestRepo(t)

	// Create a second commit.
	writeTestFile(t, filepath.Join(dir, "added.go"), "package main")
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "add file")

	profileName := "diff-no-save"
	cache := NewStateCache(profileName)

	// Verify no state exists before diff.
	assert.False(t, cache.HasState(dir))

	_, err := RunDiff(context.Background(), DiffOptions{
		Mode:        DiffModeSince,
		RootDir:     dir,
		ProfileName: profileName,
		SinceRef:    "HEAD~1",
	})
	require.NoError(t, err)

	// Verify no state exists after diff (diff is read-only).
	assert.False(t, cache.HasState(dir), "diff command must NOT save state to cache")
}

// --- Test helpers ---

// setupDiffTestRepo creates a temporary git repository with an initial commit.
func setupDiffTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@test.com")
	runGitCmd(t, dir, "config", "user.name", "Test")

	writeTestFile(t, filepath.Join(dir, "file1.go"), "package main")
	runGitCmd(t, dir, "add", ".")
	runGitCmd(t, dir, "commit", "-m", "initial")

	return dir
}

// runGitCmd runs a git command in the given directory and returns trimmed output.
func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, string(out))
	return trimNewlines(string(out))
}

// trimNewlines removes leading and trailing newlines from a string.
func trimNewlines(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

// writeTestFile writes content to a file, creating parent directories as needed.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

// --- buildCurrentSnapshot tests ---

func TestBuildCurrentSnapshot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "file1.go"), "package main")
	writeTestFile(t, filepath.Join(dir, "file2.go"), "package util")

	snap, err := buildCurrentSnapshot(context.Background(), dir, "test")
	require.NoError(t, err)
	require.NotNil(t, snap)

	assert.Equal(t, "test", snap.ProfileName)
	assert.Len(t, snap.Files, 2)
	assert.Contains(t, snap.Files, "file1.go")
	assert.Contains(t, snap.Files, "file2.go")

	// Verify hashes are non-zero.
	for _, fs := range snap.Files {
		assert.NotZero(t, fs.ContentHash)
		assert.NotZero(t, fs.Size)
		assert.NotEmpty(t, fs.ModifiedTime)
	}
}