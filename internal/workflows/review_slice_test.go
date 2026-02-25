package workflows

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestGenerateReviewSlice_Validation
// ---------------------------------------------------------------------------

func TestGenerateReviewSlice_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    ReviewSliceOptions
		wantErr string
	}{
		{
			name:    "missing root dir returns error",
			opts:    ReviewSliceOptions{BaseRef: "main", HeadRef: "HEAD"},
			wantErr: "root directory required",
		},
		{
			name:    "missing base ref returns error",
			opts:    ReviewSliceOptions{RootDir: "/tmp/repo", HeadRef: "HEAD"},
			wantErr: "base ref required",
		},
		{
			name:    "missing head ref returns error",
			opts:    ReviewSliceOptions{RootDir: "/tmp/repo", BaseRef: "main"},
			wantErr: "head ref required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := GenerateReviewSlice(context.Background(), tt.opts)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// ---------------------------------------------------------------------------
// TestCollectRepoFiles
// ---------------------------------------------------------------------------

func TestCollectRepoFiles(t *testing.T) {
	t.Parallel()

	t.Run("walks and collects files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "main.go", "package main\n")
		writeFile(t, dir, "README.md", "# Hello\n")
		writeFile(t, dir, "internal/config/config.go", "package config\n")

		files, err := collectRepoFiles(dir)
		require.NoError(t, err)

		assert.Contains(t, files, "main.go")
		assert.Contains(t, files, "README.md")
		assert.Contains(t, files, "internal/config/config.go")

		// Results should be sorted.
		assert.True(t, sort.StringsAreSorted(files), "results should be sorted")
	})

	t.Run("skips hidden dirs (.git, .hidden)", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "main.go", "package main\n")
		writeFile(t, dir, ".git/config", "[core]\n")
		writeFile(t, dir, ".hidden/secret.txt", "secret\n")

		files, err := collectRepoFiles(dir)
		require.NoError(t, err)

		assert.Contains(t, files, "main.go")

		for _, f := range files {
			assert.False(t, strings.HasPrefix(f, ".git/"),
				"should skip .git directory, found: %s", f)
			assert.False(t, strings.HasPrefix(f, ".hidden/"),
				"should skip .hidden directory, found: %s", f)
		}
	})

	t.Run("skips node_modules and vendor", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "main.go", "package main\n")
		writeFile(t, dir, "node_modules/lodash/index.js", "module.exports = {}\n")
		writeFile(t, dir, "vendor/github.com/pkg/errors/errors.go", "package errors\n")

		files, err := collectRepoFiles(dir)
		require.NoError(t, err)

		assert.Contains(t, files, "main.go")

		for _, f := range files {
			assert.False(t, strings.HasPrefix(f, "node_modules/"),
				"should skip node_modules, found: %s", f)
			assert.False(t, strings.HasPrefix(f, "vendor/"),
				"should skip vendor, found: %s", f)
		}
	})

	t.Run("returns sorted paths", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "z.go", "package main\n")
		writeFile(t, dir, "a.go", "package main\n")
		writeFile(t, dir, "m.go", "package main\n")

		files, err := collectRepoFiles(dir)
		require.NoError(t, err)

		assert.True(t, sort.StringsAreSorted(files), "files should be sorted")
		assert.Equal(t, []string{"a.go", "m.go", "z.go"}, files)
	})

	t.Run("empty directory returns empty", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		files, err := collectRepoFiles(dir)
		require.NoError(t, err)
		assert.Empty(t, files)
	})
}

// ---------------------------------------------------------------------------
// TestBuildSliceFiles
// ---------------------------------------------------------------------------

