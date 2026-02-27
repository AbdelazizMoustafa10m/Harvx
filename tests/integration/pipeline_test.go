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

// TestPipeline_BriefToFile verifies that brief -o writes a file to disk
// and that the verify command can then read and verify that file.
func TestPipeline_BriefToFile(t *testing.T) {
	t.Parallel()

	dir := setupGitRepo(t)
	outFile := filepath.Join(dir, "test-brief.md")

	// Generate brief to a specific file.
	_, stderr, code := runHarvxInDir(t, dir, []string{"brief", "-o", outFile})
	assert.Equal(t, 0, code, "brief -o should exit 0")
	assert.Contains(t, stderr, "Brief written to",
		"brief should report the output path on stderr")

	// Verify the file was actually written.
	content, err := os.ReadFile(outFile)
	require.NoError(t, err, "brief output file should exist")
	assert.NotEmpty(t, content, "brief output file should not be empty")
	assert.Contains(t, string(content), "Sample Repo",
		"brief output should include README content")

	// Run verify against the generated output.
	verifyOut, _, verifyCode := runHarvxInDir(t, dir, []string{
		"verify",
		"-o", outFile,
	})
	// Verify may return 0 (all pass) or 2 (partial/warnings).
	assert.Contains(t, []int{0, 2}, verifyCode,
		"verify should exit 0 or 2")
	assert.NotEmpty(t, verifyOut, "verify should produce a report")
}

// TestPipeline_StdoutPiping verifies that brief --stdout produces non-empty
// output with newlines, suitable for piping to other tools.
func TestPipeline_StdoutPiping(t *testing.T) {
	t.Parallel()

	stdout, _, code := runHarvx(t, []string{"brief", "--stdout"})

	assert.Equal(t, 0, code, "brief --stdout should exit 0")
	assert.NotEmpty(t, stdout, "stdout output should not be empty")
	assert.True(t, strings.Contains(stdout, "\n"),
		"stdout output should contain newlines for proper piping")
	// Output should have substantial content (at least a few lines).
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	assert.Greater(t, len(lines), 1, "output should have multiple lines")
}

// TestPipeline_SliceAndReviewSlice verifies that both slice and review-slice
// produce output for the same repository.
func TestPipeline_SliceAndReviewSlice(t *testing.T) {
	t.Parallel()

	dir, baseRef := setupGitRepoWithChange(t)

	// Run slice for the auth module.
	sliceOut, _, sliceCode := runHarvxInDir(t, dir, []string{
		"slice", "--path", "src/auth", "--stdout",
	})
	assert.Equal(t, 0, sliceCode, "slice should exit 0")
	assert.NotEmpty(t, sliceOut, "slice should produce output")

	// Run review-slice for the same repo.
	reviewOut, _, reviewCode := runHarvxInDir(t, dir, []string{
		"review-slice",
		"--base", baseRef,
		"--head", "HEAD",
		"--stdout",
	})
	assert.Equal(t, 0, reviewCode, "review-slice should exit 0")
	assert.NotEmpty(t, reviewOut, "review-slice should produce output")

	// Both should reference auth-related content since the change was in
	// src/auth/middleware.go.
	assert.Contains(t, sliceOut, "CheckAuth",
		"slice should include auth module content")
	assert.Contains(t, reviewOut, "middleware",
		"review-slice should reference the changed file")
}

// TestPipeline_MultipleSlicePaths verifies that slice --path a --path b works
// and includes content from both paths.
func TestPipeline_MultipleSlicePaths(t *testing.T) {
	t.Parallel()

	dir := setupGitRepo(t)
	stdout, _, code := runHarvxInDir(t, dir, []string{
		"slice",
		"--path", "src/auth",
		"--path", "src",
		"--stdout",
	})

	assert.Equal(t, 0, code, "slice with multiple paths should exit 0")
	assert.NotEmpty(t, stdout, "slice with multiple paths should produce output")
	// Content from the auth module should be present.
	assert.Contains(t, stdout, "CheckAuth",
		"multi-path slice should include auth module content")
}

// TestPipeline_OutputFormatMarkdown verifies that --format markdown produces
// markdown-formatted output.
func TestPipeline_OutputFormatMarkdown(t *testing.T) {
	t.Parallel()

	stdout, _, code := runHarvx(t, []string{
		"brief", "--stdout", "--format", "markdown",
	})

	assert.Equal(t, 0, code, "brief --format markdown should exit 0")
	assert.NotEmpty(t, stdout, "markdown output should not be empty")
	// Markdown output should contain markdown-style headings or code fences.
	hasMarkdownIndicators := strings.Contains(stdout, "#") ||
		strings.Contains(stdout, "```") ||
		strings.Contains(stdout, "---")
	assert.True(t, hasMarkdownIndicators,
		"markdown output should contain markdown formatting indicators")
}

// TestPipeline_OutputFormatXML verifies that --target claude produces XML output
// with angle brackets.
func TestPipeline_OutputFormatXML(t *testing.T) {
	t.Parallel()

	stdout, _, code := runHarvx(t, []string{
		"brief", "--target", "claude", "--stdout",
	})

	assert.Equal(t, 0, code, "brief --target claude should exit 0")
	assert.NotEmpty(t, stdout, "XML output should not be empty")
	assert.Contains(t, stdout, "<", "XML output should contain opening angle brackets")
	assert.Contains(t, stdout, ">", "XML output should contain closing angle brackets")
}

// TestPipeline_SliceJSON verifies that slice --json produces valid JSON with the
// expected ModuleSliceJSON fields.
func TestPipeline_SliceJSON(t *testing.T) {
	t.Parallel()

	dir := setupGitRepo(t)
	stdout, _, code := runHarvxInDir(t, dir, []string{
		"slice", "--path", "src/auth", "--json",
	})

	assert.Equal(t, 0, code, "slice --json should exit 0")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "slice --json should produce valid JSON")

	// Verify expected ModuleSliceJSON schema fields.
	assert.Contains(t, result, "token_count", "JSON should contain token_count")
	assert.Contains(t, result, "content_hash", "JSON should contain content_hash")
	assert.Contains(t, result, "module_files", "JSON should contain module_files")
	assert.Contains(t, result, "neighbor_files", "JSON should contain neighbor_files")
	assert.Contains(t, result, "total_files", "JSON should contain total_files")
	assert.Contains(t, result, "max_tokens", "JSON should contain max_tokens")
	assert.Contains(t, result, "paths", "JSON should contain paths")

	// paths should contain "src/auth".
	paths, ok := result["paths"].([]interface{})
	assert.True(t, ok, "paths should be an array")
	require.NotEmpty(t, paths, "paths should not be empty")
	assert.Equal(t, "src/auth", paths[0], "first path should be src/auth")
}
