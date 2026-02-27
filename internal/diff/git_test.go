package diff

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a temporary git repository with an initial commit
// containing file1.go. It returns the directory path.
func setupTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@test.com")
	runCmd(t, dir, "git", "config", "user.name", "Test")

	writeFile(t, filepath.Join(dir, "file1.go"), "package main")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "initial")

	return dir
}

// runCmd runs an external command in the given directory and fails the test on
// error.
func runCmd(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "command %s %v failed: %s", name, args, string(out))
	return string(out)
}

// writeFile writes content to path, creating any necessary directories.
func writeFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

// TestGitChangeType_String verifies the String method on GitChangeType.
func TestGitChangeType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change GitChangeType
		want   string
	}{
		{name: "added", change: GitAdded, want: "added"},
		{name: "modified", change: GitModified, want: "modified"},
		{name: "deleted", change: GitDeleted, want: "deleted"},
		{name: "renamed", change: GitRenamed, want: "renamed"},
		{name: "unknown value", change: GitChangeType(99), want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.change.String())
		})
	}
}

// TestNewGitDiffer verifies the constructor returns a non-nil GitDiffer.
func TestNewGitDiffer(t *testing.T) {
	t.Parallel()

	d := NewGitDiffer()
	assert.NotNil(t, d)
}

// TestGetCurrentBranch verifies branch name retrieval from a real git repo.
func TestGetCurrentBranch(t *testing.T) {
	t.Parallel()

	t.Run("returns branch name", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)
		d := NewGitDiffer()

		branch, err := d.GetCurrentBranch(context.Background(), dir)
		require.NoError(t, err)
		// Git defaults to "master" or "main" depending on git config.
		assert.NotEmpty(t, branch)
	})

	t.Run("returns empty string for detached HEAD", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)
		// Get the HEAD SHA and detach.
		sha := runCmd(t, dir, "git", "rev-parse", "HEAD")
		runCmd(t, dir, "git", "checkout", "--detach", sha[:7])

		d := NewGitDiffer()
		branch, err := d.GetCurrentBranch(context.Background(), dir)
		require.NoError(t, err)
		assert.Empty(t, branch)
	})

	t.Run("returns ErrNotGitRepo for non-git directory", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		d := NewGitDiffer()

		_, err := d.GetCurrentBranch(context.Background(), dir)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotGitRepo), "expected ErrNotGitRepo, got: %v", err)
	})
}

// TestGetHeadSHA verifies HEAD SHA retrieval from a real git repo.
func TestGetHeadSHA(t *testing.T) {
	t.Parallel()

	t.Run("returns short SHA", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)
		d := NewGitDiffer()

		sha, err := d.GetHeadSHA(context.Background(), dir)
		require.NoError(t, err)
		// git rev-parse --short defaults to 7 chars but may use more for
		// disambiguation, so check a reasonable range.
		assert.GreaterOrEqual(t, len(sha), 7, "expected at least 7-character short SHA, got %q", sha)
		assert.LessOrEqual(t, len(sha), 12, "expected at most 12-character short SHA, got %q", sha)
	})

	t.Run("returns ErrNotGitRepo for non-git directory", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		d := NewGitDiffer()

		_, err := d.GetHeadSHA(context.Background(), dir)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotGitRepo), "expected ErrNotGitRepo, got: %v", err)
	})
}

// TestValidateRef verifies ref validation on a real git repo.
func TestValidateRef(t *testing.T) {
	t.Parallel()

	t.Run("valid ref returns nil", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)
		d := NewGitDiffer()

		err := d.ValidateRef(context.Background(), dir, "HEAD")
		assert.NoError(t, err)
	})

	t.Run("invalid ref returns ErrInvalidRef", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)
		d := NewGitDiffer()

		err := d.ValidateRef(context.Background(), dir, "nonexistent-ref-abc123")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidRef), "expected ErrInvalidRef, got: %v", err)
		assert.Contains(t, err.Error(), "nonexistent-ref-abc123")
	})

	t.Run("not a git repo returns ErrNotGitRepo not ErrInvalidRef", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		d := NewGitDiffer()

		err := d.ValidateRef(context.Background(), dir, "HEAD")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotGitRepo), "expected ErrNotGitRepo, got: %v", err)
		assert.False(t, errors.Is(err, ErrInvalidRef), "should not be ErrInvalidRef for non-git directory")
	})
}

