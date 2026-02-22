package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testdataPath returns the absolute path to a file under testdata/config/.
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "config", name)
}

// TestLoadFromFile_ValidConfig loads the PRD example config and verifies that
// all fields are decoded correctly, including nested tables.
func TestLoadFromFile_ValidConfig(t *testing.T) {
	t.Parallel()

	path := testdataPath(t, "valid.toml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture not found: %s", path)
	}

	cfg, err := LoadFromFile(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Config must have a profile map.
	require.NotNil(t, cfg.Profile)

	// --- default profile ---
	def, ok := cfg.Profile["default"]
	require.True(t, ok, "profile 'default' must exist")
	require.NotNil(t, def)

	assert.Equal(t, "harvx-output.md", def.Output)
	assert.Equal(t, "markdown", def.Format)
	assert.Equal(t, 128000, def.MaxTokens)
	assert.Equal(t, "cl100k_base", def.Tokenizer)
	assert.False(t, def.Compression)
	assert.True(t, def.Redaction)
	assert.Equal(t, []string{"node_modules", "dist", ".git", "coverage", "__pycache__"}, def.Ignore)

	// --- finvault profile ---
	fv, ok := cfg.Profile["finvault"]
	require.True(t, ok, "profile 'finvault' must exist")
	require.NotNil(t, fv)

	require.NotNil(t, fv.Extends)
	assert.Equal(t, "default", *fv.Extends)
	assert.Equal(t, ".harvx/finvault-context.md", fv.Output)
	assert.Equal(t, 200000, fv.MaxTokens)
	assert.Equal(t, "o200k_base", fv.Tokenizer)
	assert.True(t, fv.Compression)
	assert.Equal(t, "claude", fv.Target)

	assert.Equal(t, []string{"CLAUDE.md", "prisma/schema.prisma"}, fv.PriorityFiles)

	assert.Equal(t, []string{
		"reports/",
		".review-workspace/",
		".harvx/",
		".next/",
	}, fv.Ignore)
}

// TestLoadFromFile_ValidConfig_RelevanceTiers verifies that nested relevance
// tier tables decode into the correct struct fields.
func TestLoadFromFile_ValidConfig_RelevanceTiers(t *testing.T) {
	t.Parallel()

	path := testdataPath(t, "valid.toml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture not found: %s", path)
	}

	cfg, err := LoadFromFile(path)
	require.NoError(t, err)

	fv := cfg.Profile["finvault"]
	require.NotNil(t, fv)

	r := fv.Relevance
	assert.Equal(t, []string{"CLAUDE.md", "prisma/schema.prisma", "*.config.*"}, r.Tier0)
	assert.Equal(t, []string{"app/api/**", "lib/services/**", "middleware.ts"}, r.Tier1)
	assert.Equal(t, []string{"components/**", "hooks/**", "lib/**"}, r.Tier2)
	assert.Equal(t, []string{"__tests__/**"}, r.Tier3)
	assert.Equal(t, []string{"docs/**", "prompts/**"}, r.Tier4)
	assert.Equal(t, []string{".github/**", "*.lock"}, r.Tier5)
}

// TestLoadFromFile_ValidConfig_RedactionConfig verifies that the nested
// redaction_config table decodes into RedactionConfig correctly.
func TestLoadFromFile_ValidConfig_RedactionConfig(t *testing.T) {
	t.Parallel()

	path := testdataPath(t, "valid.toml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture not found: %s", path)
	}

	cfg, err := LoadFromFile(path)
	require.NoError(t, err)

	fv := cfg.Profile["finvault"]
	require.NotNil(t, fv)

	rc := fv.RedactionConfig
	assert.True(t, rc.Enabled)
	assert.Equal(t, []string{"**/*test*/**", "**/fixtures/**", "docs/**/*.md"}, rc.ExcludePaths)
	assert.Equal(t, "high", rc.ConfidenceThreshold)
}

// TestLoadFromFile_MinimalConfig loads the minimal fixture which only declares
// an empty [profile.default] table and verifies the profile exists with zero
// values.
func TestLoadFromFile_MinimalConfig(t *testing.T) {
	t.Parallel()

	path := testdataPath(t, "minimal.toml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture not found: %s", path)
	}

	cfg, err := LoadFromFile(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	def, ok := cfg.Profile["default"]
	require.True(t, ok)
	require.NotNil(t, def)

	// All fields should be zero values.
	assert.Equal(t, "", def.Output)
	assert.Equal(t, "", def.Format)
	assert.Equal(t, 0, def.MaxTokens)
	assert.Nil(t, def.Extends)
}

