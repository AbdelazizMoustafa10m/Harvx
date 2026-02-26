// Package handler implements the core file synchronization logic.
package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/gosync/internal/config"
)

// ErrSyncAborted indicates the synchronization was cancelled.
var ErrSyncAborted = errors.New("sync aborted")

// Options configures the sync handler.
type Options struct {
	Source      string
	Destination string
	Config      *config.Config
	Watch       bool
	DryRun      bool
	Logger      *slog.Logger
}

// Handler performs file synchronization between directories.
type Handler struct {
	src    string
	dst    string
	cfg    *config.Config
	watch  bool
	dryRun bool
	logger *slog.Logger
}

// New creates a new sync handler with the given options.
func New(opts Options) (*Handler, error) {
	if opts.Source == "" {
		return nil, fmt.Errorf("source path is required")
	}
	if opts.Destination == "" {
		return nil, fmt.Errorf("destination path is required")
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	return &Handler{
		src:    opts.Source,
		dst:    opts.Destination,
		cfg:    opts.Config,
		watch:  opts.Watch,
		dryRun: opts.DryRun,
		logger: opts.Logger,
	}, nil
}

// Run executes the synchronization operation.
func (h *Handler) Run(ctx context.Context) error {
	h.logger.Info("starting sync",
		"source", h.src,
		"destination", h.dst,
		"dry_run", h.dryRun,
	)

	entries, err := h.collectFiles(ctx)
	if err != nil {
		return fmt.Errorf("collecting files: %w", err)
	}

	h.logger.Info("files collected", "count", len(entries))

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return ErrSyncAborted
		default:
		}

		if err := h.syncFile(entry); err != nil {
			h.logger.Warn("failed to sync file",
				"path", entry,
				"error", err,
			)
		}
	}

	return nil
}

func (h *Handler) collectFiles(ctx context.Context) ([]string, error) {
	var files []string

	err := filepath.WalkDir(h.src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(h.src, path)
		if err != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, err)
		}

		if h.shouldIgnore(relPath) {
			return nil
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}

func (h *Handler) shouldIgnore(relPath string) bool {
	for _, pattern := range h.cfg.Sync.Ignore {
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		if strings.Contains(relPath, strings.TrimSuffix(pattern, "/")) {
			return true
		}
	}
	return false
}

func (h *Handler) syncFile(relPath string) error {
	srcPath := filepath.Join(h.src, relPath)
	dstPath := filepath.Join(h.dst, relPath)

	if h.cfg.Sync.Checksum {
		srcHash, err := fileHash(srcPath)
		if err != nil {
			return fmt.Errorf("hashing source %s: %w", srcPath, err)
		}
		dstHash, _ := fileHash(dstPath)
		if srcHash == dstHash {
			h.logger.Debug("file unchanged, skipping", "path", relPath)
			return nil
		}
	}

	if h.dryRun {
		h.logger.Info("would copy", "path", relPath)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", dstPath, err)
	}

	return copyFile(srcPath, dstPath)
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("hashing %s: %w", path, err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source %s: %w", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination %s: %w", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copying to %s: %w", dst, err)
	}

	return dstFile.Sync()
}