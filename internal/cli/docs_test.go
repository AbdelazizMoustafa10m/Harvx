package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// projectRoot walks up from the current directory until it finds go.mod,
// returning the directory that contains it. This locates the repository root
// regardless of where `go test` is invoked from.
func projectRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err, "failed to get working directory")

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (no go.mod found in any parent)")
		}
		dir = parent
	}
}

// ---------------------------------------------------------------------------
// Test: hooks.json is valid JSON with expected structure
// ---------------------------------------------------------------------------

func TestHooksJSON_ValidStructure(t *testing.T) {
	t.Parallel()

	root := projectRoot(t)
	path := filepath.Join(root, "docs", "templates", "hooks.json")

	data, err := os.ReadFile(path)
	require.NoError(t, err, "hooks.json must be readable")

	// Verify it decodes as valid JSON.
	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "hooks.json must be valid JSON")

	// Verify the top-level "hooks" key exists.
	hooks, ok := parsed["hooks"]
	require.True(t, ok, "hooks.json must contain a top-level 'hooks' key")

	// Verify hooks is an object with a "SessionStart" array.
	hooksMap, ok := hooks.(map[string]any)
	require.True(t, ok, "hooks must be a JSON object")

	sessionStart, ok := hooksMap["SessionStart"]
	require.True(t, ok, "hooks must contain 'SessionStart' key")

	sessionStartArr, ok := sessionStart.([]any)
	require.True(t, ok, "hooks.SessionStart must be a JSON array")
	assert.NotEmpty(t, sessionStartArr, "hooks.SessionStart must have at least one entry")

	// Verify the first entry has "command" and "timeout" fields.
	firstHook, ok := sessionStartArr[0].(map[string]any)
	require.True(t, ok, "first SessionStart entry must be a JSON object")
	assert.Contains(t, firstHook, "command", "hook entry must have 'command' field")
	assert.Contains(t, firstHook, "timeout", "hook entry must have 'timeout' field")
}

// ---------------------------------------------------------------------------
// Test: CLAUDE.md template is under 500 tokens and contains key strings
// ---------------------------------------------------------------------------

func TestCLAUDETemplate_UnderTokenBudget(t *testing.T) {
	t.Parallel()

	root := projectRoot(t)
	path := filepath.Join(root, "docs", "templates", "CLAUDE.md")

	data, err := os.ReadFile(path)
	require.NoError(t, err, "CLAUDE.md template must be readable")

	content := string(data)

	// Estimate tokens using len/4 (the "none" tokenizer approach).
	estimatedTokens := len(content) / 4
	assert.Less(t, estimatedTokens, 500,
		"CLAUDE.md template must be under 500 tokens (estimated %d)", estimatedTokens)

	// Verify key strings are present.
	assert.Contains(t, content, "harvx brief",
		"CLAUDE.md must reference 'harvx brief'")
	assert.Contains(t, content, "Dynamic Context",
		"CLAUDE.md must contain 'Dynamic Context' section")
}

// ---------------------------------------------------------------------------
// Test: all harvx commands referenced in docs exist in known command set
// ---------------------------------------------------------------------------

