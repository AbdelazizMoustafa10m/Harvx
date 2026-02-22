package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertSamePath asserts that two paths refer to the same filesystem location,
// resolving symlinks on both sides before comparing. This is required on macOS
// where t.TempDir() returns paths under /var (a symlink to /private/var) while
// DiscoverRepoConfig uses filepath.EvalSymlinks internally and returns canonical
// /private/var paths.
func assertSamePath(t *testing.T, expected, actual string, msgAndArgs ...any) {
	t.Helper()
	if expected == "" || actual == "" {
		assert.Equal(t, expected, actual, msgAndArgs...)
		return
	}
	resolvedExpected := expected
	if r, err := filepath.EvalSymlinks(expected); err == nil {
		resolvedExpected = r
	}
	resolvedActual := actual
	if r, err := filepath.EvalSymlinks(actual); err == nil {
		resolvedActual = r
	}
	assert.Equal(t, resolvedExpected, resolvedActual, msgAndArgs...)
}

// ── DiscoverRepoConfig ────────────────────────────────────────────────────────

// TestDiscoverRepoConfig_FoundInStartDir verifies that a harvx.toml in the
// start directory is returned immediately.
func TestDiscoverRepoConfig_FoundInStartDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "harvx.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[profile.default]\n"), 0o644))

	got, err := DiscoverRepoConfig(dir)
	require.NoError(t, err)
	assertSamePath(t, configPath, got)
}

// TestDiscoverRepoConfig_FoundInParentDir verifies that a harvx.toml in a
// parent directory is found when not present in the start directory.
func TestDiscoverRepoConfig_FoundInParentDir(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	configPath := filepath.Join(parent, "harvx.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[profile.default]\n"), 0o644))

	child := filepath.Join(parent, "sub")
	require.NoError(t, os.Mkdir(child, 0o755))

	got, err := DiscoverRepoConfig(child)
	require.NoError(t, err)
	assertSamePath(t, configPath, got)
}

// TestDiscoverRepoConfig_NotFound verifies that an empty string is returned
// when no harvx.toml exists anywhere in the directory chain.
func TestDiscoverRepoConfig_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	got, err := DiscoverRepoConfig(dir)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestDiscoverRepoConfig_StopsAtGitBoundary verifies that the search stops
// at a directory containing a .git folder, even if no harvx.toml is present.
func TestDiscoverRepoConfig_StopsAtGitBoundary(t *testing.T) {
	t.Parallel()

	// Layout:
	//   grandparent/
	//     harvx.toml       <-- should NOT be found
	//     child/
	//       .git/          <-- boundary
	//       grandchild/    <-- start dir

	grandparent := t.TempDir()
	grandparentConfig := filepath.Join(grandparent, "harvx.toml")
	require.NoError(t, os.WriteFile(grandparentConfig, []byte("[profile.default]\n"), 0o644))

	child := filepath.Join(grandparent, "child")
	require.NoError(t, os.Mkdir(child, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(child, ".git"), 0o755))

	grandchild := filepath.Join(child, "grandchild")
	require.NoError(t, os.Mkdir(grandchild, 0o755))

	got, err := DiscoverRepoConfig(grandchild)
	require.NoError(t, err)
	assert.Empty(t, got, "search must stop at .git boundary and not reach grandparent config")
}

// TestDiscoverRepoConfig_FoundAtGitBoundary verifies that a harvx.toml at the
// same level as the .git directory is returned (checked before stopping).
func TestDiscoverRepoConfig_FoundAtGitBoundary(t *testing.T) {
	t.Parallel()

	// Layout:
	//   repoRoot/
	//     .git/            <-- boundary
	//     harvx.toml       <-- should be found
	//     sub/             <-- start dir

	repoRoot := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755))
	configPath := filepath.Join(repoRoot, "harvx.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[profile.default]\n"), 0o644))

	sub := filepath.Join(repoRoot, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))

	got, err := DiscoverRepoConfig(sub)
	require.NoError(t, err)
	assertSamePath(t, configPath, got)
}

