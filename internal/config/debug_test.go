package config

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── abbreviateSlice ───────────────────────────────────────────────────────────

// TestAbbreviateSlice_Empty verifies that an empty slice returns an empty string.
func TestAbbreviateSlice_Empty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", abbreviateSlice(nil))
	assert.Equal(t, "", abbreviateSlice([]string{}))
}

// TestAbbreviateSlice_OneToThreeItems verifies that 1–3 items are all shown.
func TestAbbreviateSlice_OneToThreeItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []string
		want  string
	}{
		{
			name:  "one item",
			items: []string{"a"},
			want:  "[a]",
		},
		{
			name:  "two items",
			items: []string{"a", "b"},
			want:  "[a, b]",
		},
		{
			name:  "three items",
			items: []string{"a", "b", "c"},
			want:  "[a, b, c]",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, abbreviateSlice(tt.items))
		})
	}
}

// TestAbbreviateSlice_FourOrMoreItems verifies that >3 items are abbreviated
// with "...N more" where N = len-3.
func TestAbbreviateSlice_FourOrMoreItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []string
		want  string
	}{
		{
			name:  "four items shows 1 more",
			items: []string{"a", "b", "c", "d"},
			want:  "[a, b, c ...1 more]",
		},
		{
			name:  "ten items shows 7 more",
			items: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
			want:  "[a, b, c ...7 more]",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := abbreviateSlice(tt.items)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ── BuildDebugOutput: default-only config ─────────────────────────────────────

// TestBuildDebugOutput_DefaultOnly verifies that when no config files exist all
// config entries have source "default", and both config file statuses are
// "not found".
func TestBuildDebugOutput_DefaultOnly(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	// Point global config at a nonexistent path so it is "not found" per the
	// test convention: an empty GlobalConfigPath triggers the real discovery
	// path, whereas a nonexistent path override makes global "found=false".
	// We want neither discovery to succeed, so we pass a nonexistent path.
	nonexistent := filepath.Join(dir, "no-global.toml")

	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: nonexistent,
	})
	require.NoError(t, err)
	require.NotNil(t, out)

	// Both config files should be "not found".
	require.Len(t, out.ConfigFiles, 2)

	// The global entry has Found=true because GlobalConfigPath is non-empty
	// (it is treated as "found by convention"). The repo entry must be
	// not found since dir has no harvx.toml.
	var repoEntry ConfigFileStatus
	for _, cf := range out.ConfigFiles {
		if cf.Label == "Repo" {
			repoEntry = cf
		}
	}
	assert.False(t, repoEntry.Found, "repo config must be not found in an empty temp dir")

	// All config entries should have source "default".
	for _, ce := range out.Config {
		// Unset fields get source "-"; skip those.
		if ce.Source == "-" {
			continue
		}
		assert.Equal(t, "default", ce.Source,
			"field %q must have source 'default' when no overrides are present", ce.Key)
	}

	// Active profile must be "default".
	assert.Equal(t, "default", out.ActiveProfile)
}

// TestBuildDebugOutput_GlobalFound verifies that when GlobalConfigPath points
// at an existing file, the global config file status shows found=true.
func TestBuildDebugOutput_GlobalFound(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	globalPath := writeTomlFile(t, dir, "global.toml", `
[profile.default]
format = "xml"
`)

	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        t.TempDir(), // empty, no repo config
		GlobalConfigPath: globalPath,
	})
	require.NoError(t, err)

	var globalEntry ConfigFileStatus
	for _, cf := range out.ConfigFiles {
		if cf.Label == "Global" {
			globalEntry = cf
		}
	}
	// When GlobalConfigPath override is non-empty, it is treated as found.
	assert.True(t, globalEntry.Found, "global config must show found=true when override path is set")
}

// ── BuildDebugOutput: repo config override ────────────────────────────────────

