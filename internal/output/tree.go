// Package output provides rendering utilities for producing LLM-optimized
// context documents from processed codebases. It includes a directory tree
// builder and renderer that produces Unicode box-drawing visualizations of
// project file structures.
package output

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// FileEntry is the input to BuildTree -- a flat list of file paths with
// optional metadata such as size, token count, and relevance tier.
type FileEntry struct {
	Path       string // relative path e.g. "internal/cli/root.go"
	Size       int64  // file size in bytes
	TokenCount int    // token count
	Tier       int    // relevance tier
}

// TreeNode represents a directory or file in the in-memory tree.
type TreeNode struct {
	Name       string      // directory or file name (or collapsed path segment like "src/utils")
	IsDir      bool        // true for directories
	Children   []*TreeNode // child nodes (only for directories)
	Size       int64       // file size in bytes (only for files)
	TokenCount int         // token count (only for files)
	Tier       int         // relevance tier (only for files)
}

// TreeRenderOpts controls rendering behavior.
type TreeRenderOpts struct {
	MaxDepth   int  // 0 = unlimited depth
	ShowSize   bool // show file sizes like "(1.2 KB)"
	ShowTokens bool // show token counts like "(340 tokens)"
}

// BuildTree constructs an in-memory tree from a flat list of file entries.
// Each entry's Path is a forward-slash relative path. The returned root node
// has Name "." and IsDir true. Empty input produces a root with no children.
func BuildTree(files []FileEntry) *TreeNode {
	root := &TreeNode{
		Name:  ".",
		IsDir: true,
	}

	for _, f := range files {
		cleaned := path.Clean(f.Path)
		cleaned = strings.Trim(cleaned, "/")
		if cleaned == "" || cleaned == "." {
			continue
		}

		parts := strings.Split(cleaned, "/")
		insertFile(root, parts, f)
	}

	collapseTree(root)
	sortTree(root)

	return root
}

// insertFile walks or creates intermediate directory nodes for each path
// segment, then inserts the file as a leaf node.
func insertFile(parent *TreeNode, parts []string, entry FileEntry) {
	current := parent

	// Walk/create directory nodes for all but the last segment.
	for _, dirName := range parts[:len(parts)-1] {
		found := false
		for _, child := range current.Children {
			if child.IsDir && child.Name == dirName {
				current = child
				found = true
				break
			}
		}
		if !found {
			dir := &TreeNode{
				Name:  dirName,
				IsDir: true,
			}
			current.Children = append(current.Children, dir)
			current = dir
		}
	}

	// Insert the file leaf.
	fileName := parts[len(parts)-1]
	leaf := &TreeNode{
		Name:       fileName,
		IsDir:      false,
		Size:       entry.Size,
		TokenCount: entry.TokenCount,
		Tier:       entry.Tier,
	}
	current.Children = append(current.Children, leaf)
}

// collapseTree performs a recursive post-order traversal, merging directory
// nodes that have exactly one child which is also a directory. The parent name
// becomes "parent/child" and the parent takes the grandchildren.
func collapseTree(node *TreeNode) {
	if !node.IsDir {
		return
	}

	// First, recurse into all children.
	for _, child := range node.Children {
		collapseTree(child)
	}

	// Now attempt to collapse: if this directory has exactly one child and
	// that child is also a directory, merge them.
	for len(node.Children) == 1 && node.Children[0].IsDir {
		child := node.Children[0]
		// Don't collapse the root "." node's name into the child.
		if node.Name == "." {
			break
		}
		node.Name = node.Name + "/" + child.Name
		node.Children = child.Children
	}
}

// sortTree recursively sorts each node's children: directories come before
// files, and within each group items are sorted alphabetically
// (case-insensitive).
func sortTree(node *TreeNode) {
	if !node.IsDir || len(node.Children) == 0 {
		return
	}

	sort.SliceStable(node.Children, func(i, j int) bool {
		ci, cj := node.Children[i], node.Children[j]

		// Directories before files.
		if ci.IsDir != cj.IsDir {
			return ci.IsDir
		}

		// Alphabetical, case-insensitive.
		return strings.ToLower(ci.Name) < strings.ToLower(cj.Name)
	})

	for _, child := range node.Children {
		sortTree(child)
	}
}

// RenderTree produces a string with Unicode box-drawing characters representing
// the directory tree. It uses emoji indicators (folder and file) and supports
// depth limiting and optional metadata annotations.
func RenderTree(root *TreeNode, opts TreeRenderOpts) string {
	var sb strings.Builder

	// Render root line.
	sb.WriteString("\U0001F4C1 ")
	sb.WriteString(root.Name)
	sb.WriteByte('\n')

	// Render children.
	for i, child := range root.Children {
		isLast := i == len(root.Children)-1
		renderNode(&sb, child, "", isLast, 1, opts)
	}

	// Trim trailing newline for clean output.
	result := sb.String()
	result = strings.TrimRight(result, "\n")

	return result
}

// renderNode is the recursive helper that renders a single tree node with the
// appropriate box-drawing prefix, then recurses into children.
func renderNode(sb *strings.Builder, node *TreeNode, prefix string, isLast bool, depth int, opts TreeRenderOpts) {
	// Choose connector.
	connector := "\u251C\u2500\u2500 " // "├── "
	if isLast {
		connector = "\u2514\u2500\u2500 " // "└── "
	}

	// Choose emoji.
	emoji := "\U0001F4C4 " // 📄
	if node.IsDir {
		emoji = "\U0001F4C1 " // 📁
	}

	// Write this node's line.
	sb.WriteString(prefix)
	sb.WriteString(connector)
	sb.WriteString(emoji)
	sb.WriteString(node.Name)

	// Append metadata for files.
	if !node.IsDir {
		meta := formatMetadata(node, opts)
		sb.WriteString(meta)
	}

	sb.WriteByte('\n')

	// If this is a directory, render children (respecting depth limit).
	if node.IsDir && len(node.Children) > 0 {
		childPrefix := prefix
		if isLast {
			childPrefix += "    " // 4 spaces
		} else {
			childPrefix += "\u2502   " // "│   "
		}

		// Check depth limit.
		if opts.MaxDepth > 0 && depth >= opts.MaxDepth {
			// Render truncation indicator.
			sb.WriteString(childPrefix)
			sb.WriteString("\u2514\u2500\u2500 ") // "└── "
			sb.WriteString("...")
			sb.WriteByte('\n')
			return
		}

		for i, child := range node.Children {
			childIsLast := i == len(node.Children)-1
			renderNode(sb, child, childPrefix, childIsLast, depth+1, opts)
		}
	}
}

// formatMetadata returns a metadata annotation string for a file node, such as
// " (1.2 KB, 340 tokens)". Returns an empty string if neither ShowSize nor
// ShowTokens is enabled.
func formatMetadata(node *TreeNode, opts TreeRenderOpts) string {
	if !opts.ShowSize && !opts.ShowTokens {
		return ""
	}

	var parts []string

	if opts.ShowSize {
		parts = append(parts, humanizeSize(node.Size))
	}
	if opts.ShowTokens {
		parts = append(parts, fmt.Sprintf("%d tokens", node.TokenCount))
	}

	return " (" + strings.Join(parts, ", ") + ")"
}

// humanizeSize formats a byte count into a human-readable string with
// appropriate units (B, KB, MB, GB). Values 1 KB and above use one decimal
// place.
func humanizeSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
