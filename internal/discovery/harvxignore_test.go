package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHarvxignoreMatcher_InvalidRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		root    string
		wantErr string
	}{
		{
			name:    "nonexistent directory",
			root:    "/nonexistent/path/that/does/not/exist",
			wantErr: "stat root path",
		},
		{
			name:    "file instead of directory",
			root:    createTempFile(t),
			wantErr: "is not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewHarvxignoreMatcher(tt.root)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestNewHarvxignoreMatcher_NoHarvxignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644))

	m, err := NewHarvxignoreMatcher(dir)
	require.NoError(t, err)
	assert.Equal(t, 0, m.PatternCount())
	assert.False(t, m.IsIgnored("file.txt", false))
	assert.False(t, m.IsIgnored("anything/at/all.go", false))
}

func TestNewHarvxignoreMatcher_EmptyHarvxignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".harvxignore"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644))

	m, err := NewHarvxignoreMatcher(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, m.PatternCount())
	assert.False(t, m.IsIgnored("file.txt", false))
}

func TestHarvxignoreMatcher_BasicPatterns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeHarvxignore(t, dir, "*.draft.md\nscratch/\n*.wip\n")

	m, err := NewHarvxignoreMatcher(dir)
	require.NoError(t, err)

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		{name: "matches draft.md", path: "design.draft.md", isDir: false, expect: true},
		{name: "matches scratch dir", path: "scratch", isDir: true, expect: true},
		{name: "matches wip file", path: "feature.wip", isDir: false, expect: true},
		{name: "file in scratch", path: "scratch/notes.txt", isDir: false, expect: true},
		{name: "normal md not matched", path: "README.md", isDir: false, expect: false},
		{name: "normal go not matched", path: "main.go", isDir: false, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestHarvxignoreMatcher_NegationPatterns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeHarvxignore(t, dir, "*.log\n!important.log\n")

	m, err := NewHarvxignoreMatcher(dir)
	require.NoError(t, err)

	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{name: "regular log ignored", path: "error.log", expect: true},
		{name: "debug.log ignored", path: "debug.log", expect: true},
		{name: "important.log NOT ignored (negated)", path: "important.log", expect: false},
		{name: "non-log not ignored", path: "main.go", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, false)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, false)", tt.path)
		})
	}
}

func TestHarvxignoreMatcher_DirectoryPatterns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeHarvxignore(t, dir, "docs/internal/\ntmp/\n")

	m, err := NewHarvxignoreMatcher(dir)
	require.NoError(t, err)

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		{name: "docs/internal dir", path: "docs/internal", isDir: true, expect: true},
		{name: "file in docs/internal", path: "docs/internal/spec.md", isDir: false, expect: true},
		{name: "tmp dir", path: "tmp", isDir: true, expect: true},
		{name: "docs dir not ignored", path: "docs", isDir: true, expect: false},
		{name: "docs/public not ignored", path: "docs/public", isDir: true, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestHarvxignoreMatcher_NestedHarvxignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Root .harvxignore: ignore *.draft.md globally.
	writeHarvxignore(t, dir, "*.draft.md\n")

	// Nested src/.harvxignore: ignore *.generated.ts only under src/.
	srcDir := filepath.Join(dir, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	writeHarvxignore(t, srcDir, "*.generated.ts\n")

	m, err := NewHarvxignoreMatcher(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, m.PatternCount())

	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		// Root patterns apply everywhere.
		{name: "draft.md at root", path: "design.draft.md", expect: true},
		{name: "draft.md in src", path: "src/spec.draft.md", expect: true},

		// Nested patterns apply only under src/.
		{name: "generated.ts in src", path: "src/types.generated.ts", expect: true},
		{name: "generated.ts at root NOT ignored", path: "types.generated.ts", expect: false},

		// Normal files pass through.
		{name: "normal ts in src", path: "src/main.ts", expect: false},
		{name: "normal file at root", path: "README.md", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, false)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, false)", tt.path)
		})
	}
}

func TestHarvxignoreMatcher_EmptyPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeHarvxignore(t, dir, "*.log\n")

	m, err := NewHarvxignoreMatcher(dir)
	require.NoError(t, err)

	assert.False(t, m.IsIgnored("", false), "empty path should not be ignored")
	assert.False(t, m.IsIgnored(".", false), "dot path should not be ignored")
	assert.False(t, m.IsIgnored("./", true), "dot-slash path should not be ignored")
}

func TestHarvxignoreMatcher_LeadingDotSlash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeHarvxignore(t, dir, "*.log\n")

	m, err := NewHarvxignoreMatcher(dir)
	require.NoError(t, err)

	assert.True(t, m.IsIgnored("./error.log", false))
	assert.True(t, m.IsIgnored("./src/app.log", false))
	assert.False(t, m.IsIgnored("./main.go", false))
}

