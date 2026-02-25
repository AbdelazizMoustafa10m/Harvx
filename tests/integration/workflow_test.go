//go:build integration

package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflow_BriefGeneration verifies that the brief command produces output
// containing README content from the sample repo.
func TestWorkflow_BriefGeneration(t *testing.T) {
	t.Parallel()

	stdout, _, code := runHarvx(t, []string{"brief", "--stdout"})

	assert.Equal(t, 0, code, "brief --stdout should exit 0")
	assert.NotEmpty(t, stdout, "brief should produce output")
	assert.Contains(t, stdout, "Sample Repo", "brief should include README content")
}

// TestWorkflow_BriefToStdout verifies that --stdout sends all output to stdout
// (nothing should be written to a file).
func TestWorkflow_BriefToStdout(t *testing.T) {
	t.Parallel()

	stdout, stderr, code := runHarvx(t, []string{"brief", "--stdout"})

	assert.Equal(t, 0, code, "brief --stdout should exit 0")
	assert.NotEmpty(t, stdout, "brief content should go to stdout")
	// When --stdout is set, the status message should NOT appear on stderr
	// because the content goes directly to stdout. The stderr should not
	// contain the "Brief written to" file output message.
	assert.NotContains(t, stderr, "Brief written to",
		"with --stdout, should not report file write on stderr")
}

// TestWorkflow_BriefJSON verifies that --json produces valid JSON with the
// expected schema fields from BriefJSON.
func TestWorkflow_BriefJSON(t *testing.T) {
	t.Parallel()

	stdout, _, code := runHarvx(t, []string{"brief", "--json"})

	assert.Equal(t, 0, code, "brief --json should exit 0")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "brief --json should produce valid JSON")

	// Verify expected BriefJSON schema fields.
	assert.Contains(t, result, "token_count", "JSON should contain token_count")
	assert.Contains(t, result, "content_hash", "JSON should contain content_hash")
	assert.Contains(t, result, "files_included", "JSON should contain files_included")
	assert.Contains(t, result, "sections", "JSON should contain sections")
	assert.Contains(t, result, "max_tokens", "JSON should contain max_tokens")

	// token_count should be a positive number.
	tokenCount, ok := result["token_count"].(float64)
	assert.True(t, ok, "token_count should be a number")
	assert.Greater(t, tokenCount, float64(0), "token_count should be positive")

	// content_hash should be a non-empty hex string.
	contentHash, ok := result["content_hash"].(string)
	assert.True(t, ok, "content_hash should be a string")
	assert.NotEmpty(t, contentHash, "content_hash should not be empty")
}

// TestWorkflow_BriefClaudeXML verifies that --target claude produces XML-formatted
// output suitable for Claude's XML preference.
func TestWorkflow_BriefClaudeXML(t *testing.T) {
	t.Parallel()

	stdout, _, code := runHarvx(t, []string{"brief", "--target", "claude", "--stdout"})

	assert.Equal(t, 0, code, "brief --target claude should exit 0")
	assert.NotEmpty(t, stdout, "XML output should not be empty")

	// Claude XML output should contain XML tags (angle brackets).
	assert.Contains(t, stdout, "<", "XML output should contain opening tags")
	assert.Contains(t, stdout, ">", "XML output should contain closing tags")
}

// TestWorkflow_SliceModule verifies that slice --path src/auth works in a git repo
// and produces output containing the auth module files.
func TestWorkflow_SliceModule(t *testing.T) {
	t.Parallel()

	dir := setupGitRepo(t)
	stdout, _, code := runHarvxInDir(t, dir, []string{"slice", "--path", "src/auth", "--stdout"})

	assert.Equal(t, 0, code, "slice should exit 0")
	assert.NotEmpty(t, stdout, "slice should produce output")
	// The slice should include middleware.go content from the auth module.
	assert.Contains(t, stdout, "CheckAuth", "slice should include auth module content")
}

