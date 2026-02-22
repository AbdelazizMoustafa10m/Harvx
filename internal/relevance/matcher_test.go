package relevance

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// defaultMatcher returns a TierMatcher built from DefaultTierDefinitions.
func defaultMatcher(t *testing.T) *TierMatcher {
	t.Helper()
	m := NewTierMatcher(DefaultTierDefinitions())
	require.NotNil(t, m)
	return m
}

// ----------------------------------------------------------------------------
// TestNewTierMatcher
// ----------------------------------------------------------------------------

// TestNewTierMatcherNilDefs verifies that nil input is handled gracefully.
func TestNewTierMatcherNilDefs(t *testing.T) {
	t.Parallel()

	m := NewTierMatcher(nil)
	require.NotNil(t, m)
	assert.Empty(t, m.tiers)
}

// TestNewTierMatcherEmptyDefs verifies that an empty slice is handled.
func TestNewTierMatcherEmptyDefs(t *testing.T) {
	t.Parallel()

	m := NewTierMatcher([]TierDefinition{})
	require.NotNil(t, m)
	assert.Empty(t, m.tiers)
}

// TestNewTierMatcherDoesNotMutateInput verifies that NewTierMatcher does not
// sort or modify the caller's slice.
func TestNewTierMatcherDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	defs := []TierDefinition{
		{Tier: Tier3Tests, Patterns: []string{"*_test.go"}},
		{Tier: Tier0Critical, Patterns: []string{"go.mod"}},
	}
	original := []TierDefinition{
		{Tier: Tier3Tests, Patterns: []string{"*_test.go"}},
		{Tier: Tier0Critical, Patterns: []string{"go.mod"}},
	}

	_ = NewTierMatcher(defs)

	// The caller's slice must remain in original order.
	require.Len(t, defs, len(original))
	for i := range defs {
		assert.Equal(t, original[i].Tier, defs[i].Tier)
		assert.Equal(t, original[i].Patterns, defs[i].Patterns)
	}
}

// TestNewTierMatcherSortsByTier verifies that internal entries are sorted by
// ascending tier number regardless of input order.
func TestNewTierMatcherSortsByTier(t *testing.T) {
	t.Parallel()

	defs := []TierDefinition{
		{Tier: Tier5Low, Patterns: []string{"*.lock"}},
		{Tier: Tier2Secondary, Patterns: []string{"utils/**"}},
		{Tier: Tier0Critical, Patterns: []string{"go.mod"}},
	}
	m := NewTierMatcher(defs)
	require.Len(t, m.tiers, 3)
	assert.Equal(t, Tier0Critical, m.tiers[0].tier)
	assert.Equal(t, Tier2Secondary, m.tiers[1].tier)
	assert.Equal(t, Tier5Low, m.tiers[2].tier)
}

// TestNewTierMatcherFiltersInvalidPatterns verifies that syntactically invalid
// patterns are silently discarded and valid ones are retained.
func TestNewTierMatcherFiltersInvalidPatterns(t *testing.T) {
	t.Parallel()

	defs := []TierDefinition{
		{Tier: Tier0Critical, Patterns: []string{
			"go.mod",       // valid
			"[invalid",     // invalid -- unclosed bracket
			"Dockerfile",   // valid
		}},
	}
	m := NewTierMatcher(defs)
	require.Len(t, m.tiers, 1)
	assert.Equal(t, []string{"go.mod", "Dockerfile"}, m.tiers[0].patterns)
}

// TestNewTierMatcherAllInvalidPatterns verifies a definition with only invalid
// patterns still results in a tier entry (just with no patterns).
func TestNewTierMatcherAllInvalidPatterns(t *testing.T) {
	t.Parallel()

	defs := []TierDefinition{
		{Tier: Tier1Primary, Patterns: []string{"[bad", "[also-bad"}},
	}
	m := NewTierMatcher(defs)
	require.Len(t, m.tiers, 1)
	assert.Empty(t, m.tiers[0].patterns)
}