// TestLoadFromFile_InvalidSyntax verifies that malformed TOML returns an error
// that mentions the file path.
func TestLoadFromFile_InvalidSyntax(t *testing.T) {
	t.Parallel()

	path := testdataPath(t, "invalid_syntax.toml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture not found: %s", path)
	}

	_, err := LoadFromFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_syntax.toml", "error must mention the file path")
}

// TestLoadFromFile_UnknownKeys verifies that unknown TOML keys do not cause
// an error (they are warned about via slog).
func TestLoadFromFile_UnknownKeys(t *testing.T) {
	t.Parallel()

	path := testdataPath(t, "unknown_keys.toml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture not found: %s", path)
	}

	cfg, err := LoadFromFile(path)
	require.NoError(t, err, "unknown keys must not cause an error")
	require.NotNil(t, cfg)

	// Known fields should still be decoded correctly.
	def, ok := cfg.Profile["default"]
	require.True(t, ok)
	assert.Equal(t, "harvx-output.md", def.Output)
	assert.Equal(t, "markdown", def.Format)
	assert.Equal(t, 128000, def.MaxTokens)
}

// TestLoadFromFile_NonExistentFile verifies that a missing file returns an
// error.
func TestLoadFromFile_NonExistentFile(t *testing.T) {
	t.Parallel()

	_, err := LoadFromFile("/nonexistent/path/harvx.toml")
	require.Error(t, err)
}

// TestLoadFromString_ValidTOML exercises the in-memory variant using the PRD
// example TOML embedded as a string literal.
func TestLoadFromString_ValidTOML(t *testing.T) {
	t.Parallel()

	const data = `
[profile.default]
output = "harvx-output.md"
format = "markdown"
max_tokens = 128000
tokenizer = "cl100k_base"
compression = false
redaction = true
ignore = ["node_modules", ".git"]
`

	cfg, err := LoadFromString(data, "<inline>")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	def, ok := cfg.Profile["default"]
	require.True(t, ok)
	assert.Equal(t, "harvx-output.md", def.Output)
	assert.Equal(t, "markdown", def.Format)
	assert.Equal(t, 128000, def.MaxTokens)
	assert.Equal(t, "cl100k_base", def.Tokenizer)
	assert.False(t, def.Compression)
	assert.True(t, def.Redaction)
	assert.Equal(t, []string{"node_modules", ".git"}, def.Ignore)
}

// TestLoadFromString_ExtendsField verifies that the *string extends field
// decodes correctly when set and remains nil when absent.
func TestLoadFromString_ExtendsField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		toml        string
		wantExtends *string
	}{
		{
			name: "extends set",
			toml: `
[profile.child]
extends = "default"
`,
			wantExtends: strPtr("default"),
		},
		{
			name: "extends absent",
			toml: `
[profile.child]
output = "out.md"
`,
			wantExtends: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := LoadFromString(tt.toml, "<test>")
			require.NoError(t, err)

			child := cfg.Profile["child"]
			require.NotNil(t, child)

			if tt.wantExtends == nil {
				assert.Nil(t, child.Extends)
			} else {
				require.NotNil(t, child.Extends)
				assert.Equal(t, *tt.wantExtends, *child.Extends)
			}
		})
	}
}

// TestLoadFromString_EmptyDocument verifies that an empty TOML document
// returns an empty (but non-nil) Config without error.
func TestLoadFromString_EmptyDocument(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromString("", "<empty>")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.Profile)
}

// TestLoadFromString_InvalidSyntax verifies that malformed TOML returns an
// error that mentions the source name.
func TestLoadFromString_InvalidSyntax(t *testing.T) {
	t.Parallel()

	_, err := LoadFromString("[broken", "<test>")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "<test>")
}

// TestLoadFromString_NestedRelevance verifies that inline [profile.x.relevance]
// tables decode correctly.
func TestLoadFromString_NestedRelevance(t *testing.T) {
	t.Parallel()

	const data = `
[profile.custom]
output = "out.md"

[profile.custom.relevance]
tier_0 = ["go.mod", "Makefile"]
tier_1 = ["src/**", "cmd/**"]
tier_3 = ["**/*_test.go"]
`

	cfg, err := LoadFromString(data, "<test>")
	require.NoError(t, err)

	p := cfg.Profile["custom"]
	require.NotNil(t, p)

	assert.Equal(t, []string{"go.mod", "Makefile"}, p.Relevance.Tier0)
	assert.Equal(t, []string{"src/**", "cmd/**"}, p.Relevance.Tier1)
	assert.Nil(t, p.Relevance.Tier2, "Tier2 was not set, should be nil")
	assert.Equal(t, []string{"**/*_test.go"}, p.Relevance.Tier3)
}

