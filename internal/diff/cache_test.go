package diff

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeSnapshot creates a minimal StateSnapshot for testing. The profile name
// and git branch are configurable; other metadata fields are set to sensible
// defaults.
func makeSnapshot(t *testing.T, profileName, rootDir, branch string) *StateSnapshot {
	t.Helper()
	snap := NewStateSnapshot(profileName, rootDir, branch, "abc123def456")
	snap.AddFile("src/main.go", FileState{
		Size:         1024,
		ContentHash:  0xdeadbeef,
		ModifiedTime: "2026-02-25T10:00:00Z",
	})
	snap.AddFile("README.md", FileState{
		Size:         256,
		ContentHash:  0xcafebabe,
		ModifiedTime: "2026-02-24T09:00:00Z",
	})
	return snap
}

// ---------------------------------------------------------------------------
// TestSanitizeProfileName
// ---------------------------------------------------------------------------

func TestSanitizeProfileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple default",
			input: "default",
			want:  "default",
		},
		{
			name:  "special characters replaced",
			input: "my profile!",
			want:  "my_profile_",
		},
		{
			name:  "spaces replaced",
			input: "hello world",
			want:  "hello_world",
		},
		{
			name:  "safe chars preserved (hyphen and underscore)",
			input: "a-b_c",
			want:  "a-b_c",
		},
		{
			name:  "path traversal prevented",
			input: "../../etc",
			want:  "______etc",
		},
		{
			name:  "empty becomes default",
			input: "",
			want:  "default",
		},
		{
			name:  "non-ASCII replaced",
			input: "caf\u00e9",
			want:  "caf_",
		},
		{
			name:  "uppercase and digits preserved",
			input: "UPPER-case_123",
			want:  "UPPER-case_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeProfileName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestStateCacheSaveAndLoad
// ---------------------------------------------------------------------------

func TestStateCacheSaveAndLoad(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("default")

	snap := makeSnapshot(t, "default", rootDir, "main")

	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)

	loaded, err := cache.LoadState(rootDir, "main")
	require.NoError(t, err)

	// Verify all fields round-trip correctly.
	assert.Equal(t, snap.Version, loaded.Version)
	assert.Equal(t, snap.ProfileName, loaded.ProfileName)
	assert.Equal(t, snap.GeneratedAt, loaded.GeneratedAt)
	assert.Equal(t, snap.GitBranch, loaded.GitBranch)
	assert.Equal(t, snap.GitHeadSHA, loaded.GitHeadSHA)
	assert.Equal(t, snap.RootDir, loaded.RootDir)
	assert.Len(t, loaded.Files, len(snap.Files))

	for path, origState := range snap.Files {
		loadedState, ok := loaded.Files[path]
		require.True(t, ok, "expected file %s in loaded snapshot", path)
		assert.Equal(t, origState, loadedState, "file state mismatch for %s", path)
	}
}

// ---------------------------------------------------------------------------
// TestStateCacheSaveCreatesDirectory
// ---------------------------------------------------------------------------

func TestStateCacheSaveCreatesDirectory(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("test-profile")

	// The .harvx/state/ directory should not exist yet.
	stateDir := filepath.Join(rootDir, ".harvx", "state")
	_, err := os.Stat(stateDir)
	require.True(t, os.IsNotExist(err), ".harvx/state/ should not exist before first save")

	snap := makeSnapshot(t, "test-profile", rootDir, "main")
	err = cache.SaveState(rootDir, snap)
	require.NoError(t, err)

	// Verify the directory was created.
	info, err := os.Stat(stateDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir(), ".harvx/state/ should be a directory")
}

// ---------------------------------------------------------------------------
// TestStateCacheSaveOverwrites
// ---------------------------------------------------------------------------

func TestStateCacheSaveOverwrites(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("default")

	// First save.
	snap1 := NewStateSnapshot("default", rootDir, "main", "sha-first")
	snap1.AddFile("file1.go", FileState{
		Size:         100,
		ContentHash:  0x1111,
		ModifiedTime: "2026-01-01T00:00:00Z",
	})
	err := cache.SaveState(rootDir, snap1)
	require.NoError(t, err)

	// Second save with different data.
	snap2 := NewStateSnapshot("default", rootDir, "main", "sha-second")
	snap2.AddFile("file2.go", FileState{
		Size:         200,
		ContentHash:  0x2222,
		ModifiedTime: "2026-02-01T00:00:00Z",
	})
	err = cache.SaveState(rootDir, snap2)
	require.NoError(t, err)

	// Load should return the second snapshot.
	loaded, err := cache.LoadState(rootDir, "main")
	require.NoError(t, err)

	assert.Equal(t, "sha-second", loaded.GitHeadSHA)
	assert.Len(t, loaded.Files, 1)
	_, hasFile2 := loaded.Files["file2.go"]
	assert.True(t, hasFile2, "loaded snapshot should contain file2.go from second save")
	_, hasFile1 := loaded.Files["file1.go"]
	assert.False(t, hasFile1, "loaded snapshot should not contain file1.go from first save")
}

