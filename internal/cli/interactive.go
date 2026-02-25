package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tui"
	"github.com/spf13/cobra"
)

// runInteractive launches the Bubble Tea TUI with the resolved configuration
// and pipeline. It creates the tea.Program with alt screen and mouse support.
func runInteractive(_ *cobra.Command, cfg *config.ResolvedConfig, p *pipeline.Pipeline) error {
	model, err := tui.New(cfg, p)
	if err != nil {
		return fmt.Errorf("initializing TUI: %w", err)
	}

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	slog.Debug("launching interactive TUI")

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	// Check if the TUI model produced a result we need to handle.
	_ = finalModel
	return nil
}

// ShouldLaunchInteractive determines whether the TUI should be launched
// automatically based on smart default detection. The TUI launches when:
//   - The --interactive/-i flag is explicitly set, OR
//   - Stdin is a terminal AND no harvx.toml exists in the directory tree
//     (indicating a first-time / exploratory use case)
//
// The flag takes explicit precedence: --interactive=false disables auto-launch.
// When stdin is not a terminal (piped, CI, tests), the smart default is skipped.
func ShouldLaunchInteractive(cmd *cobra.Command, fv *config.FlagValues) bool {
	// If the flag was explicitly set by the user, respect it.
	if cmd.Flags().Changed("interactive") {
		return fv.Interactive
	}

	// Smart default requires a terminal (stdin must be a TTY).
	if !isTerminal(os.Stdin.Fd()) {
		return false
	}

	// Smart default: check if harvx.toml exists in the directory tree.
	return !hasConfigInTree(fv.Dir)
}

// isTerminal reports whether the given file descriptor refers to a terminal.
// This is extracted to a variable for testing.
var isTerminal = defaultIsTerminal

func defaultIsTerminal(fd uintptr) bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// hasConfigInTree walks up from dir looking for harvx.toml. Returns true
// if found, false otherwise. This mirrors the config discovery logic but
// is intentionally simplified for the smart default check.
func hasConfigInTree(dir string) bool {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}

	// Walk up to 20 levels (matching config.DiscoverRepoConfig behavior).
	for i := 0; i < 20; i++ {
		candidate := filepath.Join(absDir, "harvx.toml")
		if _, err := os.Stat(candidate); err == nil {
			return true
		}

		// Also check .harvx.toml variant.
		candidate = filepath.Join(absDir, ".harvx.toml")
		if _, err := os.Stat(candidate); err == nil {
			return true
		}

		parent := filepath.Dir(absDir)
		if parent == absDir {
			break // reached filesystem root
		}
		absDir = parent
	}

	return false
}