// ----------------------------------------------------------------------------
// TestMatch -- default tier definitions
// ----------------------------------------------------------------------------

// TestMatchDefaultTiers exercises all file path examples from the T-027 spec
// against DefaultTierDefinitions, verifying expected tier assignments.
func TestMatchDefaultTiers(t *testing.T) {
	t.Parallel()

	m := defaultMatcher(t)

	tests := []struct {
		name     string
		filePath string
		wantTier Tier
	}{
		// --- Tier 0: Critical / Config ---
		{name: "go.mod", filePath: "go.mod", wantTier: Tier0Critical},
		{name: "package.json", filePath: "package.json", wantTier: Tier0Critical},
		{name: "Dockerfile", filePath: "Dockerfile", wantTier: Tier0Critical},
		{name: "next.config.js (*.config.*)", filePath: "next.config.js", wantTier: Tier0Critical},
		{name: "vite.config.ts", filePath: "vite.config.ts", wantTier: Tier0Critical},
		{name: "tsconfig.json", filePath: "tsconfig.json", wantTier: Tier0Critical},
		{name: "Cargo.toml", filePath: "Cargo.toml", wantTier: Tier0Critical},
		{name: "Makefile", filePath: "Makefile", wantTier: Tier0Critical},
		{name: "pyproject.toml", filePath: "pyproject.toml", wantTier: Tier0Critical},
		{name: "setup.py", filePath: "setup.py", wantTier: Tier0Critical},
		{name: "requirements.txt", filePath: "requirements.txt", wantTier: Tier0Critical},
		{name: ".env.example", filePath: ".env.example", wantTier: Tier0Critical},
		{name: "docker-compose.yml", filePath: "docker-compose.yml", wantTier: Tier0Critical},
		{name: "docker-compose.yaml", filePath: "docker-compose.yaml", wantTier: Tier0Critical},

		// --- Tier 1: Primary source ---
		{name: "src/main.go", filePath: "src/main.go", wantTier: Tier1Primary},
		{name: "internal/server/handler.go", filePath: "internal/server/handler.go", wantTier: Tier1Primary},
		{name: "cmd/harvx/main.go", filePath: "cmd/harvx/main.go", wantTier: Tier1Primary},
		{name: "lib/util.ts", filePath: "lib/util.ts", wantTier: Tier1Primary},
		{name: "app/page.tsx", filePath: "app/page.tsx", wantTier: Tier1Primary},
		{name: "pkg/server/server.go", filePath: "pkg/server/server.go", wantTier: Tier1Primary},

		// --- Tier 2: Secondary source (explicit patterns) ---
		{name: "components/Button.tsx", filePath: "components/Button.tsx", wantTier: Tier2Secondary},
		{name: "utils/helpers.go (unmatched -> default)", filePath: "utils/helpers.go", wantTier: Tier2Secondary},
		{name: "helpers/format.go", filePath: "helpers/format.go", wantTier: Tier2Secondary},
		{name: "services/auth.go", filePath: "services/auth.go", wantTier: Tier2Secondary},
		{name: "api/v1/handler.go", filePath: "api/v1/handler.go", wantTier: Tier2Secondary},
		{name: "handlers/http.go", filePath: "handlers/http.go", wantTier: Tier2Secondary},
		{name: "controllers/user.go", filePath: "controllers/user.go", wantTier: Tier2Secondary},
		{name: "models/user.go", filePath: "models/user.go", wantTier: Tier2Secondary},

		// --- Tier 2: Unmatched -> DefaultUnmatchedTier ---
		{name: "random.go (unmatched)", filePath: "random.go", wantTier: Tier2Secondary},
		{name: "deep/nested/file.go (unmatched)", filePath: "deep/nested/file.go", wantTier: Tier2Secondary},

		// --- Tier 3: Tests ---
		{name: "main_test.go", filePath: "main_test.go", wantTier: Tier3Tests},
		// src/app.test.ts matches Tier1Primary ("src/**") before Tier3Tests ("*.test.ts").
		// First-wins: Tier1Primary.
		{name: "src/app.test.ts", filePath: "src/app.test.ts", wantTier: Tier1Primary},
		{name: "__tests__/unit/foo.ts", filePath: "__tests__/unit/foo.ts", wantTier: Tier3Tests},
		{name: "handler.spec.js", filePath: "handler.spec.js", wantTier: Tier3Tests},
		{name: "test/integration.go", filePath: "test/integration.go", wantTier: Tier3Tests},
		{name: "tests/e2e/flow.ts", filePath: "tests/e2e/flow.ts", wantTier: Tier3Tests},
		{name: "spec/models/user_spec.rb", filePath: "spec/models/user_spec.rb", wantTier: Tier3Tests},
		// internal/foo/bar_test.go matches Tier1Primary ("internal/**") before Tier3Tests ("*_test.go").
		// First-wins: Tier1Primary.
		{name: "internal/foo/bar_test.go", filePath: "internal/foo/bar_test.go", wantTier: Tier1Primary},

		// --- Tier 4: Docs ---
		{name: "README.md", filePath: "README.md", wantTier: Tier4Docs},
		{name: "docs/architecture.md", filePath: "docs/architecture.md", wantTier: Tier4Docs},
		{name: "CHANGELOG.md", filePath: "CHANGELOG.md", wantTier: Tier4Docs},
		{name: "LICENSE", filePath: "LICENSE", wantTier: Tier4Docs},
		{name: "notes.txt", filePath: "notes.txt", wantTier: Tier4Docs},

		// --- Tier 5: Low / CI ---
		{name: ".github/workflows/ci.yml", filePath: ".github/workflows/ci.yml", wantTier: Tier5Low},
		{name: "package-lock.json", filePath: "package-lock.json", wantTier: Tier5Low},
		{name: "yarn.lock", filePath: "yarn.lock", wantTier: Tier5Low},
		{name: "go.sum", filePath: "go.sum", wantTier: Tier5Low},
		{name: "Pipfile.lock", filePath: "Pipfile.lock", wantTier: Tier5Low},
		{name: "poetry.lock", filePath: "poetry.lock", wantTier: Tier5Low},
		{name: "pnpm-lock.yaml", filePath: "pnpm-lock.yaml", wantTier: Tier5Low},
		{name: ".circleci/config.yml", filePath: ".circleci/config.yml", wantTier: Tier5Low},
		{name: ".gitlab-ci.yml", filePath: ".gitlab-ci.yml", wantTier: Tier5Low},
		{name: "deps.lock", filePath: "deps.lock", wantTier: Tier5Low},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.Match(tt.filePath)
			assert.Equal(t, tt.wantTier, got,
				"Match(%q): want tier %d (%s), got tier %d (%s)",
				tt.filePath, tt.wantTier, tt.wantTier, got, got)
		})
	}
}