// TestLoadFromString_MultipleProfiles verifies that multiple profiles decode
// independently and that profile names are case-sensitive map keys.
func TestLoadFromString_MultipleProfiles(t *testing.T) {
	t.Parallel()

	const data = `
[profile.alpha]
output = "alpha.md"
max_tokens = 50000

[profile.Beta]
output = "beta.md"
max_tokens = 100000
`

	cfg, err := LoadFromString(data, "<test>")
	require.NoError(t, err)
	require.Len(t, cfg.Profile, 2)

	alpha := cfg.Profile["alpha"]
	require.NotNil(t, alpha)
	assert.Equal(t, "alpha.md", alpha.Output)
	assert.Equal(t, 50000, alpha.MaxTokens)

	// Profile names are case-sensitive: "Beta" != "beta".
	betaCaps := cfg.Profile["Beta"]
	require.NotNil(t, betaCaps)
	assert.Equal(t, "beta.md", betaCaps.Output)

	betaLower := cfg.Profile["beta"]
	assert.Nil(t, betaLower, "profile 'beta' (lowercase) must not exist")
}

// TestLoadFromString_TargetField verifies that the target enum-like string
// field decodes correctly for all valid values.
func TestLoadFromString_TargetField(t *testing.T) {
	t.Parallel()

	targets := []string{"claude", "chatgpt", "generic", ""}

	for _, target := range targets {
		t.Run("target="+target, func(t *testing.T) {
			t.Parallel()

			data := `[profile.p]` + "\ntarget = \"" + target + "\"\n"
			if target == "" {
				data = `[profile.p]` + "\n"
			}

			cfg, err := LoadFromString(data, "<test>")
			require.NoError(t, err)

			p := cfg.Profile["p"]
			require.NotNil(t, p)
			assert.Equal(t, target, p.Target)
		})
	}
}

// TestLoadFromFile_RoundTrip loads the valid.toml fixture and writes a temp
// file to confirm field values survive a decode. This is a simplified round-
// trip test; full TOML encoding is tested in golden tests (T-025).
func TestLoadFromFile_RoundTrip(t *testing.T) {
	t.Parallel()

	path := testdataPath(t, "valid.toml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture not found: %s", path)
	}

	cfg1, err := LoadFromFile(path)
	require.NoError(t, err)

	// Re-encode the finvault profile to a temp TOML string and decode again.
	fv1 := cfg1.Profile["finvault"]
	require.NotNil(t, fv1)

	// Build a minimal TOML representation of the finvault profile to re-parse.
	tomlData := `
[profile.finvault]
extends = "default"
output = ".harvx/finvault-context.md"
max_tokens = 200000
tokenizer = "o200k_base"
compression = true
target = "claude"
priority_files = ["CLAUDE.md", "prisma/schema.prisma"]
ignore = ["reports/", ".review-workspace/", ".harvx/", ".next/"]

[profile.finvault.relevance]
tier_0 = ["CLAUDE.md", "prisma/schema.prisma", "*.config.*"]
`

	cfg2, err := LoadFromString(tomlData, "<round-trip>")
	require.NoError(t, err)

	fv2 := cfg2.Profile["finvault"]
	require.NotNil(t, fv2)

	assert.Equal(t, fv1.Output, fv2.Output)
	assert.Equal(t, fv1.MaxTokens, fv2.MaxTokens)
	assert.Equal(t, fv1.Tokenizer, fv2.Tokenizer)
	assert.Equal(t, fv1.Compression, fv2.Compression)
	assert.Equal(t, fv1.Target, fv2.Target)
	assert.Equal(t, fv1.PriorityFiles, fv2.PriorityFiles)
}

// TestLoadFromFile_InvalidSyntax_ContainsLineInfo verifies that a malformed
// TOML file produces an error message that includes positional information
// (line and/or column numbers). BurntSushi/toml formats these as "(line X,
// column Y)" in its error messages.
func TestLoadFromFile_InvalidSyntax_ContainsLineInfo(t *testing.T) {
	t.Parallel()

	path := testdataPath(t, "invalid_syntax.toml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture not found: %s", path)
	}

	_, err := LoadFromFile(path)
	require.Error(t, err)

	// BurntSushi/toml includes "line" in its parse error output.
	errMsg := err.Error()
	assert.True(t,
		containsAny(errMsg, "line", "Line", "column", "Column"),
		"parse error must contain line/column info; got: %s", errMsg)
}

