package discovery

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/harvx/harvx/internal/pipeline"
)

// WalkerConfig holds configuration for the file discovery walker.
type WalkerConfig struct {
	// Root is the target directory to walk.
	Root string

	// GitignoreMatcher handles .gitignore pattern matching (from T-011).
	GitignoreMatcher Ignorer

	// HarvxignoreMatcher handles .harvxignore pattern matching (from T-012).
	HarvxignoreMatcher Ignorer

	// DefaultIgnorer handles built-in default ignore patterns (from T-012).
	DefaultIgnorer Ignorer

	// PatternFilter applies include/exclude/extension filtering (from T-014).
	PatternFilter *PatternFilter

	// GitTrackedOnly restricts discovery to git-tracked files when true.
	GitTrackedOnly bool

	// SkipLargeFiles is the file size threshold in bytes. Files exceeding this
	// size are skipped. A value of 0 disables large file skipping.
	SkipLargeFiles int64

	// Concurrency is the maximum number of parallel file-reading workers.
	// Defaults to runtime.NumCPU() if <= 0.
	Concurrency int
}

// Walker is the core file discovery engine that traverses a directory tree,
// applies all filtering criteria, and reads file contents in parallel using
// bounded concurrency via errgroup.
type Walker struct {
	logger *slog.Logger
}

// NewWalker creates a new Walker instance.
func NewWalker() *Walker {
	return &Walker{
		logger: slog.Default().With("component", "walker"),
	}
}