// ----------------------------------------------------------------------------
// TestMatch -- first-tier-wins rule
// ----------------------------------------------------------------------------

// TestMatchFirstTierWins verifies that when a file matches patterns in multiple
// tiers, the lowest-numbered (highest-priority) tier is returned.
func TestMatchFirstTierWins(t *testing.T) {
	t.Parallel()

	// "internal/foo_test.go" matches both Tier1Primary ("internal/**") and
	// Tier3Tests ("*_test.go"). Tier 1 must win.
	m := defaultMatcher(t)
	got := m.Match("internal/foo_test.go")
	assert.Equal(t, Tier1Primary, got,
		"file matching both Tier1 and Tier3 must be assigned Tier1")
}

// TestMatchFirstTierWinsCustom verifies first-wins with a custom definition set.
func TestMatchFirstTierWinsCustom(t *testing.T) {
	t.Parallel()

	defs := []TierDefinition{
		{Tier: Tier0Critical, Patterns: []string{"app/**"}},
		{Tier: Tier3Tests, Patterns: []string{"app/test/**"}},
	}
	m := NewTierMatcher(defs)

	// "app/test/foo.ts" matches Tier0Critical ("app/**") first.
	got := m.Match("app/test/foo.ts")
	assert.Equal(t, Tier0Critical, got)
}

