package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/workflows"
)

// resetWorkspaceFlags resets the package-level workspace flag variables and
// Cobra flag state to their defaults. Call this in t.Cleanup after any test
// that executes the workspace command.
func resetWorkspaceFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		workspaceJSON = false
		workspaceDeep = false
		// Reset Cobra flag Changed state to prevent leaking into other tests.
		if f := workspaceCmd.Flags().Lookup("json"); f != nil {
			f.Changed = false
			_ = f.Value.Set("false")
		}
		if f := workspaceCmd.Flags().Lookup("deep"); f != nil {
			f.Changed = false
			_ = f.Value.Set("false")
		}
		// Reset global flags that may have been set via --dir.
		if f := rootCmd.PersistentFlags().Lookup("dir"); f != nil {
			f.Changed = false
			_ = f.Value.Set(".")
		}
	})
}

func TestWorkspaceCmd_Registration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "workspace" {
			found = true
			break
		}
	}
	assert.True(t, found, "workspace command should be registered on root")
}

func TestWorkspaceCmd_Properties(t *testing.T) {
	assert.Equal(t, "workspace", workspaceCmd.Use)
	assert.NotEmpty(t, workspaceCmd.Short)
	assert.NotEmpty(t, workspaceCmd.Long)
}

func TestWorkspaceCmd_JSONFlag(t *testing.T) {
	f := workspaceCmd.Flags().Lookup("json")
	require.NotNil(t, f, "--json flag should be registered")
	assert.Equal(t, "false", f.DefValue)
}

func TestWorkspaceCmd_DeepFlag(t *testing.T) {
	f := workspaceCmd.Flags().Lookup("deep")
	require.NotNil(t, f, "--deep flag should be registered")
	assert.Equal(t, "false", f.DefValue)
}

func TestWorkspaceInitCmd_Registration(t *testing.T) {
	found := false
	for _, cmd := range workspaceCmd.Commands() {
		if cmd.Name() == "init" {
			found = true
			break
		}
	}
	assert.True(t, found, "init subcommand should be registered on workspace")
}

func TestWorkspaceCmd_JSONOutput(t *testing.T) {
	resetWorkspaceFlags(t)

	// Create a temp dir with a workspace.toml.
	tmpDir := t.TempDir()
	harvxDir := filepath.Join(tmpDir, ".harvx")
	require.NoError(t, os.MkdirAll(harvxDir, 0o755))

	workspaceToml := `[workspace]
name = "Test"
description = "Test workspace"

[[workspace.repos]]
name = "test-repo"
path = "."
description = "Test repo"
`
	require.NoError(t, os.WriteFile(filepath.Join(harvxDir, "workspace.toml"), []byte(workspaceToml), 0o644))

	// Place a .git dir to bound the config discovery.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"workspace", "--json", "--dir", tmpDir})
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)
	defer rootCmd.SetArgs(nil)

	err := rootCmd.Execute()
	require.NoError(t, err)

	var meta workflows.WorkspaceJSON
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &meta), "stdout: %s", stdout.String())

	assert.Equal(t, "Test", meta.Name)
	assert.Equal(t, "Test workspace", meta.Description)
	assert.Equal(t, 1, meta.RepoCount)
	assert.Greater(t, meta.TokenCount, 0)
	assert.NotEmpty(t, meta.ContentHash)
	assert.Contains(t, meta.Repos, "test-repo")
}

func TestWorkspaceCmd_StdoutOutput(t *testing.T) {
	resetWorkspaceFlags(t)

	tmpDir := t.TempDir()
	harvxDir := filepath.Join(tmpDir, ".harvx")
	require.NoError(t, os.MkdirAll(harvxDir, 0o755))

	workspaceToml := `[workspace]
name = "Test"
[[workspace.repos]]
name = "test-repo"
path = "."
description = "A test repository"
`
	require.NoError(t, os.WriteFile(filepath.Join(harvxDir, "workspace.toml"), []byte(workspaceToml), 0o644))

	// Place a .git dir to bound the config discovery.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755))

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"workspace", "--stdout", "--dir", tmpDir})
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)
	defer rootCmd.SetArgs(nil)

	// Reset the --stdout flag after test to avoid leaking into other tests.
	t.Cleanup(func() {
		if f := rootCmd.PersistentFlags().Lookup("stdout"); f != nil {
			f.Changed = false
			_ = f.Value.Set("false")
		}
	})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Workspace: Test")
	assert.Contains(t, output, "test-repo")
}

func TestWorkspaceCmd_NoConfigFound(t *testing.T) {
	resetWorkspaceFlags(t)

	tmpDir := t.TempDir()
	// Create a .git directory so discovery stops here.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755))

	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"workspace", "--dir", tmpDir})
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)
	defer rootCmd.SetArgs(nil)

	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no .harvx/workspace.toml found")
}

func TestWorkspaceInitCmd_CreatesFile(t *testing.T) {
	resetWorkspaceFlags(t)

	tmpDir := t.TempDir()

	var stderr bytes.Buffer
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"workspace", "init", "--dir", tmpDir})
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)
	defer rootCmd.SetArgs(nil)

	err := rootCmd.Execute()
	require.NoError(t, err)

	outputPath := filepath.Join(tmpDir, ".harvx", "workspace.toml")
	assert.FileExists(t, outputPath)

	// Verify it is valid TOML by loading it.
	_, loadErr := config.LoadWorkspaceConfig(outputPath)
	require.NoError(t, loadErr)

	assert.Contains(t, stderr.String(), "Created")
}

func TestWorkspaceInitCmd_AlreadyExists(t *testing.T) {
	resetWorkspaceFlags(t)

	tmpDir := t.TempDir()
	harvxDir := filepath.Join(tmpDir, ".harvx")
	require.NoError(t, os.MkdirAll(harvxDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(harvxDir, "workspace.toml"), []byte("existing"), 0o644))

	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"workspace", "init", "--dir", tmpDir})
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)
	defer rootCmd.SetArgs(nil)

	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestWorkspaceCmd_GlobalFlagInheritance(t *testing.T) {
	// Verify global flags are accessible from workspace command.
	pf := workspaceCmd.InheritedFlags()
	assert.NotNil(t, pf.Lookup("dir"))
	assert.NotNil(t, pf.Lookup("target"))
	assert.NotNil(t, pf.Lookup("stdout"))
	assert.NotNil(t, pf.Lookup("output"))
	assert.NotNil(t, pf.Lookup("verbose"))
}

func TestWorkspaceCmd_HelpText(t *testing.T) {
	assert.Contains(t, workspaceCmd.Long, "workspace.toml")
	assert.Contains(t, workspaceCmd.Long, "harvx workspace init")
}