// TestLoadFromString_InvalidSyntax_ContainsLineInfo verifies that a malformed
// in-memory TOML string produces an error with positional information from the
// TOML decoder.
func TestLoadFromString_InvalidSyntax_ContainsLineInfo(t *testing.T) {
	t.Parallel()

	// Deliberately malformed: unclosed section header.
	_, err := LoadFromString("[profile.default\noutput = \"out.md\"\n", "<inline-bad>")
	require.Error(t, err)

	errMsg := err.Error()
	assert.True(t,
		containsAny(errMsg, "line", "Line", "column", "Column"),
		"parse error must contain line/column info; got: %s", errMsg)
}

// TestLoadFromFile_EmptyFile loads an empty file created in a TempDir and
// verifies the loader returns a non-nil empty Config with no error.
func TestLoadFromFile_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.toml")
	require.NoError(t, os.WriteFile(empty, []byte{}, 0o644))

	cfg, err := LoadFromFile(empty)
	require.NoError(t, err, "empty file must not return an error")
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.Profile, "empty file must produce a Config with no profiles")
}

// TestLoadFromFile_TempDirValidTOML verifies LoadFromFile against a fully
// written temp file -- exercising the file path in the success path.
func TestLoadFromFile_TempDirValidTOML(t *testing.T) {
	t.Parallel()

	const data = `
[profile.default]
output = "harvx-output.md"
format = "markdown"
max_tokens = 128000
tokenizer = "cl100k_base"
compression = false
redaction = true
ignore = ["node_modules", ".git", "dist"]
`

	dir := t.TempDir()
	path := filepath.Join(dir, "harvx.toml")
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	cfg, err := LoadFromFile(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	def, ok := cfg.Profile["default"]
	require.True(t, ok, "profile 'default' must exist")
	require.NotNil(t, def)

	assert.Equal(t, "harvx-output.md", def.Output)
	assert.Equal(t, "markdown", def.Format)
	assert.Equal(t, 128000, def.MaxTokens)
	assert.Equal(t, "cl100k_base", def.Tokenizer)
	assert.False(t, def.Compression)
	assert.True(t, def.Redaction)
	assert.Equal(t, []string{"node_modules", ".git", "dist"}, def.Ignore)
}

// TestLoadFromFile_ErrorContainsFilePath verifies that when a TOML file has a
// syntax error the returned error message contains the file path, enabling
// users to identify which file caused the problem.
func TestLoadFromFile_ErrorContainsFilePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad-config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[broken toml"), 0o644))

	_, err := LoadFromFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad-config.toml",
		"error must mention the file name to help the user debug")
}

// TestLoadFromString_ErrorContainsSourceName verifies that LoadFromString
// includes the caller-supplied name in the error message so log output and
// error chains are traceable back to the config source.
func TestLoadFromString_ErrorContainsSourceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sourceName string
		badTOML    string
	}{
		{
			name:       "inline source name",
			sourceName: "<inline-config>",
			badTOML:    "[[broken",
		},
		{
			name:       "file path as source name",
			sourceName: "/home/user/.harvx.toml",
			badTOML:    "[unclosed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := LoadFromString(tt.badTOML, tt.sourceName)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.sourceName,
				"error must contain the source name %q", tt.sourceName)
		})
	}
}

// TestLoadFromString_UnknownKeysNoError verifies that LoadFromString does not
// return an error when the TOML contains keys unknown to the Config struct.
// Known fields must still decode correctly alongside the unknown ones.
func TestLoadFromString_UnknownKeysNoError(t *testing.T) {
	t.Parallel()

	const data = `
[profile.default]
output = "harvx-output.md"
max_tokens = 64000
future_ai_option = "experimental"
unknown_bool = true
`

	cfg, err := LoadFromString(data, "<test-unknown-keys>")
	require.NoError(t, err, "unknown keys must not cause an error")
	require.NotNil(t, cfg)

	def, ok := cfg.Profile["default"]
	require.True(t, ok)
	assert.Equal(t, "harvx-output.md", def.Output,
		"known field 'output' must decode despite unknown keys")
	assert.Equal(t, 64000, def.MaxTokens,
		"known field 'max_tokens' must decode despite unknown keys")
}

