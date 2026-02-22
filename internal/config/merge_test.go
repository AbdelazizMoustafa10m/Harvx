package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── mergeString ───────────────────────────────────────────────────────────────

func TestMergeString_OverrideNonEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "xml", mergeString("markdown", "xml"))
}

func TestMergeString_OverrideEmpty_KeepsBase(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "markdown", mergeString("markdown", ""))
}

func TestMergeString_BothEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", mergeString("", ""))
}

func TestMergeString_BaseEmpty_OverrideNonEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "xml", mergeString("", "xml"))
}

// ── mergeInt ─────────────────────────────────────────────────────────────────

func TestMergeInt_OverrideNonZero(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 64000, mergeInt(128000, 64000))
}

func TestMergeInt_OverrideZero_KeepsBase(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 128000, mergeInt(128000, 0))
}

func TestMergeInt_BothZero(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, mergeInt(0, 0))
}

func TestMergeInt_BaseZero_OverrideNonZero(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 200000, mergeInt(0, 200000))
}

// ── mergeSlice ────────────────────────────────────────────────────────────────

func TestMergeSlice_OverrideNonEmpty_ReplacesBase(t *testing.T) {
	t.Parallel()
	base := []string{"node_modules", "dist"}
	override := []string{"reports/", ".review-workspace/"}
	result := mergeSlice(base, override)
	assert.Equal(t, []string{"reports/", ".review-workspace/"}, result)
}

func TestMergeSlice_OverrideNil_KeepsBase(t *testing.T) {
	t.Parallel()
	base := []string{"node_modules", "dist"}
	result := mergeSlice(base, nil)
	assert.Equal(t, []string{"node_modules", "dist"}, result)
}

func TestMergeSlice_OverrideEmpty_KeepsBase(t *testing.T) {
	t.Parallel()
	base := []string{"node_modules", "dist"}
	result := mergeSlice(base, []string{})
	assert.Equal(t, []string{"node_modules", "dist"}, result)
}

func TestMergeSlice_BothNil_ReturnsNil(t *testing.T) {
	t.Parallel()
	result := mergeSlice(nil, nil)
	assert.Nil(t, result)
}

func TestMergeSlice_BaseNil_OverrideNonEmpty(t *testing.T) {
	t.Parallel()
	override := []string{"a", "b"}
	result := mergeSlice(nil, override)
	assert.Equal(t, []string{"a", "b"}, result)
}

// TestMergeSlice_ReturnsCopy verifies that the returned slice does not share
// the backing array with the input slices (DC-1: no aliasing).
func TestMergeSlice_ReturnsCopy(t *testing.T) {
	t.Parallel()
	base := []string{"a", "b"}
	override := []string{"c", "d"}

	result := mergeSlice(base, override)
	// Mutate result; override must not be affected.
	result[0] = "mutated"
	assert.Equal(t, "c", override[0], "mutating result must not affect override")

	result2 := mergeSlice(base, nil)
	// Mutate result2; base must not be affected.
	result2[0] = "mutated"
	assert.Equal(t, "a", base[0], "mutating result2 must not affect base")
}

// ── mergeRelevance ────────────────────────────────────────────────────────────

func TestMergeRelevance_OverrideTierReplacesBase(t *testing.T) {
	t.Parallel()
	base := RelevanceConfig{
		Tier0: []string{"go.mod", "package.json"},
		Tier1: []string{"src/**"},
		Tier2: []string{"utils/**"},
	}
	override := RelevanceConfig{
		Tier0: []string{"CLAUDE.md", "*.config.*"},
		// Tier1 and Tier2 not set -- should be inherited
	}

	result := mergeRelevance(base, override)

	assert.Equal(t, []string{"CLAUDE.md", "*.config.*"}, result.Tier0,
		"non-empty override tier must replace base")
	assert.Equal(t, []string{"src/**"}, result.Tier1,
		"unset override tier must inherit base")
	assert.Equal(t, []string{"utils/**"}, result.Tier2,
		"unset override tier must inherit base")
}

