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

// newTestLint builds an isolated command tree containing only
// `harvx profiles lint` so each test gets a fresh command state.
func newTestLint() *cobra.Command {
	root := &cobra.Command{
		Use:           "harvx",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	pCmd := &cobra.Command{Use: "profiles"}
	lintCmd := &cobra.Command{
		Use:  "lint",
		RunE: runProfilesLint,
	}
	lintCmd.Flags().String("profile", "", "lint only the specified profile name")
	pCmd.AddCommand(lintCmd)
	root.AddCommand(pCmd)
	return root
}

// changeDirForTest changes the working directory to dir for the duration of
// the test, restoring the original directory in a cleanup function.
func changeDirForTest(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		if chErr := os.Chdir(orig); chErr != nil {
			t.Logf("cleanup: chdir back failed: %v", chErr)
		}
	})
}

// ── profiles lint ─────────────────────────────────────────────────────────────

// TestProfilesLint_CleanConfigNoErrors verifies that a valid configuration
// without any issues produces exit 0 and "No issues found" in the output.
func TestProfilesLint_CleanConfigNoErrors(t *testing.T) {
	dir := t.TempDir()
	content := `
[profile.default]
format = "markdown"
max_tokens = 128000
tokenizer = "cl100k_base"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestLint()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "lint"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No issues found")
}

// TestProfilesLint_InvalidFormatReturnsError verifies that a profile with an
// invalid format value causes exit code 1 and prints "X" in the output.
func TestProfilesLint_InvalidFormatReturnsError(t *testing.T) {
	dir := t.TempDir()
	content := `
[profile.bad]
format = "html"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestLint()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "lint"})

	err := root.Execute()
	require.Error(t, err, "invalid format must cause a non-nil error return")
	assert.Contains(t, buf.String(), "X",
		"output must contain 'X' icon for errors")
}

// TestProfilesLint_OverlappingTiersWarning verifies that duplicate patterns in
// multiple tiers cause a warning output containing "!".
func TestProfilesLint_OverlappingTiersWarning(t *testing.T) {
	dir := t.TempDir()
	content := `
[profile.overlap]
format = "markdown"

[profile.overlap.relevance]
tier_0 = ["go.mod", "internal/**"]
tier_1 = ["go.mod"]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestLint()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "lint"})

	// Warnings only -- must not fail.
	err := root.Execute()
	require.NoError(t, err, "warnings alone must not cause a non-nil error")
	assert.Contains(t, buf.String(), "!",
		"output must contain '!' icon for warnings")
}

// TestProfilesLint_ProfileFlagFiltersToOneProfile verifies that the --profile
// flag restricts linting to only the named profile.
func TestProfilesLint_ProfileFlagFiltersToOneProfile(t *testing.T) {
	dir := t.TempDir()
	content := `
[profile.good]
format = "markdown"

[profile.bad]
format = "html"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	// Lint only the "good" profile -- should report no errors.
	root := newTestLint()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "lint", "--profile", "good"})

	err := root.Execute()
	require.NoError(t, err, "linting only the clean profile must succeed")
	assert.Contains(t, buf.String(), "No issues found")
}

// TestProfilesLint_ProfileFlagUnknownProfile verifies that specifying a
// non-existent profile name with --profile returns an error.
func TestProfilesLint_ProfileFlagUnknownProfile(t *testing.T) {
	dir := t.TempDir()
	content := `
[profile.myprofile]
format = "markdown"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestLint()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "lint", "--profile", "nonexistent"})

	err := root.Execute()
	require.Error(t, err, "unknown profile must return an error")
	assert.Contains(t, err.Error(), "nonexistent")
}

// TestProfilesLint_OutputFormatHasIcons verifies that errors use "X" and
// warnings use "!" in the output.
func TestProfilesLint_OutputFormatHasIcons(t *testing.T) {
	dir := t.TempDir()
	content := `
[profile.p]
format = "html"

[profile.q]
format = "markdown"

[profile.q.relevance]
tier_0 = ["go.mod"]
tier_1 = ["go.mod"]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestLint()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "lint"})

	_ = root.Execute()
	output := buf.String()

	assert.Contains(t, output, "X", "error icon 'X' must appear")
	assert.Contains(t, output, "!", "warning icon '!' must appear")
}

// TestProfilesLint_ExitCode1WhenErrors verifies that RunE returns a non-nil
// error when lint errors are present, which causes cobra to exit 1.
func TestProfilesLint_ExitCode1WhenErrors(t *testing.T) {
	dir := t.TempDir()
	content := `
[profile.broken]
format = "pdf"
tokenizer = "gpt4"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestLint()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "lint"})

	err := root.Execute()
	require.Error(t, err, "lint with errors must return a non-nil error")
}

// TestProfilesLint_ExitCode0WhenOnlyWarnings verifies that warnings alone do
// not cause a non-nil error return from RunE.
func TestProfilesLint_ExitCode0WhenOnlyWarnings(t *testing.T) {
	dir := t.TempDir()
	// Large max_tokens triggers a soft-cap warning without any hard error.
	content := `
[profile.largecap]
format = "markdown"
max_tokens = 600000
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestLint()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "lint"})

	err := root.Execute()
	require.NoError(t, err, "warnings-only lint must return nil (exit 0)")
	output := buf.String()
	assert.Contains(t, output, "!", "output must contain '!' for the warning")
}

// TestProfilesLint_NoConfigUsesDefaults verifies that running lint in a
// directory with no harvx.toml reports "No issues found" (using built-in
// defaults, which are valid).
func TestProfilesLint_NoConfigUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestLint()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "lint"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "No issues found")
}

// TestProfilesLint_SummaryLineShown verifies that the summary line with error
// and warning counts is shown when issues are found.
func TestProfilesLint_SummaryLineShown(t *testing.T) {
	dir := t.TempDir()
	content := `
[profile.bad]
format = "html"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestLint()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "lint"})

	_ = root.Execute()
	output := buf.String()

	assert.Contains(t, output, "Result:", "output must contain a summary 'Result:' line")
	assert.Contains(t, output, "error(s)", "summary must mention error count")
}

// TestProfilesLintCmd_Registered verifies that the lint subcommand is
// registered on the global profilesCmd.
func TestProfilesLintCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range profilesCmd.Commands() {
		if cmd.Use == "lint" {
			found = true
			break
		}
	}
	assert.True(t, found, "profiles command must have a 'lint' subcommand")
}
