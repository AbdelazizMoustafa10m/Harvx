package filetree

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/harvx/harvx/internal/discovery"
)

// Model is the file tree Bubble Tea model. It manages a tree of Node structs,
// keyboard navigation, viewport scrolling, and lazy directory loading.
type Model struct {
	root    *Node
	visible []*Node // cached flattened visible nodes
	cursor  int     // index into visible
	offset  int     // scroll offset for viewport
	width   int
	height  int // viewport height in lines
	rootDir string
	ignorer discovery.Ignorer
	loading map[string]bool // dirs currently being loaded
	ready   bool
	logger  *slog.Logger
}

// New creates a new file tree Model rooted at the given directory. The ignorer
// is used to filter out files matching ignore patterns during lazy loading.
// Pass nil for ignorer if no filtering is desired.
func New(rootDir string, ignorer discovery.Ignorer) Model {
	root := NewNode("", rootDir, true)
	root.Expanded = true
	root.loaded = false

	return Model{
		root:    root,
		rootDir: rootDir,
		ignorer: ignorer,
		loading: make(map[string]bool),
		logger:  slog.Default().With("component", "filetree"),
	}
}

// Init returns a command to scan top-level entries in the root directory.
func (m Model) Init() tea.Cmd {
	return loadTopLevelCmd(m.rootDir, m.ignorer)
}

// SetSize updates the viewport dimensions used for scroll calculations.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Root returns the root node of the tree.
func (m Model) Root() *Node {
	return m.root
}

// Cursor returns the current cursor position (index into visible nodes).
func (m Model) Cursor() int {
	return m.cursor
}

// Offset returns the current scroll offset.
func (m Model) Offset() int {
	return m.offset
}

// Width returns the viewport width.
func (m Model) Width() int {
	return m.width
}

// Height returns the viewport height.
func (m Model) Height() int {
	return m.height
}

// Ready reports whether the model has been initialized with directory content.
func (m Model) Ready() bool {
	return m.ready
}

// IsLoading reports whether the given directory path is currently being loaded.
func (m Model) IsLoading(path string) bool {
	return m.loading[path]
}

// SelectedFiles returns the relative paths of all included files in the tree.
func (m Model) SelectedFiles() []string {
	return m.root.IncludedFiles()
}

// Visible returns the cached list of visible nodes. This is useful for testing
// and rendering.
func (m Model) Visible() []*Node {
	return m.visible
}

// refreshVisible rebuilds the visible node list from the root node.
func (m *Model) refreshVisible() {
	m.visible = m.root.VisibleNodes()
}

// clampCursor ensures the cursor stays within the valid range of visible nodes.
func (m *Model) clampCursor() {
	if len(m.visible) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
}

// adjustScroll ensures the cursor is visible within the viewport by adjusting
// the scroll offset.
func (m *Model) adjustScroll() {
	if m.height <= 0 {
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.height {
		m.offset = m.cursor - m.height + 1
	}
}

// cursorNode returns the node at the current cursor position, or nil if the
// visible list is empty.
func (m Model) cursorNode() *Node {
	if len(m.visible) == 0 || m.cursor < 0 || m.cursor >= len(m.visible) {
		return nil
	}
	return m.visible[m.cursor]
}
