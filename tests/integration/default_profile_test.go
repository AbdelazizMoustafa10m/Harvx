//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultProfile_AllRepos verifies that the default profile produces valid
// Markdown output for each test repository type. This is the broadest smoke test
// confirming that Harvx can process Go, TypeScript, Python, and monorepo projects.
func TestDefaultProfile_AllRepos(t *testing.T) {
	t.Parallel()

	repos := testRepos()
	for _, repo := range repos {
		t.Run(repo.Name, func(t *testing.T) {
			t.Parallel()
			skipIfMissing(t, repo)

			stdout, _, exitCode := runHarvxInDir(t, repo.Dir, []string{"brief", "--stdout"})

			assert.Equal(t, 0, exitCode,
				"brief --stdout should exit 0 for %s repo", repo.Name)
			assert.NotEmpty(t, stdout,
				"brief --stdout should produce non-empty output for %s repo", repo.Name)

			// Output should contain markdown formatting indicators.
			hasMarkdown := strings.Contains(stdout, "#") ||
				strings.Contains(stdout, "```")
			assert.True(t, hasMarkdown,
				"output for %s should contain markdown indicators (# or ```)", repo.Name)

			// Output should have multiple lines (more than 5).
			lines := strings.Split(strings.TrimSpace(stdout), "\n")
			assert.Greater(t, len(lines), 5,
				"output for %s should have more than 5 lines, got %d", repo.Name, len(lines))
		})
	}
}

// TestDefaultProfile_BriefJSON_AllRepos verifies that brief --json produces
// valid JSON with expected metadata fields for each test repository type.
// Uses brief --json instead of preview --json because the preview pipeline
// is not yet wired to discovery services and always returns zero counts.
func TestDefaultProfile_BriefJSON_AllRepos(t *testing.T) {
	t.Parallel()

	repos := testRepos()
	for _, repo := range repos {
		t.Run(repo.Name, func(t *testing.T) {
			t.Parallel()
			skipIfMissing(t, repo)

			stdout, _, exitCode := runHarvxInDir(t, repo.Dir, []string{"brief", "--json"})

			assert.Equal(t, 0, exitCode,
				"brief --json should exit 0 for %s repo", repo.Name)

			var result map[string]any
			err := json.Unmarshal([]byte(stdout), &result)
			require.NoError(t, err,
				"brief --json should produce valid JSON for %s repo", repo.Name)

			// files_included should be a non-empty array.
			filesRaw, ok := result["files_included"].([]any)
			require.True(t, ok,
				"files_included should be an array for %s repo", repo.Name)
			assert.GreaterOrEqual(t, len(filesRaw), 1,
				"files_included for %s should have >= 1 entry, got %d",
				repo.Name, len(filesRaw))

			// token_count should be positive.
			tokenCount, ok := result["token_count"].(float64)
			require.True(t, ok,
				"token_count should be a number for %s repo", repo.Name)
			assert.Greater(t, tokenCount, float64(0),
				"token_count for %s should be positive", repo.Name)

			// content_hash should be present and non-empty.
			contentHash, ok := result["content_hash"].(string)
			assert.True(t, ok,
				"content_hash should be a string for %s repo", repo.Name)
			assert.NotEmpty(t, contentHash,
				"content_hash should not be empty for %s repo", repo.Name)
		})
	}
}

// TestDefaultProfile_GoRepo_ContainsGoContent verifies that processing a Go
// repository produces output containing identifiable content from the repo.
// The brief workflow outputs README content and module map -- not full source
// code -- so we check for content from the Go CLI repo's README.
func TestDefaultProfile_GoRepo_ContainsGoContent(t *testing.T) {
	t.Parallel()

	repo := repoByName("go-cli")
	skipIfMissing(t, repo)

	stdout, _, exitCode := runHarvxInDir(t, repo.Dir, []string{"brief", "--stdout"})

	require.Equal(t, 0, exitCode, "brief --stdout should exit 0 for go-cli repo")
	require.NotEmpty(t, stdout, "brief output should not be empty")

	// The go-cli repo README contains "gosync" and "go install".
	hasGoContent := strings.Contains(stdout, "gosync") ||
		strings.Contains(stdout, "go install") ||
		strings.Contains(stdout, "go.mod")
	assert.True(t, hasGoContent,
		"Go repo brief should contain identifiable Go project content (gosync, go install, or go.mod)")
}

