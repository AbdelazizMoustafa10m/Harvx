package workflows

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_Validation
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    ModuleSliceOptions
		wantErr string
	}{
		{
			name:    "missing root dir returns error",
			opts:    ModuleSliceOptions{Paths: []string{"internal/auth"}},
			wantErr: "root directory required",
		},
		{
			name:    "missing paths returns error",
			opts:    ModuleSliceOptions{RootDir: "/tmp/repo"},
			wantErr: "at least one --path is required",
		},
		{
			name:    "empty paths returns error",
			opts:    ModuleSliceOptions{RootDir: "/tmp/repo", Paths: []string{}},
			wantErr: "at least one --path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := GenerateModuleSlice(tt.opts)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_InvalidPath
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_InvalidPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main\n")

	_, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"nonexistent/dir"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent/dir")
	assert.Contains(t, err.Error(), "does not exist")
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_SingleDirectory
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_SingleDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create files inside the module path.
	writeFile(t, dir, "internal/auth/handler.go", "package auth\n\nfunc Handle() {}\n")
	writeFile(t, dir, "internal/auth/handler_test.go", "package auth\n\nimport \"testing\"\n\nfunc TestHandle(t *testing.T) {}\n")
	writeFile(t, dir, "internal/auth/middleware.go", "package auth\n\nfunc Middleware() {}\n")

	// Create files outside the module path.
	writeFile(t, dir, "internal/config/config.go", "package config\n\ntype Config struct{}\n")
	writeFile(t, dir, "main.go", "package main\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"internal/auth"},
		Depth:   0, // no neighbors
	})
	require.NoError(t, err)

	// All three auth files should be module files.
	assert.Len(t, result.ModuleFiles, 3)
	assert.Contains(t, result.ModuleFiles, "internal/auth/handler.go")
	assert.Contains(t, result.ModuleFiles, "internal/auth/handler_test.go")
	assert.Contains(t, result.ModuleFiles, "internal/auth/middleware.go")

	// No neighbors since depth is 0.
	assert.Empty(t, result.NeighborFiles)

	// Total files should match module files.
	assert.Equal(t, 3, result.TotalFiles)

	// Config and main should NOT be in module files.
	assert.NotContains(t, result.ModuleFiles, "internal/config/config.go")
	assert.NotContains(t, result.ModuleFiles, "main.go")
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_SingleFile
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_SingleFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "internal/auth/handler.go", "package auth\n\nfunc Handle() {}\n")
	writeFile(t, dir, "internal/auth/handler_test.go", "package auth\n\nimport \"testing\"\n\nfunc TestHandle(t *testing.T) {}\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"internal/auth/handler.go"},
		Depth:   0,
	})
	require.NoError(t, err)

	// Only the single specified file should be a module file.
	assert.Len(t, result.ModuleFiles, 1)
	assert.Contains(t, result.ModuleFiles, "internal/auth/handler.go")
	assert.NotContains(t, result.ModuleFiles, "internal/auth/handler_test.go")
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_MultiplePaths
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_MultiplePaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "internal/auth/handler.go", "package auth\n\nfunc Handle() {}\n")
	writeFile(t, dir, "internal/middleware/cors.go", "package middleware\n\nfunc CORS() {}\n")
	writeFile(t, dir, "internal/config/config.go", "package config\n\ntype Config struct{}\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"internal/auth", "internal/middleware"},
		Depth:   0,
	})
	require.NoError(t, err)

	// Both module paths should contribute files.
	assert.Len(t, result.ModuleFiles, 2)
	assert.Contains(t, result.ModuleFiles, "internal/auth/handler.go")
	assert.Contains(t, result.ModuleFiles, "internal/middleware/cors.go")

	// Config should be excluded.
	assert.NotContains(t, result.ModuleFiles, "internal/config/config.go")
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_WithNeighbors
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_WithNeighbors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Module file.
	writeFile(t, dir, "internal/auth/handler.go", "package auth\n\nfunc Handle() {}\n")

	// Test file for the module (should be discovered as neighbor).
	writeFile(t, dir, "internal/auth/handler_test.go", "package auth\n\nimport \"testing\"\n\nfunc TestHandle(t *testing.T) {}\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"internal/auth/handler.go"},
		Depth:   1,
	})
	require.NoError(t, err)

	// The single file is a module file.
	assert.Contains(t, result.ModuleFiles, "internal/auth/handler.go")

	// The test file should be a neighbor (discovered via test-file heuristic).
	assert.Contains(t, result.NeighborFiles, "internal/auth/handler_test.go")
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_NoFilesFound
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_NoFilesFound(t *testing.T) {
	t.Parallel()

	// The "no files found" error occurs when a path exists (passes os.Stat)
	// but collectRepoFiles finds no files under it. This happens when all
	// content is in hidden subdirectories that get skipped.
	// This case is covered by TestGenerateModuleSlice_EmptyDirectory below.
	// Here we verify the error message format with multiple paths.

	dir := t.TempDir()

	// Create directories that exist but whose files are all hidden.
	writeFile(t, dir, "pkg_a/.hidden/secret.go", "package secret\n")
	writeFile(t, dir, "pkg_b/.hidden/secret.go", "package secret\n")
	writeFile(t, dir, "main.go", "package main\n")

	_, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"pkg_a", "pkg_b"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no files found under path(s)")
	assert.Contains(t, err.Error(), "pkg_a")
	assert.Contains(t, err.Error(), "pkg_b")
}