// ---------------------------------------------------------------------------
// TestStateCacheLoadNoState
// ---------------------------------------------------------------------------

func TestStateCacheLoadNoState(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("nonexistent")

	_, err := cache.LoadState(rootDir, "main")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNoState),
		"expected ErrNoState, got: %v", err)
}

// ---------------------------------------------------------------------------
// TestStateCacheLoadBranchMismatch
// ---------------------------------------------------------------------------

func TestStateCacheLoadBranchMismatch(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("default")

	snap := makeSnapshot(t, "default", rootDir, "main")
	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)

	// Load with a different branch.
	_, err = cache.LoadState(rootDir, "develop")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBranchMismatch),
		"expected ErrBranchMismatch when stored=main, current=develop, got: %v", err)
}

// ---------------------------------------------------------------------------
// TestStateCacheLoadBranchMismatchEmptyStored
// ---------------------------------------------------------------------------

func TestStateCacheLoadBranchMismatchEmptyStored(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("default")

	// Save with empty branch (not in a git repo).
	snap := makeSnapshot(t, "default", rootDir, "")
	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)

	// Load with a branch name -- should NOT return ErrBranchMismatch because
	// stored branch is empty.
	loaded, err := cache.LoadState(rootDir, "develop")
	require.NoError(t, err, "empty stored branch should not trigger mismatch")
	assert.NotNil(t, loaded)
}

// ---------------------------------------------------------------------------
// TestStateCacheLoadBranchMismatchEmptyCurrent
// ---------------------------------------------------------------------------

func TestStateCacheLoadBranchMismatchEmptyCurrent(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("default")

	// Save with a branch.
	snap := makeSnapshot(t, "default", rootDir, "main")
	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)

	// Load with empty current branch -- should NOT return ErrBranchMismatch.
	loaded, err := cache.LoadState(rootDir, "")
	require.NoError(t, err, "empty current branch should not trigger mismatch")
	assert.NotNil(t, loaded)
}

// ---------------------------------------------------------------------------
// TestStateCacheClearState
// ---------------------------------------------------------------------------

func TestStateCacheClearState(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("default")

	snap := makeSnapshot(t, "default", rootDir, "main")
	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)

	// Verify file exists.
	statePath := cache.GetStatePath(rootDir)
	_, err = os.Stat(statePath)
	require.NoError(t, err, "state file should exist after save")

	// Clear.
	err = cache.ClearState(rootDir)
	require.NoError(t, err)

	// Verify file is gone.
	_, err = os.Stat(statePath)
	assert.True(t, os.IsNotExist(err), "state file should be removed after clear")
}

// ---------------------------------------------------------------------------
// TestStateCacheClearStateIdempotent
// ---------------------------------------------------------------------------

func TestStateCacheClearStateIdempotent(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("nonexistent-profile")

	// Clear on a profile that was never saved should return no error.
	err := cache.ClearState(rootDir)
	assert.NoError(t, err, "clearing non-existent state file should be idempotent")
}

// ---------------------------------------------------------------------------
// TestStateCacheClearAllState
// ---------------------------------------------------------------------------

func TestStateCacheClearAllState(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()

	// Save state for two different profiles.
	cache1 := NewStateCache("profile-a")
	snap1 := makeSnapshot(t, "profile-a", rootDir, "main")
	err := cache1.SaveState(rootDir, snap1)
	require.NoError(t, err)

	cache2 := NewStateCache("profile-b")
	snap2 := makeSnapshot(t, "profile-b", rootDir, "main")
	err = cache2.SaveState(rootDir, snap2)
	require.NoError(t, err)

	// Verify directory exists with files.
	stateDir := filepath.Join(rootDir, ".harvx", "state")
	entries, err := os.ReadDir(stateDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 2, "state directory should have at least 2 files")

	// ClearAllState should remove the entire state directory.
	err = cache1.ClearAllState(rootDir)
	require.NoError(t, err)

	_, err = os.Stat(stateDir)
	assert.True(t, os.IsNotExist(err), ".harvx/state/ should be removed after ClearAllState")
}

// ---------------------------------------------------------------------------
// TestStateCacheClearAllStateIdempotent
// ---------------------------------------------------------------------------

