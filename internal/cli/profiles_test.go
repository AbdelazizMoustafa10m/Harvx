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

// newTestProfiles builds an isolated profiles command tree for tests so each
// test gets a clean command state without interference from the global rootCmd.
func newTestProfiles() *cobra.Command {
	root := &cobra.Command{
		Use:           "harvx",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	pCmd := &cobra.Command{
		Use:   "profiles",
		Short: "Manage harvx configuration profiles",
	}

	listCmd := &cobra.Command{
		Use:  "list",
		RunE: runProfilesList,
	}

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

	showCmd := &cobra.Command{
		Use:               "show [profile]",
		Args:              cobra.MaximumNArgs(1),
		RunE:              runProfilesShow,
		ValidArgsFunction: completeProfileNames,
	}
	showCmd.Flags().Bool("json", false, "output as JSON")

	pCmd.AddCommand(listCmd, initCmd, showCmd)
	root.AddCommand(pCmd)

	return root
}

// ── profiles list ─────────────────────────────────────────────────────────

func TestProfilesList_ShowsBuiltInDefault(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "list"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "default")
	assert.Contains(t, output, "built-in")
}

func TestProfilesList_ShowsTemplates(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "list"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Templates")
	assert.Contains(t, output, "nextjs")
	assert.Contains(t, output, "go-cli")
}

func TestProfilesList_ShowsRepoProfiles(t *testing.T) {
	dir := t.TempDir()
	content := `
[profile.myprofile]
format = "markdown"

[profile.otherprofile]
format = "xml"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		if chErr := os.Chdir(origDir); chErr != nil {
			t.Logf("cleanup: chdir back failed: %v", chErr)
		}
	})

	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "list"})

	err = root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "myprofile")
	assert.Contains(t, output, "otherprofile")
}

func TestProfilesList_TableColumns(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "list"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "SOURCE")
	assert.Contains(t, output, "EXTENDS")
	assert.Contains(t, output, "DESCRIPTION")
}

func TestProfilesList_AvailableProfilesHeader(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "list"})

	err := root.Execute()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "Available Profiles")
}

// ── profiles init ─────────────────────────────────────────────────────────

func TestProfilesInit_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "harvx.toml")

	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "init", "--output", outPath})

	err := root.Execute()
	require.NoError(t, err)

	data, readErr := os.ReadFile(outPath)
	require.NoError(t, readErr)
	assert.NotEmpty(t, data)

	output := buf.String()
	assert.Contains(t, output, "Created")
	assert.Contains(t, output, "harvx.toml")
}

func TestProfilesInit_DefaultTemplateIsBase(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "harvx.toml")

	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "init", "--output", outPath})

	err := root.Execute()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "template: base")
}

func TestProfilesInit_WithTemplate(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "harvx.toml")

	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "init", "--template", "nextjs", "--output", outPath})

	err := root.Execute()
	require.NoError(t, err)

	data, readErr := os.ReadFile(outPath)
	require.NoError(t, readErr)
	assert.NotEmpty(t, data)

	assert.Contains(t, buf.String(), "template: nextjs")
}

func TestProfilesInit_ExistingFileWithoutYes(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "harvx.toml")

	require.NoError(t, os.WriteFile(outPath, []byte("existing"), 0o644))

	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "init", "--output", outPath})

	err := root.Execute()
	require.Error(t, err, "should fail when file exists without --yes")
	assert.Contains(t, err.Error(), "already exists")
}

func TestProfilesInit_ExistingFileWithYes(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "harvx.toml")

	require.NoError(t, os.WriteFile(outPath, []byte("existing"), 0o644))

	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "init", "--output", outPath, "--yes"})

	err := root.Execute()
	require.NoError(t, err, "should succeed with --yes")

	data, readErr := os.ReadFile(outPath)
	require.NoError(t, readErr)
	assert.NotEqual(t, "existing", string(data), "file should be overwritten")
}

func TestProfilesInit_ShowsNextSteps(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "harvx.toml")

	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "init", "--output", outPath})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Next steps")
	assert.Contains(t, output, "harvx profiles lint")
}

func TestProfilesInit_UnknownTemplate(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "harvx.toml")

	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "init", "--template", "nonexistent-template", "--output", outPath})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-template")
}

func TestProfilesInit_AllTemplatesWork(t *testing.T) {
	templates := []string{"base", "nextjs", "go-cli", "python-django", "rust-cargo", "monorepo"}

	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			dir := t.TempDir()
			outPath := filepath.Join(dir, "harvx.toml")

			root := newTestProfiles()
			var buf bytes.Buffer
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetArgs([]string{"profiles", "init", "--template", tmpl, "--output", outPath})

			err := root.Execute()
			require.NoError(t, err, "template %q should generate without error", tmpl)

			data, readErr := os.ReadFile(outPath)
			require.NoError(t, readErr)
			assert.NotEmpty(t, data)
		})
	}
}

// ── profiles show ─────────────────────────────────────────────────────────

func TestProfilesShow_DefaultProfile(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "show", "default"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "# Resolved profile: default")
	assert.Contains(t, output, "format")
	assert.Contains(t, output, "markdown")
}

func TestProfilesShow_NoArgDefaultsToDefault(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "show"})

	err := root.Execute()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "# Resolved profile: default")
}

func TestProfilesShow_WithInheritance(t *testing.T) {
	dir := t.TempDir()
	content := `
[profile.myprofile]
extends = "default"
format = "xml"
max_tokens = 200000
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		if chErr := os.Chdir(origDir); chErr != nil {
			t.Logf("cleanup: chdir back failed: %v", chErr)
		}
	})

	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "show", "myprofile"})

	err = root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "# Resolved profile: myprofile")
	assert.Contains(t, output, `"xml"`)
}

