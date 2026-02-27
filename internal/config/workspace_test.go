package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── LoadWorkspaceConfig ───────────────────────────────────────────────────────

// TestLoadWorkspaceConfig_FullFixture loads the test fixture workspace.toml and
// verifies that all fields are decoded correctly.
func TestLoadWorkspaceConfig_FullFixture(t *testing.T) {
	t.Parallel()

	path := testdataPath(t, "workspace.toml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture not found: %s", path)
	}

	cfg, err := LoadWorkspaceConfig(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "MyOrg Platform", cfg.Workspace.Name)
	assert.Equal(t, "Microservices platform with shared UI library", cfg.Workspace.Description)
	require.Len(t, cfg.Workspace.Repos, 3)

	// First repo: api-gateway.
	gw := cfg.Workspace.Repos[0]
	assert.Equal(t, "api-gateway", gw.Name)
	assert.Equal(t, "~/work/api-gateway", gw.Path)
	assert.Equal(t, "Express.js API gateway, handles auth and routing", gw.Description)
	assert.Equal(t, []string{"src/server.ts", "src/routes/"}, gw.Entrypoints)
	assert.Equal(t, []string{"user-service", "billing-service"}, gw.IntegratesWith)
	assert.Nil(t, gw.SharedSchemas)
	assert.Nil(t, gw.Docs)

	// Second repo: user-service.
	us := cfg.Workspace.Repos[1]
	assert.Equal(t, "user-service", us.Name)
	assert.Equal(t, []string{"api-gateway"}, us.IntegratesWith)
	assert.Equal(t, []string{"proto/user.proto"}, us.SharedSchemas)

	// Third repo: billing-service.
	bs := cfg.Workspace.Repos[2]
	assert.Equal(t, "billing-service", bs.Name)
	assert.Equal(t, []string{"api-gateway", "user-service"}, bs.IntegratesWith)
	assert.Equal(t, []string{"proto/billing.proto"}, bs.SharedSchemas)
	assert.Equal(t, []string{"docs/billing-api.md", "docs/stripe-integration.md"}, bs.Docs)
}

// TestLoadWorkspaceConfig_MinimalTOML verifies that a workspace.toml with just
// the workspace name and one minimal repo parses correctly.
func TestLoadWorkspaceConfig_MinimalTOML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.toml")
	data := `
[workspace]
name = "Minimal"

[[workspace.repos]]
name = "core"
path = "/tmp/core"
`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	cfg, err := LoadWorkspaceConfig(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "Minimal", cfg.Workspace.Name)
	assert.Empty(t, cfg.Workspace.Description)
	require.Len(t, cfg.Workspace.Repos, 1)
	assert.Equal(t, "core", cfg.Workspace.Repos[0].Name)
	assert.Equal(t, "/tmp/core", cfg.Workspace.Repos[0].Path)
	assert.Nil(t, cfg.Workspace.Repos[0].Entrypoints)
	assert.Nil(t, cfg.Workspace.Repos[0].IntegratesWith)
	assert.Nil(t, cfg.Workspace.Repos[0].SharedSchemas)
	assert.Nil(t, cfg.Workspace.Repos[0].Docs)
}

// TestLoadWorkspaceConfig_EmptyWorkspace verifies that a workspace.toml with
// an empty [workspace] section (no repos) parses without error.
func TestLoadWorkspaceConfig_EmptyWorkspace(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.toml")
	require.NoError(t, os.WriteFile(path, []byte("[workspace]\n"), 0o644))

	cfg, err := LoadWorkspaceConfig(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Empty(t, cfg.Workspace.Name)
	assert.Empty(t, cfg.Workspace.Repos)
}

// TestLoadWorkspaceConfig_InvalidSyntax verifies that malformed TOML returns
// an error that mentions the file path.
func TestLoadWorkspaceConfig_InvalidSyntax(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad-workspace.toml")
	require.NoError(t, os.WriteFile(path, []byte("[broken toml"), 0o644))

	_, err := LoadWorkspaceConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad-workspace.toml",
		"error must mention the file path")
}