// TestBuildDebugOutput_RepoOverride verifies that a field set in harvx.toml
// reports source "repo" and the repo config file status is "loaded".
func TestBuildDebugOutput_RepoOverride(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	writeTomlFile(t, dir, "harvx.toml", `
[profile.default]
output = "custom.md"
format = "xml"
`)

	nonexistentGlobal := filepath.Join(dir, "no-global.toml")
	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: nonexistentGlobal,
	})
	require.NoError(t, err)

	// Repo config file must be "loaded".
	var repoEntry ConfigFileStatus
	for _, cf := range out.ConfigFiles {
		if cf.Label == "Repo" {
			repoEntry = cf
		}
	}
	assert.True(t, repoEntry.Found, "repo config must show found=true when harvx.toml exists")

	// Fields set in repo config must show source "repo".
	outputEntry := findConfigEntry(t, out, "output")
	assert.Equal(t, "repo", outputEntry.Source)

	formatEntry := findConfigEntry(t, out, "format")
	assert.Equal(t, "repo", formatEntry.Source)
}

// ── BuildDebugOutput: env var override ────────────────────────────────────────

// TestBuildDebugOutput_EnvVarOverride verifies that a field set via HARVX_*
// env var reports the correct "env (HARVX_...)" source label, and the
// corresponding EnvVarStatus entry shows Applied=true.
func TestBuildDebugOutput_EnvVarOverride(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvMaxTokens, "200000")

	dir := t.TempDir()
	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	// max_tokens must have source "env (HARVX_MAX_TOKENS)".
	entry := findConfigEntry(t, out, "max_tokens")
	assert.Equal(t, "env (HARVX_MAX_TOKENS)", entry.Source)

	// HARVX_MAX_TOKENS must appear as applied in EnvVars.
	ev := findEnvVarStatus(t, out, EnvMaxTokens)
	assert.True(t, ev.Applied)
	assert.Equal(t, "200000", ev.Value)
}

// TestBuildDebugOutput_EnvVarOutput verifies the HARVX_OUTPUT env var.
func TestBuildDebugOutput_EnvVarOutput(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvOutput, "env-override.md")

	dir := t.TempDir()
	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	entry := findConfigEntry(t, out, "output")
	assert.Equal(t, "env (HARVX_OUTPUT)", entry.Source)
	assert.Equal(t, "env-override.md", entry.Value)
}

// ── BuildDebugOutput: CLI flag override ──────────────────────────────────────

// TestBuildDebugOutput_CLIFlagOverride verifies that a field set via CLIFlags
// reports the correct "flag (--...)" source label.
func TestBuildDebugOutput_CLIFlagOverride(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
		CLIFlags:         map[string]any{"output": "flagged.md"},
	})
	require.NoError(t, err)

	entry := findConfigEntry(t, out, "output")
	assert.Equal(t, "flag (--output)", entry.Source)
	assert.Equal(t, "flagged.md", entry.Value)
}

// TestBuildDebugOutput_CLIFlagFormat verifies that format set via CLIFlags
// reports "flag (--format)".
func TestBuildDebugOutput_CLIFlagFormat(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
		CLIFlags:         map[string]any{"format": "xml"},
	})
	require.NoError(t, err)

	entry := findConfigEntry(t, out, "format")
	assert.Equal(t, "flag (--format)", entry.Source)
}

// ── BuildDebugOutput: missing vs. present config files ───────────────────────

// TestBuildDebugOutput_RepoConfigNotFound verifies that when no harvx.toml
// exists in TargetDir, the Repo ConfigFileStatus shows Found=false and the
// display path starts with "./".
func TestBuildDebugOutput_RepoConfigNotFound(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	var repoEntry ConfigFileStatus
	for _, cf := range out.ConfigFiles {
		if cf.Label == "Repo" {
			repoEntry = cf
		}
	}
	assert.False(t, repoEntry.Found)
	assert.True(t, strings.HasPrefix(repoEntry.Path, "./"),
		"repo path must start with './', got: %q", repoEntry.Path)
}

// TestBuildDebugOutput_RepoConfigFound verifies that when harvx.toml exists
// in TargetDir, the Repo ConfigFileStatus shows Found=true.
func TestBuildDebugOutput_RepoConfigFound(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	writeTomlFile(t, dir, "harvx.toml", "[profile.default]\nformat = \"xml\"\n")

	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	var repoEntry ConfigFileStatus
	for _, cf := range out.ConfigFiles {
		if cf.Label == "Repo" {
			repoEntry = cf
		}
	}
	assert.True(t, repoEntry.Found, "repo config must be 'loaded' when harvx.toml exists")
}

// ── BuildDebugOutput: profile inheritance chain ───────────────────────────────