// TestDiscoverRepoConfig_ClosestWins verifies that when multiple harvx.toml
// files exist in the tree, the one closest to startDir is returned.
func TestDiscoverRepoConfig_ClosestWins(t *testing.T) {
	t.Parallel()

	// Layout:
	//   parent/
	//     harvx.toml       <-- further
	//     child/
	//       harvx.toml     <-- closer, should win

	parent := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(parent, "harvx.toml"), []byte("[profile.default]\n"), 0o644))

	child := filepath.Join(parent, "child")
	require.NoError(t, os.Mkdir(child, 0o755))
	childConfig := filepath.Join(child, "harvx.toml")
	require.NoError(t, os.WriteFile(childConfig, []byte("[profile.other]\n"), 0o644))

	got, err := DiscoverRepoConfig(child)
	require.NoError(t, err)
	assertSamePath(t, childConfig, got)
}

// TestDiscoverRepoConfig_DeepNesting verifies multi-level nesting without a
// .git boundary still finds the config in an ancestor directory.
func TestDiscoverRepoConfig_DeepNesting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "harvx.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[profile.default]\n"), 0o644))

	// Create 5 levels of nested subdirectories.
	current := root
	for i := range 5 {
		current = filepath.Join(current, "level"+string(rune('0'+i)))
		require.NoError(t, os.Mkdir(current, 0o755))
	}

	got, err := DiscoverRepoConfig(current)
	require.NoError(t, err)
	assertSamePath(t, configPath, got)
}

// ── DiscoverGlobalConfig ──────────────────────────────────────────────────────

// TestDiscoverGlobalConfig_FileExists verifies that an existing global config
// is returned correctly via XDG_CONFIG_HOME on non-Windows platforms, or
// via APPDATA on Windows.
func TestDiscoverGlobalConfig_FileExists(t *testing.T) {
	var configBase string

	if runtime.GOOS == "windows" {
		configBase = t.TempDir()
		t.Setenv("APPDATA", configBase)
	} else {
		configBase = t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", configBase)
	}

	configDir := filepath.Join(configBase, "harvx")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	configPath := filepath.Join(configDir, "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[profile.default]\n"), 0o644))

	got, err := DiscoverGlobalConfig()
	require.NoError(t, err)
	assert.Equal(t, configPath, got)
}

// TestDiscoverGlobalConfig_FileDoesNotExist verifies that an empty string is
// returned (not an error) when the global config file does not exist.
func TestDiscoverGlobalConfig_FileDoesNotExist(t *testing.T) {
	configBase := t.TempDir()

	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", configBase)
	} else {
		t.Setenv("XDG_CONFIG_HOME", configBase)
	}
	// Do NOT create harvx/config.toml under configBase

	got, err := DiscoverGlobalConfig()
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestDiscoverGlobalConfig_XDGConfigHome verifies that when XDG_CONFIG_HOME is
// set, the function uses it instead of the default ~/.config path.
func TestDiscoverGlobalConfig_XDGConfigHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG_CONFIG_HOME is not applicable on Windows")
	}

	xdgBase := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgBase)

	configDir := filepath.Join(xdgBase, "harvx")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	configPath := filepath.Join(configDir, "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[profile.default]\n"), 0o644))

	got, err := DiscoverGlobalConfig()
	require.NoError(t, err)
	assert.Equal(t, configPath, got)
}

// TestDiscoverGlobalConfig_XDGConfigHome_NoFile verifies that XDG_CONFIG_HOME
// is used but empty string is returned when the file doesn't exist there.
func TestDiscoverGlobalConfig_XDGConfigHome_NoFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG_CONFIG_HOME is not applicable on Windows")
	}

	xdgBase := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgBase)
	// No harvx/config.toml under xdgBase

	got, err := DiscoverGlobalConfig()
	require.NoError(t, err)
	assert.Empty(t, got)
}

// ── Resolver integration ──────────────────────────────────────────────────────

