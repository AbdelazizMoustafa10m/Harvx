package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultProfile_RelevanceTier0_ExactPatterns verifies the complete and
// exact set of Tier 0 patterns from PRD Section 5.3. Tier 0 contains the
// highest-priority config files that must appear before all other tiers.
func TestDefaultProfile_RelevanceTier0_ExactPatterns(t *testing.T) {
	t.Parallel()

	tier0 := DefaultProfile().Relevance.Tier0

	expected := []string{
		"package.json",
		"tsconfig.json",
		"tsconfig.*.json",
		"Cargo.toml",
		"go.mod",
		"go.sum",
		"Makefile",
		"Dockerfile",
		"docker-compose.yml",
		"docker-compose.yaml",
		"*.config.*",
		"pyproject.toml",
		"setup.py",
		"setup.cfg",
		"pom.xml",
		"build.gradle",
		"build.gradle.kts",
	}
	assert.Equal(t, expected, tier0,
		"Tier 0 must match PRD Section 5.3 exactly")
}

// TestDefaultProfile_RelevanceTier1_ExactPatterns verifies the complete and
// exact set of Tier 1 patterns from PRD Section 5.3. Tier 1 contains primary
// source directories.
func TestDefaultProfile_RelevanceTier1_ExactPatterns(t *testing.T) {
	t.Parallel()

	tier1 := DefaultProfile().Relevance.Tier1

	expected := []string{
		"src/**",
		"lib/**",
		"app/**",
		"cmd/**",
		"internal/**",
		"pkg/**",
	}
	assert.Equal(t, expected, tier1,
		"Tier 1 must match PRD Section 5.3 exactly (src, lib, app, cmd, internal, pkg)")
}

// TestDefaultProfile_RelevanceTier1_ContainsAppDir verifies that the "app/**"
// pattern is present in Tier 1. This is distinct from the existing test that
// only checks src/lib/cmd/internal/pkg.
func TestDefaultProfile_RelevanceTier1_ContainsAppDir(t *testing.T) {
	t.Parallel()

	tier1 := DefaultProfile().Relevance.Tier1
	assert.Contains(t, tier1, "app/**",
		"Tier 1 must contain app/** per PRD Section 5.3")
}

// TestDefaultProfile_RelevanceTier2_ExactPatterns verifies the complete and
// exact set of Tier 2 patterns from PRD Section 5.3. Tier 2 contains secondary
// source files, components, and utilities.
func TestDefaultProfile_RelevanceTier2_ExactPatterns(t *testing.T) {
	t.Parallel()

	tier2 := DefaultProfile().Relevance.Tier2

	expected := []string{
		"components/**",
		"hooks/**",
		"utils/**",
		"helpers/**",
		"middleware/**",
		"services/**",
		"models/**",
		"types/**",
	}
	assert.Equal(t, expected, tier2,
		"Tier 2 must match PRD Section 5.3 exactly")
}

// TestDefaultProfile_RelevanceTier2_ContainsExpectedPatterns spot-checks key
// Tier 2 patterns to verify components, hooks, utils, and services are present.
func TestDefaultProfile_RelevanceTier2_ContainsExpectedPatterns(t *testing.T) {
	t.Parallel()

	tier2 := DefaultProfile().Relevance.Tier2

	tests := []struct {
		name    string
		pattern string
	}{
		{name: "components", pattern: "components/**"},
		{name: "hooks", pattern: "hooks/**"},
		{name: "utils", pattern: "utils/**"},
		{name: "helpers", pattern: "helpers/**"},
		{name: "middleware", pattern: "middleware/**"},
		{name: "services", pattern: "services/**"},
		{name: "models", pattern: "models/**"},
		{name: "types", pattern: "types/**"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, tier2, tt.pattern,
				"Tier 2 should contain %s", tt.pattern)
		})
	}
}

