package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitignoreMatcher_InvalidRoot(t *testing.T) {
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
			_, err := NewGitignoreMatcher(tt.root)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestNewGitignoreMatcher_NoGitignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644))

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)
	assert.Equal(t, 0, m.PatternCount())
	assert.False(t, m.IsIgnored("file.txt", false))
	assert.False(t, m.IsIgnored("anything/at/all.go", false))
}

func TestNewGitignoreMatcher_EmptyGitignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644))

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, m.PatternCount())
	assert.False(t, m.IsIgnored("file.txt", false))
}

func TestGitignoreMatcher_BasicPatterns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGitignore(t, dir, "*.log\n*.tmp\n.env\n")

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		{name: "matches wildcard .log", path: "error.log", isDir: false, expect: true},
		{name: "matches wildcard .tmp", path: "cache.tmp", isDir: false, expect: true},
		{name: "matches dotenv", path: ".env", isDir: false, expect: true},
		{name: "does not match .go file", path: "main.go", isDir: false, expect: false},
		{name: "does not match .md file", path: "README.md", isDir: false, expect: false},
		{name: "matches nested .log", path: "src/app.log", isDir: false, expect: true},
		{name: "matches deep nested .tmp", path: "a/b/c/d.tmp", isDir: false, expect: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestGitignoreMatcher_DirectoryPatterns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGitignore(t, dir, "build/\nnode_modules/\n")

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		{name: "build dir ignored", path: "build", isDir: true, expect: true},
		{name: "node_modules dir ignored", path: "node_modules", isDir: true, expect: true},
		{name: "file inside build dir", path: "build/output.js", isDir: false, expect: true},
		{name: "nested build dir", path: "src/build", isDir: true, expect: true},
		// A file named "build" (not a directory) should NOT match a directory-only pattern.
		// However, sabhiram/go-gitignore does match "build/" against files too when using MatchesPath.
		// We test the directory case explicitly.
		{name: "file in node_modules", path: "node_modules/express/index.js", isDir: false, expect: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestGitignoreMatcher_NegationPatterns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGitignore(t, dir, "*.log\n!important.log\n")

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		{name: "regular .log is ignored", path: "error.log", isDir: false, expect: true},
		{name: "debug.log is ignored", path: "debug.log", isDir: false, expect: true},
		{name: "important.log is NOT ignored (negated)", path: "important.log", isDir: false, expect: false},
		{name: "non-log file is not ignored", path: "main.go", isDir: false, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestGitignoreMatcher_DoublestarPatterns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGitignore(t, dir, "**/*.tmp\n**/test/**/*.snap\n")

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		{name: "root level .tmp", path: "file.tmp", isDir: false, expect: true},
		{name: "nested .tmp", path: "src/file.tmp", isDir: false, expect: true},
		{name: "deep nested .tmp", path: "a/b/c/file.tmp", isDir: false, expect: true},
		{name: "snap in test dir", path: "src/test/unit/output.snap", isDir: false, expect: true},
		{name: "non-tmp file", path: "src/main.go", isDir: false, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestGitignoreMatcher_NestedGitignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Root .gitignore: ignore *.log globally.
	writeGitignore(t, dir, "*.log\n")

	// Nested src/.gitignore: ignore *.generated.go only under src/.
	srcDir := filepath.Join(dir, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	writeGitignore(t, srcDir, "*.generated.go\n")

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, m.PatternCount())

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		// Root patterns apply everywhere.
		{name: "root log at root", path: "app.log", isDir: false, expect: true},
		{name: "root log in src", path: "src/app.log", isDir: false, expect: true},
		{name: "root log in lib", path: "lib/debug.log", isDir: false, expect: true},

		// Nested patterns apply only under src/.
		{name: "generated.go in src", path: "src/types.generated.go", isDir: false, expect: true},
		{name: "generated.go in src subdir", path: "src/models/schema.generated.go", isDir: false, expect: true},

		// Nested patterns do NOT apply outside src/.
		{name: "generated.go at root", path: "types.generated.go", isDir: false, expect: false},
		{name: "generated.go in lib", path: "lib/types.generated.go", isDir: false, expect: false},

		// Normal files are not ignored.
		{name: "normal go file in src", path: "src/main.go", isDir: false, expect: false},
		{name: "normal file at root", path: "README.md", isDir: false, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestGitignoreMatcher_DeeplyNestedGitignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Root: ignore *.log.
	writeGitignore(t, dir, "*.log\n")

	// a/b/.gitignore: ignore *.dat.
	abDir := filepath.Join(dir, "a", "b")
	require.NoError(t, os.MkdirAll(abDir, 0755))
	writeGitignore(t, abDir, "*.dat\n")

	// Create a/b/c directory for testing.
	require.NoError(t, os.MkdirAll(filepath.Join(abDir, "c"), 0755))

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, m.PatternCount())

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		// Root patterns inherited everywhere.
		{name: "log at root", path: "app.log", isDir: false, expect: true},
		{name: "log in a/b", path: "a/b/error.log", isDir: false, expect: true},
		{name: "log in a/b/c", path: "a/b/c/deep.log", isDir: false, expect: true},

		// a/b patterns apply under a/b and its children.
		{name: "dat in a/b", path: "a/b/data.dat", isDir: false, expect: true},
		{name: "dat in a/b/c", path: "a/b/c/data.dat", isDir: false, expect: true},

		// a/b patterns do NOT apply outside a/b.
		{name: "dat at root", path: "data.dat", isDir: false, expect: false},
		{name: "dat in a", path: "a/data.dat", isDir: false, expect: false},

		// Normal files pass through.
		{name: "txt in a/b/c", path: "a/b/c/file.txt", isDir: false, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestGitignoreMatcher_CommentsAndBlankLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGitignore(t, dir, "# This is a comment\n\n# Another comment\n\n*.secret\n\n# More comments\n*.cache\n")

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		{name: "secret file matched", path: "passwords.secret", isDir: false, expect: true},
		{name: "cache file matched", path: "data.cache", isDir: false, expect: true},
		{name: "normal file not matched", path: "main.go", isDir: false, expect: false},
		// Comments and blank lines should not create patterns.
		{name: "file starting with hash not matched", path: "#readme", isDir: false, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestGitignoreMatcher_EmptyPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGitignore(t, dir, "*.log\n")

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)

	assert.False(t, m.IsIgnored("", false), "empty path should not be ignored")
	assert.False(t, m.IsIgnored(".", false), "dot path should not be ignored")
	assert.False(t, m.IsIgnored("./", true), "dot-slash path should not be ignored")
}

func TestGitignoreMatcher_PathWithLeadingDotSlash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGitignore(t, dir, "*.log\n")

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)

	// Paths with "./" prefix should be normalized.
	assert.True(t, m.IsIgnored("./error.log", false))
	assert.True(t, m.IsIgnored("./src/app.log", false))
	assert.False(t, m.IsIgnored("./main.go", false))
}

func TestGitignoreMatcher_PatternCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T) string
		want  int
	}{
		{
			name: "no gitignore files",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0644))
				return dir
			},
			want: 0,
		},
		{
			name: "one root gitignore",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeGitignore(t, dir, "*.log\n")
				return dir
			},
			want: 1,
		},
		{
			name: "multiple nested gitignores",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeGitignore(t, dir, "*.log\n")
				subDir := filepath.Join(dir, "sub")
				require.NoError(t, os.MkdirAll(subDir, 0755))
				writeGitignore(t, subDir, "*.tmp\n")
				deepDir := filepath.Join(dir, "a", "b")
				require.NoError(t, os.MkdirAll(deepDir, 0755))
				writeGitignore(t, deepDir, "*.dat\n")
				return dir
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)
			m, err := NewGitignoreMatcher(dir)
			require.NoError(t, err)
			assert.Equal(t, tt.want, m.PatternCount())
		})
	}
}