// TestGetChangedFiles verifies diff detection between two refs in a real git repo.
func TestGetChangedFiles(t *testing.T) {
	t.Parallel()

	t.Run("detects added modified and deleted files", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)

		// Get SHA of initial commit.
		initialSHA := runCmd(t, dir, "git", "rev-parse", "--short", "HEAD")
		initialSHA = initialSHA[:7]

		// Make changes: add a file, modify existing, delete nothing yet.
		writeFile(t, filepath.Join(dir, "file2.go"), "package main\n// new file")
		writeFile(t, filepath.Join(dir, "file1.go"), "package main\n// modified")
		runCmd(t, dir, "git", "add", ".")
		runCmd(t, dir, "git", "commit", "-m", "second commit")

		// Add a third commit that deletes file1.go.
		require.NoError(t, os.Remove(filepath.Join(dir, "file1.go")))
		runCmd(t, dir, "git", "add", ".")
		runCmd(t, dir, "git", "commit", "-m", "third commit - delete file1")

		d := NewGitDiffer()
		changes, err := d.GetChangedFiles(context.Background(), dir, initialSHA, "HEAD")
		require.NoError(t, err)

		// file1.go was modified then deleted, so git diff initial..HEAD shows D.
		// file2.go was added.
		var addedPaths, deletedPaths []string
		for _, c := range changes {
			switch c.Status {
			case GitAdded:
				addedPaths = append(addedPaths, c.Path)
			case GitDeleted:
				deletedPaths = append(deletedPaths, c.Path)
			}
		}

		assert.Contains(t, addedPaths, "file2.go")
		assert.Contains(t, deletedPaths, "file1.go")
	})

	t.Run("handles renamed files", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)

		initialSHA := runCmd(t, dir, "git", "rev-parse", "--short", "HEAD")
		initialSHA = initialSHA[:7]

		// Rename file1.go to renamed.go.
		runCmd(t, dir, "git", "mv", "file1.go", "renamed.go")
		runCmd(t, dir, "git", "commit", "-m", "rename file1 to renamed")

		d := NewGitDiffer()
		changes, err := d.GetChangedFiles(context.Background(), dir, initialSHA, "HEAD")
		require.NoError(t, err)
		require.NotEmpty(t, changes)

		// Git should detect a rename.
		hasRename := false
		for _, c := range changes {
			if c.Status == GitRenamed {
				hasRename = true
				assert.Equal(t, "renamed.go", c.Path)
				assert.Equal(t, "file1.go", c.OldPath)
			}
		}
		assert.True(t, hasRename, "expected a rename to be detected")
	})

	t.Run("no changes between same refs", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)

		d := NewGitDiffer()
		changes, err := d.GetChangedFiles(context.Background(), dir, "HEAD", "HEAD")
		require.NoError(t, err)
		assert.Empty(t, changes)
	})

	t.Run("invalid base ref returns error", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)
		d := NewGitDiffer()

		_, err := d.GetChangedFiles(context.Background(), dir, "nonexistent", "HEAD")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidRef))
	})

	t.Run("invalid head ref returns error", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)
		d := NewGitDiffer()

		_, err := d.GetChangedFiles(context.Background(), dir, "HEAD", "nonexistent")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidRef))
	})
}

// TestGetChangedFilesSince verifies diff detection since a ref.
func TestGetChangedFilesSince(t *testing.T) {
	t.Parallel()

	t.Run("detects changes since ref", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)

		// Get SHA of initial commit.
		initialSHA := runCmd(t, dir, "git", "rev-parse", "--short", "HEAD")
		initialSHA = initialSHA[:7]

		// Make changes.
		writeFile(t, filepath.Join(dir, "new_file.go"), "package main")
		writeFile(t, filepath.Join(dir, "file1.go"), "package main\n// changed")
		runCmd(t, dir, "git", "add", ".")
		runCmd(t, dir, "git", "commit", "-m", "second commit")

		d := NewGitDiffer()
		changes, err := d.GetChangedFilesSince(context.Background(), dir, initialSHA)
		require.NoError(t, err)

		var paths []string
		for _, c := range changes {
			paths = append(paths, c.Path)
		}
		assert.Contains(t, paths, "new_file.go")
		assert.Contains(t, paths, "file1.go")
	})

	t.Run("HEAD~1 detects last commit changes", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)

		// Create a second commit with a new file.
		writeFile(t, filepath.Join(dir, "second.go"), "package main")
		runCmd(t, dir, "git", "add", ".")
		runCmd(t, dir, "git", "commit", "-m", "second commit")

		d := NewGitDiffer()
		changes, err := d.GetChangedFilesSince(context.Background(), dir, "HEAD~1")
		require.NoError(t, err)

		require.Len(t, changes, 1)
		assert.Equal(t, "second.go", changes[0].Path)
		assert.Equal(t, GitAdded, changes[0].Status)
	})

	t.Run("invalid ref returns error", func(t *testing.T) {
		t.Parallel()

		dir := setupTestRepo(t)
		d := NewGitDiffer()

		_, err := d.GetChangedFilesSince(context.Background(), dir, "nonexistent-ref")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidRef))
	})
}

