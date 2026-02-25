package diff

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildSnapshot creates a StateSnapshot with n files named file_00000.go through
// file_NNNNN.go. Each file has a deterministic size and content hash derived from
// its index. This is used by multiple tests and the benchmark.
func buildSnapshot(t testing.TB, n int) *StateSnapshot {
	t.Helper()
	snap := NewStateSnapshot("test", "/tmp", "main", "abc123")
	for i := 0; i < n; i++ {
		snap.AddFile(fmt.Sprintf("file_%05d.go", i), FileState{
			Size:         int64(i * 100),
			ContentHash:  uint64(i),
			ModifiedTime: "2026-01-01T00:00:00Z",
		})
	}
	return snap
}

func TestCompareStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		previous      *StateSnapshot
		current       *StateSnapshot
		wantAdded     []string
		wantModified  []string
		wantDeleted   []string
		wantUnchanged int
	}{
		{
			name:          "both empty",
			previous:      newEmptySnapshot(t),
			current:       newEmptySnapshot(t),
			wantAdded:     nil,
			wantModified:  nil,
			wantDeleted:   nil,
			wantUnchanged: 0,
		},
		{
			name:     "previous empty current has files",
			previous: newEmptySnapshot(t),
			current: snapshotWithFiles(t, map[string]FileState{
				"alpha.go": {Size: 100, ContentHash: 0x1, ModifiedTime: "2026-01-01T00:00:00Z"},
				"beta.go":  {Size: 200, ContentHash: 0x2, ModifiedTime: "2026-01-01T00:00:00Z"},
				"gamma.go": {Size: 300, ContentHash: 0x3, ModifiedTime: "2026-01-01T00:00:00Z"},
			}),
			wantAdded:     []string{"alpha.go", "beta.go", "gamma.go"},
			wantModified:  nil,
			wantDeleted:   nil,
			wantUnchanged: 0,
		},
		{
			name: "previous has files current empty",
			previous: snapshotWithFiles(t, map[string]FileState{
				"alpha.go": {Size: 100, ContentHash: 0x1, ModifiedTime: "2026-01-01T00:00:00Z"},
				"beta.go":  {Size: 200, ContentHash: 0x2, ModifiedTime: "2026-01-01T00:00:00Z"},
				"gamma.go": {Size: 300, ContentHash: 0x3, ModifiedTime: "2026-01-01T00:00:00Z"},
			}),
			current:       newEmptySnapshot(t),
			wantAdded:     nil,
			wantModified:  nil,
			wantDeleted:   []string{"alpha.go", "beta.go", "gamma.go"},
			wantUnchanged: 0,
		},
		{
			name: "identical snapshots",
			previous: snapshotWithFiles(t, map[string]FileState{
				"main.go":   {Size: 100, ContentHash: 0xaaa, ModifiedTime: "2026-01-01T00:00:00Z"},
				"util.go":   {Size: 200, ContentHash: 0xbbb, ModifiedTime: "2026-01-01T00:00:00Z"},
				"README.md": {Size: 50, ContentHash: 0xccc, ModifiedTime: "2026-01-01T00:00:00Z"},
			}),
			current: snapshotWithFiles(t, map[string]FileState{
				"main.go":   {Size: 100, ContentHash: 0xaaa, ModifiedTime: "2026-01-01T00:00:00Z"},
				"util.go":   {Size: 200, ContentHash: 0xbbb, ModifiedTime: "2026-01-01T00:00:00Z"},
				"README.md": {Size: 50, ContentHash: 0xccc, ModifiedTime: "2026-01-01T00:00:00Z"},
			}),
			wantAdded:     nil,
			wantModified:  nil,
			wantDeleted:   nil,
			wantUnchanged: 3,
		},
		{
			name: "mixed scenario",
			previous: snapshotWithFiles(t, map[string]FileState{
				// unchanged (4 files)
				"keep1.go": {Size: 100, ContentHash: 0x1, ModifiedTime: "2026-01-01T00:00:00Z"},
				"keep2.go": {Size: 200, ContentHash: 0x2, ModifiedTime: "2026-01-01T00:00:00Z"},
				"keep3.go": {Size: 300, ContentHash: 0x3, ModifiedTime: "2026-01-01T00:00:00Z"},
				"keep4.go": {Size: 400, ContentHash: 0x4, ModifiedTime: "2026-01-01T00:00:00Z"},
				// modified (3 files -- hash will differ in current)
				"mod1.go": {Size: 500, ContentHash: 0x50, ModifiedTime: "2026-01-01T00:00:00Z"},
				"mod2.go": {Size: 600, ContentHash: 0x60, ModifiedTime: "2026-01-01T00:00:00Z"},
				"mod3.go": {Size: 700, ContentHash: 0x70, ModifiedTime: "2026-01-01T00:00:00Z"},
				// deleted (1 file -- absent from current)
				"removed.go": {Size: 800, ContentHash: 0x80, ModifiedTime: "2026-01-01T00:00:00Z"},
			}),
			current: snapshotWithFiles(t, map[string]FileState{
				// unchanged (4 files -- same hash)
				"keep1.go": {Size: 100, ContentHash: 0x1, ModifiedTime: "2026-01-01T00:00:00Z"},
				"keep2.go": {Size: 200, ContentHash: 0x2, ModifiedTime: "2026-01-01T00:00:00Z"},
				"keep3.go": {Size: 300, ContentHash: 0x3, ModifiedTime: "2026-01-01T00:00:00Z"},
				"keep4.go": {Size: 400, ContentHash: 0x4, ModifiedTime: "2026-01-01T00:00:00Z"},
				// modified (3 files -- different hash)
				"mod1.go": {Size: 550, ContentHash: 0x51, ModifiedTime: "2026-01-02T00:00:00Z"},
				"mod2.go": {Size: 650, ContentHash: 0x61, ModifiedTime: "2026-01-02T00:00:00Z"},
				"mod3.go": {Size: 750, ContentHash: 0x71, ModifiedTime: "2026-01-02T00:00:00Z"},
				// added (2 files -- absent from previous)
				"new1.go": {Size: 900, ContentHash: 0x90, ModifiedTime: "2026-01-02T00:00:00Z"},
				"new2.go": {Size: 950, ContentHash: 0x95, ModifiedTime: "2026-01-02T00:00:00Z"},
			}),
			wantAdded:     []string{"new1.go", "new2.go"},
			wantModified:  []string{"mod1.go", "mod2.go", "mod3.go"},
			wantDeleted:   []string{"removed.go"},
			wantUnchanged: 4,
		},
		{
			name:     "sorted output",
			previous: newEmptySnapshot(t),
			current: snapshotWithFiles(t, map[string]FileState{
				"z.go": {Size: 10, ContentHash: 0x1, ModifiedTime: "2026-01-01T00:00:00Z"},
				"a.go": {Size: 20, ContentHash: 0x2, ModifiedTime: "2026-01-01T00:00:00Z"},
				"m.go": {Size: 30, ContentHash: 0x3, ModifiedTime: "2026-01-01T00:00:00Z"},
			}),
			wantAdded:     []string{"a.go", "m.go", "z.go"},
			wantModified:  nil,
			wantDeleted:   nil,
			wantUnchanged: 0,
		},
		{
			name:     "nil previous",
			previous: nil,
			current: snapshotWithFiles(t, map[string]FileState{
				"file1.go": {Size: 100, ContentHash: 0x1, ModifiedTime: "2026-01-01T00:00:00Z"},
				"file2.go": {Size: 200, ContentHash: 0x2, ModifiedTime: "2026-01-01T00:00:00Z"},
				"file3.go": {Size: 300, ContentHash: 0x3, ModifiedTime: "2026-01-01T00:00:00Z"},
			}),
			wantAdded:     []string{"file1.go", "file2.go", "file3.go"},
			wantModified:  nil,
			wantDeleted:   nil,
			wantUnchanged: 0,
		},
		{
			name: "nil current",
			previous: snapshotWithFiles(t, map[string]FileState{
				"file1.go": {Size: 100, ContentHash: 0x1, ModifiedTime: "2026-01-01T00:00:00Z"},
				"file2.go": {Size: 200, ContentHash: 0x2, ModifiedTime: "2026-01-01T00:00:00Z"},
				"file3.go": {Size: 300, ContentHash: 0x3, ModifiedTime: "2026-01-01T00:00:00Z"},
			}),
			current:       nil,
			wantAdded:     nil,
			wantModified:  nil,
			wantDeleted:   []string{"file1.go", "file2.go", "file3.go"},
			wantUnchanged: 0,
		},
		{
			name: "same path different hash",
			previous: snapshotWithFiles(t, map[string]FileState{
				"config.toml": {Size: 100, ContentHash: 0xold, ModifiedTime: "2026-01-01T00:00:00Z"},
			}),
			current: snapshotWithFiles(t, map[string]FileState{
				"config.toml": {Size: 150, ContentHash: 0xfeed, ModifiedTime: "2026-01-02T00:00:00Z"},
			}),
			wantAdded:     nil,
			wantModified:  []string{"config.toml"},
			wantDeleted:   nil,
			wantUnchanged: 0,
		},
		{
			name: "same path same hash",
			previous: snapshotWithFiles(t, map[string]FileState{
				"config.toml": {Size: 100, ContentHash: 0xbeef, ModifiedTime: "2026-01-01T00:00:00Z"},
			}),
			current: snapshotWithFiles(t, map[string]FileState{
				"config.toml": {Size: 100, ContentHash: 0xbeef, ModifiedTime: "2026-01-01T00:00:00Z"},
			}),
			wantAdded:     nil,
			wantModified:  nil,
			wantDeleted:   nil,
			wantUnchanged: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CompareStates(tt.previous, tt.current)
			require.NotNil(t, result)

			assert.Equal(t, tt.wantAdded, result.Added, "Added mismatch")
			assert.Equal(t, tt.wantModified, result.Modified, "Modified mismatch")
			assert.Equal(t, tt.wantDeleted, result.Deleted, "Deleted mismatch")
			assert.Equal(t, tt.wantUnchanged, result.Unchanged, "Unchanged mismatch")
		})
	}
}

