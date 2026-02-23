package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixturePath returns the path to an integration test fixture directory.
// Fixtures are located under <repo-root>/testdata/integration/profiles/.
// Since Go sets the test CWD to the package directory (internal/config/),
// we navigate up two levels to reach the repository root.
func fixturePath(t *testing.T, relPath string) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "integration", "profiles", relPath)
}

// nonexistentGlobal returns a path to a file that does not exist, suitable for
// use as GlobalConfigPath when the test wants to disable global config loading.
func nonexistentGlobal(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "nonexistent-global.toml")
}

// ── Scenario 1: defaults only ─────────────────────────────────────────────────

// TestIntegration_Scenario1_DefaultsOnly verifies that when no harvx.toml is
// present and no env vars or CLI flags are set, Resolve returns the built-in
// DefaultProfile values.
func TestIntegration_Scenario1_DefaultsOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clearHarvxEnv(t)

	rc, err := Resolve(ResolveOptions{
		TargetDir:        fixturePath(t, "scenario-1-defaults-only"),
		GlobalConfigPath: nonexistentGlobal(t),
	})

	require.NoError(t, err)
	require.NotNil(t, rc)

	want := DefaultProfile()
	assert.Equal(t, want.Format, rc.Profile.Format, "format must equal DefaultProfile")
	assert.Equal(t, want.MaxTokens, rc.Profile.MaxTokens, "max_tokens must equal DefaultProfile")
	assert.Equal(t, want.Tokenizer, rc.Profile.Tokenizer, "tokenizer must equal DefaultProfile")
	assert.Equal(t, want.Output, rc.Profile.Output, "output must equal DefaultProfile")
	assert.Equal(t, want.Compression, rc.Profile.Compression, "compression must equal DefaultProfile")
	assert.Equal(t, want.Redaction, rc.Profile.Redaction, "redaction must equal DefaultProfile")

	// Spot-check expected values directly for clarity.
	assert.Equal(t, "markdown", rc.Profile.Format)
	assert.Equal(t, 128000, rc.Profile.MaxTokens)
	assert.Equal(t, "cl100k_base", rc.Profile.Tokenizer)

	assert.Equal(t, "default", rc.ProfileName)
}

// ── Scenario 2: repo config only ──────────────────────────────────────────────

// TestIntegration_Scenario2_RepoConfig verifies that a harvx.toml in the
// target directory overrides the built-in defaults.
func TestIntegration_Scenario2_RepoConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clearHarvxEnv(t)

	rc, err := Resolve(ResolveOptions{
		TargetDir:        fixturePath(t, "scenario-2-repo-config"),
		GlobalConfigPath: nonexistentGlobal(t),
	})

	require.NoError(t, err)
	require.NotNil(t, rc)

	// The fixture sets max_tokens=50000 and format="markdown".
	assert.Equal(t, 50000, rc.Profile.MaxTokens, "repo harvx.toml must set MaxTokens=50000")
	assert.Equal(t, "markdown", rc.Profile.Format, "repo harvx.toml must set Format=markdown")

	// Tokenizer was not set in the repo config; it must still be the default.
	assert.Equal(t, DefaultProfile().Tokenizer, rc.Profile.Tokenizer,
		"tokenizer not in repo config must remain at default")

	// Source attribution: repo-set fields come from SourceRepo.
	assert.Equal(t, SourceRepo, rc.Sources["max_tokens"])
	assert.Equal(t, SourceRepo, rc.Sources["format"])
}

// ── Scenario 3: global config + repo config ────────────────────────────────────

// TestIntegration_Scenario3_GlobalPlusRepo verifies that the global config
// and the repo config merge correctly with repo taking precedence.
func TestIntegration_Scenario3_GlobalPlusRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clearHarvxEnv(t)

	scenarioDir := fixturePath(t, "scenario-3-global-plus-repo")

	rc, err := Resolve(ResolveOptions{
		TargetDir:        scenarioDir,
		GlobalConfigPath: filepath.Join(scenarioDir, "global.toml"),
	})

	require.NoError(t, err)
	require.NotNil(t, rc)

	// global.toml sets tokenizer="o200k_base"; repo harvx.toml sets max_tokens=100000.
	assert.Equal(t, "o200k_base", rc.Profile.Tokenizer,
		"tokenizer from global config must be applied")
	assert.Equal(t, 100000, rc.Profile.MaxTokens,
		"max_tokens from repo config must override global")

	// Source attribution.
	assert.Equal(t, SourceGlobal, rc.Sources["tokenizer"],
		"tokenizer must be attributed to global source")
	assert.Equal(t, SourceRepo, rc.Sources["max_tokens"],
		"max_tokens must be attributed to repo source")
}

// ── Scenario 4: profile inheritance ───────────────────────────────────────────

// TestIntegration_Scenario4_Inheritance verifies profile inheritance:
// child -> base -> default, verifying that each level gets the right values.
func TestIntegration_Scenario4_Inheritance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		profileName    string
		wantFormat     string
		wantMaxTokens  int
		wantTokenizer  string
	}{
		{
			profileName:   "default",
			wantFormat:    "markdown",
			wantMaxTokens: 128000,
			wantTokenizer: "cl100k_base",
		},
		{
			profileName:   "base",
			wantFormat:    "markdown", // inherited from default
			wantMaxTokens: 80000,     // overrides default
			wantTokenizer: "cl100k_base",
		},
		{
			profileName:   "child",
			wantFormat:    "xml",    // overrides base
			wantMaxTokens: 60000,   // overrides base
			wantTokenizer: "cl100k_base",
		},
	}

	for _, tt := range tests {
		t.Run(tt.profileName, func(t *testing.T) {
			clearHarvxEnv(t)

			rc, err := Resolve(ResolveOptions{
				ProfileName:      tt.profileName,
				TargetDir:        fixturePath(t, "scenario-4-inheritance"),
				GlobalConfigPath: nonexistentGlobal(t),
			})

			require.NoError(t, err)
			require.NotNil(t, rc)

			assert.Equal(t, tt.wantFormat, rc.Profile.Format,
				"profile %q: unexpected format", tt.profileName)
			assert.Equal(t, tt.wantMaxTokens, rc.Profile.MaxTokens,
				"profile %q: unexpected max_tokens", tt.profileName)
			assert.Equal(t, tt.wantTokenizer, rc.Profile.Tokenizer,
				"profile %q: unexpected tokenizer", tt.profileName)
			assert.Equal(t, tt.profileName, rc.ProfileName)
		})
	}
}