// TestLoadFromString_NestedRedactionConfig verifies that a fully specified
// [profile.x.redaction_config] table decodes into RedactionConfig correctly.
func TestLoadFromString_NestedRedactionConfig(t *testing.T) {
	t.Parallel()

	const data = `
[profile.prod]
output = "prod-output.md"

[profile.prod.redaction_config]
enabled = true
exclude_paths = ["testdata/**", "docs/**/*.md"]
confidence_threshold = "medium"
`

	cfg, err := LoadFromString(data, "<test>")
	require.NoError(t, err)

	p := cfg.Profile["prod"]
	require.NotNil(t, p)

	rc := p.RedactionConfig
	assert.True(t, rc.Enabled, "redaction_config.enabled must be true")
	assert.Equal(t, []string{"testdata/**", "docs/**/*.md"}, rc.ExcludePaths)
	assert.Equal(t, "medium", rc.ConfidenceThreshold)
}

// TestLoadFromString_RedactionConfig_ZeroValue verifies that when
// [profile.x.redaction_config] is absent the RedactionConfig fields are zero
// values (Enabled=false, ConfidenceThreshold="", ExcludePaths=nil).
func TestLoadFromString_RedactionConfig_ZeroValue(t *testing.T) {
	t.Parallel()

	const data = `
[profile.bare]
output = "bare.md"
`

	cfg, err := LoadFromString(data, "<test>")
	require.NoError(t, err)

	p := cfg.Profile["bare"]
	require.NotNil(t, p)

	rc := p.RedactionConfig
	assert.False(t, rc.Enabled,
		"RedactionConfig.Enabled must be false when section is absent")
	assert.Empty(t, rc.ConfidenceThreshold,
		"RedactionConfig.ConfidenceThreshold must be empty when section is absent")
	assert.Nil(t, rc.ExcludePaths,
		"RedactionConfig.ExcludePaths must be nil when section is absent")
}

// TestLoadFromString_IncludeField verifies that the include glob patterns
// decode correctly into Profile.Include.
func TestLoadFromString_IncludeField(t *testing.T) {
	t.Parallel()

	const data = `
[profile.custom]
output = "out.md"
include = ["internal/**/*.go", "cmd/**/*.go", "*.md"]
`

	cfg, err := LoadFromString(data, "<test>")
	require.NoError(t, err)

	p := cfg.Profile["custom"]
	require.NotNil(t, p)
	assert.Equal(t, []string{"internal/**/*.go", "cmd/**/*.go", "*.md"}, p.Include)
}

// TestLoadFromString_PriorityFilesField verifies that the priority_files list
// decodes into Profile.PriorityFiles in the correct order.
func TestLoadFromString_PriorityFilesField(t *testing.T) {
	t.Parallel()

	const data = `
[profile.ordered]
output = "ordered.md"
priority_files = [
  "CLAUDE.md",
  "README.md",
  "go.mod",
]
`

	cfg, err := LoadFromString(data, "<test>")
	require.NoError(t, err)

	p := cfg.Profile["ordered"]
	require.NotNil(t, p)
	assert.Equal(t, []string{"CLAUDE.md", "README.md", "go.mod"}, p.PriorityFiles)
}

// TestLoadFromString_CaseSensitiveProfileNames verifies that profile names
// are treated as case-sensitive map keys (AC 10). "Alpha" and "alpha" are
// distinct profiles.
func TestLoadFromString_CaseSensitiveProfileNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tomlData    string
		lookupKey   string
		shouldExist bool
		wantOutput  string
	}{
		{
			name: "uppercase key exists",
			tomlData: `
[profile.Alpha]
output = "alpha-upper.md"
`,
			lookupKey:   "Alpha",
			shouldExist: true,
			wantOutput:  "alpha-upper.md",
		},
		{
			name: "lowercase key does not exist when only uppercase defined",
			tomlData: `
[profile.Alpha]
output = "alpha-upper.md"
`,
			lookupKey:   "alpha",
			shouldExist: false,
		},
		{
			name: "mixed case key DEFAULT is not the same as default",
			tomlData: `
[profile.DEFAULT]
output = "default-upper.md"
`,
			lookupKey:   "default",
			shouldExist: false,
		},
		{
			name: "exact lowercase default key exists",
			tomlData: `
[profile.default]
output = "default-lower.md"
`,
			lookupKey:   "default",
			shouldExist: true,
			wantOutput:  "default-lower.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := LoadFromString(tt.tomlData, "<test>")
			require.NoError(t, err)

			p, ok := cfg.Profile[tt.lookupKey]
			if tt.shouldExist {
				assert.True(t, ok, "profile %q must exist", tt.lookupKey)
				require.NotNil(t, p)
				assert.Equal(t, tt.wantOutput, p.Output)
			} else {
				assert.False(t, ok,
					"profile %q must not exist (profile names are case-sensitive)",
					tt.lookupKey)
				assert.Nil(t, p)
			}
		})
	}
}

