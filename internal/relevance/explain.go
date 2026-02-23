// Package relevance — this file implements the explain and inclusion summary
// functionality (T-032).
package relevance

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/harvx/harvx/internal/tokenizer"
)

// PatternMatch records a single tier-pattern combination that matched a file.
// It is used in ExplainResult.AllMatches to expose every overlapping rule that
// would claim the file, helping users debug profile configuration conflicts.
type PatternMatch struct {
	// Tier is the tier number that contains the matching pattern.
	Tier int

	// Pattern is the glob pattern that matched the file.
	Pattern string
}

// ExplainResult holds the detailed tier-matching explanation for a single file.
// The caller is responsible for enriching WouldBeIncluded and ExclusionReason
// after budget enforcement (option 2 from the T-032 spec).
type ExplainResult struct {
	// FilePath is the queried file path (as supplied to Explain).
	FilePath string

	// AssignedTier is the tier the file was assigned to. This is the
	// lowest-numbered tier that contained a matching pattern, or
	// int(DefaultUnmatchedTier) when no pattern matched.
	AssignedTier int

	// MatchedPattern is the specific glob pattern that caused the assignment.
	// Empty when IsDefault is true.
	MatchedPattern string

	// MatchedTierDef is the index (in the original tiers slice) of the
	// TierDefinition that contained MatchedPattern. -1 when IsDefault is true.
	MatchedTierDef int

	// IsDefault is true when no pattern matched and the file was assigned the
	// DefaultUnmatchedTier (tier 2).
	IsDefault bool

	// AllMatches contains every pattern across every tier that matched this
	// file, sorted by ascending tier number then lexicographically by pattern.
	// Useful for diagnosing overlapping rules.
	AllMatches []PatternMatch

	// WouldBeIncluded is set by the caller after budget enforcement. When true
	// the file survived budget enforcement; when false it was excluded.
	// The zero value (false) means the budget context has not been applied yet.
	WouldBeIncluded bool

	// ExclusionReason is a short machine-readable string describing why the
	// file was excluded (e.g. "budget_exceeded", "filtered_by_ignore"). Empty
	// when the file is included or when the budget context has not been applied.
	ExclusionReason string

	// TokenCount is the number of tokens counted for this file. Populated by
	// the caller alongside WouldBeIncluded when budget context is available.
	TokenCount int
}

// Explain returns a detailed matching explanation for filePath against the
// provided tier definitions. It evaluates every pattern in every tier to
// collect all overlapping matches.
//
// The assigned tier is determined by first-match-wins (lowest tier number
// first, then pattern order within the tier). If no pattern matches,
// IsDefault is true and AssignedTier is int(DefaultUnmatchedTier).
//
// WouldBeIncluded and ExclusionReason are left at their zero values; the
// caller must enrich them after budget enforcement.
func Explain(filePath string, tiers []TierDefinition) *ExplainResult {
	normalised := normalisePath(filePath)

	// Sort a working copy of the definitions so we evaluate in ascending tier
	// order, which matches the first-match-wins semantic of TierMatcher.Match.
	sorted := make([]TierDefinition, len(tiers))
	copy(sorted, tiers)
	sortTierDefinitions(sorted)

	result := &ExplainResult{
		FilePath:       filePath,
		MatchedTierDef: -1,
	}

	var allMatches []PatternMatch

	for defIdx, def := range sorted {
		for _, pattern := range def.Patterns {
			// Skip syntactically invalid patterns to match TierMatcher behaviour.
			if !doublestar.ValidatePattern(pattern) {
				continue
			}

			matched, err := doublestar.Match(pattern, normalised)
			if err != nil {
				// ValidatePattern already caught bad patterns above; this
				// branch should be unreachable.
				continue
			}

			if !matched {
				continue
			}

			allMatches = append(allMatches, PatternMatch{
				Tier:    int(def.Tier),
				Pattern: pattern,
			})

			// Record the first (highest-priority) match as the assigned one.
			if result.MatchedTierDef == -1 {
				result.AssignedTier = int(def.Tier)
				result.MatchedPattern = pattern
				result.MatchedTierDef = defIdx
			}
		}
	}

	if result.MatchedTierDef == -1 {
		// No pattern matched — apply default tier.
		result.IsDefault = true
		result.AssignedTier = int(DefaultUnmatchedTier)
		result.MatchedPattern = ""
	}

	// Sort AllMatches: ascending tier, then lexicographic pattern for determinism.
	sort.Slice(allMatches, func(i, j int) bool {
		if allMatches[i].Tier != allMatches[j].Tier {
			return allMatches[i].Tier < allMatches[j].Tier
		}
		return allMatches[i].Pattern < allMatches[j].Pattern
	})

	result.AllMatches = allMatches
	return result
}