func TestGitignoreMatcher_SkipsGitDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGitignore(t, dir, "*.log\n")

	// Create a .git directory with a .gitignore inside it (should be skipped).
	gitDir := filepath.Join(dir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	writeGitignore(t, gitDir, "*.everything\n")

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)

	// Only the root .gitignore should be loaded; .git/.gitignore is skipped.
	assert.Equal(t, 1, m.PatternCount())
}

func TestGitignoreMatcher_ParentRulesInherited(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeGitignore(t, dir, "*.log\ntemp/\n")

	// Create subdirectories without their own .gitignore.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src", "pkg", "internal"), 0755))

	m, err := NewGitignoreMatcher(dir)
	require.NoError(t, err)

	// Root rules should apply at every level.
	assert.True(t, m.IsIgnored("error.log", false))
	assert.True(t, m.IsIgnored("src/error.log", false))
	assert.True(t, m.IsIgnored("src/pkg/error.log", false))
	assert.True(t, m.IsIgnored("src/pkg/internal/error.log", false))
	assert.True(t, m.IsIgnored("temp", true))
	assert.True(t, m.IsIgnored("src/temp", true))
}

func TestGitignoreMatcher_FixtureRoot(t *testing.T) {
	t.Parallel()

	// Use the testdata/gitignore/root fixture.
	fixtureDir := filepath.Join(findProjectRoot(t), "testdata", "gitignore", "root")

	m, err := NewGitignoreMatcher(fixtureDir)
	require.NoError(t, err)
	assert.Equal(t, 2, m.PatternCount(), "root + src/.gitignore")

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		// Root patterns.
		{name: "log file at root", path: "error.log", isDir: false, expect: true},
		{name: "tmp file", path: "cache.tmp", isDir: false, expect: true},
		{name: "node_modules dir", path: "node_modules", isDir: true, expect: true},
		{name: "build dir", path: "build", isDir: true, expect: true},
		{name: "dotenv", path: ".env", isDir: false, expect: true},
		{name: "bak file deep", path: "deep/nested/file.bak", isDir: false, expect: true},
		{name: "readme not ignored", path: "README.md", isDir: false, expect: false},

		// Nested src/ patterns.
		{name: "generated.go in src", path: "src/types.generated.go", isDir: false, expect: true},
		{name: "vendor dir in src", path: "src/vendor", isDir: true, expect: true},
		{name: "generated.go at root not ignored by src rule", path: "types.generated.go", isDir: false, expect: false},
		{name: "normal go file in src", path: "src/main.go", isDir: false, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestGitignoreMatcher_FixtureNegation(t *testing.T) {
	t.Parallel()

	fixtureDir := filepath.Join(findProjectRoot(t), "testdata", "gitignore", "negation")

	m, err := NewGitignoreMatcher(fixtureDir)
	require.NoError(t, err)

	assert.True(t, m.IsIgnored("error.log", false))
	assert.True(t, m.IsIgnored("debug.log", false))
	assert.False(t, m.IsIgnored("important.log", false), "negation should override")
	assert.False(t, m.IsIgnored("main.go", false))
}

func TestGitignoreMatcher_FixtureComments(t *testing.T) {
	t.Parallel()

	fixtureDir := filepath.Join(findProjectRoot(t), "testdata", "gitignore", "comments")

	m, err := NewGitignoreMatcher(fixtureDir)
	require.NoError(t, err)

	assert.True(t, m.IsIgnored("passwords.secret", false))
	assert.True(t, m.IsIgnored("data.cache", false))
	assert.False(t, m.IsIgnored("README.md", false))
}

func TestGitignoreMatcher_FixtureDeep(t *testing.T) {
	t.Parallel()

	fixtureDir := filepath.Join(findProjectRoot(t), "testdata", "gitignore", "deep")

	m, err := NewGitignoreMatcher(fixtureDir)
	require.NoError(t, err)
	assert.Equal(t, 2, m.PatternCount())

	// Root *.log applies everywhere.
	assert.True(t, m.IsIgnored("app.log", false))
	assert.True(t, m.IsIgnored("a/b/c/deep.log", false))

	// a/b *.dat applies under a/b only.
	assert.True(t, m.IsIgnored("a/b/data.dat", false))
	assert.True(t, m.IsIgnored("a/b/c/data.dat", false))
	assert.False(t, m.IsIgnored("a/data.dat", false))
	assert.False(t, m.IsIgnored("data.dat", false))
}

func TestGitignoreMatcher_FixtureEmpty(t *testing.T) {
	t.Parallel()

	fixtureDir := filepath.Join(findProjectRoot(t), "testdata", "gitignore", "empty")

	m, err := NewGitignoreMatcher(fixtureDir)
	require.NoError(t, err)
	assert.Equal(t, 0, m.PatternCount())
	assert.False(t, m.IsIgnored("file.txt", false))
	assert.False(t, m.IsIgnored("anything", false))
}

func BenchmarkGitignoreMatcher_IsIgnored(b *testing.B) {
	dir := b.TempDir()

	// Create a .gitignore with many patterns.
	var patterns string
	for i := 0; i < 50; i++ {
		patterns += "*.ext" + filepath.Ext("."+string(rune('a'+i%26))) + "\n"
	}
	patterns += "node_modules/\n.env\n**/*.log\n**/*.tmp\nbuild/\ndist/\n"

	require.NoError(b, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(patterns), 0644))

	m, err := NewGitignoreMatcher(dir)
	require.NoError(b, err)

	paths := make([]string, 10000)
	for i := range paths {
		paths[i] = filepath.Join("src", "pkg", "internal", "file_"+string(rune('a'+i%26))+".go")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range paths {
			m.IsIgnored(p, false)
		}
	}
}

// --- Test helpers ---

// createTempFile creates a temporary file and returns its path. The file is
// cleaned up when the test completes.
func createTempFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-file-*")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

// writeGitignore writes a .gitignore file in the given directory with the
// specified content.
func writeGitignore(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(content), 0644))
}

// findProjectRoot walks up from the current working directory to find the
// project root (the directory containing go.mod).
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}
