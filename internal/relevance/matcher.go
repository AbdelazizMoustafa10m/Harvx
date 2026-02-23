// Package relevance implements tier-based file sorting and token budget management.
// This file implements the glob-based file-to-tier matching engine (T-027).
package relevance

import (
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// TierMatcher assigns each file path to exactly one relevance tier using
// glob patterns defined in a slice of TierDefinition. Tiers are evaluated in
// ascending order (Tier0Critical first); the first matching pattern wins.
// Files that match no pattern are assigned DefaultUnmatchedTier (Tier2Secondary).
//
// Construct once via NewTierMatcher and reuse for all files; pattern
// validation happens at construction time so per-file matching is allocation-free.
type TierMatcher struct {
	// tiers holds the validated tier definitions in priority order (lowest
	// tier number first). Within each tier, patterns are stored in their
	// original order so first-match semantics are preserved.
	tiers []tierEntry
}

// tierEntry pairs a Tier with its pre-validated patterns.
type tierEntry struct {
	tier     Tier
	patterns []string // only syntactically valid patterns are kept
}

// NewTierMatcher constructs a TierMatcher from the supplied tier definitions.
// Definitions are sorted by ascending Tier value so that Tier0Critical is
// evaluated before Tier1Primary, and so on.
//
// Patterns that fail doublestar.ValidatePattern are silently discarded; a
// definition with no valid patterns is kept in the list (it simply never
// matches anything, which is harmless).
//
// Pass nil or an empty slice to get a matcher that assigns every file to
// DefaultUnmatchedTier.
func NewTierMatcher(defs []TierDefinition) *TierMatcher {
	// Sort a copy of defs by tier number so the caller's slice is never mutated.
	sorted := make([]TierDefinition, len(defs))
	copy(sorted, defs)
	sortTierDefinitions(sorted)

	entries := make([]tierEntry, 0, len(sorted))
	for _, d := range sorted {
		valid := make([]string, 0, len(d.Patterns))
		for _, p := range d.Patterns {
			if doublestar.ValidatePattern(p) {
				valid = append(valid, p)
			}
		}
		entries = append(entries, tierEntry{tier: d.Tier, patterns: valid})
	}

	return &TierMatcher{tiers: entries}
}

// sortTierDefinitions sorts defs in place by ascending Tier value using
// insertion sort (the list is short -- at most 6 entries in practice).
func sortTierDefinitions(defs []TierDefinition) {
	for i := 1; i < len(defs); i++ {
		key := defs[i]
		j := i - 1
		for j >= 0 && defs[j].Tier > key.Tier {
			defs[j+1] = defs[j]
			j--
		}
		defs[j+1] = key
	}
}

// Match returns the Tier for the given file path.
//
// filePath must be a relative path using forward slashes (e.g. "src/main.go").
// On Windows, call filepath.ToSlash before passing the path. The path is
// normalised internally so leading "./" components are stripped.
//
// Matching is performed by iterating tiers from lowest number (highest
// priority) to highest number. Within each tier patterns are checked in
// definition order. The Tier of the first matching pattern is returned.
// If no pattern matches, DefaultUnmatchedTier is returned.
func (m *TierMatcher) Match(filePath string) Tier {
	normalised := normalisePath(filePath)

	for _, entry := range m.tiers {
		for _, pattern := range entry.patterns {
			matched, err := doublestar.Match(pattern, normalised)
			if err != nil {
				// ValidatePattern already filtered bad patterns at construction
				// time; this branch should be unreachable in practice.
				continue
			}
			if matched {
				return entry.tier
			}
		}
	}

	return DefaultUnmatchedTier
}

// ClassifyFiles bulk-classifies a slice of file paths against the provided tier
// definitions and returns a map of filePath -> Tier. The function constructs a
// fresh TierMatcher from tiers so it can be called without a pre-built matcher.
//
// Performance: O(n * m) where n = len(files) and m = total patterns across all
// tiers. Suitable for repositories up to 50 000 files with typical pattern
// counts (~20 patterns).
//
// The returned map uses the original (non-normalised) file paths as keys so
// callers can look up results with the same paths they supplied.
func ClassifyFiles(files []string, tiers []TierDefinition) map[string]Tier {
	matcher := NewTierMatcher(tiers)
	result := make(map[string]Tier, len(files))
	for _, f := range files {
		result[f] = matcher.Match(f)
	}
	return result
}

// normalisePath strips a leading "./" from path and converts any OS-specific
// separators to forward slashes, ensuring compatibility with doublestar.Match
// which splits on "/".
func normalisePath(path string) string {
	// Always replace backslashes with forward slashes so that callers on any
	// platform can pass Windows-style paths and have them matched correctly.
	// strings.ReplaceAll is used instead of filepath.ToSlash because
	// filepath.ToSlash is a no-op on non-Windows systems.
	path = strings.ReplaceAll(path, `\`, "/")
	// Strip optional leading "./".
	path = strings.TrimPrefix(path, "./")
	return path
}
