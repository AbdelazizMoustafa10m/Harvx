//go:build integration

package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeterminism_BriefContentHash verifies that running brief twice produces
// identical content hashes (as reported by the --json output).
func TestDeterminism_BriefContentHash(t *testing.T) {
	t.Parallel()

	// Run 1: Get the content hash from JSON output.
	stdout1, _, code1 := runHarvx(t, []string{"brief", "--json"})
	require.Equal(t, 0, code1, "first brief --json run should exit 0")

	var result1 map[string]interface{}
	err := json.Unmarshal([]byte(stdout1), &result1)
	require.NoError(t, err, "first run should produce valid JSON")

	hash1, ok := result1["content_hash"].(string)
	require.True(t, ok, "content_hash should be a string")
	require.NotEmpty(t, hash1, "content_hash should not be empty")

	// Run 2: Get the content hash again.
	stdout2, _, code2 := runHarvx(t, []string{"brief", "--json"})
	require.Equal(t, 0, code2, "second brief --json run should exit 0")

	var result2 map[string]interface{}
	err = json.Unmarshal([]byte(stdout2), &result2)
	require.NoError(t, err, "second run should produce valid JSON")

	hash2, ok := result2["content_hash"].(string)
	require.True(t, ok, "content_hash should be a string")

	// Content hashes must be identical.
	assert.Equal(t, hash1, hash2,
		"content_hash should be identical across two runs of brief --json")
}

// TestDeterminism_BriefOutput verifies that running brief twice produces
// byte-identical output by comparing SHA-256 digests of the stdout content.
func TestDeterminism_BriefOutput(t *testing.T) {
	t.Parallel()

	// Run 1.
	stdout1, _, code1 := runHarvx(t, []string{"brief", "--stdout"})
	require.Equal(t, 0, code1, "first brief --stdout run should exit 0")
	require.NotEmpty(t, stdout1, "first run should produce output")

	// Run 2.
	stdout2, _, code2 := runHarvx(t, []string{"brief", "--stdout"})
	require.Equal(t, 0, code2, "second brief --stdout run should exit 0")
	require.NotEmpty(t, stdout2, "second run should produce output")

	// Compare SHA-256 digests.
	digest1 := sha256sum(stdout1)
	digest2 := sha256sum(stdout2)
	assert.Equal(t, digest1, digest2,
		"brief --stdout should produce byte-identical output across two runs")
}

// TestDeterminism_SliceOutput verifies that running slice twice in the same
// git repo produces identical output.
func TestDeterminism_SliceOutput(t *testing.T) {
	t.Parallel()

	dir := setupGitRepo(t)

	// Run 1.
	stdout1, _, code1 := runHarvxInDir(t, dir, []string{
		"slice", "--path", "src/auth", "--stdout",
	})
	require.Equal(t, 0, code1, "first slice run should exit 0")
	require.NotEmpty(t, stdout1, "first slice run should produce output")

	// Run 2.
	stdout2, _, code2 := runHarvxInDir(t, dir, []string{
		"slice", "--path", "src/auth", "--stdout",
	})
	require.Equal(t, 0, code2, "second slice run should exit 0")
	require.NotEmpty(t, stdout2, "second slice run should produce output")

	// Compare SHA-256 digests.
	digest1 := sha256sum(stdout1)
	digest2 := sha256sum(stdout2)
	assert.Equal(t, digest1, digest2,
		"slice --stdout should produce byte-identical output across two runs")
}

// TestDeterminism_PreviewJSON verifies that preview --json produces
// deterministic output across two runs.
func TestDeterminism_PreviewJSON(t *testing.T) {
	t.Parallel()

	// Run 1.
	stdout1, _, code1 := runHarvx(t, []string{"preview", "--json"})
	require.Equal(t, 0, code1, "first preview --json run should exit 0")

	var result1 map[string]interface{}
	err := json.Unmarshal([]byte(stdout1), &result1)
	require.NoError(t, err, "first run should produce valid JSON")

	// Run 2.
	stdout2, _, code2 := runHarvx(t, []string{"preview", "--json"})
	require.Equal(t, 0, code2, "second preview --json run should exit 0")

	var result2 map[string]interface{}
	err = json.Unmarshal([]byte(stdout2), &result2)
	require.NoError(t, err, "second run should produce valid JSON")

	// Compare the structural fields (excluding timing-sensitive ones).
	assert.Equal(t, result1["total_files"], result2["total_files"],
		"total_files should be deterministic")
	assert.Equal(t, result1["total_tokens"], result2["total_tokens"],
		"total_tokens should be deterministic")
	assert.Equal(t, result1["tokenizer"], result2["tokenizer"],
		"tokenizer should be deterministic")
}

// TestDeterminism_BriefPerformance verifies that brief completes in under
// 5 seconds on the sample repo.
func TestDeterminism_BriefPerformance(t *testing.T) {
	t.Parallel()

	start := time.Now()
	_, _, code := runHarvx(t, []string{"brief", "--stdout"})
	elapsed := time.Since(start)

	assert.Equal(t, 0, code, "brief should exit 0")
	assert.Less(t, elapsed, 5*time.Second,
		"brief should complete in under 5 seconds, took %s", elapsed)
}

// TestDeterminism_PreviewPerformance verifies that preview --json completes
// in under 2 seconds on the sample repo.
func TestDeterminism_PreviewPerformance(t *testing.T) {
	t.Parallel()

	start := time.Now()
	_, _, code := runHarvx(t, []string{"preview", "--json"})
	elapsed := time.Since(start)

	assert.Equal(t, 0, code, "preview --json should exit 0")
	assert.Less(t, elapsed, 2*time.Second,
		"preview --json should complete in under 2 seconds, took %s", elapsed)
}

// sha256sum computes the SHA-256 digest of a string and returns it as hex.
func sha256sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
