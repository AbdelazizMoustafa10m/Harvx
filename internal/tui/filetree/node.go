// Package filetree implements the interactive file tree component for the
// Harvx TUI. It provides a navigable, togglable tree of files and directories
// with tri-state inclusion, lazy directory loading, and viewport scrolling.
package filetree

import (
	"path/filepath"
	"sort"
	"strings"
)

// InclusionState represents the tri-state inclusion of a node in the context
// output. Directories use Partial when they contain a mix of included and
// excluded children.
type InclusionState int

const (
	// Excluded means the node is not included in context output.
	Excluded InclusionState = iota
	// Included means the node is included in context output.
	Included
	// Partial means the directory has mixed included/excluded children.
	Partial
)

// String returns a human-readable representation of the inclusion state.
func (s InclusionState) String() string {
	switch s {
	case Excluded:
		return "excluded"
	case Included:
		return "included"
	case Partial:
		return "partial"
	default:
		return "unknown"
	}
}

// Node represents a file or directory in the tree. Nodes form a tree structure
// via the Children and Parent fields. Each node tracks its inclusion state for
// context generation.
type Node struct {
	// Path is the relative path from the repository root (forward slashes).
	Path string

	// Name is the base name for display.
	Name string

	// IsDir indicates whether this node represents a directory.
	IsDir bool

	// Included is the tri-state inclusion of this node.
	Included InclusionState

	// Expanded indicates whether a directory node's children are visible.
	Expanded bool

	// Tier is the relevance tier (0-5) for this file.
	Tier int

	// HasSecrets indicates whether secret content was detected in this file.
	HasSecrets bool

	// IsPriority indicates whether this file is marked as a priority file.
	IsPriority bool

	// TokenCount is the number of tokens in this file's content.
	TokenCount int

	// Children holds the child nodes for directory nodes.
	Children []*Node

	// Parent points to this node's parent directory node. Nil for the root.
	Parent *Node

	// loaded indicates whether a directory's children have been loaded from disk.
	loaded bool

	// depth is the nesting level (0 for top-level nodes).
	depth int
}

// NewNode creates a new Node with the given path, name, and directory flag.
// Directories are initialized with Expanded set to false.
func NewNode(path, name string, isDir bool) *Node {
	return &Node{
		Path:     filepath.ToSlash(path),
		Name:     name,
		IsDir:    isDir,
		Included: Excluded,
	}
}

// Depth returns the nesting level of this node in the tree.
func (n *Node) Depth() int {
	return n.depth
}

// Loaded returns whether a directory node's children have been loaded.
func (n *Node) Loaded() bool {
	return n.loaded
}

// SetLoaded marks a directory node as having its children loaded.
func (n *Node) SetLoaded(v bool) {
	n.loaded = v
}

// Toggle toggles the inclusion state of this node. For files, it toggles
// between Included and Excluded. For directories, it sets all descendants
// recursively to the new state, then propagates up to recalculate ancestor
// states.
func (n *Node) Toggle() {
	var newState InclusionState
	if n.Included == Included {
		newState = Excluded
	} else {
		newState = Included
	}

	n.setStateRecursive(newState)

	if n.Parent != nil {
		n.Parent.propagateUp()
	}
}

// setStateRecursive sets this node and all its descendants to the given state.
func (n *Node) setStateRecursive(state InclusionState) {
	n.Included = state
	for _, child := range n.Children {
		child.setStateRecursive(state)
	}
}

// propagateUp walks from this node to the root, recalculating each ancestor's
// inclusion state based on its children. A directory is Included if all children
// are Included, Excluded if all children are Excluded, and Partial otherwise.
func (n *Node) propagateUp() {
	if !n.IsDir || len(n.Children) == 0 {
		if n.Parent != nil {
			n.Parent.propagateUp()
		}
		return
	}

	allIncluded := true
	allExcluded := true

	for _, child := range n.Children {
		switch child.Included {
		case Included:
			allExcluded = false
		case Excluded:
			allIncluded = false
		case Partial:
			allIncluded = false
			allExcluded = false
		}
	}

	switch {
	case allIncluded:
		n.Included = Included
	case allExcluded:
		n.Included = Excluded
	default:
		n.Included = Partial
	}

	if n.Parent != nil {
		n.Parent.propagateUp()
	}
}

// AddChild adds a child node to this node, setting the child's parent pointer
// and depth.
func (n *Node) AddChild(child *Node) {
	child.Parent = n
	child.depth = n.depth + 1
	n.Children = append(n.Children, child)
}

// SortChildren sorts this node's children with directories first, then
// alphabetically by name (case-insensitive).
func (n *Node) SortChildren() {
	sort.Slice(n.Children, func(i, j int) bool {
		ci, cj := n.Children[i], n.Children[j]
		if ci.IsDir != cj.IsDir {
			return ci.IsDir
		}
		return strings.ToLower(ci.Name) < strings.ToLower(cj.Name)
	})
}

// VisibleNodes returns a flattened list of nodes visible from expanded branches.
// The root node itself is not included; only its descendants are returned.
// A directory's children are included only if the directory is expanded.
func (n *Node) VisibleNodes() []*Node {
	var result []*Node
	n.collectVisible(&result)
	return result
}

// collectVisible recursively collects visible nodes into the result slice.
func (n *Node) collectVisible(result *[]*Node) {
	for _, child := range n.Children {
		*result = append(*result, child)
		if child.IsDir && child.Expanded {
			child.collectVisible(result)
		}
	}
}

// FindByPath searches for a node by its relative path in this node's subtree.
// Returns nil if no node with the given path is found.
func (n *Node) FindByPath(path string) *Node {
	normalized := filepath.ToSlash(path)
	if n.Path == normalized {
		return n
	}
	for _, child := range n.Children {
		if found := child.FindByPath(normalized); found != nil {
			return found
		}
	}
	return nil
}

// IncludedFiles returns the paths of all included leaf nodes (files) in this
// node's subtree.
func (n *Node) IncludedFiles() []string {
	var paths []string
	n.collectIncluded(&paths)
	return paths
}

// collectIncluded recursively collects included file paths.
func (n *Node) collectIncluded(paths *[]string) {
	for _, child := range n.Children {
		if child.IsDir {
			child.collectIncluded(paths)
		} else if child.Included == Included {
			*paths = append(*paths, child.Path)
		}
	}
}