// TestMatchWithinTierPatternOrder verifies that within a single tier, the
// first pattern that matches wins (positional ordering).
func TestMatchWithinTierPatternOrder(t *testing.T) {
	t.Parallel()

	// Both patterns in Tier0Critical match "special.go".
	// The first pattern is "special.go", the second is "*.go".
	// The result should still be Tier0Critical regardless of which pattern
	// matches -- but it confirms the engine iterates patterns in order.
	defs := []TierDefinition{
		{Tier: Tier0Critical, Patterns: []string{"special.go", "*.go"}},
		{Tier: Tier1Primary, Patterns: []string{"*.go"}},
	}
	m := NewTierMatcher(defs)

	got := m.Match("special.go")
	assert.Equal(t, Tier0Critical, got)

	got2 := m.Match("other.go")
	assert.Equal(t, Tier0Critical, got2,
		"*.go in Tier0Critical should match before Tier1Primary")
}

// ----------------------------------------------------------------------------
// TestMatch -- unmatched / empty
// ----------------------------------------------------------------------------

// TestMatchUnmatchedReturnsDefault verifies that files with no matching pattern
// return DefaultUnmatchedTier.
func TestMatchUnmatchedReturnsDefault(t *testing.T) {
	t.Parallel()

	m := defaultMatcher(t)
	got := m.Match("some/random/file.xyz")
	assert.Equal(t, DefaultUnmatchedTier, got)
}

// TestMatchEmptyTierList verifies that with no tier definitions, every file
// returns DefaultUnmatchedTier.
func TestMatchEmptyTierList(t *testing.T) {
	t.Parallel()

	m := NewTierMatcher(nil)

	files := []string{
		"go.mod",
		"src/main.go",
		"README.md",
		"random.bin",
	}
	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			assert.Equal(t, DefaultUnmatchedTier, m.Match(f),
				"empty matcher must return DefaultUnmatchedTier for %q", f)
		})
	}
}

// TestMatchEmptyFilePath verifies that an empty string does not panic and returns
// DefaultUnmatchedTier (it matches no pattern).
func TestMatchEmptyFilePath(t *testing.T) {
	t.Parallel()

	m := defaultMatcher(t)
	got := m.Match("")
	assert.Equal(t, DefaultUnmatchedTier, got)
}

// ----------------------------------------------------------------------------
// TestMatch -- path normalisation
// ----------------------------------------------------------------------------

// TestMatchLeadingDotSlashStripped verifies that paths with a leading "./" are
// normalised and match the same patterns as paths without it.
func TestMatchLeadingDotSlashStripped(t *testing.T) {
	t.Parallel()

	m := defaultMatcher(t)

	tests := []struct {
		with    string
		without string
	}{
		{"./go.mod", "go.mod"},
		{"./src/main.go", "src/main.go"},
		{"./.github/workflows/ci.yml", ".github/workflows/ci.yml"},
		{"./README.md", "README.md"},
	}

	for _, tt := range tests {
		t.Run(tt.with, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, m.Match(tt.without), m.Match(tt.with),
				"Match(%q) should equal Match(%q)", tt.with, tt.without)
		})
	}
}

// TestMatchWindowsStylePaths verifies that backslash paths are converted to
// forward slashes before matching (Windows compatibility).
func TestMatchWindowsStylePaths(t *testing.T) {
	t.Parallel()

	m := defaultMatcher(t)

	// On any platform, backslash separators should be normalised.
	tests := []struct {
		winPath  string
		expected Tier
	}{
		{`src\main.go`, Tier1Primary},
		{`internal\server\handler.go`, Tier1Primary},
		{`.github\workflows\ci.yml`, Tier5Low},
	}

	for _, tt := range tests {
		t.Run(tt.winPath, func(t *testing.T) {
			t.Parallel()
			got := m.Match(tt.winPath)
			assert.Equal(t, tt.expected, got,
				"Match(%q) should handle backslash separators", tt.winPath)
		})
	}
}

