package workflows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/config"
)

// newTestWorkspaceConfig creates a WorkspaceConfig suitable for testing with
// multiple repos. Paths default to empty unless overridden.
func newTestWorkspaceConfig() *config.WorkspaceConfig {
	return &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name:        "MyOrg Platform",
			Description: "Microservices platform with shared UI library",
			Repos: []config.WorkspaceRepo{
				{
					Name:           "api-gateway",
					Path:           "",
					Description:    "Express.js API gateway, handles auth and routing",
					Entrypoints:    []string{"src/server.ts", "src/routes/"},
					IntegratesWith: []string{"user-service", "billing-service"},
				},
				{
					Name:           "user-service",
					Path:           "",
					Description:    "User management microservice (Go)",
					Entrypoints:    []string{"cmd/server/main.go", "internal/handlers/"},
					IntegratesWith: []string{"api-gateway"},
					SharedSchemas:  []string{"proto/user.proto"},
				},
				{
					Name:           "billing-service",
					Path:           "",
					Description:    "Billing and payments service",
					Entrypoints:    []string{"src/main.py"},
					IntegratesWith: []string{"api-gateway", "user-service"},
					SharedSchemas:  []string{"proto/billing.proto"},
					Docs:           []string{"docs/billing-api.md", "docs/stripe-integration.md"},
				},
			},
		},
	}
}

func TestGenerateWorkspace_BasicMarkdown(t *testing.T) {
	t.Parallel()

	cfg := newTestWorkspaceConfig()
	result, err := GenerateWorkspace(WorkspaceOptions{
		Config: cfg,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.Content)
	assert.Greater(t, result.TokenCount, 0)
	assert.NotZero(t, result.ContentHash)
	assert.NotEmpty(t, result.FormattedHash)
	assert.Equal(t, 3, result.RepoCount)

	// Verify markdown structure.
	assert.Contains(t, result.Content, "# Workspace: MyOrg Platform")
	assert.Contains(t, result.Content, "> Microservices platform with shared UI library")
	assert.Contains(t, result.Content, "## Repositories")
	assert.Contains(t, result.Content, "### api-gateway")
	assert.Contains(t, result.Content, "### billing-service")
	assert.Contains(t, result.Content, "### user-service")
	assert.Contains(t, result.Content, "## Integration Graph")
	assert.Contains(t, result.Content, "## Shared Schemas")
}

func TestGenerateWorkspace_ConfigRequired(t *testing.T) {
	t.Parallel()

	_, err := GenerateWorkspace(WorkspaceOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestGenerateWorkspace_Deterministic(t *testing.T) {
	t.Parallel()

	cfg := newTestWorkspaceConfig()
	opts := WorkspaceOptions{Config: cfg}

	// Run 5 times.
	results := make([]*WorkspaceResult, 5)
	for i := 0; i < 5; i++ {
		var err error
		results[i], err = GenerateWorkspace(opts)
		require.NoError(t, err)
	}

	for i := 1; i < len(results); i++ {
		assert.Equal(t, results[0].Content, results[i].Content,
			"run %d content must be identical to run 0", i)
		assert.Equal(t, results[0].ContentHash, results[i].ContentHash,
			"run %d content hash must be identical to run 0", i)
		assert.Equal(t, results[0].FormattedHash, results[i].FormattedHash,
			"run %d formatted hash must be identical to run 0", i)
		assert.Equal(t, results[0].TokenCount, results[i].TokenCount,
			"run %d token count must be identical to run 0", i)
	}
}

func TestGenerateWorkspace_ReposSortedByName(t *testing.T) {
	t.Parallel()

	// Provide repos in reverse order.
	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name: "Test",
			Repos: []config.WorkspaceRepo{
				{Name: "zeta-service"},
				{Name: "alpha-service"},
				{Name: "mu-service"},
			},
		},
	}

	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	// Verify alpha comes before mu comes before zeta in the output.
	alphaIdx := strings.Index(result.Content, "### alpha-service")
	muIdx := strings.Index(result.Content, "### mu-service")
	zetaIdx := strings.Index(result.Content, "### zeta-service")

	assert.Greater(t, alphaIdx, -1, "alpha-service should be in output")
	assert.Greater(t, muIdx, -1, "mu-service should be in output")
	assert.Greater(t, zetaIdx, -1, "zeta-service should be in output")
	assert.Less(t, alphaIdx, muIdx, "alpha should come before mu")
	assert.Less(t, muIdx, zetaIdx, "mu should come before zeta")
}

func TestGenerateWorkspace_ClaudeXMLTarget(t *testing.T) {
	t.Parallel()

	cfg := newTestWorkspaceConfig()
	result, err := GenerateWorkspace(WorkspaceOptions{
		Config: cfg,
		Target: "claude",
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, `<workspace name="MyOrg Platform">`)
	assert.Contains(t, result.Content, "<description>Microservices platform with shared UI library</description>")
	assert.Contains(t, result.Content, "<repos>")
	assert.Contains(t, result.Content, `<repo name="api-gateway"`)
	assert.Contains(t, result.Content, "<entrypoints>")
	assert.Contains(t, result.Content, "<integrates-with>")
	assert.Contains(t, result.Content, "</workspace>")
	assert.Contains(t, result.Content, "<integrations>")
	assert.Contains(t, result.Content, "<shared-schemas>")
}

func TestGenerateWorkspace_EmptyRepos(t *testing.T) {
	t.Parallel()

	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name:        "Empty Workspace",
			Description: "No repos yet",
		},
	}

	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Content, "# Workspace: Empty Workspace")
	assert.Equal(t, 0, result.RepoCount)
	assert.NotContains(t, result.Content, "## Repositories")
	assert.NotContains(t, result.Content, "## Integration Graph")
	assert.NotContains(t, result.Content, "## Shared Schemas")
}