func TestMergeRelevance_AllTiersOverridden(t *testing.T) {
	t.Parallel()
	base := RelevanceConfig{
		Tier0: []string{"go.mod"},
		Tier1: []string{"src/**"},
		Tier2: []string{"utils/**"},
		Tier3: []string{"**/*_test.go"},
		Tier4: []string{"docs/**"},
		Tier5: []string{".github/**"},
	}
	override := RelevanceConfig{
		Tier0: []string{"CLAUDE.md"},
		Tier1: []string{"app/**"},
		Tier2: []string{"components/**"},
		Tier3: []string{"__tests__/**"},
		Tier4: []string{"docs/**", "prompts/**"},
		Tier5: []string{"*.lock"},
	}

	result := mergeRelevance(base, override)

	assert.Equal(t, []string{"CLAUDE.md"}, result.Tier0)
	assert.Equal(t, []string{"app/**"}, result.Tier1)
	assert.Equal(t, []string{"components/**"}, result.Tier2)
	assert.Equal(t, []string{"__tests__/**"}, result.Tier3)
	assert.Equal(t, []string{"docs/**", "prompts/**"}, result.Tier4)
	assert.Equal(t, []string{"*.lock"}, result.Tier5)
}

func TestMergeRelevance_EmptyOverride_KeepsBase(t *testing.T) {
	t.Parallel()
	base := RelevanceConfig{
		Tier0: []string{"go.mod"},
		Tier1: []string{"src/**"},
	}
	override := RelevanceConfig{}

	result := mergeRelevance(base, override)

	assert.Equal(t, []string{"go.mod"}, result.Tier0)
	assert.Equal(t, []string{"src/**"}, result.Tier1)
}

// ── mergeRedactionConfig ──────────────────────────────────────────────────────

// TestMergeRedactionConfig_EnabledFalseWins verifies that override.Enabled=false
// always wins over base.Enabled=true (bool always takes override value).
func TestMergeRedactionConfig_EnabledFalseWins(t *testing.T) {
	t.Parallel()
	base := RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: "high",
		ExcludePaths:        []string{"docs/**"},
	}
	override := RedactionConfig{
		Enabled: false,
		// ExcludePaths not set -- should preserve base's
	}

	result := mergeRedactionConfig(base, override)

	assert.False(t, result.Enabled,
		"override Enabled=false must win over base Enabled=true")
	assert.Equal(t, "high", result.ConfidenceThreshold,
		"unset ConfidenceThreshold must be inherited from base")
	assert.Equal(t, []string{"docs/**"}, result.ExcludePaths,
		"unset ExcludePaths must be inherited from base (T-019 req 6)")
}

// TestMergeRedactionConfig_EnabledTrueWinsOverFalse verifies that
// override.Enabled=true wins over base.Enabled=false.
func TestMergeRedactionConfig_EnabledTrueWinsOverFalse(t *testing.T) {
	t.Parallel()
	base := RedactionConfig{Enabled: false}
	override := RedactionConfig{Enabled: true}

	result := mergeRedactionConfig(base, override)

	assert.True(t, result.Enabled)
}

// TestMergeRedactionConfig_ExcludePathsReplaceBase verifies that a non-empty
// override ExcludePaths completely replaces the base slice.
func TestMergeRedactionConfig_ExcludePathsReplaceBase(t *testing.T) {
	t.Parallel()
	base := RedactionConfig{
		Enabled:      true,
		ExcludePaths: []string{"docs/**"},
	}
	override := RedactionConfig{
		// Enabled not explicitly set (false zero val) -- override wins
		ExcludePaths: []string{"tests/**", "fixtures/**"},
	}

	result := mergeRedactionConfig(base, override)

	assert.Equal(t, []string{"tests/**", "fixtures/**"}, result.ExcludePaths,
		"override ExcludePaths must replace base entirely (T-019 req 7)")
	// bool: override value wins (false)
	assert.False(t, result.Enabled)
}

// TestMergeRedactionConfig_EmptyOverrideExcludePaths_KeepsBase verifies that
// when override sets no ExcludePaths the base paths are preserved.
func TestMergeRedactionConfig_EmptyOverrideExcludePaths_KeepsBase(t *testing.T) {
	t.Parallel()
	base := RedactionConfig{
		Enabled:      true,
		ExcludePaths: []string{"fixtures/**"},
	}
	override := RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: "medium",
		// ExcludePaths not set
	}

	result := mergeRedactionConfig(base, override)

	assert.Equal(t, []string{"fixtures/**"}, result.ExcludePaths,
		"base ExcludePaths must be preserved when override is empty")
	assert.Equal(t, "medium", result.ConfidenceThreshold)
}