func TestBuildSliceFiles(t *testing.T) {
	t.Parallel()

	simpleCounter := func(text string) int {
		return (len(text) + 3) / 4
	}

	t.Run("reads existing files correctly", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "main.go", "package main\n")
		writeFile(t, dir, "util.go", "package main\n\nfunc Util() {}\n")

		files := buildSliceFiles(dir, []string{"main.go", "util.go"}, true, simpleCounter)

		require.Len(t, files, 2)
		assert.Equal(t, "main.go", files[0].Path)
		assert.Equal(t, "package main\n", files[0].Content)
		assert.True(t, files[0].IsChanged)
		assert.Greater(t, files[0].Tokens, 0)

		assert.Equal(t, "util.go", files[1].Path)
		assert.True(t, files[1].IsChanged)
	})

	t.Run("skips unreadable files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "readable.go", "package main\n")
		// "missing.go" does not exist on disk.

		files := buildSliceFiles(dir, []string{"readable.go", "missing.go"}, true, simpleCounter)

		require.Len(t, files, 1)
		assert.Equal(t, "readable.go", files[0].Path)
	})

	t.Run("sets IsChanged flag correctly", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "changed.go", "package main\n")
		writeFile(t, dir, "neighbor.go", "package main\n")

		changedFiles := buildSliceFiles(dir, []string{"changed.go"}, true, simpleCounter)
		neighborFiles := buildSliceFiles(dir, []string{"neighbor.go"}, false, simpleCounter)

		require.Len(t, changedFiles, 1)
		assert.True(t, changedFiles[0].IsChanged)

		require.Len(t, neighborFiles, 1)
		assert.False(t, neighborFiles[0].IsChanged)
	})

	t.Run("counts tokens using provided counter", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		content := "package main\n\nfunc Hello() string { return \"hello\" }\n"
		writeFile(t, dir, "main.go", content)

		customCounter := func(text string) int { return 42 }

		files := buildSliceFiles(dir, []string{"main.go"}, true, customCounter)

		require.Len(t, files, 1)
		assert.Equal(t, 42, files[0].Tokens)
	})

	t.Run("empty paths list returns empty", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		files := buildSliceFiles(dir, []string{}, true, simpleCounter)
		assert.Empty(t, files)
	})
}

// ---------------------------------------------------------------------------
// TestEnforceSliceBudget
// ---------------------------------------------------------------------------