func TestDiffResult_HasChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		diff DiffResult
		want bool
	}{
		{
			name: "only added",
			diff: DiffResult{
				Added: []string{"new.go"},
			},
			want: true,
		},
		{
			name: "only modified",
			diff: DiffResult{
				Modified: []string{"changed.go"},
			},
			want: true,
		},
		{
			name: "only deleted",
			diff: DiffResult{
				Deleted: []string{"removed.go"},
			},
			want: true,
		},
		{
			name: "all empty",
			diff: DiffResult{
				Unchanged: 10,
			},
			want: false,
		},
		{
			name: "all populated",
			diff: DiffResult{
				Added:     []string{"a.go"},
				Modified:  []string{"b.go"},
				Deleted:   []string{"c.go"},
				Unchanged: 5,
			},
			want: true,
		},
		{
			name: "zero value",
			diff: DiffResult{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.diff.HasChanges())
		})
	}
}

func TestDiffResult_TotalChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		diff DiffResult
		want int
	}{
		{
			name: "empty result",
			diff: DiffResult{},
			want: 0,
		},
		{
			name: "only added",
			diff: DiffResult{
				Added: []string{"a.go", "b.go", "c.go"},
			},
			want: 3,
		},
		{
			name: "mixed",
			diff: DiffResult{
				Added:    []string{"a.go", "b.go"},
				Modified: []string{"c.go", "d.go", "e.go"},
				Deleted:  []string{"f.go"},
			},
			want: 6,
		},
		{
			name: "unchanged does not count",
			diff: DiffResult{
				Added:     []string{"a.go"},
				Unchanged: 100,
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.diff.TotalChanged())
		})
	}
}

