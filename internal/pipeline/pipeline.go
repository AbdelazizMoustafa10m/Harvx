package pipeline

import (
	"context"
	"log/slog"

	"github.com/harvx/harvx/internal/config"
)

// Run executes the harvx context generation pipeline. It is the central
// orchestrator that coordinates discovery, filtering, relevance sorting,
// content loading, tokenization, redaction, compression, and rendering.
//
// Currently this is a stub that logs the resolved configuration and returns
// nil. Each pipeline stage will be implemented by later tasks.
func Run(ctx context.Context, cfg *config.FlagValues) error {
	slog.Info("Starting Harvx context generation",
		"dir", cfg.Dir,
		"output", cfg.Output,
		"format", cfg.Format,
	)

	slog.Debug("resolved configuration",
		"dir", cfg.Dir,
		"output", cfg.Output,
		"format", cfg.Format,
		"target", cfg.Target,
		"filters", cfg.Filters,
		"includes", cfg.Includes,
		"excludes", cfg.Excludes,
		"git_tracked_only", cfg.GitTrackedOnly,
		"skip_large_files", cfg.SkipLargeFiles,
		"stdout", cfg.Stdout,
		"line_numbers", cfg.LineNumbers,
		"no_redact", cfg.NoRedact,
	)

	// TODO: Implement discovery, filtering, rendering pipeline
	return nil
}