// ----------------------------------------------------------------------------
// TestMatch -- special characters in paths
// ----------------------------------------------------------------------------

// TestMatchSpecialCharacters verifies that files with unusual but valid
// characters are handled without panicking or incorrect results.
func TestMatchSpecialCharacters(t *testing.T) {
	t.Parallel()

	m := defaultMatcher(t)

	// These files contain characters that are valid in file names but might
	// trip up naïve pattern matchers.
	tests := []struct {
		name     string
		filePath string
		wantTier Tier
	}{
		{
			name:     "file with spaces (unmatched)",
			filePath: "my file.go",
			wantTier: Tier2Secondary,
		},
		{
			name:     "file with hyphens",
			filePath: "my-component.tsx",
			wantTier: Tier2Secondary,
		},
		{
			name:     "deeply nested path",
			filePath: "a/b/c/d/e/f/g/h/i/j/k/l.go",
			wantTier: Tier2Secondary,
		},
		{
			name:     "hidden file at root (unmatched)",
			filePath: ".eslintrc",
			wantTier: Tier2Secondary,
		},
		{
			name:     "file with parentheses",
			filePath: "src/(group)/page.tsx",
			wantTier: Tier1Primary,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Must not panic.
			got := m.Match(tt.filePath)
			assert.Equal(t, tt.wantTier, got)
		})
	}
}

// ----------------------------------------------------------------------------
// TestMatch -- custom / profile-override tier definitions
// ----------------------------------------------------------------------------

// TestMatchCustomTierDefinitions verifies that profile-defined tiers replace
// default tiers entirely (no merging).
func TestMatchCustomTierDefinitions(t *testing.T) {
	t.Parallel()

	// Custom profile: only two tiers, very different patterns.
	customDefs := []TierDefinition{
		{Tier: Tier0Critical, Patterns: []string{"app/api/**"}},
		{Tier: Tier5Low, Patterns: []string{"scripts/**"}},
	}
	m := NewTierMatcher(customDefs)

	tests := []struct {
		filePath string
		wantTier Tier
	}{
		// Matches Tier0Critical custom pattern.
		{"app/api/handler.go", Tier0Critical},
		{"app/api/v1/users.go", Tier0Critical},
		// Matches Tier5Low custom pattern.
		{"scripts/deploy.sh", Tier5Low},
		// "go.mod" no longer has a special meaning -- falls back to default.
		{"go.mod", DefaultUnmatchedTier},
		// "src/main.go" is not in custom tiers -- falls back.
		{"src/main.go", DefaultUnmatchedTier},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			t.Parallel()
			got := m.Match(tt.filePath)
			assert.Equal(t, tt.wantTier, got)
		})
	}
}

// TestMatchCustomTierOutOfOrder verifies that out-of-order tier definitions are
// re-sorted so priority is always by tier number, not input order.
func TestMatchCustomTierOutOfOrder(t *testing.T) {
	t.Parallel()

	// Supply Tier3 before Tier1 -- Tier1 should still win for "src/main.go".
	defs := []TierDefinition{
		{Tier: Tier3Tests, Patterns: []string{"src/**"}}, // same pattern in lower-priority tier
		{Tier: Tier1Primary, Patterns: []string{"src/**"}},
	}
	m := NewTierMatcher(defs)

	got := m.Match("src/main.go")
	assert.Equal(t, Tier1Primary, got,
		"Tier1Primary must win over Tier3Tests even when supplied in wrong order")
}

// ----------------------------------------------------------------------------
// TestMatch -- no-file-in-multiple-tiers invariant
// ----------------------------------------------------------------------------