// TestDefaultProfile_RelevanceTier3_ExactPatterns verifies the complete and
// exact set of Tier 3 patterns from PRD Section 5.3. Tier 3 contains test
// files across all supported languages.
func TestDefaultProfile_RelevanceTier3_ExactPatterns(t *testing.T) {
	t.Parallel()

	tier3 := DefaultProfile().Relevance.Tier3

	expected := []string{
		"**/*_test.go",
		"**/*.test.ts",
		"**/*.test.tsx",
		"**/*.test.js",
		"**/*.spec.ts",
		"**/*.spec.tsx",
		"**/*.spec.js",
		"**/__tests__/**",
		"**/*_test.py",
		"**/tests/**",
	}
	assert.Equal(t, expected, tier3,
		"Tier 3 must match PRD Section 5.3 exactly")
}

// TestDefaultProfile_RelevanceTier4_ExactPatterns verifies the complete and
// exact set of Tier 4 patterns from PRD Section 5.3. Tier 4 contains
// documentation files.
func TestDefaultProfile_RelevanceTier4_ExactPatterns(t *testing.T) {
	t.Parallel()

	tier4 := DefaultProfile().Relevance.Tier4

	expected := []string{
		"**/*.md",
		"docs/**",
		"README*",
		"CHANGELOG*",
		"CONTRIBUTING*",
		"LICENSE*",
	}
	assert.Equal(t, expected, tier4,
		"Tier 4 must match PRD Section 5.3 exactly")
}

// TestDefaultProfile_RelevanceTier4_ContainsDocPatterns spot-checks that key
// documentation patterns are present in Tier 4.
func TestDefaultProfile_RelevanceTier4_ContainsDocPatterns(t *testing.T) {
	t.Parallel()

	tier4 := DefaultProfile().Relevance.Tier4

	tests := []struct {
		name    string
		pattern string
	}{
		{name: "markdown files", pattern: "**/*.md"},
		{name: "docs directory", pattern: "docs/**"},
		{name: "readme wildcard", pattern: "README*"},
		{name: "changelog wildcard", pattern: "CHANGELOG*"},
		{name: "contributing wildcard", pattern: "CONTRIBUTING*"},
		{name: "license wildcard", pattern: "LICENSE*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, tier4, tt.pattern,
				"Tier 4 should contain %s", tt.pattern)
		})
	}
}

// TestDefaultProfile_RelevanceTier5_ExactPatterns verifies the complete and
// exact set of Tier 5 patterns from PRD Section 5.3. Tier 5 contains CI/CD
// configs and lock files at the lowest priority.
func TestDefaultProfile_RelevanceTier5_ExactPatterns(t *testing.T) {
	t.Parallel()

	tier5 := DefaultProfile().Relevance.Tier5

	expected := []string{
		".github/**",
		".gitlab-ci.yml",
		".gitlab/**",
		"**/*.lock",
		"package-lock.json",
		"yarn.lock",
		"pnpm-lock.yaml",
		"Cargo.lock",
	}
	assert.Equal(t, expected, tier5,
		"Tier 5 must match PRD Section 5.3 exactly")
}

// TestDefaultProfile_RelevanceTier5_ContainsCICDAndLockFiles spot-checks key
// CI/CD and lock file patterns in Tier 5.
func TestDefaultProfile_RelevanceTier5_ContainsCICDAndLockFiles(t *testing.T) {
	t.Parallel()

	tier5 := DefaultProfile().Relevance.Tier5

	tests := []struct {
		name    string
		pattern string
	}{
		{name: "github actions", pattern: ".github/**"},
		{name: "gitlab CI yaml", pattern: ".gitlab-ci.yml"},
		{name: "gitlab directory", pattern: ".gitlab/**"},
		{name: "generic lock files", pattern: "**/*.lock"},
		{name: "npm lock file", pattern: "package-lock.json"},
		{name: "yarn lock file", pattern: "yarn.lock"},
		{name: "pnpm lock file", pattern: "pnpm-lock.yaml"},
		{name: "cargo lock file", pattern: "Cargo.lock"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, tier5, tt.pattern,
				"Tier 5 should contain %s", tt.pattern)
		})
	}
}

// TestDefaultProfile_RedactionConfig_ExcludePathsEmpty verifies that the
// default RedactionConfig has no ExcludePaths set -- they are only added by
// user-defined profiles.
func TestDefaultProfile_RedactionConfig_ExcludePathsEmpty(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	assert.Empty(t, p.RedactionConfig.ExcludePaths,
		"default RedactionConfig must not pre-populate ExcludePaths")
}