// TestGenerateModuleSlice_EmptyDirectory tests a directory that exists but
// contains only a hidden subdirectory (skipped by collectRepoFiles).
func TestGenerateModuleSlice_EmptyDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create an "internal/empty" directory that exists (so stat passes) but
	// has files only in a hidden subdir (which collectRepoFiles skips).
	writeFile(t, dir, "internal/empty/.hidden/secret.go", "package secret\n")
	writeFile(t, dir, "main.go", "package main\n")

	_, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"internal/empty"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no files found under path(s)")
	assert.Contains(t, err.Error(), "internal/empty")
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_BudgetPrioritizesModuleFiles
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_BudgetPrioritizesModuleFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create a large module file and a neighbor.
	bigContent := strings.Repeat("x", 80000) + "\n" // ~20000 tokens at 4 chars/token
	writeFile(t, dir, "internal/auth/handler.go", bigContent)
	writeFile(t, dir, "internal/auth/handler_test.go", "package auth\n\nimport \"testing\"\n\nfunc TestHandle(t *testing.T) {}\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir:   dir,
		Paths:     []string{"internal/auth/handler.go"},
		MaxTokens: 20000,
		Depth:     1,
	})
	require.NoError(t, err)

	// Module file should always be included even when over budget.
	assert.Contains(t, result.ModuleFiles, "internal/auth/handler.go")

	// Neighbor should be excluded because the module file alone exceeds budget.
	assert.Empty(t, result.NeighborFiles)
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_DeterministicOutput
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_DeterministicOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "internal/auth/handler.go", "package auth\n\nfunc Handle() {}\n")
	writeFile(t, dir, "internal/auth/middleware.go", "package auth\n\nfunc Middleware() {}\n")

	opts := ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"internal/auth"},
		Depth:   0,
	}

	result1, err := GenerateModuleSlice(opts)
	require.NoError(t, err)

	result2, err := GenerateModuleSlice(opts)
	require.NoError(t, err)

	// Two runs with identical inputs must produce identical output.
	assert.Equal(t, result1.Content, result2.Content)
	assert.Equal(t, result1.ContentHash, result2.ContentHash)
	assert.Equal(t, result1.FormattedHash, result2.FormattedHash)
	assert.Equal(t, result1.TokenCount, result2.TokenCount)
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_MarkdownFormat
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_MarkdownFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "internal/auth/handler.go", "package auth\n\nfunc Handle() {}\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"internal/auth"},
		Depth:   0,
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "# Module Slice")
	assert.Contains(t, result.Content, "`internal/auth`")
	assert.Contains(t, result.Content, "## Module Files")
	assert.Contains(t, result.Content, "### `internal/auth/handler.go`")
	assert.Contains(t, result.Content, "```go")
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_XMLFormat
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_XMLFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "internal/auth/handler.go", "package auth\n\nfunc Handle() {}\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"internal/auth"},
		Target:  "claude",
		Depth:   0,
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "<!-- Module Slice")
	assert.Contains(t, result.Content, "<module-slice>")
	assert.Contains(t, result.Content, "</module-slice>")
	assert.Contains(t, result.Content, "<module-files>")
	assert.Contains(t, result.Content, "</module-files>")
	assert.Contains(t, result.Content, `<file path="internal/auth/handler.go">`)
	assert.Contains(t, result.Content, "<content>")
	assert.Contains(t, result.Content, "</content>")
	assert.NotContains(t, result.Content, "# Module Slice")
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_AssertIncludeSuccess
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_AssertIncludeSuccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "internal/auth/handler.go", "package auth\n\nfunc Handle() {}\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir:       dir,
		Paths:         []string{"internal/auth"},
		AssertInclude: []string{"internal/auth/*.go"},
		Depth:         0,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_AssertIncludeFailure
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_AssertIncludeFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "internal/auth/handler.go", "package auth\n\nfunc Handle() {}\n")

	_, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir:       dir,
		Paths:         []string{"internal/auth"},
		AssertInclude: []string{"nonexistent/**/*.go"},
		Depth:         0,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assert-include failed")
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_CustomTokenCounter
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_CustomTokenCounter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "internal/auth/handler.go", "package auth\n")

	customCounter := func(text string) int {
		return 42
	}

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir:      dir,
		Paths:        []string{"internal/auth"},
		TokenCounter: customCounter,
		Depth:        0,
	})
	require.NoError(t, err)
	assert.Equal(t, 42, result.TokenCount)
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_NeighborNotDuplicated
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_NeighborNotDuplicated(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Module file and its test.
	writeFile(t, dir, "internal/auth/handler.go", "package auth\n\nfunc Handle() {}\n")
	writeFile(t, dir, "internal/auth/handler_test.go", "package auth\n\nimport \"testing\"\n\nfunc TestHandle(t *testing.T) {}\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"internal/auth"},
		Depth:   1,
	})
	require.NoError(t, err)

	// Both files should be module files since they're under internal/auth.
	assert.Contains(t, result.ModuleFiles, "internal/auth/handler.go")
	assert.Contains(t, result.ModuleFiles, "internal/auth/handler_test.go")

	// The test file should NOT appear as a neighbor since it is already a module file.
	for _, n := range result.NeighborFiles {
		assert.NotEqual(t, "internal/auth/handler_test.go", n,
			"module files should not appear in neighbor list")
	}
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_DeeplyNestedPath
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_DeeplyNestedPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "src/components/dashboard/widgets/chart.go", "package widgets\n\nfunc Chart() {}\n")
	writeFile(t, dir, "src/components/dashboard/widgets/table.go", "package widgets\n\nfunc Table() {}\n")
	writeFile(t, dir, "src/components/dashboard/main.go", "package dashboard\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"src/components/dashboard/widgets"},
		Depth:   0,
	})
	require.NoError(t, err)

	assert.Len(t, result.ModuleFiles, 2)
	assert.Contains(t, result.ModuleFiles, "src/components/dashboard/widgets/chart.go")
	assert.Contains(t, result.ModuleFiles, "src/components/dashboard/widgets/table.go")

	// Parent directory file should not be included.
	assert.NotContains(t, result.ModuleFiles, "src/components/dashboard/main.go")
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_ContentHash
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_ContentHash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "internal/auth/handler.go", "package auth\n\nfunc Handle() {}\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"internal/auth"},
		Depth:   0,
	})
	require.NoError(t, err)

	assert.NotZero(t, result.ContentHash)
	assert.NotEmpty(t, result.FormattedHash)
	assert.Greater(t, result.TokenCount, 0)
}