func TestEnforceSliceBudget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		changed       []sliceFile
		neighbors     []sliceFile
		maxTokens     int
		wantChanged   int
		wantNeighbors int
	}{
		{
			name: "all files fit within budget",
			changed: []sliceFile{
				{Path: "a.go", Tokens: 100},
				{Path: "b.go", Tokens: 100},
			},
			neighbors: []sliceFile{
				{Path: "c.go", Tokens: 100},
				{Path: "d.go", Tokens: 100},
			},
			maxTokens:     500,
			wantChanged:   2,
			wantNeighbors: 2,
		},
		{
			name: "changed files exceed budget -- no neighbors",
			changed: []sliceFile{
				{Path: "a.go", Tokens: 300},
				{Path: "b.go", Tokens: 300},
			},
			neighbors: []sliceFile{
				{Path: "c.go", Tokens: 100},
			},
			maxTokens:     500,
			wantChanged:   2,
			wantNeighbors: 0,
		},
		{
			name: "partial neighbors included based on budget",
			changed: []sliceFile{
				{Path: "a.go", Tokens: 200},
			},
			neighbors: []sliceFile{
				{Path: "b.go", Tokens: 100},
				{Path: "c.go", Tokens: 100},
				{Path: "d.go", Tokens: 200}, // would exceed budget
			},
			maxTokens:     400,
			wantChanged:   1,
			wantNeighbors: 2, // b.go + c.go fit, d.go does not
		},
		{
			name:    "empty neighbors list",
			changed: []sliceFile{{Path: "a.go", Tokens: 100}},
			neighbors: []sliceFile{},
			maxTokens:     500,
			wantChanged:   1,
			wantNeighbors: 0,
		},
		{
			name:          "empty changed list",
			changed:       []sliceFile{},
			neighbors:     []sliceFile{{Path: "a.go", Tokens: 100}},
			maxTokens:     500,
			wantChanged:   0,
			wantNeighbors: 1,
		},
		{
			name: "exact budget boundary",
			changed: []sliceFile{
				{Path: "a.go", Tokens: 200},
			},
			neighbors: []sliceFile{
				{Path: "b.go", Tokens: 300},
			},
			maxTokens:     500,
			wantChanged:   1,
			wantNeighbors: 1,
		},
		{
			name: "neighbor exactly one token over budget",
			changed: []sliceFile{
				{Path: "a.go", Tokens: 200},
			},
			neighbors: []sliceFile{
				{Path: "b.go", Tokens: 301},
			},
			maxTokens:     500,
			wantChanged:   1,
			wantNeighbors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotChanged, gotNeighbors := enforceSliceBudget(tt.changed, tt.neighbors, tt.maxTokens)
			assert.Len(t, gotChanged, tt.wantChanged)

			if tt.wantNeighbors == 0 {
				assert.Empty(t, gotNeighbors)
			} else {
				assert.Len(t, gotNeighbors, tt.wantNeighbors)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRenderSliceMarkdown
// ---------------------------------------------------------------------------

func TestRenderSliceMarkdown(t *testing.T) {
	t.Parallel()

	t.Run("produces correct markdown with changed and neighbor sections", func(t *testing.T) {
		t.Parallel()

		changed := []sliceFile{
			{Path: "main.go", Content: "package main\n", IsChanged: true},
		}
		neighbors := []sliceFile{
			{Path: "util.go", Content: "package main\n\nfunc Util() {}\n", IsChanged: false},
		}
		deleted := []string{}
		opts := ReviewSliceOptions{
			BaseRef: "origin/main",
			HeadRef: "HEAD",
		}

		output := renderSliceMarkdown(changed, neighbors, deleted, opts, "abc123", 100)

		assert.Contains(t, output, "# Review Slice")
		assert.Contains(t, output, "origin/main")
		assert.Contains(t, output, "HEAD")
		assert.Contains(t, output, "abc123")
		assert.Contains(t, output, "## Changed Files")
		assert.Contains(t, output, "### `main.go`")
		assert.Contains(t, output, "```go")
		assert.Contains(t, output, "package main")
		assert.Contains(t, output, "## Neighborhood Context")
		assert.Contains(t, output, "### `util.go`")
	})

	t.Run("includes deleted files section", func(t *testing.T) {
		t.Parallel()

		changed := []sliceFile{
			{Path: "main.go", Content: "package main\n"},
		}
		deleted := []string{"old_file.go", "deprecated.go"}
		opts := ReviewSliceOptions{BaseRef: "v1", HeadRef: "v2"}

		output := renderSliceMarkdown(changed, nil, deleted, opts, "hash", 50)

		assert.Contains(t, output, "## Deleted Files")
		assert.Contains(t, output, "- `old_file.go`")
		assert.Contains(t, output, "- `deprecated.go`")
	})

	t.Run("handles empty neighbor list", func(t *testing.T) {
		t.Parallel()

		changed := []sliceFile{
			{Path: "main.go", Content: "package main\n"},
		}
		opts := ReviewSliceOptions{BaseRef: "main", HeadRef: "feature"}

		output := renderSliceMarkdown(changed, nil, nil, opts, "hash", 50)

		assert.Contains(t, output, "## Changed Files")
		assert.NotContains(t, output, "## Neighborhood Context")
		assert.NotContains(t, output, "## Deleted Files")
	})
}

// ---------------------------------------------------------------------------
// TestRenderSliceXML
// ---------------------------------------------------------------------------

func TestRenderSliceXML(t *testing.T) {
	t.Parallel()

	t.Run("produces correct XML with changed-files and neighborhood tags", func(t *testing.T) {
		t.Parallel()

		changed := []sliceFile{
			{Path: "main.go", Content: "package main\n"},
		}
		neighbors := []sliceFile{
			{Path: "util.go", Content: "package main\n"},
		}
		opts := ReviewSliceOptions{
			BaseRef: "origin/main",
			HeadRef: "HEAD",
			Target:  "claude",
		}

		output := renderSliceXML(changed, neighbors, nil, opts, "abc123", 100)

		assert.Contains(t, output, "<!-- Review Slice")
		assert.Contains(t, output, "<review-slice>")
		assert.Contains(t, output, "</review-slice>")
		assert.Contains(t, output, "<changed-files>")
		assert.Contains(t, output, "</changed-files>")
		assert.Contains(t, output, `<file path="main.go">`)
		assert.Contains(t, output, "<content>")
		assert.Contains(t, output, "</content>")
		assert.Contains(t, output, "<neighborhood>")
		assert.Contains(t, output, "</neighborhood>")
		assert.Contains(t, output, `<file path="util.go">`)
	})

	t.Run("includes deleted-files section", func(t *testing.T) {
		t.Parallel()

		changed := []sliceFile{
			{Path: "main.go", Content: "package main\n"},
		}
		deleted := []string{"old.go", "removed.go"}
		opts := ReviewSliceOptions{BaseRef: "v1", HeadRef: "v2", Target: "claude"}

		output := renderSliceXML(changed, nil, deleted, opts, "hash", 50)

		assert.Contains(t, output, "<deleted-files>")
		assert.Contains(t, output, `<file path="old.go"/>`)
		assert.Contains(t, output, `<file path="removed.go"/>`)
		assert.Contains(t, output, "</deleted-files>")
	})

	t.Run("omits empty sections", func(t *testing.T) {
		t.Parallel()

		changed := []sliceFile{
			{Path: "main.go", Content: "package main\n"},
		}
		opts := ReviewSliceOptions{BaseRef: "main", HeadRef: "feature", Target: "claude"}

		output := renderSliceXML(changed, nil, nil, opts, "hash", 50)

		assert.NotContains(t, output, "<neighborhood>")
		assert.NotContains(t, output, "<deleted-files>")
	})
}

// ---------------------------------------------------------------------------
// TestRenderEmptySlice
// ---------------------------------------------------------------------------

func TestRenderEmptySlice(t *testing.T) {
	t.Parallel()

	t.Run("markdown format", func(t *testing.T) {
		t.Parallel()

		opts := ReviewSliceOptions{BaseRef: "main", HeadRef: "feature"}

		output := renderEmptySlice(opts)

		assert.Contains(t, output, "# Review Slice")
		assert.Contains(t, output, "`main`")
		assert.Contains(t, output, "`feature`")
		assert.Contains(t, output, "No files changed")
	})

	t.Run("claude XML format", func(t *testing.T) {
		t.Parallel()

		opts := ReviewSliceOptions{BaseRef: "main", HeadRef: "feature", Target: "claude"}

		output := renderEmptySlice(opts)

		assert.Contains(t, output, "<!-- Review Slice")
		assert.Contains(t, output, "<review-slice>")
		assert.Contains(t, output, "</review-slice>")
		assert.Contains(t, output, "<message>")
		assert.Contains(t, output, "No files changed between main and feature.")
	})
}

// ---------------------------------------------------------------------------
// TestLanguageFromPath
// ---------------------------------------------------------------------------

func TestLanguageFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"app.ts", "typescript"},
		{"component.tsx", "typescript"},
		{"util.js", "javascript"},
		{"button.jsx", "javascript"},
		{"script.py", "python"},
		{"lib.rs", "rust"},
		{"App.java", "java"},
		{"main.c", "c"},
		{"main.cpp", "cpp"},
		{"main.cc", "cpp"},
		{"data.json", "json"},
		{"config.yaml", "yaml"},
		{"config.yml", "yaml"},
		{"settings.toml", "toml"},
		{"README.md", "markdown"},
		{"build.sh", "bash"},
		{"run.bash", "bash"},
		{"app.rb", "ruby"},
		{"App.swift", "swift"},
		{"Main.kt", "kotlin"},
		{"query.sql", "sql"},
		{"style.css", "css"},
		{"page.html", "html"},
		{"page.htm", "html"},
		{"doc.xml", "xml"},
		{"unknown.xyz", ""},
		{"noextension", ""},
		{".hidden", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, languageFromPath(tt.path))
		})
	}
}