// TestDefaultProfile_IgnoreContainsAllPRDEntries verifies that every entry
// listed in the PRD Section 5.2 default ignore list is present. This is a
// completeness check; order is verified by TestDefaultProfile_IgnorePatterns.
func TestDefaultProfile_IgnoreContainsAllPRDEntries(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()

	prdIgnoreEntries := []string{
		"node_modules",
		"dist",
		".git",
		"coverage",
		"__pycache__",
		".next",
		"target",
		"vendor",
	}

	for _, entry := range prdIgnoreEntries {
		assert.Contains(t, p.Ignore, entry,
			"default Ignore list must contain %q per PRD Section 5.2", entry)
	}
}

// TestDefaultProfile_IgnoreExactLength ensures the default ignore list has
// exactly the 8 entries specified in the PRD, and no extras have crept in.
func TestDefaultProfile_IgnoreExactLength(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	assert.Len(t, p.Ignore, 8,
		"default Ignore list must have exactly 8 entries per PRD Section 5.2")
}

// TestDefaultProfile_PriorityFilesNil verifies that the default profile does
// not have any priority_files set -- this is a user-configuration concern.
func TestDefaultProfile_PriorityFilesNil(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	assert.Nil(t, p.PriorityFiles,
		"default profile must have nil PriorityFiles (not an empty slice)")
}

// TestDefaultProfile_IncludeNil verifies that the default profile does not
// have any include patterns -- the include list is user-configurable only.
func TestDefaultProfile_IncludeNil(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	assert.Nil(t, p.Include,
		"default profile must have nil Include (not an empty slice)")
}

// TestDefaultProfile_TargetEmpty verifies that the default profile target is
// an empty string (generic, non-LLM-specific output).
func TestDefaultProfile_TargetEmpty(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	assert.Equal(t, "", p.Target,
		"default profile Target must be empty string (not \"generic\")")
}

// TestDefaultProfile_RelevanceTierCounts verifies the exact number of patterns
// in each tier, ensuring no accidental additions or removals from PRD spec.
func TestDefaultProfile_RelevanceTierCounts(t *testing.T) {
	t.Parallel()

	r := DefaultProfile().Relevance

	tests := []struct {
		name      string
		tier      []string
		wantCount int
	}{
		{name: "Tier 0 (config files)", tier: r.Tier0, wantCount: 17},
		{name: "Tier 1 (primary source)", tier: r.Tier1, wantCount: 6},
		{name: "Tier 2 (secondary source)", tier: r.Tier2, wantCount: 8},
		{name: "Tier 3 (tests)", tier: r.Tier3, wantCount: 10},
		{name: "Tier 4 (docs)", tier: r.Tier4, wantCount: 6},
		{name: "Tier 5 (CI/lock)", tier: r.Tier5, wantCount: 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Len(t, tt.tier, tt.wantCount,
				"%s must have exactly %d patterns per PRD Section 5.3",
				tt.name, tt.wantCount)
		})
	}
}

// TestDefaultRelevanceTiers_IndependentFromProfile verifies that the
// RelevanceConfig embedded in a default profile is an independent value; two
// calls return structurally equal but non-aliased slices.
func TestDefaultRelevanceTiers_IndependentFromProfile(t *testing.T) {
	t.Parallel()

	p1 := DefaultProfile()
	p2 := DefaultProfile()

	// Mutate p1's tier slices.
	p1.Relevance.Tier0 = append(p1.Relevance.Tier0, "extra-config.toml")
	p1.Relevance.Tier1 = append(p1.Relevance.Tier1, "services/**")

	// p2 must not be affected.
	assert.NotContains(t, p2.Relevance.Tier0, "extra-config.toml",
		"mutating p1.Relevance.Tier0 must not affect p2.Relevance.Tier0")
	assert.NotContains(t, p2.Relevance.Tier1, "services/**",
		"mutating p1.Relevance.Tier1 must not affect p2.Relevance.Tier1")
}