// TestBuildDebugOutput_InheritanceChain verifies that when a profile extends
// another, InheritChain contains both names and ActiveProfile shows the
// "extends:" label.
func TestBuildDebugOutput_InheritanceChain(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	writeTomlFile(t, dir, "harvx.toml", `
[profile.default]
format = "markdown"

[profile.child]
extends = "default"
format = "xml"
`)

	out, err := BuildDebugOutput(DebugOptions{
		ProfileName:      "child",
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	// InheritChain must be ["child", "default"].
	require.Len(t, out.InheritChain, 2)
	assert.Equal(t, "child", out.InheritChain[0])
	assert.Equal(t, "default", out.InheritChain[1])

	// ActiveProfile must show the "extends:" notation.
	assert.Equal(t, "child (extends: default)", out.ActiveProfile)
}

// TestBuildDebugOutput_SingleProfileNoExtends verifies that a single-element
// chain produces a plain ActiveProfile name with no "extends:" annotation.
func TestBuildDebugOutput_SingleProfileNoExtends(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	assert.Equal(t, "default", out.ActiveProfile)
	assert.NotContains(t, out.ActiveProfile, "extends")
}

// ── BuildDebugOutput: all known env vars reported ────────────────────────────

// TestBuildDebugOutput_AllEnvVarsReported verifies that all known HARVX_* env
// vars appear in out.EnvVars, and that an unset var shows Applied=false while
// a set var shows Applied=true.
func TestBuildDebugOutput_AllEnvVarsReported(t *testing.T) {
	clearHarvxEnv(t)

	knownVars := []string{
		EnvProfile,
		EnvMaxTokens,
		EnvFormat,
		EnvTokenizer,
		EnvOutput,
		EnvTarget,
		EnvCompress,
		EnvRedact,
		EnvLogFormat,
	}

	dir := t.TempDir()
	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	// Every known var must appear in out.EnvVars.
	reported := make(map[string]EnvVarStatus)
	for _, ev := range out.EnvVars {
		reported[ev.Name] = ev
	}
	for _, name := range knownVars {
		assert.Contains(t, reported, name,
			"env var %q must appear in DebugOutput.EnvVars", name)
	}

	// Total count must match the number of known vars.
	assert.Len(t, out.EnvVars, len(knownVars))

	// All should be not applied (cleared by clearHarvxEnv).
	for _, ev := range out.EnvVars {
		assert.False(t, ev.Applied,
			"env var %q must show Applied=false when not set", ev.Name)
	}
}

// TestBuildDebugOutput_SetEnvVarApplied verifies that when a HARVX_* env var
// is set, its EnvVarStatus entry shows Applied=true and the correct value.
func TestBuildDebugOutput_SetEnvVarApplied(t *testing.T) {
	clearHarvxEnv(t)
	t.Setenv(EnvFormat, "xml")
	t.Setenv(EnvLogFormat, "json")

	dir := t.TempDir()
	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	formatEV := findEnvVarStatus(t, out, EnvFormat)
	assert.True(t, formatEV.Applied)
	assert.Equal(t, "xml", formatEV.Value)

	logEV := findEnvVarStatus(t, out, EnvLogFormat)
	assert.True(t, logEV.Applied)
	assert.Equal(t, "json", logEV.Value)
}

// ── BuildDebugOutput: slice abbreviation in output ────────────────────────────

// TestBuildDebugOutput_SliceAbbreviation verifies that when a profile has more
// than 3 ignore patterns, the value in the ConfigEntry is abbreviated with
// "...N more".
func TestBuildDebugOutput_SliceAbbreviation(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	writeTomlFile(t, dir, "harvx.toml", `
[profile.default]
ignore = ["a", "b", "c", "d", "e"]
`)

	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	entry := findConfigEntry(t, out, "ignore")
	// 5 items: first 3 shown + "...2 more".
	assert.Contains(t, entry.Value, "...2 more",
		"ignore field with 5 items must be abbreviated as '...2 more', got: %q", entry.Value)
	assert.True(t, strings.HasPrefix(entry.Value, "[a, b, c"),
		"abbreviated value must start with first 3 items, got: %q", entry.Value)
}

// TestBuildDebugOutput_SliceUpToThreeNotAbbreviated verifies that 3 or fewer
// slice items are never abbreviated.
func TestBuildDebugOutput_SliceUpToThreeNotAbbreviated(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	writeTomlFile(t, dir, "harvx.toml", `
[profile.default]
ignore = ["x", "y", "z"]
`)

	out, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	entry := findConfigEntry(t, out, "ignore")
	assert.NotContains(t, entry.Value, "more",
		"3 items must not be abbreviated, got: %q", entry.Value)
	assert.Equal(t, "[x, y, z]", entry.Value)
}

// ── FormatDebugOutput ─────────────────────────────────────────────────────────

// TestFormatDebugOutput_Header verifies that the text output starts with the
// expected header and section headings.
func TestFormatDebugOutput_Header(t *testing.T) {
	out := sampleDebugOutput()

	var buf bytes.Buffer
	err := FormatDebugOutput(out, &buf)
	require.NoError(t, err)

	text := buf.String()
	assert.Contains(t, text, "Harvx Configuration Debug")
	assert.Contains(t, text, "==========================")
	assert.Contains(t, text, "Config Files:")
	assert.Contains(t, text, "Active Profile:")
	assert.Contains(t, text, "Environment Variables:")
	assert.Contains(t, text, "Resolved Configuration:")
}

// TestFormatDebugOutput_ConfigFileStatus verifies that "not found" and "loaded"
// labels are rendered correctly for each config file.
func TestFormatDebugOutput_ConfigFileStatus(t *testing.T) {
	out := &DebugOutput{
		ConfigFiles: []ConfigFileStatus{
			{Label: "Global", Path: "~/.config/harvx/config.toml", Found: false},
			{Label: "Repo", Path: "./harvx.toml", Found: true},
		},
		ActiveProfile: "default",
		EnvVars:       []EnvVarStatus{},
		Config:        []ConfigEntry{},
	}

	var buf bytes.Buffer
	err := FormatDebugOutput(out, &buf)
	require.NoError(t, err)

	text := buf.String()
	assert.Contains(t, text, "not found")
	assert.Contains(t, text, "loaded")
}

// TestFormatDebugOutput_EnvVarApplied verifies that applied env vars show
// "(applied)" and unset vars show "(not set)".
func TestFormatDebugOutput_EnvVarApplied(t *testing.T) {
	out := &DebugOutput{
		ConfigFiles:   []ConfigFileStatus{},
		ActiveProfile: "default",
		EnvVars: []EnvVarStatus{
			{Name: "HARVX_MAX_TOKENS", Value: "150000", Applied: true},
			{Name: "HARVX_COMPRESS", Applied: false},
		},
		Config: []ConfigEntry{},
	}

	var buf bytes.Buffer
	err := FormatDebugOutput(out, &buf)
	require.NoError(t, err)

	text := buf.String()
	assert.Contains(t, text, "(applied)")
	assert.Contains(t, text, "(not set)")
	assert.Contains(t, text, "150000")
}

// TestFormatDebugOutput_ConfigTableHeaders verifies that the resolved config
// table contains KEY, VALUE, and SOURCE column headers.
func TestFormatDebugOutput_ConfigTableHeaders(t *testing.T) {
	out := sampleDebugOutput()

	var buf bytes.Buffer
	err := FormatDebugOutput(out, &buf)
	require.NoError(t, err)

	text := buf.String()
	assert.Contains(t, text, "KEY")
	assert.Contains(t, text, "VALUE")
	assert.Contains(t, text, "SOURCE")
}

// TestFormatDebugOutput_ConfigEntries verifies that config entries appear in
// the tabwriter output.
func TestFormatDebugOutput_ConfigEntries(t *testing.T) {
	out := &DebugOutput{
		ConfigFiles:   []ConfigFileStatus{},
		ActiveProfile: "default",
		EnvVars:       []EnvVarStatus{},
		Config: []ConfigEntry{
			{Key: "output", Value: "harvx-output.md", Source: "default"},
			{Key: "format", Value: "xml", Source: "repo"},
		},
	}

	var buf bytes.Buffer
	err := FormatDebugOutput(out, &buf)
	require.NoError(t, err)

	text := buf.String()
	assert.Contains(t, text, "output")
	assert.Contains(t, text, "harvx-output.md")
	assert.Contains(t, text, "format")
	assert.Contains(t, text, "xml")
	assert.Contains(t, text, "repo")
}

// ── FormatDebugOutputJSON ─────────────────────────────────────────────────────

// TestFormatDebugOutputJSON_ValidJSON verifies that the JSON output is valid
// and can be unmarshalled.
func TestFormatDebugOutputJSON_ValidJSON(t *testing.T) {
	out := sampleDebugOutput()

	var buf bytes.Buffer
	err := FormatDebugOutputJSON(out, &buf)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err, "FormatDebugOutputJSON must produce valid JSON")
}

// TestFormatDebugOutputJSON_ExpectedTopLevelFields verifies that the JSON
// output contains the required top-level fields.
func TestFormatDebugOutputJSON_ExpectedTopLevelFields(t *testing.T) {
	out := sampleDebugOutput()

	var buf bytes.Buffer
	err := FormatDebugOutputJSON(out, &buf)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))

	for _, field := range []string{"config_files", "active_profile", "env_vars", "config"} {
		assert.Contains(t, parsed, field,
			"JSON output must contain top-level key %q", field)
	}
}