// ---------------------------------------------------------------------------
// TestResolveImportPath
// ---------------------------------------------------------------------------

func TestResolveImportPath(t *testing.T) {
	t.Parallel()

	t.Run("relative import resolves correctly", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"src/util.ts": true,
		}

		results := resolveImportPath("src/main.ts", "./util", allFiles)
		assert.Contains(t, results, "src/util.ts")
	})

	t.Run("absolute import finds file", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"internal/config/config.go": true,
		}

		results := resolveImportPath("cmd/main.go", "internal/config/config.go", allFiles)
		assert.Contains(t, results, "internal/config/config.go")
	})

	t.Run("extension probing (.go, .ts, .js, etc.)", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"src/util.ts":  true,
			"src/helper.js": true,
			"pkg/lib.go":  true,
		}

		// Should find src/util.ts via extension probing.
		results := resolveImportPath("src/main.ts", "./util", allFiles)
		assert.Contains(t, results, "src/util.ts")

		// Should find pkg/lib.go via extension probing for absolute path.
		results2 := resolveImportPath("cmd/main.go", "pkg/lib", allFiles)
		assert.Contains(t, results2, "pkg/lib.go")
	})

	t.Run("index file resolution", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"src/components/index.ts": true,
		}

		results := resolveImportPath("src/app.ts", "./components", allFiles)
		assert.Contains(t, results, "src/components/index.ts")
	})

	t.Run("Go package directory listing", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"internal/config/config.go":  true,
			"internal/config/helpers.go": true,
			"internal/config/types.go":   true,
		}

		results := resolveImportPath("cmd/main.go", "internal/config", allFiles)
		assert.Contains(t, results, "internal/config/config.go")
		assert.Contains(t, results, "internal/config/helpers.go")
		assert.Contains(t, results, "internal/config/types.go")
	})

	t.Run("parent directory relative import", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"src/shared/util.js": true,
		}

		results := resolveImportPath("src/pages/home.js", "../shared/util", allFiles)
		assert.Contains(t, results, "src/shared/util.js")
	})

	t.Run("no match returns empty", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"src/main.go": true,
		}

		results := resolveImportPath("src/main.go", "nonexistent/package", allFiles)
		assert.Empty(t, results)
	})
}

