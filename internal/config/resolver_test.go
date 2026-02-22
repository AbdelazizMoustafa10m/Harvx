package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// writeTomlFile writes content to a temporary TOML file and returns its path.
func writeTomlFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// ── Layer 1: defaults ─────────────────────────────────────────────────────────

// TestResolve_DefaultsOnly verifies that when no config files, env vars, or
// CLI flags are provided, the resolved profile equals DefaultProfile().
func TestResolve_DefaultsOnly(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	rc, err := Resolve(ResolveOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nonexistent-global.toml"),
	})

	require.NoError(t, err)
	require.NotNil(t, rc)

	want := DefaultProfile()
	assert.Equal(t, want.Format, rc.Profile.Format)
	assert.Equal(t, want.MaxTokens, rc.Profile.MaxTokens)
	assert.Equal(t, want.Tokenizer, rc.Profile.Tokenizer)
	assert.Equal(t, want.Output, rc.Profile.Output)
	assert.Equal(t, want.Compression, rc.Profile.Compression)
	assert.Equal(t, want.Redaction, rc.Profile.Redaction)

	assert.Equal(t, "default", rc.ProfileName)
}

// TestResolve_DefaultsOnly_SourceTracking verifies that all field sources are
// SourceDefault when no overriding layers are present.
func TestResolve_DefaultsOnly_SourceTracking(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	rc, err := Resolve(ResolveOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nonexistent-global.toml"),
	})

	require.NoError(t, err)

	for key, src := range rc.Sources {
		assert.Equal(t, SourceDefault, src,
			"field %q must have SourceDefault when only defaults are loaded", key)
	}
}

// ── Layer 2: global config ────────────────────────────────────────────────────

// TestResolve_GlobalConfigOverridesDefaults verifies that a global config file
// overrides the default values for the specified fields.
func TestResolve_GlobalConfigOverridesDefaults(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	globalPath := writeTomlFile(t, dir, "global.toml", `
[profile.default]
format = "xml"
max_tokens = 100000
output = "global-output.md"
`)

	rc, err := Resolve(ResolveOptions{
		TargetDir:        t.TempDir(), // empty target dir → no repo config
		GlobalConfigPath: globalPath,
	})

	require.NoError(t, err)
	assert.Equal(t, "xml", rc.Profile.Format)
	assert.Equal(t, 100000, rc.Profile.MaxTokens)
	assert.Equal(t, "global-output.md", rc.Profile.Output)

	// Fields set by global config must be tracked as SourceGlobal.
	assert.Equal(t, SourceGlobal, rc.Sources["format"])
	assert.Equal(t, SourceGlobal, rc.Sources["max_tokens"])
	assert.Equal(t, SourceGlobal, rc.Sources["output"])

	// Fields not overridden must remain SourceDefault.
	assert.Equal(t, SourceDefault, rc.Sources["tokenizer"])
}

// TestResolve_GlobalConfig_MissingFile verifies that a missing global config
// is silently ignored and the pipeline continues with defaults.
func TestResolve_GlobalConfig_MissingFile(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	rc, err := Resolve(ResolveOptions{
		TargetDir:        dir,
		GlobalConfigPath: "/nonexistent/path/config.toml",
	})

	require.NoError(t, err)
	assert.Equal(t, DefaultProfile().Format, rc.Profile.Format)
}

// ── Layer 3: repo config ──────────────────────────────────────────────────────

// TestResolve_RepoConfigOverridesGlobal verifies that repo config values take
// precedence over global config values.
func TestResolve_RepoConfigOverridesGlobal(t *testing.T) {
	clearHarvxEnv(t)

	globalDir := t.TempDir()
	globalPath := writeTomlFile(t, globalDir, "global.toml", `
[profile.default]
format = "markdown"
max_tokens = 100000
output = "global-output.md"
`)

	repoDir := t.TempDir()
	writeTomlFile(t, repoDir, "harvx.toml", `
[profile.default]
format = "xml"
max_tokens = 200000
output = "repo-output.md"
compression = true
`)

	rc, err := Resolve(ResolveOptions{
		TargetDir:        repoDir,
		GlobalConfigPath: globalPath,
	})

	require.NoError(t, err)
	assert.Equal(t, "xml", rc.Profile.Format)
	assert.Equal(t, 200000, rc.Profile.MaxTokens)
	assert.Equal(t, "repo-output.md", rc.Profile.Output)
	assert.True(t, rc.Profile.Compression)

	// Fields overridden by repo config must be tracked as SourceRepo.
	assert.Equal(t, SourceRepo, rc.Sources["format"])
	assert.Equal(t, SourceRepo, rc.Sources["max_tokens"])
	assert.Equal(t, SourceRepo, rc.Sources["output"])
	assert.Equal(t, SourceRepo, rc.Sources["compression"])

	// Tokenizer was only set in defaults, not overridden by global or repo.
	assert.Equal(t, SourceDefault, rc.Sources["tokenizer"])
}