// Walk discovers files in the directory tree rooted at cfg.Root, applying all
// configured filters, and reads file contents in parallel. It returns a
// DiscoveryResult with the discovered files sorted alphabetically by path.
//
// The walk proceeds in two phases:
//  1. Walking: filepath.WalkDir traverses the tree, applying ignore rules,
//     binary detection, size limits, and pattern filters. Matching files are
//     collected as FileDescriptors.
//  2. Content loading: errgroup workers read file contents in parallel with
//     bounded concurrency. Per-file errors are captured in FileDescriptor.Error
//     rather than aborting the entire walk.
//
// Context cancellation stops both phases promptly.
func (w *Walker) Walk(ctx context.Context, cfg WalkerConfig) (*pipeline.DiscoveryResult, error) {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = runtime.NumCPU()
	}

	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, fmt.Errorf("resolving root path %s: %w", cfg.Root, err)
	}

	// Verify root exists and is a directory.
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat root %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root %s is not a directory", root)
	}

	// Build composite ignorer from all ignore sources.
	composite := NewCompositeIgnorer(
		cfg.DefaultIgnorer,
		cfg.GitignoreMatcher,
		cfg.HarvxignoreMatcher,
	)

	// Load git-tracked file set if needed.
	var gitTracked map[string]bool
	if cfg.GitTrackedOnly {
		gitTracked, err = GitTrackedFiles(root)
		if err != nil {
			return nil, fmt.Errorf("loading git tracked files: %w", err)
		}
		w.logger.Debug("git-tracked-only mode",
			"tracked_files", len(gitTracked),
		)
	}

	// Symlink resolver for loop detection.
	symResolver := NewSymlinkResolver()

	// Phase 1: Walk and collect file descriptors.
	var files []*pipeline.FileDescriptor
	skipReasons := make(map[string]int)
	var mu sync.Mutex
	totalFound := 0

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		// Check context cancellation.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if walkErr != nil {
			w.logger.Debug("walk error",
				"path", path,
				"error", walkErr,
			)
			return nil // Skip entries with errors, don't abort.
		}

		// Compute relative path.
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		// Skip the root directory itself.
		if relPath == "." {
			return nil
		}

		isDir := d.IsDir()

		// Skip .git directory always.
		if isDir && d.Name() == ".git" {
			w.logger.Debug("skipping .git directory", "path", relPath)
			return fs.SkipDir
		}

		// Check composite ignorer (defaults, .gitignore, .harvxignore).
		if composite.IsIgnored(relPath, isDir) {
			w.logger.Debug("ignored by pattern",
				"path", relPath,
				"is_dir", isDir,
			)
			if isDir {
				mu.Lock()
				skipReasons["ignored_dir"]++
				mu.Unlock()
				return fs.SkipDir
			}
			mu.Lock()
			totalFound++
			skipReasons["ignored"]++
			mu.Unlock()
			return nil
		}

		// For directories, no further processing needed.
		if isDir {
			return nil
		}

		// Count every non-directory entry we encounter.
		mu.Lock()
		totalFound++
		mu.Unlock()

		// Handle symlinks.
		isSymlink := d.Type()&os.ModeSymlink != 0
		absPath := path
		if isSymlink {
			realPath, isLoop, err := symResolver.Resolve(path)
			if err != nil {
				w.logger.Debug("symlink error",
					"path", relPath,
					"error", err,
				)
				mu.Lock()
				skipReasons["symlink_error"]++
				mu.Unlock()
				return nil
			}
			if isLoop {
				w.logger.Debug("symlink loop",
					"path", relPath,
				)
				mu.Lock()
				skipReasons["symlink_loop"]++
				mu.Unlock()
				return nil
			}
			symResolver.MarkVisited(realPath)
			absPath = realPath
		}

		// Git-tracked-only check.
		if cfg.GitTrackedOnly && gitTracked != nil {
			if !gitTracked[relPath] {
				w.logger.Debug("not git-tracked",
					"path", relPath,
				)
				mu.Lock()
				skipReasons["not_tracked"]++
				mu.Unlock()
				return nil
			}
		}

		// Get file info for size checks and binary detection.
		fileInfo, err := os.Stat(absPath)
		if err != nil {
			w.logger.Debug("stat error",
				"path", relPath,
				"error", err,
			)
			mu.Lock()
			skipReasons["stat_error"]++
			mu.Unlock()
			return nil
		}

		// Large file check.
		if cfg.SkipLargeFiles > 0 && fileInfo.Size() > cfg.SkipLargeFiles {
			w.logger.Debug("large file skipped",
				"path", relPath,
				"size", fileInfo.Size(),
				"threshold", cfg.SkipLargeFiles,
			)
			mu.Lock()
			skipReasons["large_file"]++
			mu.Unlock()
			return nil
		}

		// Binary detection.
		isBin, binErr := IsBinary(absPath)
		if binErr != nil {
			// Can't determine if binary (e.g., permission denied).
			// Include the file; the content-reading phase will capture the error.
			w.logger.Debug("binary detection error, including file anyway",
				"path", relPath,
				"error", binErr,
			)
		}
		if isBin {
			w.logger.Debug("binary file skipped",
				"path", relPath,
			)
			mu.Lock()
			skipReasons["binary"]++
			mu.Unlock()
			return nil
		}

		// Pattern filter (include/exclude/extension).
		if cfg.PatternFilter != nil && cfg.PatternFilter.HasFilters() {
			if !cfg.PatternFilter.Matches(relPath) {
				w.logger.Debug("pattern filter excluded",
					"path", relPath,
				)
				mu.Lock()
				skipReasons["pattern_filter"]++
				mu.Unlock()
				return nil
			}
		}

		fd := &pipeline.FileDescriptor{
			Path:      relPath,
			AbsPath:   absPath,
			Size:      fileInfo.Size(),
			IsSymlink: isSymlink,
			Tier:      pipeline.DefaultTier,
		}
		mu.Lock()
		files = append(files, fd)
		mu.Unlock()

		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("walking directory %s: %w", root, walkErr)
	}

	// Sort files by path for deterministic output.
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Phase 2: Read file contents in parallel.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(cfg.Concurrency)

	for _, fd := range files {
		fd := fd // capture loop variable
		g.Go(func() error {
			content, err := readFile(gctx, fd.AbsPath)
			if err != nil {
				fd.Error = fmt.Errorf("reading %s: %w", fd.Path, err)
				w.logger.Debug("file read error",
					"path", fd.Path,
					"error", err,
				)
				return nil // Non-fatal: capture error, continue.
			}
			fd.Content = content
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("reading file contents: %w", err)
	}

	// Build result slice (convert pointers to values).
	resultFiles := make([]pipeline.FileDescriptor, len(files))
	for i, fd := range files {
		resultFiles[i] = *fd
	}

	totalSkipped := 0
	for _, count := range skipReasons {
		totalSkipped += count
	}

	result := &pipeline.DiscoveryResult{
		Files:        resultFiles,
		TotalFound:   totalFound,
		TotalSkipped: totalSkipped,
		SkipReasons:  skipReasons,
	}

	w.logger.Info("discovery complete",
		"files", len(resultFiles),
		"total_found", totalFound,
		"total_skipped", totalSkipped,
	)

	return result, nil
}

// readFile reads the entire content of a file. It respects context cancellation
// by checking the context before reading. Returns the file content as a string.
func readFile(ctx context.Context, path string) (string, error) {
	// Check cancellation before reading the file.
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	return string(data), nil
}