// ---------------------------------------------------------------------------
// TestFindTestFiles
// ---------------------------------------------------------------------------

func TestFindTestFiles(t *testing.T) {
	t.Parallel()

	t.Run("Go file finds _test.go", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"pkg/handler.go":      true,
			"pkg/handler_test.go": true,
		}

		results := findTestFiles("pkg/handler.go", allFiles)
		assert.Equal(t, []string{"pkg/handler_test.go"}, results)
	})

	t.Run("TypeScript file finds .test.ts and .spec.ts", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"src/app.ts":      true,
			"src/app.test.ts": true,
			"src/app.spec.ts": true,
		}

		results := findTestFiles("src/app.ts", allFiles)
		assert.Contains(t, results, "src/app.test.ts")
		assert.Contains(t, results, "src/app.spec.ts")
	})

	t.Run("JavaScript file finds .test.js and .spec.js", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"lib/util.js":      true,
			"lib/util.test.js": true,
			"lib/util.spec.js": true,
		}

		results := findTestFiles("lib/util.js", allFiles)
		assert.Contains(t, results, "lib/util.test.js")
		assert.Contains(t, results, "lib/util.spec.js")
	})

	t.Run("Python file finds test_foo.py and foo_test.py", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"mymod/foo.py":      true,
			"mymod/test_foo.py": true,
			"mymod/foo_test.py": true,
		}

		results := findTestFiles("mymod/foo.py", allFiles)
		assert.Contains(t, results, "mymod/test_foo.py")
		assert.Contains(t, results, "mymod/foo_test.py")
	})

	t.Run("TypeScript file finds __tests__/ file", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"src/app.ts":               true,
			"src/__tests__/app.ts":     true,
		}

		results := findTestFiles("src/app.ts", allFiles)
		assert.Contains(t, results, "src/__tests__/app.ts")
	})

	t.Run("Python tests/ directory pattern", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"mymod/foo.py":             true,
			"mymod/tests/test_foo.py":  true,
		}

		results := findTestFiles("mymod/foo.py", allFiles)
		assert.Contains(t, results, "mymod/tests/test_foo.py")
	})

	t.Run("file with no test equivalent", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"pkg/handler.go": true,
			// No handler_test.go exists.
		}

		results := findTestFiles("pkg/handler.go", allFiles)
		assert.Empty(t, results)
	})

	t.Run("unsupported extension returns empty", func(t *testing.T) {
		t.Parallel()

		allFiles := map[string]bool{
			"config/settings.toml": true,
		}

		results := findTestFiles("config/settings.toml", allFiles)
		assert.Empty(t, results)
	})
}