// TestBuildDiffResultFromGit verifies conversion of GitFileChange to DiffResult.
func TestBuildDiffResultFromGit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		changes      []GitFileChange
		wantAdded    []string
		wantModified []string
		wantDeleted  []string
		wantChanges  bool
	}{
		{
			name:        "empty changes",
			changes:     nil,
			wantChanges: false,
		},
		{
			name: "only added",
			changes: []GitFileChange{
				{Path: "c.go", Status: GitAdded},
				{Path: "a.go", Status: GitAdded},
				{Path: "b.go", Status: GitAdded},
			},
			wantAdded:   []string{"a.go", "b.go", "c.go"},
			wantChanges: true,
		},
		{
			name: "only modified",
			changes: []GitFileChange{
				{Path: "z.go", Status: GitModified},
				{Path: "a.go", Status: GitModified},
			},
			wantModified: []string{"a.go", "z.go"},
			wantChanges:  true,
		},
		{
			name: "only deleted",
			changes: []GitFileChange{
				{Path: "old.go", Status: GitDeleted},
			},
			wantDeleted: []string{"old.go"},
			wantChanges: true,
		},
		{
			name: "renamed becomes delete plus add",
			changes: []GitFileChange{
				{Path: "new_name.go", OldPath: "old_name.go", Status: GitRenamed},
			},
			wantAdded:   []string{"new_name.go"},
			wantDeleted: []string{"old_name.go"},
			wantChanges: true,
		},
		{
			name: "mixed changes sorted",
			changes: []GitFileChange{
				{Path: "z.go", Status: GitAdded},
				{Path: "m.go", Status: GitModified},
				{Path: "a.go", Status: GitDeleted},
				{Path: "new.go", OldPath: "old.go", Status: GitRenamed},
			},
			wantAdded:    []string{"new.go", "z.go"},
			wantModified: []string{"m.go"},
			wantDeleted:  []string{"a.go", "old.go"},
			wantChanges:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := BuildDiffResultFromGit(tt.changes)
			require.NotNil(t, result)

			assert.Equal(t, tt.wantAdded, result.Added, "Added mismatch")
			assert.Equal(t, tt.wantModified, result.Modified, "Modified mismatch")
			assert.Equal(t, tt.wantDeleted, result.Deleted, "Deleted mismatch")
			assert.Equal(t, tt.wantChanges, result.HasChanges(), "HasChanges mismatch")
		})
	}
}

// TestParseNameStatus verifies parsing of git diff --name-status output.
func TestParseNameStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		output  string
		want    []GitFileChange
		wantErr bool
	}{
		{
			name:   "empty output",
			output: "",
			want:   nil,
		},
		{
			name:   "single added file",
			output: "A\tfile.go",
			want: []GitFileChange{
				{Path: "file.go", Status: GitAdded},
			},
		},
		{
			name:   "single modified file",
			output: "M\tfile.go",
			want: []GitFileChange{
				{Path: "file.go", Status: GitModified},
			},
		},
		{
			name:   "single deleted file",
			output: "D\tfile.go",
			want: []GitFileChange{
				{Path: "file.go", Status: GitDeleted},
			},
		},
		{
			name:   "renamed file with score",
			output: "R100\told.go\tnew.go",
			want: []GitFileChange{
				{Path: "new.go", OldPath: "old.go", Status: GitRenamed},
			},
		},
		{
			name:   "renamed file with partial score",
			output: "R075\told.go\tnew.go",
			want: []GitFileChange{
				{Path: "new.go", OldPath: "old.go", Status: GitRenamed},
			},
		},
		{
			name:   "copied file treated as added",
			output: "C100\toriginal.go\tcopy.go",
			want: []GitFileChange{
				{Path: "copy.go", Status: GitAdded},
			},
		},
		{
			name:   "multiple changes",
			output: "A\tnew.go\nM\tchanged.go\nD\tremoved.go",
			want: []GitFileChange{
				{Path: "new.go", Status: GitAdded},
				{Path: "changed.go", Status: GitModified},
				{Path: "removed.go", Status: GitDeleted},
			},
		},
		{
			name:   "trailing newline ignored",
			output: "A\tfile.go\n",
			want: []GitFileChange{
				{Path: "file.go", Status: GitAdded},
			},
		},
		{
			name:    "malformed line no tab",
			output:  "Afile.go",
			wantErr: true,
		},
		{
			name:    "malformed rename missing third field",
			output:  "R100\told.go",
			wantErr: true,
		},
		{
			name:    "unknown status code",
			output:  "X\tfile.go",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseNameStatus(tt.output)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestNotGitRepo verifies that operations on a non-git directory return
// ErrNotGitRepo.
func TestNotGitRepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	d := NewGitDiffer()

	t.Run("GetCurrentBranch", func(t *testing.T) {
		t.Parallel()
		_, err := d.GetCurrentBranch(context.Background(), dir)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotGitRepo))
	})

	t.Run("GetHeadSHA", func(t *testing.T) {
		t.Parallel()
		_, err := d.GetHeadSHA(context.Background(), dir)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotGitRepo))
	})

	t.Run("ValidateRef", func(t *testing.T) {
		t.Parallel()
		err := d.ValidateRef(context.Background(), dir, "HEAD")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotGitRepo))
	})

	t.Run("GetChangedFiles", func(t *testing.T) {
		t.Parallel()
		_, err := d.GetChangedFiles(context.Background(), dir, "HEAD~1", "HEAD")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotGitRepo))
	})

	t.Run("GetChangedFilesSince", func(t *testing.T) {
		t.Parallel()
		_, err := d.GetChangedFilesSince(context.Background(), dir, "HEAD~1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotGitRepo))
	})
}

