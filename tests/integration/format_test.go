//go:build integration

package integration

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormat_MarkdownOutput_HasExpectedSections verifies that brief --format markdown
// produces output with standard markdown structural elements: headings, code fences,
// and sufficient content across multiple lines.
func TestFormat_MarkdownOutput_HasExpectedSections(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")
	stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--stdout", "--format", "markdown",
	})

	require.Equal(t, 0, code, "brief --format markdown should exit 0")
	require.NotEmpty(t, stdout, "markdown output should not be empty")

	lines := strings.Split(strings.TrimSpace(stdout), "\n")

	// Verify the output has multiple lines (> 10).
	assert.Greater(t, len(lines), 10,
		"markdown output should have more than 10 lines, got %d", len(lines))

	// Verify at least one markdown heading (line starting with #).
	hasHeading := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			hasHeading = true
			break
		}
	}
	assert.True(t, hasHeading,
		"markdown output should contain at least one heading (line starting with #)")

	// Verify at least one code fence.
	assert.Contains(t, stdout, "```",
		"markdown output should contain at least one code fence")
}

// TestFormat_XMLOutput_WellFormed verifies that brief --target claude produces
// well-formed XML output that can be parsed without syntax errors.
func TestFormat_XMLOutput_WellFormed(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")
	stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
		"brief", "--stdout", "--target", "claude",
	})

	require.Equal(t, 0, code, "brief --target claude should exit 0")
	require.NotEmpty(t, stdout, "XML output should not be empty")

	// Verify output contains XML tags.
	assert.Contains(t, stdout, "<", "XML output should contain opening angle brackets")
	assert.Contains(t, stdout, ">", "XML output should contain closing angle brackets")

	// Verify the output is parseable by encoding/xml.
	// Iterate through all tokens; any xml.SyntaxError means malformed XML.
	decoder := xml.NewDecoder(strings.NewReader(stdout))
	for {
		_, err := decoder.Token()
		if err != nil {
			// EOF is the expected termination.
			if err.Error() == "EOF" {
				break
			}
			// Check if it is an XML syntax error (malformed).
			if _, ok := err.(*xml.SyntaxError); ok {
				t.Fatalf("XML output is malformed: %v", err)
			}
			// Other errors (e.g., io.EOF wrapped) are acceptable.
			break
		}
	}
}

// TestFormat_XMLOutput_AllRepos verifies that brief --target claude produces valid
// XML-like output across all test repositories.
func TestFormat_XMLOutput_AllRepos(t *testing.T) {
	t.Parallel()

	for _, repo := range testRepos() {
		t.Run(repo.Name, func(t *testing.T) {
			t.Parallel()

			stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
				"brief", "--stdout", "--target", "claude",
			})

			assert.Equal(t, 0, code,
				"brief --target claude should exit 0 for repo %s", repo.Name)
			assert.NotEmpty(t, stdout,
				"XML output should not be empty for repo %s", repo.Name)
			assert.Contains(t, stdout, "<",
				"XML output should contain opening angle brackets for repo %s", repo.Name)
			assert.Contains(t, stdout, ">",
				"XML output should contain closing angle brackets for repo %s", repo.Name)
		})
	}
}

// TestFormat_MarkdownOutput_AllRepos verifies that brief --format markdown produces
// output with markdown indicators across all test repositories.
func TestFormat_MarkdownOutput_AllRepos(t *testing.T) {
	t.Parallel()

	for _, repo := range testRepos() {
		t.Run(repo.Name, func(t *testing.T) {
			t.Parallel()

			stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
				"brief", "--stdout", "--format", "markdown",
			})

			assert.Equal(t, 0, code,
				"brief --format markdown should exit 0 for repo %s", repo.Name)
			assert.NotEmpty(t, stdout,
				"markdown output should not be empty for repo %s", repo.Name)

			// Markdown output should contain headings, code fences, or horizontal rules.
			hasMarkdownIndicators := strings.Contains(stdout, "#") ||
				strings.Contains(stdout, "```") ||
				strings.Contains(stdout, "---")
			assert.True(t, hasMarkdownIndicators,
				"markdown output should contain markdown formatting indicators for repo %s", repo.Name)
		})
	}
}

// TestFormat_JSONMetadata_AllRepos verifies that preview --json produces valid JSON
// with expected metadata fields across all test repositories.
func TestFormat_JSONMetadata_AllRepos(t *testing.T) {
	t.Parallel()

	for _, repo := range testRepos() {
		t.Run(repo.Name, func(t *testing.T) {
			t.Parallel()

			stdout, _, code := runHarvxInDir(t, repo.Dir, []string{
				"preview", "--json",
			})

			require.Equal(t, 0, code,
				"preview --json should exit 0 for repo %s", repo.Name)

			var result map[string]interface{}
			err := json.Unmarshal([]byte(stdout), &result)
			require.NoError(t, err,
				"preview --json should produce valid JSON for repo %s", repo.Name)

			assert.Contains(t, result, "total_files",
				"JSON should contain total_files for repo %s", repo.Name)
			assert.Contains(t, result, "total_tokens",
				"JSON should contain total_tokens for repo %s", repo.Name)
		})
	}
}

// TestFormat_Determinism_AllRepos verifies that brief --stdout produces
// byte-identical output across two consecutive runs for every test repository.
func TestFormat_Determinism_AllRepos(t *testing.T) {
	t.Parallel()

	for _, repo := range testRepos() {
		t.Run(repo.Name, func(t *testing.T) {
			t.Parallel()

			// Run 1.
			stdout1, _, code1 := runHarvxInDir(t, repo.Dir, []string{
				"brief", "--stdout",
			})
			require.Equal(t, 0, code1,
				"first brief --stdout run should exit 0 for repo %s", repo.Name)
			require.NotEmpty(t, stdout1,
				"first run should produce output for repo %s", repo.Name)

			// Run 2.
			stdout2, _, code2 := runHarvxInDir(t, repo.Dir, []string{
				"brief", "--stdout",
			})
			require.Equal(t, 0, code2,
				"second brief --stdout run should exit 0 for repo %s", repo.Name)
			require.NotEmpty(t, stdout2,
				"second run should produce output for repo %s", repo.Name)

			// Compare SHA-256 digests.
			digest1 := sha256sum(stdout1)
			digest2 := sha256sum(stdout2)
			assert.Equal(t, digest1, digest2,
				"brief --stdout should produce byte-identical output across two runs for repo %s", repo.Name)
		})
	}
}
