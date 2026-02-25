package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializeSelectionToTOML_Basic(t *testing.T) {
	t.Parallel()

	result, err := serializeSelectionToTOML("myprofile", []string{
		"internal/config/types.go",
		"cmd/harvx/main.go",
		"go.mod",
	})
	require.NoError(t, err)

	// Should contain the profile name.
	assert.Contains(t, result, "myprofile")
	// Should contain include key.
	assert.Contains(t, result, "include")
	// Paths should be sorted.
	assert.Contains(t, result, "cmd/harvx/main.go")
	assert.Contains(t, result, "go.mod")
	assert.Contains(t, result, "internal/config/types.go")
}

func TestSerializeSelectionToTOML_EmptyName(t *testing.T) {
	t.Parallel()

	_, err := serializeSelectionToTOML("", []string{"a.go"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile name must not be empty")
}

func TestSerializeSelectionToTOML_EmptyPaths(t *testing.T) {
	t.Parallel()

	result, err := serializeSelectionToTOML("empty", nil)
	require.NoError(t, err)
	assert.Contains(t, result, "empty")
}

func TestSerializeSelectionToTOML_SortsDeterministically(t *testing.T) {
	t.Parallel()

	paths := []string{"z.go", "a.go", "m.go"}
	result1, err := serializeSelectionToTOML("test", paths)
	require.NoError(t, err)

	result2, err := serializeSelectionToTOML("test", paths)
	require.NoError(t, err)

	assert.Equal(t, result1, result2, "output should be deterministic")
}

func TestAppendProfileToFile_CreatesFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "harvx.toml")

	err := appendProfileToFile(path, "newprofile", []string{"main.go", "lib/util.go"})
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Contains(t, string(content), "newprofile")
	assert.Contains(t, string(content), "main.go")
	assert.Contains(t, string(content), "lib/util.go")
}

func TestAppendProfileToFile_AppendsToExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "harvx.toml")

	// Write initial content.
	err := os.WriteFile(path, []byte("[profile.default]\nformat = \"markdown\"\n"), 0o644)
	require.NoError(t, err)

	err = appendProfileToFile(path, "custom", []string{"app.go"})
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	// Original content should be preserved.
	assert.Contains(t, string(content), "[profile.default]")
	assert.Contains(t, string(content), "format = \"markdown\"")
	// New profile should be appended.
	assert.Contains(t, string(content), "custom")
	assert.Contains(t, string(content), "app.go")
}