// TestResolve_RepoConfig_MissingFile verifies that a missing harvx.toml is
// silently ignored.
func TestResolve_RepoConfig_MissingFile(t *testing.T) {
	clearHarvxEnv(t)

	emptyDir := t.TempDir()
	rc, err := Resolve(ResolveOptions{
		TargetDir:        emptyDir,
		GlobalConfigPath: filepath.Join(emptyDir, "nonexistent.toml"),
	})

	require.NoError(t, err)
	assert.Equal(t, DefaultProfile().Format, rc.Profile.Format)
}

// ── Layer 3 alt: standalone profile file ──────────────────────────────────────

// TestResolve_ProfileFile_SkipsRepoConfig verifies that when ProfileFile is
// set, the repo config (harvx.toml) is not loaded.
func TestResolve_ProfileFile_SkipsRepoConfig(t *testing.T) {
	clearHarvxEnv(t)

	// Repo dir with a harvx.toml that sets format=xml.
	repoDir := t.TempDir()
	writeTomlFile(t, repoDir, "harvx.toml", `
[profile.default]
format = "xml"
`)

	// Standalone profile file that sets format=markdown.
	profileDir := t.TempDir()
	profileFile := writeTomlFile(t, profileDir, "myprofile.toml", `
[profile.default]
format = "markdown"
max_tokens = 64000
`)

	rc, err := Resolve(ResolveOptions{
		TargetDir:        repoDir,   // has harvx.toml with xml
		ProfileFile:      profileFile, // standalone file wins
		GlobalConfigPath: filepath.Join(repoDir, "nonexistent.toml"),
	})

	require.NoError(t, err)
	assert.Equal(t, "markdown", rc.Profile.Format,
		"standalone profile file must override repo config")
	assert.Equal(t, 64000, rc.Profile.MaxTokens)
}

// ── Layer 4: environment variables ───────────────────────────────────────────

// TestResolve_EnvOverridesRepo verifies that HARVX_* env vars override repo
// config values.
func TestResolve_EnvOverridesRepo(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvFormat, "xml")
	t.Setenv(EnvMaxTokens, "99000")

	repoDir := t.TempDir()
	writeTomlFile(t, repoDir, "harvx.toml", `
[profile.default]
format = "markdown"
max_tokens = 50000
`)

	rc, err := Resolve(ResolveOptions{
		TargetDir:        repoDir,
		GlobalConfigPath: filepath.Join(repoDir, "nonexistent.toml"),
	})

	require.NoError(t, err)
	assert.Equal(t, "xml", rc.Profile.Format)
	assert.Equal(t, 99000, rc.Profile.MaxTokens)

	assert.Equal(t, SourceEnv, rc.Sources["format"])
	assert.Equal(t, SourceEnv, rc.Sources["max_tokens"])
}

// TestResolve_EnvProfile_SelectsNamedProfile verifies that HARVX_PROFILE
// selects a non-default profile from the config file.
func TestResolve_EnvProfile_SelectsNamedProfile(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvProfile, "myprofile")

	dir := t.TempDir()
	writeTomlFile(t, dir, "harvx.toml", `
[profile.default]
format = "markdown"

[profile.myprofile]
format = "xml"
max_tokens = 77000
`)

	rc, err := Resolve(ResolveOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nonexistent.toml"),
	})

	require.NoError(t, err)
	assert.Equal(t, "xml", rc.Profile.Format)
	assert.Equal(t, 77000, rc.Profile.MaxTokens)
	assert.Equal(t, "myprofile", rc.ProfileName)
}

// ── Layer 5: CLI flags ────────────────────────────────────────────────────────

// TestResolve_CLIFlagsOverrideEnv verifies that CLI flags have the highest
// precedence, overriding even HARVX_* env vars.
func TestResolve_CLIFlagsOverrideEnv(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvFormat, "xml")

	dir := t.TempDir()
	rc, err := Resolve(ResolveOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nonexistent.toml"),
		CLIFlags: map[string]any{
			"format": "markdown",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "markdown", rc.Profile.Format,
		"CLI flag must override HARVX_FORMAT env var")
	assert.Equal(t, SourceFlag, rc.Sources["format"])
}