// TierLabel returns a short human-readable label for a tier number.
//
// Default mappings:
//
//	0 → "Config"
//	1 → "Source"
//	2 → "Secondary"
//	3 → "Tests"
//	4 → "Docs"
//	5 → "CI/Lock"
//
// Unknown tiers return "Tier<n>".
func TierLabel(tier int) string {
	switch tier {
	case 0:
		return "Config"
	case 1:
		return "Source"
	case 2:
		return "Secondary"
	case 3:
		return "Tests"
	case 4:
		return "Docs"
	case 5:
		return "CI/Lock"
	default:
		return fmt.Sprintf("Tier%d", tier)
	}
}

// tierDisplayLabel returns the label used in FormatExplain's "Tier:" line.
// Tier 1 uses the slightly longer "Source Code" form to match the spec example;
// all other tiers use TierLabel.
func tierDisplayLabel(tier int) string {
	if tier == 1 {
		return "Source Code"
	}
	return TierLabel(tier)
}

// FormatExplain renders an ExplainResult as a human-readable multi-line string
// suitable for terminal output. It is used by the `harvx profiles explain`
// subcommand.
//
// Example output (file matched by a pattern):
//
//	File: src/api/handler.go
//	Tier: 1 (Source Code)
//	Matched Pattern: src/** (from tier 1)
//	Budget Status: Included (tokens: 450)
//
//	All matching patterns:
//	  - Tier 1: src/**
//
// When IsDefault is true the "Matched Pattern" line reads "(default, unmatched)".
// The "Budget Status" line is only included when WouldBeIncluded is true or
// ExclusionReason is non-empty (i.e. when budget context has been applied).
func FormatExplain(result *ExplainResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "File: %s\n", result.FilePath)
	fmt.Fprintf(&b, "Tier: %d (%s)\n", result.AssignedTier, tierDisplayLabel(result.AssignedTier))

	if result.IsDefault {
		fmt.Fprintf(&b, "Matched Pattern: (default, unmatched)\n")
	} else {
		fmt.Fprintf(&b, "Matched Pattern: %s (from tier %d)\n",
			result.MatchedPattern, result.AssignedTier)
	}

	// Budget Status is only shown when the caller has enriched the result with
	// budget context (WouldBeIncluded set to true, or ExclusionReason non-empty).
	if result.WouldBeIncluded || result.ExclusionReason != "" {
		if result.WouldBeIncluded {
			fmt.Fprintf(&b, "Budget Status: Included (tokens: %d)\n", result.TokenCount)
		} else {
			fmt.Fprintf(&b, "Budget Status: Excluded (%s)\n", result.ExclusionReason)
		}
	}

	b.WriteString("\nAll matching patterns:\n")
	for _, pm := range result.AllMatches {
		fmt.Fprintf(&b, "  - Tier %d: %s\n", pm.Tier, pm.Pattern)
	}
	if result.IsDefault || len(result.AllMatches) == 0 {
		fmt.Fprintf(&b, "  - Tier %d: (default, unmatched)\n", int(DefaultUnmatchedTier))
	}

	return b.String()
}

