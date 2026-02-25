package workflows

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestDiscoverNeighbors
// ---------------------------------------------------------------------------

func TestDiscoverNeighbors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		opts             NeighborOptions
		wantAllNeighbors []string
		wantTestFiles    []string
		wantImporters    []string
	}{
		{
			name: "depth 0 returns empty result",
			opts: NeighborOptions{
				RootDir:      t.TempDir(),
				ChangedFiles: []string{"src/main.go"},
				AllFiles:     []string{"src/main.go", "src/main_test.go"},
				Depth:        0,
			},
			wantAllNeighbors: nil,
			wantTestFiles:    nil,
			wantImporters:    nil,
		},
		{
			name: "depth 1 with Go test files finds _test.go for changed .go file",
			opts: NeighborOptions{
				RootDir:      t.TempDir(),
				ChangedFiles: []string{"src/main.go"},
				AllFiles:     []string{"src/main.go", "src/main_test.go", "src/util.go"},
				Depth:        1,
			},
			wantTestFiles:    []string{"src/main_test.go"},
			wantAllNeighbors: []string{"src/main_test.go"},
		},
		{
			name: "depth 1 with TypeScript test files finds .test.ts for changed .ts file",
			opts: NeighborOptions{
				RootDir:      t.TempDir(),
				ChangedFiles: []string{"src/app.ts"},
				AllFiles:     []string{"src/app.ts", "src/app.test.ts", "src/app.spec.ts"},
				Depth:        1,
			},
			wantTestFiles:    []string{"src/app.spec.ts", "src/app.test.ts"},
			wantAllNeighbors: []string{"src/app.spec.ts", "src/app.test.ts"},
		},
		{
			name: "changed files excluded from AllNeighbors",
			opts: NeighborOptions{
				RootDir:      t.TempDir(),
				ChangedFiles: []string{"src/main.go", "src/main_test.go"},
				AllFiles:     []string{"src/main.go", "src/main_test.go", "src/util.go"},
				Depth:        1,
			},
			// main_test.go is a changed file, so it should NOT appear in neighbors
			wantAllNeighbors: nil,
			wantTestFiles:    nil,
		},
		{
			name: "empty changed files returns empty result",
			opts: NeighborOptions{
				RootDir:      t.TempDir(),
				ChangedFiles: []string{},
				AllFiles:     []string{"src/main.go", "src/main_test.go"},
				Depth:        1,
			},
			wantAllNeighbors: nil,
			wantTestFiles:    nil,
			wantImporters:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := DiscoverNeighbors(tt.opts)
			require.NotNil(t, result)

			if tt.wantAllNeighbors == nil {
				assert.Empty(t, result.AllNeighbors)
			} else {
				assert.Equal(t, tt.wantAllNeighbors, result.AllNeighbors)
			}

			if tt.wantTestFiles == nil {
				assert.Empty(t, result.TestFiles)
			} else {
				assert.Equal(t, tt.wantTestFiles, result.TestFiles)
			}

			if tt.wantImporters == nil {
				assert.Empty(t, result.ImporterFiles)
			} else {
				assert.Equal(t, tt.wantImporters, result.ImporterFiles)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestFindTestCandidates
// ---------------------------------------------------------------------------

func TestFindTestCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		want     []string
	}{
		{
			name:     "Go file generates _test.go",
			filePath: "pkg/handler.go",
			want:     []string{filepath.Clean("pkg/handler_test.go")},
		},
		{
			name:     "TypeScript file generates .test.ts, .spec.ts, __tests__/",
			filePath: "src/app.ts",
			want: []string{
				filepath.Clean("src/app.test.ts"),
				filepath.Clean("src/app.spec.ts"),
				filepath.Clean("src/__tests__/app.ts"),
			},
		},
		{
			name:     "TypeScript tsx file generates .test.tsx, .spec.tsx, __tests__/",
			filePath: "src/Button.tsx",
			want: []string{
				filepath.Clean("src/Button.test.tsx"),
				filepath.Clean("src/Button.spec.tsx"),
				filepath.Clean("src/__tests__/Button.tsx"),
			},
		},
		{
			name:     "JavaScript file generates .test.js, .spec.js, __tests__/",
			filePath: "lib/util.js",
			want: []string{
				filepath.Clean("lib/util.test.js"),
				filepath.Clean("lib/util.spec.js"),
				filepath.Clean("lib/__tests__/util.js"),
			},
		},
		{
			name:     "JavaScript jsx file generates .test.jsx, .spec.jsx, __tests__/",
			filePath: "components/Card.jsx",
			want: []string{
				filepath.Clean("components/Card.test.jsx"),
				filepath.Clean("components/Card.spec.jsx"),
				filepath.Clean("components/__tests__/Card.jsx"),
			},
		},
		{
			name:     "Python file generates test_foo.py, foo_test.py, tests/test_foo.py",
			filePath: "mymod/foo.py",
			want: []string{
				filepath.Clean("mymod/test_foo.py"),
				filepath.Clean("mymod/foo_test.py"),
				filepath.Clean("mymod/tests/test_foo.py"),
			},
		},
		{
			name:     "unknown extension generates generic patterns",
			filePath: "config/settings.rb",
			want: []string{
				filepath.Clean("config/settings_test.rb"),
				filepath.Clean("config/settings.test.rb"),
				filepath.Clean("config/settings.spec.rb"),
			},
		},
		{
			name:     "file at root directory",
			filePath: "main.go",
			want:     []string{filepath.Clean("main_test.go")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := findTestCandidates(tt.filePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolveImportToPath
// ---------------------------------------------------------------------------

func TestResolveImportToPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		importPath  string
		importerDir string
		rootDir     string
		want        string
	}{
		{
			name:        "empty path returns empty",
			importPath:  "",
			importerDir: "src",
			rootDir:     "/repo",
			want:        "",
		},
		{
			name:        "relative import ./foo resolves against importer dir",
			importPath:  "./foo",
			importerDir: "src/pkg",
			rootDir:     "/repo",
			want:        filepath.Clean("src/pkg/foo"),
		},
		{
			name:        "relative import ../bar resolves against importer dir",
			importPath:  "../bar",
			importerDir: "src/pkg",
			rootDir:     "/repo",
			want:        filepath.Clean("src/bar"),
		},
		{
			name:        "relative import ../sibling/baz resolves correctly",
			importPath:  "../sibling/baz",
			importerDir: "src/pkg",
			rootDir:     "/repo",
			want:        filepath.Clean("src/sibling/baz"),
		},
		{
			name:        "Go module import with /internal/ strips prefix",
			importPath:  "github.com/harvx/harvx/internal/config",
			importerDir: "internal/cli",
			rootDir:     "/repo",
			want:        filepath.Clean("internal/config"),
		},
		{
			name:        "Go module import with /cmd/ strips prefix",
			importPath:  "github.com/harvx/harvx/cmd/harvx",
			importerDir: "internal/cli",
			rootDir:     "/repo",
			want:        filepath.Clean("cmd/harvx"),
		},
		{
			name:        "Go module import with /pkg/ strips prefix",
			importPath:  "github.com/example/project/pkg/utils",
			importerDir: "internal/handler",
			rootDir:     "/repo",
			want:        filepath.Clean("pkg/utils"),
		},
		{
			name:        "import starting with internal/ returned as-is",
			importPath:  "internal/config",
			importerDir: "cmd/main",
			rootDir:     "/repo",
			want:        filepath.Clean("internal/config"),
		},
		{
			name:        "import starting with cmd/ returned as-is",
			importPath:  "cmd/harvx",
			importerDir: "internal/cli",
			rootDir:     "/repo",
			want:        filepath.Clean("cmd/harvx"),
		},
		{
			name:        "import starting with pkg/ returned as-is",
			importPath:  "pkg/util",
			importerDir: "internal/handler",
			rootDir:     "/repo",
			want:        filepath.Clean("pkg/util"),
		},
		{
			name:        "unrecognized import returns empty",
			importPath:  "fmt",
			importerDir: "src",
			rootDir:     "/repo",
			want:        "",
		},
		{
			name:        "unrecognized third party import returns empty",
			importPath:  "github.com/stretchr/testify/assert",
			importerDir: "internal/test",
			rootDir:     "/repo",
			want:        "",
		},
		{
			name:        "stdlib import returns empty",
			importPath:  "os/exec",
			importerDir: "internal/runner",
			rootDir:     "/repo",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolveImportToPath(tt.importPath, tt.importerDir, tt.rootDir)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestFilterExcluding
// ---------------------------------------------------------------------------

func TestFilterExcluding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		items      []string
		excludeSet map[string]bool
		want       []string
	}{
		{
			name:       "filters out excluded items",
			items:      []string{"a.go", "b.go", "c.go"},
			excludeSet: map[string]bool{"b.go": true},
			want:       []string{"a.go", "c.go"},
		},
		{
			name:       "returns sorted result",
			items:      []string{"z.go", "a.go", "m.go"},
			excludeSet: map[string]bool{},
			want:       []string{"a.go", "m.go", "z.go"},
		},
		{
			name:       "empty items returns nil",
			items:      []string{},
			excludeSet: map[string]bool{"a.go": true},
			want:       nil,
		},
		{
			name:       "nil items returns nil",
			items:      nil,
			excludeSet: map[string]bool{"a.go": true},
			want:       nil,
		},
		{
			name:       "empty exclude set returns all items sorted",
			items:      []string{"c.go", "a.go"},
			excludeSet: map[string]bool{},
			want:       []string{"a.go", "c.go"},
		},
		{
			name:       "all items excluded returns nil",
			items:      []string{"a.go", "b.go"},
			excludeSet: map[string]bool{"a.go": true, "b.go": true},
			want:       nil,
		},
		{
			name:       "exclude items not in list has no effect",
			items:      []string{"a.go"},
			excludeSet: map[string]bool{"nonexistent.go": true},
			want:       []string{"a.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := filterExcluding(tt.items, tt.excludeSet)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestFileExistsInSet
// ---------------------------------------------------------------------------

func TestFileExistsInSet(t *testing.T) {
	t.Parallel()

	fileSet := map[string]bool{
		"src/main.go": true,
		"README.md":   true,
	}

	assert.True(t, fileExistsInSet("src/main.go", fileSet))
	assert.True(t, fileExistsInSet("README.md", fileSet))
	assert.False(t, fileExistsInSet("missing.go", fileSet))
	assert.False(t, fileExistsInSet("", fileSet))
	assert.False(t, fileExistsInSet("src/main.go", nil))
}

// ---------------------------------------------------------------------------
// TestHasParseableExtension
// ---------------------------------------------------------------------------

func TestHasParseableExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{"main.go", true},
		{"app.ts", true},
		{"component.tsx", true},
		{"util.js", true},
		{"button.jsx", true},
		{"script.py", true},
		{"README.md", false},
		{"config.toml", false},
		{"Makefile", false},
		{"image.png", false},
		{"style.css", false},
		{"data.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, hasParseableExtension(tt.path))
		})
	}
}

// ---------------------------------------------------------------------------
// TestDiscoverNeighbors_WithImporters
// ---------------------------------------------------------------------------

func TestDiscoverNeighbors_WithImporters(t *testing.T) {
	t.Parallel()

	// Create a temp directory with files that import each other.
	dir := t.TempDir()

	// changed.go is a Go file that is "changed".
	writeFile(t, dir, "internal/config/config.go", `package config

type Config struct {
	Name string
}
`)

	// importer.go imports the changed file.
	writeFile(t, dir, "internal/cli/root.go", `package cli

import (
	"github.com/harvx/harvx/internal/config"
)

func Run() {
	_ = config.Config{}
}
`)

	// config_test.go is a test file for the changed file.
	writeFile(t, dir, "internal/config/config_test.go", `package config

import "testing"

func TestConfig(t *testing.T) {}
`)

	opts := NeighborOptions{
		RootDir:      dir,
		ChangedFiles: []string{"internal/config/config.go"},
		AllFiles: []string{
			"internal/cli/root.go",
			"internal/config/config.go",
			"internal/config/config_test.go",
		},
		Depth: 1,
	}

	result := DiscoverNeighbors(opts)
	require.NotNil(t, result)

	// Should find the test file.
	assert.Contains(t, result.TestFiles, "internal/config/config_test.go")

	// Should find the importer.
	assert.Contains(t, result.ImporterFiles, "internal/cli/root.go")

	// AllNeighbors should contain both but not the changed file.
	assert.Contains(t, result.AllNeighbors, "internal/config/config_test.go")
	assert.Contains(t, result.AllNeighbors, "internal/cli/root.go")
	assert.NotContains(t, result.AllNeighbors, "internal/config/config.go")
}

// ---------------------------------------------------------------------------
// TestFindRelatedTests
// ---------------------------------------------------------------------------

func TestFindRelatedTests(t *testing.T) {
	t.Parallel()

	allFileSet := map[string]bool{
		"src/main.go":      true,
		"src/main_test.go": true,
		"src/util.go":      true,
		"src/util_test.go": true,
		"src/helper.go":    true,
	}

	tests := []struct {
		name         string
		changedFiles []string
		want         []string
	}{
		{
			name:         "finds test for single changed file",
			changedFiles: []string{"src/main.go"},
			want:         []string{"src/main_test.go"},
		},
		{
			name:         "finds tests for multiple changed files",
			changedFiles: []string{"src/main.go", "src/util.go"},
			want:         []string{"src/main_test.go", "src/util_test.go"},
		},
		{
			name:         "no test file exists",
			changedFiles: []string{"src/helper.go"},
			want:         nil,
		},
		{
			name:         "empty changed files",
			changedFiles: []string{},
			want:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := findRelatedTests("", tt.changedFiles, allFileSet)
			if tt.want == nil {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
