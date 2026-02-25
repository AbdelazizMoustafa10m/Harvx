//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExitCode_Success verifies that basic commands return exit code 0.
func TestExitCode_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "version",
			args: []string{"version"},
		},
		{
			name: "brief --stdout",
			args: []string{"brief", "--stdout"},
		},
		{
			name: "preview --json",
			args: []string{"preview", "--json"},
		},
		{
			name: "workspace --stdout",
			args: []string{"workspace", "--stdout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, code := runHarvx(t, tt.args)
			assert.Equal(t, 0, code, "%s should exit 0", tt.name)
		})
	}
}

// TestExitCode_InvalidFlag verifies that an unknown flag returns exit code 1.
func TestExitCode_InvalidFlag(t *testing.T) {
	t.Parallel()

	_, stderr, code := runHarvx(t, []string{"brief", "--nonexistent-flag-xyz"})

	assert.Equal(t, 1, code, "unknown flag should return exit code 1")
	assert.NotEmpty(t, stderr, "stderr should contain an error message for unknown flag")
}

// TestExitCode_InvalidProfile verifies that referencing a non-existent profile
// name handles gracefully (the --profile flag selects a profile from harvx.toml).
func TestExitCode_InvalidProfile(t *testing.T) {
	t.Parallel()

	// Using a non-existent profile name. The behavior depends on whether
	// the config layer treats it as fatal or falls back to defaults.
	_, _, code := runHarvx(t, []string{"brief", "--stdout", "--profile", "nonexistent-profile-xyz"})

	// Profile resolution may or may not fail depending on implementation.
	// The key assertion: the command should not panic (it should return a
	// defined exit code).
	assert.Contains(t, []int{0, 1}, code,
		"invalid profile should exit 0 (graceful fallback) or 1 (error), not crash")
}

// TestExitCode_AssertIncludePass verifies that --assert-include with an existing
// pattern (README.md) returns exit code 0.
func TestExitCode_AssertIncludePass(t *testing.T) {
	t.Parallel()

	_, _, code := runHarvx(t, []string{
		"brief", "--stdout", "--assert-include", "README.md",
	})

	assert.Equal(t, 0, code,
		"assert-include with matching pattern should return exit code 0")
}

// TestExitCode_AssertIncludeFail verifies that --assert-include with a missing
// pattern returns exit code 1 and the error message includes the pattern.
func TestExitCode_AssertIncludeFail(t *testing.T) {
	t.Parallel()

	_, stderr, code := runHarvx(t, []string{
		"brief", "--stdout", "--assert-include", "nonexistent-file.xyz",
	})

	assert.Equal(t, 1, code,
		"assert-include with missing pattern should return exit code 1")
	// The error message should include the pattern or mention "assert-include".
	hasPatternRef := strings.Contains(stderr, "nonexistent-file.xyz") ||
		strings.Contains(stderr, "assert-include")
	assert.True(t, hasPatternRef,
		"error message should reference the failed pattern, got stderr: %s",
		truncateForTest(stderr, 500))
}

// TestExitCode_VersionSuccess verifies that version always returns exit code 0.
func TestExitCode_VersionSuccess(t *testing.T) {
	t.Parallel()

	_, _, code := runHarvx(t, []string{"version"})
	assert.Equal(t, 0, code, "version should always return exit code 0")

	_, _, codeJSON := runHarvx(t, []string{"version", "--json"})
	assert.Equal(t, 0, codeJSON, "version --json should always return exit code 0")
}

// TestExitCode_ReviewSliceMissingFlags verifies that review-slice without
// the required --base and --head flags returns exit code 1.
func TestExitCode_ReviewSliceMissingFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no flags at all",
			args: []string{"review-slice", "--stdout"},
		},
		{
			name: "missing --head",
			args: []string{"review-slice", "--base", "abc123", "--stdout"},
		},
		{
			name: "missing --base",
			args: []string{"review-slice", "--head", "HEAD", "--stdout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, stderr, code := runHarvx(t, tt.args)
			assert.Equal(t, 1, code,
				"review-slice without required flags should exit 1")
			assert.NotEmpty(t, stderr,
				"stderr should contain an error about missing flags")
		})
	}
}

// TestExitCode_SliceMissingPath verifies that slice without the required
// --path flag returns exit code 1.
func TestExitCode_SliceMissingPath(t *testing.T) {
	t.Parallel()

	_, stderr, code := runHarvx(t, []string{"slice", "--stdout"})

	assert.Equal(t, 1, code, "slice without --path should exit 1")
	assert.NotEmpty(t, stderr, "stderr should contain error about missing --path flag")
}

// truncateForTest returns the first n characters of s, for test output readability.
func truncateForTest(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
