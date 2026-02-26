// Package watcher provides file system change notification for gosync.
package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Event represents a file system change event.
type Event struct {
	Path      string
	Operation string
	Timestamp time.Time
}

// Options configures the file watcher.
type Options struct {
	Root         string
	Debounce     time.Duration
	IgnoreHidden bool
	Logger       *slog.Logger
}

// Watcher monitors a directory tree for file changes.
type Watcher struct {
	root     string
	debounce time.Duration
	ignore   bool
	logger   *slog.Logger
	fsw      *fsnotify.Watcher
}

// New creates a new file system watcher.
func New(opts Options) (*Watcher, error) {
	if opts.Root == "" {
		return nil, fmt.Errorf("watcher: root path is required")
	}
	if opts.Debounce == 0 {
		opts.Debounce = 200 * time.Millisecond
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("watcher: creating fsnotify watcher: %w", err)
	}

	return &Watcher{
		root:     opts.Root,
		debounce: opts.Debounce,
		ignore:   opts.IgnoreHidden,
		logger:   opts.Logger,
		fsw:      fsw,
	}, nil
}

// Watch starts watching the root directory and sends events on the returned channel.
// The channel is closed when the context is cancelled.
func (w *Watcher) Watch(ctx context.Context) (<-chan Event, error) {
	if err := w.fsw.Add(w.root); err != nil {
		return nil, fmt.Errorf("watcher: adding root %s: %w", w.root, err)
	}

	events := make(chan Event, 32)

	go func() {
		defer close(events)
		defer w.fsw.Close()

		timer := time.NewTimer(w.debounce)
		timer.Stop()
		var pending []Event

		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-w.fsw.Events:
				if !ok {
					return
				}
				if w.ignore && isHidden(ev.Name) {
					continue
				}
				pending = append(pending, Event{
					Path:      ev.Name,
					Operation: ev.Op.String(),
					Timestamp: time.Now(),
				})
				timer.Reset(w.debounce)

			case err, ok := <-w.fsw.Errors:
				if !ok {
					return
				}
				w.logger.Warn("watcher error", "error", err)

			case <-timer.C:
				for _, ev := range pending {
					select {
					case events <- ev:
					case <-ctx.Done():
						return
					}
				}
				pending = pending[:0]
			}
		}
	}()

	return events, nil
}

func isHidden(path string) bool {
	base := filepath.Base(path)
	return len(base) > 0 && base[0] == '.'
}