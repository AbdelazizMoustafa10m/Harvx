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

// smartDefaultHint is printed to stderr when the TUI is auto-launched via the
// smart default (no args, no config file). It guides first-time users toward
// the explicit flag and headless usage patterns.
const smartDefaultHint = "Tip: Run 'harvx -i' to always open the interactive mode, or create a harvx.toml for headless use."

// runInteractive launches the Bubble Tea TUI with the resolved configuration
// and pipeline. It creates the tea.Program with alt screen and mouse support.
// When smartDefault is true (the TUI was auto-launched because no args and no
// config were detected), it prints a one-line hint to stderr before launching.
func runInteractive(_ *cobra.Command, cfg *config.ResolvedConfig, p *pipeline.Pipeline, smartDefault bool) error {
	if smartDefault {
		fmt.Fprintln(os.Stderr, smartDefaultHint)
	}

	model, err := tui.New(cfg, p)
	if err != nil {
		return fmt.Errorf("initializing TUI: %w", err)
	}

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	slog.Debug("launching interactive TUI", "smart_default", smartDefault)

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
//   - The process was invoked with no subcommand and no flags (bare binary),
//     stdin is a terminal, AND no harvx.toml exists in the directory tree
//     (indicating a first-time / exploratory use case)
//
// The flag takes explicit precedence: --interactive=false disables auto-launch.
// When stdin is not a terminal (piped, CI, tests), the smart default is skipped.
func ShouldLaunchInteractive(cmd *cobra.Command, fv *config.FlagValues) bool {
	// If the flag was explicitly set by the user, respect it.
	if cmd.Flags().Changed("interactive") {
		return fv.Interactive
	}

	// Smart default requires: no args, terminal, and no config file.
	if !getOsArgs().isNoArgs() {
		return false
	}

	if !isTerminal(os.Stdin.Fd()) {
		return false
	}

	// Smart default: check if harvx.toml exists in the directory tree.
	return !hasConfigInTree(fv.Dir)
}

// IsNoArgsInvocation reports whether harvx was invoked with no subcommand
// and no flags -- just the bare binary name. This is used by the smart default
// to determine if the TUI should be auto-launched for first-time users.
// It returns true only when osArgs contains a single element (the binary name).
func IsNoArgsInvocation(osArgs []string) bool {
	return len(osArgs) <= 1
}

// osArgsProvider abstracts access to os.Args for testing. The default
// implementation returns the real os.Args; tests can replace getOsArgs
// to inject controlled values without mutating global state.
type osArgsProvider struct {
	args []string
}

func (p osArgsProvider) isNoArgs() bool {
	return IsNoArgsInvocation(p.args)
}

// getOsArgs returns the current process arguments. It is a package-level
// variable so tests can override it without mutating os.Args.
var getOsArgs = func() osArgsProvider {
	return osArgsProvider{args: os.Args}
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