// TestResolve_AutoDiscoversRepoConfig verifies that when TargetDir points to a
// subdirectory, Resolve walks up and discovers the harvx.toml in a parent.
func TestResolve_AutoDiscoversRepoConfig(t *testing.T) {
	clearHarvxEnv(t)

	// Create a parent dir with harvx.toml and a child subdir as TargetDir.
	parent := t.TempDir()
	writeTomlFile(t, parent, "harvx.toml", `
[profile.default]
format = "xml"
max_tokens = 55000
`)

	child := filepath.Join(parent, "src")
	require.NoError(t, os.Mkdir(child, 0o755))

	rc, err := Resolve(ResolveOptions{
		TargetDir:        child,
		GlobalConfigPath: filepath.Join(parent, "nonexistent.toml"),
	})

	require.NoError(t, err)
	assert.Equal(t, "xml", rc.Profile.Format,
		"auto-discovered parent harvx.toml must be loaded")
	assert.Equal(t, 55000, rc.Profile.MaxTokens)
	assert.Equal(t, SourceRepo, rc.Sources["format"])
}

// TestResolve_AutoDiscovery_StopsAtGitBoundary verifies that the resolver does
// not load a harvx.toml that is above the .git boundary.
func TestResolve_AutoDiscovery_StopsAtGitBoundary(t *testing.T) {
	clearHarvxEnv(t)

	// Layout:
	//   grandparent/
	//     harvx.toml       <-- must NOT be loaded
	//     repoRoot/
	//       .git/          <-- boundary
	//       src/           <-- TargetDir

	grandparent := t.TempDir()
	writeTomlFile(t, grandparent, "harvx.toml", `
[profile.default]
format = "xml"
`)

	repoRoot := filepath.Join(grandparent, "repo")
	require.NoError(t, os.Mkdir(repoRoot, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755))

	src := filepath.Join(repoRoot, "src")
	require.NoError(t, os.Mkdir(src, 0o755))

	rc, err := Resolve(ResolveOptions{
		TargetDir:        src,
		GlobalConfigPath: filepath.Join(grandparent, "nonexistent.toml"),
	})

	require.NoError(t, err)
	assert.Equal(t, DefaultProfile().Format, rc.Profile.Format,
		"harvx.toml above .git boundary must not be loaded")
}

// ── Additional DiscoverRepoConfig edge cases ──────────────────────────────────

// TestDiscoverRepoConfig_FoundTwoLevelsUp verifies that a harvx.toml exactly
// two parent directories above the start directory is found.
func TestDiscoverRepoConfig_FoundTwoLevelsUp(t *testing.T) {
	t.Parallel()

	// Layout:
	//   grandparent/
	//     harvx.toml       <-- target
	//     parent/
	//       child/         <-- start dir

	grandparent := t.TempDir()
	configPath := filepath.Join(grandparent, "harvx.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[profile.default]\n"), 0o644))

	parent := filepath.Join(grandparent, "parent")
	require.NoError(t, os.Mkdir(parent, 0o755))

	child := filepath.Join(parent, "child")
	require.NoError(t, os.Mkdir(child, 0o755))

	got, err := DiscoverRepoConfig(child)
	require.NoError(t, err)
	assertSamePath(t, configPath, got, "harvx.toml exactly two levels up must be found")
}

// TestDiscoverRepoConfig_MaxDepthPreventsRunaway verifies that the function
// returns an empty string (not an error) when the directory tree exceeds
// maxSearchDepth levels with no harvx.toml present.
func TestDiscoverRepoConfig_MaxDepthPreventsRunaway(t *testing.T) {
	t.Parallel()

	// Build a directory chain of maxSearchDepth+5 levels to ensure the cap
	// kicks in before we reach the root.
	root := t.TempDir()

	current := root
	for i := 0; i < maxSearchDepth+5; i++ {
		next := filepath.Join(current, "level")
		require.NoError(t, os.Mkdir(next, 0o755))
		current = next
	}
	// current is now maxSearchDepth+5 levels below root; no harvx.toml anywhere.

	got, err := DiscoverRepoConfig(current)
	require.NoError(t, err, "max depth must not produce an error")
	assert.Empty(t, got, "no harvx.toml found within maxSearchDepth must return empty string")
}

