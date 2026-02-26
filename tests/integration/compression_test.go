//go:build integration

package integration

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompression_GoRepo_ReducesTokens verifies that --compress does not
// increase total tokens compared to a run without compression on a Go repo.
// Compression should either reduce tokens or leave them unchanged (for files
// that are already concise).
func TestCompression_GoRepo_ReducesTokens(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")

	// Run without compression.
	stdoutNormal, _, codeNormal := runHarvxInDir(t, repo.Dir, []string{
		"preview", "--json",
	})
	require.Equal(t, 0, codeNormal, "preview --json should exit 0")

	var resultNormal map[string]interface{}
	err := json.Unmarshal([]byte(stdoutNormal), &resultNormal)
	require.NoError(t, err, "preview --json should produce valid JSON without compression")

	tokensNormal, ok := resultNormal["total_tokens"].(float64)
	require.True(t, ok, "total_tokens should be a number")

	// Run with compression.
	stdoutCompress, _, codeCompress := runHarvxInDir(t, repo.Dir, []string{
		"preview", "--json", "--compress",
	})
	require.Equal(t, 0, codeCompress, "preview --json --compress should exit 0")

	var resultCompress map[string]interface{}
	err = json.Unmarshal([]byte(stdoutCompress), &resultCompress)
	require.NoError(t, err, "preview --json should produce valid JSON with compression")

	tokensCompress, ok := resultCompress["total_tokens"].(float64)
	require.True(t, ok, "total_tokens should be a number with compression")

	// Compressed tokens should be less than or equal to uncompressed tokens.
	assert.LessOrEqual(t, tokensCompress, tokensNormal,
		"compressed total_tokens (%v) should be <= uncompressed total_tokens (%v)",
		tokensCompress, tokensNormal)
}

// TestCompression_TSRepo_ReducesTokens verifies that --compress does not
// increase total tokens compared to a run without compression on a TypeScript repo.
func TestCompression_TSRepo_ReducesTokens(t *testing.T) {
	t.Parallel()

	repo := repoByName("ts-nextjs")

	// Run without compression.
	stdoutNormal, _, codeNormal := runHarvxInDir(t, repo.Dir, []string{
		"preview", "--json",
	})
	require.Equal(t, 0, codeNormal, "preview --json should exit 0")

	var resultNormal map[string]interface{}
	err := json.Unmarshal([]byte(stdoutNormal), &resultNormal)
	require.NoError(t, err, "preview --json should produce valid JSON without compression")

	tokensNormal, ok := resultNormal["total_tokens"].(float64)
	require.True(t, ok, "total_tokens should be a number")

	// Run with compression.
	stdoutCompress, _, codeCompress := runHarvxInDir(t, repo.Dir, []string{
		"preview", "--json", "--compress",
	})
	require.Equal(t, 0, codeCompress, "preview --json --compress should exit 0")

	var resultCompress map[string]interface{}
	err = json.Unmarshal([]byte(stdoutCompress), &resultCompress)
	require.NoError(t, err, "preview --json should produce valid JSON with compression")

	tokensCompress, ok := resultCompress["total_tokens"].(float64)
	require.True(t, ok, "total_tokens should be a number with compression")

	// Compressed tokens should be less than or equal to uncompressed tokens.
	assert.LessOrEqual(t, tokensCompress, tokensNormal,
		"compressed total_tokens (%v) should be <= uncompressed total_tokens (%v)",
		tokensCompress, tokensNormal)
}

// TestCompression_AllRepos_ExitZero verifies that running brief with
// --compress exits 0 and produces non-empty output across all OSS test repos.
func TestCompression_AllRepos_ExitZero(t *testing.T) {
	t.Parallel()

	for _, repo := range testRepos() {
		t.Run(repo.Name, func(t *testing.T) {
			t.Parallel()

			stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
				"brief", "--stdout", "--compress",
			})

			assert.Equal(t, 0, code,
				"%s: brief --stdout --compress should exit 0", repo.Name)
			assert.NotEmpty(t, stdout,
				"%s: brief --stdout --compress should produce non-empty output", repo.Name)
		})
	}
}

// TestCompression_CompressEngine_Regex verifies that --compress-engine regex
// exits 0 and produces non-empty output.
func TestCompression_CompressEngine_Regex(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")
	stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--stdout", "--compress", "--compress-engine", "regex",
	})

	assert.Equal(t, 0, code,
		"brief --stdout --compress --compress-engine regex should exit 0")
	assert.NotEmpty(t, stdout,
		"brief with regex compression engine should produce non-empty output")
}

// TestCompression_CompressEngine_AST verifies that --compress-engine ast
// exits 0 and produces non-empty output.
func TestCompression_CompressEngine_AST(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")
	stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--stdout", "--compress", "--compress-engine", "ast",
	})

	assert.Equal(t, 0, code,
		"brief --stdout --compress --compress-engine ast should exit 0")
	assert.NotEmpty(t, stdout,
		"brief with ast compression engine should produce non-empty output")
}