func TestDiffResult_Summary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		diff DiffResult
		want string
	}{
		{
			name: "all zeros",
			diff: DiffResult{},
			want: "0 added, 0 modified, 0 deleted (0 unchanged)",
		},
		{
			name: "mixed",
			diff: DiffResult{
				Added:     []string{"a.go", "b.go"},
				Modified:  []string{"c.go", "d.go", "e.go"},
				Deleted:   []string{"f.go"},
				Unchanged: 4,
			},
			want: "2 added, 3 modified, 1 deleted (4 unchanged)",
		},
		{
			name: "only added",
			diff: DiffResult{
				Added: []string{"a.go", "b.go", "c.go", "d.go", "e.go"},
			},
			want: "5 added, 0 modified, 0 deleted (0 unchanged)",
		},
		{
			name: "only unchanged",
			diff: DiffResult{
				Unchanged: 42,
			},
			want: "0 added, 0 modified, 0 deleted (42 unchanged)",
		},
		{
			name: "only deleted",
			diff: DiffResult{
				Deleted: []string{"old.go"},
			},
			want: "0 added, 0 modified, 1 deleted (0 unchanged)",
		},
		{
			name: "only modified",
			diff: DiffResult{
				Modified: []string{"x.go", "y.go"},
			},
			want: "0 added, 2 modified, 0 deleted (0 unchanged)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.diff.Summary())
		})
	}
}

func BenchmarkCompareStates_10000Files(b *testing.B) {
	const totalFiles = 10000
	const numModified = 50
	const numAdded = 25
	const numDeleted = 25

	// Build previous snapshot with totalFiles files.
	previous := buildSnapshot(b, totalFiles)

	// Build current snapshot: start with the same files, then apply changes.
	current := NewStateSnapshot("test", "/tmp", "main", "abc123")

	// Copy all files from previous, except the last numDeleted (those are deleted).
	for i := 0; i < totalFiles-numDeleted; i++ {
		path := fmt.Sprintf("file_%05d.go", i)
		fs := previous.Files[path]

		// Modify the first numModified files by changing their hash.
		if i < numModified {
			fs.ContentHash = uint64(i + totalFiles)
		}

		current.AddFile(path, fs)
	}

	// Add numAdded new files.
	for i := 0; i < numAdded; i++ {
		current.AddFile(fmt.Sprintf("new_%05d.go", i), FileState{
			Size:         int64(i * 100),
			ContentHash:  uint64(i + totalFiles*2),
			ModifiedTime: "2026-01-02T00:00:00Z",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := CompareStates(previous, current)
		// Prevent the compiler from optimizing away the call.
		if result == nil {
			b.Fatal("unexpected nil result")
		}
	}
}

// newEmptySnapshot creates a StateSnapshot with no files.
func newEmptySnapshot(t testing.TB) *StateSnapshot {
	t.Helper()
	return NewStateSnapshot("test", "/tmp", "main", "abc123")
}

// snapshotWithFiles creates a StateSnapshot pre-populated with the given files.
func snapshotWithFiles(t testing.TB, files map[string]FileState) *StateSnapshot {
	t.Helper()
	snap := NewStateSnapshot("test", "/tmp", "main", "abc123")
	for path, fs := range files {
		snap.AddFile(path, fs)
	}
	return snap
}