// TestLoadWorkspaceConfig_NonExistentFile verifies that a missing file returns
// an error.
func TestLoadWorkspaceConfig_NonExistentFile(t *testing.T) {
	t.Parallel()

	_, err := LoadWorkspaceConfig("/nonexistent/workspace.toml")
	require.Error(t, err)
}

// TestLoadWorkspaceConfig_UnknownKeysNoError verifies that unknown TOML keys
// do not cause an error (forward compatibility).
func TestLoadWorkspaceConfig_UnknownKeysNoError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.toml")
	data := `
[workspace]
name = "Test"
future_field = "something"

[[workspace.repos]]
name = "core"
path = "/tmp/core"
unknown_option = true
`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	cfg, err := LoadWorkspaceConfig(path)
	require.NoError(t, err, "unknown keys must not cause an error")
	require.NotNil(t, cfg)
	assert.Equal(t, "Test", cfg.Workspace.Name)
	require.Len(t, cfg.Workspace.Repos, 1)
	assert.Equal(t, "core", cfg.Workspace.Repos[0].Name)
}

// TestLoadWorkspaceConfig_EmptyFile verifies that an empty file returns a
// non-nil config with zero values.
func TestLoadWorkspaceConfig_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.toml")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))

	cfg, err := LoadWorkspaceConfig(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.Workspace.Name)
	assert.Empty(t, cfg.Workspace.Repos)
}

// ── DiscoverWorkspaceConfig ──────────────────────────────────────────────────

// TestDiscoverWorkspaceConfig_FoundInStartDir verifies that a
// .harvx/workspace.toml in the start directory is returned immediately.
func TestDiscoverWorkspaceConfig_FoundInStartDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	harvxDir := filepath.Join(dir, ".harvx")
	require.NoError(t, os.Mkdir(harvxDir, 0o755))
	configPath := filepath.Join(harvxDir, "workspace.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[workspace]\n"), 0o644))

	got, err := DiscoverWorkspaceConfig(dir)
	require.NoError(t, err)
	assertSamePath(t, configPath, got)
}

// TestDiscoverWorkspaceConfig_FoundInParentDir verifies that a
// .harvx/workspace.toml in a parent directory is found when not present in the
// start directory.
func TestDiscoverWorkspaceConfig_FoundInParentDir(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	harvxDir := filepath.Join(parent, ".harvx")
	require.NoError(t, os.Mkdir(harvxDir, 0o755))
	configPath := filepath.Join(harvxDir, "workspace.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[workspace]\n"), 0o644))

	child := filepath.Join(parent, "sub")
	require.NoError(t, os.Mkdir(child, 0o755))

	got, err := DiscoverWorkspaceConfig(child)
	require.NoError(t, err)
	assertSamePath(t, configPath, got)
}

// TestDiscoverWorkspaceConfig_NotFound verifies that an empty string is
// returned when no .harvx/workspace.toml exists anywhere in the directory chain.
func TestDiscoverWorkspaceConfig_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	got, err := DiscoverWorkspaceConfig(dir)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestDiscoverWorkspaceConfig_StopsAtGitBoundary verifies that the search
// stops at a directory containing a .git folder.
func TestDiscoverWorkspaceConfig_StopsAtGitBoundary(t *testing.T) {
	t.Parallel()

	// Layout:
	//   grandparent/
	//     .harvx/workspace.toml  <-- should NOT be found
	//     child/
	//       .git/                <-- boundary
	//       grandchild/          <-- start dir

	grandparent := t.TempDir()
	harvxDir := filepath.Join(grandparent, ".harvx")
	require.NoError(t, os.Mkdir(harvxDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(harvxDir, "workspace.toml"),
		[]byte("[workspace]\n"), 0o644,
	))

	child := filepath.Join(grandparent, "child")
	require.NoError(t, os.Mkdir(child, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(child, ".git"), 0o755))

	grandchild := filepath.Join(child, "grandchild")
	require.NoError(t, os.Mkdir(grandchild, 0o755))

	got, err := DiscoverWorkspaceConfig(grandchild)
	require.NoError(t, err)
	assert.Empty(t, got, "search must stop at .git boundary")
}