func TestGenerateWorkspace_EmptyName(t *testing.T) {
	t.Parallel()

	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Repos: []config.WorkspaceRepo{
				{Name: "my-repo", Description: "A repo"},
			},
		},
	}

	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "# Workspace: Workspace")
}

func TestGenerateWorkspace_IntegrationEdges(t *testing.T) {
	t.Parallel()

	cfg := newTestWorkspaceConfig()
	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	// Check integration edges are present.
	assert.Contains(t, result.Content, "api-gateway \u2192 billing-service, user-service")
	assert.Contains(t, result.Content, "billing-service \u2192 api-gateway, user-service")
	assert.Contains(t, result.Content, "user-service \u2192 api-gateway")
}

func TestGenerateWorkspace_SharedSchemas(t *testing.T) {
	t.Parallel()

	cfg := newTestWorkspaceConfig()
	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "`proto/billing.proto` (billing-service)")
	assert.Contains(t, result.Content, "`proto/user.proto` (user-service)")
}

func TestGenerateWorkspace_EntrypointsAndDocs(t *testing.T) {
	t.Parallel()

	cfg := newTestWorkspaceConfig()
	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "`src/server.ts`")
	assert.Contains(t, result.Content, "`src/routes/`")
	assert.Contains(t, result.Content, "`docs/billing-api.md`")
	assert.Contains(t, result.Content, "`docs/stripe-integration.md`")
}

func TestGenerateWorkspace_CustomTokenCounter(t *testing.T) {
	t.Parallel()

	calls := 0
	counter := func(text string) int {
		calls++
		return len(text) / 4
	}

	cfg := newTestWorkspaceConfig()
	result, err := GenerateWorkspace(WorkspaceOptions{
		Config:       cfg,
		TokenCounter: counter,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Greater(t, calls, 0, "custom token counter should be called")
}

func TestGenerateWorkspace_ContentHashChanges(t *testing.T) {
	t.Parallel()

	cfg1 := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name:  "Version 1",
			Repos: []config.WorkspaceRepo{{Name: "repo-a"}},
		},
	}

	cfg2 := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name:  "Version 2",
			Repos: []config.WorkspaceRepo{{Name: "repo-a"}},
		},
	}

	result1, err := GenerateWorkspace(WorkspaceOptions{Config: cfg1})
	require.NoError(t, err)

	result2, err := GenerateWorkspace(WorkspaceOptions{Config: cfg2})
	require.NoError(t, err)

	assert.NotEqual(t, result1.ContentHash, result2.ContentHash,
		"different workspace names must produce different hashes")
}

func TestGenerateWorkspace_DeepMode(t *testing.T) {
	t.Parallel()

	// Create a temp directory to use as a repo path.
	repoDir := t.TempDir()

	// Create some files and directories.
	require.NoError(t, os.Mkdir(filepath.Join(repoDir, "src"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(repoDir, "docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main"), 0o644))

	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name: "Deep Test",
			Repos: []config.WorkspaceRepo{
				{
					Name: "my-repo",
					Path: repoDir,
				},
			},
		},
	}

	// Without deep.
	resultNormal, err := GenerateWorkspace(WorkspaceOptions{
		Config: cfg,
		Deep:   false,
	})
	require.NoError(t, err)
	assert.NotContains(t, resultNormal.Content, "Directory listing")

	// With deep.
	resultDeep, err := GenerateWorkspace(WorkspaceOptions{
		Config: cfg,
		Deep:   true,
	})
	require.NoError(t, err)
	assert.Contains(t, resultDeep.Content, "Directory listing")
	assert.Contains(t, resultDeep.Content, "src/")
	assert.Contains(t, resultDeep.Content, "docs/")
	assert.Contains(t, resultDeep.Content, "README.md")
	assert.Contains(t, resultDeep.Content, "main.go")
}