// TestWorkflow_ReviewSlice verifies that review-slice with valid git refs produces
// output containing the changed file content.
func TestWorkflow_ReviewSlice(t *testing.T) {
	t.Parallel()

	dir, baseRef := setupGitRepoWithChange(t)
	stdout, _, code := runHarvxInDir(t, dir, []string{
		"review-slice",
		"--base", baseRef,
		"--head", "HEAD",
		"--stdout",
	})

	assert.Equal(t, 0, code, "review-slice should exit 0")
	assert.NotEmpty(t, stdout, "review-slice should produce output")
	// The changed file was middleware.go ("checking auth" -> "checking auth v2").
	assert.Contains(t, stdout, "middleware", "review-slice should reference the changed file")
}

// TestWorkflow_ReviewSliceJSON verifies that review-slice --json has the expected
// ReviewSliceJSON fields.
func TestWorkflow_ReviewSliceJSON(t *testing.T) {
	t.Parallel()

	dir, baseRef := setupGitRepoWithChange(t)
	stdout, _, code := runHarvxInDir(t, dir, []string{
		"review-slice",
		"--base", baseRef,
		"--head", "HEAD",
		"--json",
	})

	assert.Equal(t, 0, code, "review-slice --json should exit 0")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "review-slice --json should produce valid JSON")

	// Verify expected ReviewSliceJSON schema fields.
	assert.Contains(t, result, "token_count", "JSON should contain token_count")
	assert.Contains(t, result, "content_hash", "JSON should contain content_hash")
	assert.Contains(t, result, "changed_files", "JSON should contain changed_files")
	assert.Contains(t, result, "neighbor_files", "JSON should contain neighbor_files")
	assert.Contains(t, result, "deleted_files", "JSON should contain deleted_files")
	assert.Contains(t, result, "total_files", "JSON should contain total_files")
	assert.Contains(t, result, "max_tokens", "JSON should contain max_tokens")
	assert.Contains(t, result, "base_ref", "JSON should contain base_ref")
	assert.Contains(t, result, "head_ref", "JSON should contain head_ref")

	// changed_files should contain the modified file.
	changedFiles, ok := result["changed_files"].([]interface{})
	assert.True(t, ok, "changed_files should be an array")
	assert.NotEmpty(t, changedFiles, "changed_files should not be empty")
}

// TestWorkflow_Workspace verifies that the workspace command reads
// .harvx/workspace.toml and produces output.
func TestWorkflow_Workspace(t *testing.T) {
	t.Parallel()

	stdout, _, code := runHarvx(t, []string{"workspace", "--stdout"})

	assert.Equal(t, 0, code, "workspace --stdout should exit 0")
	assert.NotEmpty(t, stdout, "workspace should produce output")
	assert.Contains(t, stdout, "sample-workspace",
		"workspace output should reference the workspace name from workspace.toml")
}

// TestWorkflow_WorkspaceJSON verifies that workspace --json produces valid JSON
// with the expected WorkspaceJSON fields.
func TestWorkflow_WorkspaceJSON(t *testing.T) {
	t.Parallel()

	stdout, _, code := runHarvx(t, []string{"workspace", "--json"})

	assert.Equal(t, 0, code, "workspace --json should exit 0")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "workspace --json should produce valid JSON")

	// Verify expected WorkspaceJSON schema fields.
	assert.Contains(t, result, "name", "JSON should contain name")
	assert.Contains(t, result, "description", "JSON should contain description")
	assert.Contains(t, result, "repo_count", "JSON should contain repo_count")
	assert.Contains(t, result, "token_count", "JSON should contain token_count")
	assert.Contains(t, result, "content_hash", "JSON should contain content_hash")

	// The workspace name should match the workspace.toml config.
	name, ok := result["name"].(string)
	assert.True(t, ok, "name should be a string")
	assert.Equal(t, "sample-workspace", name, "workspace name should match config")

	// repo_count should be at least 1.
	repoCount, ok := result["repo_count"].(float64)
	assert.True(t, ok, "repo_count should be a number")
	assert.GreaterOrEqual(t, repoCount, float64(1), "should have at least 1 repo")
}

