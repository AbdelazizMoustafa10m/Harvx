package diff

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

const (
	// harvxDir is the hidden directory name used for Harvx state storage.
	harvxDir = ".harvx"

	// stateDir is the subdirectory under harvxDir for state snapshot files.
	stateDir = "state"
)

// safeProfileRe matches characters that are NOT safe for use in filenames.
var safeProfileRe = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// sanitizeProfileName replaces unsafe filename characters with underscores.
// If the result is empty after sanitization, "default" is returned.
func sanitizeProfileName(name string) string {
	s := safeProfileRe.ReplaceAllString(name, "_")
	if s == "" {
		return "default"
	}
	return s
}

// StateCache manages reading and writing state snapshots to disk. State files
// are stored at .harvx/state/<profile-name>.json relative to the project root.
// Each StateCache instance is scoped to a single profile name.
type StateCache struct {
	profileName string
}

// NewStateCache creates a new StateCache scoped to the given profile name.
// The profile name is sanitized for filesystem safety: only [a-zA-Z0-9_-]
// characters are allowed, others are replaced with underscores.
func NewStateCache(profileName string) *StateCache {
	return &StateCache{
		profileName: sanitizeProfileName(profileName),
	}
}

// GetStatePath returns the expected file path for this profile's state file,
// relative to rootDir.
func (c *StateCache) GetStatePath(rootDir string) string {
	return filepath.Join(rootDir, harvxDir, stateDir, c.profileName+".json")
}

// stateDirectory returns the path to the .harvx/state/ directory under rootDir.
func (c *StateCache) stateDirectory(rootDir string) string {
	return filepath.Join(rootDir, harvxDir, stateDir)
}

// HasState checks whether a cached state file exists for this profile.
func (c *StateCache) HasState(rootDir string) bool {
	_, err := os.Stat(c.GetStatePath(rootDir))
	return err == nil
}

// SaveState persists a StateSnapshot to .harvx/state/<profile-name>.json
// relative to rootDir. The directory is created if it does not exist. Writes
// are atomic: content is written to a temporary file in the same directory,
// then renamed to the final path. This prevents corruption if the process is
// interrupted during the write.
func (c *StateCache) SaveState(rootDir string, snapshot *StateSnapshot) error {
	dir := c.stateDirectory(rootDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating state directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state snapshot: %w", err)
	}

	finalPath := c.GetStatePath(rootDir)

	// Write to a temporary file in the same directory so that os.Rename is
	// atomic on POSIX systems (same filesystem).
	tmpFile, err := os.CreateTemp(dir, ".state-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file for state: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up the temp file on any error path.
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing state to temp file: %w", err)
	}

	// Ensure data is flushed to disk before renaming.
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("syncing state temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing state temp file: %w", err)
	}

	// Set desired permissions before rename.
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return fmt.Errorf("setting state file permissions: %w", err)
	}

	// Atomic rename. On Windows, os.Rename may fail if the destination exists,
	// so we fall back to remove-then-rename.
	if err := os.Rename(tmpPath, finalPath); err != nil {
		if runtime.GOOS == "windows" {
			os.Remove(finalPath)
			if err := os.Rename(tmpPath, finalPath); err != nil {
				return fmt.Errorf("renaming state file (windows fallback): %w", err)
			}
		} else {
			return fmt.Errorf("renaming state file: %w", err)
		}
	}

	success = true
	return nil
}

// LoadState reads the state file for this profile from rootDir and returns the
// parsed StateSnapshot. If no state file exists, ErrNoState is returned. If
// currentBranch is non-empty and the stored GitBranch is also non-empty and
// they differ, ErrBranchMismatch is returned (wrapped with both branch names).
func (c *StateCache) LoadState(rootDir, currentBranch string) (*StateSnapshot, error) {
	statePath := c.GetStatePath(rootDir)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoState
		}
		return nil, fmt.Errorf("reading state file %s: %w", statePath, err)
	}

	snap, err := ParseStateSnapshot(data)
	if err != nil {
		return nil, fmt.Errorf("parsing state file %s: %w", statePath, err)
	}

	// Check for branch mismatch when both branches are known.
	if currentBranch != "" && snap.GitBranch != "" && currentBranch != snap.GitBranch {
		return nil, fmt.Errorf(
			"cached state is for branch %q but current branch is %q: %w",
			snap.GitBranch, currentBranch, ErrBranchMismatch,
		)
	}

	return snap, nil
}

// ClearState deletes the cached state file for this profile. If the file does
// not exist, nil is returned (idempotent operation).
func (c *StateCache) ClearState(rootDir string) error {
	statePath := c.GetStatePath(rootDir)

	if err := os.Remove(statePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("removing state file %s: %w", statePath, err)
	}

	return nil
}

// ClearAllState deletes the entire .harvx/state/ directory and all state files
// within it. If the directory does not exist, nil is returned.
func (c *StateCache) ClearAllState(rootDir string) error {
	dir := c.stateDirectory(rootDir)

	if err := os.RemoveAll(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("removing state directory %s: %w", dir, err)
	}

	return nil
}