// GenerateInclusionSummary renders a human-readable summary of a BudgetResult,
// showing per-tier file counts, token usage, and exclusion info. It is used by
// the output renderer to include a breakdown in the generated document header.
//
// When no budget was configured (BudgetUsed == 0 and no excluded files), the
// Total line omits the budget fraction. Otherwise the Total line shows tokens
// used, budget capacity, and percentage consumed.
//
// Example output:
//
//	Files: 342 included, 48 excluded
//
//	By Tier:
//	  Tier 0 (Config):      5 files,   2,100 tokens
//	  Tier 1 (Source):      48 files,  45,000 tokens
//	  Tier 2 (Secondary):  180 files,  35,000 tokens
//	  Tier 3 (Tests):       62 files,   5,000 tokens (42 excluded by budget)
//	  Tier 4 (Docs):        30 files,   1,500 tokens
//	  Tier 5 (CI/Lock):     17 files,     820 tokens (6 excluded by budget)
//
//	Total: 89,420 tokens / 200,000 budget (45%)
func GenerateInclusionSummary(result *tokenizer.BudgetResult) string {
	totalIncluded := len(result.IncludedFiles)
	totalExcluded := len(result.ExcludedFiles)

	var b strings.Builder

	fmt.Fprintf(&b, "Files: %s included, %s excluded\n",
		formatInt(totalIncluded), formatInt(totalExcluded))

	b.WriteString("\nBy Tier:\n")

	// Collect all unique tier keys from both included (TierStats) and excluded
	// files so that tiers with only excluded files also appear in the table.
	tierKeys := result.Summary.SortedTierKeys()

	// Also add any tier keys that appear only in ExcludedFiles but not in
	// TierStats (which tracks only processed files via budget enforcement).
	extraTiers := make(map[int]struct{})
	for _, tier := range tierKeys {
		extraTiers[tier] = struct{}{}
	}
	for _, fd := range result.ExcludedFiles {
		if fd == nil {
			continue
		}
		if _, ok := extraTiers[fd.Tier]; !ok {
			extraTiers[fd.Tier] = struct{}{}
			tierKeys = append(tierKeys, fd.Tier)
		}
	}
	sort.Ints(tierKeys)

	// Compute max label width for alignment.
	maxLabelWidth := 0
	for _, tier := range tierKeys {
		label := fmt.Sprintf("Tier %d (%s)", tier, TierLabel(tier))
		if len(label) > maxLabelWidth {
			maxLabelWidth = len(label)
		}
	}

	for _, tier := range tierKeys {
		stat := result.Summary.TierStats[tier]
		label := fmt.Sprintf("Tier %d (%s)", tier, TierLabel(tier))
		padding := strings.Repeat(" ", maxLabelWidth-len(label))

		filesCount := stat.FilesIncluded
		tokensUsed := stat.TokensUsed
		excluded := stat.FilesExcluded

		if excluded > 0 {
			fmt.Fprintf(&b, "  %s:%s  %s files,  %s tokens (%s excluded by budget)\n",
				label, padding,
				formatInt(filesCount),
				formatInt(tokensUsed),
				formatInt(excluded),
			)
		} else {
			fmt.Fprintf(&b, "  %s:%s  %s files,  %s tokens\n",
				label, padding,
				formatInt(filesCount),
				formatInt(tokensUsed),
			)
		}
	}

	// Budget line: only shown when budget enforcement was active (i.e. there
	// was an active budget, indicated by BudgetUsed > 0 or excluded files).
	hasBudget := result.BudgetUsed > 0 || totalExcluded > 0
	b.WriteString("\n")
	if hasBudget {
		budgetTotal := result.BudgetUsed + result.BudgetRemaining
		pct := 0
		if budgetTotal > 0 {
			pct = (result.TotalTokens * 100) / budgetTotal
		}
		fmt.Fprintf(&b, "Total: %s tokens / %s budget (%d%%)\n",
			formatInt(result.TotalTokens),
			formatInt(budgetTotal),
			pct,
		)
	} else {
		fmt.Fprintf(&b, "Total: %s tokens (no budget)\n",
			formatInt(result.TotalTokens),
		)
	}

	return b.String()
}

// formatInt formats an integer with comma thousands separators (e.g. 1234567
// becomes "1,234,567"). It is used for human-readable token and file counts.
func formatInt(n int) string {
	if n < 0 {
		return "-" + formatInt(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	if len(s) > 0 {
		parts = append([]string{s}, parts...)
	}
	return strings.Join(parts, ",")
}