// ---------------------------------------------------------------------------
// TestDiscoverNeighborsInternal
// ---------------------------------------------------------------------------

func TestDiscoverNeighborsInternal(t *testing.T) {
	t.Parallel()

	t.Run("depth 0 returns empty", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "main.go", "package main\n")

		result := discoverNeighbors(dir, []string{"main.go"}, []string{"main.go"}, 0)
		assert.Empty(t, result)
	})

	t.Run("depth 0 negative returns empty", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "main.go", "package main\n")

		result := discoverNeighbors(dir, []string{"main.go"}, []string{"main.go"}, -1)
		assert.Empty(t, result)
	})

	t.Run("depth 1 finds test files for changed files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "pkg/handler.go", "package pkg\n\nfunc Handle() {}\n")
		writeFile(t, dir, "pkg/handler_test.go", `package pkg

import "testing"

func TestHandle(t *testing.T) {}
`)

		changedPaths := []string{"pkg/handler.go"}
		allFiles := []string{"pkg/handler.go", "pkg/handler_test.go"}

		result := discoverNeighbors(dir, changedPaths, allFiles, 1)
		assert.Contains(t, result, "pkg/handler_test.go")
		assert.NotContains(t, result, "pkg/handler.go",
			"changed files should not appear in neighbors")
	})

	t.Run("depth 1 finds importers via file reading", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		// Create a changed Go file.
		writeFile(t, dir, "internal/config/config.go", "package config\n\ntype Config struct{}\n")

		// Create a Go file that imports the changed file via module path.
		writeFile(t, dir, "internal/cli/root.go", `package cli

import (
	"github.com/harvx/harvx/internal/config"
)

func Run() {
	_ = config.Config{}
}
`)

		changedPaths := []string{"internal/config/config.go"}
		allFiles := []string{
			"internal/cli/root.go",
			"internal/config/config.go",
		}

		result := discoverNeighbors(dir, changedPaths, allFiles, 1)
		assert.Contains(t, result, "internal/cli/root.go")
	})

	t.Run("same-directory heuristic for unsupported extensions", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "docs/guide.md", "# Guide\n")
		writeFile(t, dir, "docs/faq.md", "# FAQ\n")
		writeFile(t, dir, "docs/changelog.md", "# Changelog\n")
		writeFile(t, dir, "other/readme.md", "# Other\n")

		changedPaths := []string{"docs/guide.md"}
		allFiles := []string{
			"docs/changelog.md",
			"docs/faq.md",
			"docs/guide.md",
			"other/readme.md",
		}

		result := discoverNeighbors(dir, changedPaths, allFiles, 1)
		// Same-directory heuristic should include faq.md and changelog.md
		// but not other/readme.md
		assert.Contains(t, result, "docs/faq.md")
		assert.Contains(t, result, "docs/changelog.md")
		assert.NotContains(t, result, "other/readme.md")
		assert.NotContains(t, result, "docs/guide.md",
			"changed file should not appear in neighbors")
	})

	t.Run("TypeScript forward import discovery", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "src/app.ts", `import { helper } from './util'

export function app() { return helper() }
`)
		writeFile(t, dir, "src/util.ts", `export function helper() { return 42 }
`)

		changedPaths := []string{"src/app.ts"}
		allFiles := []string{"src/app.ts", "src/util.ts"}

		result := discoverNeighbors(dir, changedPaths, allFiles, 1)
		assert.Contains(t, result, "src/util.ts")
	})

	t.Run("empty changed paths returns empty", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "main.go", "package main\n")

		result := discoverNeighbors(dir, []string{}, []string{"main.go"}, 1)
		assert.Empty(t, result)
	})

	t.Run("results are sorted", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		writeFile(t, dir, "pkg/z.go", "package pkg\n")
		writeFile(t, dir, "pkg/a.go", "package pkg\n")
		writeFile(t, dir, "pkg/m.go", "package pkg\n")
		writeFile(t, dir, "pkg/changed.md", "# Changed\n")

		changedPaths := []string{"pkg/changed.md"}
		allFiles := []string{"pkg/a.go", "pkg/changed.md", "pkg/m.go", "pkg/z.go"}

		result := discoverNeighbors(dir, changedPaths, allFiles, 1)
		assert.True(t, sort.StringsAreSorted(result), "neighbors should be sorted")
	})
}