// TestDiscoverWorkspaceConfig_FoundAtGitBoundary verifies that a
// .harvx/workspace.toml at the same level as .git is returned.
func TestDiscoverWorkspaceConfig_FoundAtGitBoundary(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755))
	harvxDir := filepath.Join(repoRoot, ".harvx")
	require.NoError(t, os.Mkdir(harvxDir, 0o755))
	configPath := filepath.Join(harvxDir, "workspace.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[workspace]\n"), 0o644))

	sub := filepath.Join(repoRoot, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))

	got, err := DiscoverWorkspaceConfig(sub)
	require.NoError(t, err)
	assertSamePath(t, configPath, got)
}

// TestDiscoverWorkspaceConfig_SymlinkResolution verifies that symlinks in the
// directory chain are resolved before walking.
func TestDiscoverWorkspaceConfig_SymlinkResolution(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows")
	}

	real := t.TempDir()
	harvxDir := filepath.Join(real, ".harvx")
	require.NoError(t, os.Mkdir(harvxDir, 0o755))
	configPath := filepath.Join(harvxDir, "workspace.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[workspace]\n"), 0o644))

	sub := filepath.Join(real, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))

	linkBase := t.TempDir()
	link := filepath.Join(linkBase, "link")
	require.NoError(t, os.Symlink(sub, link))

	got, err := DiscoverWorkspaceConfig(link)
	require.NoError(t, err)

	resolvedConfig, err := filepath.EvalSymlinks(configPath)
	require.NoError(t, err)
	assert.Equal(t, resolvedConfig, got)
}

// TestDiscoverWorkspaceConfig_TableDriven exercises a range of directory
// layouts in a table-driven style.
func TestDiscoverWorkspaceConfig_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T) (startDir, wantConfig string)
		wantEmpty bool
	}{
		{
			name: "config in start dir",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				dir := t.TempDir()
				hDir := filepath.Join(dir, ".harvx")
				require.NoError(t, os.Mkdir(hDir, 0o755))
				cfg := filepath.Join(hDir, "workspace.toml")
				require.NoError(t, os.WriteFile(cfg, []byte("[workspace]\n"), 0o644))
				return dir, cfg
			},
		},
		{
			name: "config one level up",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				parent := t.TempDir()
				hDir := filepath.Join(parent, ".harvx")
				require.NoError(t, os.Mkdir(hDir, 0o755))
				cfg := filepath.Join(hDir, "workspace.toml")
				require.NoError(t, os.WriteFile(cfg, []byte("[workspace]\n"), 0o644))
				child := filepath.Join(parent, "sub")
				require.NoError(t, os.Mkdir(child, 0o755))
				return child, cfg
			},
		},
		{
			name:      "no config anywhere",
			wantEmpty: true,
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				return t.TempDir(), ""
			},
		},
		{
			name:      "git boundary stops before config",
			wantEmpty: true,
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				gp := t.TempDir()
				hDir := filepath.Join(gp, ".harvx")
				require.NoError(t, os.Mkdir(hDir, 0o755))
				require.NoError(t, os.WriteFile(
					filepath.Join(hDir, "workspace.toml"),
					[]byte("[workspace]\n"), 0o644,
				))
				repo := filepath.Join(gp, "repo")
				require.NoError(t, os.Mkdir(repo, 0o755))
				require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0o755))
				start := filepath.Join(repo, "pkg")
				require.NoError(t, os.Mkdir(start, 0o755))
				return start, ""
			},
		},
		{
			name: "config at same level as .git is found",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				root := t.TempDir()
				require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))
				hDir := filepath.Join(root, ".harvx")
				require.NoError(t, os.Mkdir(hDir, 0o755))
				cfg := filepath.Join(hDir, "workspace.toml")
				require.NoError(t, os.WriteFile(cfg, []byte("[workspace]\n"), 0o644))
				child := filepath.Join(root, "pkg")
				require.NoError(t, os.Mkdir(child, 0o755))
				return child, cfg
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			startDir, wantConfig := tt.setup(t)

			got, err := DiscoverWorkspaceConfig(startDir)
			require.NoError(t, err)

			if tt.wantEmpty {
				assert.Empty(t, got)
			} else {
				assertSamePath(t, wantConfig, got)
			}
		})
	}
}