func TestGenerateWorkspace_DeepModeXML(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repoDir, "lib"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "index.ts"), []byte("export {}"), 0o644))

	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name: "Deep XML Test",
			Repos: []config.WorkspaceRepo{
				{
					Name: "frontend",
					Path: repoDir,
				},
			},
		},
	}

	result, err := GenerateWorkspace(WorkspaceOptions{
		Config: cfg,
		Deep:   true,
		Target: "claude",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Content, "<directory-listing>")
	assert.Contains(t, result.Content, "</directory-listing>")
	assert.Contains(t, result.Content, "lib/")
}

func TestGenerateWorkspace_PathNotFound(t *testing.T) {
	t.Parallel()

	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name: "Missing Path Test",
			Repos: []config.WorkspaceRepo{
				{
					Name:        "nonexistent",
					Path:        "/tmp/does-not-exist-harvx-test-xyz",
					Description: "This path does not exist",
				},
			},
		},
	}

	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "(not found)")
}

func TestGenerateWorkspace_PathFound(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name: "Found Path Test",
			Repos: []config.WorkspaceRepo{
				{
					Name: "my-repo",
					Path: repoDir,
				},
			},
		},
	}

	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	assert.Contains(t, result.Content, repoDir)
	assert.NotContains(t, result.Content, "(not found)")
}

func TestGenerateWorkspace_NoIntegrations(t *testing.T) {
	t.Parallel()

	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name: "No Integrations",
			Repos: []config.WorkspaceRepo{
				{Name: "standalone"},
			},
		},
	}

	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	assert.NotContains(t, result.Content, "## Integration Graph")
}

func TestGenerateWorkspace_NoSharedSchemas(t *testing.T) {
	t.Parallel()

	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name: "No Schemas",
			Repos: []config.WorkspaceRepo{
				{Name: "simple", IntegratesWith: []string{}},
			},
		},
	}

	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	assert.NotContains(t, result.Content, "## Shared Schemas")
}

func TestGenerateWorkspace_StandardModeTokenBudget(t *testing.T) {
	t.Parallel()

	cfg := newTestWorkspaceConfig()
	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	// Standard mode should be small: <= ~2K tokens.
	assert.LessOrEqual(t, result.TokenCount, 2000,
		"standard mode output should be within 2K token budget")
}

func TestGenerateWorkspace_DeepModeNonexistentPath(t *testing.T) {
	t.Parallel()

	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name: "Deep Missing",
			Repos: []config.WorkspaceRepo{
				{
					Name:        "ghost",
					Path:        "/tmp/does-not-exist-harvx-deep-test-xyz",
					Description: "Missing repo",
				},
			},
		},
	}

	// Deep mode with missing path should not crash.
	result, err := GenerateWorkspace(WorkspaceOptions{
		Config: cfg,
		Deep:   true,
	})
	require.NoError(t, err)
	assert.Contains(t, result.Content, "(not found)")
	assert.NotContains(t, result.Content, "Directory listing")
}

func TestBuildIntegrationEdges_Sorted(t *testing.T) {
	t.Parallel()

	repos := []config.WorkspaceRepo{
		{Name: "b-service", IntegratesWith: []string{"c-service", "a-service"}},
		{Name: "a-service", IntegratesWith: []string{"b-service"}},
	}

	edges := buildIntegrationEdges(repos)

	require.Len(t, edges, 2)
	// Repos are already sorted by name from the caller, so b comes after a
	// but within each edge targets are sorted.
	assert.Equal(t, "b-service \u2192 a-service, c-service", edges[0])
	assert.Equal(t, "a-service \u2192 b-service", edges[1])
}

func TestBuildSharedSchemas_Sorted(t *testing.T) {
	t.Parallel()

	repos := []config.WorkspaceRepo{
		{Name: "svc-b", SharedSchemas: []string{"proto/z.proto", "proto/a.proto"}},
		{Name: "svc-a", SharedSchemas: []string{"proto/m.proto"}},
	}

	schemas := buildSharedSchemas(repos)

	require.Len(t, schemas, 3)
	// Should be sorted alphabetically.
	assert.Equal(t, "`proto/a.proto` (svc-b)", schemas[0])
	assert.Equal(t, "`proto/m.proto` (svc-a)", schemas[1])
	assert.Equal(t, "`proto/z.proto` (svc-b)", schemas[2])
}