// TestNoFileInMultipleTiers verifies that ClassifyFiles assigns each file to
// exactly one tier (the invariant stated in the acceptance criteria).
func TestNoFileInMultipleTiers(t *testing.T) {
	t.Parallel()

	files := []string{
		"go.mod",
		"src/main.go",
		"internal/foo/bar.go",
		"components/Button.tsx",
		"main_test.go",
		"README.md",
		".github/workflows/ci.yml",
		"package-lock.json",
		"random.xyz",
	}

	result := ClassifyFiles(files, DefaultTierDefinitions())

	// Each file should appear exactly once in the result map.
	require.Len(t, result, len(files),
		"ClassifyFiles must return exactly one entry per input file")

	for _, f := range files {
		_, ok := result[f]
		assert.True(t, ok, "file %q must be present in result", f)
	}
}

// ----------------------------------------------------------------------------
// TestClassifyFiles
// ----------------------------------------------------------------------------

// TestClassifyFilesEmptyFiles verifies that an empty file list returns an empty map.
func TestClassifyFilesEmptyFiles(t *testing.T) {
	t.Parallel()

	result := ClassifyFiles([]string{}, DefaultTierDefinitions())
	assert.Empty(t, result)
}

// TestClassifyFilesNilFiles verifies that a nil file list returns an empty map.
func TestClassifyFilesNilFiles(t *testing.T) {
	t.Parallel()

	result := ClassifyFiles(nil, DefaultTierDefinitions())
	assert.Empty(t, result)
}

// TestClassifyFilesNilTiers verifies that nil tiers assigns all files to
// DefaultUnmatchedTier.
func TestClassifyFilesNilTiers(t *testing.T) {
	t.Parallel()

	files := []string{"go.mod", "src/main.go", "README.md"}
	result := ClassifyFiles(files, nil)

	for _, f := range files {
		tier, ok := result[f]
		require.True(t, ok)
		assert.Equal(t, DefaultUnmatchedTier, tier,
			"file %q must be DefaultUnmatchedTier when tiers is nil", f)
	}
}

// TestClassifyFilesPreservesOriginalKeys verifies that the map keys are the
// original (non-normalised) file paths as supplied by the caller.
func TestClassifyFilesPreservesOriginalKeys(t *testing.T) {
	t.Parallel()

	files := []string{"./go.mod", "./src/main.go"}
	result := ClassifyFiles(files, DefaultTierDefinitions())

	for _, f := range files {
		_, ok := result[f]
		assert.True(t, ok, "original key %q must be present in result", f)
	}
}

// TestClassifyFilesBulkResult verifies specific assignments from ClassifyFiles.
func TestClassifyFilesBulkResult(t *testing.T) {
	t.Parallel()

	files := []string{
		"go.mod",
		"src/main.go",
		"components/Button.tsx",
		"main_test.go",
		"README.md",
		".github/workflows/ci.yml",
		"package-lock.json",
		"mystery.xyz",
	}

	result := ClassifyFiles(files, DefaultTierDefinitions())

	expected := map[string]Tier{
		"go.mod":                    Tier0Critical,
		"src/main.go":               Tier1Primary,
		"components/Button.tsx":     Tier2Secondary,
		"main_test.go":              Tier3Tests,
		"README.md":                 Tier4Docs,
		".github/workflows/ci.yml":  Tier5Low,
		"package-lock.json":         Tier5Low,
		"mystery.xyz":               DefaultUnmatchedTier,
	}

	for filePath, wantTier := range expected {
		t.Run(filePath, func(t *testing.T) {
			got, ok := result[filePath]
			require.True(t, ok, "key %q must be present", filePath)
			assert.Equal(t, wantTier, got)
		})
	}
}