// ── ValidateWorkspace ────────────────────────────────────────────────────────

// TestValidateWorkspace_NoWarningsForValidConfig verifies that a fully valid
// workspace config produces no warnings.
func TestValidateWorkspace_NoWarningsForValidConfig(t *testing.T) {
	t.Parallel()

	// Create real directories so path existence checks pass.
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	cfg := &WorkspaceConfig{
		Workspace: WorkspaceManifest{
			Name:        "Valid",
			Description: "A valid workspace",
			Repos: []WorkspaceRepo{
				{
					Name:           "svc-a",
					Path:           dir1,
					Description:    "Service A",
					IntegratesWith: []string{"svc-b"},
				},
				{
					Name:           "svc-b",
					Path:           dir2,
					Description:    "Service B",
					IntegratesWith: []string{"svc-a"},
				},
			},
		},
	}

	warnings := ValidateWorkspace(cfg)
	assert.Empty(t, warnings)
}

// TestValidateWorkspace_MissingRepoPath verifies that a non-existent repo path
// produces a warning.
func TestValidateWorkspace_MissingRepoPath(t *testing.T) {
	t.Parallel()

	cfg := &WorkspaceConfig{
		Workspace: WorkspaceManifest{
			Repos: []WorkspaceRepo{
				{
					Name: "missing",
					Path: "/nonexistent/path/to/repo",
				},
			},
		},
	}

	warnings := ValidateWorkspace(cfg)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "does not exist")
	assert.Contains(t, warnings[0], "missing")
}

// TestValidateWorkspace_UnknownIntegrationTarget verifies that referencing an
// unknown repo name in integrates_with produces a warning.
func TestValidateWorkspace_UnknownIntegrationTarget(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfg := &WorkspaceConfig{
		Workspace: WorkspaceManifest{
			Repos: []WorkspaceRepo{
				{
					Name:           "svc-a",
					Path:           dir,
					IntegratesWith: []string{"unknown-repo"},
				},
			},
		},
	}

	warnings := ValidateWorkspace(cfg)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "unknown repo")
	assert.Contains(t, warnings[0], "unknown-repo")
}

// TestValidateWorkspace_DuplicateRepoNames verifies that duplicate repo names
// produce a warning.
func TestValidateWorkspace_DuplicateRepoNames(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfg := &WorkspaceConfig{
		Workspace: WorkspaceManifest{
			Repos: []WorkspaceRepo{
				{Name: "svc-a", Path: dir},
				{Name: "svc-a", Path: dir},
			},
		},
	}

	warnings := ValidateWorkspace(cfg)
	require.NotEmpty(t, warnings)

	hasDuplicate := false
	for _, w := range warnings {
		if w == `duplicate repo name "svc-a"` {
			hasDuplicate = true
			break
		}
	}
	assert.True(t, hasDuplicate, "expected a duplicate repo name warning; got: %v", warnings)
}

// TestValidateWorkspace_NilConfig verifies that nil input returns nil warnings.
func TestValidateWorkspace_NilConfig(t *testing.T) {
	t.Parallel()

	warnings := ValidateWorkspace(nil)
	assert.Nil(t, warnings)
}