// TestDiscoverRepoConfig_MaxDepthFindsConfigJustAtLimit verifies that a
// harvx.toml exactly at depth maxSearchDepth-1 (the last allowed step) is
// still found.
func TestDiscoverRepoConfig_MaxDepthFindsConfigJustAtLimit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "harvx.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[profile.default]\n"), 0o644))

	// Build exactly maxSearchDepth-1 subdirectories so that the root is the
	// last directory checked before the depth limit is hit.
	current := root
	for i := 0; i < maxSearchDepth-1; i++ {
		next := filepath.Join(current, "level")
		require.NoError(t, os.Mkdir(next, 0o755))
		current = next
	}

	got, err := DiscoverRepoConfig(current)
	require.NoError(t, err)
	assertSamePath(t, configPath, got,
		"harvx.toml exactly maxSearchDepth-1 levels above startDir must be found")
}

// TestDiscoverRepoConfig_SymlinkResolution verifies that a symlink to a
// directory is resolved before walking so that the canonical path is used.
func TestDiscoverRepoConfig_SymlinkResolution(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows")
	}

	// Layout:
	//   real/
	//     harvx.toml       <-- will be found
	//     sub/             <-- real subdir
	//   link/              <-- symlink → real/sub

	real := t.TempDir()
	configPath := filepath.Join(real, "harvx.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[profile.default]\n"), 0o644))

	sub := filepath.Join(real, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))

	// Create a symlink to real/sub in a separate temp dir so the link
	// itself does not live inside the real tree.
	linkBase := t.TempDir()
	link := filepath.Join(linkBase, "link")
	require.NoError(t, os.Symlink(sub, link))

	// Starting from the symlinked path should still discover real/harvx.toml
	// because EvalSymlinks resolves it to real/sub first.
	got, err := DiscoverRepoConfig(link)
	require.NoError(t, err)
	// The returned path is the canonical (resolved) path.
	resolvedConfig, err := filepath.EvalSymlinks(configPath)
	require.NoError(t, err)
	assert.Equal(t, resolvedConfig, got,
		"symlink resolution must yield the canonical harvx.toml path")
}

// TestDiscoverRepoConfig_PermissionDeniedParent verifies that encountering an
// intermediate parent directory with restricted permissions does not cause an
// error. The implementation uses simple os.Stat calls and string operations to
// walk upward; a permission-denied stat on an intermediate level is silently
// skipped and the walk continues to the grandparent.
func TestDiscoverRepoConfig_PermissionDeniedParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission model differs on Windows; chmod 000 has no effect")
	}

	// Skip when running as root because permission checks are bypassed.
	if os.Getuid() == 0 {
		t.Skip("running as root; permission restrictions are not enforced")
	}

	// Layout:
	//   outer/
	//     harvx.toml       <-- will be found (stat on outer/* is permitted)
	//     restricted/      <-- mode 0o000: stat on entries inside is denied
	//       child/         <-- startDir

	outer := t.TempDir()
	outerConfig := filepath.Join(outer, "harvx.toml")
	require.NoError(t, os.WriteFile(outerConfig, []byte("[profile.default]\n"), 0o644))

	restricted := filepath.Join(outer, "restricted")
	require.NoError(t, os.Mkdir(restricted, 0o755))

	child := filepath.Join(restricted, "child")
	require.NoError(t, os.Mkdir(child, 0o755))

	// Revoke execute (traverse) permission on the intermediate directory so
	// stat calls for paths inside it fail with permission denied.
	require.NoError(t, os.Chmod(restricted, 0o000))
	t.Cleanup(func() {
		// Restore permissions so t.TempDir cleanup can succeed.
		_ = os.Chmod(restricted, 0o755)
	})

	// The function must not return an error. The Stat for
	// restricted/harvx.toml silently fails (condition statErr == nil is
	// false), the walker moves to the grandparent (outer), and finds
	// outer/harvx.toml.
	got, err := DiscoverRepoConfig(child)
	require.NoError(t, err, "permission denied on an intermediate directory must not cause an error")
	assertSamePath(t, outerConfig, got,
		"walker must skip the permission-denied intermediate level and find the config in the grandparent")
}