// TestFormatDebugOutputJSON_ConfigFilesStructure verifies that config_files
// entries have label, path, and found fields.
func TestFormatDebugOutputJSON_ConfigFilesStructure(t *testing.T) {
	out := &DebugOutput{
		ConfigFiles: []ConfigFileStatus{
			{Label: "Global", Path: "~/.config/harvx/config.toml", Found: false},
			{Label: "Repo", Path: "./harvx.toml", Found: true},
		},
		ActiveProfile: "myprofile",
		EnvVars:       []EnvVarStatus{},
		Config:        []ConfigEntry{},
	}

	var buf bytes.Buffer
	err := FormatDebugOutputJSON(out, &buf)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))

	files, ok := parsed["config_files"].([]any)
	require.True(t, ok, "config_files must be an array")
	require.Len(t, files, 2)

	first, ok := files[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Global", first["label"])
	assert.Equal(t, false, first["found"])
}

// TestFormatDebugOutputJSON_ActiveProfileField verifies that active_profile
// is correctly serialised.
func TestFormatDebugOutputJSON_ActiveProfileField(t *testing.T) {
	out := &DebugOutput{
		ConfigFiles:   []ConfigFileStatus{},
		ActiveProfile: "finvault (extends: base -> default)",
		EnvVars:       []EnvVarStatus{},
		Config:        []ConfigEntry{},
	}

	var buf bytes.Buffer
	err := FormatDebugOutputJSON(out, &buf)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))

	assert.Equal(t, "finvault (extends: base -> default)", parsed["active_profile"])
}

