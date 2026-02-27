package filetree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testIgnorer is a simple Ignorer for testing that ignores paths matching
// a set of names.
type testIgnorer struct {
	ignored map[string]bool
}

func (ig *testIgnorer) IsIgnored(path string, isDir bool) bool {
	return ig.ignored[path]
}

func TestScanDirectory_Simple(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "src"), 0755))

	children, err := scanDirectory(dir, "", nil)
	require.NoError(t, err)
	require.Len(t, children, 3)

	names := make(map[string]bool)
	for _, c := range children {
		names[c.Name] = true
	}
	assert.True(t, names["main.go"])
	assert.True(t, names["README.md"])
	assert.True(t, names["src"])
}

func TestScanDirectory_Subdirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	require.NoError(t, os.Mkdir(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "app.go"), []byte("package src"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "util.go"), []byte("package src"), 0644))

	children, err := scanDirectory(dir, "src", nil)
	require.NoError(t, err)
	require.Len(t, children, 2)

	// Verify relative paths include the parent directory.
	for _, c := range children {
		assert.Contains(t, c.Path, "src/")
	}
}

func TestScanDirectory_SkipsGitDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644))

	children, err := scanDirectory(dir, "", nil)
	require.NoError(t, err)
	require.Len(t, children, 1)
	assert.Equal(t, "main.go", children[0].Name)
}

func TestScanDirectory_SkipsBinaryFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644))
	// Write a binary file (contains null byte).
	require.NoError(t, os.WriteFile(filepath.Join(dir, "binary.exe"), []byte{0x00, 0xFF, 0xFE}, 0644))

	children, err := scanDirectory(dir, "", nil)
	require.NoError(t, err)
	require.Len(t, children, 1)
	assert.Equal(t, "main.go", children[0].Name)
}

func TestScanDirectory_RespectsIgnorer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.log"), []byte("log data"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "vendor"), 0755))

	ignorer := &testIgnorer{
		ignored: map[string]bool{
			"test.log": true,
			"vendor":   true,
		},
	}

	children, err := scanDirectory(dir, "", ignorer)
	require.NoError(t, err)
	require.Len(t, children, 1)
	assert.Equal(t, "main.go", children[0].Name)
}

func TestScanDirectory_NonexistentDir(t *testing.T) {
	t.Parallel()

	_, err := scanDirectory("/nonexistent/path", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading directory")
}

func TestScanDirectory_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	children, err := scanDirectory(dir, "", nil)
	require.NoError(t, err)
	assert.Empty(t, children)
}

func TestLoadTopLevelCmd(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "src"), 0755))

	cmd := loadTopLevelCmd(dir, nil)
	require.NotNil(t, cmd)

	msg := cmd()
	dlm, ok := msg.(DirLoadedMsg)
	require.True(t, ok)
	assert.Empty(t, dlm.Path)
	assert.NoError(t, dlm.Err)
	assert.Len(t, dlm.Children, 2)
}

func TestLoadDirCmd(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	require.NoError(t, os.Mkdir(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "app.go"), []byte("package src"), 0644))

	cmd := loadDirCmd(dir, "src", nil)
	require.NotNil(t, cmd)

	msg := cmd()
	dlm, ok := msg.(DirLoadedMsg)
	require.True(t, ok)
	assert.Equal(t, "src", dlm.Path)
	assert.NoError(t, dlm.Err)
	assert.Len(t, dlm.Children, 1)
	assert.Equal(t, "app.go", dlm.Children[0].Name)
	assert.Equal(t, "src/app.go", dlm.Children[0].Path)
}

func TestLoadDirCmd_Error(t *testing.T) {
	t.Parallel()

	cmd := loadDirCmd("/nonexistent/path", "subdir", nil)
	require.NotNil(t, cmd)

	msg := cmd()
	dlm, ok := msg.(DirLoadedMsg)
	require.True(t, ok)
	assert.Error(t, dlm.Err)
}

func TestScanDirectory_NodeTypes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.go"), []byte("package main"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "pkg"), 0755))

	children, err := scanDirectory(dir, "", nil)
	require.NoError(t, err)
	require.Len(t, children, 2)

	var fileNode, dirNode *Node
	for _, c := range children {
		if c.IsDir {
			dirNode = c
		} else {
			fileNode = c
		}
	}

	require.NotNil(t, fileNode)
	require.NotNil(t, dirNode)
	assert.Equal(t, "file.go", fileNode.Name)
	assert.Equal(t, "file.go", fileNode.Path)
	assert.False(t, fileNode.IsDir)
	assert.Equal(t, "pkg", dirNode.Name)
	assert.Equal(t, "pkg", dirNode.Path)
	assert.True(t, dirNode.IsDir)
}