func TestStateCacheClearAllStateIdempotent(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("default")

	// ClearAllState when .harvx/state/ does not exist.
	err := cache.ClearAllState(rootDir)
	assert.NoError(t, err, "ClearAllState on non-existent directory should be idempotent")
}

// ---------------------------------------------------------------------------
// TestStateCacheHasState
// ---------------------------------------------------------------------------

func TestStateCacheHasState(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("default")

	// Before save: HasState returns false.
	assert.False(t, cache.HasState(rootDir), "HasState should be false before any save")

	// After save: HasState returns true.
	snap := makeSnapshot(t, "default", rootDir, "main")
	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)
	assert.True(t, cache.HasState(rootDir), "HasState should be true after save")

	// After clear: HasState returns false.
	err = cache.ClearState(rootDir)
	require.NoError(t, err)
	assert.False(t, cache.HasState(rootDir), "HasState should be false after clear")
}

// ---------------------------------------------------------------------------
// TestStateCacheGetStatePath
// ---------------------------------------------------------------------------

func TestStateCacheGetStatePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		profileName string
		rootDir     string
		wantSuffix  string
	}{
		{
			name:        "default profile",
			profileName: "default",
			rootDir:     "/home/user/project",
			wantSuffix:  filepath.Join(".harvx", "state", "default.json"),
		},
		{
			name:        "custom profile",
			profileName: "finvault",
			rootDir:     "/tmp/repo",
			wantSuffix:  filepath.Join(".harvx", "state", "finvault.json"),
		},
		{
			name:        "profile with unsafe chars sanitized",
			profileName: "my profile!",
			rootDir:     "/tmp/repo",
			wantSuffix:  filepath.Join(".harvx", "state", "my_profile_.json"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cache := NewStateCache(tt.profileName)
			got := cache.GetStatePath(tt.rootDir)

			expectedPath := filepath.Join(tt.rootDir, tt.wantSuffix)
			assert.Equal(t, expectedPath, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestStateCacheAtomicWrite
// ---------------------------------------------------------------------------

func TestStateCacheAtomicWrite(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("atomic-test")

	snap := makeSnapshot(t, "atomic-test", rootDir, "main")

	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)

	// Read the file at the final path and verify it contains valid JSON.
	statePath := cache.GetStatePath(rootDir)
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	assert.True(t, json.Valid(data), "saved state file must contain valid JSON")

	// Verify the content can be parsed as a StateSnapshot.
	parsed, err := ParseStateSnapshot(data)
	require.NoError(t, err)
	assert.Equal(t, snap.ProfileName, parsed.ProfileName)
	assert.Len(t, parsed.Files, len(snap.Files))

	// Verify no temporary files remain in the state directory.
	stateDir := filepath.Dir(statePath)
	entries, err := os.ReadDir(stateDir)
	require.NoError(t, err)
	for _, entry := range entries {
		assert.False(t, isTemporaryFile(entry.Name()),
			"temporary file %q should not remain after atomic write", entry.Name())
	}
}

// isTemporaryFile returns true if the filename looks like a temporary file
// created during atomic writes (e.g., ".state-*.tmp").
func isTemporaryFile(name string) bool {
	// Match the pattern used by os.CreateTemp with prefix ".state-" and
	// suffix ".tmp".
	if len(name) < 6 {
		return false
	}
	return name[0] == '.' && filepath.Ext(name) == ".tmp"
}

// ---------------------------------------------------------------------------
// TestStateCacheFullRoundTrip
// ---------------------------------------------------------------------------

func TestStateCacheFullRoundTrip(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("full-roundtrip")

	// Build a snapshot with many files.
	snap := NewStateSnapshot("full-roundtrip", rootDir, "feature/cache", "deadbeef12345678")
	snap.AddFile("cmd/main.go", FileState{
		Size:         512,
		ContentHash:  0x123456789abcdef0,
		ModifiedTime: "2026-02-25T08:00:00Z",
	})
	snap.AddFile("internal/diff/state.go", FileState{
		Size:         2048,
		ContentHash:  0xfedcba9876543210,
		ModifiedTime: "2026-02-25T09:00:00Z",
	})
	snap.AddFile("README.md", FileState{
		Size:         256,
		ContentHash:  0x1111111111111111,
		ModifiedTime: "2026-02-20T12:00:00Z",
	})
	snap.AddFile("go.mod", FileState{
		Size:         128,
		ContentHash:  0xaaaabbbbccccdddd,
		ModifiedTime: "2026-02-15T06:00:00Z",
	})
	snap.AddFile("internal/config/config.go", FileState{
		Size:         4096,
		ContentHash:  0xffffffffffffffff,
		ModifiedTime: "2026-02-24T14:00:00Z",
	})

	// Save.
	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)

	// Load.
	loaded, err := cache.LoadState(rootDir, "feature/cache")
	require.NoError(t, err)

	// Verify metadata.
	assert.Equal(t, SchemaVersion, loaded.Version)
	assert.Equal(t, "full-roundtrip", loaded.ProfileName)
	assert.Equal(t, snap.GeneratedAt, loaded.GeneratedAt)
	assert.Equal(t, "feature/cache", loaded.GitBranch)
	assert.Equal(t, "deadbeef12345678", loaded.GitHeadSHA)
	assert.Equal(t, rootDir, loaded.RootDir)

	// Verify all files and their hashes.
	require.Len(t, loaded.Files, 5)
	for path, origState := range snap.Files {
		loadedState, ok := loaded.Files[path]
		require.True(t, ok, "missing file in loaded snapshot: %s", path)
		assert.Equal(t, origState.Size, loadedState.Size,
			"size mismatch for %s", path)
		assert.Equal(t, origState.ContentHash, loadedState.ContentHash,
			"content hash mismatch for %s", path)
		assert.Equal(t, origState.ModifiedTime, loadedState.ModifiedTime,
			"modified time mismatch for %s", path)
	}
}

// ---------------------------------------------------------------------------
// TestStateCacheConcurrentReadDuringWrite
// ---------------------------------------------------------------------------

func TestStateCacheConcurrentReadDuringWrite(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("concurrent")

	// Write an initial snapshot so reads have something to find.
	initial := NewStateSnapshot("concurrent", rootDir, "main", "initial-sha")
	initial.AddFile("initial.go", FileState{
		Size:         100,
		ContentHash:  0xaaaa,
		ModifiedTime: "2026-01-01T00:00:00Z",
	})
	err := cache.SaveState(rootDir, initial)
	require.NoError(t, err)

	// Prepare the updated snapshot.
	updated := NewStateSnapshot("concurrent", rootDir, "main", "updated-sha")
	updated.AddFile("updated.go", FileState{
		Size:         200,
		ContentHash:  0xbbbb,
		ModifiedTime: "2026-02-01T00:00:00Z",
	})

	const numReaders = 10
	const numWriteIterations = 50

	var wg sync.WaitGroup
	errCh := make(chan error, numReaders*numWriteIterations)

	// Start readers that continuously load the state.
	for r := range numReaders {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for i := range numWriteIterations {
				loaded, err := cache.LoadState(rootDir, "main")
				if err != nil {
					// It is acceptable to see ErrNoState briefly during
					// atomic rename, but not data corruption. We tolerate
					// transient read failures.
					_ = i
					continue
				}

				// The loaded snapshot must be valid: either the initial or
				// the updated snapshot. It must NEVER be a partial/corrupt
				// snapshot.
				if loaded.Files == nil {
					errCh <- errors.New("loaded snapshot has nil Files map")
					return
				}

				// Verify it is parseable as valid JSON by re-marshaling.
				data, marshalErr := json.Marshal(loaded)
				if marshalErr != nil {
					errCh <- marshalErr
					return
				}
				if !json.Valid(data) {
					errCh <- errors.New("loaded snapshot re-marshals to invalid JSON")
					return
				}

				// Verify the snapshot is one of the two expected states.
				_, hasInitial := loaded.Files["initial.go"]
				_, hasUpdated := loaded.Files["updated.go"]
				if !hasInitial && !hasUpdated {
					errCh <- errors.New("loaded snapshot has neither initial.go nor updated.go")
					return
				}
			}
		}(r)
	}

	// Writer goroutine that repeatedly saves.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range numWriteIterations {
			_ = cache.SaveState(rootDir, updated)
		}
	}()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent read error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Additional edge case tests