// TestDefaultProfile_TSRepo_ContainsTSContent verifies that processing a
// TypeScript/Next.js repository produces output containing identifiable
// content from the repo. The brief includes README and config sections,
// so we check for content from the ts-nextjs repo's README.
func TestDefaultProfile_TSRepo_ContainsTSContent(t *testing.T) {
	t.Parallel()

	repo := repoByName("ts-nextjs")
	skipIfMissing(t, repo)

	stdout, _, exitCode := runHarvxInDir(t, repo.Dir, []string{"brief", "--stdout"})

	require.Equal(t, 0, exitCode, "brief --stdout should exit 0 for ts-nextjs repo")
	require.NotEmpty(t, stdout, "brief output should not be empty")

	// The ts-nextjs repo README contains "DevBlog" and "Next.js".
	hasTSContent := strings.Contains(stdout, "DevBlog") ||
		strings.Contains(stdout, "Next.js") ||
		strings.Contains(stdout, "TypeScript")
	assert.True(t, hasTSContent,
		"TypeScript repo brief should contain identifiable project content (DevBlog, Next.js, or TypeScript)")
}

// TestDefaultProfile_PythonRepo_ContainsPythonContent verifies that processing a
// Python/FastAPI repository produces output containing identifiable content
// from the repo. The brief includes README and config sections, so we check
// for content from the python-fastapi repo's README.
func TestDefaultProfile_PythonRepo_ContainsPythonContent(t *testing.T) {
	t.Parallel()

	repo := repoByName("python-fastapi")
	skipIfMissing(t, repo)

	stdout, _, exitCode := runHarvxInDir(t, repo.Dir, []string{"brief", "--stdout"})

	require.Equal(t, 0, exitCode, "brief --stdout should exit 0 for python-fastapi repo")
	require.NotEmpty(t, stdout, "brief output should not be empty")

	// The python-fastapi repo README contains "TaskAPI" and "FastAPI".
	hasPythonContent := strings.Contains(stdout, "TaskAPI") ||
		strings.Contains(stdout, "FastAPI") ||
		strings.Contains(stdout, "pyproject")
	assert.True(t, hasPythonContent,
		"Python repo brief should contain identifiable project content (TaskAPI, FastAPI, or pyproject)")
}

// TestDefaultProfile_MonoRepo_ContainsMultipleLanguages verifies that processing
// a monorepo produces output indicating multi-language project structure. The
// brief includes the README which describes both Go and TypeScript components.
func TestDefaultProfile_MonoRepo_ContainsMultipleLanguages(t *testing.T) {
	t.Parallel()

	repo := repoByName("monorepo")
	skipIfMissing(t, repo)

	stdout, _, exitCode := runHarvxInDir(t, repo.Dir, []string{"brief", "--stdout"})

	require.Equal(t, 0, exitCode, "brief --stdout should exit 0 for monorepo")
	require.NotEmpty(t, stdout, "brief output should not be empty")

	// The monorepo README mentions "Acme Platform", "Go", "TypeScript",
	// "packages/", and "services/". Check for multi-language indicators.
	hasGoIndicator := strings.Contains(stdout, "Go") ||
		strings.Contains(stdout, "go run")
	assert.True(t, hasGoIndicator,
		"monorepo brief should reference Go (from README content)")

	hasTSIndicator := strings.Contains(stdout, "TypeScript") ||
		strings.Contains(stdout, "packages/") ||
		strings.Contains(stdout, "npm")
	assert.True(t, hasTSIndicator,
		"monorepo brief should reference TypeScript or JS tooling (from README content)")
}

// skipIfMissing skips the test if the test repo directory does not exist.
// This allows the test suite to run even when some fixture repositories
// have not been set up.
func skipIfMissing(t *testing.T, repo TestRepo) {
	t.Helper()

	info, err := os.Stat(repo.Dir)
	if err != nil || !info.IsDir() {
		t.Skipf("test repo %q not found at %s; skipping", repo.Name, repo.Dir)
	}

	// Also verify the directory is not empty (just a .gitkeep).
	entries, err := os.ReadDir(repo.Dir)
	if err != nil || len(entries) <= 1 {
		t.Skipf("test repo %q at %s appears empty; skipping",
			repo.Name, repo.Dir)
	}
}