func TestHarvxignoreMatcher_FixtureBasic(t *testing.T) {
	t.Parallel()

	fixtureDir := filepath.Join(findProjectRoot(t), "testdata", "harvxignore", "basic")

	m, err := NewHarvxignoreMatcher(fixtureDir)
	require.NoError(t, err)
	assert.Equal(t, 1, m.PatternCount())

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		{name: "draft.md matched", path: "design.draft.md", isDir: false, expect: true},
		{name: "scratch dir matched", path: "scratch", isDir: true, expect: true},
		{name: "wip file matched", path: "feature.wip", isDir: false, expect: true},
		{name: "docs/internal dir matched", path: "docs/internal", isDir: true, expect: true},
		{name: "file in docs/internal", path: "docs/internal/spec.md", isDir: false, expect: true},
		{name: "normal file not matched", path: "main.go", isDir: false, expect: false},
		{name: "normal md not matched", path: "README.md", isDir: false, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestHarvxignoreMatcher_FixtureNegation(t *testing.T) {
	t.Parallel()

	fixtureDir := filepath.Join(findProjectRoot(t), "testdata", "harvxignore", "negation")

	m, err := NewHarvxignoreMatcher(fixtureDir)
	require.NoError(t, err)

	assert.True(t, m.IsIgnored("error.log", false))
	assert.True(t, m.IsIgnored("debug.log", false))
	assert.False(t, m.IsIgnored("important.log", false), "negation should override")
	assert.True(t, m.IsIgnored("temp", true))
	assert.False(t, m.IsIgnored("main.go", false))
}

func TestHarvxignoreMatcher_FixtureEmpty(t *testing.T) {
	t.Parallel()

	fixtureDir := filepath.Join(findProjectRoot(t), "testdata", "harvxignore", "empty")

	m, err := NewHarvxignoreMatcher(fixtureDir)
	require.NoError(t, err)
	assert.Equal(t, 0, m.PatternCount())
	assert.False(t, m.IsIgnored("file.txt", false))
	assert.False(t, m.IsIgnored("anything", false))
}

func TestHarvxignoreMatcher_PatternCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T) string
		want  int
	}{
		{
			name: "no harvxignore files",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0644))
				return dir
			},
			want: 0,
		},
		{
			name: "one root harvxignore",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeHarvxignore(t, dir, "*.log\n")
				return dir
			},
			want: 1,
		},
		{
			name: "multiple nested harvxignores",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeHarvxignore(t, dir, "*.log\n")
				subDir := filepath.Join(dir, "sub")
				require.NoError(t, os.MkdirAll(subDir, 0755))
				writeHarvxignore(t, subDir, "*.tmp\n")
				deepDir := filepath.Join(dir, "a", "b")
				require.NoError(t, os.MkdirAll(deepDir, 0755))
				writeHarvxignore(t, deepDir, "*.dat\n")
				return dir
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)
			m, err := NewHarvxignoreMatcher(dir)
			require.NoError(t, err)
			assert.Equal(t, tt.want, m.PatternCount())
		})
	}
}

func TestHarvxignoreMatcher_ImplementsIgnorer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := NewHarvxignoreMatcher(dir)
	require.NoError(t, err)

	var ig Ignorer = m
	assert.NotNil(t, ig)
	assert.False(t, ig.IsIgnored("test.go", false))
}

func BenchmarkHarvxignoreMatcher_IsIgnored(b *testing.B) {
	dir := b.TempDir()

	var patterns string
	patterns += "*.draft.md\nscratch/\n*.wip\ndocs/internal/\n"
	patterns += "*.generated.ts\n*.generated.go\n"
	patterns += "temp/\n*.bak\n**/*.snap\n"

	require.NoError(b, os.WriteFile(filepath.Join(dir, ".harvxignore"), []byte(patterns), 0644))

	m, err := NewHarvxignoreMatcher(dir)
	require.NoError(b, err)

	paths := []string{
		"main.go",
		"design.draft.md",
		"src/app.ts",
		"scratch/notes.txt",
		"feature.wip",
		"README.md",
		"internal/config/config.go",
		"docs/internal/spec.md",
		"temp/cache.dat",
		"src/types.generated.ts",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range paths {
			m.IsIgnored(p, false)
		}
	}
}

// --- Test helper ---

// writeHarvxignore writes a .harvxignore file in the given directory with the
// specified content.
func writeHarvxignore(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".harvxignore"), []byte(content), 0644))
}
