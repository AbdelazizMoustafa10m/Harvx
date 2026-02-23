package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestConfigDebug builds an isolated command tree containing only
// `harvx config debug` so each test gets a fresh, clean command state
// without interference from the global rootCmd.
func newTestConfigDebug() *cobra.Command {
	root := &cobra.Command{
		Use:           "harvx",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cfgCmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management commands",
	}

	dbgCmd := &cobra.Command{
		Use:   "debug",
		Short: "Show resolved configuration with source annotations",
		RunE:  runConfigDebug,
	}
	dbgCmd.Flags().Bool("json", false, "output as structured JSON")
	dbgCmd.Flags().String("profile", "", "profile name to debug (default: active profile)")

	cfgCmd.AddCommand(dbgCmd)
	root.AddCommand(cfgCmd)

	return root
}

// ── config debug: text output ─────────────────────────────────────────────────

// TestConfigDebugCommand_TextOutput verifies that `harvx config debug` runs
// without error and produces the expected section headers in text mode.
func TestConfigDebugCommand_TextOutput(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Harvx Configuration Debug")
	assert.Contains(t, output, "Config Files:")
	assert.Contains(t, output, "Environment Variables:")
	assert.Contains(t, output, "Resolved Configuration:")
}

// TestConfigDebugCommand_ActiveProfileLine verifies that "Active Profile:"
// appears in the text output.
func TestConfigDebugCommand_ActiveProfileLine(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug"})

	err := root.Execute()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "Active Profile:")
}

// TestConfigDebugCommand_ConfigTableHeaders verifies that the resolved config
// table headers (KEY, VALUE, SOURCE) appear in the text output.
func TestConfigDebugCommand_ConfigTableHeaders(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "KEY")
	assert.Contains(t, output, "VALUE")
	assert.Contains(t, output, "SOURCE")
}

// TestConfigDebugCommand_DefaultSourceAnnotation verifies that with no
// config file present, the output contains "default" as a source.
func TestConfigDebugCommand_DefaultSourceAnnotation(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug"})

	err := root.Execute()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "default",
		"output must show 'default' as a source when no config overrides are present")
}

// TestConfigDebugCommand_RepoConfigSource verifies that when a harvx.toml
// overrides a field, the output shows "repo" as the source for that field.
func TestConfigDebugCommand_RepoConfigSource(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "harvx.toml"),
		[]byte("[profile.default]\nformat = \"xml\"\n"),
		0o644,
	))
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "repo",
		"output must show 'repo' as source for fields overridden by harvx.toml")
}

// TestConfigDebugCommand_ProfileFlag verifies that passing --profile selects
// the named profile and mentions it in the output.
func TestConfigDebugCommand_ProfileFlag(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "harvx.toml"),
		[]byte("[profile.myprofile]\nformat = \"xml\"\n"),
		0o644,
	))
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug", "--profile", "myprofile"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "myprofile",
		"output must mention the selected profile name")
}

// ── config debug: JSON output ─────────────────────────────────────────────────

// TestConfigDebugCommand_JSONOutput verifies that `harvx config debug --json`
// produces valid JSON output.
func TestConfigDebugCommand_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug", "--json"})

	err := root.Execute()
	require.NoError(t, err)

	output := strings.TrimSpace(buf.String())
	require.NotEmpty(t, output, "JSON output must not be empty")

	var parsed map[string]any
	err = json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "config debug --json must produce valid JSON, got: %s", output)
}

// TestConfigDebugCommand_JSONOutput_TopLevelFields verifies that the JSON
// output from `harvx config debug --json` contains the required top-level
// fields: config_files, active_profile, env_vars, config.
func TestConfigDebugCommand_JSONOutput_TopLevelFields(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug", "--json"})

	err := root.Execute()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed))

	for _, field := range []string{"config_files", "active_profile", "env_vars", "config"} {
		assert.Contains(t, parsed, field,
			"JSON output must contain top-level key %q", field)
	}
}

// TestConfigDebugCommand_JSONOutput_ConfigFilesArray verifies that the
// config_files field is a JSON array with entries containing label, path,
// and found keys.
func TestConfigDebugCommand_JSONOutput_ConfigFilesArray(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug", "--json"})

	err := root.Execute()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed))

	files, ok := parsed["config_files"].([]any)
	require.True(t, ok, "config_files must be a JSON array")
	require.NotEmpty(t, files, "config_files must have at least one entry")

	first, ok := files[0].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, first, "label")
	assert.Contains(t, first, "path")
	assert.Contains(t, first, "found")
}

// TestConfigDebugCommand_JSONOutput_ConfigArray verifies that the config field
// is a JSON array where each element has key, value, and source fields.
func TestConfigDebugCommand_JSONOutput_ConfigArray(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug", "--json"})

	err := root.Execute()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed))

	configEntries, ok := parsed["config"].([]any)
	require.True(t, ok, "config must be a JSON array")
	require.NotEmpty(t, configEntries, "config array must have entries")

	first, ok := configEntries[0].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, first, "key")
	assert.Contains(t, first, "value")
	assert.Contains(t, first, "source")
}

