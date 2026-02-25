package diff

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiffMode represents the strategy for computing changes between two states.
type DiffMode int

const (
	// DiffModeCache compares the current project state against a cached state
	// snapshot from a previous run.
	DiffModeCache DiffMode = iota + 1

	// DiffModeSince compares the current project state against a specific git
	// ref using git diff --name-status.
	DiffModeSince

	// DiffModeBaseHead compares two git refs directly, useful for PR review
	// workflows.
	DiffModeBaseHead
)

// String returns the human-readable name of the diff mode.
func (m DiffMode) String() string {
	switch m {
	case DiffModeCache:
		return "cache"
	case DiffModeSince:
		return "since"
	case DiffModeBaseHead:
		return "base-head"
	default:
		return "unknown"
	}
}

// DiffOptions configures a diff operation. The Mode field determines which
// comparison strategy is used, and the remaining fields provide the parameters
// for that strategy.
type DiffOptions struct {
	// Mode selects the comparison strategy.
	Mode DiffMode

	// RootDir is the absolute path to the repository root directory.
	RootDir string

	// ProfileName is the profile name used for cache key selection.
	ProfileName string

	// SinceRef is the git ref to diff against when Mode is DiffModeSince.
	SinceRef string

	// BaseRef is the base git ref when Mode is DiffModeBaseHead.
	BaseRef string

	// HeadRef is the head git ref when Mode is DiffModeBaseHead.
	HeadRef string
}

// DiffOutput holds the complete result of a diff operation, including the
// classified file changes and a human-readable summary.
type DiffOutput struct {
	// Result contains the classified file changes (added, modified, deleted).
	Result *DiffResult

	// Summary is a human-readable summary of the changes.
	Summary string
}

// RunDiff executes a diff operation based on the given options. It dispatches
// to the appropriate comparison strategy based on opts.Mode and returns a
// DiffOutput containing the classified changes and summary.
func RunDiff(ctx context.Context, opts DiffOptions) (*DiffOutput, error) {
	if opts.RootDir == "" {
		return nil, fmt.Errorf("root directory required")
	}

	slog.Debug("running diff",
		"mode", opts.Mode.String(),
		"root", opts.RootDir,
		"profile", opts.ProfileName,
	)

	var result *DiffResult
	var err error

	switch opts.Mode {
	case DiffModeCache:
		result, err = runCacheDiff(ctx, opts)
	case DiffModeSince:
		result, err = runSinceDiff(ctx, opts)
	case DiffModeBaseHead:
		result, err = runBaseHeadDiff(ctx, opts)
	default:
		return nil, fmt.Errorf("unknown diff mode: %d", opts.Mode)
	}

	if err != nil {
		return nil, err
	}

	summary := FormatChangeSummary(result)

	return &DiffOutput{
		Result:  result,
		Summary: summary,
	}, nil
}

// runCacheDiff compares the current project state against a cached state
// snapshot. It loads the cached state for the given profile, builds a current
// snapshot by scanning the project directory, and compares the two.
func runCacheDiff(ctx context.Context, opts DiffOptions) (*DiffResult, error) {
	cache := NewStateCache(opts.ProfileName)

	if !cache.HasState(opts.RootDir) {
		return nil, ErrNoState
	}

	// Determine the current branch for cache branch-mismatch detection.
	gitDiffer := NewGitDiffer()
	currentBranch, err := gitDiffer.GetCurrentBranch(ctx, opts.RootDir)
	if err != nil {
		// Not a git repo is fine for cache-based diffing; just skip branch check.
		slog.Debug("could not determine current branch", "error", err)
		currentBranch = ""
	}

	previous, err := cache.LoadState(opts.RootDir, currentBranch)
	if err != nil {
		return nil, fmt.Errorf("loading cached state: %w", err)
	}

	// Build a current snapshot by scanning the directory tree.
	current, err := buildCurrentSnapshot(ctx, opts.RootDir, opts.ProfileName)
	if err != nil {
		return nil, fmt.Errorf("building current state: %w", err)
	}

	return CompareStates(previous, current), nil
}

// runSinceDiff compares the current project state against a git ref using
// git diff --name-status.
func runSinceDiff(ctx context.Context, opts DiffOptions) (*DiffResult, error) {
	if opts.SinceRef == "" {
		return nil, fmt.Errorf("since ref required for DiffModeSince")
	}

	gitDiffer := NewGitDiffer()

	changes, err := gitDiffer.GetChangedFilesSince(ctx, opts.RootDir, opts.SinceRef)
	if err != nil {
		return nil, fmt.Errorf("getting changes since %s: %w", opts.SinceRef, err)
	}

	return BuildDiffResultFromGit(changes), nil
}

// runBaseHeadDiff compares two git refs using git diff --name-status.
func runBaseHeadDiff(ctx context.Context, opts DiffOptions) (*DiffResult, error) {
	if opts.BaseRef == "" || opts.HeadRef == "" {
		return nil, fmt.Errorf("both base and head refs required for DiffModeBaseHead")
	}

	gitDiffer := NewGitDiffer()

	changes, err := gitDiffer.GetChangedFiles(ctx, opts.RootDir, opts.BaseRef, opts.HeadRef)
	if err != nil {
		return nil, fmt.Errorf("getting changes %s..%s: %w", opts.BaseRef, opts.HeadRef, err)
	}

	return BuildDiffResultFromGit(changes), nil
}

