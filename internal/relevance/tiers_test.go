package relevance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTierConstants verifies each Tier constant has the correct integer value.
func TestTierConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tier      Tier
		wantValue int
	}{
		{name: "Tier0Critical", tier: Tier0Critical, wantValue: 0},
		{name: "Tier1Primary", tier: Tier1Primary, wantValue: 1},
		{name: "Tier2Secondary", tier: Tier2Secondary, wantValue: 2},
		{name: "Tier3Tests", tier: Tier3Tests, wantValue: 3},
		{name: "Tier4Docs", tier: Tier4Docs, wantValue: 4},
		{name: "Tier5Low", tier: Tier5Low, wantValue: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantValue, int(tt.tier))
		})
	}
}

// TestTierOrdering verifies the strict ordering Tier0Critical < Tier1Primary < ... < Tier5Low.
func TestTierOrdering(t *testing.T) {
	t.Parallel()

	assert.Less(t, int(Tier0Critical), int(Tier1Primary), "Tier0Critical must be less than Tier1Primary")
	assert.Less(t, int(Tier1Primary), int(Tier2Secondary), "Tier1Primary must be less than Tier2Secondary")
	assert.Less(t, int(Tier2Secondary), int(Tier3Tests), "Tier2Secondary must be less than Tier3Tests")
	assert.Less(t, int(Tier3Tests), int(Tier4Docs), "Tier3Tests must be less than Tier4Docs")
	assert.Less(t, int(Tier4Docs), int(Tier5Low), "Tier4Docs must be less than Tier5Low")
}

// TestTierString verifies String() returns the expected label for all 6 known tiers
// and falls back to "tier%d" for unknown values.
func TestTierString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tier Tier
		want string
	}{
		{name: "Tier0Critical", tier: Tier0Critical, want: "critical"},
		{name: "Tier1Primary", tier: Tier1Primary, want: "primary"},
		{name: "Tier2Secondary", tier: Tier2Secondary, want: "secondary"},
		{name: "Tier3Tests", tier: Tier3Tests, want: "tests"},
		{name: "Tier4Docs", tier: Tier4Docs, want: "docs"},
		{name: "Tier5Low", tier: Tier5Low, want: "low"},
		{name: "unknown positive", tier: Tier(99), want: "tier99"},
		{name: "unknown negative", tier: Tier(-1), want: "tier-1"},
		{name: "unknown large", tier: Tier(100), want: "tier100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.tier.String())
		})
	}
}

// TestDefaultUnmatchedTier verifies DefaultUnmatchedTier equals Tier2Secondary (value 2).
func TestDefaultUnmatchedTier(t *testing.T) {
	t.Parallel()

	assert.Equal(t, Tier2Secondary, DefaultUnmatchedTier)
	assert.Equal(t, 2, int(DefaultUnmatchedTier))
}

// TestDefaultTierDefinitionsCount verifies DefaultTierDefinitions returns exactly 6 entries.
func TestDefaultTierDefinitionsCount(t *testing.T) {
	t.Parallel()

	defs := DefaultTierDefinitions()
	require.Len(t, defs, 6, "DefaultTierDefinitions must return exactly 6 entries")
}

// TestDefaultTierDefinitionsCoverage verifies each tier 0-5 appears exactly once.
func TestDefaultTierDefinitionsCoverage(t *testing.T) {
	t.Parallel()

	defs := DefaultTierDefinitions()

	counts := make(map[Tier]int, 6)
	for _, d := range defs {
		counts[d.Tier]++
	}

	tiers := []Tier{
		Tier0Critical,
		Tier1Primary,
		Tier2Secondary,
		Tier3Tests,
		Tier4Docs,
		Tier5Low,
	}

	for _, tier := range tiers {
		t.Run(tier.String(), func(t *testing.T) {
			assert.Equal(t, 1, counts[tier], "tier %s must appear exactly once", tier)
		})
	}
}