// TestLoadFromFile_UnknownKeys_KnownRelevanceFieldDecodes verifies that when
// a TOML file mixes unknown keys alongside known [profile.x.relevance] keys,
// the known relevance field still decodes correctly.
func TestLoadFromFile_UnknownKeys_KnownRelevanceFieldDecodes(t *testing.T) {
	t.Parallel()

	path := testdataPath(t, "unknown_keys.toml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture not found: %s", path)
	}

	cfg, err := LoadFromFile(path)
	require.NoError(t, err, "unknown keys must not cause an error")

	def := cfg.Profile["default"]
	require.NotNil(t, def)

	// The unknown_keys.toml has tier_0 = ["go.mod", "Makefile"] under
	// [profile.default.relevance] alongside an unknown undocumented_tier_6.
	assert.Equal(t, []string{"go.mod", "Makefile"}, def.Relevance.Tier0,
		"known relevance.tier_0 must decode correctly alongside unknown keys")
}

// TestLoadFromString_AllProfileFields verifies that every field in the Profile
// struct decodes from a complete TOML document. This exercises all struct tags
// from types.go in a single integration-style decode.
func TestLoadFromString_AllProfileFields(t *testing.T) {
	t.Parallel()

	const data = `
[profile.full]
extends = "default"
output = "full-output.xml"
format = "xml"
max_tokens = 50000
tokenizer = "o200k_base"
compression = true
redaction = false
target = "chatgpt"
ignore = ["vendor/**", "dist/**"]
priority_files = ["main.go", "go.mod"]
include = ["internal/**"]

[profile.full.relevance]
tier_0 = ["go.mod"]
tier_1 = ["cmd/**"]
tier_2 = ["utils/**"]
tier_3 = ["**/*_test.go"]
tier_4 = ["*.md"]
tier_5 = [".github/**"]

[profile.full.redaction_config]
enabled = false
exclude_paths = ["fixtures/**"]
confidence_threshold = "low"
`

	cfg, err := LoadFromString(data, "<full-test>")
	require.NoError(t, err)

	p := cfg.Profile["full"]
	require.NotNil(t, p, "profile 'full' must exist")

	// Profile-level fields.
	require.NotNil(t, p.Extends)
	assert.Equal(t, "default", *p.Extends)
	assert.Equal(t, "full-output.xml", p.Output)
	assert.Equal(t, "xml", p.Format)
	assert.Equal(t, 50000, p.MaxTokens)
	assert.Equal(t, "o200k_base", p.Tokenizer)
	assert.True(t, p.Compression)
	assert.False(t, p.Redaction)
	assert.Equal(t, "chatgpt", p.Target)
	assert.Equal(t, []string{"vendor/**", "dist/**"}, p.Ignore)
	assert.Equal(t, []string{"main.go", "go.mod"}, p.PriorityFiles)
	assert.Equal(t, []string{"internal/**"}, p.Include)

	// Relevance tiers.
	assert.Equal(t, []string{"go.mod"}, p.Relevance.Tier0)
	assert.Equal(t, []string{"cmd/**"}, p.Relevance.Tier1)
	assert.Equal(t, []string{"utils/**"}, p.Relevance.Tier2)
	assert.Equal(t, []string{"**/*_test.go"}, p.Relevance.Tier3)
	assert.Equal(t, []string{"*.md"}, p.Relevance.Tier4)
	assert.Equal(t, []string{".github/**"}, p.Relevance.Tier5)

	// RedactionConfig.
	assert.False(t, p.RedactionConfig.Enabled)
	assert.Equal(t, []string{"fixtures/**"}, p.RedactionConfig.ExcludePaths)
	assert.Equal(t, "low", p.RedactionConfig.ConfidenceThreshold)
}

// containsAny returns true if s contains at least one of the given substrings.
// It is used to verify that error messages include positional information which
// may appear in different capitalizations depending on the TOML library version.
func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// strPtr is a test helper that returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}