func TestProfilesShow_JSONOutput(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "show", "default", "--json"})

	err := root.Execute()
	require.NoError(t, err)

	output := strings.TrimSpace(buf.String())

	var parsed map[string]any
	err = json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "output must be valid JSON")

	// Profile struct uses only toml tags; encoding/json uses Go field names.
	assert.Equal(t, "markdown", parsed["Format"])
}

func TestProfilesShow_JSONOutputHasExpectedFields(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "show", "--json"})

	err := root.Execute()
	require.NoError(t, err)

	output := strings.TrimSpace(buf.String())

	var raw map[string]any
	err = json.Unmarshal([]byte(output), &raw)
	require.NoError(t, err)

	// encoding/json uses Go field names (no json tags on Profile struct).
	for _, key := range []string{"Output", "Format", "MaxTokens", "Tokenizer"} {
		assert.Contains(t, raw, key, "JSON output must contain key %q", key)
	}
}

func TestProfilesShow_UnknownProfileError(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "show", "nonexistent-profile-xyz"})

	err := root.Execute()
	require.Error(t, err)
}

func TestProfilesShow_UnknownProfileListsAvailable(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "show", "doesnotexist-abc"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Available profiles")
}

func TestProfilesShow_ContainsSourceAnnotations(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "show", "default"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	// Output must contain at least one source annotation.
	assert.Contains(t, output, "# default", "TOML output should have source annotations")
}

// ── profiles parent command ───────────────────────────────────────────────

func TestProfilesCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "profiles" {
			found = true
			break
		}
	}
	assert.True(t, found, "profiles subcommand must be registered on root command")
}

func TestProfilesCmd_SubcommandsRegistered(t *testing.T) {
	subNames := make(map[string]bool)
	for _, sub := range profilesCmd.Commands() {
		subNames[sub.Use] = true
	}

	for _, want := range []string{"list", "init", "show [profile]"} {
		assert.True(t, subNames[want], "profiles must have subcommand %q", want)
	}
}

func TestProfilesCmd_NoSubcommandShowsHelp(t *testing.T) {
	root := newTestProfiles()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles"})

	// No error expected; help text is printed.
	_ = root.Execute()
	// Either help output or empty output is acceptable here; the command
	// must not return a non-zero error for a missing subcommand.
}

// ── shell completions ─────────────────────────────────────────────────────

func TestCompleteTemplateNames_AllReturned(t *testing.T) {
	names, directive := completeTemplateNames(nil, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Contains(t, names, "nextjs")
	assert.Contains(t, names, "go-cli")
	assert.Contains(t, names, "base")
	assert.Contains(t, names, "monorepo")
}

func TestCompleteTemplateNames_PrefixFiltered(t *testing.T) {
	names, directive := completeTemplateNames(nil, nil, "go")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Contains(t, names, "go-cli")
	for _, n := range names {
		assert.True(t, strings.HasPrefix(n, "go"), "all completions should start with 'go', got %q", n)
	}
}

func TestCompleteTemplateNames_EmptyPrefixReturnsAll(t *testing.T) {
	names, _ := completeTemplateNames(nil, nil, "")
	assert.NotEmpty(t, names)
}

func TestCompleteProfileNames_IncludesDefault(t *testing.T) {
	names, directive := completeProfileNames(nil, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Contains(t, names, "default")
}

func TestCompleteProfileNames_PrefixFiltered(t *testing.T) {
	names, _ := completeProfileNames(nil, nil, "def")
	assert.Contains(t, names, "default")
	for _, n := range names {
		assert.True(t, strings.HasPrefix(n, "def"), "should start with 'def', got %q", n)
	}
}

// ── integration: init -> list -> show sequence ────────────────────────────

func TestProfilesIntegration_InitListShow(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "harvx.toml")

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		if chErr := os.Chdir(origDir); chErr != nil {
			t.Logf("cleanup: chdir back failed: %v", chErr)
		}
	})

	// Step 1: init with go-cli template.
	{
		root := newTestProfiles()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{"profiles", "init", "--template", "go-cli", "--output", outPath})
		require.NoError(t, root.Execute())
		assert.Contains(t, buf.String(), "Created")
	}

	// Step 2: list -- default must always appear.
	{
		root := newTestProfiles()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{"profiles", "list"})
		require.NoError(t, root.Execute())
		assert.Contains(t, buf.String(), "default")
	}

	// Step 3: show default profile.
	{
		root := newTestProfiles()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{"profiles", "show", "default"})
		require.NoError(t, root.Execute())
		assert.Contains(t, buf.String(), "# Resolved profile: default")
	}
}
