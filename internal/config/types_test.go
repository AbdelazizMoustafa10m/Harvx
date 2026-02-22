package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultProfile_Values verifies that DefaultProfile returns a profile
// matching the PRD Section 5.2 specification exactly.
func TestDefaultProfile_Values(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	require.NotNil(t, p)

	assert.Equal(t, "harvx-output.md", p.Output)
	assert.Equal(t, "markdown", p.Format)
	assert.Equal(t, 128000, p.MaxTokens)
	assert.Equal(t, "cl100k_base", p.Tokenizer)
	assert.False(t, p.Compression)
	assert.True(t, p.Redaction)
	assert.Equal(t, "", p.Target)
	assert.Nil(t, p.Extends)
}

// TestDefaultProfile_IgnorePatterns verifies the built-in ignore list.
func TestDefaultProfile_IgnorePatterns(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()

	expected := []string{
		"node_modules",
		"dist",
		".git",
		"coverage",
		"__pycache__",
		".next",
		"target",
		"vendor",
	}
	assert.Equal(t, expected, p.Ignore)
}

// TestDefaultProfile_IsFreshCopy verifies that each call returns an independent
// copy so mutations in one caller do not affect others.
func TestDefaultProfile_IsFreshCopy(t *testing.T) {
	t.Parallel()

	p1 := DefaultProfile()
	p2 := DefaultProfile()

	p1.Output = "mutated.md"
	p1.Ignore = append(p1.Ignore, "extra")

	assert.Equal(t, "harvx-output.md", p2.Output, "mutation of p1 must not affect p2")
	assert.NotContains(t, p2.Ignore, "extra", "slice mutation must not affect p2")
}

// TestDefaultProfile_RelevanceTiers verifies that the default relevance config
// is populated with non-empty tier lists per PRD Section 5.3.
func TestDefaultProfile_RelevanceTiers(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	r := p.Relevance

	assert.NotEmpty(t, r.Tier0, "Tier 0 (config files) must not be empty")
	assert.NotEmpty(t, r.Tier1, "Tier 1 (primary source) must not be empty")
	assert.NotEmpty(t, r.Tier2, "Tier 2 (secondary source) must not be empty")
	assert.NotEmpty(t, r.Tier3, "Tier 3 (tests) must not be empty")
	assert.NotEmpty(t, r.Tier4, "Tier 4 (docs) must not be empty")
	assert.NotEmpty(t, r.Tier5, "Tier 5 (CI/lock files) must not be empty")
}

// TestDefaultProfile_RelevanceTier0_ContainsConfigFiles checks that well-known
// config file names appear in Tier 0.
func TestDefaultProfile_RelevanceTier0_ContainsConfigFiles(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	tier0 := p.Relevance.Tier0

	mustContain := []string{"package.json", "go.mod", "Makefile", "Dockerfile"}
	for _, name := range mustContain {
		assert.Contains(t, tier0, name, "Tier 0 should contain %s", name)
	}
}

// TestDefaultProfile_RelevanceTier1_ContainsPrimaryDirs checks that primary
// source dirs appear in Tier 1.
func TestDefaultProfile_RelevanceTier1_ContainsPrimaryDirs(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	tier1 := p.Relevance.Tier1

	mustContain := []string{"src/**", "lib/**", "cmd/**", "internal/**", "pkg/**"}
	for _, pat := range mustContain {
		assert.Contains(t, tier1, pat, "Tier 1 should contain %s", pat)
	}
}

// TestDefaultProfile_RelevanceTier3_ContainsTestPatterns checks that common
// test file patterns appear in Tier 3.
func TestDefaultProfile_RelevanceTier3_ContainsTestPatterns(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()
	tier3 := p.Relevance.Tier3

	mustContain := []string{"**/*_test.go", "**/*.test.ts", "**/*.spec.js"}
	for _, pat := range mustContain {
		assert.Contains(t, tier3, pat, "Tier 3 should contain %s", pat)
	}
}

// TestDefaultProfile_RedactionConfig verifies the built-in redaction defaults.
func TestDefaultProfile_RedactionConfig(t *testing.T) {
	t.Parallel()

	p := DefaultProfile()

	assert.True(t, p.RedactionConfig.Enabled)
	assert.Equal(t, "high", p.RedactionConfig.ConfidenceThreshold)
}

// TestConfig_ZeroValue verifies that the zero value of Config is usable
// (nil map access is handled gracefully).
func TestConfig_ZeroValue(t *testing.T) {
	t.Parallel()

	var cfg Config
	// A nil map lookup returns the zero value and does not panic.
	p := cfg.Profile["default"]
	assert.Nil(t, p)
}

// TestProfile_ExtendsPointer verifies that the Extends field behaves correctly
// as a string pointer.
func TestProfile_ExtendsPointer(t *testing.T) {
	t.Parallel()

	// nil means no inheritance.
	p := &Profile{}
	assert.Nil(t, p.Extends)

	// Non-nil means inherit from named profile.
	parent := "default"
	p.Extends = &parent
	require.NotNil(t, p.Extends)
	assert.Equal(t, "default", *p.Extends)
}