// buildCurrentSnapshot scans the directory tree at rootDir and builds a
// StateSnapshot representing the current project state. This is a lightweight
// scan that captures file sizes and modification times but does not hash
// content (hashing is deferred to the full pipeline).
func buildCurrentSnapshot(ctx context.Context, rootDir, profileName string) (*StateSnapshot, error) {
	gitDiffer := NewGitDiffer()

	gitBranch, _ := gitDiffer.GetCurrentBranch(ctx, rootDir)
	gitSHA, _ := gitDiffer.GetHeadSHA(ctx, rootDir)

	snap := NewStateSnapshot(profileName, rootDir, gitBranch, gitSHA)

	// Use the XXH3 hasher to hash files for comparison.
	hasher := NewXXH3Hasher()

	// Walk the directory and build the snapshot. For now, do a simple walk
	// that captures all regular files. The full pipeline's discovery engine
	// is used by the generate command; here we do a lighter walk.
	err := walkDir(ctx, rootDir, func(relPath, absPath string, size int64, modTime string) error {
		hash, hashErr := hasher.HashFile(absPath)
		if hashErr != nil {
			slog.Debug("skipping file due to hash error", "path", relPath, "error", hashErr)
			return nil
		}

		snap.AddFile(relPath, FileState{
			Size:         size,
			ContentHash:  hash,
			ModifiedTime: modTime,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", rootDir, err)
	}

	return snap, nil
}

// walkDir walks the directory tree rooted at rootDir and calls fn for each
// regular file. It skips hidden directories (starting with '.') and common
// build/vendor directories.
func walkDir(ctx context.Context, rootDir string, fn func(relPath, absPath string, size int64, modTime string) error) error {
	return walkDirImpl(ctx, rootDir, rootDir, fn)
}

// walkDirImpl is the recursive implementation of walkDir.
func walkDirImpl(ctx context.Context, rootDir, currentDir string, fn func(relPath, absPath string, size int64, modTime string) error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	entries, err := readDirEntries(currentDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.name

		// Skip hidden entries and common non-source directories.
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.isDir && isSkippedDir(name) {
			continue
		}

		absPath := filepath.Join(currentDir, name)

		relPath, err := filepath.Rel(rootDir, absPath)
		if err != nil {
			relPath = strings.TrimPrefix(absPath, rootDir+string(filepath.Separator))
		}

		if entry.isDir {
			if err := walkDirImpl(ctx, rootDir, absPath, fn); err != nil {
				return err
			}
			continue
		}

		if err := fn(relPath, absPath, entry.size, entry.modTime); err != nil {
			return err
		}
	}

	return nil
}

// dirEntry holds minimal file information for directory walking.
type dirEntry struct {
	name    string
	isDir   bool
	size    int64
	modTime string
}

// readDirEntries reads directory entries using os.ReadDir and returns sorted
// dirEntry values with file metadata.
func readDirEntries(dir string) ([]dirEntry, error) {
	osEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	entries := make([]dirEntry, 0, len(osEntries))
	for _, e := range osEntries {
		// Skip symlinks and irregular files (keep regular files and directories).
		if !e.IsDir() && (e.Type()&fs.ModeSymlink != 0 || !e.Type().IsRegular()) {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}
		entries = append(entries, dirEntry{
			name:    e.Name(),
			isDir:   e.IsDir(),
			size:    info.Size(),
			modTime: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	return entries, nil
}

// skippedDirs contains directory names that should be skipped during the
// lightweight directory walk used for cache-based diffing.
var skippedDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	"dist":         true,
	"build":        true,
	"target":       true,
}

// isSkippedDir reports whether the directory name should be skipped.
func isSkippedDir(name string) bool {
	return skippedDirs[name]
}

// FormatChangeSummary generates a human-readable change summary header
// including counts and file paths for each change category.
func FormatChangeSummary(result *DiffResult) string {
	if result == nil {
		return "No changes detected."
	}

	if !result.HasChanges() {
		return "No changes detected."
	}

	var sb strings.Builder

	sb.WriteString("Change Summary: ")
	sb.WriteString(result.Summary())
	sb.WriteString("\n")

	if len(result.Added) > 0 {
		sb.WriteString("\nAdded:\n")
		for _, path := range result.Added {
			sb.WriteString("  + ")
			sb.WriteString(path)
			sb.WriteString("\n")
		}
	}

	if len(result.Modified) > 0 {
		sb.WriteString("\nModified:\n")
		for _, path := range result.Modified {
			sb.WriteString("  ~ ")
			sb.WriteString(path)
			sb.WriteString("\n")
		}
	}

	if len(result.Deleted) > 0 {
		sb.WriteString("\nDeleted:\n")
		for _, path := range result.Deleted {
			sb.WriteString("  - ")
			sb.WriteString(path)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// DetermineDiffMode selects the appropriate DiffMode based on the provided
// flags. It validates mutual exclusion constraints and returns an error for
// invalid flag combinations.
func DetermineDiffMode(sinceRef, baseRef, headRef string) (DiffMode, error) {
	hasSince := sinceRef != ""
	hasBase := baseRef != ""
	hasHead := headRef != ""

	// --since and --base/--head are mutually exclusive.
	if hasSince && (hasBase || hasHead) {
		return 0, fmt.Errorf("--since and --base/--head are mutually exclusive; use one or the other")
	}

	// --base and --head must be used together.
	if hasBase && !hasHead {
		return 0, fmt.Errorf("--base requires --head; both must be specified together")
	}
	if hasHead && !hasBase {
		return 0, fmt.Errorf("--head requires --base; both must be specified together")
	}

	switch {
	case hasSince:
		return DiffModeSince, nil
	case hasBase && hasHead:
		return DiffModeBaseHead, nil
	default:
		return DiffModeCache, nil
	}
}