func TestDocsReferenceValidCommands(t *testing.T) {
	t.Parallel()

	root := projectRoot(t)

	knownCommands := map[string]bool{
		"generate":     true,
		"gen":          true,
		"version":      true,
		"completion":   true,
		"preview":      true,
		"brief":        true,
		"review-slice": true,
		"slice":        true,
		"workspace":    true,
		"diff":         true,
		"cache":        true,
		"config":       true,
		"profiles":     true,
	}

	// Regex to match `harvx <subcommand>` patterns.
	// Matches: harvx brief, harvx review-slice, harvx generate, etc.
	cmdPattern := regexp.MustCompile(`harvx\s+([a-z][-a-z]*)`)

	docFiles := []struct {
		name string
		path string
	}{
		{"session-bootstrap", filepath.Join(root, "docs", "guides", "session-bootstrap.md")},
		{"review-pipeline", filepath.Join(root, "docs", "guides", "review-pipeline.md")},
		{"workspace-setup", filepath.Join(root, "docs", "guides", "workspace-setup.md")},
		{"quick-context", filepath.Join(root, "docs", "recipes", "quick-context.md")},
		{"pipeline-review", filepath.Join(root, "docs", "recipes", "pipeline-review.md")},
		{"ci-integration", filepath.Join(root, "docs", "recipes", "ci-integration.md")},
	}

	for _, doc := range docFiles {
		t.Run(doc.name, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(doc.path)
			require.NoError(t, err, "doc file %s must be readable", doc.name)

			matches := cmdPattern.FindAllStringSubmatch(string(data), -1)
			require.NotEmpty(t, matches,
				"doc file %s should reference at least one harvx command", doc.name)

			for _, match := range matches {
				subcommand := match[1]
				assert.True(t, knownCommands[subcommand],
					"doc %s references unknown command 'harvx %s'", doc.name, subcommand)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test: GitHub Actions YAML examples have valid structure
// ---------------------------------------------------------------------------

func TestCIRecipe_YAMLStructure(t *testing.T) {
	t.Parallel()

	root := projectRoot(t)
	path := filepath.Join(root, "docs", "recipes", "ci-integration.md")

	data, err := os.ReadFile(path)
	require.NoError(t, err, "ci-integration.md must be readable")

	content := string(data)

	// Extract YAML code blocks (```yaml ... ```).
	yamlBlockPattern := regexp.MustCompile("(?s)```yaml\\s*\\n(.*?)```")
	yamlBlocks := yamlBlockPattern.FindAllStringSubmatch(content, -1)
	require.NotEmpty(t, yamlBlocks,
		"ci-integration.md must contain at least one YAML code block")

	// The first YAML block should be a complete workflow with name/on/jobs.
	firstBlock := yamlBlocks[0][1]

	expectedKeys := []struct {
		key  string
		desc string
	}{
		{"name:", "workflow name"},
		{"on:", "trigger definition"},
		{"jobs:", "jobs definition"},
	}

	for _, ek := range expectedKeys {
		assert.True(t, strings.Contains(firstBlock, ek.key),
			"first YAML block must contain %s (%s)", ek.key, ek.desc)
	}
}

// ---------------------------------------------------------------------------
// Test: all 8 expected documentation files from T-074 exist
// ---------------------------------------------------------------------------

func TestDocumentationFilesExist(t *testing.T) {
	t.Parallel()

	root := projectRoot(t)

	expectedFiles := []struct {
		name string
		path string
	}{
		{"session-bootstrap guide", filepath.Join("docs", "guides", "session-bootstrap.md")},
		{"review-pipeline guide", filepath.Join("docs", "guides", "review-pipeline.md")},
		{"workspace-setup guide", filepath.Join("docs", "guides", "workspace-setup.md")},
		{"CLAUDE.md template", filepath.Join("docs", "templates", "CLAUDE.md")},
		{"hooks.json template", filepath.Join("docs", "templates", "hooks.json")},
		{"quick-context recipe", filepath.Join("docs", "recipes", "quick-context.md")},
		{"pipeline-review recipe", filepath.Join("docs", "recipes", "pipeline-review.md")},
		{"ci-integration recipe", filepath.Join("docs", "recipes", "ci-integration.md")},
	}

	for _, ef := range expectedFiles {
		t.Run(ef.name, func(t *testing.T) {
			t.Parallel()

			fullPath := filepath.Join(root, ef.path)
			info, err := os.Stat(fullPath)
			require.NoError(t, err, "documentation file %s must exist at %s", ef.name, ef.path)
			assert.True(t, info.Size() > 0, "documentation file %s must not be empty", ef.name)
		})
	}
}
