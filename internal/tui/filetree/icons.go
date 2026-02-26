package filetree

// Icon and indicator constants for file tree rendering.
const (
	// Inclusion state indicators.
	IncludedIcon = "[✓]"
	ExcludedIcon = "[✗]"
	PartialIcon  = "[◐]"

	// Directory expand/collapse indicators.
	DirCollapsed = "▸"
	DirExpanded  = "▾"

	// File type indicator.
	FileIcon = " "

	// Special indicators.
	PriorityIcon = "★"
	SecretIcon   = "🛡"

	// Tree-drawing characters.
	TreeBranch = "├── "
	TreeLast   = "└── "
	TreePipe   = "│   "
	TreeSpace  = "    "
)