package discovery

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// SymlinkResolver tracks visited real paths to detect symlink loops during file
// discovery. It resolves symlinks to their real paths using filepath.EvalSymlinks
// and maintains a set of visited paths to break cycles.
//
// SymlinkResolver is safe for concurrent use. All access to the visited set is
// protected by a sync.RWMutex.
type SymlinkResolver struct {
	visited map[string]bool
	mu      sync.RWMutex
	logger  *slog.Logger
}

// NewSymlinkResolver creates a new SymlinkResolver with an empty visited set.
func NewSymlinkResolver() *SymlinkResolver {
	return &SymlinkResolver{
		visited: make(map[string]bool),
		logger:  slog.Default().With("component", "symlink-resolver"),
	}
}

// Resolve resolves the given path through any symlinks and checks for loops.
// It returns:
//   - realPath: the resolved real filesystem path (empty string on error).
//   - isLoop: true if the resolved path has already been visited (cycle detected).
//   - err: non-nil if the symlink is dangling (target does not exist) or another
//     filesystem error occurs.
//
// When a loop is detected, the caller should skip the path. When an error is
// returned (e.g., dangling symlink), the caller should skip with a warning.
//
// Resolve does NOT automatically mark the path as visited; the caller must call
// MarkVisited after deciding to process the path. This two-step design allows
// the caller to check before committing.
func (s *SymlinkResolver) Resolve(path string) (realPath string, isLoop bool, err error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, fmt.Errorf("dangling symlink %s: %w", path, err)
		}
		return "", false, fmt.Errorf("resolving symlink %s: %w", path, err)
	}

	s.mu.RLock()
	loop := s.visited[resolved]
	s.mu.RUnlock()

	if loop {
		s.logger.Debug("symlink loop detected",
			"path", path,
			"real_path", resolved,
		)
		return resolved, true, nil
	}

	return resolved, false, nil
}

// MarkVisited records a real path as visited. After calling MarkVisited,
// subsequent calls to Resolve for paths that resolve to the same real path
// will return isLoop=true.
func (s *SymlinkResolver) MarkVisited(realPath string) {
	s.mu.Lock()
	s.visited[realPath] = true
	s.mu.Unlock()
}

// Reset clears the visited set. This is useful when starting a new discovery
// pass over the same directory tree.
func (s *SymlinkResolver) Reset() {
	s.mu.Lock()
	s.visited = make(map[string]bool)
	s.mu.Unlock()
}

// VisitedCount returns the number of unique real paths that have been visited.
// This is useful for diagnostics and logging.
func (s *SymlinkResolver) VisitedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.visited)
}

// IsSymlink reports whether the file at the given path is a symbolic link.
// It uses os.Lstat (which does not follow symlinks) to check the file mode.
// Returns false for regular files and directories.
//
// This is a standalone helper function with no shared state, safe for
// concurrent use.
func IsSymlink(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, fmt.Errorf("lstat %s: %w", path, err)
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}