// ── Scenario 5: env var overrides ─────────────────────────────────────────────

// TestIntegration_Scenario5_EnvOverrides verifies that HARVX_MAX_TOKENS
// overrides the repo config value.
func TestIntegration_Scenario5_EnvOverrides(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clearHarvxEnv(t)
	t.Setenv(EnvMaxTokens, "75000")

	rc, err := Resolve(ResolveOptions{
		TargetDir:        fixturePath(t, "scenario-5-env-overrides"),
		GlobalConfigPath: nonexistentGlobal(t),
	})

	require.NoError(t, err)
	require.NotNil(t, rc)

	// The repo config sets max_tokens=50000 but the env var sets 75000.
	assert.Equal(t, 75000, rc.Profile.MaxTokens,
		"HARVX_MAX_TOKENS=75000 must override repo config's 50000")

	// Source attribution.
	assert.Equal(t, SourceEnv, rc.Sources["max_tokens"],
		"max_tokens must be attributed to env source")
}

// ── Scenario 6: CLI flags override env ────────────────────────────────────────

// TestIntegration_Scenario6_CLIFlags verifies that explicit CLI flags override
// both env vars and repo config values.
func TestIntegration_Scenario6_CLIFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clearHarvxEnv(t)
	t.Setenv(EnvMaxTokens, "75000")

	rc, err := Resolve(ResolveOptions{
		TargetDir:        fixturePath(t, "scenario-6-cli-flags"),
		GlobalConfigPath: nonexistentGlobal(t),
		CLIFlags:         map[string]any{"max_tokens": 60000},
	})

	require.NoError(t, err)
	require.NotNil(t, rc)

	// CLI flag (60000) must win over env var (75000) and repo config (50000).
	assert.Equal(t, 60000, rc.Profile.MaxTokens,
		"CLI flag max_tokens=60000 must override env HARVX_MAX_TOKENS=75000")

	// Source attribution.
	assert.Equal(t, SourceFlag, rc.Sources["max_tokens"],
		"max_tokens must be attributed to flag source")
}

// ── Scenario 7: template init ─────────────────────────────────────────────────

// TestIntegration_Scenario7_TemplateInit verifies that a rendered template
// produces valid TOML that can be loaded and passes validation.
func TestIntegration_Scenario7_TemplateInit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Render the "nextjs" template with a project name.
	tomlContent, err := RenderTemplate("nextjs", "myproject")
	require.NoError(t, err)
	require.NotEmpty(t, tomlContent, "rendered template must not be empty")

	// Write to a temporary file and load it.
	tempDir := t.TempDir()
	tomlPath := filepath.Join(tempDir, "harvx.toml")
	require.NoError(t, os.WriteFile(tomlPath, []byte(tomlContent), 0o644))

	cfg, err := LoadFromFile(tomlPath)
	require.NoError(t, err, "rendered template must be valid TOML")
	require.NotNil(t, cfg)

	// Validation must produce no hard errors.
	issues := Validate(cfg)
	for _, issue := range issues {
		if issue.Severity == "error" {
			t.Errorf("rendered nextjs template has validation error: %s", issue.Error())
		}
	}
}

// ── Scenario 8: complex finvault profile ──────────────────────────────────────

// TestIntegration_Scenario8_ComplexFinvault verifies that the full finvault
// profile with all advanced fields resolves correctly.
func TestIntegration_Scenario8_ComplexFinvault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clearHarvxEnv(t)

	rc, err := Resolve(ResolveOptions{
		ProfileName:      "finvault",
		TargetDir:        fixturePath(t, "scenario-8-complex-finvault"),
		GlobalConfigPath: nonexistentGlobal(t),
	})

	require.NoError(t, err)
	require.NotNil(t, rc)

	// Core profile fields.
	assert.Equal(t, "xml", rc.Profile.Format,
		"finvault profile must set format=xml")
	assert.Equal(t, "o200k_base", rc.Profile.Tokenizer,
		"finvault profile must set tokenizer=o200k_base")
	assert.Equal(t, true, rc.Profile.Compression,
		"finvault profile must enable compression")
	assert.Equal(t, "claude", rc.Profile.Target,
		"finvault profile must set target=claude")
	assert.Equal(t, ".harvx/finvault-context.md", rc.Profile.Output,
		"finvault profile must set the correct output path")

	// The finvault profile sets target=claude which triggers the ApplyTargetPreset
	// in the resolver (after env, before CLI flags). The preset sets format=xml
	// and max_tokens=200000. The repo config also explicitly sets max_tokens=200000
	// so the final value is 200000 regardless of preset order.
	assert.Equal(t, 200000, rc.Profile.MaxTokens,
		"finvault profile max_tokens must be 200000")

	// Redaction config.
	assert.True(t, rc.Profile.Redaction,
		"finvault profile must enable redaction")

	// Profile name must be "finvault".
	assert.Equal(t, "finvault", rc.ProfileName)
}
