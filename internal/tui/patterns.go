package tui

import (
	"sort"
	"strings"

	"github.com/harvx/harvx/internal/tui/filetree"
)

// MinimizedPatterns holds the result of pattern minimization on a file tree.
// Include patterns capture selected files/directories; Ignore patterns capture
// excluded files/directories; PriorityFiles are tier-0 included files.
type MinimizedPatterns struct {
	// Include contains glob patterns for included files and directories.
	// Directory patterns use "dir/**" form.
	Include []string

	// Ignore contains glob patterns for excluded files and directories.
	// Directory patterns use "dir/**" form.
	Ignore []string

	// PriorityFiles are included files with Tier == 0.
	PriorityFiles []string

	// TierFiles maps tier numbers (0-5) to lists of included file paths
	// assigned to that tier.
	TierFiles map[int][]string
}

// MinimizePatterns walks the file tree bottom-up and produces minimized glob
// patterns for the current selection state. For each directory:
//   - If all children are Included, emit "dir/**" instead of individual files
//   - If all children are Excluded, emit ignore "dir/**"
//   - If mixed, recurse into children
//
// The root node itself is not emitted as a pattern.
func MinimizePatterns(root *filetree.Node) MinimizedPatterns {
	result := MinimizedPatterns{
		TierFiles: make(map[int][]string),
	}

	if root == nil {
		return result
	}

	minimizeNode(root, &result)

	// Sort all slices for deterministic output.
	sort.Strings(result.Include)
	sort.Strings(result.Ignore)
	sort.Strings(result.PriorityFiles)
	for tier := range result.TierFiles {
		sort.Strings(result.TierFiles[tier])
	}

	return result
}

// minimizeNode recursively processes a node and its children. It decides
// whether to emit a directory glob or recurse into individual children.
func minimizeNode(n *filetree.Node, result *MinimizedPatterns) {
	for _, child := range n.Children {
		if child.IsDir {
			switch child.Included {
			case filetree.Included:
				// All children included -- emit directory glob.
				glob := child.Path + "/**"
				result.Include = append(result.Include, glob)
				// Still collect tier info and priority files from leaves.
				collectTierInfo(child, result)
			case filetree.Excluded:
				// All children excluded -- emit directory ignore.
				glob := child.Path + "/**"
				result.Ignore = append(result.Ignore, glob)
			case filetree.Partial:
				// Mixed -- recurse into individual children.
				minimizeNode(child, result)
			}
		} else {
			// Leaf file node.
			if child.Included == filetree.Included {
				result.Include = append(result.Include, child.Path)
				result.TierFiles[child.Tier] = append(result.TierFiles[child.Tier], child.Path)
				if child.Tier == 0 {
					result.PriorityFiles = append(result.PriorityFiles, child.Path)
				}
			} else if child.Included == filetree.Excluded {
				// Only emit individual file ignores if the parent is Partial
				// (not all-excluded). The parent directory glob handles the
				// all-excluded case.
				if n.Included == filetree.Partial {
					result.Ignore = append(result.Ignore, child.Path)
				}
			}
		}
	}
}

// collectTierInfo walks all leaf files in a fully-included directory subtree
// and records their tier assignments and priority status.
func collectTierInfo(n *filetree.Node, result *MinimizedPatterns) {
	for _, child := range n.Children {
		if child.IsDir {
			collectTierInfo(child, result)
		} else {
			result.TierFiles[child.Tier] = append(result.TierFiles[child.Tier], child.Path)
			if child.Tier == 0 {
				result.PriorityFiles = append(result.PriorityFiles, child.Path)
			}
		}
	}
}

// HasManualIncludes reports whether the minimized patterns contain any include
// patterns that are not directory globs. These represent files that were
// manually selected by the user in the TUI rather than being captured by a
// directory-level "dir/**" glob.
func (p MinimizedPatterns) HasManualIncludes() bool {
	for _, inc := range p.Include {
		if !strings.HasSuffix(inc, "/**") {
			return true
		}
	}
	return false
}