// TestContextCancellation verifies that a cancelled context stops git commands.
func TestContextCancellation(t *testing.T) {
	t.Parallel()

	dir := setupTestRepo(t)
	d := NewGitDiffer()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Allow the context to expire.
	time.Sleep(5 * time.Millisecond)

	_, err := d.GetCurrentBranch(ctx, dir)
	require.Error(t, err)
}

// TestIntegration_MultipleCommitDiff creates a test repo with multiple commits
// and verifies that diffing between the first and last commit produces correct
// results.
func TestIntegration_MultipleCommitDiff(t *testing.T) {
	t.Parallel()

	dir := setupTestRepo(t)

	// Record initial SHA.
	initialSHA := runCmd(t, dir, "git", "rev-parse", "--short", "HEAD")
	initialSHA = initialSHA[:7]

	// Commit 2: add file2.go, modify file1.go.
	writeFile(t, filepath.Join(dir, "file2.go"), "package util")
	writeFile(t, filepath.Join(dir, "file1.go"), "package main\n\nfunc main() {}")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "commit 2")

	// Commit 3: add file3.go, delete file2.go.
	writeFile(t, filepath.Join(dir, "file3.go"), "package helper")
	require.NoError(t, os.Remove(filepath.Join(dir, "file2.go")))
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "commit 3")

	// Commit 4: add dir/file4.go.
	writeFile(t, filepath.Join(dir, "dir", "file4.go"), "package dir")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "commit 4")

	d := NewGitDiffer()
	changes, err := d.GetChangedFiles(context.Background(), dir, initialSHA, "HEAD")
	require.NoError(t, err)

	result := BuildDiffResultFromGit(changes)
	assert.True(t, result.HasChanges())

	// file1.go was modified (net diff from initial to HEAD).
	assert.Contains(t, result.Modified, "file1.go")
	// file3.go and dir/file4.go were added.
	assert.Contains(t, result.Added, "file3.go")
	assert.Contains(t, result.Added, "dir/file4.go")
	// file2.go was added then deleted, so net diff shows nothing for it
	// (git is smart enough to recognize this).

	// Verify sorting.
	if len(result.Added) > 1 {
		for i := 0; i < len(result.Added)-1; i++ {
			assert.True(t, result.Added[i] < result.Added[i+1],
				"Added not sorted: %q >= %q", result.Added[i], result.Added[i+1])
		}
	}
}

// TestGetChangedFilesSince_SpecificSHA verifies diffing since a specific
// commit SHA works correctly.
func TestGetChangedFilesSince_SpecificSHA(t *testing.T) {
	t.Parallel()

	dir := setupTestRepo(t)

	// Get the initial SHA.
	initialSHA := runCmd(t, dir, "git", "rev-parse", "HEAD")
	initialSHA = initialSHA[:len(initialSHA)-1] // trim newline

	// Create a second commit.
	writeFile(t, filepath.Join(dir, "added.go"), "package added")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "add file")

	d := NewGitDiffer()
	changes, err := d.GetChangedFilesSince(context.Background(), dir, initialSHA)
	require.NoError(t, err)

	require.Len(t, changes, 1)
	assert.Equal(t, "added.go", changes[0].Path)
	assert.Equal(t, GitAdded, changes[0].Status)
}

// TestBuildDiffResultFromGit_EmptyChanges verifies that an empty change list
// produces a DiffResult with HasChanges() == false.
func TestBuildDiffResultFromGit_EmptyChanges(t *testing.T) {
	t.Parallel()

	result := BuildDiffResultFromGit(nil)
	require.NotNil(t, result)
	assert.False(t, result.HasChanges())
	assert.Equal(t, 0, result.TotalChanged())
	assert.Nil(t, result.Added)
	assert.Nil(t, result.Modified)
	assert.Nil(t, result.Deleted)
}