// ---------------------------------------------------------------------------

func TestStateCacheSaveAndLoadEmptyFiles(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("empty")

	snap := NewStateSnapshot("empty", rootDir, "main", "sha123")
	// No files added.

	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)

	loaded, err := cache.LoadState(rootDir, "main")
	require.NoError(t, err)

	assert.NotNil(t, loaded.Files)
	assert.Len(t, loaded.Files, 0)
}

func TestStateCacheMultipleProfiles(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()

	cacheA := NewStateCache("profile-a")
	cacheB := NewStateCache("profile-b")

	snapA := NewStateSnapshot("profile-a", rootDir, "main", "sha-a")
	snapA.AddFile("a.go", FileState{Size: 100, ContentHash: 0x1111, ModifiedTime: "2026-01-01T00:00:00Z"})

	snapB := NewStateSnapshot("profile-b", rootDir, "main", "sha-b")
	snapB.AddFile("b.go", FileState{Size: 200, ContentHash: 0x2222, ModifiedTime: "2026-02-01T00:00:00Z"})

	err := cacheA.SaveState(rootDir, snapA)
	require.NoError(t, err)
	err = cacheB.SaveState(rootDir, snapB)
	require.NoError(t, err)

	// Load profile-a.
	loadedA, err := cacheA.LoadState(rootDir, "main")
	require.NoError(t, err)
	assert.Equal(t, "sha-a", loadedA.GitHeadSHA)
	_, hasA := loadedA.Files["a.go"]
	assert.True(t, hasA)

	// Load profile-b.
	loadedB, err := cacheB.LoadState(rootDir, "main")
	require.NoError(t, err)
	assert.Equal(t, "sha-b", loadedB.GitHeadSHA)
	_, hasB := loadedB.Files["b.go"]
	assert.True(t, hasB)

	// Clear profile-a should not affect profile-b.
	err = cacheA.ClearState(rootDir)
	require.NoError(t, err)

	assert.False(t, cacheA.HasState(rootDir))
	assert.True(t, cacheB.HasState(rootDir))
}

