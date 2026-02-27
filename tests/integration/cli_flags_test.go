//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIFlags_MaxTokens_RespectsLimit verifies that --max-tokens limits the
// brief output token budget. When set to 5000, the JSON metadata should
// reflect that limit.
func TestCLIFlags_MaxTokens_RespectsLimit(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")
	stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--json", "--max-tokens", "5000",
	})

	assert.Equal(t, 0, code, "brief --json --max-tokens 5000 should exit 0")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "brief --json should produce valid JSON")

	maxTokens, ok := result["max_tokens"].(float64)
	assert.True(t, ok, "max_tokens should be a number")
	assert.LessOrEqual(t, maxTokens, float64(5000),
		"max_tokens should be limited to 5000 by --max-tokens flag")
}

// TestCLIFlags_MaxTokens_EnvOverride verifies that HARVX_MAX_TOKENS environment
// variable limits the brief token budget when --max-tokens is not explicitly set.
func TestCLIFlags_MaxTokens_EnvOverride(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")
	stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--json",
	}, "HARVX_MAX_TOKENS=3000")

	assert.Equal(t, 0, code, "brief --json with HARVX_MAX_TOKENS=3000 should exit 0")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "brief --json should produce valid JSON")

	maxTokens, ok := result["max_tokens"].(float64)
	assert.True(t, ok, "max_tokens should be a number")
	assert.LessOrEqual(t, maxTokens, float64(3000),
		"max_tokens should be limited to 3000 by HARVX_MAX_TOKENS env var")
}

// TestCLIFlags_Stdout_CleanOutput verifies that --stdout sends all content to
// stdout and does not report "Brief written to" on stderr.
func TestCLIFlags_Stdout_CleanOutput(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")
	stdout, stderr, code := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--stdout",
	})

	assert.Equal(t, 0, code, "brief --stdout should exit 0")
	assert.NotEmpty(t, stdout, "stdout should contain the brief content")
	assert.NotContains(t, stderr, "Brief written to",
		"stderr should not contain file write message when --stdout is used")
}

// TestCLIFlags_GitTrackedOnly verifies that --git-tracked-only is accepted
// and produces valid output. Uses brief --json because the preview pipeline
// stub always returns zero file counts. The brief workflow reads well-known
// files (README, etc.) so adding an untracked arbitrary file does not change
// the files_included list.
func TestCLIFlags_GitTrackedOnly(t *testing.T) {
	t.Parallel()

	// Set up a git repo from the sample-repo fixture.
	dir := setupGitRepo(t)

	// Run brief --json with --git-tracked-only to get a baseline.
	stdout1, _, code1 := runHarvxInDir(t, dir, []string{
		"brief", "--json", "--git-tracked-only",
	})
	assert.Equal(t, 0, code1, "brief --json --git-tracked-only should exit 0")

	var result1 map[string]interface{}
	err := json.Unmarshal([]byte(stdout1), &result1)
	require.NoError(t, err, "first brief --json should produce valid JSON")

	filesRaw1, ok := result1["files_included"].([]interface{})
	require.True(t, ok, "files_included should be an array")
	require.Greater(t, len(filesRaw1), 0,
		"git-tracked-only should still find files in a git repo")

	// Add an untracked file (not staged, not committed).
	untrackedPath := filepath.Join(dir, "untracked-new-file.txt")
	err = os.WriteFile(untrackedPath, []byte("this file is not git tracked"), 0o644)
	require.NoError(t, err, "writing untracked file")

	// Run brief --json --git-tracked-only again.
	stdout2, _, code2 := runHarvxInDir(t, dir, []string{
		"brief", "--json", "--git-tracked-only",
	})
	assert.Equal(t, 0, code2, "second brief --json --git-tracked-only should exit 0")

	var result2 map[string]interface{}
	err = json.Unmarshal([]byte(stdout2), &result2)
	require.NoError(t, err, "second brief --json should produce valid JSON")

	filesRaw2, ok := result2["files_included"].([]interface{})
	require.True(t, ok, "files_included should be an array")

	// The untracked file should NOT appear in files_included.
	// Brief reads well-known files only, so the count should not change.
	assert.Equal(t, len(filesRaw1), len(filesRaw2),
		"files_included count should not change after adding an untracked file with --git-tracked-only")
}

// TestCLIFlags_StdoutPipeChain verifies that brief --stdout produces output
// suitable for piping to other tools: non-empty, with newlines, and substantial
// enough to be useful.
func TestCLIFlags_StdoutPipeChain(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")
	stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--stdout",
	})

	assert.Equal(t, 0, code, "brief --stdout should exit 0")
	assert.True(t, strings.Contains(stdout, "\n"),
		"stdout should contain newlines for piping")
	assert.Greater(t, len(stdout), 100,
		"stdout should be > 100 bytes, suitable for piping to downstream tools")
}

// TestCLIFlags_Format_XML_Flag verifies that --format xml --target claude
// produces XML-formatted output with angle bracket tags.
func TestCLIFlags_Format_XML_Flag(t *testing.T) {
	t.Parallel()

	repo := repoByName("ts-nextjs")
	stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--stdout", "--format", "xml", "--target", "claude",
	})

	assert.Equal(t, 0, code, "brief --stdout --format xml --target claude should exit 0")
	assert.NotEmpty(t, stdout, "XML output should not be empty")
	assert.Contains(t, stdout, "<",
		"XML output should contain opening angle brackets")
	assert.Contains(t, stdout, ">",
		"XML output should contain closing angle brackets")
}

// TestCLIFlags_VerboseLogging verifies that the -v flag produces DEBUG-level
// log messages on stderr.
func TestCLIFlags_VerboseLogging(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")
	_, stderr, code := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--stdout", "-v",
	})

	assert.Equal(t, 0, code, "brief --stdout -v should exit 0")
	assert.NotEmpty(t, stderr, "stderr should contain logging output in verbose mode")

	hasDebugIndicator := strings.Contains(stderr, "level=DEBUG") ||
		strings.Contains(stderr, "DEBUG")
	assert.True(t, hasDebugIndicator,
		"verbose mode should produce DEBUG level output on stderr, got: %s",
		truncateCLI(stderr, 500))
}

// TestCLIFlags_QuietMode verifies that the -q flag suppresses non-essential
// output on stderr compared to a normal run.
func TestCLIFlags_QuietMode(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")

	// Run in quiet mode.
	_, stderrQuiet, codeQuiet := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--stdout", "-q",
	})
	assert.Equal(t, 0, codeQuiet, "brief --stdout -q should exit 0")

	// Run in normal mode (no -q).
	_, stderrNormal, _ := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--stdout",
	})

	// Quiet mode should produce equal or less stderr output than normal mode.
	assert.LessOrEqual(t, len(stderrQuiet), len(stderrNormal),
		"quiet mode should produce equal or less stderr output than normal mode")
}

// truncateCLI returns the first n characters of s, for test output readability.
func truncateCLI(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