// TestResolve_CLIFlags_OverrideAllLayers verifies that CLI flags win over
// defaults, global config, repo config, and env vars simultaneously.
func TestResolve_CLIFlags_OverrideAllLayers(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvFormat, "xml")
	t.Setenv(EnvMaxTokens, "50000")

	globalDir := t.TempDir()
	globalPath := writeTomlFile(t, globalDir, "global.toml", `
[profile.default]
format = "markdown"
max_tokens = 100000
`)

	repoDir := t.TempDir()
	writeTomlFile(t, repoDir, "harvx.toml", `
[profile.default]
format = "plain"
max_tokens = 200000
`)

	rc, err := Resolve(ResolveOptions{
		TargetDir:        repoDir,
		GlobalConfigPath: globalPath,
		CLIFlags: map[string]any{
			"format":     "xml",
			"max_tokens": 42000,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "xml", rc.Profile.Format)
	assert.Equal(t, 42000, rc.Profile.MaxTokens)

	assert.Equal(t, SourceFlag, rc.Sources["format"])
	assert.Equal(t, SourceFlag, rc.Sources["max_tokens"])
}

// ── Target presets ────────────────────────────────────────────────────────────

// TestResolve_TargetPreset_AppliedBeforeCLIFlags verifies that when a target
// is set via env var, the preset is applied, but CLI flags can still override
// the preset values.
func TestResolve_TargetPreset_AppliedBeforeCLIFlags(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvTarget, "claude") // preset would set format=xml, maxTokens=200000

	dir := t.TempDir()
	rc, err := Resolve(ResolveOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nonexistent.toml"),
		CLIFlags: map[string]any{
			"format": "markdown", // CLI overrides preset
		},
	})

	require.NoError(t, err)
	// CLI flag wins over target preset.
	assert.Equal(t, "markdown", rc.Profile.Format,
		"CLI --format must override target preset format")
	// MaxTokens from claude preset (200000) should still apply since no CLI flag for it.
	assert.Equal(t, 200000, rc.Profile.MaxTokens,
		"target preset MaxTokens must apply when no CLI flag overrides it")
}

// TestResolve_TargetPreset_ClaudeNoOverride verifies that when target=claude
// is set in config and no CLI flags override format/max_tokens, the preset
// values are used.
func TestResolve_TargetPreset_ClaudeNoOverride(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	writeTomlFile(t, dir, "harvx.toml", `
[profile.default]
target = "claude"
`)

	rc, err := Resolve(ResolveOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nonexistent.toml"),
	})

	require.NoError(t, err)
	assert.Equal(t, "xml", rc.Profile.Format)
	assert.Equal(t, 200000, rc.Profile.MaxTokens)
}

// ── Profile name resolution ───────────────────────────────────────────────────

// TestResolve_ProfileName_ExplicitOption verifies that ProfileName in
// ResolveOptions takes precedence over HARVX_PROFILE.
func TestResolve_ProfileName_ExplicitOption(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvProfile, "envprofile")

	dir := t.TempDir()
	writeTomlFile(t, dir, "harvx.toml", `
[profile.default]
format = "markdown"

[profile.envprofile]
format = "xml"

[profile.explicit]
format = "plain"
`)

	rc, err := Resolve(ResolveOptions{
		ProfileName:      "explicit",
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nonexistent.toml"),
	})

	require.NoError(t, err)
	assert.Equal(t, "explicit", rc.ProfileName)
	assert.Equal(t, "plain", rc.Profile.Format)
}

// TestResolve_ProfileName_DefaultFallback verifies that when neither
// ProfileName nor HARVX_PROFILE is set, "default" is used.
func TestResolve_ProfileName_DefaultFallback(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	rc, err := Resolve(ResolveOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nonexistent.toml"),
	})

	require.NoError(t, err)
	assert.Equal(t, "default", rc.ProfileName)
}

// ── Error cases ───────────────────────────────────────────────────────────────

// TestResolve_InvalidRepoConfig_ReturnsError verifies that a malformed
// harvx.toml causes Resolve to return an error.
func TestResolve_InvalidRepoConfig_ReturnsError(t *testing.T) {
	clearHarvxEnv(t)

	repoDir := t.TempDir()
	writeTomlFile(t, repoDir, "harvx.toml", `[broken toml`)

	_, err := Resolve(ResolveOptions{
		TargetDir:        repoDir,
		GlobalConfigPath: filepath.Join(repoDir, "nonexistent.toml"),
	})

	require.Error(t, err)
}

// TestResolve_InvalidGlobalConfig_ReturnsError verifies that a malformed
// global config causes Resolve to return an error.
func TestResolve_InvalidGlobalConfig_ReturnsError(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	globalPath := writeTomlFile(t, dir, "global.toml", `[broken`)

	_, err := Resolve(ResolveOptions{
		TargetDir:        t.TempDir(),
		GlobalConfigPath: globalPath,
	})

	require.Error(t, err)
}