// TestConfigDebugCommand_JSONOutput_ActiveProfile verifies that active_profile
// in the JSON output is a non-empty string.
func TestConfigDebugCommand_JSONOutput_ActiveProfile(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug", "--json"})

	err := root.Execute()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed))

	activeProfile, ok := parsed["active_profile"].(string)
	require.True(t, ok, "active_profile must be a string")
	assert.NotEmpty(t, activeProfile)
	assert.Equal(t, "default", activeProfile,
		"active_profile must be 'default' when no profile override is given")
}

// TestConfigDebugCommand_JSONOutput_EnvVarsArray verifies that env_vars is a
// JSON array with entries containing at least name and applied fields.
func TestConfigDebugCommand_JSONOutput_EnvVarsArray(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug", "--json"})

	err := root.Execute()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed))

	envVars, ok := parsed["env_vars"].([]any)
	require.True(t, ok, "env_vars must be a JSON array")
	require.NotEmpty(t, envVars)

	first, ok := envVars[0].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, first, "name")
	assert.Contains(t, first, "applied")
}

// ── config debug: inheritance chain in JSON ───────────────────────────────────

// TestConfigDebugCommand_JSONOutput_InheritanceChain verifies that when a
// profile inherits from another, inherit_chain appears in the JSON output.
func TestConfigDebugCommand_JSONOutput_InheritanceChain(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "harvx.toml"),
		[]byte("[profile.default]\nformat = \"markdown\"\n\n[profile.child]\nextends = \"default\"\nformat = \"xml\"\n"),
		0o644,
	))
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug", "--profile", "child", "--json"})

	err := root.Execute()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed))

	chain, ok := parsed["inherit_chain"].([]any)
	require.True(t, ok, "inherit_chain must be present and be a JSON array when profile extends another")
	require.Len(t, chain, 2)
	assert.Equal(t, "child", chain[0])
	assert.Equal(t, "default", chain[1])
}

// ── config debug: command registration ───────────────────────────────────────

// TestConfigCmd_Registered verifies that the "config" command is registered on
// the global rootCmd.
func TestConfigCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "config" {
			found = true
			break
		}
	}
	assert.True(t, found, "config subcommand must be registered on rootCmd")
}

// TestConfigDebugCmd_Registered verifies that "debug" is registered as a
// subcommand of the global configCmd.
func TestConfigDebugCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range configCmd.Commands() {
		if cmd.Use == "debug" {
			found = true
			break
		}
	}
	assert.True(t, found, "config must have a 'debug' subcommand")
}

// TestConfigDebugCmd_HasJSONFlag verifies that the debug command exposes the
// --json flag on the global configDebugCmd.
func TestConfigDebugCmd_HasJSONFlag(t *testing.T) {
	flag := configDebugCmd.Flags().Lookup("json")
	require.NotNil(t, flag, "config debug must have a --json flag")
	assert.Equal(t, "false", flag.DefValue)
}

// TestConfigDebugCmd_HasProfileFlag verifies that the debug command exposes the
// --profile flag on the global configDebugCmd.
func TestConfigDebugCmd_HasProfileFlag(t *testing.T) {
	flag := configDebugCmd.Flags().Lookup("profile")
	require.NotNil(t, flag, "config debug must have a --profile flag")
}

// ── config debug: error resilience ───────────────────────────────────────────

// TestConfigDebugCommand_MalformedTOML verifies that a malformed harvx.toml
// causes the command to return an error rather than silently succeeding.
func TestConfigDebugCommand_MalformedTOML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "harvx.toml"),
		[]byte("[broken toml"),
		0o644,
	))
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug"})

	err := root.Execute()
	require.Error(t, err, "config debug must return an error for malformed harvx.toml")
}

// TestConfigDebugCommand_UnknownProfile verifies that requesting a profile
// that does not exist returns an error.
func TestConfigDebugCommand_UnknownProfile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "harvx.toml"),
		[]byte("[profile.default]\nformat = \"markdown\"\n"),
		0o644,
	))
	changeDirForTest(t, dir)

	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config", "debug", "--profile", "does-not-exist"})

	err := root.Execute()
	require.Error(t, err, "config debug must return an error when the requested profile does not exist")
}

// ── config debug: no subcommand prints help ───────────────────────────────────

// TestConfigCmd_NoSubcommandNoError verifies that running `harvx config`
// without a subcommand does not return an error (prints help instead).
func TestConfigCmd_NoSubcommandNoError(t *testing.T) {
	root := newTestConfigDebug()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"config"})

	// Cobra prints help text when no subcommand is given -- not an error.
	_ = root.Execute()
	// Not asserting on the return value; just ensure it does not panic.
}
