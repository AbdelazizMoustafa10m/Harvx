package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harvx/harvx/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldLaunchInteractive_ExplicitFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flagVal  bool
		expected bool
	}{
		{"explicit true", true, true},
		{"explicit false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().BoolP("interactive", "i", false, "")
			// Simulate the flag being explicitly set.
			require.NoError(t, cmd.Flags().Set("interactive", boolToStr(tt.flagVal)))

			fv := &config.FlagValues{Interactive: tt.flagVal, Dir: t.TempDir()}
			assert.Equal(t, tt.expected, ShouldLaunchInteractive(cmd, fv))
		})
	}
}

func TestShouldLaunchInteractive_SmartDefault_NoConfig(t *testing.T) {
	// Cannot be parallel: overrides package-level isTerminal.
	origIsTerminal := isTerminal
	isTerminal = func(_ uintptr) bool { return true }
	defer func() { isTerminal = origIsTerminal }()

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().BoolP("interactive", "i", false, "")
	// Do NOT set the flag -- smart default applies.

	dir := t.TempDir() // empty dir, no harvx.toml
	fv := &config.FlagValues{Dir: dir}

	assert.True(t, ShouldLaunchInteractive(cmd, fv),
		"should launch TUI when no harvx.toml in tree and terminal is attached")
}

func TestShouldLaunchInteractive_SmartDefault_WithConfig(t *testing.T) {
	// Cannot be parallel: overrides package-level isTerminal.
	origIsTerminal := isTerminal
	isTerminal = func(_ uintptr) bool { return true }
	defer func() { isTerminal = origIsTerminal }()

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().BoolP("interactive", "i", false, "")

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte("[default]\n"), 0o644))

	fv := &config.FlagValues{Dir: dir}

	assert.False(t, ShouldLaunchInteractive(cmd, fv),
		"should NOT launch TUI when harvx.toml exists")
}

func TestShouldLaunchInteractive_SmartDefault_NoTerminal(t *testing.T) {
	// Cannot be parallel: overrides package-level isTerminal.
	origIsTerminal := isTerminal
	isTerminal = func(_ uintptr) bool { return false }
	defer func() { isTerminal = origIsTerminal }()

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().BoolP("interactive", "i", false, "")

	dir := t.TempDir() // no harvx.toml
	fv := &config.FlagValues{Dir: dir}

	assert.False(t, ShouldLaunchInteractive(cmd, fv),
		"should NOT launch TUI when stdin is not a terminal")
}

func TestHasConfigInTree_Found(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harvx.toml"), []byte(""), 0o644))

	assert.True(t, hasConfigInTree(dir))
}

func TestHasConfigInTree_HiddenVariant(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".harvx.toml"), []byte(""), 0o644))

	assert.True(t, hasConfigInTree(dir))
}

func TestHasConfigInTree_ParentDir(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	child := filepath.Join(parent, "subdir")
	require.NoError(t, os.Mkdir(child, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(parent, "harvx.toml"), []byte(""), 0o644))

	assert.True(t, hasConfigInTree(child))
}

func TestHasConfigInTree_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	assert.False(t, hasConfigInTree(dir))
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