func TestFormatCodeList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []string
		want  string
	}{
		{
			name:  "single item",
			items: []string{"src/main.ts"},
			want:  "`src/main.ts`",
		},
		{
			name:  "multiple items",
			items: []string{"src/main.ts", "src/routes/"},
			want:  "`src/main.ts`, `src/routes/`",
		},
		{
			name:  "empty list",
			items: []string{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatCodeList(tt.items)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadDirListing_MaxEntries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Create 35 visible files (exceeding maxDirEntries of 30).
	for i := 0; i < 35; i++ {
		name := filepath.Join(dir, strings.Repeat("a", i+1)+".txt")
		require.NoError(t, os.WriteFile(name, []byte("x"), 0o644))
	}

	listing := readDirListing(dir, 30)
	assert.Contains(t, listing, "... (truncated)")

	// Count lines.
	lines := strings.Split(strings.TrimSpace(listing), "\n")
	assert.LessOrEqual(t, len(lines), 31, "should have at most 30 entries + truncation marker")
}

func TestReadDirListing_SkipsHidden(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("x"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

	listing := readDirListing(dir, 30)
	assert.Contains(t, listing, "visible.txt")
	assert.NotContains(t, listing, ".hidden")
	assert.NotContains(t, listing, ".git")
}

func TestReadDirListing_DirectoryTrailingSlash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644))

	listing := readDirListing(dir, 30)
	assert.Contains(t, listing, "subdir/")
	assert.Contains(t, listing, "file.txt")
	assert.NotContains(t, listing, "file.txt/")
}

func TestReadDirListing_NonexistentDir(t *testing.T) {
	t.Parallel()

	listing := readDirListing("/tmp/does-not-exist-harvx-xyz", 30)
	assert.Empty(t, listing)
}

func TestXmlEscape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no special chars", input: "hello world", want: "hello world"},
		{name: "ampersand", input: "a & b", want: "a &amp; b"},
		{name: "less than", input: "a < b", want: "a &lt; b"},
		{name: "greater than", input: "a > b", want: "a &gt; b"},
		{name: "all special", input: "a & b < c > d", want: "a &amp; b &lt; c &gt; d"},
		{name: "empty", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, xmlEscape(tt.input))
		})
	}
}

func TestPathExistsOnDisk(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	assert.True(t, pathExistsOnDisk(dir))
	assert.False(t, pathExistsOnDisk(filepath.Join(dir, "nonexistent")))
}

func TestGenerateWorkspace_XMLNoDescription(t *testing.T) {
	t.Parallel()

	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name: "Minimal",
			Repos: []config.WorkspaceRepo{
				{Name: "only-repo"},
			},
		},
	}

	result, err := GenerateWorkspace(WorkspaceOptions{
		Config: cfg,
		Target: "claude",
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, `<workspace name="Minimal">`)
	// No description tag when description is empty.
	assert.NotContains(t, result.Content, "<description>")
}

func TestGenerateWorkspace_XMLEscapesDescription(t *testing.T) {
	t.Parallel()

	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name:        "Special Chars",
			Description: "Uses A & B, handles <tags>",
			Repos: []config.WorkspaceRepo{
				{
					Name:        "xml-repo",
					Description: "Handles <events> & more",
				},
			},
		},
	}

	result, err := GenerateWorkspace(WorkspaceOptions{
		Config: cfg,
		Target: "claude",
	})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "Uses A &amp; B, handles &lt;tags&gt;")
	assert.Contains(t, result.Content, "Handles &lt;events&gt; &amp; more")
}

func TestGenerateWorkspace_DoesNotMutateCaller(t *testing.T) {
	t.Parallel()

	cfg := newTestWorkspaceConfig()
	originalFirst := cfg.Workspace.Repos[0].Name

	_, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	// The caller's config should not have been sorted.
	assert.Equal(t, originalFirst, cfg.Workspace.Repos[0].Name,
		"GenerateWorkspace must not mutate the caller's repo slice")
}

func TestGenerateWorkspace_SingleRepo(t *testing.T) {
	t.Parallel()

	cfg := &config.WorkspaceConfig{
		Workspace: config.WorkspaceManifest{
			Name:        "Single",
			Description: "Just one repo",
			Repos: []config.WorkspaceRepo{
				{
					Name:        "mono",
					Description: "The monorepo",
					Entrypoints: []string{"src/index.ts"},
				},
			},
		},
	}

	result, err := GenerateWorkspace(WorkspaceOptions{Config: cfg})
	require.NoError(t, err)

	assert.Contains(t, result.Content, "### mono")
	assert.Equal(t, 1, result.RepoCount)
	assert.NotContains(t, result.Content, "## Integration Graph")
}
