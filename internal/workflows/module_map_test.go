package workflows

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateModuleMap_KnownDirectories(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create known directories.
	for _, name := range []string{"cmd", "internal", "docs", "scripts"} {
		require.NoError(t, os.Mkdir(filepath.Join(dir, name), 0o755))
	}

	entries, err := GenerateModuleMap(dir)
	require.NoError(t, err)
	require.Len(t, entries, 4)

	// Verify sorted order.
	assert.Equal(t, "cmd", entries[0].Name)
	assert.Equal(t, "CLI entry points", entries[0].Description)
	assert.Equal(t, "docs", entries[1].Name)
	assert.Equal(t, "Documentation", entries[1].Description)
	assert.Equal(t, "internal", entries[2].Name)
	assert.Contains(t, entries[2].Description, "Private packages")
	assert.Equal(t, "scripts", entries[3].Name)
	assert.Equal(t, "Automation and build scripts", entries[3].Description)
}

func TestGenerateModuleMap_HiddenDirectoriesExcluded(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".hidden"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".github"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "src"), 0o755))

	entries, err := GenerateModuleMap(dir)
	require.NoError(t, err)

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}

	assert.Contains(t, names, ".github", ".github is a known directory and should be included")
	assert.Contains(t, names, "src")
	assert.NotContains(t, names, ".hidden", "unknown hidden directories should be excluded")
}

func TestGenerateModuleMap_InfersFromContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create a directory with Go files.
	goDir := filepath.Join(dir, "mypackage")
	require.NoError(t, os.Mkdir(goDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "main.go"), []byte("package main"), 0o644))

	entries, err := GenerateModuleMap(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.Equal(t, "mypackage", entries[0].Name)
	assert.Equal(t, "Go source code", entries[0].Description)
}

func TestGenerateModuleMap_EmptyDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	entries, err := GenerateModuleMap(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestGenerateModuleMap_SkipsFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create a file (not a directory).
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0o644))
	// Create a directory.
	require.NoError(t, os.Mkdir(filepath.Join(dir, "src"), 0o755))

	entries, err := GenerateModuleMap(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "src", entries[0].Name)
}

func TestGenerateModuleMap_DeterministicOrder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	for _, name := range []string{"zulu", "alpha", "mike", "bravo"} {
		require.NoError(t, os.Mkdir(filepath.Join(dir, name), 0o755))
	}

	entries1, err := GenerateModuleMap(dir)
	require.NoError(t, err)

	entries2, err := GenerateModuleMap(dir)
	require.NoError(t, err)

	require.Len(t, entries1, 4)
	require.Len(t, entries2, 4)

	for i := range entries1 {
		assert.Equal(t, entries1[i].Name, entries2[i].Name)
		assert.Equal(t, entries1[i].Description, entries2[i].Description)
	}

	// Verify alphabetical order.
	assert.Equal(t, "alpha", entries1[0].Name)
	assert.Equal(t, "bravo", entries1[1].Name)
	assert.Equal(t, "mike", entries1[2].Name)
	assert.Equal(t, "zulu", entries1[3].Name)
}

func TestRenderModuleMap(t *testing.T) {
	t.Parallel()

	entries := []ModuleMapEntry{
		{Name: "cmd", Description: "CLI entry points"},
		{Name: "internal", Description: "Private packages"},
	}

	rendered := RenderModuleMap(entries)
	assert.Contains(t, rendered, "- `cmd/` -- CLI entry points")
	assert.Contains(t, rendered, "- `internal/` -- Private packages")
}

func TestRenderModuleMap_Empty(t *testing.T) {
	t.Parallel()
	rendered := RenderModuleMap(nil)
	assert.Empty(t, rendered)
}

