package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestProfilesFull builds an isolated command tree that includes every
// profiles subcommand (list, init, show, lint, explain) and the config debug
// subcommand, so integration tests exercise the full command surface without
// depending on the global rootCmd state.
func newTestProfilesFull() *cobra.Command {
	root := &cobra.Command{
		Use:           "harvx",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// ── profiles parent ───────────────────────────────────────────────────
	pCmd := &cobra.Command{
		Use:   "profiles",
		Short: "Manage harvx configuration profiles",
	}

	// profiles list
	listCmd := &cobra.Command{
		Use:  "list",
		RunE: runProfilesList,
	}

	// profiles init
	initCmd := &cobra.Command{
		Use:  "init",
		RunE: runProfilesInit,
	}
	initCmd.Flags().String("template", "base", "template name")
	initCmd.Flags().StringP("output", "o", "harvx.toml", "output path")
	initCmd.Flags().Bool("yes", false, "overwrite without prompting")
	if err := initCmd.RegisterFlagCompletionFunc("template", completeTemplateNames); err != nil {
		panic("registering template completion: " + err.Error())
	}

	// profiles show
	showCmd := &cobra.Command{
		Use:               "show [profile]",
		Args:              cobra.MaximumNArgs(1),
		RunE:              runProfilesShow,
		ValidArgsFunction: completeProfileNames,
	}
	showCmd.Flags().Bool("json", false, "output as JSON")

	// profiles lint
	lintCmd := &cobra.Command{
		Use:  "lint",
		RunE: runProfilesLint,
	}
	lintCmd.Flags().String("profile", "", "lint only the specified profile name")

	// profiles explain
	explainCmd := &cobra.Command{
		Use:  "explain <filepath>",
		Args: cobra.ExactArgs(1),
		RunE: runProfilesExplain,
	}
	explainCmd.Flags().String("profile", "", "profile name to explain against")

	pCmd.AddCommand(listCmd, initCmd, showCmd, lintCmd, explainCmd)
	root.AddCommand(pCmd)

	// ── config parent ─────────────────────────────────────────────────────
	cfgCmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management commands",
	}

	dbgCmd := &cobra.Command{
		Use:  "debug",
		RunE: runConfigDebug,
	}
	dbgCmd.Flags().Bool("json", false, "output as structured JSON")
	dbgCmd.Flags().String("profile", "", "profile name to debug")

	cfgCmd.AddCommand(dbgCmd)
	root.AddCommand(cfgCmd)

	return root
}

// runCmd is a convenience helper that wires output capture, sets args, and
// executes the root command, returning both the combined stdout/stderr output
// and any error from Execute.
func runCmd(t *testing.T, root *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// ── TestCLI_ProfilesList_DefaultOnly ─────────────────────────────────────────

// TestCLI_ProfilesList_DefaultOnly verifies that with no harvx.toml in CWD the
// list output still contains the built-in "default" profile and labels it
// "built-in".
func TestCLI_ProfilesList_DefaultOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	out, err := runCmd(t, root, "profiles", "list")

	require.NoError(t, err)
	assert.Contains(t, out, "default",
		"output must contain the built-in default profile name")
	assert.Contains(t, out, "built-in",
		"output must label the default profile as 'built-in'")
}

// ── TestCLI_ProfilesList_WithRepoConfig ───────────────────────────────────────

// TestCLI_ProfilesList_WithRepoConfig verifies that a profile defined in a
// harvx.toml present in the CWD appears in the list output.
func TestCLI_ProfilesList_WithRepoConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	content := `
[profile.myprofile]
format = "markdown"
max_tokens = 128000
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	out, err := runCmd(t, root, "profiles", "list")

	require.NoError(t, err)
	assert.Contains(t, out, "myprofile",
		"output must contain the repo-level profile name")
}

// ── TestCLI_ProfilesShow_Default ─────────────────────────────────────────────

// TestCLI_ProfilesShow_Default verifies that `profiles show default` produces
// the expected header and mentions the "markdown" format.
func TestCLI_ProfilesShow_Default(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	out, err := runCmd(t, root, "profiles", "show", "default")

	require.NoError(t, err)
	assert.Contains(t, out, "# Resolved profile: default",
		"output must start with the resolved profile header")
	assert.Contains(t, out, "markdown",
		"output must mention the default format value 'markdown'")
}

// ── TestCLI_ProfilesShow_WithInheritedProfile ─────────────────────────────────

// TestCLI_ProfilesShow_WithInheritedProfile verifies that a profile that
// extends "default" is fully resolved and the profile name appears in the
// output header.
func TestCLI_ProfilesShow_WithInheritedProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	content := `
[profile.myapi]
extends = "default"
format  = "xml"
max_tokens = 64000
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	out, err := runCmd(t, root, "profiles", "show", "myapi")

	require.NoError(t, err)
	assert.Contains(t, out, "myapi",
		"output must contain the requested profile name")
}

// ── TestCLI_ProfilesLint_CleanConfig ─────────────────────────────────────────