func TestMergeRedactionConfig_ConfidenceThreshold_OverrideWins(t *testing.T) {
	t.Parallel()
	base := RedactionConfig{ConfidenceThreshold: "high"}
	override := RedactionConfig{ConfidenceThreshold: "low"}

	result := mergeRedactionConfig(base, override)

	assert.Equal(t, "low", result.ConfidenceThreshold)
}

func TestMergeRedactionConfig_ConfidenceThreshold_EmptyOverride_KeepsBase(t *testing.T) {
	t.Parallel()
	base := RedactionConfig{ConfidenceThreshold: "high"}
	override := RedactionConfig{ConfidenceThreshold: ""}

	result := mergeRedactionConfig(base, override)

	assert.Equal(t, "high", result.ConfidenceThreshold)
}

// ── mergeProfile ─────────────────────────────────────────────────────────────

// TestMergeProfile_StringScalars verifies that non-empty override string fields
// replace base, and empty override fields fall back to base.
func TestMergeProfile_StringScalars(t *testing.T) {
	t.Parallel()
	base := &Profile{
		Output:    "harvx-output.md",
		Format:    "markdown",
		Tokenizer: "cl100k_base",
		Target:    "generic",
	}
	override := &Profile{
		Format: "xml",
		// Output, Tokenizer, Target not set -- fall back to base
	}

	result := mergeProfile(base, override)

	assert.Equal(t, "harvx-output.md", result.Output, "unset Output must inherit base")
	assert.Equal(t, "xml", result.Format, "set Format must override base")
	assert.Equal(t, "cl100k_base", result.Tokenizer, "unset Tokenizer must inherit base")
	assert.Equal(t, "generic", result.Target, "unset Target must inherit base")
}

// TestMergeProfile_IntScalar verifies that a non-zero override MaxTokens
// replaces the base value, and a zero override keeps the base value.
func TestMergeProfile_IntScalar(t *testing.T) {
	t.Parallel()
	base := &Profile{MaxTokens: 128000}
	overrideNonZero := &Profile{MaxTokens: 64000}
	overrideZero := &Profile{MaxTokens: 0}

	assert.Equal(t, 64000, mergeProfile(base, overrideNonZero).MaxTokens,
		"non-zero override must win")
	assert.Equal(t, 128000, mergeProfile(base, overrideZero).MaxTokens,
		"zero override must fall back to base")
}

// TestMergeProfile_BoolScalars verifies that bool fields always take the
// override value (false is a valid explicit override).
func TestMergeProfile_BoolScalars(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		baseCompression bool
		baseRedaction   bool
		ovCompression   bool
		ovRedaction     bool
	}{
		{
			name:            "false overrides true",
			baseCompression: true, baseRedaction: true,
			ovCompression: false, ovRedaction: false,
		},
		{
			name:            "true overrides false",
			baseCompression: false, baseRedaction: false,
			ovCompression: true, ovRedaction: true,
		},
		{
			name:            "false keeps false",
			baseCompression: false, baseRedaction: false,
			ovCompression: false, ovRedaction: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := &Profile{Compression: tt.baseCompression, Redaction: tt.baseRedaction}
			override := &Profile{Compression: tt.ovCompression, Redaction: tt.ovRedaction}
			result := mergeProfile(base, override)
			assert.Equal(t, tt.ovCompression, result.Compression, "Compression")
			assert.Equal(t, tt.ovRedaction, result.Redaction, "Redaction")
		})
	}
}

// TestMergeProfile_ExtendsAlwaysCleared verifies that mergeProfile always
// returns a profile with Extends == nil regardless of inputs.
func TestMergeProfile_ExtendsAlwaysCleared(t *testing.T) {
	t.Parallel()
	base := &Profile{Extends: strPtr("grandparent")}
	override := &Profile{Extends: strPtr("parent")}

	result := mergeProfile(base, override)

	assert.Nil(t, result.Extends, "merged profile Extends must always be nil")
}