func TestGenerateModuleMap_InvalidDirectory(t *testing.T) {
	t.Parallel()
	_, err := GenerateModuleMap("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Additional module map tests for T-070 coverage
// ---------------------------------------------------------------------------

// TestGenerateModuleMap_AllKnownDirectories tests a broader set of well-known
// directory names to ensure their conventional descriptions are correct.
func TestGenerateModuleMap_AllKnownDirectories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		desc string
	}{
		{"cmd", "CLI entry points"},
		{"internal", "Private packages (not importable externally)"},
		{"pkg", "Public library packages"},
		{"lib", "Shared libraries"},
		{"src", "Source code"},
		{"api", "API definitions and handlers"},
		{"docs", "Documentation"},
		{"test", "Test files and fixtures"},
		{"tests", "Test files and fixtures"},
		{"testdata", "Test fixture data"},
		{"scripts", "Automation and build scripts"},
		{"tools", "Development tools and utilities"},
		{"build", "Build configuration and packaging"},
		{"deploy", "Deployment configuration"},
		{"config", "Configuration files"},
		{"migrations", "Database migrations"},
		{"vendor", "Vendored dependencies"},
		{"assets", "Static assets (images, fonts, etc.)"},
		{"templates", "Template files"},
		{"components", "UI components"},
		{"middleware", "HTTP/gRPC middleware"},
		{"services", "Service layer implementations"},
		{"models", "Data models and schemas"},
		{"proto", "Protocol Buffer definitions"},
		{"examples", "Example code and usage"},
		{".github", "GitHub workflows and configuration"},
		{"grammars", "Parser grammars (e.g., tree-sitter WASM)"},
		{"frontend", "Frontend application code"},
		{"backend", "Backend application code"},
		{"server", "Server implementation"},
		{"client", "Client implementation"},
		{"e2e", "End-to-end tests"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, tt.name), 0o755))

			entries, err := GenerateModuleMap(dir)
			require.NoError(t, err)
			require.Len(t, entries, 1)
			assert.Equal(t, tt.name, entries[0].Name)
			assert.Equal(t, tt.desc, entries[0].Description)
		})
	}
}

// TestGenerateModuleMap_InfersTypescript verifies that a directory containing
// .ts files is described as "TypeScript source code".
func TestGenerateModuleMap_InfersTypescript(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	tsDir := filepath.Join(dir, "mylib")
	require.NoError(t, os.Mkdir(tsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tsDir, "index.ts"), []byte("export {}"), 0o644))

	entries, err := GenerateModuleMap(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "TypeScript source code", entries[0].Description)
}

// TestGenerateModuleMap_InfersPython verifies that a directory containing
// .py files is described as "Python source code".
func TestGenerateModuleMap_InfersPython(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	pyDir := filepath.Join(dir, "mylib")
	require.NoError(t, os.Mkdir(pyDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pyDir, "app.py"), []byte("print()"), 0o644))

	entries, err := GenerateModuleMap(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "Python source code", entries[0].Description)
}

// TestGenerateModuleMap_InfersRust verifies that a directory containing
// .rs files is described as "Rust source code".
func TestGenerateModuleMap_InfersRust(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	rsDir := filepath.Join(dir, "mylib")
	require.NoError(t, os.Mkdir(rsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rsDir, "main.rs"), []byte("fn main(){}"), 0o644))

	entries, err := GenerateModuleMap(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "Rust source code", entries[0].Description)
}

// TestGenerateModuleMap_InfersMarkdownDocs verifies that a directory with
// only .md files is described as "Documentation".
func TestGenerateModuleMap_InfersMarkdownDocs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	mdDir := filepath.Join(dir, "notes")
	require.NoError(t, os.Mkdir(mdDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mdDir, "design.md"), []byte("# Design"), 0o644))

	entries, err := GenerateModuleMap(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "Documentation", entries[0].Description)
}

// TestGenerateModuleMap_UnknownDirNoFiles verifies that an unknown directory
// with no recognizable files gets the generic "Project directory" description.
func TestGenerateModuleMap_UnknownDirNoFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	unknownDir := filepath.Join(dir, "custom")
	require.NoError(t, os.Mkdir(unknownDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(unknownDir, "data.csv"), []byte("a,b,c"), 0o644))

	entries, err := GenerateModuleMap(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "Project directory", entries[0].Description)
}

// TestRenderModuleMap_Format verifies that the render output has the correct
// format with backtick-wrapped dir names and double-dash separator.
func TestRenderModuleMap_Format(t *testing.T) {
	t.Parallel()

	entries := []ModuleMapEntry{
		{Name: "api", Description: "API definitions and handlers"},
	}

	rendered := RenderModuleMap(entries)
	assert.Equal(t, "- `api/` -- API definitions and handlers\n", rendered)
}