// TestCLI_ProfilesLint_CleanConfig verifies that linting a well-formed
// harvx.toml exits with code 0 and reports no issues.
func TestCLI_ProfilesLint_CleanConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	content := `
[profile.default]
format     = "markdown"
max_tokens = 128000
tokenizer  = "cl100k_base"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	out, err := runCmd(t, root, "profiles", "lint")

	require.NoError(t, err, "linting a clean config must return exit 0")
	assert.Contains(t, out, "No issues found",
		"output must report 'No issues found' for a valid config")
}

// ── TestCLI_ProfilesLint_BrokenConfig ────────────────────────────────────────

// TestCLI_ProfilesLint_BrokenConfig verifies that linting a harvx.toml with
// an invalid format value returns an error (non-zero exit) and the output
// contains the error indicator.
func TestCLI_ProfilesLint_BrokenConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	// "html" is not a valid format; lint must treat this as an error.
	content := `
[profile.broken]
format = "html"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	out, err := runCmd(t, root, "profiles", "lint")

	require.Error(t, err, "linting an invalid config must return a non-nil error")
	assert.Contains(t, out, "X",
		"output must contain the error indicator 'X' for invalid config values")
}

// ── TestCLI_ProfilesExplain_SomeFile ─────────────────────────────────────────

// TestCLI_ProfilesExplain_SomeFile verifies that `profiles explain` produces
// explanation output for a typical source file path.
func TestCLI_ProfilesExplain_SomeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	out, err := runCmd(t, root, "profiles", "explain", "src/main.ts")

	require.NoError(t, err)
	assert.Contains(t, out, "Explaining: src/main.ts",
		"output must show the file path being explained")
	assert.Contains(t, out, "Rule trace:",
		"output must contain a rule trace section")
}

// ── TestCLI_ConfigDebug_Output ────────────────────────────────────────────────

// TestCLI_ConfigDebug_Output verifies that `config debug` produces the
// expected header and section markers.
func TestCLI_ConfigDebug_Output(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	out, err := runCmd(t, root, "config", "debug")

	require.NoError(t, err)
	assert.Contains(t, out, "Harvx Configuration Debug",
		"output must contain the standard header 'Harvx Configuration Debug'")
	assert.Contains(t, out, "Resolved Configuration:",
		"output must contain the 'Resolved Configuration:' section")
}

// ── Full sequence: init -> list -> show -> lint ───────────────────────────────

// TestCLI_FullSequence_InitListShowLint exercises the complete happy-path
// workflow: generate a config with `init`, verify the profile appears in
// `list`, inspect it with `show`, then validate it with `lint`.  Each step
// uses a fresh command tree to ensure there is no cross-command state
// pollution.
func TestCLI_FullSequence_InitListShowLint(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	outPath := filepath.Join(dir, "harvx.toml")
	changeDirForTest(t, dir)

	// Step 1: init with the go-cli template.
	{
		root := newTestProfilesFull()
		out, err := runCmd(t, root, "profiles", "init", "--template", "go-cli", "--output", outPath)
		require.NoError(t, err, "profiles init must succeed")
		assert.Contains(t, out, "Created", "init output must confirm file creation")
	}

	// Step 2: list -- default must always appear alongside template profiles.
	{
		root := newTestProfilesFull()
		out, err := runCmd(t, root, "profiles", "list")
		require.NoError(t, err, "profiles list must succeed after init")
		assert.Contains(t, out, "default", "default profile must always appear in list")
	}

	// Step 3: show the built-in default.
	{
		root := newTestProfilesFull()
		out, err := runCmd(t, root, "profiles", "show", "default")
		require.NoError(t, err, "profiles show default must succeed")
		assert.Contains(t, out, "# Resolved profile: default")
	}

	// Step 4: lint the generated config -- the go-cli template must be valid.
	{
		root := newTestProfilesFull()
		_, err := runCmd(t, root, "profiles", "lint")
		require.NoError(t, err, "profiles lint must succeed for a template-generated config")
	}
}

// ── Edge cases ────────────────────────────────────────────────────────────────

// TestCLI_ProfilesShow_UnknownProfile verifies that requesting a profile that
// does not exist returns an error without panicking.
func TestCLI_ProfilesShow_UnknownProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	_, err := runCmd(t, root, "profiles", "show", "no-such-profile-xyz")

	require.Error(t, err, "show with an unknown profile must return an error")
}

// TestCLI_ProfilesExplain_ExcludedPath verifies that a path matching the
// built-in ignore list (e.g. "node_modules") is reported as EXCLUDED.
func TestCLI_ProfilesExplain_ExcludedPath(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	out, err := runCmd(t, root, "profiles", "explain", "node_modules")

	require.NoError(t, err)
	assert.Contains(t, out, "EXCLUDED",
		"output must report EXCLUDED for a path that matches the built-in ignore list")
}

// TestCLI_ProfilesLint_NoConfig verifies that running lint in a directory
// with no harvx.toml uses built-in defaults and reports no issues.
func TestCLI_ProfilesLint_NoConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	out, err := runCmd(t, root, "profiles", "lint")

	require.NoError(t, err,
		"lint with no harvx.toml must succeed (falls back to built-in defaults)")
	assert.Contains(t, out, "No issues found")
}

// TestCLI_ConfigDebug_WithRepoOverride verifies that when a harvx.toml
// overrides a field the debug output annotates that field with "repo" as the
// source.
func TestCLI_ConfigDebug_WithRepoOverride(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "harvx.toml"),
		[]byte("[profile.default]\nformat = \"xml\"\n"),
		0o644,
	))
	changeDirForTest(t, dir)

	root := newTestProfilesFull()
	out, err := runCmd(t, root, "config", "debug")

	require.NoError(t, err)
	assert.Contains(t, out, "repo",
		"output must show 'repo' as source for fields overridden by harvx.toml")
}