// TestFormatDebugOutputJSON_InheritChainOmittedWhenEmpty verifies that
// inherit_chain is not present in JSON when the chain has a single element
// (omitempty).
func TestFormatDebugOutputJSON_InheritChainOmittedWhenEmpty(t *testing.T) {
	out := &DebugOutput{
		ConfigFiles:   []ConfigFileStatus{},
		ActiveProfile: "default",
		InheritChain:  nil,
		EnvVars:       []EnvVarStatus{},
		Config:        []ConfigEntry{},
	}

	var buf bytes.Buffer
	err := FormatDebugOutputJSON(out, &buf)
	require.NoError(t, err)

	text := buf.String()
	assert.NotContains(t, text, "inherit_chain",
		"inherit_chain must be omitted when nil (omitempty)")
}

// TestFormatDebugOutputJSON_InheritChainPresentWhenSet verifies that
// inherit_chain appears in JSON when the chain has multiple elements.
func TestFormatDebugOutputJSON_InheritChainPresentWhenSet(t *testing.T) {
	out := &DebugOutput{
		ConfigFiles:   []ConfigFileStatus{},
		ActiveProfile: "child (extends: default)",
		InheritChain:  []string{"child", "default"},
		EnvVars:       []EnvVarStatus{},
		Config:        []ConfigEntry{},
	}

	var buf bytes.Buffer
	err := FormatDebugOutputJSON(out, &buf)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))

	chain, ok := parsed["inherit_chain"].([]any)
	require.True(t, ok, "inherit_chain must be present when non-empty")
	assert.Len(t, chain, 2)
	assert.Equal(t, "child", chain[0])
	assert.Equal(t, "default", chain[1])
}

// ── sourceDetailLabel ─────────────────────────────────────────────────────────

