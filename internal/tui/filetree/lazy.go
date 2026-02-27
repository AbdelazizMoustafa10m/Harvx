package filetree

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/harvx/harvx/internal/discovery"
)

// DirLoadedMsg is sent when a lazy directory load completes. It carries the
// children nodes for the loaded directory, or an error if loading failed.
type DirLoadedMsg struct {
	// Path is the relative path of the directory that was loaded. Empty string
	// means the root directory.
	Path string

	// Children contains the nodes for entries in the loaded directory.
	Children []*Node

	// Err is non-nil if the directory could not be read.
	Err error
}

// loadTopLevelCmd returns a tea.Cmd that scans the root directory and returns
// a DirLoadedMsg with top-level entries.
func loadTopLevelCmd(rootDir string, ignorer discovery.Ignorer) tea.Cmd {
	return func() tea.Msg {
		children, err := scanDirectory(rootDir, "", ignorer)
		return DirLoadedMsg{
			Path:     "",
			Children: children,
			Err:      err,
		}
	}
}

// loadDirCmd returns a tea.Cmd that scans a subdirectory and returns a
// DirLoadedMsg with its entries. The dirPath is relative to rootDir.
func loadDirCmd(rootDir, dirPath string, ignorer discovery.Ignorer) tea.Cmd {
	return func() tea.Msg {
		children, err := scanDirectory(rootDir, dirPath, ignorer)
		return DirLoadedMsg{
			Path:     dirPath,
			Children: children,
			Err:      err,
		}
	}
}

// scanDirectory reads directory entries from the filesystem, filters out
// ignored and binary entries, and returns a slice of Node structs. The dirPath
// is relative to rootDir; an empty dirPath reads the root directory itself.
func scanDirectory(rootDir, dirPath string, ignorer discovery.Ignorer) ([]*Node, error) {
	absDir := rootDir
	if dirPath != "" {
		absDir = filepath.Join(rootDir, filepath.FromSlash(dirPath))
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", absDir, err)
	}

	var children []*Node
	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden .git directory.
		if name == ".git" {
			continue
		}

		// Compute the relative path.
		var relPath string
		if dirPath == "" {
			relPath = name
		} else {
			relPath = filepath.ToSlash(filepath.Join(filepath.FromSlash(dirPath), name))
		}

		isDir := entry.IsDir()

		// Check ignore patterns.
		if ignorer != nil && ignorer.IsIgnored(relPath, isDir) {
			continue
		}

		// For files, check if binary.
		if !isDir {
			absPath := filepath.Join(absDir, name)
			isBin, binErr := discovery.IsBinary(absPath)
			if binErr != nil {
				// Skip files we cannot read for binary detection.
				continue
			}
			if isBin {
				continue
			}
		}

		node := NewNode(relPath, name, isDir)
		children = append(children, node)
	}

	return children, nil
}