// TestWorkflow_FullPipeline verifies the brief -> review-slice -> verify chain
// using a git repo with changes.
func TestWorkflow_FullPipeline(t *testing.T) {
	t.Parallel()

	dir, baseRef := setupGitRepoWithChange(t)

	// Step 1: Generate a brief.
	briefOut, _, briefCode := runHarvxInDir(t, dir, []string{"brief", "--stdout"})
	assert.Equal(t, 0, briefCode, "brief should exit 0")
	assert.NotEmpty(t, briefOut, "brief should produce output")

	// Step 2: Generate a review-slice for the change.
	reviewOut, _, reviewCode := runHarvxInDir(t, dir, []string{
		"review-slice",
		"--base", baseRef,
		"--head", "HEAD",
		"--stdout",
	})
	assert.Equal(t, 0, reviewCode, "review-slice should exit 0")
	assert.NotEmpty(t, reviewOut, "review-slice should produce output")
	assert.Contains(t, reviewOut, "middleware",
		"review-slice should reference the changed middleware file")

	// Step 3: Generate brief to file, then verify it.
	_, briefFileStderr, briefFileCode := runHarvxInDir(t, dir, []string{"brief"})
	assert.Equal(t, 0, briefFileCode, "brief to file should exit 0")
	assert.Contains(t, briefFileStderr, "Brief written to",
		"brief should report the output file path on stderr")

	// Step 4: Run verify on the generated brief output.
	// The verify command looks for the default output file (harvx-brief.md).
	verifyOut, _, verifyCode := runHarvxInDir(t, dir, []string{
		"verify",
		"-o", "harvx-brief.md",
	})
	// Verify may return 0 (all pass) or 2 (partial, with warnings).
	assert.Contains(t, []int{0, 2}, verifyCode,
		"verify should exit 0 or 2, not 1")
	assert.NotEmpty(t, verifyOut, "verify should produce a report")
}

// TestWorkflow_PreviewJSON verifies that preview --json returns valid JSON
// with the expected PreviewResult fields.
func TestWorkflow_PreviewJSON(t *testing.T) {
	t.Parallel()

	stdout, _, code := runHarvx(t, []string{"preview", "--json"})

	assert.Equal(t, 0, code, "preview --json should exit 0")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "preview --json should produce valid JSON")

	// Verify expected PreviewResult schema fields.
	assert.Contains(t, result, "total_files", "JSON should contain total_files")
	assert.Contains(t, result, "total_tokens", "JSON should contain total_tokens")
	assert.Contains(t, result, "tokenizer", "JSON should contain tokenizer")
}

// TestWorkflow_VersionJSON verifies that version --json returns valid JSON
// with the expected version fields.
func TestWorkflow_VersionJSON(t *testing.T) {
	t.Parallel()

	stdout, _, code := runHarvx(t, []string{"version", "--json"})

	assert.Equal(t, 0, code, "version --json should exit 0")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "version --json should produce valid JSON")

	// Verify expected versionInfo schema fields.
	assert.Contains(t, result, "version", "JSON should contain version")
	assert.Contains(t, result, "commit", "JSON should contain commit")
	assert.Contains(t, result, "goVersion", "JSON should contain goVersion")
	assert.Contains(t, result, "os", "JSON should contain os")
	assert.Contains(t, result, "arch", "JSON should contain arch")

	// version should be a non-empty string.
	version, ok := result["version"].(string)
	assert.True(t, ok, "version should be a string")
	assert.NotEmpty(t, version, "version should not be empty")

	// os should contain the current platform.
	osVal, ok := result["os"].(string)
	assert.True(t, ok, "os should be a string")
	assert.True(t, strings.Contains(osVal, "darwin") || strings.Contains(osVal, "linux") || strings.Contains(osVal, "windows"),
		"os should be a recognized platform, got: %s", osVal)
}