// TestValidateWorkspace_EmptyRepos verifies that an empty repos list produces
// no warnings.
func TestValidateWorkspace_EmptyRepos(t *testing.T) {
	t.Parallel()

	cfg := &WorkspaceConfig{
		Workspace: WorkspaceManifest{
			Name: "Empty",
		},
	}

	warnings := ValidateWorkspace(cfg)
	assert.Empty(t, warnings)
}

// TestValidateWorkspace_MultipleWarnings verifies that multiple validation
// issues are all reported.
func TestValidateWorkspace_MultipleWarnings(t *testing.T) {
	t.Parallel()

	cfg := &WorkspaceConfig{
		Workspace: WorkspaceManifest{
			Repos: []WorkspaceRepo{
				{
					Name:           "svc-a",
					Path:           "/nonexistent/a",
					IntegratesWith: []string{"no-such-repo"},
				},
				{
					Name: "svc-a", // duplicate name
					Path: "/nonexistent/b",
				},
			},
		},
	}

	warnings := ValidateWorkspace(cfg)
	// Expect at least: duplicate name, two missing paths, one unknown integration.
	assert.GreaterOrEqual(t, len(warnings), 3,
		"expected at least 3 warnings (duplicate name, missing paths, unknown integration)")
}

// TestValidateWorkspace_AllRepoPathsInvalid verifies that warnings are
// produced for every repo with an invalid path but no errors are returned.
func TestValidateWorkspace_AllRepoPathsInvalid(t *testing.T) {
	t.Parallel()

	cfg := &WorkspaceConfig{
		Workspace: WorkspaceManifest{
			Name: "Bad Paths",
			Repos: []WorkspaceRepo{
				{Name: "a", Path: "/no/such/path/a"},
				{Name: "b", Path: "/no/such/path/b"},
				{Name: "c", Path: "/no/such/path/c"},
			},
		},
	}

	warnings := ValidateWorkspace(cfg)
	assert.Len(t, warnings, 3, "each invalid path should produce one warning")
	for _, w := range warnings {
		assert.Contains(t, w, "does not exist")
	}
}

// ── ExpandPath ──────────────────────────────────────────────────────────────