// TestResolve_ProfileFile_ProfileNotFound_ReturnsError verifies that when a
// standalone ProfileFile is given but the profile name is not found, an error
// is returned.
func TestResolve_ProfileFile_ProfileNotFound_ReturnsError(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	profileFile := writeTomlFile(t, dir, "myprofile.toml", `
[profile.other]
format = "xml"
`)

	_, err := Resolve(ResolveOptions{
		ProfileName:      "missing",
		ProfileFile:      profileFile,
		GlobalConfigPath: filepath.Join(dir, "nonexistent.toml"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

// ── Full pipeline integration ─────────────────────────────────────────────────

// TestResolve_FullPipeline verifies all 5 layers interact correctly with the
// correct precedence order: default < global < repo < env < flag.
func TestResolve_FullPipeline(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvTokenizer, "o200k_base") // env overrides repo
	t.Setenv(EnvOutput, "env-output.md")

	globalDir := t.TempDir()
	globalPath := writeTomlFile(t, globalDir, "global.toml", `
[profile.default]
format = "markdown"
max_tokens = 100000
output = "global-output.md"
tokenizer = "cl100k_base"
`)

	repoDir := t.TempDir()
	writeTomlFile(t, repoDir, "harvx.toml", `
[profile.default]
format = "xml"
max_tokens = 150000
output = "repo-output.md"
tokenizer = "cl100k_base"
`)

	rc, err := Resolve(ResolveOptions{
		TargetDir:        repoDir,
		GlobalConfigPath: globalPath,
		CLIFlags: map[string]any{
			"max_tokens": 42000, // CLI wins over everything
		},
	})

	require.NoError(t, err)

	// format: repo (xml) wins over global (markdown)
	assert.Equal(t, "xml", rc.Profile.Format)
	assert.Equal(t, SourceRepo, rc.Sources["format"])

	// max_tokens: CLI (42000) wins over repo (150000)
	assert.Equal(t, 42000, rc.Profile.MaxTokens)
	assert.Equal(t, SourceFlag, rc.Sources["max_tokens"])

	// output: env (env-output.md) wins over repo (repo-output.md)
	assert.Equal(t, "env-output.md", rc.Profile.Output)
	assert.Equal(t, SourceEnv, rc.Sources["output"])

	// tokenizer: env (o200k_base) wins over repo (cl100k_base)
	assert.Equal(t, "o200k_base", rc.Profile.Tokenizer)
	assert.Equal(t, SourceEnv, rc.Sources["tokenizer"])
}

// TestResolve_ReturnsNewInstanceEachCall verifies that each Resolve call
// returns a fresh ResolvedConfig (no shared state between calls).
func TestResolve_ReturnsNewInstanceEachCall(t *testing.T) {
	// Not parallel: mutates environment via clearHarvxEnv.
	clearHarvxEnv(t)

	dir := t.TempDir()
	opts := ResolveOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nonexistent.toml"),
	}

	rc1, err := Resolve(opts)
	require.NoError(t, err)

	rc2, err := Resolve(opts)
	require.NoError(t, err)

	// Mutate rc1; rc2 must not be affected.
	rc1.Profile.Format = "mutated"
	rc1.Sources["format"] = SourceFlag

	assert.NotEqual(t, "mutated", rc2.Profile.Format,
		"mutating rc1 must not affect rc2")
	assert.NotEqual(t, SourceFlag, rc2.Sources["format"],
		"mutating rc1.Sources must not affect rc2.Sources")
}

// TestResolve_ProfileName_FromOpts verifies the ProfileName field in
// ResolvedConfig matches the resolved profile name.
func TestResolve_ProfileName_FromOpts(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	writeTomlFile(t, dir, "harvx.toml", `
[profile.myprofile]
format = "xml"
`)

	rc, err := Resolve(ResolveOptions{
		ProfileName:      "myprofile",
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nonexistent.toml"),
	})

	require.NoError(t, err)
	assert.Equal(t, "myprofile", rc.ProfileName)
}

// TestResolve_NonExistentProfile_ExplicitOpts returns an error when a
// non-default profile is explicitly requested but not found in any config.
func TestResolve_NonExistentProfile_ExplicitOpts(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	writeTomlFile(t, dir, "harvx.toml", `
[profile.default]
format = "markdown"

[profile.other]
format = "xml"
`)

	_, err := Resolve(ResolveOptions{
		ProfileName:      "nonexistent",
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nofile.toml"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

// TestResolve_NonExistentProfile_EnvVar returns an error when HARVX_PROFILE
// is set to a profile that does not exist in any config file.
func TestResolve_NonExistentProfile_EnvVar(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvProfile, "ghost")

	dir := t.TempDir()
	writeTomlFile(t, dir, "harvx.toml", `
[profile.default]
format = "markdown"
`)

	_, err := Resolve(ResolveOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "nofile.toml"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost")
}