// TestSourceDetailLabel_AllSources verifies the human-readable labels
// produced by sourceDetailLabel for every Source constant and the key
// types that produce embedded env var / flag names.
func TestSourceDetailLabel_AllSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		src  Source
		want string
	}{
		{name: "default", key: "format", src: SourceDefault, want: "default"},
		{name: "global", key: "format", src: SourceGlobal, want: "global"},
		{name: "repo", key: "format", src: SourceRepo, want: "repo"},
		{name: "env with known key", key: "max_tokens", src: SourceEnv, want: "env (HARVX_MAX_TOKENS)"},
		{name: "env with unknown key", key: "unknown_key", src: SourceEnv, want: "env"},
		{name: "flag with known key", key: "output", src: SourceFlag, want: "flag (--output)"},
		{name: "flag with unknown key", key: "unknown_key", src: SourceFlag, want: "flag"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sourceDetailLabel(tt.key, tt.src)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ── buildActiveProfileLabel ───────────────────────────────────────────────────

// TestBuildActiveProfileLabel verifies the display format for chains of
// various lengths.
func TestBuildActiveProfileLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		chain []string
		want  string
	}{
		{name: "empty chain", chain: nil, want: "default"},
		{name: "single element", chain: []string{"default"}, want: "default"},
		{name: "two elements", chain: []string{"child", "default"}, want: "child (extends: default)"},
		{
			name:  "three elements",
			chain: []string{"finvault", "base", "default"},
			want:  "finvault (extends: base -> default)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildActiveProfileLabel(tt.chain)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ── integration: BuildDebugOutput → FormatDebugOutput ────────────────────────

// TestBuildAndFormat_Integration verifies that the full pipeline from
// BuildDebugOutput through FormatDebugOutput produces the expected section
// headers for a default-only configuration.
func TestBuildAndFormat_Integration(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	result, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	err = FormatDebugOutput(result, &buf)
	require.NoError(t, err)

	text := buf.String()
	assert.Contains(t, text, "Harvx Configuration Debug")
	assert.Contains(t, text, "Config Files:")
	assert.Contains(t, text, "Active Profile:")
	assert.Contains(t, text, "Environment Variables:")
	assert.Contains(t, text, "Resolved Configuration:")
}

// TestBuildAndFormatJSON_Integration verifies that the full pipeline from
// BuildDebugOutput through FormatDebugOutputJSON produces valid JSON.
func TestBuildAndFormatJSON_Integration(t *testing.T) {
	clearHarvxEnv(t)

	dir := t.TempDir()
	result, err := BuildDebugOutput(DebugOptions{
		TargetDir:        dir,
		GlobalConfigPath: filepath.Join(dir, "no-global.toml"),
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	err = FormatDebugOutputJSON(result, &buf)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err, "full pipeline JSON output must be valid")

	for _, field := range []string{"config_files", "active_profile", "env_vars", "config"} {
		assert.Contains(t, parsed, field)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// sampleDebugOutput returns a minimal DebugOutput suitable for format tests.
func sampleDebugOutput() *DebugOutput {
	return &DebugOutput{
		ConfigFiles: []ConfigFileStatus{
			{Label: "Global", Path: "~/.config/harvx/config.toml", Found: false},
			{Label: "Repo", Path: "./harvx.toml", Found: false},
		},
		ActiveProfile: "default",
		InheritChain:  nil,
		EnvVars: []EnvVarStatus{
			{Name: "HARVX_MAX_TOKENS", Applied: false},
			{Name: "HARVX_FORMAT", Applied: false},
		},
		Config: []ConfigEntry{
			{Key: "output", Value: "harvx-output.md", Source: "default"},
			{Key: "format", Value: "markdown", Source: "default"},
		},
	}
}

// findConfigEntry returns the ConfigEntry with the given key. It fails the
// test immediately if the key is not present.
func findConfigEntry(t *testing.T, out *DebugOutput, key string) ConfigEntry {
	t.Helper()
	for _, ce := range out.Config {
		if ce.Key == key {
			return ce
		}
	}
	t.Fatalf("config entry %q not found in DebugOutput", key)
	return ConfigEntry{}
}

// findEnvVarStatus returns the EnvVarStatus for the given env var name. It
// fails the test immediately if the name is not present.
func findEnvVarStatus(t *testing.T, out *DebugOutput, name string) EnvVarStatus {
	t.Helper()
	for _, ev := range out.EnvVars {
		if ev.Name == name {
			return ev
		}
	}
	t.Fatalf("env var %q not found in DebugOutput.EnvVars", name)
	return EnvVarStatus{}
}