// TestMergeProfile_DoesNotMutateInputs verifies that neither base nor override
// is modified by mergeProfile.
func TestMergeProfile_DoesNotMutateInputs(t *testing.T) {
	t.Parallel()
	base := &Profile{
		Format:    "markdown",
		Ignore:    []string{"node_modules"},
		Extends:   strPtr("root"),
		MaxTokens: 128000,
	}
	override := &Profile{
		Format:    "xml",
		Ignore:    []string{"dist"},
		Extends:   strPtr("default"),
		MaxTokens: 64000,
	}

	_ = mergeProfile(base, override)

	// base must not be mutated
	assert.Equal(t, "markdown", base.Format)
	assert.Equal(t, []string{"node_modules"}, base.Ignore)
	assert.Equal(t, "root", *base.Extends)
	assert.Equal(t, 128000, base.MaxTokens)

	// override must not be mutated
	assert.Equal(t, "xml", override.Format)
	assert.Equal(t, []string{"dist"}, override.Ignore)
	assert.Equal(t, "default", *override.Extends)
	assert.Equal(t, 64000, override.MaxTokens)
}

// TestMergeProfile_FullMerge exercises all fields together to confirm the
// correct merge rules apply end-to-end.
func TestMergeProfile_FullMerge(t *testing.T) {
	t.Parallel()

	base := &Profile{
		Output:      "harvx-output.md",
		Format:      "markdown",
		MaxTokens:   128000,
		Tokenizer:   "cl100k_base",
		Compression: false,
		Redaction:   true,
		Target:      "generic",
		Ignore:      []string{"node_modules", "dist"},
		PriorityFiles: []string{"README.md"},
		Include:     []string{"**/*.go"},
		Relevance: RelevanceConfig{
			Tier0: []string{"go.mod"},
			Tier1: []string{"src/**"},
		},
		RedactionConfig: RedactionConfig{
			Enabled:             true,
			ConfidenceThreshold: "high",
			ExcludePaths:        []string{"docs/**"},
		},
	}
	override := &Profile{
		Output:        ".harvx/finvault-context.md",
		MaxTokens:     200000,
		Tokenizer:     "o200k_base",
		Compression:   true,
		Target:        "claude",
		Ignore:        []string{"reports/", ".review-workspace/"},
		PriorityFiles: []string{"CLAUDE.md", "prisma/schema.prisma"},
		Relevance: RelevanceConfig{
			Tier0: []string{"CLAUDE.md", "prisma/schema.prisma"},
			Tier1: []string{"app/api/**", "lib/services/**"},
		},
		RedactionConfig: RedactionConfig{
			Enabled:             true,
			ConfidenceThreshold: "high",
			ExcludePaths:        []string{"**/*test*/**", "**/fixtures/**"},
		},
	}

	result := mergeProfile(base, override)

	// override wins for all set string scalars
	assert.Equal(t, ".harvx/finvault-context.md", result.Output)
	assert.Equal(t, "o200k_base", result.Tokenizer)
	assert.Equal(t, "claude", result.Target)
	// Format was not set in override -- base wins
	assert.Equal(t, "markdown", result.Format)
	// int: override wins
	assert.Equal(t, 200000, result.MaxTokens)
	// bools: override always wins
	assert.True(t, result.Compression)
	assert.False(t, result.Redaction) // override zero val (false) wins
	// slices: override replaces entirely
	assert.Equal(t, []string{"reports/", ".review-workspace/"}, result.Ignore)
	assert.Equal(t, []string{"CLAUDE.md", "prisma/schema.prisma"}, result.PriorityFiles)
	// Include was not set in override -- base wins
	assert.Equal(t, []string{"**/*.go"}, result.Include)
	// relevance tiers: override tiers replace base
	assert.Equal(t, []string{"CLAUDE.md", "prisma/schema.prisma"}, result.Relevance.Tier0)
	assert.Equal(t, []string{"app/api/**", "lib/services/**"}, result.Relevance.Tier1)
	// redaction config: field-by-field
	assert.True(t, result.RedactionConfig.Enabled)
	assert.Equal(t, "high", result.RedactionConfig.ConfidenceThreshold)
	assert.Equal(t, []string{"**/*test*/**", "**/fixtures/**"}, result.RedactionConfig.ExcludePaths)
	// Extends must always be cleared
	assert.Nil(t, result.Extends)
}