// TestDiscoverRepoConfig_NonExistentStartDir verifies that passing a
// subdirectory that does not exist within a known temp base returns gracefully
// (no crash, no error). The EvalSymlinks call in the implementation will fail
// and fall back to the abs path; subsequent Stat calls also fail but the
// function continues walking upward until it hits the temp root (which may or
// may not have a harvx.toml). We only assert no error is returned.
func TestDiscoverRepoConfig_NonExistentStartDir(t *testing.T) {
	t.Parallel()

	// Use a known temp base + a nonexistent subpath so EvalSymlinks fails.
	base := t.TempDir()
	nonExistent := filepath.Join(base, "does", "not", "exist")

	_, err := DiscoverRepoConfig(nonExistent)
	// The implementation must not return an error regardless of whether
	// it finds or misses a harvx.toml above the nonexistent path.
	require.NoError(t, err)
}

// TestDiscoverRepoConfig_FilesystemRootNoConfig verifies that the function
// terminates at the filesystem root without panicking when no harvx.toml
// exists anywhere in a shallow temp tree.
func TestDiscoverRepoConfig_FilesystemRootNoConfig(t *testing.T) {
	t.Parallel()

	// t.TempDir() is typically a few levels from the FS root on CI and
	// local machines. This test validates correct termination via the
	// "parent == dir" guard in the implementation.
	dir := t.TempDir()

	got, err := DiscoverRepoConfig(dir)
	require.NoError(t, err)
	// Either empty (no harvx.toml in the real FS above it) or non-empty
	// (there is a harvx.toml somewhere above — we only assert no error).
	_ = got // result depends on the host's filesystem; only safety matters
}

// ── Additional DiscoverGlobalConfig edge cases ────────────────────────────────

// TestDiscoverGlobalConfig_ReturnsNonEmptyWhenFileExists verifies explicitly
// that the returned string is non-empty when the global config file is present.
func TestDiscoverGlobalConfig_ReturnsNonEmptyWhenFileExists(t *testing.T) {
	// Not parallel: uses t.Setenv which mutates process-wide env.
	configBase := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", configBase)
	} else {
		t.Setenv("XDG_CONFIG_HOME", configBase)
	}

	configDir := filepath.Join(configBase, "harvx")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	configPath := filepath.Join(configDir, "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[profile.default]\n"), 0o644))

	got, err := DiscoverGlobalConfig()
	require.NoError(t, err)
	assert.NotEmpty(t, got, "must return non-empty path when global config exists")
	assert.Equal(t, configPath, got)
}

// TestDiscoverGlobalConfig_DefaultDotConfig verifies that on non-Windows
// platforms, when XDG_CONFIG_HOME is not set, the function uses ~/.config as
// the base directory. This is verified by placing a config there via the
// home-directory path derivation.
func TestDiscoverGlobalConfig_DefaultDotConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("~/.config is not applicable on Windows")
	}

	// Point HOME to a temp dir so we control what ~/.config looks like.
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("XDG_CONFIG_HOME", "") // ensure XDG is unset

	configDir := filepath.Join(fakeHome, ".config", "harvx")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	configPath := filepath.Join(configDir, "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[profile.default]\n"), 0o644))

	got, err := DiscoverGlobalConfig()
	require.NoError(t, err)
	assert.Equal(t, configPath, got,
		"~/.config/harvx/config.toml must be returned when XDG_CONFIG_HOME is unset")
}