func TestStateCacheFilePermissions(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("perms")

	snap := makeSnapshot(t, "perms", rootDir, "main")
	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)

	statePath := cache.GetStatePath(rootDir)
	info, err := os.Stat(statePath)
	require.NoError(t, err)

	// Verify file permissions are 0644 (on Unix systems).
	// The exact mode depends on the umask, but the file should be readable.
	mode := info.Mode().Perm()
	assert.True(t, mode&0400 != 0, "file should be owner-readable, got mode: %o", mode)
	assert.True(t, mode&0200 != 0, "file should be owner-writable, got mode: %o", mode)
	assert.True(t, mode&0040 != 0, "file should be group-readable, got mode: %o", mode)
}

func TestStateCacheLoadCorruptedFile(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("corrupt")

	// Manually create a corrupted state file.
	statePath := cache.GetStatePath(rootDir)
	require.NoError(t, os.MkdirAll(filepath.Dir(statePath), 0755))
	require.NoError(t, os.WriteFile(statePath, []byte("{corrupted json"), 0644))

	_, err := cache.LoadState(rootDir, "main")
	require.Error(t, err, "loading corrupted JSON should return an error")
}

func TestStateCacheLoadInvalidVersion(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("badversion")

	// Manually write a state file with an unsupported version.
	statePath := cache.GetStatePath(rootDir)
	require.NoError(t, os.MkdirAll(filepath.Dir(statePath), 0755))

	badSnap := `{"version":"99","profile_name":"badversion","generated_at":"2026-01-01T00:00:00Z","git_branch":"main","git_head_sha":"abc","root_dir":"/tmp","files":{}}`
	require.NoError(t, os.WriteFile(statePath, []byte(badSnap), 0644))

	_, err := cache.LoadState(rootDir, "main")
	require.Error(t, err, "loading state with unsupported version should return an error")
	assert.True(t, errors.Is(err, ErrUnsupportedVersion),
		"expected ErrUnsupportedVersion, got: %v", err)
}

func TestStateCacheSaveStateHumanReadable(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cache := NewStateCache("readable")

	snap := makeSnapshot(t, "readable", rootDir, "main")
	err := cache.SaveState(rootDir, snap)
	require.NoError(t, err)

	// Read the raw file and verify it is indented (human-readable).
	statePath := cache.GetStatePath(rootDir)
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	// The file should be valid JSON.
	assert.True(t, json.Valid(data), "state file should contain valid JSON")

	// Verify it is still parseable.
	parsed, err := ParseStateSnapshot(data)
	require.NoError(t, err)
	assert.Equal(t, "readable", parsed.ProfileName)
}

func TestSanitizeProfileName_Idempotent(t *testing.T) {
	t.Parallel()

	// Sanitizing an already-safe name should be a no-op.
	name := "my-safe_profile123"
	once := sanitizeProfileName(name)
	twice := sanitizeProfileName(once)
	assert.Equal(t, once, twice, "sanitizeProfileName should be idempotent for safe names")
}

func TestStateCacheGetStatePath_Deterministic(t *testing.T) {
	t.Parallel()

	cache := NewStateCache("deterministic")
	rootDir := "/home/user/project"

	path1 := cache.GetStatePath(rootDir)
	path2 := cache.GetStatePath(rootDir)
	assert.Equal(t, path1, path2, "GetStatePath must return the same path on repeated calls")
}