// TestClassifyFilesDuplicatePaths verifies that duplicate paths in the input
// are all present in the result with the correct tier.
func TestClassifyFilesDuplicatePaths(t *testing.T) {
	t.Parallel()

	// Go maps only hold one value per key, so a duplicate path will just
	// overwrite. The important thing is no panic and correct tier.
	files := []string{"go.mod", "go.mod", "src/main.go"}
	result := ClassifyFiles(files, DefaultTierDefinitions())

	assert.Equal(t, Tier0Critical, result["go.mod"])
	assert.Equal(t, Tier1Primary, result["src/main.go"])
}

// ----------------------------------------------------------------------------
// TestSortTierDefinitions (internal)
// ----------------------------------------------------------------------------

// TestSortTierDefinitionsAlreadySorted verifies a no-op when input is sorted.
func TestSortTierDefinitionsAlreadySorted(t *testing.T) {
	t.Parallel()

	defs := []TierDefinition{
		{Tier: Tier0Critical},
		{Tier: Tier1Primary},
		{Tier: Tier3Tests},
		{Tier: Tier5Low},
	}
	original := make([]TierDefinition, len(defs))
	copy(original, defs)

	sortTierDefinitions(defs)

	for i, d := range defs {
		assert.Equal(t, original[i].Tier, d.Tier)
	}
}

// TestSortTierDefinitionsReversed verifies reverse input is correctly sorted.
func TestSortTierDefinitionsReversed(t *testing.T) {
	t.Parallel()

	defs := []TierDefinition{
		{Tier: Tier5Low},
		{Tier: Tier4Docs},
		{Tier: Tier3Tests},
		{Tier: Tier2Secondary},
		{Tier: Tier1Primary},
		{Tier: Tier0Critical},
	}

	sortTierDefinitions(defs)

	for i := 0; i < len(defs)-1; i++ {
		assert.LessOrEqual(t, int(defs[i].Tier), int(defs[i+1].Tier))
	}
}

// ----------------------------------------------------------------------------
// TestNormalisePath (internal)
// ----------------------------------------------------------------------------

// TestNormalisePath verifies all normalisation cases.
func TestNormalisePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"go.mod", "go.mod"},
		{"./go.mod", "go.mod"},
		{"src/main.go", "src/main.go"},
		{"./src/main.go", "src/main.go"},
		{"", ""},
		{"./", ""},
		// Double dot-slash is NOT stripped (only single leading ./).
		{"././go.mod", "./go.mod"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.input), func(t *testing.T) {
			t.Parallel()
			got := normalisePath(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ----------------------------------------------------------------------------
// Benchmark
// ----------------------------------------------------------------------------

// BenchmarkClassifyFiles10K measures throughput for 10 000 files against the
// default tier definitions (~20 patterns).
func BenchmarkClassifyFiles10K(b *testing.B) {
	// Build a representative set of 10 000 file paths.
	patterns := []string{
		"go.mod", "package.json", "Dockerfile", "Makefile",
		"src/%d/main.go", "internal/%d/handler.go", "cmd/%d/main.go",
		"components/%d/widget.tsx", "utils/%d/helpers.go",
		"main_test.go", "internal/%d/handler_test.go",
		"README.md", "docs/%d/guide.md",
		".github/workflows/%d.yml", "package-lock.json",
		"random/%d/unknown.xyz",
	}

	files := make([]string, 0, 10000)
	for i := 0; len(files) < 10000; i++ {
		for _, p := range patterns {
			if len(files) >= 10000 {
				break
			}
			if strings.Contains(p, "%d") {
				files = append(files, fmt.Sprintf(p, i))
			} else {
				files = append(files, p)
			}
		}
	}

	defs := DefaultTierDefinitions()

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		_ = ClassifyFiles(files, defs)
	}
}

// BenchmarkMatchSingle measures the per-file Match cost with the default tiers.
func BenchmarkMatchSingle(b *testing.B) {
	m := NewTierMatcher(DefaultTierDefinitions())
	path := "internal/server/handler.go"

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		_ = m.Match(path)
	}
}