// TestDiscoverGlobalConfig_XDGTakesPrecedenceOverHome verifies that when both
// XDG_CONFIG_HOME and a ~/.config-based file exist, XDG_CONFIG_HOME wins.
func TestDiscoverGlobalConfig_XDGTakesPrecedenceOverHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG_CONFIG_HOME is not applicable on Windows")
	}

	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	// Create a config in the ~/.config fallback path.
	homeConfigDir := filepath.Join(fakeHome, ".config", "harvx")
	require.NoError(t, os.MkdirAll(homeConfigDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(homeConfigDir, "config.toml"),
		[]byte("[profile.default]\nformat = \"markdown\"\n"),
		0o644,
	))

	// Create a config in the XDG path (different content).
	xdgBase := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgBase)
	xdgConfigDir := filepath.Join(xdgBase, "harvx")
	require.NoError(t, os.MkdirAll(xdgConfigDir, 0o755))
	xdgConfigPath := filepath.Join(xdgConfigDir, "config.toml")
	require.NoError(t, os.WriteFile(xdgConfigPath, []byte("[profile.default]\nformat = \"xml\"\n"), 0o644))

	got, err := DiscoverGlobalConfig()
	require.NoError(t, err)
	assert.Equal(t, xdgConfigPath, got,
		"XDG_CONFIG_HOME must take precedence over ~/.config")
}

// ── Additional resolver integration tests ────────────────────────────────────

// TestResolve_ProfileFile_SkipsAutoDiscovery verifies explicitly that when
// ProfileFile is set, DiscoverRepoConfig is not invoked — even if a harvx.toml
// exists in the TargetDir that would conflict with the standalone file.
func TestResolve_ProfileFile_SkipsAutoDiscovery(t *testing.T) {
	clearHarvxEnv(t)

	// Repo dir has harvx.toml with format=xml.
	repoDir := t.TempDir()
	writeTomlFile(t, repoDir, "harvx.toml", `
[profile.default]
format = "xml"
max_tokens = 999999
`)

	// Standalone profile with format=markdown.
	profileDir := t.TempDir()
	standaloneFile := writeTomlFile(t, profileDir, "standalone.toml", `
[profile.default]
format = "markdown"
max_tokens = 64000
`)

	rc, err := Resolve(ResolveOptions{
		TargetDir:        repoDir,
		ProfileFile:      standaloneFile,
		GlobalConfigPath: filepath.Join(repoDir, "nonexistent.toml"),
	})

	require.NoError(t, err)
	// The standalone file's values must win; repo harvx.toml is ignored.
	assert.Equal(t, "markdown", rc.Profile.Format,
		"--profile-file must prevent auto-discovery of harvx.toml")
	assert.Equal(t, 64000, rc.Profile.MaxTokens,
		"max_tokens from standalone file must be used, not from harvx.toml")
	assert.Equal(t, SourceRepo, rc.Sources["format"],
		"standalone profile file is still attributed as SourceRepo")
}

// TestResolve_DirFlagRespectedAsStartDirectory verifies that TargetDir (mapped
// from the --dir flag) is used as the starting directory for auto-discovery,
// not the process working directory.
func TestResolve_DirFlagRespectedAsStartDirectory(t *testing.T) {
	clearHarvxEnv(t)

	// Layout:
	//   dirA/
	//     harvx.toml  format=xml     <-- pointed to by TargetDir
	//   dirB/                         <-- another dir (not used)

	dirA := t.TempDir()
	writeTomlFile(t, dirA, "harvx.toml", `
[profile.default]
format = "xml"
max_tokens = 77000
`)

	dirB := t.TempDir() // no harvx.toml

	rc, err := Resolve(ResolveOptions{
		TargetDir:        dirA,
		GlobalConfigPath: filepath.Join(dirB, "nonexistent.toml"),
	})

	require.NoError(t, err)
	assert.Equal(t, "xml", rc.Profile.Format,
		"harvx.toml in TargetDir (--dir) must be discovered and loaded")
	assert.Equal(t, 77000, rc.Profile.MaxTokens)
	assert.Equal(t, SourceRepo, rc.Sources["format"])
}

// TestResolve_DirFlagDefault_UsesCurrentDir verifies that when TargetDir is
// empty, the resolver defaults to "." without error. It should not panic and
// should return the default profile (since there is no harvx.toml in "." during
// test execution — or if one exists, it loads fine).
func TestResolve_DirFlagDefault_UsesCurrentDir(t *testing.T) {
	clearHarvxEnv(t)

	// We cannot guarantee there is no harvx.toml in "." during test runs, so
	// we only assert that no error is returned and a valid config is produced.
	rc, err := Resolve(ResolveOptions{
		TargetDir:        "", // empty → defaults to "."
		GlobalConfigPath: filepath.Join(t.TempDir(), "nonexistent.toml"),
	})

	require.NoError(t, err)
	require.NotNil(t, rc)
	require.NotNil(t, rc.Profile)
	assert.NotEmpty(t, rc.ProfileName)
}