// ---------------------------------------------------------------------------
// TestRenderSlice_TargetDispatch
// ---------------------------------------------------------------------------

func TestRenderSlice_TargetDispatch(t *testing.T) {
	t.Parallel()

	changed := []sliceFile{
		{Path: "main.go", Content: "package main\n"},
	}

	t.Run("target claude produces XML", func(t *testing.T) {
		t.Parallel()

		opts := ReviewSliceOptions{
			BaseRef: "main", HeadRef: "feature",
			Target: "claude",
		}

		output := renderSlice(changed, nil, nil, opts, "hash", 50)
		assert.Contains(t, output, "<review-slice>")
		assert.NotContains(t, output, "# Review Slice")
	})

	t.Run("target empty produces markdown", func(t *testing.T) {
		t.Parallel()

		opts := ReviewSliceOptions{
			BaseRef: "main", HeadRef: "feature",
		}

		output := renderSlice(changed, nil, nil, opts, "hash", 50)
		assert.Contains(t, output, "# Review Slice")
		assert.NotContains(t, output, "<review-slice>")
	})

	t.Run("target generic produces markdown", func(t *testing.T) {
		t.Parallel()

		opts := ReviewSliceOptions{
			BaseRef: "main", HeadRef: "feature",
			Target: "generic",
		}

		output := renderSlice(changed, nil, nil, opts, "hash", 50)
		assert.Contains(t, output, "# Review Slice")
	})
}

// ---------------------------------------------------------------------------
// TestSkipDirs
// ---------------------------------------------------------------------------

func TestSkipDirs(t *testing.T) {
	t.Parallel()

	expectedSkipped := []string{".git", "node_modules", "vendor", "dist", "__pycache__"}
	for _, d := range expectedSkipped {
		assert.True(t, skipDirs[d], "%s should be in skipDirs", d)
	}

	nonSkipped := []string{"src", "internal", "cmd", "lib", "pkg", "test"}
	for _, d := range nonSkipped {
		assert.False(t, skipDirs[d], "%s should not be in skipDirs", d)
	}
}

// ---------------------------------------------------------------------------
// TestCollectRepoFiles_SkipsDistAndPycache
// ---------------------------------------------------------------------------

func TestCollectRepoFiles_SkipsDistAndPycache(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	writeFile(t, dir, "main.go", "package main\n")
	writeFile(t, dir, "dist/bundle.js", "compiled\n")
	writeFile(t, dir, "__pycache__/module.cpython-39.pyc", "\x00\x01\x02")

	files, err := collectRepoFiles(dir)
	require.NoError(t, err)

	assert.Contains(t, files, "main.go")
	for _, f := range files {
		assert.False(t, strings.HasPrefix(f, "dist/"),
			"should skip dist, found: %s", f)
		assert.False(t, strings.HasPrefix(f, "__pycache__/"),
			"should skip __pycache__, found: %s", f)
	}
}

// ---------------------------------------------------------------------------
// TestBuildSliceFiles_ForwardSlashPaths
// ---------------------------------------------------------------------------

