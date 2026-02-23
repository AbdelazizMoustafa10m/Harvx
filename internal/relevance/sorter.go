// Package relevance - this file implements the relevance sorter (T-028).
// Files are sorted by ascending tier number (0 first, 5 last) with a
// secondary alphabetical sort on Path within each tier. The sort is stable
// and deterministic.
package relevance

import (
	"cmp"
	"slices"
	"sort"

	"github.com/harvx/harvx/internal/pipeline"
)

// SortByRelevance returns a new slice of FileDescriptor pointers sorted by
// ascending Tier (primary key) and then alphabetically by Path (secondary key).
// The input slice is never mutated. The sort is stable: descriptors that share
// identical Tier and Path values retain their original relative order.
func SortByRelevance(files []*pipeline.FileDescriptor) []*pipeline.FileDescriptor {
	out := make([]*pipeline.FileDescriptor, len(files))
	copy(out, files)

	slices.SortStableFunc(out, func(a, b *pipeline.FileDescriptor) int {
		if n := cmp.Compare(a.Tier, b.Tier); n != 0 {
			return n
		}
		return cmp.Compare(a.Path, b.Path)
	})

	return out
}

// GroupByTier partitions a slice of FileDescriptor pointers into a map keyed
// by tier number. Each map value is a slice that preserves the original
// insertion order of the input. Files that share a tier are grouped together
// without any additional sorting.
func GroupByTier(files []*pipeline.FileDescriptor) map[int][]*pipeline.FileDescriptor {
	result := make(map[int][]*pipeline.FileDescriptor)
	for _, fd := range files {
		result[fd.Tier] = append(result[fd.Tier], fd)
	}
	return result
}

// TierStat holds aggregate statistics for a single relevance tier.
type TierStat struct {
	// Tier is the relevance tier number (0–5).
	Tier int

	// FileCount is the number of files assigned to this tier.
	FileCount int

	// TotalTokens is the sum of TokenCount across all files in this tier.
	TotalTokens int

	// FilePaths is the sorted list of relative paths for files in this tier.
	FilePaths []string
}

// TierSummary computes per-tier statistics for the provided files. Only tiers
// that contain at least one file are included in the result. The returned slice
// is sorted by ascending Tier value, and the FilePaths within each TierStat
// are sorted alphabetically.
func TierSummary(files []*pipeline.FileDescriptor) []TierStat {
	type accumulator struct {
		tokens int
		paths  []string
	}

	acc := make(map[int]*accumulator)
	for _, fd := range files {
		a, ok := acc[fd.Tier]
		if !ok {
			a = &accumulator{}
			acc[fd.Tier] = a
		}
		a.tokens += fd.TokenCount
		a.paths = append(a.paths, fd.Path)
	}

	stats := make([]TierStat, 0, len(acc))
	for tier, a := range acc {
		sort.Strings(a.paths)
		stats = append(stats, TierStat{
			Tier:        tier,
			FileCount:   len(a.paths),
			TotalTokens: a.tokens,
			FilePaths:   a.paths,
		})
	}

	// Sort result by ascending tier so callers receive a deterministic order.
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Tier < stats[j].Tier
	})

	return stats
}

// ClassifyAndSort classifies a slice of FileDescriptor pointers using the
// provided tier definitions and returns them sorted by relevance. The Tier
// field on each FileDescriptor is updated in place before sorting.
//
// It constructs a TierMatcher from tiers (per T-027), assigns the matched
// Tier to each descriptor, and returns the result of SortByRelevance.
func ClassifyAndSort(files []*pipeline.FileDescriptor, tiers []TierDefinition) []*pipeline.FileDescriptor {
	matcher := NewTierMatcher(tiers)
	for _, fd := range files {
		fd.Tier = int(matcher.Match(fd.Path))
	}
	return SortByRelevance(files)
}