// ---------------------------------------------------------------------------
// TestGenerateModuleSlice_SortedModuleFiles
// ---------------------------------------------------------------------------

func TestGenerateModuleSlice_SortedModuleFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, dir, "internal/auth/z_handler.go", "package auth\n")
	writeFile(t, dir, "internal/auth/a_handler.go", "package auth\n")
	writeFile(t, dir, "internal/auth/m_handler.go", "package auth\n")

	result, err := GenerateModuleSlice(ModuleSliceOptions{
		RootDir: dir,
		Paths:   []string{"internal/auth"},
		Depth:   0,
	})
	require.NoError(t, err)

	assert.True(t, sort.StringsAreSorted(result.ModuleFiles),
		"module files should be sorted")
}

// ---------------------------------------------------------------------------
// TestIsModuleFile
// ---------------------------------------------------------------------------

func TestIsModuleFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		filePath    string
		modulePaths []string
		want        bool
	}{
		{
			name:        "file under directory path",
			filePath:    "internal/auth/handler.go",
			modulePaths: []string{"internal/auth"},
			want:        true,
		},
		{
			name:        "file matches exact path",
			filePath:    "internal/auth/handler.go",
			modulePaths: []string{"internal/auth/handler.go"},
			want:        true,
		},
		{
			name:        "file outside directory",
			filePath:    "internal/config/config.go",
			modulePaths: []string{"internal/auth"},
			want:        false,
		},
		{
			name:        "file in sibling directory with shared prefix",
			filePath:    "internal/auth_extra/handler.go",
			modulePaths: []string{"internal/auth"},
			want:        false,
		},
		{
			name:        "multiple module paths match first",
			filePath:    "internal/auth/handler.go",
			modulePaths: []string{"internal/auth", "internal/config"},
			want:        true,
		},
		{
			name:        "multiple module paths match second",
			filePath:    "internal/config/config.go",
			modulePaths: []string{"internal/auth", "internal/config"},
			want:        true,
		},
		{
			name:        "deeply nested file matches parent path",
			filePath:    "src/components/dashboard/widgets/chart.go",
			modulePaths: []string{"src/components/dashboard"},
			want:        true,
		},
		{
			name:        "empty module paths",
			filePath:    "main.go",
			modulePaths: []string{},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isModuleFile(tt.filePath, tt.modulePaths)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestRenderModuleSliceMarkdown
// ---------------------------------------------------------------------------

func TestRenderModuleSliceMarkdown(t *testing.T) {
	t.Parallel()

	t.Run("produces correct markdown with module and neighbor sections", func(t *testing.T) {
		t.Parallel()

		moduleFiles := []sliceFile{
			{Path: "internal/auth/handler.go", Content: "package auth\n", IsChanged: true},
		}
		neighborFiles := []sliceFile{
			{Path: "internal/middleware/auth.go", Content: "package middleware\n", IsChanged: false},
		}
		opts := ModuleSliceOptions{
			Paths: []string{"internal/auth"},
		}

		output := renderModuleSliceMarkdown(moduleFiles, neighborFiles, opts, "abc123", 100)

		assert.Contains(t, output, "# Module Slice")
		assert.Contains(t, output, "`internal/auth`")
		assert.Contains(t, output, "abc123")
		assert.Contains(t, output, "## Module Files")
		assert.Contains(t, output, "### `internal/auth/handler.go`")
		assert.Contains(t, output, "```go")
		assert.Contains(t, output, "## Neighborhood Context")
		assert.Contains(t, output, "### `internal/middleware/auth.go`")
	})

	t.Run("handles empty neighbor list", func(t *testing.T) {
		t.Parallel()

		moduleFiles := []sliceFile{
			{Path: "main.go", Content: "package main\n"},
		}
		opts := ModuleSliceOptions{Paths: []string{"."}}

		output := renderModuleSliceMarkdown(moduleFiles, nil, opts, "hash", 50)

		assert.Contains(t, output, "## Module Files")
		assert.NotContains(t, output, "## Neighborhood Context")
	})

	t.Run("multiple paths in header", func(t *testing.T) {
		t.Parallel()

		moduleFiles := []sliceFile{
			{Path: "internal/auth/handler.go", Content: "package auth\n"},
		}
		opts := ModuleSliceOptions{
			Paths: []string{"internal/auth", "internal/middleware"},
		}

		output := renderModuleSliceMarkdown(moduleFiles, nil, opts, "hash", 50)

		assert.Contains(t, output, "`internal/auth`")
		assert.Contains(t, output, "`internal/middleware`")
	})

	t.Run("content without trailing newline is handled", func(t *testing.T) {
		t.Parallel()

		moduleFiles := []sliceFile{
			{Path: "no_newline.go", Content: "package main"},
		}
		opts := ModuleSliceOptions{Paths: []string{"."}}

		output := renderModuleSliceMarkdown(moduleFiles, nil, opts, "hash", 10)

		assert.Contains(t, output, "package main\n```")
	})
}

// ---------------------------------------------------------------------------
// TestRenderModuleSliceXML
// ---------------------------------------------------------------------------

func TestRenderModuleSliceXML(t *testing.T) {
	t.Parallel()

	t.Run("produces correct XML with module-files and neighborhood tags", func(t *testing.T) {
		t.Parallel()

		moduleFiles := []sliceFile{
			{Path: "internal/auth/handler.go", Content: "package auth\n"},
		}
		neighborFiles := []sliceFile{
			{Path: "internal/middleware/auth.go", Content: "package middleware\n"},
		}
		opts := ModuleSliceOptions{
			Paths:  []string{"internal/auth"},
			Target: "claude",
		}

		output := renderModuleSliceXML(moduleFiles, neighborFiles, opts, "abc123", 100)

		assert.Contains(t, output, "<!-- Module Slice")
		assert.Contains(t, output, "internal/auth")
		assert.Contains(t, output, "<module-slice>")
		assert.Contains(t, output, "</module-slice>")
		assert.Contains(t, output, "<module-files>")
		assert.Contains(t, output, "</module-files>")
		assert.Contains(t, output, `<file path="internal/auth/handler.go">`)
		assert.Contains(t, output, "<content>")
		assert.Contains(t, output, "</content>")
		assert.Contains(t, output, "<neighborhood>")
		assert.Contains(t, output, "</neighborhood>")
		assert.Contains(t, output, `<file path="internal/middleware/auth.go">`)
	})

	t.Run("omits empty sections", func(t *testing.T) {
		t.Parallel()

		moduleFiles := []sliceFile{
			{Path: "main.go", Content: "package main\n"},
		}
		opts := ModuleSliceOptions{Paths: []string{"."}, Target: "claude"}

		output := renderModuleSliceXML(moduleFiles, nil, opts, "hash", 50)

		assert.NotContains(t, output, "<neighborhood>")
	})

	t.Run("content without trailing newline is handled", func(t *testing.T) {
		t.Parallel()

		moduleFiles := []sliceFile{
			{Path: "no_newline.go", Content: "package main"},
		}
		opts := ModuleSliceOptions{Paths: []string{"."}, Target: "claude"}

		output := renderModuleSliceXML(moduleFiles, nil, opts, "hash", 10)

		assert.Contains(t, output, "package main\n</content>")
	})
}

// ---------------------------------------------------------------------------
// TestRenderModuleSlice_TargetDispatch
// ---------------------------------------------------------------------------

func TestRenderModuleSlice_TargetDispatch(t *testing.T) {
	t.Parallel()

	moduleFiles := []sliceFile{
		{Path: "main.go", Content: "package main\n"},
	}

	t.Run("target claude produces XML", func(t *testing.T) {
		t.Parallel()

		opts := ModuleSliceOptions{
			Paths:  []string{"."},
			Target: "claude",
		}

		output := renderModuleSlice(moduleFiles, nil, opts, "hash", 50)
		assert.Contains(t, output, "<module-slice>")
		assert.NotContains(t, output, "# Module Slice")
	})

	t.Run("target empty produces markdown", func(t *testing.T) {
		t.Parallel()

		opts := ModuleSliceOptions{
			Paths: []string{"."},
		}

		output := renderModuleSlice(moduleFiles, nil, opts, "hash", 50)
		assert.Contains(t, output, "# Module Slice")
		assert.NotContains(t, output, "<module-slice>")
	})

	t.Run("target generic produces markdown", func(t *testing.T) {
		t.Parallel()

		opts := ModuleSliceOptions{
			Paths:  []string{"."},
			Target: "generic",
		}

		output := renderModuleSlice(moduleFiles, nil, opts, "hash", 50)
		assert.Contains(t, output, "# Module Slice")
	})
}

// ---------------------------------------------------------------------------
// TestDefaultModuleSliceMaxTokens
// ---------------------------------------------------------------------------

func TestDefaultModuleSliceMaxTokens(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 20000, DefaultModuleSliceMaxTokens)
}