// TestExpandPath exercises various path expansion scenarios.
func TestExpandPath(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name    string
		path    string
		baseDir string
		want    string
	}{
		{
			name:    "tilde expands to home",
			path:    "~/projects/repo",
			baseDir: "/base",
			want:    filepath.Join(home, "projects", "repo"),
		},
		{
			name:    "tilde alone expands to home",
			path:    "~",
			baseDir: "/base",
			want:    home,
		},
		{
			name:    "absolute path unchanged",
			path:    "/absolute/path",
			baseDir: "/base",
			want:    "/absolute/path",
		},
		{
			name:    "relative path resolved against baseDir",
			path:    "relative/path",
			baseDir: "/base/dir",
			want:    filepath.Join("/base/dir", "relative/path"),
		},
		{
			name:    "dot path resolved against baseDir",
			path:    "./local",
			baseDir: "/base/dir",
			want:    filepath.Join("/base/dir", "local"),
		},
		{
			name:    "empty path unchanged",
			path:    "",
			baseDir: "/base",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExpandPath(tt.path, tt.baseDir)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ── GenerateWorkspaceInit ──────────────────────────────────────────────────

// TestGenerateWorkspaceInit_ValidTOML verifies that the generated starter
// workspace.toml is syntactically valid and parseable.
func TestGenerateWorkspaceInit_ValidTOML(t *testing.T) {
	t.Parallel()

	content := GenerateWorkspaceInit()
	require.NotEmpty(t, content)

	// Write to a temp file and load it to verify it parses.
	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	cfg, err := LoadWorkspaceConfig(path)
	require.NoError(t, err, "generated workspace.toml must be parseable")
	require.NotNil(t, cfg)

	assert.Equal(t, "My Workspace", cfg.Workspace.Name)
	assert.Equal(t, "Description of the workspace", cfg.Workspace.Description)
	require.Len(t, cfg.Workspace.Repos, 1)
	assert.Equal(t, "repo-name", cfg.Workspace.Repos[0].Name)
	assert.Equal(t, "~/work/repo-name", cfg.Workspace.Repos[0].Path)
	assert.Equal(t, "Short description of this repository", cfg.Workspace.Repos[0].Description)
	assert.Equal(t, []string{"src/main.ts"}, cfg.Workspace.Repos[0].Entrypoints)
}

// TestGenerateWorkspaceInit_ContainsExpectedContent verifies that the
// generated template contains the expected TOML sections and comments.
func TestGenerateWorkspaceInit_ContainsExpectedContent(t *testing.T) {
	t.Parallel()

	content := GenerateWorkspaceInit()

	assert.Contains(t, content, "[workspace]")
	assert.Contains(t, content, "[[workspace.repos]]")
	assert.Contains(t, content, "# Harvx Workspace Manifest")
	assert.Contains(t, content, "harvx workspace --help")
	assert.Contains(t, content, "integrates_with")
}

// TestGenerateWorkspaceInit_Deterministic verifies that two calls return the
// same output (no random/time-based content).
func TestGenerateWorkspaceInit_Deterministic(t *testing.T) {
	t.Parallel()

	first := GenerateWorkspaceInit()
	second := GenerateWorkspaceInit()
	assert.Equal(t, first, second, "output must be deterministic")
}

// ── Integration: LoadWorkspaceConfig + ValidateWorkspace ──────────────────────

// TestLoadAndValidateWorkspace_IntegrationWithExpandedPaths verifies the full
// flow of loading a workspace config, expanding paths, and validating.
func TestLoadAndValidateWorkspace_IntegrationWithExpandedPaths(t *testing.T) {
	t.Parallel()

	// Create a real directory for one repo.
	realDir := t.TempDir()

	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.toml")
	data := `
[workspace]
name = "Integration Test"
description = "Tests the full flow"

[[workspace.repos]]
name = "real-repo"
path = "` + realDir + `"
description = "A repo with an existing path"
integrates_with = ["fake-repo"]

[[workspace.repos]]
name = "fake-repo"
path = "/nonexistent/fake"
description = "A repo with a non-existing path"
integrates_with = ["real-repo"]
`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	cfg, err := LoadWorkspaceConfig(path)
	require.NoError(t, err)

	warnings := ValidateWorkspace(cfg)
	// The only warning should be about fake-repo's missing path.
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "fake-repo")
	assert.Contains(t, warnings[0], "does not exist")
}

// TestLoadAndValidateWorkspace_ExpandPathsBeforeValidation shows the
// recommended pattern of expanding paths before validation.
func TestLoadAndValidateWorkspace_ExpandPathsBeforeValidation(t *testing.T) {
	t.Parallel()

	configDir := t.TempDir()

	// Create a subdirectory to use as a relative path repo.
	subName := "myrepo"
	subDir := filepath.Join(configDir, subName)
	require.NoError(t, os.Mkdir(subDir, 0o755))

	// Create another subdirectory to use as an absolute path repo.
	absRepo := filepath.Join(configDir, "abs-repo-dir")
	require.NoError(t, os.Mkdir(absRepo, 0o755))

	path := filepath.Join(configDir, "workspace.toml")
	data := `
[workspace]
name = "Expand Test"

[[workspace.repos]]
name = "abs-repo"
path = "` + absRepo + `"

[[workspace.repos]]
name = "rel-repo"
path = "` + subName + `"
`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	cfg, err := LoadWorkspaceConfig(path)
	require.NoError(t, err)

	// Expand paths relative to the config file's directory.
	for i := range cfg.Workspace.Repos {
		cfg.Workspace.Repos[i].Path = ExpandPath(
			cfg.Workspace.Repos[i].Path,
			configDir,
		)
	}

	warnings := ValidateWorkspace(cfg)
	assert.Empty(t, warnings, "all paths should exist after expansion")
}