// TestResolve_AlwaysAttemptsGlobalConfig verifies that even when no repo
// config exists, the resolver still loads the global config.
func TestResolve_AlwaysAttemptsGlobalConfig(t *testing.T) {
	clearHarvxEnv(t)

	globalDir := t.TempDir()
	globalPath := writeTomlFile(t, globalDir, "config.toml", `
[profile.default]
format = "xml"
max_tokens = 88000
`)

	emptyRepoDir := t.TempDir() // no harvx.toml

	rc, err := Resolve(ResolveOptions{
		TargetDir:        emptyRepoDir,
		GlobalConfigPath: globalPath,
	})

	require.NoError(t, err)
	assert.Equal(t, "xml", rc.Profile.Format,
		"global config must be loaded even when no repo harvx.toml exists")
	assert.Equal(t, 88000, rc.Profile.MaxTokens)
	assert.Equal(t, SourceGlobal, rc.Sources["format"])
	assert.Equal(t, SourceGlobal, rc.Sources["max_tokens"])
}

// ── Table-driven summary for DiscoverRepoConfig ───────────────────────────────

// TestDiscoverRepoConfig_TableDriven exercises a range of directory layouts
// in a table-driven style to validate correct config discovery behaviour.
func TestDiscoverRepoConfig_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T) (startDir, wantConfig string)
		wantEmpty  bool
	}{
		{
			name: "config in start dir",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				dir := t.TempDir()
				cfg := filepath.Join(dir, "harvx.toml")
				require.NoError(t, os.WriteFile(cfg, []byte("[profile.default]\n"), 0o644))
				return dir, cfg
			},
		},
		{
			name: "config one level up",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				parent := t.TempDir()
				cfg := filepath.Join(parent, "harvx.toml")
				require.NoError(t, os.WriteFile(cfg, []byte("[profile.default]\n"), 0o644))
				child := filepath.Join(parent, "sub")
				require.NoError(t, os.Mkdir(child, 0o755))
				return child, cfg
			},
		},
		{
			name: "config two levels up",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				gp := t.TempDir()
				cfg := filepath.Join(gp, "harvx.toml")
				require.NoError(t, os.WriteFile(cfg, []byte("[profile.default]\n"), 0o644))
				p := filepath.Join(gp, "p")
				require.NoError(t, os.Mkdir(p, 0o755))
				c := filepath.Join(p, "c")
				require.NoError(t, os.Mkdir(c, 0o755))
				return c, cfg
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
			name:      "git boundary stops before config in grandparent",
			wantEmpty: true,
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				gp := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(gp, "harvx.toml"), []byte("[profile.default]\n"), 0o644,
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
				cfg := filepath.Join(root, "harvx.toml")
				require.NoError(t, os.WriteFile(cfg, []byte("[profile.default]\n"), 0o644))
				child := filepath.Join(root, "pkg")
				require.NoError(t, os.Mkdir(child, 0o755))
				return child, cfg
			},
		},
		{
			name: "closest config wins when multiple exist",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				parent := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(parent, "harvx.toml"), []byte("[profile.default]\n"), 0o644,
				))
				child := filepath.Join(parent, "child")
				require.NoError(t, os.Mkdir(child, 0o755))
				childCfg := filepath.Join(child, "harvx.toml")
				require.NoError(t, os.WriteFile(childCfg, []byte("[profile.other]\n"), 0o644))
				return child, childCfg
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			startDir, wantConfig := tt.setup(t)

			got, err := DiscoverRepoConfig(startDir)
			require.NoError(t, err)

			if tt.wantEmpty {
				assert.Empty(t, got)
			} else {
				assertSamePath(t, wantConfig, got)
			}
		})
	}
}
