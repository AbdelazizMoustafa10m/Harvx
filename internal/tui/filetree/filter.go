package filetree

import (
	"github.com/harvx/harvx/internal/tui/search"
)

// FilterState holds the current filter configuration for the file tree.
type FilterState struct {
	// SearchQuery is the current text search filter.
	SearchQuery string
	// TierFilter is the active tier filter. -1 means "All" (no tier filter).
	TierFilter int
}

// NewFilterState creates a default filter state with no active filters.
func NewFilterState() FilterState {
	return FilterState{TierFilter: -1}
}

// HasSearchFilter returns whether a text search filter is active.
func (f FilterState) HasSearchFilter() bool {
	return f.SearchQuery != ""
}

// HasTierFilter returns whether a tier filter is active.
func (f FilterState) HasTierFilter() bool {
	return f.TierFilter >= 0
}

// HasAnyFilter returns whether any filter is active.
func (f FilterState) HasAnyFilter() bool {
	return f.HasSearchFilter() || f.HasTierFilter()
}

// TierLabel returns a human-readable label for the current tier filter.
func (f FilterState) TierLabel() string {
	if f.TierFilter < 0 {
		return "All"
	}
	labels := map[int]string{
		0: "0 (critical)",
		1: "1 (primary)",
		2: "2 (secondary)",
		3: "3 (tests)",
		4: "4 (docs)",
		5: "5 (low)",
	}
	if label, ok := labels[f.TierFilter]; ok {
		return label
	}
	return "All"
}

// CycleTier advances the tier filter: All -> 0 -> 1 -> ... -> 5 -> All.
func (f FilterState) CycleTier() FilterState {
	f.TierFilter++
	if f.TierFilter > 5 {
		f.TierFilter = -1
	}
	return f
}

// FilterNodes applies the current filter state to a list of nodes, returning
// only nodes that match all active filters. Directories are included if any
// of their descendants match.
func FilterNodes(nodes []*Node, filter FilterState) []*Node {
	if !filter.HasAnyFilter() {
		return nodes
	}

	result := make([]*Node, 0, len(nodes))
	for _, node := range nodes {
		if matchesFilter(node, filter) {
			result = append(result, node)
		}
	}
	return result
}

// matchesFilter checks whether a single node passes all active filters.
// Directories always pass if they have any matching descendant in the
// visible list (they are included as context for matching children).
func matchesFilter(node *Node, filter FilterState) bool {
	if node.IsDir {
		// Include directories that have matching file descendants.
		return hasMatchingDescendant(node, filter)
	}

	// Check tier filter.
	if filter.HasTierFilter() && node.Tier != filter.TierFilter {
		return false
	}

	// Check search filter.
	if filter.HasSearchFilter() {
		return search.MatchSubstring(node.Path, filter.SearchQuery)
	}

	return true
}

// hasMatchingDescendant checks whether a directory has any file descendant
// that matches the filter.
func hasMatchingDescendant(dir *Node, filter FilterState) bool {
	for _, child := range dir.Children {
		if child.IsDir {
			if hasMatchingDescendant(child, filter) {
				return true
			}
		} else {
			if filter.HasTierFilter() && child.Tier != filter.TierFilter {
				continue
			}
			if filter.HasSearchFilter() && !search.MatchSubstring(child.Path, filter.SearchQuery) {
				continue
			}
			return true
		}
	}
	return false
}

// VisibleFiles returns all non-directory (file) nodes from a list.
func VisibleFiles(nodes []*Node) []*Node {
	result := make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		if !n.IsDir {
			result = append(result, n)
		}
	}
	return result
}

// SelectAll sets all given nodes to Included state. For directories, all
// descendants are also set. Parent states are propagated up after changes.
func SelectAll(nodes []*Node) {
	for _, node := range nodes {
		if node.IsDir {
			node.setStateRecursive(Included)
		} else {
			node.Included = Included
		}
	}
	propagateAllParents(nodes)
}

// DeselectAll sets all given nodes to Excluded state. For directories, all
// descendants are also set. Parent states are propagated up after changes.
func DeselectAll(nodes []*Node) {
	for _, node := range nodes {
		if node.IsDir {
			node.setStateRecursive(Excluded)
		} else {
			node.Included = Excluded
		}
	}
	propagateAllParents(nodes)
}

// propagateAllParents propagates inclusion state up from all affected nodes.
func propagateAllParents(nodes []*Node) {
	seen := make(map[*Node]bool)
	for _, node := range nodes {
		p := node.Parent
		for p != nil && !seen[p] {
			seen[p] = true
			p.propagateUp()
			p = p.Parent
		}
	}
}