func TestBuildSliceFiles_ForwardSlashPaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	writeFile(t, dir, "internal/config/config.go", "package config\n")

	// Pass forward-slash path (as collectRepoFiles produces).
	files := buildSliceFiles(dir, []string{"internal/config/config.go"}, true, estimateTokens)
	require.Len(t, files, 1)
	assert.Equal(t, "internal/config/config.go", files[0].Path)
	assert.Equal(t, "package config\n", files[0].Content)
}

// ---------------------------------------------------------------------------
// TestEnforceSliceBudget_ChangedAlwaysIncluded
// ---------------------------------------------------------------------------

func TestEnforceSliceBudget_ChangedAlwaysIncluded(t *testing.T) {
	t.Parallel()

	// Even when changed files exceed the budget, they are ALL included.
	changed := []sliceFile{
		{Path: "huge.go", Tokens: 50000},
	}
	neighbors := []sliceFile{
		{Path: "small.go", Tokens: 10},
	}

	gotChanged, gotNeighbors := enforceSliceBudget(changed, neighbors, 100)
	assert.Len(t, gotChanged, 1, "changed files are always included regardless of budget")
	assert.Equal(t, "huge.go", gotChanged[0].Path)
	assert.Empty(t, gotNeighbors, "no budget remaining for neighbors")
}

// ---------------------------------------------------------------------------
// TestRenderSliceMarkdown_ContentNewlineHandling
// ---------------------------------------------------------------------------

func TestRenderSliceMarkdown_ContentNewlineHandling(t *testing.T) {
	t.Parallel()

	// Content without trailing newline should still render valid markdown.
	changed := []sliceFile{
		{Path: "no_newline.go", Content: "package main"},
	}
	opts := ReviewSliceOptions{BaseRef: "a", HeadRef: "b"}

	output := renderSliceMarkdown(changed, nil, nil, opts, "hash", 10)

	// The renderer adds a newline before the closing fence if content lacks one.
	assert.Contains(t, output, "package main\n```")
}

// ---------------------------------------------------------------------------
// TestRenderSliceXML_ContentNewlineHandling
// ---------------------------------------------------------------------------

func TestRenderSliceXML_ContentNewlineHandling(t *testing.T) {
	t.Parallel()

	changed := []sliceFile{
		{Path: "no_newline.go", Content: "package main"},
	}
	opts := ReviewSliceOptions{BaseRef: "a", HeadRef: "b", Target: "claude"}

	output := renderSliceXML(changed, nil, nil, opts, "hash", 10)

	// Content without trailing newline should get one injected.
	assert.Contains(t, output, "package main\n</content>")
}

// ---------------------------------------------------------------------------
// TestDefaultConstants
// ---------------------------------------------------------------------------

func TestDefaultConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 20000, DefaultSliceMaxTokens)
	assert.Equal(t, 1, DefaultSliceDepth)
}

// ---------------------------------------------------------------------------
// TestCollectRepoFiles_NestedHiddenDirectories
// ---------------------------------------------------------------------------

func TestCollectRepoFiles_NestedHiddenDirectories(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Top-level hidden dir skipped, but file in non-hidden subdir visible.
	writeFile(t, dir, "src/main.go", "package main\n")
	writeFile(t, dir, ".config/settings.json", `{"key": "value"}`)
	writeFile(t, dir, "src/.hidden/secret.go", "package secret\n")

	files, err := collectRepoFiles(dir)
	require.NoError(t, err)

	assert.Contains(t, files, "src/main.go")

	for _, f := range files {
		assert.False(t, strings.Contains(f, ".config"),
			"should not contain .config directory files, found: %s", f)
		assert.False(t, strings.Contains(f, ".hidden"),
			"should not contain .hidden directory files, found: %s", f)
	}
}

// ---------------------------------------------------------------------------
// TestCollectRepoFiles_InvalidRoot
// ---------------------------------------------------------------------------

func TestCollectRepoFiles_InvalidRoot(t *testing.T) {
	t.Parallel()

	_, err := collectRepoFiles(filepath.Join(os.TempDir(), "nonexistent-dir-for-test-xyz"))
	assert.Error(t, err)
}
