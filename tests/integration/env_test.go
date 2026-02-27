//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnv_HarvxMaxTokens verifies that HARVX_MAX_TOKENS limits the brief output
// token budget. When set to a low value, the brief JSON should reflect that limit.
func TestEnv_HarvxMaxTokens(t *testing.T) {
	t.Parallel()

	stdout, _, code := runHarvx(t, []string{"brief", "--json"}, "HARVX_MAX_TOKENS=5000")

	assert.Equal(t, 0, code, "brief --json with HARVX_MAX_TOKENS should exit 0")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "should produce valid JSON")

	// The max_tokens in the output should reflect the env var limit.
	maxTokens, ok := result["max_tokens"].(float64)
	assert.True(t, ok, "max_tokens should be a number")
	// The brief resolves max tokens from profile first, then env/flag.
	// With HARVX_MAX_TOKENS=5000 and the brief logic (checks if <= 32000),
	// the budget should be at most 5000.
	assert.LessOrEqual(t, maxTokens, float64(5000),
		"max_tokens should be limited by HARVX_MAX_TOKENS env var")
}

// TestEnv_HarvxStdout verifies that HARVX_STDOUT=true sends output to stdout
// without needing the --stdout flag.
func TestEnv_HarvxStdout(t *testing.T) {
	t.Parallel()

	stdout, stderr, code := runHarvx(t, []string{"brief"}, "HARVX_STDOUT=true")

	assert.Equal(t, 0, code, "brief with HARVX_STDOUT=true should exit 0")
	assert.NotEmpty(t, stdout, "output should go to stdout when HARVX_STDOUT=true")
	assert.NotContains(t, stderr, "Brief written to",
		"should not report file write when HARVX_STDOUT=true")
}

// TestEnv_HarvxVerbose verifies that HARVX_VERBOSE=1 produces debug-level
// output on stderr.
func TestEnv_HarvxVerbose(t *testing.T) {
	t.Parallel()

	_, stderr, code := runHarvx(t, []string{"brief", "--stdout"}, "HARVX_VERBOSE=1")

	assert.Equal(t, 0, code, "brief with HARVX_VERBOSE=1 should exit 0")
	// Verbose mode enables debug logging. The stderr should contain debug
	// level messages or at least logging output.
	assert.NotEmpty(t, stderr, "stderr should contain logging output in verbose mode")
	// Debug-level messages typically contain "level=DEBUG" in text format.
	hasDebugIndicator := strings.Contains(stderr, "level=DEBUG") ||
		strings.Contains(stderr, "DEBUG")
	assert.True(t, hasDebugIndicator,
		"verbose mode should produce DEBUG level output on stderr, got: %s",
		truncate(stderr, 500))
}

// TestEnv_HarvxQuiet verifies that HARVX_QUIET=1 suppresses non-essential
// output on stderr.
func TestEnv_HarvxQuiet(t *testing.T) {
	t.Parallel()

	_, stderrQuiet, codeQuiet := runHarvx(t, []string{"brief", "--stdout"}, "HARVX_QUIET=1")
	_, stderrNormal, _ := runHarvx(t, []string{"brief", "--stdout"})

	assert.Equal(t, 0, codeQuiet, "brief with HARVX_QUIET=1 should exit 0")
	// Quiet mode should produce less stderr output than normal mode, or
	// at least should not contain INFO level messages.
	assert.LessOrEqual(t, len(stderrQuiet), len(stderrNormal),
		"quiet mode should produce equal or less stderr output than normal mode")
}

// TestEnv_HarvxLogFormatJSON verifies that HARVX_LOG_FORMAT=json produces JSON
// formatted log entries on stderr instead of text format.
func TestEnv_HarvxLogFormatJSON(t *testing.T) {
	t.Parallel()

	// Use verbose mode to ensure there are log messages to check the format of.
	_, stderr, code := runHarvx(t, []string{"brief", "--stdout"},
		"HARVX_VERBOSE=1", "HARVX_LOG_FORMAT=json")

	assert.Equal(t, 0, code, "brief with HARVX_LOG_FORMAT=json should exit 0")

	if stderr == "" {
		t.Skip("no log output produced, cannot verify format")
	}

	// JSON log format should produce lines that are valid JSON objects.
	// Check that at least one line in stderr is valid JSON.
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	foundJSON := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if json.Unmarshal([]byte(line), &obj) == nil {
			foundJSON = true
			// JSON log entries should have "level" and "msg" fields.
			assert.Contains(t, obj, "level", "JSON log entry should have a level field")
			assert.Contains(t, obj, "msg", "JSON log entry should have a msg field")
			break
		}
	}
	assert.True(t, foundJSON,
		"HARVX_LOG_FORMAT=json should produce at least one JSON log line on stderr")
}

// TestEnv_CLIFlagPrecedence verifies that CLI flags override environment variables.
// Specifically, --max-tokens on the CLI should override HARVX_MAX_TOKENS.
func TestEnv_CLIFlagPrecedence(t *testing.T) {
	t.Parallel()

	// Set HARVX_MAX_TOKENS=1000 via env, but override with --max-tokens=5000 on CLI.
	stdout, _, code := runHarvx(t, []string{
		"brief", "--json", "--max-tokens", "5000",
	}, "HARVX_MAX_TOKENS=1000")

	assert.Equal(t, 0, code, "brief --json should exit 0")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "should produce valid JSON")

	// The CLI flag (5000) should take precedence over the env var (1000).
	maxTokens, ok := result["max_tokens"].(float64)
	assert.True(t, ok, "max_tokens should be a number")
	assert.Equal(t, float64(5000), maxTokens,
		"CLI --max-tokens should override HARVX_MAX_TOKENS env var")
}

// truncate returns the first n characters of s, or s if shorter.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
