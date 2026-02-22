package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/harvx/harvx/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestExplain builds an isolated command tree containing only
// `harvx profiles explain` so each test gets a fresh command state.
func newTestExplain() *cobra.Command {
	root := &cobra.Command{
		Use:           "harvx",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	pCmd := &cobra.Command{Use: "profiles"}
	explainCmd := &cobra.Command{
		Use:  "explain <filepath>",
		Args: cobra.ExactArgs(1),
		RunE: runProfilesExplain,
	}
	explainCmd.Flags().String("profile", "", "profile name")
	pCmd.AddCommand(explainCmd)
	root.AddCommand(pCmd)
	return root
}

// ── profiles explain ──────────────────────────────────────────────────────────

// TestProfilesExplain_IncludedFile verifies that a .go file not in ignore lists
// shows "INCLUDED" in the output.
func TestProfilesExplain_IncludedFile(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestExplain()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "explain", "src/main.go"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "INCLUDED",
		"output must show INCLUDED for a regular source file")
}

// TestProfilesExplain_ExcludedFile verifies that a path matching the default
// ignore pattern shows "EXCLUDED" in the output. The default ignore list
// includes "node_modules" which matches the literal path "node_modules".
func TestProfilesExplain_ExcludedFile(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestExplain()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	// Use the literal "node_modules" which matches the default ignore pattern.
	root.SetArgs([]string{"profiles", "explain", "node_modules"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "EXCLUDED",
		"output must show EXCLUDED for node_modules path")
}

// TestProfilesExplain_ProfileFlagUsed verifies that passing --profile default
// works without error.
func TestProfilesExplain_ProfileFlagUsed(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestExplain()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "explain", "--profile", "default", "go.mod"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "default",
		"output must mention the default profile name")
}

// TestProfilesExplain_OutputContainsRuleTrace verifies that the output always
// contains the "Rule trace:" header.
func TestProfilesExplain_OutputContainsRuleTrace(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestExplain()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "explain", "internal/config/explain.go"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Rule trace:",
		"output must always contain 'Rule trace:' header")
}

// TestProfilesExplain_ExplainingLineShown verifies that the "Explaining:" line
// with the file path is always printed.
func TestProfilesExplain_ExplainingLineShown(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestExplain()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "explain", "cmd/harvx/main.go"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Explaining: cmd/harvx/main.go")
}

// TestProfilesExplain_RequiresArg verifies that running the explain command
// without a filepath argument returns an error.
func TestProfilesExplain_RequiresArg(t *testing.T) {
	root := newTestExplain()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "explain"})

	err := root.Execute()
	require.Error(t, err, "explain without a filepath argument must return an error")
}

// TestProfilesExplain_RepoProfileUsed verifies that when a harvx.toml with a
// named profile is present in the current directory, --profile resolves it.
func TestProfilesExplain_RepoProfileUsed(t *testing.T) {
	dir := t.TempDir()
	content := `
[profile.myprofile]
format = "markdown"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(content), 0o644))
	changeDirForTest(t, dir)

	root := newTestExplain()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"profiles", "explain", "--profile", "myprofile", "src/app.go"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "myprofile",
		"output must reference the custom profile name")
}

// TestProfilesExplain_ExcludedByShows verifies that the "Excluded by:" field
// appears in output when a file is excluded by an ignore pattern.
func TestProfilesExplain_ExcludedByShows(t *testing.T) {
	dir := t.TempDir()
	changeDirForTest(t, dir)

	root := newTestExplain()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	// "dist" is in the default ignore list and matches literally.
	root.SetArgs([]string{"profiles", "explain", "dist"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Excluded by:",
		"output must contain 'Excluded by:' when file is excluded")
}

// TestProfilesExplainCmd_Registered verifies that the explain subcommand is
// registered on the global profilesCmd.
func TestProfilesExplainCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range profilesCmd.Commands() {
		if cmd.Use == "explain <filepath>" {
			found = true
			break
		}
	}
	assert.True(t, found, "profiles command must have an 'explain <filepath>' subcommand")
}

// ── formatTier ────────────────────────────────────────────────────────────────

// TestFormatTier_Values verifies the human-readable string for all tier values
// 0-5 and -1 (untiered).
func TestFormatTier_Values(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tier int
		want string
	}{
		{name: "tier 0", tier: 0, want: "0 (critical priority)"},
		{name: "tier 1", tier: 1, want: "1 (high priority)"},
		{name: "tier 2", tier: 2, want: "2 (medium priority)"},
		{name: "tier 3", tier: 3, want: "3 (low priority)"},
		{name: "tier 4", tier: 4, want: "4 (documentation)"},
		{name: "tier 5", tier: 5, want: "5 (lowest priority)"},
		{name: "tier -1 untiered", tier: -1, want: "untiered (default inclusion)"},
		{name: "tier 99 untiered", tier: 99, want: "untiered (default inclusion)"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatTier(tt.tier)
			assert.Equal(t, tt.want, got,
				"formatTier(%d) must return %q", tt.tier, tt.want)
		})
	}
}

// ── formatCompression ─────────────────────────────────────────────────────────

// TestFormatCompression_Supported verifies that a non-empty language name
// produces "yes (... supported)".
func TestFormatCompression_Supported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		lang string
		want string
	}{
		{lang: "Go", want: "yes (Go supported)"},
		{lang: "TypeScript", want: "yes (TypeScript supported)"},
		{lang: "Python", want: "yes (Python supported)"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.lang, func(t *testing.T) {
			t.Parallel()
			got := formatCompression(tt.lang)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestFormatCompression_Unsupported verifies that an empty language name
// produces "no (file type not supported)".
func TestFormatCompression_Unsupported(t *testing.T) {
	t.Parallel()

	got := formatCompression("")
	assert.Equal(t, "no (file type not supported)", got)
}

// ── formatRedaction ───────────────────────────────────────────────────────────

// TestFormatRedaction verifies the human-readable redaction status strings
// by calling formatRedaction with config.ExplainResult values.
func TestFormatRedaction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result config.ExplainResult
		want   string
	}{
		{
			name:   "excluded file",
			result: config.ExplainResult{Included: false},
			want:   "n/a (file excluded)",
		},
		{
			name:   "included with redaction on",
			result: config.ExplainResult{Included: true, RedactionOn: true},
			want:   "enabled (not in exclude_paths)",
		},
		{
			name:   "included with redaction off",
			result: config.ExplainResult{Included: true, RedactionOn: false},
			want:   "disabled",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatRedaction(tt.result)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestFormatPriority verifies the priority status strings.
func TestFormatPriority(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "in priority_files", formatPriority(true))
	assert.Equal(t, "not in priority_files", formatPriority(false))
}