// TestDefaultTierDefinitionsPatterns verifies specific required patterns are present
// in their designated tiers and absent where they must not appear.
func TestDefaultTierDefinitionsPatterns(t *testing.T) {
	t.Parallel()

	defs := DefaultTierDefinitions()

	// Build a lookup: tier -> set of patterns for O(1) membership checks.
	patternsByTier := make(map[Tier]map[string]struct{}, len(defs))
	for _, d := range defs {
		set := make(map[string]struct{}, len(d.Patterns))
		for _, p := range d.Patterns {
			set[p] = struct{}{}
		}
		patternsByTier[d.Tier] = set
	}

	hasPattern := func(t *testing.T, tier Tier, pattern string) {
		t.Helper()
		set, ok := patternsByTier[tier]
		require.True(t, ok, "tier %s not found in DefaultTierDefinitions", tier)
		assert.Contains(t, set, pattern, "tier %s must contain pattern %q", tier, pattern)
	}

	lacksPattern := func(t *testing.T, tier Tier, pattern string) {
		t.Helper()
		set, ok := patternsByTier[tier]
		require.True(t, ok, "tier %s not found in DefaultTierDefinitions", tier)
		assert.NotContains(t, set, pattern, "tier %s must NOT contain pattern %q", tier, pattern)
	}

	t.Run("Tier0Critical patterns", func(t *testing.T) {
		t.Parallel()
		hasPattern(t, Tier0Critical, "go.mod")
		hasPattern(t, Tier0Critical, "Makefile")
		hasPattern(t, Tier0Critical, "Dockerfile")
	})

	t.Run("Tier1Primary patterns", func(t *testing.T) {
		t.Parallel()
		hasPattern(t, Tier1Primary, "src/**")
		hasPattern(t, Tier1Primary, "internal/**")
		hasPattern(t, Tier1Primary, "cmd/**")
	})

	t.Run("Tier3Tests patterns", func(t *testing.T) {
		t.Parallel()
		hasPattern(t, Tier3Tests, "*_test.go")
		hasPattern(t, Tier3Tests, "__tests__/**")
	})

	t.Run("Tier4Docs patterns", func(t *testing.T) {
		t.Parallel()
		hasPattern(t, Tier4Docs, "*.md")
		hasPattern(t, Tier4Docs, "docs/**")
	})

	t.Run("Tier5Low patterns", func(t *testing.T) {
		t.Parallel()
		hasPattern(t, Tier5Low, ".github/**")
		hasPattern(t, Tier5Low, "*.lock")
		hasPattern(t, Tier5Low, "go.sum")
	})

	t.Run("go.sum absent from Tier0Critical", func(t *testing.T) {
		t.Parallel()
		lacksPattern(t, Tier0Critical, "go.sum")
	})
}

// TestTierDefinitionConstructability verifies a custom TierDefinition can be created
// with arbitrary patterns and that all fields are correctly set.
func TestTierDefinitionConstructability(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		tier         Tier
		patterns     []string
	}{
		{
			name:     "custom critical tier",
			tier:     Tier0Critical,
			patterns: []string{"custom.json", "custom/**"},
		},
		{
			name:     "custom tier with single pattern",
			tier:     Tier3Tests,
			patterns: []string{"e2e/**"},
		},
		{
			name:     "custom tier with empty patterns",
			tier:     Tier5Low,
			patterns: []string{},
		},
		{
			name:     "unknown tier value",
			tier:     Tier(42),
			patterns: []string{"*.wasm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def := TierDefinition{
				Tier:     tt.tier,
				Patterns: tt.patterns,
			}

			assert.Equal(t, tt.tier, def.Tier)
			assert.Equal(t, tt.patterns, def.Patterns)
		})
	}
}

// TestDefaultTierDefinitionsImmutability verifies that mutating the slice returned
// by one call to DefaultTierDefinitions does not affect a subsequent call.
func TestDefaultTierDefinitionsImmutability(t *testing.T) {
	t.Parallel()

	first := DefaultTierDefinitions()
	second := DefaultTierDefinitions()

	// Record the original length of patterns in the first definition.
	originalLen := len(first[0].Patterns)

	// Mutate the first slice: append a spurious pattern to the first entry.
	first[0].Patterns = append(first[0].Patterns, "__MUTATION_SENTINEL__")

	// The second slice must be unaffected.
	assert.Len(t, second[0].Patterns, originalLen,
		"mutating the first call's result must not affect a subsequent call")

	// The first slice element must be a different underlying array from the second.
	assert.NotEqual(t, len(first[0].Patterns), len(second[0].Patterns),
		"slices must be independent copies")
}

// TestDefaultTierDefinitionsNonEmptyPatterns verifies every tier in the defaults
// has at least one pattern defined.
func TestDefaultTierDefinitionsNonEmptyPatterns(t *testing.T) {
	t.Parallel()

	defs := DefaultTierDefinitions()
	for _, d := range defs {
		t.Run(d.Tier.String(), func(t *testing.T) {
			assert.NotEmpty(t, d.Patterns, "tier %s must have at least one pattern", d.Tier)
		})
	}
}
